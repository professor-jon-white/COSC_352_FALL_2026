package ingest

import (
	"runtime"
	"sync"
	"time"

	"baltimore-intel/internal/model"
	"baltimore-intel/internal/normalize"
)

type reason int

const (
	ok reason = iota
	badDate
	wrongYear
	blankKey
)

func (s *Stats) count(r reason) {
	s.Read++
	switch r {
	case ok:
		s.Kept++
	case badDate:
		s.BadDate++
	case wrongYear:
		s.WrongYear++
	case blankKey:
		s.BlankKey++
	}
}

func (s *Stats) add(o Stats) {
	s.Read += o.Read
	s.Kept += o.Kept
	s.BadDate += o.BadDate
	s.WrongYear += o.WrongYear
	s.BlankKey += o.BlankKey
}

// partial is one worker's private result. No other goroutine touches it,
// so no locks are needed.
type partial[T any] struct {
	items []T
	st    Stats
}

// parallel is the pipeline: one reader streams rows into batches, a pool of
// workers parses the batches, and the main goroutine merges the results.
func parallel[T any](path string, required []string, parse func(row) (T, reason)) ([]T, Stats, error) {
	const batchSize = 2000
	workers := runtime.NumCPU()

	batches := make(chan []row, workers*2)    // buffered: reader can run ahead
	results := make(chan partial[T], workers) // buffered: workers never block on exit

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var p partial[T] // worker-local state
			for batch := range batches {
				for _, r := range batch {
					item, why := parse(r)
					p.st.count(why)
					if why == ok {
						p.items = append(p.items, item)
					}
				}
			}
			results <- p
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	// Reader stage: runs on the calling goroutine.
	batch := make([]row, 0, batchSize)
	malformed, err := streamCSV(path, required, func(r row) {
		batch = append(batch, r)
		if len(batch) == batchSize {
			batches <- batch
			batch = make([]row, 0, batchSize)
		}
	})
	if len(batch) > 0 {
		batches <- batch
	}
	close(batches) // tells workers there is no more input

	// Merge stage: combine each worker's private result once, at the end.
	var all []T
	var st Stats
	for p := range results {
		all = append(all, p.items...)
		st.add(p.st)
	}
	st.Malformed = malformed
	return all, st, err
}

func ReadCrimeParallel(path string, year int) ([]model.CrimeRecord, Stats, error) {
	cols := []string{"CrimeDateTime", "Description", "Neighborhood"}
	return parallel(path, cols, func(r row) (model.CrimeRecord, reason) {
		t, err := time.Parse(dateLayout, r.get("CrimeDateTime"))
		if err != nil {
			return model.CrimeRecord{}, badDate
		}
		if t.Year() != year {
			return model.CrimeRecord{}, wrongYear
		}
		key := normalize.Key(r.get("Neighborhood"))
		if key == "" {
			return model.CrimeRecord{}, blankKey
		}
		return model.CrimeRecord{When: t, Description: r.get("Description"), Neighborhood: key}, ok
	})
}

func ReadRequestsParallel(path string, year int) ([]model.ServiceRequest, Stats, error) {
	cols := []string{"CreatedDate", "SRType", "Agency", "Neighborhood"}
	return parallel(path, cols, func(r row) (model.ServiceRequest, reason) {
		t, err := time.Parse(dateLayout, r.get("CreatedDate"))
		if err != nil {
			return model.ServiceRequest{}, badDate
		}
		if t.Year() != year {
			return model.ServiceRequest{}, wrongYear
		}
		key := normalize.Key(r.get("Neighborhood"))
		if key == "" {
			return model.ServiceRequest{}, blankKey
		}
		return model.ServiceRequest{Created: t, Type: r.get("SRType"), Agency: r.get("Agency"), Neighborhood: key}, ok
	})
}
