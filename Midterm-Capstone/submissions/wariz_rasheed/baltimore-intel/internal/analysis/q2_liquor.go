package analysis

import (
	"fmt"
	"math"
	"sort"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
	"baltimoreintel/internal/stats"
)

// LiquorVsNightCrime (licenses x crime): does liquor-license density track
// night-time violent crime?
type LiquorVsNightCrime struct{}

// ID implements Analysis.
func (LiquorVsNightCrime) ID() string { return "Q2" }

type licenseTier struct {
	Label  string
	Lo, Hi int
}

var licenseTiers = []licenseTier{
	{"0 licenses", 0, 0},
	{"1-2 licenses", 1, 2},
	{"3-5 licenses", 3, 5},
	{"6+ licenses", 6, math.MaxInt32},
}

func nightShare(night, timed int) string {
	return pct(night, timed)
}

// Run implements Analysis.
func (LiquorVsNightCrime) Run(d *model.Dataset) report.Result {
	res := report.Result{
		ID:       "Q2",
		Title:    "Do liquor licenses track night-time violent crime?",
		Audience: "Board of Liquor License Commissioners / Baltimore Police Department",
		Question: "Do neighborhoods with more active liquor licenses see more violent crime between " +
			"9 PM and 4 AM (Eastern time)?",
	}
	hoods := universe(d)
	if len(hoods) < 3 {
		res.Finding = "Not enough neighborhoods with data in the analysis window."
		return res
	}

	lic := make([]float64, len(hoods))
	night := make([]float64, len(hoods))
	for i, h := range hoods {
		lic[i] = float64(h.Licenses)
		night[i] = float64(h.NightViolent)
	}
	rho := stats.Spearman(lic, night)

	type tierAgg struct {
		hoods, night, violent, timed int
	}
	aggs := make([]tierAgg, len(licenseTiers))
	for _, h := range hoods {
		for ti, t := range licenseTiers {
			if h.Licenses >= t.Lo && h.Licenses <= t.Hi {
				aggs[ti].hoods++
				aggs[ti].night += h.NightViolent
				aggs[ti].violent += h.Violent
				aggs[ti].timed += h.TimedViolent
				break
			}
		}
	}

	chart := report.Chart{Title: "Average night-time violent crimes per neighborhood, by number of active liquor licenses"}
	tierTable := report.Table{
		Title:   "Neighborhoods grouped by active liquor licenses",
		Headers: []string{"License tier", "Neighborhoods", "Avg violent crimes", "Avg night violent crimes", "Night share of violent"},
	}
	avgNight := make([]float64, len(licenseTiers))
	for ti, t := range licenseTiers {
		a := aggs[ti]
		avgViolent, avgN := 0.0, 0.0
		if a.hoods > 0 {
			avgViolent = float64(a.violent) / float64(a.hoods)
			avgN = float64(a.night) / float64(a.hoods)
		}
		avgNight[ti] = avgN
		chart.Bars = append(chart.Bars, report.Bar{
			Label: t.Label,
			Value: avgN,
			Text:  fmt.Sprintf("%s (n=%d neighborhoods)", f1(avgN), a.hoods),
		})
		tierTable.Rows = append(tierTable.Rows, []string{
			t.Label, fmt.Sprintf("%d", a.hoods), f1(avgViolent), f1(avgN), nightShare(a.night, a.timed),
		})
	}

	// Top neighborhoods by license count.
	order := make([]int, len(hoods))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		ha, hb := hoods[order[a]], hoods[order[b]]
		if ha.Licenses != hb.Licenses {
			return ha.Licenses > hb.Licenses
		}
		return ha.Name < hb.Name
	})
	if len(order) > 10 {
		order = order[:10]
	}
	topTable := report.Table{
		Title:   "Ten neighborhoods with the most active liquor licenses",
		Headers: []string{"Neighborhood", "Licenses", "Violent crimes", "Night violent crimes", "Night share of violent"},
	}
	for _, i := range order {
		h := hoods[i]
		topTable.Rows = append(topTable.Rows, []string{
			pretty(h.Name), fmt.Sprintf("%d", h.Licenses), commas(h.Violent),
			commas(h.NightViolent), nightShare(h.NightViolent, h.TimedViolent),
		})
	}

	// Compare the busiest and quietest non-empty tiers for the headline.
	lo, hi := -1, -1
	for ti := range licenseTiers {
		if aggs[ti].hoods > 0 {
			if lo < 0 {
				lo = ti
			}
			hi = ti
		}
	}
	res.Finding = fmt.Sprintf("Across %d neighborhoods, active liquor licenses and night-time violent crime show a %s (rho = %s).",
		len(hoods), stats.Describe(rho), f2(rho))
	if lo >= 0 && hi > lo {
		res.Finding += fmt.Sprintf(" Neighborhoods in the \"%s\" tier average %s night-time violent crimes, versus %s in the \"%s\" tier.",
			licenseTiers[hi].Label, f1(avgNight[hi]), f1(avgNight[lo]), licenseTiers[lo].Label)
		res.Finding += fmt.Sprintf(" Night-time makes up %s of violent crime with a known time in the \"%s\" tier, versus %s in the \"%s\" tier.",
			nightShare(aggs[hi].night, aggs[hi].timed), licenseTiers[hi].Label,
			nightShare(aggs[lo].night, aggs[lo].timed), licenseTiers[lo].Label)
	}
	res.Charts = []report.Chart{chart}
	res.Tables = []report.Table{tierTable, topTable}
	res.Notes = []string{
		"Night is 21:00-03:59 Eastern. Crime timestamps are published in UTC and converted to local time; " +
			"records at exactly local midnight are treated as 'time unknown' and excluded from night counts.",
		"Licenses are matched to neighborhoods through their street block. Licenses whose block never appears in the crime or 311 data cannot be placed in a neighborhood (see the data overview).",
		"Licenses cluster in commercial and entertainment districts that also draw many visitors, so a positive link is not evidence that licenses cause crime.",
	}
	return res
}
