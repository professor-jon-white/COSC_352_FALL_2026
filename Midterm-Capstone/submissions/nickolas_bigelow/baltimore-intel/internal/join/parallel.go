// Package join contains the cross-dataset matching logic and the generic
// concurrency helpers it is built on.
package join

import (
	"sync"
)

// ParallelReduce splits items into contiguous shards, folds each shard into a
// private accumulator on its own goroutine, then merges the accumulators.
// Workers never share mutable state, so the fold needs no locks and there is
// no false sharing on a common map.
func ParallelReduce[T any, A any](items []T, workers int,
	newAcc func() A, fold func(acc A, item *T), merge func(dst, src A)) A {

	if workers < 1 {
		workers = 1
	}
	if workers > len(items) {
		workers = max(1, len(items))
	}
	accs := make([]A, workers)
	var wg sync.WaitGroup
	shard := (len(items) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		accs[w] = newAcc()
		lo, hi := w*shard, min((w+1)*shard, len(items))
		if lo >= hi {
			continue
		}
		wg.Add(1)
		go func(acc A, part []T) {
			defer wg.Done()
			for i := range part {
				fold(acc, &part[i]) // pointer: avoid copying 24-40 byte records
			}
		}(accs[w], items[lo:hi])
	}
	wg.Wait()
	out := accs[0]
	for _, a := range accs[1:] {
		merge(out, a)
	}
	return out
}

// ParallelFor runs fn(i) for i in [0,n) across workers. fn may write to
// index i of a pre-sized output slice: distinct indices never race.
func ParallelFor(n, workers int, fn func(i int)) {
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	shard := (n + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo, hi := w*shard, min((w+1)*shard, n)
		if lo >= hi {
			continue
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for i := lo; i < hi; i++ {
				fn(i)
			}
		}(lo, hi)
	}
	wg.Wait()
}
