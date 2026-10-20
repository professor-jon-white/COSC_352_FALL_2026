package analysis

import (
	"fmt"
	"math"
	"sort"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/model"
	"baltimore-intel/internal/report"
	"baltimore-intel/internal/stats"
)

// ServiceEquity (Q4, Department of Transportation and the 311 agencies):
// are high-crime neighborhoods served as quickly as everyone else?
type ServiceEquity struct{}

func (ServiceEquity) ID() string { return "q4-service-equity" }

var tierNames = [3]string{"Low-crime third", "Middle third", "High-crime third"}

type equityAcc struct {
	tierDays  [3][]float64
	tierAdj   [3][]float64 // days / citywide median for that request type
	tierN     [3]int
	tierOpen  [3]int
	tierOnDue [3]int // closed by due date
	tierDue   [3]int // closed and had a due date
	agency    map[model.ID]*agencyAcc
}

type agencyAcc struct {
	n, open, onDue, withDue int
	days                    []float64
	tierDays                [3][]float64
}

func (ServiceEquity) Run(j *join.Joined) (*report.Section, error) {
	ds := j.DS
	// 1. Crime terciles over neighborhoods that have 311 activity.
	ids := j.Hoods(func(h join.HoodStat) bool { return h.Requests >= 25 })
	if len(ids) < 6 {
		return nil, fmt.Errorf("too few neighborhoods with 311 data")
	}
	sort.Slice(ids, func(a, b int) bool {
		va, vb := j.Hood[ids[a]].Violent, j.Hood[ids[b]].Violent
		if va != vb {
			return va < vb
		}
		return ids[a] < ids[b]
	})
	tier := make([]int8, len(j.Hood))
	for i := range tier {
		tier[i] = -1
	}
	for k, id := range ids {
		tier[id] = int8(k * 3 / len(ids))
	}

	// 2. Citywide median close time per request type (controls for mix:
	//    a pothole and a rat complaint have very different normal times).
	typeDays := join.ParallelReduce(ds.Requests, j.Workers,
		func() map[model.ID][]float64 { return map[model.ID][]float64{} },
		func(m map[model.ID][]float64, q *model.ServiceRequest) {
			if q.Closed > 0 {
				m[q.Type] = append(m[q.Type], float64(q.Closed-q.Created)/86400)
			}
		},
		func(dst, src map[model.ID][]float64) {
			for k, v := range src {
				dst[k] = append(dst[k], v...)
			}
		})
	typeMed := make(map[model.ID]float64, len(typeDays))
	for k, v := range typeDays {
		if len(v) >= 20 {
			typeMed[k] = math.Max(stats.Median(v), 0.25) // floor avoids dividing by ~0 for same-day types
		}
	}

	// 3. Main map-reduce over requests.
	acc := join.ParallelReduce(ds.Requests, j.Workers,
		func() *equityAcc { return &equityAcc{agency: map[model.ID]*agencyAcc{}} },
		func(a *equityAcc, q *model.ServiceRequest) {
			t := tier[q.Hood]
			ag := a.agency[q.Agency]
			if ag == nil {
				ag = &agencyAcc{}
				a.agency[q.Agency] = ag
			}
			ag.n++
			if q.Closed == 0 {
				ag.open++
				if t >= 0 {
					a.tierOpen[t]++
					a.tierN[t]++
				}
				return
			}
			d := float64(q.Closed-q.Created) / 86400
			ag.days = append(ag.days, d)
			if q.Due > 0 {
				ag.withDue++
				if q.Closed <= q.Due {
					ag.onDue++
				}
			}
			if t < 0 {
				return
			}
			a.tierN[t]++
			a.tierDays[t] = append(a.tierDays[t], d)
			ag.tierDays[t] = append(ag.tierDays[t], d)
			if m, ok := typeMed[q.Type]; ok {
				a.tierAdj[t] = append(a.tierAdj[t], d/m)
			}
			if q.Due > 0 {
				a.tierDue[t]++
				if q.Closed <= q.Due {
					a.tierOnDue[t]++
				}
			}
		},
		func(dst, src *equityAcc) {
			for t := 0; t < 3; t++ {
				dst.tierDays[t] = append(dst.tierDays[t], src.tierDays[t]...)
				dst.tierAdj[t] = append(dst.tierAdj[t], src.tierAdj[t]...)
				dst.tierN[t] += src.tierN[t]
				dst.tierOpen[t] += src.tierOpen[t]
				dst.tierOnDue[t] += src.tierOnDue[t]
				dst.tierDue[t] += src.tierDue[t]
			}
			for k, s := range src.agency {
				d := dst.agency[k]
				if d == nil {
					dst.agency[k] = s
					continue
				}
				d.n += s.n
				d.open += s.open
				d.onDue += s.onDue
				d.withDue += s.withDue
				d.days = append(d.days, s.days...)
				for t := 0; t < 3; t++ {
					d.tierDays[t] = append(d.tierDays[t], s.tierDays[t]...)
				}
			}
		})

	tierBars := &report.Bars{Title: "Median days to close a 311 request, by neighborhood violent-crime tercile", Format: "%.1f", Unit: " days"}
	adjBars := &report.Bars{Title: "Mix-adjusted wait index (1.00 = citywide normal for the same request type)", Format: "%.2f"}
	tierTbl := &report.Table{Title: "Service outcomes by crime tercile",
		Headers: []string{"Tercile", "Requests", "Still open", "Median days", "90th pct days", "Closed by due date", "Wait index"},
		Right:   []bool{false, true, true, true, true, true, true}, CSVName: "q4_service_by_tercile.csv"}
	var med, adj [3]float64
	for t := 0; t < 3; t++ {
		med[t] = stats.Median(acc.tierDays[t])
		adj[t] = stats.Median(acc.tierAdj[t])
		tierBars.Labels = append(tierBars.Labels, tierNames[t])
		tierBars.Values = append(tierBars.Values, med[t])
		adjBars.Labels = append(adjBars.Labels, tierNames[t])
		adjBars.Values = append(adjBars.Values, adj[t])
		tierTbl.Rows = append(tierTbl.Rows, []string{tierNames[t], report.Int(acc.tierN[t]),
			report.Pct(share(acc.tierOpen[t], acc.tierN[t])), report.F(med[t], 1),
			report.F(stats.Quantile(acc.tierDays[t], 0.9), 1), report.Pct(share(acc.tierOnDue[t], acc.tierDue[t])),
			report.F(adj[t], 2)})
	}

	// Agencies ranked by volume, with the low/high-crime gap per agency.
	type agRow struct {
		id model.ID
		a  *agencyAcc
	}
	var ags []agRow
	for id, a := range acc.agency {
		if id != 0 && a.n >= 50 {
			ags = append(ags, agRow{id, a})
		}
	}
	sort.Slice(ags, func(p, q int) bool {
		if ags[p].a.n != ags[q].a.n {
			return ags[p].a.n > ags[q].a.n
		}
		return ags[p].id < ags[q].id
	})
	agTbl := &report.Table{Title: "Agencies by request volume: speed and the low- vs high-crime gap",
		Headers: []string{"Agency", "Requests", "Still open", "Median days", "Closed by due date", "Low-crime median", "High-crime median", "Gap"},
		Right:   []bool{false, true, true, true, true, true, true, true}, CSVName: "q4_agencies.csv"}
	worst, worstGap := "", 0.0
	for _, r := range ags[:min(10, len(ags))] {
		lo, hi := stats.Median(r.a.tierDays[0]), stats.Median(r.a.tierDays[2])
		gap := hi - lo
		name := ds.Agencies.Name(r.id)
		if len(r.a.tierDays[0]) >= 20 && len(r.a.tierDays[2]) >= 20 && gap > worstGap {
			worst, worstGap = name, gap
		}
		agTbl.Rows = append(agTbl.Rows, []string{name, report.Int(r.a.n), report.Pct(share(r.a.open, r.a.n)),
			report.F(stats.Median(r.a.days), 1), report.Pct(share(r.a.onDue, r.a.withDue)),
			report.F(lo, 1), report.F(hi, 1), fmt.Sprintf("%+.1f d", gap)})
	}

	direction := "longer"
	if adj[2] < adj[0] {
		direction = "shorter"
	}
	finding := fmt.Sprintf("Requests from the highest-crime third of neighborhoods take a median %s days to close versus %s days in the "+
		"lowest-crime third, and %s vs %s are closed by their due date. After adjusting for request type, high-crime "+
		"neighborhoods wait %s (index %s vs %s), so the gap is not just a different mix of requests.",
		report.F(med[2], 1), report.F(med[0], 1),
		report.Pct(share(acc.tierOnDue[2], acc.tierDue[2])), report.Pct(share(acc.tierOnDue[0], acc.tierDue[0])),
		direction, report.F(adj[2], 2), report.F(adj[0], 2))
	if worst != "" {
		finding += fmt.Sprintf(" The widest agency gap is at %s (%+.1f days).", worst, worstGap)
	}

	return &report.Section{
		ID: "q4", Stakeholder: "Department of Transportation & 311 service agencies",
		Question: "Are 311 requests from high-crime neighborhoods resolved as quickly as those from elsewhere?",
		Datasets: []string{"311 Service Requests", "Part 1 Crime"},
		Finding:  finding,
		Blocks:   []report.Block{tierBars, adjBars, tierTbl, agTbl},
		Actions: []string{
			"Add a neighborhood-tercile breakdown to each agency's CitiStat dashboard so service gaps are visible every month.",
			"Where the gap is largest, check whether crews avoid or deprioritize certain areas and set equal due-date targets.",
			"Pair DOT work orders with BPD presence where crews report safety concerns, rather than letting requests age.",
		},
		Notes: []string{
			"Terciles split the neighborhoods with 311 activity into three equal-sized groups by violent-crime count.",
			"Wait index = median of (days to close / citywide median for that request type); types with fewer than 20 closed requests are excluded.",
		},
	}, nil
}
