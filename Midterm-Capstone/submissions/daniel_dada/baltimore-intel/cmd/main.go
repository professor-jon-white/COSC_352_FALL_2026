package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"baltimore-intel/internal/analysis"
	"baltimore-intel/internal/ingest"
	"baltimore-intel/internal/join"
	"baltimore-intel/internal/model"
)

func main() {
	dataDir := flag.String("data", "./data", "folder containing the CSV files")
	year := flag.Int("year", 2024, "year to analyze")
	mode := flag.String("mode", "parallel", "sequential or parallel")
	verify := flag.Bool("verify", false, "check that the parallel join equals the sequential join")
	flag.Parse()

	// A fresh clone has only the small sample, so fall back to it
	// when the full data folder has no crime.csv.
	if _, err := os.Stat(filepath.Join(*dataDir, "crime.csv")); err != nil && *dataDir == "./data" {
		*dataDir = "./data/sample"
		fmt.Println("Full data not found in ./data; using the bundled sample in ./data/sample")
	}

	start := time.Now()
	crimePath := filepath.Join(*dataDir, "crime.csv")
	reqPath := filepath.Join(*dataDir, fmt.Sprintf("311_%d.csv", *year))

	var (
		crimes     []model.CrimeRecord
		reqs       []model.ServiceRequest
		cst, rst   ingest.Stats
		cerr, rerr error
	)

	// Stage 1: ingest.
	if *mode == "sequential" {
		crimes, cst, cerr = ingest.ReadCrime(crimePath, *year)
		reqs, rst, rerr = ingest.ReadRequests(reqPath, *year)
	} else {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			crimes, cst, cerr = ingest.ReadCrimeParallel(crimePath, *year)
		}()
		go func() {
			defer wg.Done()
			reqs, rst, rerr = ingest.ReadRequestsParallel(reqPath, *year)
		}()
		wg.Wait()
	}
	if cerr != nil || rerr != nil {
		fmt.Fprintln(os.Stderr, "error:", cerr, rerr)
		os.Exit(1)
	}
	loadTime := time.Since(start)

	fmt.Println("Mode: ", *mode)
	fmt.Println("Crime:", cst)
	fmt.Println("311:  ", rst)

	// Stage 2: join and aggregate.
	joinStart := time.Now()
	var ds *join.Dataset
	if *mode == "sequential" {
		ds = join.Build(crimes, reqs)
	} else {
		ds = join.BuildParallel(crimes, reqs)
	}
	joinTime := time.Since(joinStart)
	fmt.Printf("Joined %d neighborhoods (%d non-complaint 311 rows excluded)\n", len(ds.ByHood), ds.Excluded)

	if *verify {
		ref := join.Build(crimes, reqs)
		par := join.BuildParallel(crimes, reqs)
		if reflect.DeepEqual(ref, par) {
			fmt.Println("VERIFY: parallel join is identical to sequential join")
		} else {
			fmt.Println("VERIFY FAILED: parallel and sequential joins differ")
			os.Exit(1)
		}
	}

	// Stage 3: analyze and report.
	var out strings.Builder
	for _, a := range analysis.All() {
		out.WriteString(a.Run(ds).String())
	}
	fmt.Print(out.String())

	if err := os.WriteFile("report.txt", []byte(out.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "could not write report.txt:", err)
	}

	// Resource summary.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("\n--- Performance and memory ---")
	fmt.Println("Mode:               ", *mode)
	fmt.Println("CPU threads used:   ", runtime.NumCPU())
	fmt.Println("Load time:          ", loadTime)
	fmt.Println("Join time:          ", joinTime)
	fmt.Println("Total elapsed:      ", time.Since(start))
	fmt.Printf("Heap in use:         %.1f MB\n", float64(m.HeapAlloc)/1e6)
	fmt.Printf("Total allocated:     %.1f MB\n", float64(m.TotalAlloc)/1e6)
	fmt.Printf("Memory from OS:      %.1f MB\n", float64(m.Sys)/1e6)
	fmt.Println("GC cycles:          ", m.NumGC)
}
