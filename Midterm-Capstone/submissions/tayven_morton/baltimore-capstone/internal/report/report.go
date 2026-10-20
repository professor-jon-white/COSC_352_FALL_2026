// Package report handles presentation, keeping analysis separate from formatting.
package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"baltimore-capstone/internal/analysis"
	"baltimore-capstone/internal/model"
)

func bar(value, max int) string {
	if max <= 0 || value <= 0 {
		return ""
	}
	n := int(math.Round(float64(value) / float64(max) * 30))
	if n < 1 {
		n = 1
	}
	return strings.Repeat("#", n)
}
func percentage(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(n) / float64(total)
}

func Write(w io.Writer, rows []analysis.Row, rq, vq model.Quality) error {
	// Buffer report so writer failures are propagated at the boundary.
	var b strings.Builder
	fmt.Fprintln(&b, "# Baltimore Neighborhood Service and Vacancy Analyzer")
	fmt.Fprintln(&b, "\n311 period: October 1 through December 31, 2025 (UTC). Vacancy data is a later open-notice snapshot; see data/manifest.json for extraction time.")
	fmt.Fprintf(&b, "\nAccepted requests: %d. Distinct vacancy properties with neighborhood: %d.\n", rq.Accepted, vq.Accepted)
	fmt.Fprintf(&b, "\nImportant coverage warning: %d of %d raw requests (%.2f%%) lack a neighborhood and are excluded rather than assigned a guessed location. This can bias comparisons.\n", rq.MissingNeighborhood, rq.Lines, percentage(rq.MissingNeighborhood, rq.Lines))
	fmt.Fprintln(&b, "\nSources: [Baltimore 311 requests]("+model.RequestURL+") and [Open vacant-building notices]("+model.VacancyURL+"). All findings below are computed from the bundled extracts of these sources.")
	matched := []analysis.Row{}
	unmatched := []string{}
	for _, r := range rows {
		if r.Matched {
			matched = append(matched, r)
		} else {
			unmatched = append(unmatched, fmt.Sprintf("%s (%d requests; %d vacancy properties)", r.Name, r.Requests.Count, r.Vacancies))
		}
	}
	fmt.Fprintf(&b, "\nAnalysis universe: %d neighborhoods appearing in both datasets; %d dataset-only neighborhoods excluded from comparisons. Zero notice rows do not establish zero real-world vacancy.\n", len(matched), len(unmatched))

	fmt.Fprintln(&b, "\n## Q1: Where are both request volume and vacancy counts concentrated?")
	fmt.Fprintln(&b, "\nThe following neighborhoods are at or above the 75th-percentile midrank on both measures. This is a descriptive count comparison, not a population-adjusted rate.")
	var vc, rc []int
	for _, r := range matched {
		vc = append(vc, r.Vacancies)
		rc = append(rc, r.Requests.Count)
	}
	sort.Ints(vc)
	sort.Ints(rc)
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Score == matched[j].Score {
			return matched[i].Name < matched[j].Name
		}
		return matched[i].Score > matched[j].Score
	})
	fmt.Fprintln(&b, "\n| Neighborhood | Requests | Vacancy properties |\n|---|---:|---:|")
	found := 0
	for _, r := range matched {
		if analysis.Percentile(vc, r.Vacancies) >= 75 && analysis.Percentile(rc, r.Requests.Count) >= 75 {
			fmt.Fprintf(&b, "| %s | %d | %d |\n", r.Name, r.Requests.Count, r.Vacancies)
			found++
		}
	}
	if found == 0 {
		fmt.Fprintln(&b, "\nNo neighborhoods meet both thresholds.")
	}

	low, nl := analysis.Cohort(rows, "low")
	high, nh := analysis.Cohort(rows, "high")
	fmt.Fprintln(&b, "\n## Q2: How does the mix of service needs differ?")
	fmt.Fprintf(&b, "\nLow-count cohort: %d neighborhoods, %d requests. High-count cohort: %d neighborhoods, %d requests. Cohorts use lower/upper vacancy-count thirds; ties stay together, so group sizes can differ.\n", nl, low.Count, nh, high.Count)
	if len(vc) >= 3 {
		fmt.Fprintf(&b, "\nSnapshot cutoffs: low = at most %d vacancy properties; high = more than %d vacancy properties. Both cohorts are restricted to matched neighborhoods.\n", vc[(len(vc)-1)/3], vc[2*(len(vc)-1)/3])
	}
	fmt.Fprintln(&b, "\nTop five request types in each cohort (share of all requests in that cohort):")
	for _, group := range []struct {
		name string
		s    *model.Stats
	}{{"Low", low}, {"High", high}} {
		fmt.Fprintf(&b, "\n### %s vacancy-count cohort\n\n| Type | Requests | Share |\n|---|---:|---:|\n", group.name)
		for _, p := range analysis.Top(group.s.Types, 5) {
			fmt.Fprintf(&b, "| %s | %d | %.2f%% |\n", p.Name, p.Count, percentage(p.Count, group.s.Count))
		}
	}
	if nl == 0 || nh == 0 {
		fmt.Fprintln(&b, "\nInsufficient variation to form both comparison groups.")
	}

	fmt.Fprintln(&b, "\n## Q3: Does recorded closure time differ for comparable requests?")
	fmt.Fprintln(&b, "\nOnly requests with a status beginning CLOSED, non-null dates and nonnegative duration qualify. Transferred closures are included, so this is NOT time to successful resolution. Open cases are excluded and may bias these medians downward. Comparison requires at least 20 qualifying closures per type in EACH cohort.")
	fmt.Fprintln(&b, "\n| Request type | Low n | Low median days | High n | High median days | High minus low days |\n|---|---:|---:|---:|---:|---:|")
	comparable := map[string]int{}
	for typ, a := range low.Durations {
		if len(a) >= 20 && len(high.Durations[typ]) >= 20 {
			comparable[typ] = len(a) + len(high.Durations[typ])
		}
	}
	for _, p := range analysis.Top(comparable, 10) {
		a, c := low.Durations[p.Name], high.Durations[p.Name]
		l, _ := analysis.Median(a)
		h, _ := analysis.Median(c)
		fmt.Fprintf(&b, "| %s | %d | %.2f | %d | %.2f | %+.2f |\n", p.Name, len(a), l, len(c), h, h-l)
	}
	if len(comparable) == 0 {
		fmt.Fprintln(&b, "\nNo request types meet the minimum sample rule.")
	}
	fmt.Fprintln(&b, "\nUp to ten most frequent eligible types are shown. No statistical-significance or causal claim is made.")

	fmt.Fprintln(&b, "\n## Q4: How do monthly request volumes change by vacancy cohort?")
	fmt.Fprintln(&b, "\nTotals describe cohort workload; requests per neighborhood adjust for differing group sizes, not population.")
	fmt.Fprintln(&b, "\n| Month | Low total | Low per neighborhood | High total | High per neighborhood |\n|---|---:|---:|---:|---:|")
	max := 0
	for _, s := range []*model.Stats{low, high} {
		for _, n := range s.Months {
			if n > max {
				max = n
			}
		}
	}
	for _, m := range []string{"2025-10", "2025-11", "2025-12"} {
		l, h := low.Months[m], high.Months[m]
		lp, hp := "N/A", "N/A"
		if nl > 0 {
			lp = fmt.Sprintf("%.1f", float64(l)/float64(nl))
		}
		if nh > 0 {
			hp = fmt.Sprintf("%.1f", float64(h)/float64(nh))
		}
		fmt.Fprintf(&b, "| %s | %d | %s | %d | %s |\n", m, l, lp, h, hp)
	}
	fmt.Fprintln(&b, "\n```text")
	for _, m := range []string{"2025-10", "2025-11", "2025-12"} {
		fmt.Fprintf(&b, "%s Low  %-30s %d\n", m, bar(low.Months[m], max), low.Months[m])
		fmt.Fprintf(&b, "%s High %-30s %d\n", m, bar(high.Months[m], max), high.Months[m])
	}
	fmt.Fprintln(&b, "```")
	for _, g := range []struct {
		name string
		s    *model.Stats
	}{{"Low", low}, {"High", high}} {
		first, last := g.s.Months["2025-10"], g.s.Months["2025-12"]
		if first > 0 {
			fmt.Fprintf(&b, "\n%s cohort October-to-December change: %+.2f%%.\n", g.name, 100*float64(last-first)/float64(first))
		}
	}

	fmt.Fprintln(&b, "\n## Q5: Which neighborhoods should staff review first?")
	fmt.Fprintln(&b, "\nScreening score = 50% vacancy-count percentile + 50% request-count percentile, among matched neighborhoods. Equal values receive equal midrank percentiles; alphabetic order breaks display ties. The score is a student-designed review aid, not an official funding formula.")
	fmt.Fprintln(&b, "\n| Neighborhood | Score / 100 | Requests | Vacancy properties | Top request type |\n|---|---:|---:|---:|---|")
	for i, r := range matched {
		if i == 5 {
			break
		}
		typ := "N/A"
		if p := analysis.Top(r.Requests.Types, 1); len(p) > 0 {
			typ = p[0].Name
		}
		fmt.Fprintf(&b, "| %s | %.2f | %d | %d | %s |\n", r.Name, r.Score, r.Requests.Count, r.Vacancies, typ)
	}
	fmt.Fprintln(&b, "\nSuggested action: review the listed neighborhoods jointly with housing and the agencies responsible for their dominant request categories. Validate conditions on the ground before assigning resources.")

	fmt.Fprintln(&b, "\n## Data quality and limits")
	for _, d := range []struct {
		name string
		q    model.Quality
	}{{"311", rq}, {"Vacancies", vq}} {
		q := d.q
		fmt.Fprintf(&b, "\n%s: raw=%d; accepted=%d; malformed=%d; missing ID=%d; duplicate ID/property=%d; missing neighborhood=%d; missing creation date=%d; outside period=%d; deleted=%d; present but unusable closure=%d.\n", d.name, q.Lines, q.Accepted, q.InvalidJSON, q.MissingID, q.Duplicates, q.MissingNeighborhood, q.MissingCreated, q.OutsidePeriod, q.Deleted, q.InvalidClosure)
	}
	fmt.Fprintln(&b, "\nExclusion counts are mutually exclusive first-failure counts, except unusable closure (an accepted-record flag). Vacancy deduplication uses parcel ID with address fallback; requests use service-request number. Missing fields, reporting behavior, property size, population and neighborhood spelling can affect comparisons.")
	fmt.Fprintln(&b, "\nThese data cannot establish causation. The later vacancy snapshot is not a reconstruction of 2025 vacancies. No population or total-property denominator is available in these two inputs. Current closure status may reflect updates after the request period. Dataset-only neighborhoods are excluded rather than assumed to have zero demand or vacancy.")
	fmt.Fprintln(&b, "\nRequests can include proactive agency activity, not just resident complaints. Their volume is a workload indicator, not a direct count of unmet needs.")
	fmt.Fprintln(&b, "\nDataset-only neighborhoods:")
	for _, name := range unmatched {
		fmt.Fprintln(&b, "- "+name)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func CSV(w io.Writer, rows []analysis.Row) error {
	c := csv.NewWriter(w)
	if err := c.Write([]string{"neighborhood", "requests", "vacancy_properties", "matched", "cohort", "priority_score"}); err != nil {
		return err
	}
	for _, r := range rows {
		score := ""
		if r.Matched {
			score = fmt.Sprintf("%.6f", r.Score)
		}
		if err := c.Write([]string{r.Name, strconv.Itoa(r.Requests.Count), strconv.Itoa(r.Vacancies), strconv.FormatBool(r.Matched), r.Group, score}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
