package aggregate

import (
	"testing"
	"time"

	"capstone/internal/model"
)

func summary(n string, month time.Month, crimes, requests int, closeDays float64) model.Summary {
	s := model.NewSummary(model.Key{Neighborhood: n, Month: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC)})
	s.CrimeCount = crimes
	s.CrimesByType["ROBBERY"] = crimes
	s.RequestCount = requests
	s.RequestsByType["Pothole"] = requests
	s.ClosedCount = requests
	s.CloseDaysTotal = closeDays
	return *s
}

func TestByNeighborhoodAndTop(t *testing.T) {
	in := []model.Summary{
		summary("ABELL", 1, 2, 2, 2), // Jan avg 1.0 days
		summary("ABELL", 2, 3, 6, 8), // Feb avg 1.33 days
		summary("BARCLAY", 1, 5, 1, 0),
		summary("CANTON", 1, 5, 0, 0),
	}

	by := ByNeighborhood(in)
	if len(by) != 3 {
		t.Fatalf("got %d neighborhoods, want 3", len(by))
	}
	abell := by[0]
	if !abell.Key.Month.IsZero() || abell.CrimeCount != 5 || abell.RequestCount != 8 || abell.CrimesByType["ROBBERY"] != 5 {
		t.Errorf("ABELL = %+v", abell)
	}
	// Weighted by request count: (2+8)/(2+6) = 1.25, not the mean of monthly means (1.0+1.333)/2.
	if avg, _ := abell.AvgDaysToClose(); avg != 1.25 {
		t.Errorf("ABELL AvgDaysToClose = %v, want 1.25", avg)
	}

	top := TopByCrime(by, 2)
	if len(top) != 2 || top[0].Key.Neighborhood != "ABELL" || top[1].Key.Neighborhood != "BARCLAY" {
		t.Errorf("top 2 = %s, %s; want ABELL, BARCLAY (tie broken by name)", top[0].Key.Neighborhood, top[1].Key.Neighborhood)
	}
	if by[0].Key.Neighborhood != "ABELL" || by[1].Key.Neighborhood != "BARCLAY" || by[2].Key.Neighborhood != "CANTON" {
		t.Error("TopByCrime modified its input")
	}
}

func TestMostCommon(t *testing.T) {
	if k, n := MostCommon(map[string]int{"b": 2, "a": 2, "c": 1}); k != "a" || n != 2 {
		t.Errorf("MostCommon = %q, %d; want a, 2", k, n)
	}
	if k, n := MostCommon(nil); k != "" || n != 0 {
		t.Errorf("MostCommon(nil) = %q, %d", k, n)
	}
}
