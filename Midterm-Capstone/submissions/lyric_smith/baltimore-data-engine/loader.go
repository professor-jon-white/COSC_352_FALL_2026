package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

type Job struct {
	Path   string
	Parser RecordParser
}

type LoadResult struct {
	Name    string
	Records []interface{}
	Skipped int
	Err     error
}

func loadCSV(path string, parser RecordParser) ([]interface{}, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening %s: %v", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	if _, err := r.Read(); err != nil {
		return nil, 0, fmt.Errorf("reading header of %s: %v", path, err)
	}

	var records []interface{}
	skipped := 0

	for {
		row, err := r.Read()
		if err != nil {
			break
		}
		rec, err := parser.ParseRow(row)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	return records, skipped, nil
}

func loadAll(jobs map[string]Job) map[string]LoadResult {
	results := make(map[string]LoadResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for key, job := range jobs {
		wg.Add(1)
		go func(key string, path string, parser RecordParser) {
			defer wg.Done()
			records, skipped, err := loadCSV(path, parser)
			mu.Lock()
			results[key] = LoadResult{Name: parser.Name(), Records: records, Skipped: skipped, Err: err}
			mu.Unlock()
		}(key, job.Path, job.Parser)
	}

	wg.Wait()
	return results
}
