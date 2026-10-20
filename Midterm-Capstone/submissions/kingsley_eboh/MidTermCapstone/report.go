package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func renderReport(a Analysis) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Baltimore Civic Pulse: 311 Requests × Vacant Building Rehab Locations\n\n")
	fmt.Fprintf(&b, "Generated: %s  \nMode: %s  \nParsing workers: %d  \nElapsed: %s\n\n", a.GeneratedAt, mode(a.OfflineExample, a.RawReplay), a.Workers, a.Elapsed)
	fmt.Fprintf(&b, "## Executive summary\n\nThe pipeline processed **%s 311 service requests with a neighborhood label** and **%s vacant-building rehab locations**. It matched records by Baltimore neighborhood name across **%d neighborhoods**. The neighborhood-level Spearman rank correlation between 2026 request volume and rehab locations is **%.3f** (%s). This is a descriptive comparison of two administrative datasets, not evidence that rehab activity causes service requests or vice versa.\n\n", comma(a.RequestCount), comma(a.RehabCount), a.MatchedNeighborhoods, a.Spearman, a.SpearmanLabel)
	fmt.Fprintf(&b, "## Five stakeholder questions\n\n")
	fmt.Fprintf(&b, "### 1. Which neighborhoods show the strongest combined signal of service demand and rehab locations?\n\nThe table ranks neighborhoods by the sum of their standardized 311 and rehab-location counts. It highlights places high on both city measures; it is not a per-resident rate.\n\n")
	writeNeighborhoodTable(&b, firstN(a.Neighborhoods, 5))
	fmt.Fprintf(&b, "### 2. Where are the largest concentrations of vacant-building rehab locations?\n\n")
	vac := append([]NeighborhoodSummary(nil), a.Neighborhoods...)
	sort.Slice(vac, func(i, j int) bool { return vac[i].RehabSites > vac[j].RehabSites })
	writeNeighborhoodTable(&b, firstN(vac, 5))
	fmt.Fprintf(&b, "### 3. Which neighborhoods have the most non-closed 311 requests?\n\nNon-closed means a request with a recorded status that is not marked closed, complete, resolved, or cancelled. These counts are a workload snapshot; age, service type, and agency-specific completion rules are not adjusted.\n\n")
	pending := append([]NeighborhoodSummary(nil), a.Neighborhoods...)
	sort.Slice(pending, func(i, j int) bool { return pending[i].Unresolved > pending[j].Unresolved })
	writeNeighborhoodTable(&b, firstN(pending, 5))
	fmt.Fprintf(&b, "### 4. Which 311 request types are more common in high-rehab neighborhoods?\n\nHigh-rehab neighborhoods are the top quartile among neighborhoods with at least one rehab location (cutoff: **%d rehab locations**). Enrichment compares each request type's share in that group with its share in other matched neighborhoods; values above 1 mean relatively more common. Low counts can produce unstable ratios.\n\n", a.HighRehabCutoff)
	if len(a.EnrichedTypes) == 0 {
		b.WriteString("Not enough matched request types in the high-rehab group to calculate enrichment.\n\n")
	} else {
		fmt.Fprintf(&b, "| Request type | Requests in high-rehab areas | Share of those requests | Enrichment vs. other areas |\n|---|---:|---:|---:|\n")
		for _, t := range a.EnrichedTypes {
			fmt.Fprintf(&b, "| %s | %s | %.1f%% | %.2fx |\n", escapeCell(t.Name), comma(t.Count), t.Share, t.Enrichment)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "### 5. Do the two indicators tend to rank neighborhoods similarly?\n\nSpearman rank correlation = **%.3f** (%s), calculated over %d neighborhoods with both measures. A positive association means that neighborhoods ranked higher on one measure tend to rank higher on the other; it does not explain why.\n\n", a.Spearman, a.SpearmanLabel, a.MatchedNeighborhoods)
	fmt.Fprintf(&b, "## Visual summaries\n\n### Most 311 requests by neighborhood (2026)\n\n```text\n")
	for _, n := range topBy(a.Neighborhoods, func(x NeighborhoodSummary) int { return x.Requests }, 10) {
		fmt.Fprintf(&b, "%-24s %6d %s\n", clip(n.Name, 24), n.Requests, bar(n.Requests, maxInt(a.Neighborhoods, func(x NeighborhoodSummary) int { return x.Requests }), 30))
	}
	b.WriteString("```\n\n### Most vacant-building rehab locations\n\n```text\n")
	for _, n := range topBy(a.Neighborhoods, func(x NeighborhoodSummary) int { return x.RehabSites }, 10) {
		fmt.Fprintf(&b, "%-24s %6d %s\n", clip(n.Name, 24), n.RehabSites, bar(n.RehabSites, maxInt(a.Neighborhoods, func(x NeighborhoodSummary) int { return x.RehabSites }), 30))
	}
	b.WriteString("```\n\n")
	fmt.Fprintf(&b, "## Data, method, and limitations\n\n- **311 source:** [Baltimore City 311 Customer Service Requests 2026](https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0). The city describes this as 2026 requests, updated daily. The query filters to records with a neighborhood value.\n- **Housing source:** [Vacant Building Rehabs](https://data.baltimorecity.gov/datasets/baltimore::vacant-building-rehabs), a citywide inventory of locations where vacant buildings have been rehabilitated.\n- The app saves the paginated raw 311 GeoJSON responses and the raw rehab GeoJSON snapshot under `data/raw/`; its parser joins on normalized neighborhood names. Missing or unmatched labels are excluded from neighborhood-level comparisons.\n- The feeds cover different time windows: 311 records are for calendar year 2026; rehab locations span the years included in the city inventory.\n- The comparison is not population-normalized, does not imply property-level matches, and may reflect reporting access or data quality. Rehab records do not establish whether a building remains vacant today or whether a project is complete. Use the output to frame follow-up analysis, not as a stand-alone basis for resource allocation.\n- The 311 “non-closed” count uses the recorded status field and a broad terminal-status rule; confirm against agency definitions before operational use.\n\n")
	fmt.Fprintf(&b, "## Reproduce\n\nFrom this directory, run `go run .` to fetch current records, save raw pages, and write this report, `summary.json`, and `neighborhoods.csv` to `reports/`. For the bundled raw excerpt, run `go run . -offline`.\n")
	return b.String()
}
func writeNeighborhoodTable(b *strings.Builder, rows []NeighborhoodSummary) {
	if len(rows) == 0 {
		b.WriteString("No records available.\n\n")
		return
	}
	fmt.Fprintf(b, "| Neighborhood | 311 requests | Rehab locations | Non-closed 311 | Requests per 100 rehab locations |\n|---|---:|---:|---:|---:|\n")
	for _, n := range rows {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %.1f |\n", escapeCell(n.Name), comma(n.Requests), comma(n.RehabSites), comma(n.Unresolved), n.RequestsPer100Rehabs)
	}
	b.WriteString("\n")
}
func mode(offline, replay bool) string {
	if offline {
		return "Bundled sample excerpt"
	}
	if replay {
		return "Saved Open Baltimore raw data"
	}
	return "Live Open Baltimore sources"
}
func escapeCell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
}
func topBy(rows []NeighborhoodSummary, value func(NeighborhoodSummary) int, n int) []NeighborhoodSummary {
	x := append([]NeighborhoodSummary(nil), rows...)
	sort.Slice(x, func(i, j int) bool { return value(x[i]) > value(x[j]) })
	return firstN(x, n)
}
func maxInt(rows []NeighborhoodSummary, value func(NeighborhoodSummary) int) int {
	m := 0
	for _, r := range rows {
		if value(r) > m {
			m = value(r)
		}
	}
	return m
}
func bar(v, m, n int) string {
	if m == 0 {
		return ""
	}
	count := int(math.Round(float64(v) / float64(m) * float64(n)))
	if count < 1 && v > 0 {
		count = 1
	}
	return strings.Repeat("█", count)
}
