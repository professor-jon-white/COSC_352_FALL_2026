package analysis

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
	"baltimore-intel/internal/stats"
)

// LiquorCrime (Q2, Board of Liquor License Commissioners + BPD): is violent
// crime concentrated around liquor outlets, and which outlets stand out?
type LiquorCrime struct{}

func (LiquorCrime) ID() string { return "q2-liquor-crime" }

func (LiquorCrime) Run(j *join.Joined) (*report.Section, error) {
	ds := j.DS
	if len(ds.Outlets) == 0 || len(j.Baseline) == 0 {
		return nil, fmt.Errorf("need liquor outlets and located 311 requests for the spatial join")
	}
	r := ds.RadiusM
	outV := make([]float64, len(j.Outlet))
	for i, e := range j.Outlet {
		outV[i] = float64(e.Violent)
	}
	baseV := make([]float64, len(j.Baseline))
	for i, e := range j.Baseline {
		baseV[i] = float64(e.Violent)
	}
	mOut, mBase := stats.Mean(outV), stats.Mean(baseV)
	lift := ratio(mOut, mBase)

	// Share of all located violent crime that happened within r of an outlet.
	nearV, allV := 0, 0
	for i := range ds.Crimes {
		c := &ds.Crimes[i]
		if c.Violent && c.Lat != 0 {
			allV += int(c.Incidents)
			if j.CrimeNearOutlet[i] {
				nearV += int(c.Incidents)
			}
		}
	}

	// Histogram of violent crimes near each outlet: 8 equal-width bins up to
	// the 95th percentile, plus an overflow bin, so the shape is visible
	// whatever the city's crime level.
	p95 := stats.Quantile(outV, 0.95)
	binW := max(1, int(math.Ceil(p95/8)))
	hist := &report.Bars{Title: fmt.Sprintf("Outlets by number of violent crimes within %.0f m (histogram)", r), Format: "%.0f", Unit: " outlets"}
	counts := make([]int, 9)
	for _, v := range j.Outlet {
		counts[min(8, v.Violent/binW)]++
	}
	for k := 0; k < 9; k++ {
		label := fmt.Sprintf("%d-%d", k*binW, (k+1)*binW-1)
		if k == 8 {
			label = fmt.Sprintf("%d+", 8*binW)
		}
		hist.Labels = append(hist.Labels, label+" violent")
		hist.Values = append(hist.Values, float64(counts[k]))
	}

	// Outlets ranked by nearby violent crime.
	idx := make([]int, len(ds.Outlets))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		ea, eb := j.Outlet[idx[a]], j.Outlet[idx[b]]
		if ea.Violent != eb.Violent {
			return ea.Violent > eb.Violent
		}
		return ds.Outlets[idx[a]].Name < ds.Outlets[idx[b]].Name
	})
	tbl := &report.Table{Title: fmt.Sprintf("Outlets with the most violent crime within %.0f m", r),
		Headers: []string{"#", "Establishment", "Type", "Address", "Neighborhood", "Violent", "All Part 1"},
		Right:   []bool{true, false, false, false, false, true, true}, CSVName: "q2_outlet_exposure.csv"}
	for k, i := range idx {
		if k == 15 {
			break
		}
		o := ds.Outlets[i]
		kind := o.Kind
		if kind == "" {
			kind = o.Class
		}
		tbl.Rows = append(tbl.Rows, []string{fmt.Sprint(k + 1), o.Name, kind, o.Address,
			ds.Hoods.Name(o.Hood), report.Int(j.Outlet[i].Violent), report.Int(j.Outlet[i].All)})
	}

	// By establishment type (tavern vs package store vs restaurant ...).
	byKind := map[string][]float64{}
	for i, o := range ds.Outlets {
		k := strings.TrimSpace(o.Kind)
		if k == "" {
			k = "Class " + strings.TrimSpace(o.Class)
		}
		byKind[k] = append(byKind[k], float64(j.Outlet[i].Violent))
	}
	kindBars := &report.Bars{Title: "Mean violent crimes nearby, by establishment type (types with 10+ outlets)", Format: "%.1f"}
	means := map[string]float64{}
	for k, v := range byKind {
		if len(v) >= 10 {
			means[fmt.Sprintf("%s (n=%d)", k, len(v))] = stats.Mean(v)
		}
	}
	for _, rk := range topN(means, 8) {
		kindBars.Labels = append(kindBars.Labels, rk.key)
		kindBars.Values = append(kindBars.Values, rk.value)
	}
	kindBars.Labels = append(kindBars.Labels, "Comparison locations")
	kindBars.Values = append(kindBars.Values, mBase)

	top10 := 0.0
	for k := 0; k < len(idx)/10; k++ {
		top10 += outV[idx[k]]
	}
	sumOut := 0.0
	for _, v := range outV {
		sumOut += v
	}

	kv := &report.KeyValues{Title: "Spatial join summary", Pairs: [][2]string{
		{"Liquor outlets analysed", report.Int(len(ds.Outlets))},
		{"Outlets matched to a neighborhood", report.Int(j.OutletsAssigned)},
		{fmt.Sprintf("Mean violent crimes within %.0f m of an outlet", r), report.F(mOut, 2)},
		{"Same measure at comparison locations", report.F(mBase, 2)},
		{"Outlet / comparison ratio", report.F(lift, 2) + "x"},
		{"Median near outlet / comparison", report.F(stats.Median(outV), 1) + " / " + report.F(stats.Median(baseV), 1)},
		{fmt.Sprintf("Violent crime within %.0f m of any outlet", r), report.Pct(share(nearV, allV))},
		{"Share of outlet-adjacent violence at top 10% of outlets", report.Pct(ratio(top10, sumOut))},
	}}

	verdict := "more"
	if lift < 1 {
		verdict = "less"
	}
	finding := fmt.Sprintf("Locations within %.0f m of a liquor outlet see %.2fx %s violent crime than comparable city locations "+
		"(%.2f vs %.2f incidents on average), and %s of all located violent crime occurs within %.0f m of an outlet. "+
		"Exposure is concentrated: the top 10%% of outlets account for %s of outlet-adjacent violence.",
		r, lift, verdict, mOut, mBase, report.Pct(share(nearV, allV)), r, report.Pct(ratio(top10, sumOut)))

	return &report.Section{
		ID: "q2", Stakeholder: "Board of Liquor License Commissioners & Baltimore Police Department",
		Question: "Is violent crime concentrated around liquor outlets, and which outlets stand out?",
		Datasets: []string{"Liquor Licenses", "Part 1 Crime", "311 Service Requests (comparison points & neighborhood assignment)"},
		Finding:  finding,
		Blocks:   []report.Block{kv, hist, kindBars, tbl},
		Actions: []string{
			"Flag the top-ranked outlets for license-renewal review and joint BPD/Liquor Board site inspections.",
			"Use the establishment-type comparison when weighing new license applications near existing high-exposure outlets.",
			"Pair enforcement with environmental fixes (lighting, camera coverage, loitering ordinances) around the top decile.",
		},
		Notes: []string{
			fmt.Sprintf("Comparison locations are %d evenly sampled 311 request addresses: places people live and travel, unlike random points that fall in parks or the harbor.", len(j.Baseline)),
			"This is an association, not proof that outlets cause crime: outlets cluster on busy commercial corridors where more people (and more crime) are present.",
		},
	}, nil
}
