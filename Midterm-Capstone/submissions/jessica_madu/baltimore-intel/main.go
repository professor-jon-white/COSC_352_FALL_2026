// Baltimore Civic Intelligence: joins Part 1 Crime Data with 311 Customer
// Service Requests by neighborhood and answers five questions for city leaders.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

type result struct {
	agg *Agg
	err error
	dur time.Duration
}

// start launches one ingest pipeline in its own goroutine and returns a
// channel for the result (future/promise style).
func start(ds Dataset, path string, workers int, years YearRange) <-chan result {
	ch := make(chan result, 1)
	go func() {
		t := time.Now()
		a, err := Ingest(path, ds, workers, years)
		ch <- result{a, err, time.Since(t)}
	}()
	return ch
}

func main() {
	crimePath := flag.String("crime", "data/sample/crime_sample.csv", "path to Part 1 Crime Data CSV")
	reqPath := flag.String("requests", "data/sample/311_sample.csv", "path to 311 Customer Service Requests CSV")
	crimeYears := flag.String("crime-years", "2024", "year or range (e.g. 2023-2024, or 'all') of crime rows to use")
	reqYears := flag.String("requests-years", "2025", "year or range of 311 rows to use")
	workers := flag.Int("workers", runtime.NumCPU(), "parser goroutines per dataset")
	top := flag.Int("top", 10, "rows to show in each ranking")
	out := flag.String("out", "report.txt", "also write the report to this file")
	flag.Parse()

	cy, err := parseYears(*crimeYears)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ry, err := parseYears(*reqYears)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	begin := time.Now()
	// Both datasets are ingested concurrently, each with its own worker pool.
	cc := start(NewCrimeDataset(), *crimePath, *workers, cy)
	sc := start(New311Dataset(), *reqPath, *workers, ry)
	cr, sr := <-cc, <-sc
	for _, r := range []struct {
		n string
		r result
	}{{"crime", cr}, {"311", sr}} {
		if r.r.err != nil {
			fmt.Fprintf(os.Stderr, "error ingesting %s dataset: %v\n", r.n, r.r.err)
			os.Exit(1)
		}
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	w := io.MultiWriter(os.Stdout, f)

	fmt.Fprintln(w, "BALTIMORE CIVIC INTELLIGENCE REPORT")
	fmt.Fprintf(w, "Generated %s\n\n", time.Now().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Crime dataset : %-8d rows used (%s), %d skipped, %d neighborhoods  (%v)\n", cr.agg.Total, cy, cr.agg.Skipped, len(cr.agg.ByNbhd), cr.dur.Round(time.Millisecond))
	fmt.Fprintf(w, "311 dataset   : %-8d rows used (%s), %d skipped, %d neighborhoods  (%v)\n", sr.agg.Total, ry, sr.agg.Skipped, len(sr.agg.ByNbhd), sr.dur.Round(time.Millisecond))
	joined := 0
	for k := range cr.agg.ByNbhd {
		if _, ok := sr.agg.ByNbhd[k]; ok {
			joined++
		}
	}
	fmt.Fprintf(w, "Join (neighborhood): %d neighborhoods appear in both datasets\n", joined)

	fmt.Fprintln(w, "Note: 'skipped' = rows with no neighborhood, unparseable date, or outside the selected years.")
	for i, q := range AllQuestions(*top) {
		heading(w, i+1, q.Title())
		q.Answer(w, cr.agg, sr.agg)
	}

	fmt.Fprintf(w, "\n%s\nRuntime: %d CPUs, %d workers/dataset, total %v, heap in use %.1f MB, %d GC cycles\n",
		"----------------------------------------------------------------------------",
		runtime.NumCPU(), *workers, time.Since(begin).Round(time.Millisecond), float64(ms.HeapAlloc)/1e6, ms.NumGC)
	fmt.Fprintf(os.Stderr, "\nReport also saved to %s\n", *out)
}

func parseYears(s string) (YearRange, error) {
	if s == "all" || s == "" {
		return YearRange{}, nil
	}
	var a, b int
	if n, _ := fmt.Sscanf(s, "%d-%d", &a, &b); n == 2 {
		return YearRange{a, b}, nil
	}
	if n, _ := fmt.Sscanf(s, "%d", &a); n == 1 {
		return YearRange{a, a}, nil
	}
	return YearRange{}, fmt.Errorf("bad year range %q (use 2024, 2023-2024, or all)", s)
}
