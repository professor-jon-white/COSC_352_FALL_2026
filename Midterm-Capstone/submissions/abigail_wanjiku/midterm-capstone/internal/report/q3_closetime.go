package report

import (
	"fmt"
	"io"

	"capstone/internal/model"
	"capstone/internal/stats"
)

// minClosed is the fewest closed requests a neighborhood needs before its
// close-time ratio is used in the correlation; below that, one slow request
// can swing the ratio wildly.
const minClosed = 50

// CloseTimeVsCrime asks whether 311 requests take longer to close in
// high-crime neighborhoods.
//
// A raw average is misleading because request types close at very different
// speeds (a proactive rat baiting closes in minutes; a sanitation
// enforcement case takes weeks), and neighborhoods file different mixes. So
// alongside the raw average it computes a mix-adjusted ratio:
//
//	expected = sum over request types t of closed_t * citywide avg days_t
//	ratio    = actual total days / expected total days
//
// A ratio of 1.25 means requests there take 25% longer than the same mix of
// requests takes citywide.
type CloseTimeVsCrime struct{}

func (CloseTimeVsCrime) ID() string { return "close-time" }
func (CloseTimeVsCrime) Question() string {
	return "Are 311 requests closed more slowly in high-crime neighborhoods?"
}

func (CloseTimeVsCrime) Render(w io.Writer, in *Input) error {
	expected := expectedCloseDays(in)
	rs := rows(in,
		func(s *model.Summary) float64 { return float64(s.CrimeCount) },
		func(s *model.Summary) float64 {
			if e := expected(s); e > 0 {
				return s.CloseDaysTotal / e
			}
			return 0
		},
	)

	// Pool each crime quartile's requests, so big neighborhoods weigh more
	// than small ones, exactly as if the quartile were one neighborhood.
	type group struct {
		n, crimesLo, crimesHi, closed int
		avg, adj                      float64
	}
	var groups [4]group
	var maxAvg, maxAdj float64
	for g, members := range quartiles(rs, byX) {
		gr := group{n: len(members), crimesLo: members[0].summary.CrimeCount}
		var actual, exp float64
		for _, r := range members {
			gr.crimesHi = max(gr.crimesHi, r.summary.CrimeCount)
			gr.closed += r.summary.ClosedCount
			actual += r.summary.CloseDaysTotal
			exp += expected(r.summary)
		}
		if gr.closed > 0 {
			gr.avg = actual / float64(gr.closed)
		}
		if exp > 0 {
			gr.adj = actual / exp
		}
		groups[g] = gr
		maxAvg, maxAdj = max(maxAvg, gr.avg), max(maxAdj, gr.adj)
	}

	t := newTable(w, "crime quartile", "nbhds", "crimes each", "closed 311", "avg days", "", "mix-adjusted", "")
	for g, gr := range groups {
		t.row(quartileNames[g], gr.n, fmt.Sprintf("%s-%s", num(gr.crimesLo), num(gr.crimesHi)), num(gr.closed),
			fmt.Sprintf("%.1f", gr.avg), bar(gr.avg, maxAvg), fmt.Sprintf("%.2fx", gr.adj), bar(gr.adj, maxAdj))
	}
	if err := t.flush(); err != nil {
		return err
	}

	enough := make([]row, 0, len(rs))
	for _, r := range rs {
		if r.summary.ClosedCount >= minClosed {
			enough = append(enough, r)
		}
	}
	rawAvg := func(s *model.Summary) float64 { d, _ := s.AvgDaysToClose(); return d }
	raw := make([]float64, len(enough))
	for i, r := range enough {
		raw[i] = rawAvg(r.summary)
	}
	rhoRaw := stats.Spearman(xs(enough), raw)
	rhoAdj := stats.Spearman(xs(enough), ys(enough))

	fmt.Fprintf(w, "\nSpearman rank correlation with crime (%d neighborhoods with >= %d closed requests):\n", len(enough), minClosed)
	fmt.Fprintf(w, "  raw avg days to close        %s\n", corr(rhoRaw))
	fmt.Fprintf(w, "  mix-adjusted close ratio     %s\n", corr(rhoAdj))

	hi, lo := groups[3].adj, groups[0].adj
	pct := (hi/lo - 1) * 100
	switch {
	case rhoAdj >= 0.1 && pct > 5:
		fmt.Fprintf(w, "\nAnswer: Yes. For the same mix of request types, the highest-crime quarter of neighborhoods waits %.0f%% longer than the lowest.\n", pct)
	case rhoAdj <= -0.1 && pct < -5:
		fmt.Fprintf(w, "\nAnswer: No, the opposite. For the same mix of request types, the highest-crime quarter is closed %.0f%% faster than the lowest.\n", -pct)
	default:
		fmt.Fprintf(w, "\nAnswer: No meaningful difference once request mix is accounted for (highest vs lowest crime quarter: %+.0f%%).\n", pct)
	}
	return nil
}

// expectedCloseDays returns a function giving, for a summary, the total days
// its closed requests would have taken at citywide per-type average speeds.
func expectedCloseDays(in *Input) func(*model.Summary) float64 {
	closed := map[string]int{}
	days := map[string]float64{}
	for i := range in.Neighborhoods {
		s := &in.Neighborhoods[i]
		for t, n := range s.ClosedByType {
			closed[t] += n
			days[t] += s.CloseDaysByType[t]
		}
	}
	return func(s *model.Summary) float64 {
		var e float64
		for t, n := range s.ClosedByType {
			e += float64(n) * days[t] / float64(closed[t])
		}
		return e
	}
}
