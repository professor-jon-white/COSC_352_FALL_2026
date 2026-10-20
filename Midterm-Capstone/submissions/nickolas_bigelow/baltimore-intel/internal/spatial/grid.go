// Package spatial provides a uniform-grid index for radius and nearest-point
// queries over a few hundred thousand points.
//
// Points are projected to local meters with an equirectangular projection
// centred on Baltimore (error < 0.1% across the city) so distance checks are
// two multiplications instead of trigonometry. The index is immutable after
// Build, so any number of goroutines can query it without locks.
package spatial

import (
	"math"
	"sort"
)

const (
	refLat       = 39.29
	metersPerDeg = 111_320.0
)

var cosRef = math.Cos(refLat * math.Pi / 180)

// Project converts lat/lon to local x/y meters.
func Project(lat, lon float64) (x, y float64) {
	return lon * metersPerDeg * cosRef, lat * metersPerDeg
}

// DistanceM is the local-planar distance between two points in meters.
func DistanceM(lat1, lon1, lat2, lon2 float64) float64 {
	x1, y1 := Project(lat1, lon1)
	x2, y2 := Project(lat2, lon2)
	return math.Hypot(x1-x2, y1-y2)
}

type cellKey struct{ cx, cy int32 }

// Grid indexes points by integer cell. Point IDs are int32 positions in the
// caller's slice, so the index holds no copies of the records themselves.
type Grid struct {
	cell  float64
	xs    []float64
	ys    []float64
	cells map[cellKey][]int32
}

// Build indexes n points. get(i) returns point i and ok=false to skip it
// (e.g. records with no coordinates).
func Build(n int, cellM float64, get func(i int) (lat, lon float64, ok bool)) *Grid {
	g := &Grid{cell: cellM, xs: make([]float64, n), ys: make([]float64, n), cells: map[cellKey][]int32{}}
	for i := 0; i < n; i++ {
		lat, lon, ok := get(i)
		if !ok {
			g.xs[i] = math.NaN()
			continue
		}
		x, y := Project(lat, lon)
		g.xs[i], g.ys[i] = x, y
		k := g.key(x, y)
		g.cells[k] = append(g.cells[k], int32(i))
	}
	return g
}

func (g *Grid) key(x, y float64) cellKey {
	return cellKey{int32(math.Floor(x / g.cell)), int32(math.Floor(y / g.cell))}
}

// Within calls fn for every indexed point within radius meters.
func (g *Grid) Within(lat, lon, radius float64, fn func(i int32, d float64)) {
	x, y := Project(lat, lon)
	r := int32(math.Ceil(radius / g.cell))
	c := g.key(x, y)
	r2 := radius * radius
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			for _, i := range g.cells[cellKey{c.cx + dx, c.cy + dy}] {
				ddx, ddy := g.xs[i]-x, g.ys[i]-y
				if d2 := ddx*ddx + ddy*ddy; d2 <= r2 {
					fn(i, math.Sqrt(d2))
				}
			}
		}
	}
}

// CountWithin is Within without a callback.
func (g *Grid) CountWithin(lat, lon, radius float64) int {
	n := 0
	g.Within(lat, lon, radius, func(int32, float64) { n++ })
	return n
}

// Nearest returns the closest point within maxRadius.
func (g *Grid) Nearest(lat, lon, maxRadius float64) (idx int32, dist float64, ok bool) {
	best, bestD := int32(-1), math.Inf(1)
	g.Within(lat, lon, maxRadius, func(i int32, d float64) {
		if d < bestD || (d == bestD && i < best) {
			best, bestD = i, d
		}
	})
	return best, bestD, best >= 0
}

// KNearest returns up to k closest point IDs within maxRadius, nearest first.
func (g *Grid) KNearest(lat, lon, maxRadius float64, k int) []int32 {
	type hit struct {
		i int32
		d float64
	}
	var hits []hit
	g.Within(lat, lon, maxRadius, func(i int32, d float64) { hits = append(hits, hit{i, d}) })
	sort.Slice(hits, func(a, b int) bool {
		if hits[a].d != hits[b].d {
			return hits[a].d < hits[b].d
		}
		return hits[a].i < hits[b].i
	})
	out := make([]int32, 0, k)
	for j := 0; j < len(hits) && j < k; j++ {
		out = append(out, hits[j].i)
	}
	return out
}
