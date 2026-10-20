package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func percent(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
func ordered(m map[string]*Cell, mode string) []*Cell {
	a := []*Cell{}
	for _, c := range m {
		a = append(a, c)
	}
	sort.Slice(a, func(i, j int) bool {
		if mode == "month" {
			return a[i].Name < a[j].Name
		}
		x, y := a[i].Crimes, a[j].Crimes
		if mode == "arrests" {
			x, y = a[i].Arrests, a[j].Arrests
		}
		if x == y {
			return a[i].Name < a[j].Name
		}
		return x > y
	})
	return a
}
func safe(s string) string { return strings.NewReplacer("|", "/", "\n", " ", "\r", " ").Replace(s) }
func statsTable(b *strings.Builder, m map[string]*Cell, mode string, limit int) {
	a := ordered(m, mode)
	if limit > 0 && len(a) > limit {
		a = a[:limit]
	}
	fmt.Fprint(b, "| Group | Crime records | Arrest records |\n|---|---:|---:|\n")
	for _, c := range a {
		fmt.Fprintf(b, "| %s | %d | %d |\n", safe(c.Name), c.Crimes, c.Arrests)
	}
}
func topKnown(r Result, mode string) map[string]bool {
	chosen := map[string]bool{}
	for _, c := range ordered(r.Neighborhoods, mode) {
		count := c.Crimes
		if mode == "arrests" {
			count = c.Arrests
		}
		if c.Name == "UNKNOWN" || count == 0 {
			continue
		}
		chosen[c.Name] = true
		if len(chosen) == 5 {
			break
		}
	}
	return chosen
}
func names(m map[string]bool) string {
	a := []string{}
	for n := range m {
		a = append(a, n)
	}
	sort.Strings(a)
	return strings.Join(a, ", ")
}
func profile(b *strings.Builder, r Result, src map[string]map[string]int, chosen map[string]bool) {
	inside, outside := map[string]int{}, map[string]int{}
	ni, no, ar := 0, 0, 0
	for n, values := range src {
		if n == "UNKNOWN" {
			continue
		}
		for k, v := range values {
			if chosen[n] {
				inside[k] += v
				ni += v
			} else {
				outside[k] += v
				no += v
			}
		}
	}
	for n := range chosen {
		ar += r.Neighborhoods[n].Arrests
	}
	keys := []string{}
	for k := range inside {
		keys = append(keys, k)
	}
	for k := range outside {
		if _, ok := inside[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		x, y := inside[keys[i]]+outside[keys[i]], inside[keys[j]]+outside[keys[j]]
		if x == y {
			return keys[i] < keys[j]
		}
		return x > y
	})
	fmt.Fprintf(b, "Selected neighborhoods: %s. Their arrest records: %d. Crime rows with known neighborhood: %d selected, %d elsewhere.\n\n", names(chosen), ar, ni, no)
	fmt.Fprint(b, "| Source category | Selected crime rows | Selected mix | Elsewhere crime rows | Elsewhere mix |\n|---|---:|---:|---:|---:|\n")
	for i, k := range keys {
		if i == 15 {
			break
		}
		fmt.Fprintf(b, "| %s | %d | %.1f%% | %d | %.1f%% |\n", safe(k), inside[k], percent(inside[k], ni), outside[k], percent(outside[k], no))
	}
}
func report(r Result, m Manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Baltimore Public Safety: Crime and Arrest Workload\n\nGenerated: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	if m.Demo {
		fmt.Fprint(&b, "**SYNTHETIC DEMO: invented records, not Baltimore findings.**\n\n")
	} else {
		fmt.Fprintf(&b, "Snapshot retrieved: %s. Source query interval: %s inclusive to %s exclusive.\n\nCrime rows: %d of %d available (truncated=%t). Arrest rows: %d of %d available (truncated=%t). Positive limits select a non-random source-ID prefix.\n\n", m.Retrieved, m.Start, m.End, m.Crime.Rows, m.Crime.Available, m.Crime.Truncated, m.Arrest.Rows, m.Arrest.Available, m.Arrest.Truncated)
	}
	fmt.Fprintf(&b, "Inspired by the [Public Safety Accountability Dashboard](%s), using two raw Open Baltimore datasets. Independent class project; its definitions and totals do not reproduce the official dashboard.\n\n", inspiration)
	fmt.Fprintf(&b, "**Coverage:** %d crime rows, %d distinct nonempty crime incident IDs; %d arrest rows, %d distinct nonempty arrest numbers. No person-level or case-level linkage is assumed.\n\n", r.CrimeRows, r.UniqueIncidents, r.ArrestRows, r.UniqueArrests)
	fmt.Fprintf(&b, "**Geographic coverage:** %.1f%% of crime rows and %.1f%% of arrest rows have a neighborhood. Geographic comparisons describe only the geocoded portion; missing arrest neighborhoods can materially change the picture.\n\n", percent(r.CrimeRows-r.MissingCrimeNeighborhood, r.CrimeRows), percent(r.ArrestRows-r.MissingArrestNeighborhood, r.ArrestRows))
	fmt.Fprint(&b, "## 1. Which neighborhoods have the largest recorded crime workload, and how does arrest activity compare?\n\nFor citywide coordination: full outer join of neighborhood aggregates; top 15 by crime rows. The two location fields describe different activities. Counts are records, not unique people or population-adjusted risk.\n\n")
	statsTable(&b, r.Neighborhoods, "crimes", 15)
	fmt.Fprint(&b, "\n## 2. How do monthly crime and arrest workloads change?\n\nFor staffing review: full outer join of America/New_York calendar-month aggregates. The source date filter uses ArcGIS SQL timestamps; month labels use returned Unix timestamps converted to Eastern time. A zero means no downloaded rows in that cell, not absence of real activity.\n\n")
	statsTable(&b, r.Months, "month", 0)
	fmt.Fprint(&b, "\n## 3. What offense mix is recorded in the neighborhoods with the most arrest activity?\n\nFor investigative resource planning: select the five known neighborhoods with the most arrest rows, join to crime by neighborhood, and compare crime Description shares with all other known neighborhoods. This describes contextual workload; arrests are not attributed to these offenses.\n\n")
	profile(&b, r, r.CategoryByNeighborhood, topKnown(r, "arrests"))
	fmt.Fprint(&b, "\n## 4. Where are crime-record and arrest-record shares most different?\n\nFor reviewing geographic workload distribution: compare each neighborhood's share of crime rows with its share of arrest rows, using known-neighborhood rows only. Sort by absolute percentage-point difference. Positive means crime-record share is larger. The gap is not a clearance rate, policing adequacy measure, or performance score.\n\n")
	nc, na := 0, 0
	gs := []*Cell{}
	for n, c := range r.Neighborhoods {
		if n != "UNKNOWN" {
			nc += c.Crimes
			na += c.Arrests
			gs = append(gs, c)
		}
	}
	fmt.Fprintf(&b, "Known-neighborhood denominators: %d crime rows; %d arrest rows.\n\n", nc, na)
	if nc == 0 || na == 0 {
		fmt.Fprint(&b, "Insufficient known-neighborhood records for a share comparison.\n")
	} else {
		gap := func(c *Cell) float64 { return percent(c.Crimes, nc) - percent(c.Arrests, na) }
		sort.Slice(gs, func(i, j int) bool {
			x, y := math.Abs(gap(gs[i])), math.Abs(gap(gs[j]))
			if x == y {
				return gs[i].Name < gs[j].Name
			}
			return x > y
		})
		fmt.Fprint(&b, "| Neighborhood | Crime share | Arrest share | Difference (points) |\n|---|---:|---:|---:|\n")
		for i, c := range gs {
			if i == 10 {
				break
			}
			fmt.Fprintf(&b, "| %s | %.1f%% | %.1f%% | %+.1f |\n", safe(c.Name), percent(c.Crimes, nc), percent(c.Arrests, na), gap(c))
		}
		fmt.Fprint(&b, "\n```text\nShare difference: + crime share larger; - arrest share larger\nOne # represents approximately one percentage point.\n")
		for i, c := range gs {
			if i == 10 {
				break
			}
			sign := "+"
			if gap(c) < 0 {
				sign = "-"
			}
			bars := int(math.Round(math.Abs(gap(c))))
			fmt.Fprintf(&b, "%-26s %+.1f %s%s\n", c.Name, gap(c), sign, strings.Repeat("#", bars))
		}
		fmt.Fprint(&b, "```\n")
	}
	fmt.Fprint(&b, "\n## 5. What weapon profiles and arrest workload occur in neighborhoods with the most crime records?\n\nFor violence-prevention program planning: select the five known neighborhoods with the most crime rows, join their arrest totals by neighborhood, and compare source Weapon shares with other known neighborhoods. Weapon labels are not inferred or collapsed into a firearm classification. The geographic association does not link an arrest to a weapon event.\n\n")
	profile(&b, r, r.WeaponByNeighborhood, topKnown(r, "crimes"))
	fmt.Fprintf(&b, "\n## Data quality and interpretation\n\n| Check | Crime rows | Arrest rows |\n|---|---:|---:|\n| Missing incident ID | %d | %d |\n| Missing neighborhood | %d | %d |\n| Invalid/missing date | %d | %d |\n\n", r.MissingCrimeID, r.MissingArrestID, r.MissingCrimeNeighborhood, r.MissingArrestNeighborhood, r.MissingCrimeDate, r.MissingArrestDate)
	fmt.Fprintf(&b, "Exact normalized ID overlaps found: %d. This is a diagnostic only: the sources use different identifier conventions and many arrest IDs are absent. All five analyses use aggregate neighborhood or time joins.\n\n", r.MatchedIncidents)
	fmt.Fprint(&b, "Neighborhood names are uppercased and whitespace-normalized. No spatial or fuzzy joins are inferred. UNKNOWN neighborhoods appear in overall workload tables but are excluded from geographic selections and share denominators. Crime rows may represent multiple offenses or victims, and arrest rows multiple charges; neither is a count of unique people. Arrest geography is not necessarily crime geography. Missing neighborhoods and different source coverage strongly limit comparisons. Source categories and time boundaries need review before policy use. These are descriptive workload summaries, not causal effects, official clearance rates, or population-adjusted risk measures. Records can change during pagination. BPD warns of reporting-system translation issues from May 2021; consult its documentation.\n\n")
	fmt.Fprintf(&b, "## Sources\n\n- Inspiration: %s\n- Crime: %s\n- Arrests: %s\n- BPD definitions and reporting caveat: https://www.baltimorepolice.org/crime-stats/open-data\n", inspiration, m.Crime.URL, m.Arrest.URL)
	return b.String()
}
