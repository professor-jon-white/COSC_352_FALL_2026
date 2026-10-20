package spatial

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestDistanceKnown(t *testing.T) {
	// 0.001 deg latitude ≈ 111 m everywhere.
	if d := DistanceM(39.29, -76.61, 39.291, -76.61); math.Abs(d-111.3) > 0.5 {
		t.Errorf("distance = %.2f m, want ~111.3", d)
	}
}

// The grid must return exactly what a brute-force scan returns.
func TestGridMatchesBruteForce(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	n := 5000
	lat, lon := make([]float64, n), make([]float64, n)
	for i := range lat {
		lat[i] = 39.25 + r.Float64()*0.1
		lon[i] = -76.65 + r.Float64()*0.1
	}
	g := Build(n, 100, func(i int) (float64, float64, bool) { return lat[i], lon[i], i%50 != 0 })
	for q := 0; q < 200; q++ {
		qlat, qlon := 39.25+r.Float64()*0.1, -76.65+r.Float64()*0.1
		radius := 50 + r.Float64()*400
		var want, got []int
		for i := range lat {
			if i%50 != 0 && DistanceM(qlat, qlon, lat[i], lon[i]) <= radius {
				want = append(want, i)
			}
		}
		g.Within(qlat, qlon, radius, func(i int32, _ float64) { got = append(got, int(i)) })
		sort.Ints(got)
		if len(got) != len(want) {
			t.Fatalf("query %d: grid found %d, brute force %d", q, len(got), len(want))
		}
		for k := range got {
			if got[k] != want[k] {
				t.Fatalf("query %d: mismatch", q)
			}
		}
		if len(want) > 0 {
			idx, _, ok := g.Nearest(qlat, qlon, radius)
			best := want[0]
			for _, i := range want {
				if DistanceM(qlat, qlon, lat[i], lon[i]) < DistanceM(qlat, qlon, lat[best], lon[best]) {
					best = i
				}
			}
			if !ok || int(idx) != best {
				t.Fatalf("query %d: nearest %d, want %d", q, idx, best)
			}
		}
	}
}
