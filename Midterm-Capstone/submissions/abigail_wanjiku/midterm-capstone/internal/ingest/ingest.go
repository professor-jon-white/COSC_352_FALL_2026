// Package ingest holds the CSV plumbing shared by the per-dataset loaders in
// ingest/crime and ingest/requests311: a row scanner that counts and skips
// malformed rows, row-level statistics, and value normalization.
package ingest

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	_ "time/tzdata" // embed zone data so LoadLocation works on any Windows machine
)

// TimeLayout is the timestamp format both datasets use, e.g. "4/28/2025 4:31:03 PM".
const TimeLayout = "1/2/2006 3:04:05 PM"

// Baltimore is the local time zone. Records are bucketed by local month, so a
// crime at 11 PM on Dec 31 belongs to December even though it is January in UTC.
var Baltimore = mustLoadLocation("America/New_York")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// maxExamples caps how many malformed-row messages Stats keeps for debugging.
const maxExamples = 5

// Stats counts what happened to every data row in one file.
// Read == Kept + OutOfYear + NoNeighborhood + Malformed.
type Stats struct {
	Read           int // data rows seen (header excluded)
	Kept           int // rows returned to the caller
	OutOfYear      int // valid rows dropped by the year filter
	NoNeighborhood int // valid rows dropped because the neighborhood is blank
	Malformed      int // rows skipped because they could not be parsed

	Examples      []string            // first few malformed-row messages
	neighborhoods map[string]struct{} // distinct neighborhoods among kept rows
}

// Skipped is the number of rows that were read but not kept.
func (s *Stats) Skipped() int { return s.Read - s.Kept }

// Neighborhoods returns the number of distinct neighborhoods among kept rows.
func (s *Stats) Neighborhoods() int { return len(s.neighborhoods) }

// NeighborhoodSet returns the distinct neighborhoods among kept rows.
func (s *Stats) NeighborhoodSet() map[string]struct{} { return s.neighborhoods }

// Keep records that a row in neighborhood n was kept.
func (s *Stats) Keep(n string) {
	if s.neighborhoods == nil {
		s.neighborhoods = make(map[string]struct{})
	}
	s.neighborhoods[n] = struct{}{}
	s.Kept++
}

// Merge adds o's counts and neighborhoods into s. Merging chunk stats in file
// order reproduces the stats of a single sequential scan, including which
// malformed-row examples are kept.
func (s *Stats) Merge(o *Stats) {
	s.Read += o.Read
	s.Kept += o.Kept
	s.OutOfYear += o.OutOfYear
	s.NoNeighborhood += o.NoNeighborhood
	s.Malformed += o.Malformed
	for _, ex := range o.Examples {
		if len(s.Examples) < maxExamples {
			s.Examples = append(s.Examples, ex)
		}
	}
	if s.neighborhoods == nil && len(o.neighborhoods) > 0 {
		s.neighborhoods = make(map[string]struct{}, len(o.neighborhoods))
	}
	for n := range o.neighborhoods {
		s.neighborhoods[n] = struct{}{}
	}
}

func (s *Stats) malformed(line int, err error) {
	s.Malformed++
	if len(s.Examples) < maxExamples {
		s.Examples = append(s.Examples, fmt.Sprintf("line %d: %v", line, err))
	}
}

// Scanner reads a CSV one row at a time, exposing only the requested columns.
// It is used like bufio.Scanner:
//
//	for sc.Next() { row := sc.Row(); ... }
//	if err := sc.Err(); err != nil { ... }
//
// Rows the CSV parser rejects, or that are too short to contain every
// requested column, are counted in Stats.Malformed and skipped. Only I/O
// errors stop the scan.
type Scanner struct {
	Stats Stats

	r        *csv.Reader
	idx      []int    // file column index of each requested column
	max      int      // largest index in idx
	row      []string // current row, projected to the requested columns
	lineBase int      // added to the reader's line numbers to get file line numbers
	rowsHint int      // upper bound on data rows, if known; 0 otherwise
	err      error

	// Interned strings. csv.Reader returns every field of a row as a substring
	// of one string holding the whole line, so keeping even a short field
	// would keep the entire ~400-byte line alive. Repeated values (types,
	// statuses, neighborhoods) are copied once and shared instead.
	strs  map[string]string // value -> its canonical copy
	hoods map[string]string // raw neighborhood -> interned normalized form
}

// NewScanner reads the header row of r and locates the named columns.
// It returns an error if any column is missing.
func NewScanner(r io.Reader, columns ...string) (*Scanner, error) {
	cr := newCSVReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	idx, err := columnIndex(header, columns)
	if err != nil {
		return nil, err
	}
	return newScanner(cr, idx, 0), nil
}

func newCSVReader(r io.Reader) *csv.Reader {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // ungeocoded rows omit the trailing x,y fields
	cr.ReuseRecord = true
	return cr
}

// columnIndex returns the position in header of each name in columns.
func columnIndex(header, columns []string) ([]int, error) {
	pos := make(map[string]int, len(header))
	for i, h := range header {
		pos[strings.TrimPrefix(strings.TrimSpace(h), "\xef\xbb\xbf")] = i
	}
	idx := make([]int, len(columns))
	for i, c := range columns {
		j, ok := pos[c]
		if !ok {
			return nil, fmt.Errorf("missing column %q", c)
		}
		idx[i] = j
	}
	return idx, nil
}

// newScanner returns a Scanner over data rows of cr (no header) that projects
// the columns at idx. lineBase is the file line number just before cr's first
// line, so malformed-row messages report positions in the whole file.
func newScanner(cr *csv.Reader, idx []int, lineBase int) *Scanner {
	sc := &Scanner{
		r: cr, idx: idx, row: make([]string, len(idx)), lineBase: lineBase,
		strs: make(map[string]string), hoods: make(map[string]string),
	}
	for _, j := range idx {
		sc.max = max(sc.max, j)
	}
	return sc
}

// Next advances to the next well-formed row. It returns false at end of
// input or on an I/O error; check Err afterwards.
func (sc *Scanner) Next() bool {
	for {
		rec, err := sc.r.Read()
		if err == io.EOF {
			return false
		}
		var perr *csv.ParseError
		if errors.As(err, &perr) {
			sc.Stats.Read++
			sc.Stats.malformed(sc.lineBase+perr.Line, perr.Err)
			continue
		}
		if err != nil {
			sc.err = err
			return false
		}
		sc.Stats.Read++
		if len(rec) <= sc.max {
			sc.Stats.malformed(sc.line(), fmt.Errorf("only %d fields", len(rec)))
			continue
		}
		for i, j := range sc.idx {
			sc.row[i] = strings.TrimSpace(rec[j])
		}
		return true
	}
}

// Row returns the current row's requested columns, trimmed, in the order they
// were passed to NewScanner. The slice is reused by the next call to Next.
func (sc *Scanner) Row() []string { return sc.row }

// Malformed counts the current row as malformed. Loaders call it when a field
// fails to parse (for example, a bad date).
func (sc *Scanner) Malformed(err error) { sc.Stats.malformed(sc.line(), err) }

// Err returns the I/O error that stopped the scan, if any.
func (sc *Scanner) Err() error { return sc.err }

// RowsHint returns an upper bound on the number of data rows this scanner
// will produce, or 0 if unknown. Loaders use it to pre-size result slices.
func (sc *Scanner) RowsHint() int { return sc.rowsHint }

// Intern returns a copy of s that does not share memory with the CSV line it
// came from, reusing one copy per distinct value. Use it for any field kept
// after the row is processed.
func (sc *Scanner) Intern(s string) string {
	if v, ok := sc.strs[s]; ok {
		return v
	}
	v := strings.Clone(s)
	sc.strs[v] = v
	return v
}

// Neighborhood returns the interned, normalized form of a raw neighborhood
// value, normalizing each distinct raw spelling only once.
func (sc *Scanner) Neighborhood(raw string) string {
	if v, ok := sc.hoods[raw]; ok {
		return v
	}
	v := sc.Intern(NormalizeNeighborhood(raw))
	sc.hoods[strings.Clone(raw)] = v
	return v
}

func (sc *Scanner) line() int {
	line, _ := sc.r.FieldPos(0)
	return sc.lineBase + line
}

// NormalizeNeighborhood trims a neighborhood name, collapses internal runs of
// whitespace, and uppercases it, so "  Belair-Edison " and "BELAIR-EDISON"
// compare equal.
func NormalizeNeighborhood(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

// ParseTime parses a timestamp in TimeLayout interpreted in loc and returns it
// in Baltimore time.
func ParseTime(s string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation(TimeLayout, s, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(Baltimore), nil
}
