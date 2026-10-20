package main

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

// ---------------------------------------------------------------------------
// Source description (what columns we need and what they may be called)
// ---------------------------------------------------------------------------

// FieldSpec describes one logical field and the CSV header names it may have.
type FieldSpec struct {
	Name     string
	Aliases  []string
	Required bool
}

// SourceSpec is the schema of one dataset: an ordered list of logical fields.
// The reader extracts exactly these fields (in this order) from every row, so
// the other 10-20 columns of the raw files are never copied out of the CSV
// reader's buffers.
type SourceSpec struct {
	Name   string
	Fields []FieldSpec
}

// RawRow holds the selected fields of one CSV row, indexed like SourceSpec.Fields.
type RawRow struct {
	Values []string
}

// RowStats counts what happened to rows handed to an aggregator.
type RowStats struct {
	Accepted       int
	NoNeighborhood int
	BadDate        int
	OutOfRange     int
}

func (s *RowStats) merge(o RowStats) {
	s.Accepted += o.Accepted
	s.NoNeighborhood += o.NoNeighborhood
	s.BadDate += o.BadDate
	s.OutOfRange += o.OutOfRange
}

// Rejected is the number of rows an aggregator refused.
func (s RowStats) Rejected() int {
	return s.NoNeighborhood + s.BadDate + s.OutOfRange
}

// IngestStats summarises one pipeline run.
type IngestStats struct {
	Files    int
	RowsRead int
	BadCSV   int // lines the CSV parser could not decode
	Rows     RowStats
}

// Aggregator is the Strategy a pipeline worker runs on every row. Each worker
// owns one private Aggregator (thread confinement), so implementations need no
// locks; the per-worker results are combined with Merge after the workers stop.
type Aggregator interface {
	AddRow(vals []string)
	Merge(other Aggregator)
	Stats() RowStats
}

// ---------------------------------------------------------------------------
// Stage 1: reader
// ---------------------------------------------------------------------------

// resolveColumns maps every logical field to a column index of the header
// (-1 when an optional field is absent) and fails if a required one is missing.
func resolveColumns(header []string, spec SourceSpec) ([]int, error) {
	norm := make(map[string]int, len(header))
	for i, h := range header {
		k := normalizeKey(h)
		if _, dup := norm[k]; !dup {
			norm[k] = i
		}
	}
	cols := make([]int, len(spec.Fields))
	var missing []string
	for i, f := range spec.Fields {
		cols[i] = -1
		for _, a := range f.Aliases {
			if idx, ok := norm[normalizeKey(a)]; ok {
				cols[i] = idx
				break
			}
		}
		if cols[i] < 0 && f.Required {
			missing = append(missing, fmt.Sprintf("%s (any of: %s)", f.Name, strings.Join(f.Aliases, ", ")))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required column(s) not found for the %s dataset: %s; header was: %s",
			spec.Name, strings.Join(missing, "; "), strings.Join(header, ","))
	}
	return cols, nil
}

// readFile streams one CSV file and sends its rows to out in batches. Only the
// selected columns are kept. Batches share one backing array of strings, so a
// batch costs two allocations regardless of its size. The reader never holds
// more than one batch itself; the bounded channel limits how many more can be
// in flight, which is what keeps memory usage independent of the file size.
func readFile(path string, spec SourceSpec, batchSize int, out chan<- []RawRow, st *IngestStats) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1 // tolerate ragged rows
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: reading header: %w", path, err)
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff") // UTF-8 byte-order mark
	}
	cols, err := resolveColumns(header, spec)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	nf := len(spec.Fields)
	newBatch := func() ([]RawRow, []string) {
		return make([]RawRow, 0, batchSize), make([]string, 0, batchSize*nf)
	}
	rows, backing := newBatch()
	st.Files++

	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				st.BadCSV++
				continue
			}
			return fmt.Errorf("%s: %w", path, err)
		}
		st.RowsRead++
		start := len(backing)
		for _, c := range cols {
			if c >= 0 && c < len(rec) {
				backing = append(backing, rec[c])
			} else {
				backing = append(backing, "")
			}
		}
		rows = append(rows, RawRow{Values: backing[start : start+nf : start+nf]})
		if len(rows) == batchSize {
			out <- rows // blocks when the workers are behind: back-pressure
			rows, backing = newBatch()
		}
	}
	if len(rows) > 0 {
		out <- rows
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stage 2 + 3: parallel workers, then merge (fan-out / fan-in)
// ---------------------------------------------------------------------------

// runPipeline reads all files with one reader goroutine and parses/aggregates
// them with `workers` goroutines, then merges the per-worker aggregators.
//
//	reader goroutine --(bounded chan of batches)--> N workers --> Merge --> result
//
// Thread safety comes from confinement, not locking: a batch is owned by the
// reader until it is sent and by exactly one worker after it is received, and
// each worker mutates only its own Aggregator. The only shared writes are the
// reader's stats (read by the caller only after the errc hand-off, which
// establishes happens-before) and the channel itself.
func runPipeline(files []string, spec SourceSpec, newAgg func() Aggregator,
	workers, batchSize, queueDepth int) (Aggregator, IngestStats, error) {

	if workers < 1 {
		workers = 1
	}
	if batchSize < 1 {
		batchSize = 1
	}
	if queueDepth < 1 {
		queueDepth = 1
	}

	batches := make(chan []RawRow, queueDepth)
	errc := make(chan error, 1)
	var st IngestStats

	go func() {
		defer close(batches)
		for _, path := range files {
			if err := readFile(path, spec, batchSize, batches, &st); err != nil {
				errc <- err
				return
			}
		}
		errc <- nil
	}()

	aggs := make([]Aggregator, workers)
	var wg sync.WaitGroup
	for i := range aggs {
		aggs[i] = newAgg()
		wg.Add(1)
		go func(a Aggregator) {
			defer wg.Done()
			for batch := range batches {
				for _, row := range batch {
					a.AddRow(row.Values)
				}
			}
		}(aggs[i])
	}
	wg.Wait()
	err := <-errc

	merged := aggs[0]
	for _, a := range aggs[1:] {
		merged.Merge(a)
	}
	st.Rows = merged.Stats()
	return merged, st, err
}
