// Command capstone runs the pipeline: ingest both CSVs, match records by
// neighborhood and month, aggregate, and print a report.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"capstone/internal/ingest"
	"capstone/internal/ingest/crime"
	"capstone/internal/ingest/requests311"
	"capstone/internal/match"
	"capstone/internal/model"
	"capstone/internal/report"
)

type config struct {
	crimePath, requestsPath string
	year                    int
	workers                 int
	chunkSize               int    // ingest chunk bytes; 0 = default (tests set it small)
	reports                 string // comma-separated report IDs, or "all"
	memstats                bool   // print peak memory and GC activity
}

func main() {
	var cfg config
	flag.StringVar(&cfg.crimePath, "crime", "data/crime.csv", "path to the NIBRS crime CSV")
	flag.StringVar(&cfg.requestsPath, "requests", "data/requests_311.csv", "path to the 311 requests CSV")
	flag.IntVar(&cfg.year, "year", 2025, "keep only records from this year")
	flag.IntVar(&cfg.workers, "workers", runtime.NumCPU(), "workers per file for ingest, and match shards, in concurrent mode")
	mode := flag.String("mode", "concurrent", "sequential, concurrent, or compare (run both, check they match, time each)")
	flag.StringVar(&cfg.reports, "reports", "all", "reports to print: all, or a comma-separated list of sanitation, distress, close-time, streetlights, high-311-low-crime")
	flag.BoolVar(&cfg.memstats, "memstats", false, "print peak memory and garbage collection stats (run one mode at a time for clean numbers)")
	flag.IntVar(&cfg.chunkSize, "chunk-size", 0, "ingest chunk size in bytes for concurrent mode (0 = default 1 MiB)")
	flag.Parse()

	if err := run(os.Stdout, cfg, *mode); err != nil {
		fmt.Fprintln(os.Stderr, "capstone:", err)
		os.Exit(1)
	}
}

func run(w io.Writer, cfg config, mode string) error {
	var sampler *memSampler
	if cfg.memstats {
		sampler = startMemSampler()
	}

	reports, err := report.Select(cfg.reports)
	if err != nil {
		return err
	}

	var res *result
	switch mode {
	case "sequential", "concurrent":
		if res, err = pipeline(cfg, mode == "concurrent"); err != nil {
			return err
		}
		fmt.Printf("%s: ingest %v, match %v\n\n", mode, round(res.ingestTime), round(res.matchTime))
	case "compare":
		if res, err = compare(w, cfg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown -mode %q (want sequential, concurrent, or compare)", mode)
	}

	if err := report.Ingest(w, cfg.year,
		report.Dataset{Name: "crime", Stats: res.crimeStats},
		report.Dataset{Name: "311", Stats: res.requestStats},
	); err != nil {
		return err
	}
	fmt.Fprintln(w)
	if err := report.Matched(w, res.summaries); err != nil {
		return err
	}
	if err := report.Run(w, report.NewInput(cfg.year, res.summaries), reports); err != nil {
		return err
	}
	if sampler != nil {
		printMemStats(w, mode, sampler.Stop())
	}
	return nil
}

// result is everything one pipeline run produces, plus how long it took.
type result struct {
	crimes       []model.Crime
	crimeStats   ingest.Stats
	requests     []model.ServiceRequest
	requestStats ingest.Stats
	summaries    []model.Summary

	ingestTime, matchTime time.Duration
}

// pipeline loads both files and matches them, either all sequentially or
// with concurrent ingest and sharded concurrent matching.
func pipeline(cfg config, concurrent bool) (*result, error) {
	res := &result{}
	start := time.Now()
	var err error
	if concurrent {
		err = loadConcurrent(cfg, res)
	} else {
		err = loadSequential(cfg, res)
	}
	if err != nil {
		return nil, err
	}
	res.ingestTime = time.Since(start)

	start = time.Now()
	if concurrent {
		res.summaries = match.ByNeighborhoodMonthConcurrent(res.crimes, res.requests, cfg.workers)
	} else {
		res.summaries = match.ByNeighborhoodMonth(res.crimes, res.requests)
	}
	res.matchTime = time.Since(start)
	return res, nil
}

func loadSequential(cfg config, res *result) error {
	err := withFile(cfg.crimePath, func(f io.Reader) (err error) {
		res.crimes, res.crimeStats, err = crime.Load(f, cfg.year)
		return err
	})
	if err != nil {
		return err
	}
	return withFile(cfg.requestsPath, func(f io.Reader) (err error) {
		res.requests, res.requestStats, err = requests311.Load(f, cfg.year)
		return err
	})
}

// loadConcurrent loads both files at once, each with its own worker pool.
// The two goroutines write disjoint fields of res, and wg.Wait orders those
// writes before anything below reads them.
func loadConcurrent(cfg config, res *result) error {
	opts := ingest.ConcurrentOptions{Workers: cfg.workers, ChunkSize: cfg.chunkSize}
	var crimeErr, requestErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		crimeErr = withFile(cfg.crimePath, func(f io.Reader) (err error) {
			res.crimes, res.crimeStats, err = crime.LoadConcurrent(f, cfg.year, opts)
			return err
		})
	}()
	go func() {
		defer wg.Done()
		requestErr = withFile(cfg.requestsPath, func(f io.Reader) (err error) {
			res.requests, res.requestStats, err = requests311.LoadConcurrent(f, cfg.year, opts)
			return err
		})
	}()
	wg.Wait()
	return errors.Join(crimeErr, requestErr)
}

func withFile(path string, fn func(io.Reader) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

// compare runs both modes, prints their timings, and fails if any output
// differs. It returns the concurrent result for reporting.
func compare(w io.Writer, cfg config) (*result, error) {
	// Read both files once so neither timed run pays for a cold disk cache.
	for _, p := range []string{cfg.crimePath, cfg.requestsPath} {
		if err := withFile(p, func(f io.Reader) error { _, err := io.Copy(io.Discard, f); return err }); err != nil {
			return nil, err
		}
	}

	seq, err := pipeline(cfg, false)
	if err != nil {
		return nil, err
	}
	conc, err := pipeline(cfg, true)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(w, "Timing (concurrent: %d ingest workers per file, %d match shards, %d CPUs)\n", cfg.workers, cfg.workers, runtime.NumCPU())
	fmt.Fprintf(w, "  %-12s %10s %10s %10s\n", "mode", "ingest", "match", "total")
	for _, r := range []struct {
		name string
		res  *result
	}{{"sequential", seq}, {"concurrent", conc}} {
		fmt.Fprintf(w, "  %-12s %10v %10v %10v\n", r.name, round(r.res.ingestTime), round(r.res.matchTime), round(r.res.ingestTime+r.res.matchTime))
	}
	speedup := func(a, b time.Duration) string { return fmt.Sprintf("%.1fx", a.Seconds()/b.Seconds()) }
	fmt.Fprintf(w, "  %-12s %10s %10s %10s\n\n", "speedup",
		speedup(seq.ingestTime, conc.ingestTime),
		speedup(seq.matchTime, conc.matchTime),
		speedup(seq.ingestTime+seq.matchTime, conc.ingestTime+conc.matchTime))

	if diffs := differences(seq, conc); len(diffs) > 0 {
		return nil, fmt.Errorf("sequential and concurrent results differ: %s", strings.Join(diffs, ", "))
	}
	fmt.Fprintf(w, "Results identical: %d crimes, %d 311 requests, ingest stats, %d neighborhood-month records\n\n",
		len(conc.crimes), len(conc.requests), len(conc.summaries))
	return conc, nil
}

// differences names each output that is not deeply equal between a and b.
// Records are compared in order, so this also checks that the concurrent
// loader preserves file order.
func differences(a, b *result) []string {
	var diffs []string
	check := func(name string, x, y any) {
		if !reflect.DeepEqual(x, y) {
			diffs = append(diffs, name)
		}
	}
	check("crimes", a.crimes, b.crimes)
	check("crime stats", a.crimeStats, b.crimeStats)
	check("311 requests", a.requests, b.requests)
	check("311 stats", a.requestStats, b.requestStats)
	check("summaries", a.summaries, b.summaries)
	return diffs
}

func round(d time.Duration) time.Duration { return d.Round(time.Millisecond) }
