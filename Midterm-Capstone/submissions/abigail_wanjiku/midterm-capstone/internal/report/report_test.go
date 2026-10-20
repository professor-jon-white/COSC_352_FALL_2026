package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"capstone/internal/model"
)

func TestCategories(t *testing.T) {
	for typ, want := range map[string]bool{"HCD-Illegal Dumping": true, "SW-Dirty Alley": true, "SW-Rat Rubout Proactive": false, "SW-Recycling": false} {
		if got := isSanitation(typ); got != want {
			t.Errorf("isSanitation(%q) = %v", typ, got)
		}
	}
	for typ, want := range map[string]bool{"BGE-StLight(s) Out": true, "TRM-Street Light Out": true, "RP-Street Lighting Repairs": true,
		"FCPF-Trial Request-Red Light": false, "TEST-St Light Out": false, "TRS-Parking Complaint": false} {
		if got := isStreetlight(typ); got != want {
			t.Errorf("isStreetlight(%q) = %v", typ, got)
		}
	}
}

func TestFormatting(t *testing.T) {
	if got := num(1234567); got != "1,234,567" {
		t.Errorf("num = %q", got)
	}
	if got := num(999); got != "999" {
		t.Errorf("num = %q", got)
	}
	if got := bar(50, 100); got != strings.Repeat("#", barWidth/2) {
		t.Errorf("bar(50, 100) = %q", got)
	}
	if got := bar(0.001, 100); got != "#" {
		t.Errorf("tiny positive value should show one mark, got %q", got)
	}
	if bar(0, 100) != "" || bar(5, 0) != "" {
		t.Error("zero value or zero scale should draw nothing")
	}
}

func TestSelect(t *testing.T) {
	all, err := Select("all")
	if err != nil || len(all) != 5 {
		t.Fatalf("Select(all) = %d reports, %v", len(all), err)
	}
	some, err := Select("streetlights, sanitation")
	if err != nil || len(some) != 2 || some[0].ID() != "streetlights" {
		t.Fatalf("Select subset = %v, %v", some, err)
	}
	if _, err := Select("nope"); err == nil {
		t.Error("expected an error for an unknown report")
	}
}

// sample builds 8 neighborhoods where crime and 311 rise together, except
// QUIET, which has heavy 311 use and almost no crime.
func sample() *Input {
	var months []model.Summary
	for i := range 8 {
		name := fmt.Sprintf("N%d", i)
		crimes, requests := 10*(i+1), 100*(i+1)
		if i == 7 {
			name, crimes, requests = "QUIET", 5, 900
		}
		for m := time.January; m <= time.December; m++ {
			s := model.NewSummary(model.Key{Neighborhood: name, Month: time.Date(2025, m, 1, 0, 0, 0, 0, time.UTC)})
			s.CrimeCount, s.NightOutdoorCrimes = crimes, crimes/4
			s.RequestCount = requests
			s.RequestsByType["HCD-Illegal Dumping"] = requests / 2
			s.RequestsByType["BGE-StLight(s) Out"] = requests / 10
			s.RequestsByType["SW-Rat Rubout Proactive"] = requests - requests/2 - requests/10
			s.ClosedCount, s.CloseDaysTotal = requests/2, float64(requests)
			s.ClosedByType["HCD-Illegal Dumping"], s.CloseDaysByType["HCD-Illegal Dumping"] = requests/2, float64(requests)
			months = append(months, *s)
		}
	}
	return NewInput(2025, months)
}

func TestReportsRender(t *testing.T) {
	var out bytes.Buffer
	if err := Run(&out, sample(), All()); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Q1 [sanitation]", "Q5 [high-311-low-crime]",
		"pass the strict test (QUIET)", // the planted high-311, low-crime neighborhood
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n%s", want, got)
		}
	}
	if strings.Count(got, "Answer:") != 5 {
		t.Errorf("want 5 answers, got %d\n%s", strings.Count(got, "Answer:"), got)
	}
}

func TestReportsNeedEnoughData(t *testing.T) {
	var out bytes.Buffer
	if err := Run(&out, NewInput(2025, nil), All()); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "Not enough data") != 5 {
		t.Errorf("empty input should skip every report:\n%s", out.String())
	}
}
