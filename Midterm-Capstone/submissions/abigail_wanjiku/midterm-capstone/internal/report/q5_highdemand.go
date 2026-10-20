package report

import (
	"fmt"
	"io"
	"strings"

	"capstone/internal/model"
	"capstone/internal/stats"
)

// HighDemandLowCrime lists neighborhoods with heavy 311 use but little crime.
//
// "High 311 volume" means resident-reported requests (city-initiated
// Proactive work excluded) at or above the 75th percentile across
// neighborhoods; "low crime" means at or below the median. The list is
// sorted by 311 volume and shows requests per crime and the most common
// request type, which hints at why the neighborhood calls 311 so much.
//
// Because 311 volume and crime rise together strongly, few neighborhoods pass
// both cutoffs, so the report also ranks busy neighborhoods (at least median
// 311 volume) by resident requests per crime.
type HighDemandLowCrime struct{}

func (HighDemandLowCrime) ID() string { return "high-311-low-crime" }
func (HighDemandLowCrime) Question() string {
	return "Which neighborhoods have high 311 volume but low crime?"
}

func (HighDemandLowCrime) Render(w io.Writer, in *Input) error {
	rs := rows(in,
		func(s *model.Summary) float64 { return float64(residentRequests(s)) },
		func(s *model.Summary) float64 { return float64(s.CrimeCount) },
	)
	p75 := stats.Percentile(xs(rs), 75)
	median := stats.Percentile(ys(rs), 50)

	hits := make([]row, 0, len(rs))
	for _, r := range sortedBy(rs, byX) {
		if r.x >= p75 && r.y <= median {
			hits = append(hits, r)
		}
	}
	fmt.Fprintf(w, "Strict test: resident 311 requests >= %s (75th percentile) and crimes <= %s (median).\n\n", num(int(p75)), num(int(median)))
	if len(hits) == 0 {
		fmt.Fprintln(w, "  (no neighborhood meets both thresholds)")
	} else if err := demandTable(w, hits); err != nil {
		return err
	}

	// 311 and crime rise together so strongly that few neighborhoods pass
	// both cutoffs. Ranking by requests per crime, among neighborhoods with
	// at least median 311 volume, shows the broader pattern.
	p50 := stats.Percentile(xs(rs), 50)
	busy := make([]row, 0, len(rs))
	for _, r := range rs {
		if r.x >= p50 {
			busy = append(busy, r)
		}
	}
	perCrime := func(r row) float64 { return r.x / max(r.y, 1) }
	byRatio := sortedBy(busy, perCrime)[:min(10, len(busy))]
	fmt.Fprintf(w, "\nBroader view: most 311 requests per crime, among neighborhoods with >= %s resident requests (median).\n\n", num(int(p50)))
	if err := demandTable(w, byRatio); err != nil {
		return err
	}

	names := make([]string, len(hits))
	for i, r := range hits {
		names[i] = r.name
	}
	fmt.Fprintf(w, "\nAnswer: %d of %d neighborhoods pass the strict test", len(hits), len(rs))
	if len(hits) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(names, ", "))
	}
	fmt.Fprintf(w, ".\nBy requests per crime, %s leads with %.0f resident requests per crime, against %.0f citywide.\n",
		byRatio[0].name, perCrime(byRatio[0]), citywidePerCrime(rs))
	return nil
}

func demandTable(w io.Writer, rs []row) error {
	full := 0.0
	for _, r := range rs {
		full = max(full, r.x)
	}
	t := newTable(w, "#", "neighborhood", "resident 311", "", "crimes", "311 per crime", "top request type")
	for i, r := range rs {
		top, n := topResidentType(r.summary)
		t.row(i+1, r.name, num(int(r.x)), bar(r.x, full), num(int(r.y)), fmt.Sprintf("%.0f", r.x/max(r.y, 1)),
			fmt.Sprintf("%s (%d%%)", top, n*100/max(int(r.x), 1)))
	}
	return t.flush()
}

func citywidePerCrime(rs []row) float64 {
	var req, crimes float64
	for _, r := range rs {
		req += r.x
		crimes += r.y
	}
	return req / max(crimes, 1)
}
