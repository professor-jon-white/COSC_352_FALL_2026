package main

import "sync"

type Bucket struct {
	Key      string
	Display  string
	Requests []ServiceRequest
	Crimes   []CrimeIncident
}

type Index struct {
	mu      sync.Mutex
	buckets map[string]*Bucket
}

func newIndex() *Index {
	return &Index{buckets: make(map[string]*Bucket)}
}

func (idx *Index) bucketFor(key, display string) *Bucket {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	b, ok := idx.buckets[key]
	if !ok {
		b = &Bucket{Key: key, Display: display}
		idx.buckets[key] = b
	}
	return b
}

func (idx *Index) addRequest(sr ServiceRequest) {
	key := normalizeKey(sr.Neighborhood)
	b := idx.bucketFor(key, sr.Neighborhood)
	idx.mu.Lock()
	b.Requests = append(b.Requests, sr)
	idx.mu.Unlock()
}

func (idx *Index) addCrime(ci CrimeIncident) {
	key := normalizeKey(ci.Neighborhood)
	b := idx.bucketFor(key, ci.Neighborhood)
	idx.mu.Lock()
	b.Crimes = append(b.Crimes, ci)
	idx.mu.Unlock()
}

func buildIndex(requests []ServiceRequest, crimes []CrimeIncident) *Index {
	idx := newIndex()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, sr := range requests {
			idx.addRequest(sr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, ci := range crimes {
			idx.addCrime(ci)
		}
	}()

	wg.Wait()
	return idx
}

func (idx *Index) all() []*Bucket {
	out := make([]*Bucket, 0, len(idx.buckets))
	for _, b := range idx.buckets {
		out = append(out, b)
	}
	return out
}
