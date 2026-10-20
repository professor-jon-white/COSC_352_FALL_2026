package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
)

// DistressByQuarter ranks neighborhoods each quarter by a combined distress
// index built from crime and resident 311 volume.
//
// The two counts are on very different scales (citywide there are about
// twelve 311 requests per crime), so adding them raw would just rank 311
// volume. Instead each is divided by the average across all neighborhoods
// that quarter, and the index is the mean of the two ratios:
//
//	index = (crimes / avg crimes + 311 / avg 311) / 2
//
// An index of 1.0 is a typical neighborhood; 3.0 means three times the
// typical level of crime and 311 activity combined.
type DistressByQuarter struct{}

func (DistressByQuarter) ID() string { return "distress" }
func (DistressByQuarter) Question() string {
	return "Top 10 neighborhoods by combined distress (crime + 311 volume), by quarter"
}

type quarterCounts struct{ crimes, requests int }

func (DistressByQuarter) Render(w io.Writer, in *Input) error {
	// quarters[q][neighborhood], with every neighborhood present in every
	// quarter so a quiet quarter counts as zero rather than being skipped.
	var quarters [4]map[string]*quarterCounts
	for q := range quarters {
		quarters[q] = make(map[string]*quarterCounts, len(in.Neighborhoods))
		for _, n := range in.Neighborhoods {
			quarters[q][n.Key.Neighborhood] = &quarterCounts{}
		}
	}
	for i := range in.Months {
		s := &in.Months[i]
		q := (int(s.Key.Month.Month()) - 1) / 3
		c := quarters[q][s.Key.Neighborhood]
		c.crimes += s.CrimeCount
		c.requests += residentRequests(s)
	}

	type scored struct {
		name  string
		c     quarterCounts
		index float64
	}
	var totalsR [4]int
	busiest := 0
	for q, counts := range quarters {
		for _, c := range counts {
			totalsR[q] += c.requests
		}
		busiest = max(busiest, totalsR[q])
	}

	appearances := map[string]int{}
	for q, counts := range quarters {
		var totalC int
		for _, c := range counts {
			totalC += c.crimes
		}
		totalR := totalsR[q]
		if totalC == 0 && totalR == 0 {
			continue // no data this quarter
		}
		avgC := float64(totalC) / float64(len(counts))
		avgR := float64(totalR) / float64(len(counts))

		all := make([]scored, 0, len(counts))
		for name, c := range counts {
			all = append(all, scored{name, *c, (ratio(c.crimes, avgC) + ratio(c.requests, avgR)) / 2})
		}
		slices.SortFunc(all, func(a, b scored) int {
			return cmp.Or(cmp.Compare(b.index, a.index), cmp.Compare(a.name, b.name))
		})
		top := all[:min(10, len(all))]

		fmt.Fprintf(w, "Q%d %d  (avg per neighborhood: %.1f crimes, %.1f resident 311 requests)\n", q+1, in.Year, avgC, avgR)
		if float64(totalR) < 0.6*float64(busiest) {
			// A gap in the source data, not a quiet quarter. The index is
			// relative to this quarter's own average, so rankings still hold.
			fmt.Fprintf(w, "  note: 311 volume this quarter is %.0f%% of the busiest quarter's, likely missing records;\n"+
				"  the index compares neighborhoods within the quarter, so the ranking is still usable.\n",
				100*float64(totalR)/float64(busiest))
		}
		t := newTable(w, "#", "neighborhood", "crimes", "311", "index", "")
		for i, s := range top {
			appearances[s.name]++
			t.row(i+1, s.name, num(s.c.crimes), num(s.c.requests), fmt.Sprintf("%.2f", s.index), bar(s.index, top[0].index))
		}
		if err := t.flush(); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}

	var every []string
	for name, n := range appearances {
		if n == 4 {
			every = append(every, name)
		}
	}
	slices.Sort(every)
	if len(every) == 0 {
		fmt.Fprintln(w, "Answer: no neighborhood is in the top 10 every quarter.")
	} else {
		fmt.Fprintf(w, "Answer: %d neighborhoods are in the top 10 all four quarters:\n  %s.\n", len(every), strings.Join(every, ", "))
	}
	return nil
}

func ratio(n int, avg float64) float64 {
	if avg == 0 {
		return 0
	}
	return float64(n) / avg
}
