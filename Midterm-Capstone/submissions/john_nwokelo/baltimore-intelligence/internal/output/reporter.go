package output

import (
	"baltimore-intelligence/internal/analysis"
	"fmt"
	"io"
	"sort"
	"strings"
)

func Report(w io.Writer, stats []analysis.Stats, processed, unmatched int) {
	fmt.Fprintln(w, "BALTIMORE CITY SERVICE REPORT")
	fmt.Fprintf(w, "Requests processed: %d | Geographically unmatched: %d | Neighborhoods with matches: %d\n", processed, unmatched, len(stats))
	if len(stats) == 0 {
		fmt.Fprintln(w, "No matched requests. Check coordinates and boundary dataset.")
		return
	}
	rank := func(less func(a, b analysis.Stats) bool) []analysis.Stats {
		a := append([]analysis.Stats(nil), stats...)
		sort.Slice(a, func(i, j int) bool {
			if less(a[i], a[j]) {
				return true
			}
			if less(a[j], a[i]) {
				return false
			}
			return a[i].Name < a[j].Name
		})
		return a
	}
	limit := func(a []analysis.Stats) []analysis.Stats {
		if len(a) > 10 {
			return a[:10]
		}
		return a
	}
	fmt.Fprintln(w, "\nQ1. WHICH NEIGHBORHOODS REPORT THE MOST PROBLEMS?")
	for _, s := range limit(rank(func(a, b analysis.Stats) bool { return a.Total > b.Total })) {
		fmt.Fprintf(w, "%-30.30s %5d %s\n", s.Name, s.Total, strings.Repeat("#", 1+s.Total*35/max(1, statsMax(stats))))
	}
	fmt.Fprintln(w, "\nQ2. WHAT PROBLEMS ARE MOST COMMON IN EACH NEIGHBORHOOD?")
	for _, s := range limit(rank(func(a, b analysis.Stats) bool { return a.Total > b.Total })) {
		name, count := s.Common()
		fmt.Fprintf(w, "%-30.30s %-35.35s %d\n", s.Name, name, count)
	}
	fmt.Fprintln(w, "\nQ3. WHERE IS THE HIGHEST PERCENTAGE OF PROBLEMS STILL UNRESOLVED?")
	for _, s := range limit(rank(func(a, b analysis.Stats) bool { return a.Rate() > b.Rate() })) {
		fmt.Fprintf(w, "%-30.30s %6.1f%% (%d/%d)\n", s.Name, s.Rate(), s.Open, s.Total)
	}
	fmt.Fprintln(w, "\nQ4. WHERE DOES IT TAKE THE LONGEST TO FIX PROBLEMS?")
	closedTotal, datedClosedTotal := 0, 0
	for _, s := range stats {
		closedTotal += s.Closed
		datedClosedTotal += s.DurationN
	}
	fmt.Fprintf(w, "Closed requests with valid creation and closure dates: %d / %d\n", datedClosedTotal, closedTotal)
	found := 0
	for _, s := range rank(func(a, b analysis.Stats) bool { return a.AvgDays() > b.AvgDays() }) {
		if s.DurationN == 0 {
			continue
		}
		fmt.Fprintf(w, "%-30.30s %7.1f days (n=%d)\n", s.Name, s.AvgDays(), s.DurationN)
		found++
		if found == 10 {
			break
		}
	}
	if found == 0 {
		fmt.Fprintln(w, "No qualifying closed records with valid timestamps")
	}
	fmt.Fprintln(w, "\nQ5. WHICH NEIGHBORHOODS NEED THE MOST ATTENTION?")
	fmt.Fprintln(w, "Higher scores suggest which areas to review first (not proof of greater need).")
	for _, s := range limit(rank(func(a, b analysis.Stats) bool { return a.Priority > b.Priority })) {
		fmt.Fprintf(w, "%-30.30s %5.1f / 100\n", s.Name, s.Priority)
	}
	if datedClosedTotal == 0 {
		fmt.Fprintln(w, "\nPriority = 50% relative volume + 50% unresolved rate (no valid closure intervals in input).")
		fmt.Fprintln(w, "WARNING: Q4 could not be answered. Verify date field names and formats in the downloaded requests GeoJSON.")
	} else {
		fmt.Fprintln(w, "\nPriority = 40% relative volume + 40% unresolved rate + 20% relative mean closure time.")
		if datedClosedTotal < closedTotal {
			fmt.Fprintln(w, "WARNING: Some closed requests lack valid dates, so resolution-time rankings may be biased.")
		}
	}
	fmt.Fprintln(w, "WARNING: Scores are descriptive, not causal; reporting access and population differ by area.")
}
func statsMax(a []analysis.Stats) int {
	m := 0
	for _, s := range a {
		if s.Total > m {
			m = s.Total
		}
	}
	return m
}
