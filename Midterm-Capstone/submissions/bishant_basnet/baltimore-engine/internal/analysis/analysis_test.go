package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/bishant/baltimore-engine/internal/model"
)

func TestPearson(t *testing.T) {
	if r := Pearson([]float64{1, 2, 3}, []float64{2, 4, 6}); math.Abs(r-1) > 1e-9 {
		t.Fatalf("want 1, got %v", r)
	}
	if r := Pearson([]float64{1, 2, 3}, []float64{3, 2, 1}); math.Abs(r+1) > 1e-9 {
		t.Fatalf("want -1, got %v", r)
	}
}

func TestIsBlight(t *testing.T) {
	if !IsBlight("SW-Rat Rubout") || IsBlight("TRM-Operations Review") {
		t.Fatal("blight classification wrong")
	}
}

func TestBuildContextMatchesSerial(t *testing.T) {
	now := time.Now()
	var crimes []model.CrimeRecord
	var reqs []model.ServiceRequest
	for i := 0; i < 1000; i++ {
		h := []string{"A", "B", "C", "D", "E"}[i%5]
		crimes = append(crimes, model.CrimeRecord{Neighborhood: h, When: now, HasTime: true})
		reqs = append(reqs, model.ServiceRequest{Neighborhood: h, Type: "SW-Dirty Alley", Created: now, Closed: now.Add(time.Hour), HasCreated: true, HasClosed: true})
	}
	for _, w := range []int{1, 3, 8} {
		c := BuildContext(crimes, reqs, w, 10)
		if len(c.Joined) != 5 || c.Hoods["A"].Crimes != 200 || c.Hoods["A"].Blight != 200 || len(c.Hoods["A"].ResolutionHrs) != 200 {
			t.Fatalf("workers=%d: aggregation mismatch", w)
		}
		res := RunAll(c, DefaultQuestions())
		if len(res) != 5 {
			t.Fatal("expected 5 results")
		}
	}
}
