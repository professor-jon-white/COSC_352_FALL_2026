package stats

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestBasics(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5, 9, 2, 6}
	if !near(Mean(xs), 3.875) || !near(Median(xs), 3.5) {
		t.Fatal("mean/median wrong")
	}
	if !near(Quantile(xs, 0), 1) || !near(Quantile(xs, 1), 9) {
		t.Fatal("quantile endpoints wrong")
	}
	if xs[0] != 3 {
		t.Fatal("Quantile must not reorder caller data")
	}
}

func TestCorrelations(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	if !near(Pearson(x, []float64{2, 4, 6, 8, 10}), 1) || !near(Pearson(x, []float64{5, 4, 3, 2, 1}), -1) {
		t.Fatal("pearson wrong")
	}
	// Spearman is 1 for any monotonic relationship.
	if !near(Spearman(x, []float64{1, 8, 27, 64, 125}), 1) {
		t.Fatal("spearman wrong")
	}
	if r := Ranks([]float64{10, 20, 20, 30}); !near(r[1], 2.5) || !near(r[2], 2.5) {
		t.Fatalf("tied ranks = %v", r)
	}
	a, b := LinearFit(x, []float64{3, 5, 7, 9, 11})
	if !near(a, 1) || !near(b, 2) {
		t.Fatalf("fit = %v + %v x", a, b)
	}
}

func TestZScores(t *testing.T) {
	z := ZScores([]float64{2, 4, 6})
	if !near(z[0], -1) || !near(z[1], 0) || !near(z[2], 1) {
		t.Fatalf("z = %v", z)
	}
	if z := ZScores([]float64{5, 5, 5}); z[0] != 0 {
		t.Fatal("constant series must give zeros, not NaN")
	}
}
