// Package requests311 reads the Baltimore 311 customer service requests CSV
// (data/requests_311.csv) into model.ServiceRequest records.
package requests311

import (
	"fmt"
	"io"
	"time"

	"capstone/internal/ingest"
	"capstone/internal/model"
)

// Columns read from the file, in the order the scanner returns them.
const (
	colCreated = iota
	colNeighborhood
	colType
	colStatus
	colClosed
)

var columns = []string{"CreatedDate", "Neighborhood", "SRType", "SRStatus", "CloseDate"}

// Load reads service requests from r and keeps those created in year and
// having a neighborhood. Malformed rows are counted and skipped; only a
// missing column or an I/O error returns a non-nil error.
func Load(r io.Reader, year int) ([]model.ServiceRequest, ingest.Stats, error) {
	sc, err := ingest.NewScanner(r, columns...)
	if err != nil {
		return nil, ingest.Stats{}, fmt.Errorf("requests311: %w", err)
	}
	requests := scan(sc, year)
	if err := sc.Err(); err != nil {
		return nil, sc.Stats, fmt.Errorf("requests311: %w", err)
	}
	return requests, sc.Stats, nil
}

// LoadConcurrent is Load using a pool of workers. It returns the same
// requests, in the same order, and the same stats.
func LoadConcurrent(r io.Reader, year int, opts ingest.ConcurrentOptions) ([]model.ServiceRequest, ingest.Stats, error) {
	requests, stats, err := ingest.Concurrent(r, columns, opts, func(sc *ingest.Scanner) []model.ServiceRequest {
		return scan(sc, year)
	})
	if err != nil {
		return nil, stats, fmt.Errorf("requests311: %w", err)
	}
	return requests, stats, nil
}

// scan converts and filters every row of sc. It touches only sc and its own
// result slice, so concurrent calls on different scanners are safe.
func scan(sc *ingest.Scanner, year int) []model.ServiceRequest {
	requests := make([]model.ServiceRequest, 0, sc.RowsHint())
	for sc.Next() {
		row := sc.Row()

		// 311 timestamps are already local time.
		created, err := ingest.ParseTime(row[colCreated], ingest.Baltimore)
		if err != nil {
			sc.Malformed(fmt.Errorf("CreatedDate: %w", err))
			continue
		}
		var closed time.Time // stays zero for open requests
		if row[colClosed] != "" {
			if closed, err = ingest.ParseTime(row[colClosed], ingest.Baltimore); err != nil {
				sc.Malformed(fmt.Errorf("CloseDate: %w", err))
				continue
			}
		}
		if created.Year() != year {
			sc.Stats.OutOfYear++
			continue
		}
		n := sc.Neighborhood(row[colNeighborhood])
		if n == "" {
			sc.Stats.NoNeighborhood++
			continue
		}

		sc.Stats.Keep(n)
		requests = append(requests, model.ServiceRequest{
			CreatedAt:    created,
			Neighborhood: n,
			RequestType:  sc.Intern(row[colType]),
			Status:       sc.Intern(row[colStatus]),
			ClosedAt:     closed,
		})
	}
	return requests
}
