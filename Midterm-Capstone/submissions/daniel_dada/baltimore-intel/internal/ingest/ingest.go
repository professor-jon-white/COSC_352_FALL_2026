package ingest

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"baltimore-intel/internal/model"
	"baltimore-intel/internal/normalize"
)

const dateLayout = "1/2/2006 3:04:05 PM"

// Stats reports what happened to every row, so nothing disappears silently.
type Stats struct {
	Read, Kept                              int
	BadDate, WrongYear, BlankKey, Malformed int
}

func (s Stats) String() string {
	return fmt.Sprintf("read=%d kept=%d dropped(bad date=%d, other year=%d, blank neighborhood=%d, malformed=%d)",
		s.Read, s.Kept, s.BadDate, s.WrongYear, s.BlankKey, s.Malformed)
}

// row wraps one CSV record with the header index so we can look up by name.
type row struct {
	rec []string
	idx map[string]int
}

func (r row) get(col string) string {
	i, ok := r.idx[col]
	if !ok || i >= len(r.rec) {
		return ""
	}
	return strings.TrimSpace(r.rec[i])
}

// streamCSV reads one record at a time (never ReadAll), so memory stays flat
// no matter how big the file is.
func streamCSV(path string, required []string, handle func(row)) (malformed int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1 // tolerate rows with odd column counts
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return 0, fmt.Errorf("reading header of %s: %w", path, err)
	}
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))] = i
	}
	for _, c := range required {
		if _, ok := idx[c]; !ok {
			return 0, fmt.Errorf("%s: missing required column %q", path, c)
		}
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				malformed++
				continue
			}
			return malformed, err
		}
		handle(row{rec: rec, idx: idx})
	}
	return malformed, nil
}

func ReadCrime(path string, year int) ([]model.CrimeRecord, Stats, error) {
	var out []model.CrimeRecord
	var st Stats
	cols := []string{"CrimeDateTime", "Description", "Neighborhood"}
	mal, err := streamCSV(path, cols, func(r row) {
		st.Read++
		t, err := time.Parse(dateLayout, r.get("CrimeDateTime"))
		if err != nil {
			st.BadDate++
			return
		}
		if t.Year() != year {
			st.WrongYear++
			return
		}
		key := normalize.Key(r.get("Neighborhood"))
		if key == "" {
			st.BlankKey++
			return
		}
		out = append(out, model.CrimeRecord{
			When:         t,
			Description:  strings.ToUpper(r.get("Description")),
			Neighborhood: key,
		})
		st.Kept++
	})
	st.Malformed = mal
	return out, st, err
}

func ReadRequests(path string, year int) ([]model.ServiceRequest, Stats, error) {
	var out []model.ServiceRequest
	var st Stats
	cols := []string{"CreatedDate", "SRType", "Agency", "Neighborhood"}
	mal, err := streamCSV(path, cols, func(r row) {
		st.Read++
		t, err := time.Parse(dateLayout, r.get("CreatedDate"))
		if err != nil {
			st.BadDate++
			return
		}
		if t.Year() != year {
			st.WrongYear++
			return
		}
		key := normalize.Key(r.get("Neighborhood"))
		if key == "" {
			st.BlankKey++
			return
		}
		out = append(out, model.ServiceRequest{
			Created:      t,
			Type:         r.get("SRType"),
			Agency:       r.get("Agency"),
			Neighborhood: key,
		})
		st.Kept++
	})
	st.Malformed = mal
	return out, st, err
}
