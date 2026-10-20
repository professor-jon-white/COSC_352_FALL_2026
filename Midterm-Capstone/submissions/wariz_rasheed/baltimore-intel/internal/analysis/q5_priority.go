package analysis

import (
	"fmt"
	"sort"
	"strings"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
	"baltimoreintel/internal/stats"
)

// Weights sets how much each factor contributes to the priority score. They
// should sum to 1. Because the weights are a policy choice, they live in one
// place and are printed in the report.
type Weights struct {
	Violent  float64
	Requests float64
	Slowness float64
	Licenses float64
}

// DefaultWeights puts the most weight on violent crime.
func DefaultWeights() Weights {
	return Weights{Violent: 0.45, Requests: 0.20, Slowness: 0.20, Licenses: 0.15}
}

// PriorityIndex (all three datasets): which neighborhoods should get combined,
// cross-agency attention first?
type PriorityIndex struct {
	W Weights
}

// ID implements Analysis.
func (PriorityIndex) ID() string { return "Q5" }

// Run implements Analysis.
func (p PriorityIndex) Run(d *model.Dataset) report.Result {
	res := report.Result{
		ID:       "Q5",
		Title:    "Which neighborhoods need combined intervention first?",
		Audience: "Mayor's Office",
		Question: "Combining all three datasets, which neighborhoods rank highest on violent crime, 311 demand, " +
			"slow 311 response and liquor-license density together?",
	}
	hoods := universe(d)
	n := len(hoods)
	if n < 3 {
		res.Finding = "Not enough neighborhoods with data in the analysis window to build a ranking."
		return res
	}

	// Citywide median close time stands in for neighborhoods without enough
	// completed requests, so a data gap neither helps nor hurts their score.
	var allHours []float64
	for _, h := range hoods {
		allHours = append(allHours, h.CloseHours...)
	}
	cityDays := stats.Median(allHours) / 24

	violent := make([]float64, n)
	reqs := make([]float64, n)
	days := make([]float64, n)
	lics := make([]float64, n)
	for i, h := range hoods {
		violent[i] = float64(h.Violent)
		reqs[i] = float64(h.Requests)
		lics[i] = float64(h.Licenses)
		if h.enoughClosed() {
			days[i] = stats.Median(h.CloseHours) / 24
		} else {
			days[i] = cityDays
		}
	}
	pv := stats.PercentileRanks(violent)
	pr := stats.PercentileRanks(reqs)
	ps := stats.PercentileRanks(days)
	pl := stats.PercentileRanks(lics)

	w := p.W
	scores := make([]float64, n)
	drivers := make([]string, n)
	type factor struct {
		name string
		pct  float64
	}
	for i := 0; i < n; i++ {
		scores[i] = w.Violent*pv[i] + w.Requests*pr[i] + w.Slowness*ps[i] + w.Licenses*pl[i]
		fs := []factor{
			{"violent crime", pv[i]},
			{"311 volume", pr[i]},
			{"slow response", ps[i]},
			{"licenses", pl[i]},
		}
		var high []string
		best := 0
		for k, f := range fs {
			if f.pct >= 80 {
				high = append(high, f.name)
			}
			if f.pct > fs[best].pct {
				best = k
			}
		}
		if len(high) == 0 {
			high = []string{fs[best].name}
		}
		drivers[i] = strings.Join(high, ", ")
	}

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		if scores[order[a]] != scores[order[b]] {
			return scores[order[a]] > scores[order[b]]
		}
		return hoods[order[a]].Name < hoods[order[b]].Name
	})
	top := order
	if len(top) > 15 {
		top = top[:15]
	}

	chart := report.Chart{Title: "Priority score (0-100), top neighborhoods"}
	table := report.Table{
		Title:   "Top priority neighborhoods",
		Headers: []string{"Rank", "Neighborhood", "Score", "Violent crimes", "311 requests", "Median days to close", "Licenses", "Factors in top 20%"},
	}
	for rank, i := range top {
		h := hoods[i]
		chart.Bars = append(chart.Bars, report.Bar{
			Label: trunc(pretty(h.Name), 28),
			Value: scores[i],
			Text:  f1(scores[i]),
		})
		table.Rows = append(table.Rows, []string{
			fmt.Sprintf("%d", rank+1), pretty(h.Name), f1(scores[i]), commas(h.Violent),
			commas(h.Requests), f1(days[i]), fmt.Sprintf("%d", h.Licenses), drivers[i],
		})
	}

	first := hoods[top[0]].Name
	res.Finding = fmt.Sprintf("Of %d neighborhoods ranked, %s scores highest (%s of 100)", n, pretty(first), f1(scores[top[0]]))
	if len(top) >= 3 {
		res.Finding += fmt.Sprintf(", followed by %s and %s", pretty(hoods[top[1]].Name), pretty(hoods[top[2]].Name))
	}
	res.Finding += "."
	res.Charts = []report.Chart{chart}
	res.Tables = []report.Table{table}
	res.Notes = []string{
		fmt.Sprintf("Each factor is converted to a percentile rank (0-100) across neighborhoods, then weighted: "+
			"violent crime %.0f%%, 311 volume %.0f%%, slow 311 response %.0f%%, liquor licenses %.0f%%. "+
			"The weights are a policy choice and can be changed in DefaultWeights().",
			w.Violent*100, w.Requests*100, w.Slowness*100, w.Licenses*100),
		"\"Factors in top 20%\" lists the measures on which a neighborhood ranks at or above the 80th percentile. Counts are not adjusted for population. Neighborhoods without enough completed 311 requests use the citywide median close time.",
	}
	return res
}
