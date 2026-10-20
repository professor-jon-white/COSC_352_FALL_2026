package match

import (
	"math"
	"reflect"
	"testing"
	"time"

	"capstone/internal/ingest"
	"capstone/internal/model"
)

func at(month time.Month, day, hour, minute int) time.Time {
	return time.Date(2025, month, day, hour, minute, 0, 0, ingest.Baltimore)
}

func TestByNeighborhoodMonth(t *testing.T) {
	crimes := []model.Crime{
		{OccurredAt: at(1, 5, 12, 0), Neighborhood: "ABELL", CrimeType: "BURGLARY"},
		{OccurredAt: at(1, 31, 23, 59), Neighborhood: "ABELL", CrimeType: "BURGLARY"},
		{OccurredAt: at(1, 9, 8, 0), Neighborhood: "ABELL", CrimeType: "ROBBERY"},
		{OccurredAt: at(2, 1, 0, 0), Neighborhood: "ABELL", CrimeType: "ROBBERY"},   // February
		{OccurredAt: at(1, 2, 8, 0), Neighborhood: "BARCLAY", CrimeType: "ROBBERY"}, // no 311 that month
	}
	requests := []model.ServiceRequest{
		// Created in January, closed in February: counts toward January.
		{CreatedAt: at(1, 30, 9, 0), ClosedAt: at(2, 2, 9, 0), Neighborhood: "ABELL", RequestType: "Pothole"},
		{CreatedAt: at(1, 10, 9, 0), ClosedAt: at(1, 11, 21, 0), Neighborhood: "ABELL", RequestType: "Pothole"},
		{CreatedAt: at(1, 12, 9, 0), Neighborhood: "ABELL", RequestType: "Dumping"}, // still open
		// Close time truncated to the minute, a few seconds "before" creation: 0 days.
		{CreatedAt: at(1, 13, 9, 0).Add(40 * time.Second), ClosedAt: at(1, 13, 9, 0), Neighborhood: "ABELL", RequestType: "Rats"},
		// Closed months before creation: excluded from the average.
		{CreatedAt: at(1, 14, 9, 0), ClosedAt: time.Date(2024, 9, 1, 0, 0, 0, 0, ingest.Baltimore), Neighborhood: "ABELL", RequestType: "Rats"},
		{CreatedAt: at(3, 1, 9, 0), Neighborhood: "CANTON", RequestType: "Dumping"}, // no crime that month
	}

	got := ByNeighborhoodMonth(crimes, requests)

	wantKeys := []string{"ABELL 2025-01", "ABELL 2025-02", "BARCLAY 2025-01", "CANTON 2025-03"}
	if len(got) != len(wantKeys) {
		t.Fatalf("got %d summaries, want %d", len(got), len(wantKeys))
	}
	for i, s := range got {
		if k := s.Key.Neighborhood + " " + s.Key.Month.Format("2006-01"); k != wantKeys[i] {
			t.Errorf("summary %d key = %s, want %s", i, k, wantKeys[i])
		}
	}

	jan := got[0]
	if jan.CrimeCount != 3 || jan.CrimesByType["BURGLARY"] != 2 || jan.CrimesByType["ROBBERY"] != 1 {
		t.Errorf("ABELL Jan crimes = %d %v", jan.CrimeCount, jan.CrimesByType)
	}
	if jan.RequestCount != 5 || jan.RequestsByType["Pothole"] != 2 || jan.RequestsByType["Rats"] != 2 {
		t.Errorf("ABELL Jan requests = %d %v", jan.RequestCount, jan.RequestsByType)
	}
	if jan.ClosedCount != 3 || jan.BadCloseTimes != 1 {
		t.Errorf("ClosedCount=%d BadCloseTimes=%d, want 3 and 1", jan.ClosedCount, jan.BadCloseTimes)
	}
	// (3 + 1.5 + 0) / 3 days
	if avg, ok := jan.AvgDaysToClose(); !ok || math.Abs(avg-1.5) > 1e-9 {
		t.Errorf("AvgDaysToClose = %v, %v; want 1.5", avg, ok)
	}

	if feb := got[1]; feb.CrimeCount != 1 || feb.RequestCount != 0 {
		t.Errorf("ABELL Feb = %d crimes, %d requests; want 1, 0", feb.CrimeCount, feb.RequestCount)
	}
	if canton := got[3]; canton.CrimeCount != 0 || canton.RequestCount != 1 {
		t.Errorf("CANTON Mar = %d crimes, %d requests; want 0, 1", canton.CrimeCount, canton.RequestCount)
	}
	if _, ok := got[3].AvgDaysToClose(); ok {
		t.Error("CANTON has no closed requests; AvgDaysToClose should report false")
	}
}

func TestConcurrentMatchesSequential(t *testing.T) {
	hoods := []string{"ABELL", "BARCLAY", "CANTON", "DOWNTOWN", "FRANKFORD", "HAMPDEN", "UPTON"}
	var crimes []model.Crime
	var requests []model.ServiceRequest
	for i := range 5000 {
		when := at(time.Month(i%12+1), i%28+1, i%24, i%60)
		hood := hoods[(i*7)%len(hoods)]
		crimes = append(crimes, model.Crime{OccurredAt: when, Neighborhood: hood, CrimeType: []string{"ROBBERY", "BURGLARY", "ARSON"}[i%3]})
		if i%4 == 0 {
			continue // leave some neighborhood-months crime-only
		}
		r := model.ServiceRequest{CreatedAt: when, Neighborhood: hoods[(i*3)%len(hoods)], RequestType: []string{"Pothole", "Rats"}[i%2]}
		switch i % 5 {
		case 1:
			r.ClosedAt = when.Add(time.Duration(i) * time.Minute) // fractional days, so float sums must match exactly
		case 2:
			r.ClosedAt = when.Add(-30 * 24 * time.Hour) // excluded
		}
		requests = append(requests, r)
	}

	want := ByNeighborhoodMonth(crimes, requests)
	for _, workers := range []int{0, 1, 2, 3, 5, 16} {
		got := ByNeighborhoodMonthConcurrent(crimes, requests, workers)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("workers=%d: concurrent result differs from sequential (%d vs %d summaries)", workers, len(got), len(want))
		}
	}
}

func TestShardStable(t *testing.T) {
	for _, n := range []int{1, 4, 12} {
		for _, hood := range []string{"", "ABELL", "BELAIR-EDISON"} {
			s := shard(hood, n)
			if s < 0 || s >= n || s != shard(hood, n) {
				t.Errorf("shard(%q, %d) = %d", hood, n, s)
			}
		}
	}
}

func TestIsNight(t *testing.T) {
	tests := []struct {
		hour, minute int
		want         bool
	}{
		{19, 59, false}, {20, 0, true}, {23, 30, true}, {0, 0, false}, // exact midnight is a placeholder
		{0, 1, true}, {5, 59, true}, {6, 0, false}, {12, 0, false},
	}
	for _, tc := range tests {
		if got := IsNight(at(3, 1, tc.hour, tc.minute)); got != tc.want {
			t.Errorf("IsNight(%02d:%02d) = %v, want %v", tc.hour, tc.minute, got, tc.want)
		}
	}
}

func TestNightOutdoorAndCloseByType(t *testing.T) {
	crimes := []model.Crime{
		{OccurredAt: at(3, 1, 22, 0), Neighborhood: "ABELL", Outdoor: true},  // counted
		{OccurredAt: at(3, 1, 22, 0), Neighborhood: "ABELL", Outdoor: false}, // inside
		{OccurredAt: at(3, 1, 14, 0), Neighborhood: "ABELL", Outdoor: true},  // daytime
		{OccurredAt: at(3, 1, 0, 0), Neighborhood: "ABELL", Outdoor: true},   // placeholder midnight
	}
	requests := []model.ServiceRequest{
		{CreatedAt: at(3, 1, 9, 0), ClosedAt: at(3, 3, 9, 0), Neighborhood: "ABELL", RequestType: "Pothole"},
		{CreatedAt: at(3, 1, 9, 0), ClosedAt: at(3, 2, 9, 0), Neighborhood: "ABELL", RequestType: "Pothole"},
		{CreatedAt: at(3, 1, 9, 0), ClosedAt: at(3, 1, 21, 0), Neighborhood: "ABELL", RequestType: "Rats"},
	}
	s := ByNeighborhoodMonth(crimes, requests)[0]
	if s.NightOutdoorCrimes != 1 {
		t.Errorf("NightOutdoorCrimes = %d, want 1", s.NightOutdoorCrimes)
	}
	if s.ClosedByType["Pothole"] != 2 || s.CloseDaysByType["Pothole"] != 3 || s.CloseDaysByType["Rats"] != 0.5 {
		t.Errorf("ClosedByType = %v, CloseDaysByType = %v", s.ClosedByType, s.CloseDaysByType)
	}
}
