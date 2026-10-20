package ingest

import (
	"fmt"
	"sort"
	"strings"
)

// Schema maps logical field names ("neighborhood") to column positions in a
// specific CSV file. Matching is case/punctuation-insensitive and supports
// aliases, so the engine tolerates Open Baltimore renaming columns
// (e.g. "CrimeDateTime" vs "Crime_Date_Time").
type Schema struct {
	idx    map[string]int
	Header []string
}

func normalizeHeader(h string) string {
	h = strings.TrimPrefix(h, "\ufeff")
	var b strings.Builder
	for _, r := range strings.ToLower(h) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ResolveSchema builds a Schema, failing with a helpful message (listing the
// columns actually present) if a required field cannot be found.
func ResolveSchema(header []string, fields map[string][]string, required []string) (Schema, error) {
	pos := make(map[string]int, len(header))
	for i, h := range header {
		n := normalizeHeader(h)
		if _, seen := pos[n]; !seen {
			pos[n] = i
		}
	}
	s := Schema{idx: make(map[string]int), Header: header}
	for logical, aliases := range fields {
		for _, a := range aliases {
			if i, ok := pos[normalizeHeader(a)]; ok {
				s.idx[logical] = i
				break
			}
		}
	}
	var missing []string
	for _, r := range required {
		if _, ok := s.idx[r]; !ok {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return s, fmt.Errorf("missing required column(s) %v; columns found: %v", missing, header)
	}
	return s, nil
}

// Has reports whether a logical field was found.
func (s Schema) Has(field string) bool { _, ok := s.idx[field]; return ok }

// Get returns the raw value of a logical field for a row ("" if absent).
func (s Schema) Get(row []string, field string) string {
	i, ok := s.idx[field]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}
