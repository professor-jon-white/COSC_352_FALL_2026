package pipeline

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"baltimoreintel/internal/analysis"
	"baltimoreintel/internal/report"
)

// The fixtures in testdata/ are slices of the real portal exports
// (late Dec 2025 to early Jan 2026), so they exercise the real schemas.

func build(t *testing.T, workers int, from, to time.Time) *Output {
	t.Helper()
	p, err := NewBuilder().
		Crime("testdata/crime.csv").
		Requests("testdata/requests.csv").
		Licenses("testdata/licenses.csv").
		Workers(workers).
		Window(from, to).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func wideWindow() (time.Time, time.Time) {
	return time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 11, 23, 59, 59, 0, time.UTC)
}

func TestJoinCoverage(t *testing.T) {
	from, to := wideWindow()
	out := build(t, 4, from, to)
	q := out.Data.Quality

	if out.Loads[0].Read != 2282 || out.Loads[1].Read != 299 {
		t.Errorf("rows read: crime %d (want 2282), 311 %d (want 299)", out.Loads[0].Read, out.Loads[1].Read)
	}
	if q.Crime.Rows != 2282 || q.Crime.Located != 2279 {
		t.Errorf("crime coverage = %+v, want 2282 rows / 2279 located", q.Crime)
	}
	if q.Requests.Rows != 183 || q.Requests.Ignored != 116 {
		t.Errorf("311 coverage = %+v, want 183 rows / 116 ECC ignored", q.Requests)
	}
	if q.Licenses.Rows != 829 || q.Licenses.Located != 237 {
		t.Errorf("license coverage = %+v, want 829 active / 237 located", q.Licenses)
	}
	if len(out.Data.Hoods) == 0 || len(out.Data.Blocks) == 0 || out.Data.IndexBlocks == 0 {
		t.Error("expected neighborhoods, blocks and a non-empty index")
	}
}

func TestDerivedWindow(t *testing.T) {
	out := build(t, 2, time.Time{}, time.Time{})
	if !out.Data.From.Before(out.Data.To) {
		t.Fatalf("window not derived: %v to %v", out.Data.From, out.Data.To)
	}
	// 311 sample spans 2 Jan 16:08 to 4 Jan 01:50 UTC; 210 crimes fall inside.
	if out.Data.Quality.Crime.Rows != 210 {
		t.Errorf("crimes in derived window = %d, want 210", out.Data.Quality.Crime.Rows)
	}
}

// Results must not depend on how many goroutines did the work. Run with
// `go test -race` to also prove the concurrent stages are data-race free.
func TestWorkerCountDoesNotChangeResults(t *testing.T) {
	from, to := wideWindow()
	a := build(t, 1, from, to).Data
	b := build(t, 8, from, to).Data
	if a.Quality != b.Quality {
		t.Fatalf("quality differs: %+v vs %+v", a.Quality, b.Quality)
	}
	if len(a.Hoods) != len(b.Hoods) || len(a.Blocks) != len(b.Blocks) {
		t.Fatalf("sizes differ: %d/%d hoods, %d/%d blocks", len(a.Hoods), len(b.Hoods), len(a.Blocks), len(b.Blocks))
	}
	for name, ha := range a.Hoods {
		hb, ok := b.Hoods[name]
		if !ok {
			t.Fatalf("neighborhood %q missing with 8 workers", name)
		}
		if ha.Crimes != hb.Crimes || ha.Violent != hb.Violent || ha.NightViolent != hb.NightViolent ||
			ha.Requests != hb.Requests || ha.Open != hb.Open || ha.Licenses != hb.Licenses ||
			len(ha.CloseHours) != len(hb.CloseHours) {
			t.Fatalf("neighborhood %q differs between 1 and 8 workers", name)
		}
	}
}

func TestMissingFileIsReported(t *testing.T) {
	p, err := NewBuilder().Crime("testdata/nope.csv").Requests("testdata/requests.csv").Licenses("testdata/licenses.csv").Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Run(); err == nil {
		t.Fatal("expected an error for a missing crime file")
	}
}

func TestBuilderValidates(t *testing.T) {
	if _, err := NewBuilder().Crime("a.csv").Build(); err == nil {
		t.Fatal("expected an error when paths are missing")
	}
}

func TestFullReportRenders(t *testing.T) {
	from, to := wideWindow()
	out := build(t, 4, from, to)
	results := analysis.RunAll(out.Data, analysis.All())
	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}
	for _, r := range results {
		if r.Finding == "" {
			t.Errorf("%s has an empty finding", r.ID)
		}
	}
	s := report.Summary{Title: "Test Report", Subtitle: "sub", Results: results,
		Overview: report.Table{Headers: []string{"A", "B"}, Rows: [][]string{{"1", "2"}}},
		Footer:   []string{"footer line"}}

	var term bytes.Buffer
	tr, err := report.New("terminal")
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Render(&term, s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.String(), "QUESTION 5 of 5") {
		t.Error("terminal report is missing question 5")
	}

	var page bytes.Buffer
	hr, err := report.New("html")
	if err != nil {
		t.Fatal(err)
	}
	if err := hr.Render(&page, s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.String(), "<!DOCTYPE html>") || !strings.Contains(page.String(), "Question 5") {
		t.Error("HTML report is incomplete")
	}
	if _, err := report.New("pdf"); err == nil {
		t.Error("unknown report format should be an error")
	}
}
