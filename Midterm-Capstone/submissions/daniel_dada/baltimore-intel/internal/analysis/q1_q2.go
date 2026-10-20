package analysis

import (
	"fmt"
	"sort"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
)

// ---- Q1: highest in both crime and 311 ----

type TopBoth struct{}

func (TopBoth) Name() string        { return "Q1: Neighborhoods high in BOTH crime and 311 demand" }
func (TopBoth) Stakeholder() string { return "Mayor's Office" }

func (TopBoth) Run(d *join.Dataset) report.Section {
	es := entries(d)
	crimeRank := rankBy(es, func(s *join.Stats) int { return s.Crime })
	callRank := rankBy(es, func(s *join.Stats) int { return s.Calls311 })

	// Combined score = crime rank + 311 rank (lower is worse).
	sort.SliceStable(es, func(i, j int) bool {
		a := crimeRank[es[i].Name] + callRank[es[i].Name]
		b := crimeRank[es[j].Name] + callRank[es[j].Name]
		return a < b
	})
	if len(es) > 10 {
		es = es[:10]
	}
	maxCrime := 0
	for _, e := range es {
		if e.Stats.Crime > maxCrime {
			maxCrime = e.Stats.Crime
		}
	}
	sec := report.Section{Title: TopBoth{}.Name(), Stakeholder: TopBoth{}.Stakeholder()}
	sec.Lines = append(sec.Lines, "Ranked by crime rank + 311 rank. Bars show crime count.", "")
	for i, e := range es {
		label := fmt.Sprintf("%d. %s", i+1, e.Name)
		sec.Lines = append(sec.Lines,
			report.Bar(label, e.Stats.Crime, maxCrime, 30)+fmt.Sprintf("   | 311 requests: %d", e.Stats.Calls311))
	}
	sec.Lines = append(sec.Lines, "", "Note: raw counts favor large neighborhoods; population data would be needed to normalize.")
	return sec
}

// ---- Q2: sanitation requests vs violent crime ----

type SanitationVsViolence struct{}

func (SanitationVsViolence) Name() string {
	return "Q2: Do sanitation/blight requests track violent crime?"
}
func (SanitationVsViolence) Stakeholder() string { return "Department of Public Works" }

func (SanitationVsViolence) Run(d *join.Dataset) report.Section {
	es := entries(d)
	x := make([]float64, len(es))
	y := make([]float64, len(es))
	for i, e := range es {
		x[i] = float64(e.Stats.Sanitation)
		y[i] = float64(e.Stats.Violent)
	}
	r := pearson(x, y)

	strength := "weak"
	switch a := r; {
	case a < 0 && -a > 0.7, a > 0.7:
		strength = "strong"
	case a < 0 && -a > 0.4, a > 0.4:
		strength = "moderate"
	}

	sort.SliceStable(es, func(i, j int) bool { return es[i].Stats.Sanitation > es[j].Stats.Sanitation })
	if len(es) > 10 {
		es = es[:10]
	}
	maxS := 0
	for _, e := range es {
		if e.Stats.Sanitation > maxS {
			maxS = e.Stats.Sanitation
		}
	}
	sec := report.Section{Title: SanitationVsViolence{}.Name(), Stakeholder: SanitationVsViolence{}.Stakeholder()}
	sec.Lines = append(sec.Lines,
		fmt.Sprintf("Pearson r = %.3f across %d neighborhoods (%s correlation)", r, len(x), strength),
		"Correlation is not causation: both may simply reflect neighborhood size or density.", "",
		"Top 10 by sanitation requests (bar), with violent crimes:")
	for _, e := range es {
		sec.Lines = append(sec.Lines,
			report.Bar(e.Name, e.Stats.Sanitation, maxS, 30)+fmt.Sprintf("   | violent crimes: %d", e.Stats.Violent))
	}
	return sec
}
