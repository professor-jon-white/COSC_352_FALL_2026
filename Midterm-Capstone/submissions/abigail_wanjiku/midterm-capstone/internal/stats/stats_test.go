package stats

import (
	"math"
	"slices"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRanks(t *testing.T) {
	got := Ranks([]float64{30, 10, 20, 20})
	if want := []float64{4, 1, 2.5, 2.5}; !slices.Equal(got, want) {
		t.Errorf("Ranks = %v, want %v", got, want)
	}
}

func TestSpearman(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	if r := Spearman(x, []float64{10, 100, 1000, 10000, 100000}); !near(r, 1) {
		t.Errorf("monotonic increasing: r = %v, want 1", r)
	}
	if r := Spearman(x, []float64{5, 4, 3, 2, 1}); !near(r, -1) {
		t.Errorf("reversed: r = %v, want -1", r)
	}
	if r := Spearman(x, []float64{7, 7, 7, 7, 7}); !math.IsNaN(r) {
		t.Errorf("constant y: r = %v, want NaN", r)
	}
	// Classic textbook example: r_s = 1 - 6*sum(d^2)/(n(n^2-1)) with d = [0,-1,1,0,0] -> 0.9.
	if r := Spearman(x, []float64{1, 3, 2, 4, 5}); !near(r, 0.9) {
		t.Errorf("r = %v, want 0.9", r)
	}
}

func TestPercentile(t *testing.T) {
	xs := []float64{4, 1, 3, 2, 5}
	for p, want := range map[float64]float64{0: 1, 25: 2, 50: 3, 75: 4, 100: 5, 10: 1.4} {
		if got := Percentile(xs, p); !near(got, want) {
			t.Errorf("Percentile(%v) = %v, want %v", p, got, want)
		}
	}
	if xs[0] != 4 {
		t.Error("Percentile sorted its input in place")
	}
}

func TestStrength(t *testing.T) {
	for r, want := range map[float64]string{0.05: "no meaningful", 0.25: "weak positive", -0.4: "moderate negative", 0.6: "strong positive", 0.8: "very strong positive"} {
		if got := Strength(r); got != want {
			t.Errorf("Strength(%v) = %q, want %q", r, got, want)
		}
	}
}
