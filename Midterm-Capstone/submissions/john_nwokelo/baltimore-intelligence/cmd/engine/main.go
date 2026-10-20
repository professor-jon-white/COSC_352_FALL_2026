package main

import (
	"baltimore-intelligence/internal/analysis"
	"baltimore-intelligence/internal/ingestion"
	"baltimore-intelligence/internal/models"
	"baltimore-intelligence/internal/output"
	"baltimore-intelligence/internal/processing"
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
)

func main() {
	requestsPath := flag.String("requests", "data/sample_requests.geojson", "GeoJSON 311 requests path")
	boundariesPath := flag.String("boundaries", "data/sample_neighborhoods.geojson", "GeoJSON neighborhood polygons path")
	workers := flag.Int("workers", runtime.NumCPU(), "worker count")
	reportPath := flag.String("output", "", "optional output text file")
	flag.Parse()
	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "workers must be >=1")
		os.Exit(2)
	}
	var requests []models.Request
	var neighborhoods []models.Neighborhood
	var err1, err2 error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); requests, err1 = ingestion.Requests(*requestsPath) }()
	go func() { defer wg.Done(); neighborhoods, err2 = ingestion.Neighborhoods(*boundariesPath) }()
	wg.Wait()
	if err1 != nil || err2 != nil {
		fmt.Fprintln(os.Stderr, "loading failed:", err1, err2)
		os.Exit(1)
	}
	if len(neighborhoods) == 0 || len(requests) == 0 {
		fmt.Fprintln(os.Stderr, "No valid requests or neighborhoods in inputs")
		os.Exit(1)
	}
	joined, unmatched := processing.Join(context.Background(), requests, neighborhoods, *workers)
	stats := analysis.Analyze(joined, *workers)
	w := os.Stdout
	if *reportPath != "" {
		f, e := os.Create(*reportPath)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	output.Report(w, stats, len(requests), unmatched)
}
