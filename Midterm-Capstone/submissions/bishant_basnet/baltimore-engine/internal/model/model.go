// Package model holds the typed, normalized records produced by ingestion.
// Every string field is interned (see ingest.Interner) so that millions of
// rows share a small set of backing strings instead of each row pinning its
// original CSV line in memory.
package model

import "time"

// CrimeRecord is one row of Open Baltimore "Part 1 Crime Data".
type CrimeRecord struct {
	When         time.Time
	HasTime      bool
	Neighborhood string
	Description  string
	Weapon       string
	District     string
}

// ServiceRequest is one row of Open Baltimore "311 Customer Service Requests".
type ServiceRequest struct {
	Created      time.Time
	Closed       time.Time
	HasCreated   bool
	HasClosed    bool
	Neighborhood string
	Type         string
	Agency       string
}
