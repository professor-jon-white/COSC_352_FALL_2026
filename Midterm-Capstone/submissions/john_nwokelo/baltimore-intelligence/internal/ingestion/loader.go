package ingestion

import (
	"baltimore-intelligence/internal/models"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type collection struct {
	Type     string `json:"type"`
	Features []struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Geometry   struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func str(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if b, ok := m[k]; ok {
			var s string
			if json.Unmarshal(b, &s) == nil && strings.TrimSpace(s) != "" {
				return s
			}
			var n float64
			if json.Unmarshal(b, &n) == nil {
				return strconv.FormatFloat(n, 'f', -1, 64)
			}
		}
	}
	return ""
}
func number(m map[string]json.RawMessage, keys ...string) float64 {
	s := str(m, keys...)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
func load(path string) (collection, error) {
	var c collection
	b, e := os.ReadFile(path)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	if e != nil {
		return c, fmt.Errorf("invalid GeoJSON %s: %w", path, e)
	}
	if c.Type != "FeatureCollection" {
		return c, fmt.Errorf("%s must contain a GeoJSON FeatureCollection", path)
	}
	return c, nil
}
func Requests(path string) ([]models.Request, error) {
	c, e := load(path)
	if e != nil {
		return nil, e
	}
	out := make([]models.Request, 0, len(c.Features))
	for _, f := range c.Features {
		if f.Geometry.Type != "Point" {
			continue
		}
		var coords []float64
		if json.Unmarshal(f.Geometry.Coordinates, &coords) != nil || len(coords) < 2 {
			continue
		}
		lon, lat := coords[0], coords[1]
		if math.IsNaN(lon) || math.IsNaN(lat) || lon < -180 || lon > 180 || lat < -90 || lat > 90 || lon == 0 || lat == 0 {
			continue
		}
		p := f.Properties
		out = append(out, models.Request{ID: str(p, "SRRecordID", "ServiceRequestNum", "id"), Category: str(p, "SRType", "category"), Status: str(p, "SRStatus", "status"), Neighborhood: str(p, "Neighborhood", "neighborhood"), Lon: lon, Lat: lat, Created: parseDate(p, "CreatedDate", "CreationDate", "DateCreated", "created_date"), Closed: parseDate(p, "CloseDate", "ClosedDate", "DateClosed", "closed_date")})
	}
	return out, nil
}
func Neighborhoods(path string) ([]models.Neighborhood, error) {
	c, e := load(path)
	if e != nil {
		return nil, e
	}
	out := make([]models.Neighborhood, 0, len(c.Features))
	for _, f := range c.Features {
		name := str(f.Properties, "nbrdesc", "Name", "NAME", "name")
		if name == "" {
			continue
		}
		var polys [][][]models.Point
		switch f.Geometry.Type {
		case "Polygon":
			var p [][]models.Point
			if json.Unmarshal(f.Geometry.Coordinates, &p) != nil {
				continue
			}
			polys = [][][]models.Point{p}
		case "MultiPolygon":
			if json.Unmarshal(f.Geometry.Coordinates, &polys) != nil {
				continue
			}
		default:
			continue
		}
		n := models.Neighborhood{Name: name, Polygons: polys, MinX: math.Inf(1), MaxX: math.Inf(-1), MinY: math.Inf(1), MaxY: math.Inf(-1)}
		for _, poly := range polys {
			for _, ring := range poly {
				for _, pt := range ring {
					n.MinX = math.Min(n.MinX, pt[0])
					n.MaxX = math.Max(n.MaxX, pt[0])
					n.MinY = math.Min(n.MinY, pt[1])
					n.MaxY = math.Max(n.MaxY, pt[1])
				}
			}
		}
		if !math.IsInf(n.MinX, 0) {
			out = append(out, n)
		}
	}
	return out, nil
}
