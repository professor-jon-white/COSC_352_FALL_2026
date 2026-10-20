package analysis

import (
	"fmt"
	"sort"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
	"baltimoreintel/internal/stats"
)

// DisorderVsCrime (311 x crime): which 311 request types track violent crime?
type DisorderVsCrime struct{}

// ID implements Analysis.
func (DisorderVsCrime) ID() string { return "Q1" }

type typeCorr struct {
	Type string
	N    int
	Raw  float64 // Spearman correlation with violent crime
	Adj  float64 // same, after controlling for total 311 volume
}

// Run implements Analysis.
func (DisorderVsCrime) Run(d *model.Dataset) report.Result {
	res := report.Result{
		ID:       "Q1",
		Title:    "Which 311 problems track violent crime?",
		Audience: "Department of Public Works / Department of Transportation",
		Question: "Across Baltimore's neighborhoods, which types of 311 service request rise and fall " +
			"most closely with violent crime, even after accounting for how many 311 requests each " +
			"neighborhood makes in total? These are the neglect signals crews could prioritise first.",
	}
	hoods := universe(d)
	if len(hoods) < 3 {
		res.Finding = "Not enough neighborhoods with data in the analysis window to compute correlations."
		return res
	}

	violent := make([]float64, len(hoods))
	all311 := make([]float64, len(hoods))
	typeTotals := make(map[string]int)
	grand := 0
	for i, h := range hoods {
		violent[i] = float64(h.Violent)
		all311[i] = float64(h.Requests)
		for t, n := range h.ReqByType {
			typeTotals[t] += n
			grand += n
		}
	}

	// Ignore rare request types: a correlation over a handful of requests is noise.
	threshold := maxInt(20, grand/2000)
	var cs []typeCorr
	for t, n := range typeTotals {
		if n < threshold {
			continue
		}
		x := make([]float64, len(hoods))
		for i, h := range hoods {
			x[i] = float64(h.ReqByType[t])
		}
		cs = append(cs, typeCorr{
			Type: t,
			N:    n,
			Raw:  stats.Spearman(x, violent),
			Adj:  stats.PartialSpearman(x, violent, all311),
		})
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Adj != cs[j].Adj {
			return cs[i].Adj > cs[j].Adj
		}
		return cs[i].Type < cs[j].Type
	})

	overall := stats.Spearman(all311, violent)
	if len(cs) == 0 {
		res.Finding = fmt.Sprintf("Across %d neighborhoods, total 311 volume has a %s with violent crime (rho = %s). "+
			"No individual request type had enough volume to rank.",
			len(hoods), stats.Describe(overall), f2(overall))
		return res
	}
	res.Finding = fmt.Sprintf("Across %d neighborhoods, total 311 volume has a %s with violent crime (rho = %s), "+
		"largely because busy neighborhoods report more of everything. After removing that size effect, the request type "+
		"that tracks violent crime most closely is \"%s\" (rho = %s, %s requests).",
		len(hoods), stats.Describe(overall), f2(overall), cs[0].Type, f2(cs[0].Adj), commas(cs[0].N))

	top := cs
	if len(top) > 10 {
		top = top[:10]
	}
	chart := report.Chart{Title: "Size-adjusted Spearman correlation with violent crime, top 311 request types"}
	table := report.Table{
		Title:   "Top 311 request types by size-adjusted correlation with violent crime",
		Headers: []string{"Rank", "311 request type", "Requests", "Rho (raw)", "Rho (size-adjusted)", "Strength (adjusted)"},
	}
	for i, c := range top {
		chart.Bars = append(chart.Bars, report.Bar{
			Label: trunc(c.Type, 34),
			Value: c.Adj,
			Text:  f2(c.Adj),
		})
		table.Rows = append(table.Rows, []string{
			fmt.Sprintf("%d", i+1), c.Type, commas(c.N), f2(c.Raw), f2(c.Adj), stats.Describe(c.Adj),
		})
	}
	res.Charts = []report.Chart{chart}
	res.Tables = []report.Table{table}
	res.Notes = []string{
		"Spearman rank correlation is used because neighborhood counts are heavily skewed. " +
			"'Size-adjusted' is a partial Spearman correlation that controls for each neighborhood's total 311 volume " +
			"(a stand-in for population, which is not in these datasets). Bars drawn with a lighter glyph are negative.",
		fmt.Sprintf("Only request types with at least %s requests are ranked. 311 call-centre information rows (ECC) are excluded. "+
			"These are associations to investigate, not proof that one problem causes crime.", commas(threshold)),
	}
	return res
}
