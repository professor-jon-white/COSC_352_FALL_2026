package report

import (
	"strings"

	"capstone/internal/model"
)

// sanitationTypes are the 311 request types counted as sanitation or dumping
// complaints: reports of trash, dumping, dirty streets and alleys, and rats.
// Scheduled services (bulk pickup, recycling, can replacement) and
// city-initiated "Proactive" work are deliberately left out, so the count
// reflects problems residents reported.
var sanitationTypes = map[string]bool{
	"HCD-Illegal Dumping":     true,
	"HCD-Sanitation Property": true,
	"SW-Dirty Alley":          true,
	"SW-Dirty Street":         true,
	"SW-Cleaning":             true,
	"SW-Rat Rubout":           true,
}

func isSanitation(requestType string) bool { return sanitationTypes[requestType] }

// isStreetlight reports whether a 311 request type is about street lighting:
// BGE outages and faults, damaged or missing poles, inadequate lighting.
// Red-light cameras also contain "Light" and are excluded by the patterns.
func isStreetlight(requestType string) bool {
	t := strings.ToLower(requestType)
	if strings.HasPrefix(t, "test-") {
		return false
	}
	return strings.Contains(t, "stlight") || strings.Contains(t, "street light")
}

// isProactive reports whether a request was created by the city during its
// own sweeps (e.g. "SW-Rat Rubout Proactive") rather than reported by a
// resident.
func isProactive(requestType string) bool {
	return strings.Contains(requestType, "Proactive")
}

// countTypes sums the counts in byType whose type satisfies keep.
func countTypes(byType map[string]int, keep func(string) bool) int {
	n := 0
	for t, c := range byType {
		if keep(t) {
			n += c
		}
	}
	return n
}

// residentRequests is a summary's 311 volume excluding city-initiated work.
func residentRequests(s *model.Summary) int {
	return s.RequestCount - countTypes(s.RequestsByType, isProactive)
}

// topResidentType returns the most common resident-reported request type.
func topResidentType(s *model.Summary) (string, int) {
	var best string
	var bestN int
	for t, n := range s.RequestsByType {
		if isProactive(t) {
			continue
		}
		if n > bestN || (n == bestN && t < best) {
			best, bestN = t, n
		}
	}
	return best, bestN
}
