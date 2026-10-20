// Package analysis holds one Strategy per research question. Each strategy
// reads the finished, immutable model.Dataset and returns a report.Result, so
// questions can be added, removed or run concurrently without touching the
// ingest, join or output code.
package analysis

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
)

const (
	// minHoodCrimes excludes neighborhoods with too little data for rankings.
	minHoodCrimes = 5
	// minClosed is the minimum completed 311 requests for a stable median.
	minClosed = 20
)

// Analysis is the Strategy interface implemented by every research question.
type Analysis interface {
	ID() string
	Run(d *model.Dataset) report.Result
}

// All returns the five research questions in presentation order.
func All() []Analysis {
	return []Analysis{
		DisorderVsCrime{},
		LiquorVsNightCrime{},
		ResponseEquity{},
		LicensedBlocks{},
		PriorityIndex{W: DefaultWeights()},
	}
}

// RunAll runs every analysis concurrently, one goroutine each. The dataset is
// read-only and every goroutine writes only to its own slot of the result
// slice, so no locking is required. Results keep the order of list.
func RunAll(d *model.Dataset, list []Analysis) []report.Result {
	out := make([]report.Result, len(list))
	var wg sync.WaitGroup
	for i, a := range list {
		wg.Add(1)
		go func(i int, a Analysis) {
			defer wg.Done()
			out[i] = a.Run(d)
		}(i, a)
	}
	wg.Wait()
	return out
}

// hoodRow pairs a neighborhood name with its aggregate.
type hoodRow struct {
	Name string
	*model.HoodAgg
}

// universe returns neighborhoods with enough crime records to be ranked.
func universe(d *model.Dataset) []hoodRow {
	var rows []hoodRow
	for _, name := range d.HoodNames() {
		h := d.Hoods[name]
		if h.Crimes >= minHoodCrimes {
			rows = append(rows, hoodRow{Name: name, HoodAgg: h})
		}
	}
	return rows
}

func commas(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func f1(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }
func f2(x float64) string { return strconv.FormatFloat(x, 'f', 2, 64) }

func pct(a, b int) string {
	if b == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", float64(a)/float64(b)*100)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// trunc shortens s to at most n runes, ending with "..." if it was cut.
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}

// pretty converts "MOUNT VERNON" to "Mount Vernon".
func pretty(s string) string {
	var b strings.Builder
	prevLetter := false
	for _, r := range strings.ToLower(s) {
		if !prevLetter && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
		} else {
			b.WriteRune(r)
		}
		prevLetter = unicode.IsLetter(r) || r == '\''
	}
	return b.String()
}

// enoughClosed reports whether a neighborhood has enough completed 311
// requests for its median time-to-close to be trusted.
func (r hoodRow) enoughClosed() bool {
	return len(r.CloseHours) >= minClosed
}
