// Package ingest implements the concurrent CSV ingestion pipeline:
//
//	file --> [reader goroutine] --batches--> [N parser workers] --typed chunks--> collector
//
// The reader is the only goroutine touching the file (encoding/csv is not
// thread-safe); parsing/normalization, the CPU-heavy part, fans out across a
// worker pool. Bounded channels provide back-pressure so memory stays flat
// no matter how large the input file is.
package ingest

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Stats describes one ingestion run.
type Stats struct {
	Source      string
	Path        string
	Bytes       int64
	RowsRead    int64
	RowsKept    int64
	RowsDropped int64 // valid CSV but unusable (e.g. no neighborhood)
	Malformed   int64 // CSV syntax errors
	Duration    time.Duration
}

const batchSize = 4096

// Load streams a CSV through a parser strategy using `workers` goroutines.
func Load[T any](path string, p RowParser[T], workers int) ([]T, Stats, error) {
	start := time.Now()
	st := Stats{Source: p.Name(), Path: path}
	if workers < 1 {
		workers = 1
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, st, fmt.Errorf("%s: %w", p.Name(), err)
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil {
		st.Bytes = fi.Size()
	}

	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return nil, st, fmt.Errorf("%s: reading header: %w", p.Name(), err)
	}
	schema, err := ResolveSchema(header, p.Fields(), p.Required())
	if err != nil {
		return nil, st, fmt.Errorf("%s (%s): %w", p.Name(), path, err)
	}

	batches := make(chan [][]string, workers*2)
	results := make(chan []T, workers*2)
	var read, dropped, malformed atomic.Int64

	// Stage 1: single reader.
	go func() {
		defer close(batches)
		var n int64
		batch := make([][]string, 0, batchSize)
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				malformed.Add(1)
				var pe *csv.ParseError
				if errors.As(err, &pe) {
					continue
				}
				break // I/O error: stop
			}
			n++
			batch = append(batch, row)
			if len(batch) == batchSize {
				batches <- batch
				batch = make([][]string, 0, batchSize)
			}
		}
		if len(batch) > 0 {
			batches <- batch
		}
		read.Store(n)
	}()

	// Stage 2: parser worker pool.
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range batches {
				out := make([]T, 0, len(b))
				var d int64
				for _, row := range b {
					if v, ok := p.Parse(row, schema); ok {
						out = append(out, v)
					} else {
						d++
					}
				}
				dropped.Add(d)
				results <- out
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()

	// Stage 3: collector.
	var all []T
	for chunk := range results {
		all = append(all, chunk...)
	}

	st.RowsRead = read.Load()
	st.RowsKept = int64(len(all))
	st.RowsDropped = dropped.Load()
	st.Malformed = malformed.Load()
	st.Duration = time.Since(start)
	return all, st, nil
}
