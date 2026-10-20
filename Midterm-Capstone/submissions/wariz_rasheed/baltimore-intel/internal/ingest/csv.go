// Package ingest streams raw CSV files from the Open Baltimore portal and
// converts them to compact model records using a pool of worker goroutines.
package ingest

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// batchSize is how many raw records the reader hands to a worker at once.
// Batching keeps channel overhead negligible compared with parsing work.
const batchSize = 2048

// Header maps a column name to its position in the file.
type Header map[string]int

// Index returns the position of a required column.
func (h Header) Index(name string) (int, error) {
	i, ok := h[name]
	if !ok {
		return -1, fmt.Errorf("required column %q not found in header", name)
	}
	return i, nil
}

// indexer collects the first missing-column error while a parser is built.
type indexer struct {
	h   Header
	err error
}

func (x *indexer) get(name string) int {
	i, err := x.h.Index(name)
	if err != nil && x.err == nil {
		x.err = err
	}
	return i
}

// field returns a trimmed column value, or "" if the row is short.
func field(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

// Interner de-duplicates repeated strings (neighborhood names, offense names,
// request types). Each worker owns one, so it needs no lock. Interning also
// matters for memory: encoding/csv returns fields that are slices of one large
// per-row string, so keeping such a field alive would pin the whole row.
// Interning stores a small private copy instead.
type Interner struct {
	m map[string]string
}

// NewInterner returns an empty interner.
func NewInterner() *Interner {
	return &Interner{m: make(map[string]string, 256)}
}

// Intern returns a canonical copy of s.
func (in *Interner) Intern(s string) string {
	if v, ok := in.m[s]; ok {
		return v
	}
	c := strings.Clone(s)
	in.m[c] = c
	return c
}

// RowParser converts one raw record into a typed row. It returns false to
// reject a row. Parsers must be safe for concurrent use (they hold no
// mutable state); the Interner argument is private to the calling worker.
type RowParser[T any] func(rec []string, in *Interner) (T, bool)

// Stats describes one load.
type Stats struct {
	Path      string
	Workers   int
	Read      int // data rows read
	Kept      int // rows parsed successfully
	Rejected  int // rows the parser refused (bad date, missing field, ...)
	Malformed int // rows the CSV reader could not split
}

// loadResult is one worker's private output.
type loadResult[T any] struct {
	rows     []T
	read     int
	rejected int
}

// Load streams path through a reader -> worker-pool pipeline.
//
//	reader goroutine (this one): csv.Reader -> batches of raw records
//	N worker goroutines:         batch -> parse -> private slice of T
//	fan-in:                      private slices are concatenated at the end
//
// The file is never held in memory as a whole; only parsed rows are kept.
func Load[T any](path string, workers int, newParser func(Header) (RowParser[T], error)) ([]T, Stats, error) {
	st := Stats{Path: path}
	if workers < 1 {
		workers = 1
	}
	st.Workers = workers

	f, err := os.Open(path)
	if err != nil {
		return nil, st, err
	}
	defer f.Close()

	cr := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	cr.LazyQuotes = true

	hdr, err := cr.Read()
	if err != nil {
		return nil, st, fmt.Errorf("%s: reading header: %w", path, err)
	}
	h := make(Header, len(hdr))
	for i, name := range hdr {
		name = strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")) // strip UTF-8 BOM
		h[name] = i
	}
	parse, err := newParser(h)
	if err != nil {
		return nil, st, fmt.Errorf("%s: %w", path, err)
	}

	batches := make(chan [][]string, workers*2)
	results := make(chan loadResult[T], workers)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := NewInterner()
			var res loadResult[T]
			for b := range batches {
				for _, rec := range b {
					res.read++
					v, ok := parse(rec, in)
					if !ok {
						res.rejected++
						continue
					}
					res.rows = append(res.rows, v)
				}
			}
			results <- res
		}()
	}

	batch := make([][]string, 0, batchSize)
	var readErr error
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				st.Malformed++
				continue
			}
			readErr = err
			break
		}
		batch = append(batch, rec)
		if len(batch) == batchSize {
			batches <- batch
			batch = make([][]string, 0, batchSize)
		}
	}
	if len(batch) > 0 {
		batches <- batch
	}
	close(batches)
	wg.Wait()
	close(results)

	var parts [][]T
	total := 0
	for res := range results {
		st.Read += res.read
		st.Rejected += res.rejected
		parts = append(parts, res.rows)
		total += len(res.rows)
	}
	if readErr != nil {
		return nil, st, fmt.Errorf("%s: %w", path, readErr)
	}
	all := make([]T, 0, total)
	for _, p := range parts {
		all = append(all, p...)
	}
	st.Kept = len(all)
	return all, st, nil
}
