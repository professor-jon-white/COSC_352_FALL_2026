// Package ingest implements a bounded parsing and sharded aggregation pipeline.
package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"sync"

	"baltimore-capstone/internal/model"
)

type job struct {
	seq int
	raw []byte
}
type parsed struct {
	seq  int
	attr model.Attributes
	err  error
}

// Load is deterministic even when parsing completes out of order.
// A bounded credit channel limits outstanding records, including the reorder map.
func Load(path, kind string, workers int) (model.Dataset, error) {
	out := model.Dataset{Areas: map[string]*model.Stats{}}
	if workers < 1 || workers > 64 {
		return out, fmt.Errorf("workers must be 1..64")
	}
	if kind != "requests" && kind != "vacancies" {
		return out, fmt.Errorf("unknown dataset %q", kind)
	}
	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()
	jobs := make(chan job, workers*4)
	results := make(chan parsed, workers*4)
	credits := make(chan struct{}, workers*8)
	readErr := make(chan error, 1)
	go func() {
		defer close(jobs)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 65536), 4*1024*1024)
		seq := 0
		for scanner.Scan() {
			credits <- struct{}{}
			jobs <- job{seq, append([]byte(nil), scanner.Bytes()...)}
			seq++
		}
		readErr <- scanner.Err()
	}()
	var parsers sync.WaitGroup
	for i := 0; i < workers; i++ {
		parsers.Add(1)
		go func() {
			defer parsers.Done()
			for j := range jobs {
				var feature model.Feature
				err := json.Unmarshal(j.raw, &feature)
				feature.Attributes.Area = model.Normalize(feature.Attributes.Area)
				results <- parsed{j.seq, feature.Attributes, err}
			}
		}()
	}
	go func() { parsers.Wait(); close(results) }()

	// Each reducer exclusively owns its map. Neighborhood hashing ensures all
	// records for a neighborhood reach one reducer without a shared-map lock.
	shards := make([]chan model.Attributes, workers)
	maps := make(chan map[string]*model.Stats, workers)
	for i := range shards {
		shards[i] = make(chan model.Attributes, 32)
		ch := shards[i]
		go func() {
			m := map[string]*model.Stats{}
			for a := range ch {
				if m[a.Area] == nil {
					m[a.Area] = model.NewStats()
				}
				m[a.Area].Add(a, kind)
			}
			maps <- m
		}()
	}
	seen := map[string]bool{}
	process := func(p parsed) {
		q := &out.Quality
		q.Lines++
		if p.err != nil {
			q.InvalidJSON++
			return
		}
		a := p.attr
		if a.Deleted != nil && *a.Deleted != 0 {
			q.Deleted++
			return
		}
		key := a.Key(kind)
		if key == "" {
			q.MissingID++
			return
		}
		if a.Area == "" {
			q.MissingNeighborhood++
			return
		}
		if kind == "requests" {
			if a.Created == nil {
				q.MissingCreated++
				return
			}
			if !model.InPeriod(*a.Created) {
				q.OutsidePeriod++
				return
			}
		}
		if seen[key] {
			q.Duplicates++
			return
		}
		seen[key] = true
		if kind == "requests" && a.Closed != nil {
			if _, ok := model.ClosedDays(a); !ok {
				q.InvalidClosure++
			}
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(a.Area))
		shards[int(h.Sum32()%uint32(workers))] <- a
		q.Accepted++
	}
	pending := map[int]parsed{}
	next := 0
	for p := range results {
		pending[p.seq] = p
		for {
			p, ok := pending[next]
			if !ok {
				break
			}
			process(p)
			delete(pending, next)
			next++
			<-credits
		}
	}
	for _, ch := range shards {
		close(ch)
	}
	for range shards {
		for name, s := range <-maps {
			out.Areas[name] = s
		}
	}
	if err := <-readErr; err != nil {
		return out, fmt.Errorf("read %s: %w", path, err)
	}
	if out.Quality.Lines == 0 {
		return out, fmt.Errorf("%s is empty", path)
	}
	if out.Quality.InvalidJSON > 0 {
		return out, fmt.Errorf("%s has %d malformed JSON records", path, out.Quality.InvalidJSON)
	}
	if out.Quality.Accepted == 0 {
		return out, fmt.Errorf("%s has no usable records", path)
	}
	return out, nil
}
