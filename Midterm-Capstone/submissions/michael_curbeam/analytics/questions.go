package analytics

import (
	"fmt"
	"math"
	"sort"

	"baltimore-engine/processor"
)

// Q1: top 5 neighborhoods by combined vacancies and 311 issues.
func Q1(j processor.Joined) Result {
	hoods := sortedHoods(j, func(a, b *processor.NeighborhoodStats) bool {
		return a.Vacant+a.Requests > b.Vacant+b.Requests
	})
	r := Result{
		Question: "Q1: Top 5 neighborhoods by combined vacant notices + 311 requests",
		Headers:  []string{"Rank", "Neighborhood", "Vacant Notices", "311 Requests", "Combined"},
		Note:     "311 counts cover only the most recent records ingested.",
		BarCol:   5,
	}
	for i, n := range hoods[:min(5, len(hoods))] {
		r.Rows = append(r.Rows, []string{
			fmt.Sprint(i + 1), n.Name, fmt.Sprint(n.Vacant), fmt.Sprint(n.Requests), fmt.Sprint(n.Vacant + n.Requests),
		})
	}
	return r
}

// Q2: correlation between vacancies and each 311 category across neighborhoods.
func Q2(j processor.Joined) Result {
	cats := []string{"Sanitation/Dumping", "Housing Code", "Transportation/Streets", "Water/Sewer", "Other"}
	hoods := sortedHoods(j, func(a, b *processor.NeighborhoodStats) bool { return a.Name < b.Name })

	vac := make([]float64, len(hoods))
	for i, n := range hoods {
		vac[i] = float64(n.Vacant)
	}

	r := Result{
		Question: "Q2: Correlation of vacancies with 311 request types (Pearson r across neighborhoods)",
		Headers:  []string{"311 Category", "Requests", "Pearson r", "Strength"},
		Note:     "r near +1 = categories rise with vacancy counts; near 0 = no linear relationship.",
	}
	for _, c := range cats {
		ys := make([]float64, len(hoods))
		total := 0
		for i, n := range hoods {
			ys[i] = float64(n.ByCategory[c])
			total += n.ByCategory[c]
		}
		rv := pearson(vac, ys)
		r.Rows = append(r.Rows, []string{c, fmt.Sprint(total), fmtR(rv), strength(rv)})
	}
	return r
}

func fmtR(r float64) string {
	if math.IsNaN(r) {
		return "n/a"
	}
	return fmt.Sprintf("%+.3f", r)
}

func strength(r float64) string {
	switch a := math.Abs(r); {
	case math.IsNaN(r):
		return "n/a"
	case a >= 0.7:
		return "strong"
	case a >= 0.4:
		return "moderate"
	case a >= 0.2:
		return "weak"
	default:
		return "negligible"
	}
}

// Q3: open vs. closed 311 breakdown in heavy-vacancy neighborhoods
// (top quartile by vacant notices) compared with the rest of the city.
func Q3(j processor.Joined) Result {
	hoods := sortedHoods(j, func(a, b *processor.NeighborhoodStats) bool { return a.Vacant > b.Vacant })
	cut := max(1, len(hoods)/4)
	threshold := 0
	if len(hoods) > 0 {
		threshold = hoods[cut-1].Vacant
	}

	type agg struct{ hoods, req, open, closed int }
	var heavy, rest agg
	for i, n := range hoods {
		a := &rest
		if i < cut && n.Vacant > 0 {
			a = &heavy
		}
		a.hoods++
		a.req += n.Requests
		a.open += n.Open
		a.closed += n.Closed
	}

	row := func(label string, a agg) []string {
		return []string{label, fmt.Sprint(a.hoods), fmt.Sprint(a.req), fmt.Sprint(a.open), fmt.Sprint(a.closed), pct(a.open, a.req), pct(a.closed, a.req)}
	}
	return Result{
		Question: "Q3: Open vs. closed 311 requests in heavy-vacancy areas",
		Headers:  []string{"Area", "Neighborhoods", "Requests", "Open", "Closed", "% Open", "% Closed"},
		Rows:     [][]string{row("Heavy vacancy", heavy), row("All others", rest)},
		Note:     fmt.Sprintf("Heavy vacancy = top quartile of neighborhoods (>= %d vacant notices). 'Open' includes New/Pending/Hold.", threshold),
	}
}

// Q4: top 3 ZIP codes needing immediate code enforcement. Score = open
// enforcement 311 requests + vacant notices attributed to the ZIP.
func Q4(j processor.Joined) Result {
	zips := make([]*processor.ZipStats, 0, len(j.Zips))
	for _, z := range j.Zips {
		zips = append(zips, z)
	}
	score := func(z *processor.ZipStats) int { return z.OpenEnforcement + z.Vacant }
	sort.Slice(zips, func(a, b int) bool {
		if sa, sb := score(zips[a]), score(zips[b]); sa != sb {
			return sa > sb
		}
		return zips[a].Zip < zips[b].Zip
	})

	r := Result{
		Question: "Q4: Top 3 ZIP codes needing code-enforcement intervention",
		Headers:  []string{"Rank", "ZIP", "Open Enforcement 311", "Vacant Notices", "Score"},
		BarCol:   5,
		Note:     "Score = open code-enforcement requests (HCD, sanitation, dumping) + vacant notices. Vacancies mapped to ZIP via each neighborhood's dominant 311 ZIP.",
	}
	for i, z := range zips[:min(3, len(zips))] {
		r.Rows = append(r.Rows, []string{
			fmt.Sprint(i + 1), z.Zip, fmt.Sprint(z.OpenEnforcement), fmt.Sprint(z.Vacant), fmt.Sprint(score(z)),
		})
	}
	return r
}

// Q5: citywide data-density summary across neighborhoods.
func Q5(j processor.Joined) Result {
	hoods := sortedHoods(j, func(a, b *processor.NeighborhoodStats) bool { return a.Name < b.Name })

	summarize := func(label string, get func(*processor.NeighborhoodStats) int) []string {
		vals := make([]int, len(hoods))
		total, maxIdx, withData := 0, 0, 0
		for i, n := range hoods {
			vals[i] = get(n)
			total += vals[i]
			if vals[i] > 0 {
				withData++
			}
			if vals[i] > vals[maxIdx] {
				maxIdx = i
			}
		}
		if len(hoods) == 0 {
			return []string{label, "0", "0", "0", "0", "0", "n/a"}
		}
		sort.Ints(vals)
		median := float64(vals[len(vals)/2])
		if len(vals)%2 == 0 {
			median = float64(vals[len(vals)/2-1]+vals[len(vals)/2]) / 2
		}
		return []string{
			label, fmt.Sprint(total), fmt.Sprintf("%.1f", float64(total)/float64(len(hoods))),
			fmt.Sprintf("%.1f", median), fmt.Sprint(vals[len(vals)-1]), fmt.Sprint(withData), hoods[maxIdx].Name,
		}
	}

	return Result{
		Question: fmt.Sprintf("Q5: Data density across %d neighborhoods (resource planning)", len(hoods)),
		Headers:  []string{"Metric", "Total", "Mean/Hood", "Median", "Max", "Hoods w/ Data", "Densest"},
		Rows: [][]string{
			summarize("311 requests", func(n *processor.NeighborhoodStats) int { return n.Requests }),
			summarize("Open 311", func(n *processor.NeighborhoodStats) int { return n.Open }),
			summarize("Vacant notices", func(n *processor.NeighborhoodStats) int { return n.Vacant }),
		},
	}
}
