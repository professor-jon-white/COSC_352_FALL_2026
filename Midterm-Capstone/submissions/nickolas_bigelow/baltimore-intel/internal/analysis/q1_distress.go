package analysis

import (
	"fmt"
	"math"
	"sort"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
	"baltimore-intel/internal/stats"
)

// CompoundDistress (Q1, Mayor's Office): where do public-safety, housing and
// city-service pressures pile up in the same neighborhoods?
type CompoundDistress struct{}

func (CompoundDistress) ID() string { return "q1-compound-distress" }

func (CompoundDistress) Run(j *join.Joined) (*report.Section, error) {
	ds := j.DS
	// Neighborhoods matched in the crime and 311 data with enough activity to rank.
	ids := j.Hoods(func(h join.HoodStat) bool { return h.Crimes >= 10 && h.Requests >= 25 })
	if len(ids) < 5 {
		return nil, fmt.Errorf("only %d neighborhoods matched across datasets", len(ids))
	}
	n := len(ids)
	violent, vacant, reqs := make([]float64, n), make([]float64, n), make([]float64, n)
	for i, id := range ids {
		h := j.Hood[id]
		violent[i] = float64(h.Violent)
		vacant[i] = float64(h.Vacants)
		reqs[i] = float64(h.Requests)
	}
	// log1p tames the long right tail (a few neighborhoods have 10x the median)
	// so no single measure dominates; z-scores put the three on one scale.
	logz := func(xs []float64) []float64 {
		l := make([]float64, len(xs))
		for i, x := range xs {
			l[i] = math.Log1p(x)
		}
		return stats.ZScores(l)
	}
	zv, zh, zr := logz(violent), logz(vacant), logz(reqs)
	type row struct {
		i     int
		index float64
	}
	rows := make([]row, n)
	for i := range rows {
		rows[i] = row{i, (zv[i] + zh[i] + zr[i]) / 3}
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].index != rows[b].index {
			return rows[a].index > rows[b].index
		}
		return ds.Hoods.Name(ids[rows[a].i]) < ds.Hoods.Name(ids[rows[b].i])
	})

	totV, totH := 0, 0
	for _, h := range j.Hood {
		totV += h.Violent
		totH += h.Vacants
	}
	topK := min(10, n)
	topV, topH := 0, 0
	tbl := &report.Table{Title: "Most distressed neighborhoods (composite index = mean z-score of log counts)",
		Headers: []string{"#", "Neighborhood", "Violent crimes", "Open vacant notices", "311 requests", "Liquor outlets", "Index"},
		Right:   []bool{true, false, true, true, true, true, true}, CSVName: "q1_compound_distress.csv"}
	bars := &report.Bars{Title: "Composite distress index, top 15", Format: "%+.2f"}
	for k, r := range rows {
		h := j.Hood[ids[r.i]]
		if k < topK {
			topV += h.Violent
			topH += h.Vacants
		}
		if k < 15 {
			name := ds.Hoods.Name(ids[r.i])
			tbl.Rows = append(tbl.Rows, []string{fmt.Sprint(k + 1), name, report.Int(h.Violent),
				report.Int(h.Vacants), report.Int(h.Requests), report.Int(h.Outlets), report.F(r.index, 2)})
			bars.Labels = append(bars.Labels, name)
			bars.Values = append(bars.Values, r.index)
		}
	}
	rVH := stats.Spearman(violent, vacant)
	rVR := stats.Spearman(violent, reqs)
	rHR := stats.Spearman(vacant, reqs)
	kv := &report.KeyValues{Title: "How strongly the three pressures move together (Spearman rank correlation)", Pairs: [][2]string{
		{"Violent crime vs vacant buildings", fmt.Sprintf("%s (%s)", report.F(rVH, 2), stats.Strength(rVH))},
		{"Violent crime vs 311 demand", fmt.Sprintf("%s (%s)", report.F(rVR, 2), stats.Strength(rVR))},
		{"Vacant buildings vs 311 demand", fmt.Sprintf("%s (%s)", report.F(rHR, 2), stats.Strength(rHR))},
		{"Neighborhoods ranked", report.Int(n)},
		{fmt.Sprintf("Top %d share of violent crime", topK), report.Pct(share(topV, totV))},
		{fmt.Sprintf("Top %d share of open vacant notices", topK), report.Pct(share(topH, totH))},
	}}

	names := func(k int) string {
		s := ""
		for i := 0; i < k && i < n; i++ {
			if i > 0 {
				s += ", "
			}
			if i == k-1 {
				s += "and "
			}
			s += ds.Hoods.Name(ids[rows[i].i])
		}
		return s
	}
	finding := fmt.Sprintf("%s carry the heaviest combined burden. The %d highest-ranked neighborhoods "+
		"(%s of the %d ranked) account for %s of violent crime and %s of open vacant building notices; "+
		"violent crime and vacancy have a %s rank correlation of %s.",
		names(3), topK, report.Pct(share(topK, n)), n, report.Pct(share(topV, totV)), report.Pct(share(topH, totH)),
		stats.Strength(rVH), report.F(rVH, 2))

	return &report.Section{
		ID: "q1", Stakeholder: "Mayor's Office",
		Question: "Which neighborhoods face compounded distress across public safety, housing and city services?",
		Datasets: []string{"Part 1 Crime", "Vacant Building Notices", "311 Service Requests", "Liquor Licenses"},
		Finding:  finding,
		Blocks:   []report.Block{bars, tbl, kv},
		Actions: []string{
			fmt.Sprintf("Convene a joint BPD / DHCD / DPW working group for the top 5 neighborhoods (%s) with one shared scorecard.", names(5)),
			"Target vacant-to-value acquisitions and demolition funding where violent crime and vacancy are both in the top decile.",
			"Re-run this index quarterly to track whether coordinated investment moves neighborhoods down the list.",
		},
		Notes: []string{
			"Counts are raw (no population denominator is published with these datasets); larger neighborhoods rank higher partly because of size.",
			"Neighborhoods are matched across datasets on a normalized name key (case, spacing and punctuation removed).",
		},
	}, nil
}
