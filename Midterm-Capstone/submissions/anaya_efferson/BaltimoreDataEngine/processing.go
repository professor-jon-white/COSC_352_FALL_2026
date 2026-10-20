package main

import (
	"runtime"
	"sort"
	"sync"
)

// processDataset streams CSV records through a worker pool.
func processDataset(filename string) (map[string]int, int, error) {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, workers*2)
	results := make(chan map[string]int, workers)

	var wg sync.WaitGroup

	// Start worker goroutines.
	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			localCounts := make(map[string]int)

			for neighborhood := range jobs {
				localCounts[neighborhood]++
			}

			results <- localCounts
		}()
	}

	// Read CSV records and close jobs afterward.
	type ingestionResult struct {
		count int
		err   error
	}

	ingestionDone := make(chan ingestionResult, 1)

	go func() {
		count, err := streamNeighborhoods(filename, jobs)
		close(jobs)
		ingestionDone <- ingestionResult{count, err}
	}()

	// Close results when all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Merge worker results.
	combined := make(map[string]int)

	for localCounts := range results {
		for name, count := range localCounts {
			combined[name] += count
		}
	}

	ingestion := <-ingestionDone

	if ingestion.err != nil {
		return nil, ingestion.count, ingestion.err
	}

	return combined, ingestion.count, nil
}

// combineDataParallel joins neighborhood counts
// using a bounded worker pool.
func combineDataParallel(
	requests map[string]int,
	properties map[string]int,
) []NeighborhoodData {

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	// Create a unique set of neighborhoods.
	names := make(map[string]struct{})

	for name := range requests {
		names[name] = struct{}{}
	}

	for name := range properties {
		names[name] = struct{}{}
	}

	jobs := make(chan string, workers*2)
	results := make(chan NeighborhoodData, workers*2)

	var wg sync.WaitGroup

	// Start worker goroutines.
	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for name := range jobs {
				results <- NeighborhoodData{
					Name:     name,
					Requests: requests[name],
					Vacant:   properties[name],
				}
			}
		}()
	}

	// Send neighborhoods to workers.
	go func() {
		for name := range names {
			jobs <- name
		}
		close(jobs)
	}()

	// Close results after workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect joined records.
	combined := make([]NeighborhoodData, 0, len(names))

	for item := range results {
		combined = append(combined, item)
	}

	// Ensure deterministic output order.
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Name < combined[j].Name
	})

	return combined
}
