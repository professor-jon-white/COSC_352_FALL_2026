package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func run() error {
	mode := flag.String("mode", "analyze", "analyze, fetch or demo")
	dir := flag.String("data", "data/live", "snapshot folder")
	out := flag.String("out", "public-safety-report.md", "Markdown output")
	start := flag.String("start", "2025-01-01", "inclusive source date filter YYYY-MM-DD")
	end := flag.String("end", "2025-04-01", "exclusive source date filter YYYY-MM-DD")
	limit := flag.Int("limit", 20000, "max rows per dataset; 0 downloads all within interval")
	workers := flag.Int("workers", runtime.NumCPU(), "workers (1-256)")
	flag.Parse()
	if *workers < 1 || *workers > 256 || *limit < 0 {
		return fmt.Errorf("workers must be 1-256; limit must be nonnegative")
	}
	aDate, e := time.Parse("2006-01-02", *start)
	if e != nil {
		return e
	}
	bDate, e := time.Parse("2006-01-02", *end)
	if e != nil || !bDate.After(aDate) {
		return fmt.Errorf("end must be a date after start")
	}
	if *mode == "demo" && *dir == "data/live" {
		*dir = "data/demo"
	}
	if *mode == "analyze" {
	} else if *mode == "fetch" || *mode == "demo" {
		if e = os.MkdirAll(*dir, 0755); e != nil {
			return e
		}
		if _, e = os.Stat(filepath.Join(*dir, "manifest.json")); e == nil && *mode == "fetch" {
			return fmt.Errorf("snapshot already exists in %s; choose a new -data folder to preserve provenance", *dir)
		}
		switch *mode {
		case "fetch":
			e = fetch(*dir, *start, *end, *limit)
		case "demo":
			e = demo(*dir)
		}
		if e != nil {
			return e
		}
	} else {
		return fmt.Errorf("unknown mode %q", *mode)
	}
	var a, b []Row
	var ea, eb error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a, ea = readRows(filepath.Join(*dir, "crime.json"), "CCNumber", "CrimeDateTime", "Neighborhood", "Description", "Weapon")
	}()
	go func() {
		defer wg.Done()
		b, eb = readRows(filepath.Join(*dir, "arrests.json"), "IncidentNumber", "ArrestNumber", "ArrestDateTime", "Neighborhood")
	}()
	wg.Wait()
	if ea != nil {
		return ea
	}
	if eb != nil {
		return eb
	}
	mb, e := os.ReadFile(filepath.Join(*dir, "manifest.json"))
	if e != nil {
		return e
	}
	var m Manifest
	if e = json.Unmarshal(mb, &m); e != nil {
		return e
	}
	if m.Crime.Rows != len(a) || m.Arrest.Rows != len(b) || m.Crime.URL == "" || m.Arrest.URL == "" {
		return fmt.Errorf("snapshot manifest does not match raw files")
	}
	began := time.Now()
	r := analyze(a, b, *workers)
	s := report(r, m)
	if strings.HasSuffix(strings.ToLower(*out), ".json") {
		return fmt.Errorf("use a Markdown output path, not a raw JSON file")
	}
	if e = os.WriteFile(*out, []byte(s), 0644); e != nil {
		return e
	}
	fmt.Print(s)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(os.Stderr, "\nAnalysis elapsed=%s workers=%d current Go heap=%.1f MiB\n", time.Since(began), *workers, float64(mem.Alloc)/(1<<20))
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "error:", e)
		os.Exit(1)
	}
}
