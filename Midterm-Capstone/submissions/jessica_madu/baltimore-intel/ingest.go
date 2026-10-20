package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const batchSize = 2000

// YearRange restricts which rows are kept (inclusive). Zero values = no limit.
type YearRange struct{ From, To int }

func (y YearRange) String() string {
	if y.From == 0 && y.To == 0 {
		return "all years"
	}
	if y.From == y.To {
		return fmt.Sprint(y.From)
	}
	return fmt.Sprintf("%d-%d", y.From, y.To)
}

func (y YearRange) keep(e Event) bool {
	if y.From == 0 && y.To == 0 {
		return true
	}
	if !e.HasDate {
		return false
	}
	yr := e.When.Year()
	return yr >= y.From && yr <= y.To
}

// Ingest is a three-stage PIPELINE:
//
//	reader goroutine --(batches)--> N parser workers --(local Aggs)--> merger
//
// Memory/resource design:
//   - The file is streamed; only `workers*2` batches ever sit in the channel
//     (bounded buffer = back-pressure, so a slow worker stalls the reader
//     instead of letting the heap grow).
//   - Each worker aggregates into its own private Agg (no shared mutable
//     state, no locks). The merge happens once at the end.
//   - Go's garbage collector reclaims each batch after a worker finishes it.
func Ingest(path string, ds Dataset, workers int, years YearRange) (*Agg, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1 // tolerate ragged rows
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: reading header: %w", path, err)
	}
	header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	parse, err := ds.Bind(header)
	if err != nil {
		return nil, err
	}

	jobs := make(chan [][]string, workers*2)
	results := make(chan *Agg, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := NewAgg()
			for batch := range jobs {
				for _, rec := range batch {
					if ev, ok := parse(rec); ok && years.keep(ev) {
						local.Add(ev)
					} else {
						local.Skipped++
					}
				}
			}
			results <- local
		}()
	}
	go func() { wg.Wait(); close(results) }()

	var readErr error
	go func() {
		defer close(jobs)
		batch := make([][]string, 0, batchSize)
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				readErr = err
				break
			}
			batch = append(batch, rec)
			if len(batch) == batchSize {
				jobs <- batch
				batch = make([][]string, 0, batchSize)
			}
		}
		if len(batch) > 0 {
			jobs <- batch
		}
	}()

	total := NewAgg()
	for local := range results {
		total.Merge(local)
	}
	// Safe to read readErr: it was written before close(jobs), which happens
	// before the workers exit, which happens before `results` is closed.
	if readErr != nil {
		return nil, readErr
	}
	return total, nil
}
