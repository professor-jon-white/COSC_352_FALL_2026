package processing

import (
	"baltimore-intelligence/internal/models"
	"context"
	"testing"
)

func TestJoinPolygonWithHole(t *testing.T) {
	n := models.Neighborhood{Name: "A", MinX: 0, MaxX: 10, MinY: 0, MaxY: 10, Polygons: [][][]models.Point{{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}, {{4, 4}, {6, 4}, {6, 6}, {4, 6}, {4, 4}}}}}
	rs := []models.Request{{Lon: 1, Lat: 1}, {Lon: 5, Lat: 5}, {Lon: 20, Lat: 20}}
	j, unmatched := Join(context.Background(), rs, []models.Neighborhood{n}, 2)
	if len(j) != 3 || unmatched != 2 {
		t.Fatalf("got %d results, %d unmatched", len(j), unmatched)
	}
}
