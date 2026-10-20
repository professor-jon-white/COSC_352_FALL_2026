package join

import (
	"runtime"
	"sync"

	"baltimore-intel/internal/model"
)

// merge folds a worker's private result into d. Only one goroutine ever
// calls merge, so d needs no lock.
func (d *Dataset) merge(o *Dataset) {
	d.Excluded += o.Excluded
	for i := range d.CrimeByHour {
		d.CrimeByHour[i] += o.CrimeByHour[i]
		d.LightingHour[i] += o.LightingHour[i]
		d.TransportHour[i] += o.TransportHour[i]
	}
	for k, s := range o.ByHood {
		d.stats(k).add(s)
	}
}

// BuildParallel does the same job as Build, but splits both record slices
// into chunks and aggregates each chunk in its own goroutine.
//
// Thread safety: the input slices are only read, and every worker writes
// only to its own private Dataset, so no mutex is needed. The merge happens
// once, after all workers finish.
func BuildParallel(crimes []model.CrimeRecord, reqs []model.ServiceRequest) *Dataset {
	workers := runtime.NumCPU()
	// Each slice produces at most `workers` chunks, so this capacity means
	// no worker ever blocks when sending its result.
	partials := make(chan *Dataset, 2*workers)
	var wg sync.WaitGroup

	launch := func(n int, fill func(d *Dataset, lo, hi int)) {
		if n == 0 {
			return
		}
		chunk := (n + workers - 1) / workers
		for lo := 0; lo < n; lo += chunk {
			hi := min(lo+chunk, n)
			wg.Add(1)
			go func(lo, hi int) {
				defer wg.Done()
				d := newDataset()
				fill(d, lo, hi)
				partials <- d
			}(lo, hi)
		}
	}

	launch(len(crimes), func(d *Dataset, lo, hi int) {
		for _, c := range crimes[lo:hi] {
			d.addCrime(c)
		}
	})
	launch(len(reqs), func(d *Dataset, lo, hi int) {
		for _, r := range reqs[lo:hi] {
			d.addRequest(r)
		}
	})

	wg.Wait()
	close(partials)

	result := newDataset()
	for p := range partials {
		result.merge(p)
	}
	return result
}
