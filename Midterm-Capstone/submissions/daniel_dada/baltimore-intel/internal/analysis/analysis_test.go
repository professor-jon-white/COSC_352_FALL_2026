package analysis

import (
	"math"
	"testing"
)

func TestPearson(t *testing.T) {
	if r := pearson([]float64{1, 2, 3, 4}, []float64{2, 4, 6, 8}); math.Abs(r-1) > 1e-9 {
		t.Errorf("perfect positive: r = %v, want 1", r)
	}
	if r := pearson([]float64{1, 2, 3, 4}, []float64{8, 6, 4, 2}); math.Abs(r+1) > 1e-9 {
		t.Errorf("perfect negative: r = %v, want -1", r)
	}
	if r := pearson([]float64{1, 1, 1}, []float64{1, 2, 3}); r != 0 {
		t.Errorf("constant input: r = %v, want 0", r)
	}
}
