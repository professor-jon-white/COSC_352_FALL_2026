package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"capstone/internal/aggregate"
	"capstone/internal/model"
)

// Report is one analysis question. Each question is a separate strategy type
// with the same interface, so the runner can execute any of them, in any
// combination, without knowing how a particular answer is computed.
type Report interface {
	ID() string       // short name for the -reports flag, e.g. "sanitation"
	Question() string // the question the report answers
	Render(w io.Writer, in *Input) error
}

// All returns every report in question order.
func All() []Report {
	return []Report{
		SanitationVsCrime{},
		DistressByQuarter{},
		CloseTimeVsCrime{},
		StreetlightsVsNightCrime{},
		HighDemandLowCrime{},
	}
}

// Select returns the reports named in a comma-separated list of IDs, or all
// of them for "all".
func Select(ids string) ([]Report, error) {
	if ids == "" || ids == "all" {
		return All(), nil
	}
	byID := map[string]Report{}
	known := make([]string, 0, len(All()))
	for _, r := range All() {
		byID[r.ID()] = r
		known = append(known, r.ID())
	}
	var out []Report
	for _, id := range strings.Split(ids, ",") {
		r, ok := byID[strings.TrimSpace(id)]
		if !ok {
			return nil, fmt.Errorf("unknown report %q (want all or one of %s)", id, strings.Join(known, ", "))
		}
		out = append(out, r)
	}
	return out, nil
}

// Run renders each report under a heading numbered by its position in All,
// so a question keeps its number when only some reports are selected.
func Run(w io.Writer, in *Input, reports []Report) error {
	number := map[string]int{}
	for i, r := range All() {
		number[r.ID()] = i + 1
	}
	for _, r := range reports {
		heading := fmt.Sprintf("Q%d [%s] %s", number[r.ID()], r.ID(), r.Question())
		fmt.Fprintf(w, "\n%s\n%s\n\n", heading, strings.Repeat("=", min(len(heading), 100)))
		if len(in.Neighborhoods) < 4 { // quartiles and top-N lists need a few neighborhoods
			fmt.Fprintf(w, "Not enough data: %d neighborhoods.\n", len(in.Neighborhoods))
			continue
		}
		if err := r.Render(w, in); err != nil {
			return fmt.Errorf("report %s: %w", r.ID(), err)
		}
	}
	return nil
}

// Input is the combined data every report reads. Reports must not modify it.
type Input struct {
	Year          int
	Months        []model.Summary // one per neighborhood-month, sorted
	Neighborhoods []model.Summary // one per neighborhood, all months merged, sorted by name
}

// NewInput builds the report input from neighborhood-month summaries.
func NewInput(year int, months []model.Summary) *Input {
	return &Input{Year: year, Months: months, Neighborhoods: aggregate.ByNeighborhood(months)}
}

// row is one neighborhood's numbers for a report, carried alongside its name
// so reports can sort and slice without losing track of which is which.
type row struct {
	name    string
	summary *model.Summary
	x, y    float64 // the two measures the report compares
}

// rows extracts (x, y) for every neighborhood in in.
func rows(in *Input, x, y func(*model.Summary) float64) []row {
	out := make([]row, len(in.Neighborhoods))
	for i := range in.Neighborhoods {
		s := &in.Neighborhoods[i]
		out[i] = row{name: s.Key.Neighborhood, summary: s, x: x(s), y: y(s)}
	}
	return out
}

func xs(rs []row) []float64 {
	out := make([]float64, len(rs))
	for i, r := range rs {
		out[i] = r.x
	}
	return out
}

func ys(rs []row) []float64 {
	out := make([]float64, len(rs))
	for i, r := range rs {
		out[i] = r.y
	}
	return out
}

// sortedBy returns a copy of rs sorted by key descending, ties by name.
func sortedBy(rs []row, key func(row) float64) []row {
	out := slices.Clone(rs)
	slices.SortFunc(out, func(a, b row) int {
		return cmp.Or(cmp.Compare(key(b), key(a)), cmp.Compare(a.name, b.name))
	})
	return out
}

// rankOf maps each neighborhood to its 1-based position when rs is sorted by
// key descending (1 = highest).
func rankOf(rs []row, key func(row) float64) map[string]int {
	ranks := make(map[string]int, len(rs))
	for i, r := range sortedBy(rs, key) {
		ranks[r.name] = i + 1
	}
	return ranks
}

// quartiles splits rs into four equal-sized groups by key, lowest first.
func quartiles(rs []row, key func(row) float64) [4][]row {
	sorted := sortedBy(rs, key)
	slices.Reverse(sorted) // ascending
	var q [4][]row
	for g := range q {
		q[g] = make([]row, 0, len(sorted)/4+1)
	}
	for i, r := range sorted {
		g := i * 4 / len(sorted)
		q[g] = append(q[g], r)
	}
	return q
}

var quartileNames = [4]string{"Q1 (lowest)", "Q2", "Q3", "Q4 (highest)"}

func byX(r row) float64 { return r.x }
func byY(r row) float64 { return r.y }
