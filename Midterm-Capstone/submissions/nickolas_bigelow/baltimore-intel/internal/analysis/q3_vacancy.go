package analysis

import (
	"fmt"
	"sort"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/model"
	"baltimore-intel/internal/report"
	"baltimore-intel/internal/stats"
)

// VacancySanitation (Q3, DPW + DHCD): do vacant buildings drive trash,
// dumping and rodent complaints, and where is demand out of line with vacancy?
type VacancySanitation struct{}

func (VacancySanitation) ID() string { return "q3-vacancy-sanitation" }

func (VacancySanitation) Run(j *join.Joined) (*report.Section, error) {
	ds := j.DS
	ids := j.Hoods(func(h join.HoodStat) bool { return h.Requests >= 25 })
	if len(ids) < 5 {
		return nil, fmt.Errorf("too few neighborhoods with 311 data")
	}
	x, y := make([]float64, len(ids)), make([]float64, len(ids))
	totSan, totReq := 0, 0
	for i, id := range ids {
		h := j.Hood[id]
		x[i], y[i] = float64(h.Vacants), float64(h.Sanitation)
		totSan += h.Sanitation
		totReq += h.Requests
	}
	pr, sr := stats.Pearson(x, y), stats.Spearman(x, y)
	a, b := stats.LinearFit(x, y)

	// Split neighborhoods at the median vacancy level and compare.
	medX := stats.Median(x)
	var hiS, loS []float64
	for i := range x {
		if x[i] > medX {
			hiS = append(hiS, y[i])
		} else {
			loS = append(loS, y[i])
		}
	}

	// Residuals: sanitation demand beyond what vacancy predicts points to
	// causes vacancy doesn't explain (commercial dumping, missed pickups).
	type res struct {
		id       model.ID
		resid    float64
		expected float64
	}
	rs := make([]res, len(ids))
	for i, id := range ids {
		exp := a + b*x[i]
		rs[i] = res{id, y[i] - exp, exp}
	}
	sort.Slice(rs, func(p, q int) bool {
		if rs[p].resid != rs[q].resid {
			return rs[p].resid > rs[q].resid
		}
		return ds.Hoods.Name(rs[p].id) < ds.Hoods.Name(rs[q].id)
	})
	over := &report.Table{Title: "Sanitation demand well above what vacancy predicts (largest positive residuals)",
		Headers: []string{"Neighborhood", "Open vacants", "Sanitation 311", "Expected", "Excess", "Share of 311 that is sanitation"},
		Right:   []bool{false, true, true, true, true, true}, CSVName: "q3_sanitation_excess.csv"}
	for _, r := range rs[:min(10, len(rs))] {
		h := j.Hood[r.id]
		over.Rows = append(over.Rows, []string{ds.Hoods.Name(r.id), report.Int(h.Vacants), report.Int(h.Sanitation),
			report.F(r.expected, 0), "+" + report.F(r.resid, 0), report.Pct(share(h.Sanitation, h.Requests))})
	}

	// The ten highest-vacancy neighborhoods and their sanitation load.
	vacBars := &report.Bars{Title: "Highest-vacancy neighborhoods: sanitation requests vs open vacant notices"}
	type hv struct {
		id model.ID
		v  int
	}
	var byVac []hv
	for _, id := range ids {
		byVac = append(byVac, hv{id, j.Hood[id].Vacants})
	}
	sort.Slice(byVac, func(p, q int) bool {
		if byVac[p].v != byVac[q].v {
			return byVac[p].v > byVac[q].v
		}
		return byVac[p].id < byVac[q].id
	})
	cmp := &report.Table{Title: "", Headers: []string{"Neighborhood", "Open vacants", "Sanitation 311", "Sanitation per vacant"},
		Right: []bool{false, true, true, true}, CSVName: "q3_high_vacancy.csv"}
	for _, e := range byVac[:min(10, len(byVac))] {
		h := j.Hood[e.id]
		vacBars.Labels = append(vacBars.Labels, ds.Hoods.Name(e.id))
		vacBars.Values = append(vacBars.Values, float64(h.Sanitation))
		cmp.Rows = append(cmp.Rows, []string{ds.Hoods.Name(e.id), report.Int(h.Vacants), report.Int(h.Sanitation),
			report.F(ratio(float64(h.Sanitation), float64(h.Vacants)), 2)})
	}
	vacBars.Unit = " sanitation requests"

	scatter := &report.Scatter{Title: "Each neighborhood: open vacant notices (x) vs sanitation 311 requests (y)",
		XLabel: "open vacant building notices", YLabel: "sanitation / rodent / dumping 311 requests", X: x, Y: y}

	kv := &report.KeyValues{Title: "Relationship between vacancy and sanitation demand", Pairs: [][2]string{
		{"Neighborhoods compared", report.Int(len(ids))},
		{"Sanitation share of all 311 requests", report.Pct(share(totSan, totReq))},
		{"Pearson correlation", fmt.Sprintf("%s (%s)", report.F(pr, 2), stats.Strength(pr))},
		{"Spearman rank correlation", fmt.Sprintf("%s (%s)", report.F(sr, 2), stats.Strength(sr))},
		{"Extra sanitation requests per open vacant", report.F(b, 2)},
		{"Median sanitation 311: above-median vacancy", report.F(stats.Median(hiS), 0)},
		{"Median sanitation 311: at/below-median vacancy", report.F(stats.Median(loS), 0)},
	}}

	finding := fmt.Sprintf("Yes, with a %s link: neighborhoods with more open vacant notices file more sanitation, dumping and rodent "+
		"requests (Pearson r = %s, Spearman = %s). Each additional vacant building is associated with about %s more sanitation "+
		"requests a year, and neighborhoods above the median vacancy level file %sx as many such requests as those below it. "+
		"%s files the most sanitation requests beyond what its vacancy level predicts.",
		stats.Strength(pr), report.F(pr, 2), report.F(sr, 2), report.F(b, 2),
		report.F(ratio(stats.Median(hiS), stats.Median(loS)), 1), ds.Hoods.Name(rs[0].id))

	return &report.Section{
		ID: "q3", Stakeholder: "Department of Public Works & Department of Housing and Community Development",
		Question: "Do vacant buildings drive trash, dumping and rodent complaints, and where is demand out of line with vacancy?",
		Datasets: []string{"Vacant Building Notices", "311 Service Requests"},
		Finding:  finding,
		Blocks:   []report.Block{kv, scatter, vacBars, cmp, over},
		Actions: []string{
			"Schedule proactive alley cleaning and rodent baiting on blocks with clusters of open vacant notices instead of waiting for complaints.",
			"Have DHCD share new vacant notices with DPW weekly so sanitation routes adjust as vacancy changes.",
			"Investigate the excess-demand neighborhoods for causes vacancy does not explain (commercial dumping, missed collections).",
		},
		Notes: []string{"Sanitation requests are 311 types whose name mentions trash, dumping, rats/rodents, dirty alleys, cleaning, bulk items, debris, litter or high grass."},
	}, nil
}
