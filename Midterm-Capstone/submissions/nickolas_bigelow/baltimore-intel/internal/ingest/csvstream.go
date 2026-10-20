// Package ingest turns raw Open Baltimore CSV exports into model records.
//
// Each file is parsed by a small pipeline:
//
//	reader goroutine ──batches of rows──▶ N parser goroutines ──typed slices──▶ collector
//
// Batches carry a sequence number and the collector reassembles them in file
// order, so output is identical for any worker count (reproducible reports).
//
// The reader streams the file (it is never loaded whole into memory), rows are
// grouped into batches so channel overhead is paid once per ~2k rows instead of
// once per row, and batch buffers are recycled through a sync.Pool so steady
// state parsing allocates almost nothing beyond the CSV field strings.
package ingest

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Header resolves column names case-insensitively. Several aliases can be
// given because the portal's CSV export, the ArcGIS REST API and older
// dataset versions spell some columns differently (e.g. X vs Longitude).
type Header map[string]int

func NewHeader(cols []string) Header {
	h := make(Header, len(cols))
	for i, c := range cols {
		c = strings.TrimPrefix(c, "\ufeff") // Excel / ArcGIS BOM
		h[strings.ToLower(strings.TrimSpace(c))] = i
	}
	return h
}

// Col returns the index of the first alias present, or -1.
func (h Header) Col(aliases ...string) int {
	for _, a := range aliases {
		if i, ok := h[strings.ToLower(a)]; ok {
			return i
		}
	}
	return -1
}

// Get is a bounds-safe field accessor (rows can be ragged in raw exports).
func Get(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// RowFunc parses one row. ok=false means "skip this row" with the given reason.
type RowFunc[T any] func(row []string) (rec T, ok bool, reason string)

// Stats describes one file's ingestion.
type Stats struct {
	Source   string
	File     string
	Rows     int64
	Kept     int64
	Skipped  map[string]int64
	Duration time.Duration
	Workers  int
}

func (s Stats) String() string {
	var reasons []string
	for k, v := range s.Skipped {
		reasons = append(reasons, fmt.Sprintf("%s=%d", k, v))
	}
	return fmt.Sprintf("%-26s rows=%-8d kept=%-8d %6.0fms workers=%d skipped[%s]",
		s.Source, s.Rows, s.Kept, float64(s.Duration.Microseconds())/1000, s.Workers, strings.Join(reasons, " "))
}

const batchSize = 2048

var batchPool = sync.Pool{New: func() any { b := make([][]string, 0, batchSize); return &b }}

type batch struct {
	seq  int
	rows *[][]string
}

type parsed[T any] struct {
	seq  int
	recs []T
}

// ParseCSV streams path through a worker pool. build receives the header and
// returns the row parser (so each source can validate required columns once).
func ParseCSV[T any](ctx context.Context, source, path string, workers int,
	build func(Header) (RowFunc[T], error)) ([]T, Stats, error) {

	start := time.Now()
	st := Stats{Source: source, File: path, Skipped: map[string]int64{}, Workers: workers}
	f, err := os.Open(path)
	if err != nil {
		return nil, st, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // tolerate ragged rows
	r.LazyQuotes = true    // tolerate stray quotes in free-text address fields
	r.ReuseRecord = true   // reuse the []string; field strings are still fresh
	head, err := r.Read()
	if err != nil {
		return nil, st, fmt.Errorf("%s: reading header: %w", path, err)
	}
	parse, err := build(NewHeader(head))
	if err != nil {
		return nil, st, fmt.Errorf("%s: %w", path, err)
	}
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	batches := make(chan batch, workers*2) // bounded: back-pressure caps memory
	results := make(chan parsed[T], workers)
	var rows atomic.Int64
	var readErr error

	// Stage 1: reader
	go func() {
		defer close(batches)
		b := batchPool.Get().(*[][]string)
		seq := 0
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				var pe *csv.ParseError
				if errors.As(err, &pe) {
					continue // one malformed line must not abort a 300k-row file
				}
				readErr = err
				break
			}
			row := make([]string, len(rec)) // copy slice header; strings are immutable
			copy(row, rec)
			*b = append(*b, row)
			rows.Add(1)
			if len(*b) == batchSize {
				select {
				case batches <- batch{seq, b}:
				case <-ctx.Done():
					return
				}
				seq++
				b = batchPool.Get().(*[][]string)
			}
		}
		if len(*b) > 0 {
			batches <- batch{seq, b}
		}
	}()

	// Stage 2: parsers. Each keeps local skip counters and output, so the hot
	// loop has no shared state and no locks.
	var wg sync.WaitGroup
	skipMu := sync.Mutex{}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := map[string]int64{}
			for bt := range batches {
				b := bt.rows
				out := make([]T, 0, len(*b))
				for _, row := range *b {
					rec, ok, reason := parse(row)
					if ok {
						out = append(out, rec)
					} else {
						local[reason]++
					}
				}
				*b = (*b)[:0]
				batchPool.Put(b)
				select {
				case results <- parsed[T]{bt.seq, out}:
				case <-ctx.Done():
					return
				}
			}
			skipMu.Lock()
			for k, v := range local {
				st.Skipped[k] += v
			}
			skipMu.Unlock()
		}()
	}
	go func() { wg.Wait(); close(results) }()

	// Stage 3: collector — place each batch at its sequence slot, then
	// concatenate once into an exactly-sized slice (one final allocation).
	var parts [][]T
	total := 0
	for p := range results {
		for len(parts) <= p.seq {
			parts = append(parts, nil)
		}
		parts[p.seq] = p.recs
		total += len(p.recs)
	}
	all := make([]T, 0, total)
	for _, p := range parts {
		all = append(all, p...)
	}
	if readErr != nil {
		return nil, st, fmt.Errorf("%s: %w", path, readErr)
	}
	st.Rows = rows.Load()
	st.Kept = int64(len(all))
	st.Duration = time.Since(start)
	return all, st, ctx.Err()
}
