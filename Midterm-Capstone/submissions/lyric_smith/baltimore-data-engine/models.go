package main

import (
	"strings"
	"time"
)

// ServiceRequest represents one row of the Baltimore 311 dataset.
type ServiceRequest struct {
	SRNumber     string
	SRType       string
	Neighborhood string
	District     string
	CreatedDate  time.Time
}

// CrimeIncident represents one row of the Baltimore crime dataset.
type CrimeIncident struct {
	CrimeCode    string
	Description  string
	Neighborhood string
	District     string
	CrimeDate    time.Time
}

// normalizeKey is the join key between the two datasets: it trims
// whitespace and uppercases, so "Fells Point" and "fells point " count
// as the same neighborhood.
func normalizeKey(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
