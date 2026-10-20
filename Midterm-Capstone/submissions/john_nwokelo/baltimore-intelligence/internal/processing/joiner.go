package processing

import (
	"baltimore-intelligence/internal/models"
	"context"
	"runtime"
	"sync"
)

func inRing(x, y float64, ring []models.Point) bool {
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}
func inside(n models.Neighborhood, x, y float64) bool {
	if x < n.MinX || x > n.MaxX || y < n.MinY || y > n.MaxY {
		return false
	}
	for _, p := range n.Polygons {
		if len(p) == 0 || !inRing(x, y, p[0]) {
			continue
		}
		inHole := false
		for _, hole := range p[1:] {
			if inRing(x, y, hole) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

// Join implements bounded fan-out/fan-in; neighborhoods are read-only, output has a single collector.
func Join(ctx context.Context, requests []models.Request, ns []models.Neighborhood, workers int) ([]models.Joined, int) {
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	jobs := make(chan models.Request, workers*2)
	results := make(chan models.Joined, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range jobs {
				match := ""
				for _, n := range ns {
					if inside(n, r.Lon, r.Lat) {
						match = n.Name
						break
					}
				}
				select {
				case results <- models.Joined{Request: r, Neighborhood: match}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, r := range requests {
			select {
			case jobs <- r:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()
	out := make([]models.Joined, 0, len(requests))
	unmatched := 0
	for j := range results {
		if j.Neighborhood == "" {
			unmatched++
		}
		out = append(out, j)
	}
	return out, unmatched
}
