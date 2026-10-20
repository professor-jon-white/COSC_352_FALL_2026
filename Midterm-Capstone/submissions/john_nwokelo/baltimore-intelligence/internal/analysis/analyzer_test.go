package analysis

import (
	"baltimore-intelligence/internal/models"
	"testing"
)

func TestAnalyze(t *testing.T) {
	items := []models.Joined{{Neighborhood: "A", Request: models.Request{Category: "Pothole", Status: "Closed", Created: 1000, Closed: 86401000}}, {Neighborhood: "A", Request: models.Request{Category: "Pothole", Status: "Open"}}, {Neighborhood: "B", Request: models.Request{Category: "Trash", Status: "Open"}}, {Neighborhood: "", Request: models.Request{Status: "Open"}}}
	a := Analyze(items, 3)
	if len(a) != 2 {
		t.Fatalf("expected 2 neighborhoods, got %d", len(a))
	}
	for _, v := range a {
		if v.Name == "A" {
			if v.Total != 2 || v.Open != 1 || v.Closed != 1 || v.AvgDays() != 1 {
				t.Fatalf("wrong A stats: %+v", v)
			}
		}
	}
}

// If the input has no valid closure times, the priority must use only
// measurable components rather than quietly counting unavailable time as zero.
func TestPriorityFallbackWithoutDates(t *testing.T) {
	items := []models.Joined{
		{Neighborhood: "A", Request: models.Request{Status: "Open"}},
		{Neighborhood: "B", Request: models.Request{Status: "Closed"}},
	}
	results := Analyze(items, 2)
	if len(results) != 2 {
		t.Fatal("expected two neighborhoods")
	}
	for _, s := range results {
		if s.Name == "A" && s.Priority != 100 {
			t.Fatalf("A expected 100, got %v", s.Priority)
		}
		if s.Name == "B" && s.Priority != 50 {
			t.Fatalf("B expected 50, got %v", s.Priority)
		}
	}
}
