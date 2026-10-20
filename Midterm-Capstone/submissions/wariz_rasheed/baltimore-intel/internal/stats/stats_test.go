package stats

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestMedianDoesNotMutate(t *testing.T) {
	x := []float64{5, 1, 3}
	if m := Median(x); m != 3 {
		t.Errorf("median = %v, want 3", m)
	}
	if x[0] != 5 || x[1] != 1 || x[2] != 3 {
		t.Errorf("input was modified: %v", x)
	}
	if m := Median([]float64{1, 2, 3, 4}); m != 2.5 {
		t.Errorf("even median = %v, want 2.5", m)
	}
	if Median(nil) != 0 {
		t.Error("empty median must be 0")
	}
}

func TestRanksWithTies(t *testing.T) {
	got := Ranks([]float64{10, 20, 20, 30})
	want := []float64{1, 2.5, 2.5, 4}
	for i := range want {
		if !near(got[i], want[i]) {
			t.Fatalf("ranks = %v, want %v", got, want)
		}
	}
}

func TestCorrelations(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	up := []float64{2, 4, 6, 8, 10}
	down := []float64{10, 8, 6, 4, 2}
	if r := Pearson(x, up); !near(r, 1) {
		t.Errorf("Pearson up = %v", r)
	}
	if r := Spearman(x, down); !near(r, -1) {
		t.Errorf("Spearman down = %v", r)
	}
	if r := Spearman(x, []float64{1, 1, 1, 1, 1}); r != 0 {
		t.Errorf("constant series should give 0, got %v", r)
	}
	if r := Spearman(x[:1], up[:1]); r != 0 {
		t.Errorf("single point should give 0, got %v", r)
	}
}

func TestPercentileRanks(t *testing.T) {
	got := PercentileRanks([]float64{5, 1, 3})
	want := []float64{100, 0, 50}
	for i := range want {
		if !near(got[i], want[i]) {
			t.Fatalf("percentiles = %v, want %v", got, want)
		}
	}
	if p := PercentileRanks([]float64{7}); p[0] != 50 {
		t.Errorf("single value percentile = %v, want 50", p[0])
	}
}

func TestDescribe(t *testing.T) {
	cases := map[float64]string{
		0.05:  "no meaningful relationship",
		0.2:   "weak positive relationship",
		-0.4:  "moderate negative relationship",
		0.6:   "strong positive relationship",
		-0.85: "very strong negative relationship",
	}
	for rho, want := range cases {
		if got := Describe(rho); got != want {
			t.Errorf("Describe(%v) = %q, want %q", rho, got, want)
		}
	}
}

func TestPartialSpearman(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{5, 4, 3, 2, 1}
	flat := []float64{7, 7, 7, 7, 7}
	// A constant control variable removes nothing.
	if r := PartialSpearman(x, y, flat); !near(r, -1) {
		t.Errorf("partial with constant control = %v, want -1", r)
	}
	// Controlling for a variable identical to x leaves nothing to correlate.
	if r := PartialSpearman(x, y, x); r != 0 {
		t.Errorf("partial controlling for x itself = %v, want 0", r)
	}
}
