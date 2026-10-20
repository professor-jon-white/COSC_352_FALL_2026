package main

import (
	"strconv"
	"strings"
	"time"
	"unicode"
)

// normalizeKey lower-cases s and drops every non-alphanumeric rune. It is used
// to match CSV headers against their aliases regardless of case, spaces or
// underscores ("Total_Incidents" == "total incidents" == "TotalIncidents").
func normalizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// normalizeNeighborhood canonicalises a neighborhood name so the two datasets
// can be joined on it: upper-case, "&" becomes "AND", every other
// non-alphanumeric rune becomes a single space, and runs of spaces collapse.
func normalizeNeighborhood(s string) string {
	s = strings.ReplaceAll(strings.ToUpper(s), "&", " AND ")
	var b strings.Builder
	pendingSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false
			b.WriteRune(r)
		} else {
			pendingSpace = true
		}
	}
	return b.String()
}

// dateLayouts lists the timestamp formats seen in Open Baltimore CSV exports
// (ArcGIS Hub writes "2025/03/04 17:30:00+00"; other exports use ISO 8601 or
// US-style dates). Go accepts fractional seconds when parsing even if the
// layout does not mention them.
var dateLayouts = []string{
	"2006/01/02 15:04:05-07",
	"2006/01/02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"1/2/2006 3:04:05 PM",
	"1/2/2006 15:04",
	"1/2/2006",
}

// timeParser parses timestamps. It remembers the layout that worked last and
// tries it first, so after the first row each parse is a single attempt. Each
// aggregator owns its own timeParser, so no locking is needed.
type timeParser struct {
	loc    *time.Location // if non-nil, parsed times are converted to this zone
	layout string
}

func (p *timeParser) parse(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if p.layout != "" {
		if t, err := time.Parse(p.layout, s); err == nil {
			return p.convert(t), true
		}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			p.layout = layout
			return p.convert(t), true
		}
	}
	// Epoch milliseconds, as produced by ArcGIS JSON exports.
	if len(s) >= 12 {
		if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
			return p.convert(time.UnixMilli(ms)), true
		}
	}
	return time.Time{}, false
}

func (p *timeParser) convert(t time.Time) time.Time {
	if p.loc != nil {
		return t.In(p.loc)
	}
	return t
}

// containsAny reports whether s contains any of the (non-empty) keywords.
func containsAny(s string, keywords []string) bool {
	for _, k := range keywords {
		if k != "" && strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// splitList splits a comma-separated list, trims and upper-cases each item and
// drops empty ones. Used for the keyword flags.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// splitFiles splits a comma-separated list of file paths, trimming blanks.
func splitFiles(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
