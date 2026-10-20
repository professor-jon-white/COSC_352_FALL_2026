package output

import (
	"baltimore-intelligence/internal/analysis"
	"bytes"
	"strings"
	"testing"
)

// Reporting-only regression test: simple titles must appear while every
// section still reports the same underlying measurements.
func TestSimplifiedHeadings(t *testing.T) {
	stats := []analysis.Stats{{
		Name: "Example Neighborhood", Total: 10, Open: 2, Closed: 8,
		Category:    map[string]int{"Pothole": 6, "Trash": 4},
		DurationSum: 40, DurationN: 8, Priority: 45.5,
	}}
	var out bytes.Buffer
	Report(&out, stats, 10, 0)
	got := out.String()
	headings := []string{
		"BALTIMORE CITY SERVICE REPORT",
		"Q1. WHICH NEIGHBORHOODS REPORT THE MOST PROBLEMS?",
		"Q2. WHAT PROBLEMS ARE MOST COMMON IN EACH NEIGHBORHOOD?",
		"Q3. WHERE IS THE HIGHEST PERCENTAGE OF PROBLEMS STILL UNRESOLVED?",
		"Q4. WHERE DOES IT TAKE THE LONGEST TO FIX PROBLEMS?",
		"Q5. WHICH NEIGHBORHOODS NEED THE MOST ATTENTION?",
	}
	for _, h := range headings {
		if !strings.Contains(got, h) {
			t.Errorf("missing heading %q", h)
		}
	}
	for _, value := range []string{"Pothole", "20.0% (2/10)", "5.0 days (n=8)", "45.5 / 100"} {
		if !strings.Contains(got, value) {
			t.Errorf("missing expected result %q", value)
		}
	}
}
