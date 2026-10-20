// Package ingest fetches the raw Baltimore datasets from live ArcGIS JSON APIs.
package ingest

import "context"

// result carries one fetcher's output over a channel.
type result[T any] struct {
	data []T
	err  error
}

// start runs fn in a goroutine and returns the channel its result arrives on.
func start[T any](ctx context.Context, fn func(context.Context) ([]T, error)) <-chan result[T] {
	ch := make(chan result[T], 1)
	go func() {
		defer close(ch)
		data, err := fn(ctx)
		ch <- result[T]{data, err}
	}()
	return ch
}

// FetchAll fetches both datasets in parallel goroutines, collecting each
// result over its own channel. The first error cancels the other fetch.
func FetchAll(ctx context.Context) (Datasets, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srCh := start(ctx, FetchServiceRequests)
	vbCh := start(ctx, FetchVacantBuildings)

	var d Datasets
	for i := 0; i < 2; i++ {
		select {
		case r := <-srCh:
			if r.err != nil {
				return Datasets{}, r.err
			}
			d.ServiceRequests = r.data
			srCh = nil
		case r := <-vbCh:
			if r.err != nil {
				return Datasets{}, r.err
			}
			d.VacantBuildings = r.data
			vbCh = nil
		}
	}
	return d, nil
}
