// Package crime reads the Baltimore NIBRS Group A crime CSV (data/crime.csv)
// into model.Crime records.
package crime

import (
	"fmt"
	"io"
	"time"

	"capstone/internal/ingest"
	"capstone/internal/model"
)

// Columns read from the file, in the order the scanner returns them.
const (
	colDateTime = iota
	colNeighborhood
	colDescription
	colInsideOutside
)

var columns = []string{"CrimeDateTime", "Neighborhood", "Description", "Inside_Outside"}

// Load reads crimes from r and keeps those that occurred in year (Baltimore
// time) and have a neighborhood. Malformed rows are counted and skipped; only
// a missing column or an I/O error returns a non-nil error.
func Load(r io.Reader, year int) ([]model.Crime, ingest.Stats, error) {
	sc, err := ingest.NewScanner(r, columns...)
	if err != nil {
		return nil, ingest.Stats{}, fmt.Errorf("crime: %w", err)
	}
	crimes := scan(sc, year)
	if err := sc.Err(); err != nil {
		return nil, sc.Stats, fmt.Errorf("crime: %w", err)
	}
	return crimes, sc.Stats, nil
}

// LoadConcurrent is Load using a pool of workers. It returns the same
// crimes, in the same order, and the same stats.
func LoadConcurrent(r io.Reader, year int, opts ingest.ConcurrentOptions) ([]model.Crime, ingest.Stats, error) {
	crimes, stats, err := ingest.Concurrent(r, columns, opts, func(sc *ingest.Scanner) []model.Crime {
		return scan(sc, year)
	})
	if err != nil {
		return nil, stats, fmt.Errorf("crime: %w", err)
	}
	return crimes, stats, nil
}

// scan converts and filters every row of sc. It touches only sc and its own
// result slice, so concurrent calls on different scanners are safe.
func scan(sc *ingest.Scanner, year int) []model.Crime {
	crimes := make([]model.Crime, 0, sc.RowsHint())
	for sc.Next() {
		row := sc.Row()

		// CrimeDateTime is recorded in UTC: midnight local appears as 4 or 5 AM.
		t, err := ingest.ParseTime(row[colDateTime], time.UTC)
		if err != nil {
			sc.Malformed(fmt.Errorf("CrimeDateTime: %w", err))
			continue
		}
		if t.Year() != year {
			sc.Stats.OutOfYear++
			continue
		}
		n := sc.Neighborhood(row[colNeighborhood])
		if n == "" {
			sc.Stats.NoNeighborhood++
			continue
		}

		sc.Stats.Keep(n)
		crimes = append(crimes, model.Crime{
			OccurredAt:   t,
			Neighborhood: n,
			CrimeType:    sc.Intern(row[colDescription]),
			Outdoor:      row[colInsideOutside] == "O",
		})
	}
	return crimes
}
