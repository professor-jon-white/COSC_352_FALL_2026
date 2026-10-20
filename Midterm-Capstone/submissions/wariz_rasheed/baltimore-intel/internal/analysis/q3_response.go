package analysis

import (
	"fmt"
	"sort"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
	"baltimoreintel/internal/stats"
)

// ResponseEquity (311 x crime): do high-crime neighborhoods wait longer for
// 311 service?
type ResponseEquity struct{}

// ID implements Analysis.
func (ResponseEquity) ID() string { return "Q3" }

// Run implements Analysis.
func (ResponseEquity) Run(d *model.Dataset) report.Result {
	res := report.Result{
		ID:       "Q3",
		Title:    "Do high-crime neighborhoods wait longer for 311 service?",
		Audience: "Mayor's Office (equity of city services)",
		Question: "Neighborhoods are split into four groups by their level of violent crime. " +
			"How long does it take the city to close a 311 request in each group, and how many requests are still open?",
	}
	var rows []hoodRow
	for _, h := range universe(d) {
		if h.enoughClosed() {
			rows = append(rows, h)
		}
	}
	if len(rows) < 8 {
		res.Finding = fmt.Sprintf("Only %d neighborhoods have both enough crime records and at least %d completed 311 requests; "+
			"at least 8 are needed to form four groups.", len(rows), minClosed)
		return res
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Violent != rows[j].Violent {
			return rows[i].Violent < rows[j].Violent
		}
		return rows[i].Name < rows[j].Name
	})

	labels := []string{"Q1 lowest violent crime", "Q2", "Q3", "Q4 highest violent crime"}
	n := len(rows)
	medians := make([]float64, 4)
	chart := report.Chart{Title: "Median days to close a 311 request, by violent-crime group"}
	table := report.Table{
		Title:   "311 service by neighborhood violent-crime group",
		Headers: []string{"Group", "Neighborhoods", "Avg violent crimes", "Completed requests", "Median days to close", "Requests still open"},
	}
	for q := 0; q < 4; q++ {
		group := rows[q*n/4 : (q+1)*n/4]
		var hours []float64
		violent, requests, open := 0, 0, 0
		for _, h := range group {
			hours = append(hours, h.CloseHours...)
			violent += h.Violent
			requests += h.Requests
			open += h.Open
		}
		medians[q] = stats.Median(hours) / 24
		avgViolent := 0.0
		if len(group) > 0 {
			avgViolent = float64(violent) / float64(len(group))
		}
		chart.Bars = append(chart.Bars, report.Bar{
			Label: labels[q],
			Value: medians[q],
			Text:  f1(medians[q]) + " days",
		})
		table.Rows = append(table.Rows, []string{
			labels[q], fmt.Sprintf("%d", len(group)), f1(avgViolent), commas(len(hours)),
			f1(medians[q]), pct(open, requests),
		})
	}

	violent := make([]float64, n)
	days := make([]float64, n)
	for i, h := range rows {
		violent[i] = float64(h.Violent)
		days[i] = stats.Median(h.CloseHours) / 24
	}
	rho := stats.Spearman(violent, days)

	// The ten slowest neighborhoods.
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		if days[order[a]] != days[order[b]] {
			return days[order[a]] > days[order[b]]
		}
		return rows[order[a]].Name < rows[order[b]].Name
	})
	if len(order) > 10 {
		order = order[:10]
	}
	slow := report.Table{
		Title:   "Ten neighborhoods with the slowest median 311 close time",
		Headers: []string{"Neighborhood", "Completed requests", "Median days to close", "Requests still open", "Violent crimes"},
	}
	for _, i := range order {
		h := rows[i]
		slow.Rows = append(slow.Rows, []string{
			pretty(h.Name), commas(len(h.CloseHours)), f1(days[i]), pct(h.Open, h.Requests), commas(h.Violent),
		})
	}

	res.Finding = fmt.Sprintf("Neighborhoods in the highest violent-crime group close 311 requests in a median of %s days, "+
		"versus %s days in the lowest group. Across %d neighborhoods, violent crime and median close time show a %s (rho = %s).",
		f1(medians[3]), f1(medians[0]), n, stats.Describe(rho), f2(rho))
	res.Charts = []report.Chart{chart}
	res.Tables = []report.Table{table, slow}
	res.Notes = []string{
		"Close time counts only completed requests, so requests that are still open are not in the median; " +
			"the 'still open' column shows that backlog separately.",
		fmt.Sprintf("Only neighborhoods with at least %d completed requests are included. 311 call-centre information rows (ECC) are excluded because they close instantly.", minClosed),
		"A neighborhood's mix of request types affects its median (a bulk-trash pickup is scheduled, a pothole is not), so this shows a gap in outcomes rather than its cause.",
	}
	return res
}
