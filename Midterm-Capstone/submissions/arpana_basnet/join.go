package main

import "sync"

// Match neighborhoods using a fixed number of workers.
func matchNeighborhoods(
	requests map[string]int,
	notices map[string]int,
) []string {
	const workerCount = 4

	jobs := make(chan string, len(requests))
	results := make(chan string, len(requests))

	// Queue each neighborhood for matching.
	for neighborhood := range requests {
		jobs <- neighborhood
	}
	close(jobs)

	var workers sync.WaitGroup
	workers.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer workers.Done()

			for neighborhood := range jobs {
				if _, exists := notices[neighborhood]; exists {
					results <- neighborhood
				}
			}
		}()
	}

	// Close results after all workers finish.
	go func() {
		workers.Wait()
		close(results)
	}()

	matched := make([]string, 0)
	for neighborhood := range results {
		matched = append(matched, neighborhood)
	}

	return matched
}
