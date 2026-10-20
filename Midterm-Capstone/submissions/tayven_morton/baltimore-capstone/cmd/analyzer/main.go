package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"baltimore-capstone/internal/analysis"
	"baltimore-capstone/internal/ingest"
	"baltimore-capstone/internal/model"
	"baltimore-capstone/internal/report"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
func run() error {
	data := flag.String("data", "data", "directory containing raw JSONL extracts")
	out := flag.String("out", "output", "output directory")
	n := runtime.NumCPU()
	if n > 8 {
		n = 8
	}
	workers := flag.Int("workers", n, "workers per parsing, aggregation, and join pool (1..64)")
	flag.Parse()
	if *workers < 1 || *workers > 64 {
		return fmt.Errorf("workers must be 1..64")
	}
	// Native Go scheduler can run goroutines on multiple OS threads.
	runtime.GOMAXPROCS(*workers)
	var requests, vacancies model.Dataset
	var e1, e2 error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		requests, e1 = ingest.Load(filepath.Join(*data, "requests.jsonl"), "requests", *workers)
	}()
	go func() {
		defer wg.Done()
		vacancies, e2 = ingest.Load(filepath.Join(*data, "vacancies.jsonl"), "vacancies", *workers)
	}()
	wg.Wait()
	if e1 != nil {
		return e1
	}
	if e2 != nil {
		return e2
	}
	rows := analysis.Join(requests, vacancies, *workers)
	matched := 0
	for _, r := range rows {
		if r.Matched {
			matched++
		}
	}
	if matched == 0 {
		return fmt.Errorf("no overlapping neighborhoods; inspect normalization and data")
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(*out, "report.md"))
	if err != nil {
		return err
	}
	err = report.Write(io.MultiWriter(os.Stdout, f), rows, requests.Quality, vacancies.Quality)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	c, err := os.Create(filepath.Join(*out, "neighborhoods.csv"))
	if err != nil {
		return err
	}
	err = report.CSV(c, rows)
	closeErr = c.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Fprintf(os.Stderr, "Wrote %s/report.md and neighborhoods.csv (%d joined rows).\n", *out, len(rows))
	return nil
}
