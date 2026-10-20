// Command baltimore-intel ingests, joins and analyzes Open Baltimore datasets.
//
//	baltimore-intel fetch   [-year 2022] [-max311 150000] [-data data/raw]   download live data
//	baltimore-intel sample  [-year 2022] [-seed 42] [-scale 1] [-data ...]   generate demo data
//	baltimore-intel analyze [-data data/raw] [-out reports] [-workers N] ...  run the 5 analyses
//	baltimore-intel bench   [-data data/raw] [-runs 3]                         1 worker vs N workers
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"baltimore-intel/internal/fetch"
	"baltimore-intel/internal/pipeline"
	"baltimore-intel/internal/sample"
)

const usage = `baltimore-intel: cross-dataset intelligence from Open Baltimore

Commands:
  fetch     download Part 1 Crime, 311, Liquor Licenses and Vacant Building Notices
  sample    generate synthetic data in the same schemas (offline demo)
  analyze   ingest, join and answer the five stakeholder questions
  bench     compare single-worker and multi-worker run times

Run 'baltimore-intel <command> -h' for flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var err error
	switch os.Args[1] {
	case "fetch":
		err = runFetch(ctx, os.Args[2:])
	case "sample":
		err = runSample(os.Args[2:])
	case "analyze", "run":
		err = runAnalyze(ctx, os.Args[2:])
	case "bench":
		err = runBench(ctx, os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runFetch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	dir := fs.String("data", "data/raw", "output directory for raw CSV files")
	year := fs.Int("year", 2022, "year of crime and 311 data to download")
	max311 := fs.Int("max311", 150000, "cap on 311 rows (evenly sampled across the year); 0 = all")
	workers := fs.Int("workers", 6, "concurrent download requests per dataset")
	only := fs.String("only", "", "comma-separated subset: crime,311,liquor,vacants")
	_ = fs.Parse(args)

	c := fetch.NewClient(*workers)
	want := map[string]bool{}
	for _, k := range strings.Split(*only, ",") {
		if k = strings.TrimSpace(k); k != "" {
			want[k] = true
		}
	}
	fmt.Fprintf(os.Stderr, "Downloading Open Baltimore datasets for %d into %s\n", *year, *dir)
	var lines []string
	start := time.Now()
	for _, ds := range fetch.Catalog(*year, *max311) {
		if len(want) > 0 && !want[ds.Kind] {
			continue
		}
		n, err := c.Fetch(ctx, *dir, ds)
		if err != nil {
			return fmt.Errorf("%w\n(if the portal is unreachable, run `make sample` to use bundled demo data)", err)
		}
		lines = append(lines, fmt.Sprintf("%s: %d rows from %s", ds.Title, n, ds.LayerURL))
	}
	_ = os.Remove(filepath.Join(*dir, "SAMPLE_DATA.txt")) // these files are now real
	src := fmt.Sprintf("Fetched %s for year %d.\n%s\n", time.Now().Format("2006-01-02 15:04 MST"), *year, strings.Join(lines, "\n"))
	if err := os.WriteFile(filepath.Join(*dir, "SOURCE.txt"), []byte(src), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Done in %s. Next: make run YEAR=%d\n", time.Since(start).Round(time.Second), *year)
	return nil
}

func runSample(args []string) error {
	fs := flag.NewFlagSet("sample", flag.ExitOnError)
	dir := fs.String("data", "data/raw", "output directory")
	year := fs.Int("year", 2022, "year to stamp records with")
	seed := fs.Int64("seed", 42, "random seed (same seed = identical files)")
	scale := fs.Float64("scale", 1.0, "volume multiplier (1.0 ≈ 45k crimes, 120k 311 requests)")
	_ = fs.Parse(args)
	start := time.Now()
	counts, err := sample.Generate(sample.Options{Dir: *dir, Year: *year, Seed: *seed, Scale: *scale})
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(*dir, "SOURCE.txt"))
	names := make([]string, 0, len(counts))
	for k := range counts {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Fprintf(os.Stderr, "Generated SYNTHETIC sample data in %s (%s):\n", *dir, time.Since(start).Round(time.Millisecond))
	for _, k := range names {
		fmt.Fprintf(os.Stderr, "  %-30s %8d records\n", k, counts[k])
	}
	return nil
}

func analyzeFlags(fs *flag.FlagSet) (*pipeline.Builder, func()) {
	dir := fs.String("data", "data/raw", "directory with the four raw CSV files")
	out := fs.String("out", "reports", "directory for report.md, report.html and CSV tables")
	workers := fs.Int("workers", runtime.NumCPU(), "worker goroutines per parallel stage")
	year := fs.Int("year", 0, "keep only crime/311 records from this year (0 = all)")
	radius := fs.Float64("radius", 150, "meters around a liquor outlet that count as 'near'")
	formats := fs.String("format", "terminal,md,html,csv", "outputs: terminal, md, html, csv")
	ascii := fs.Bool("ascii", false, "plain ASCII charts (no Unicode block characters)")
	memLimit := fs.Int("memlimit", 0, "soft heap limit in MB (like GOMEMLIMIT); 0 = none")
	b := pipeline.NewBuilder()
	return b, func() {
		pipeline.SetMemoryLimit(*memLimit)
		b.DataDir(*dir).OutputDir(*out).Workers(*workers).Year(*year).Radius(*radius).Formats(*formats).ASCII(*ascii)
	}
}

func runAnalyze(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	b, apply := analyzeFlags(fs)
	_ = fs.Parse(args)
	apply()
	p, err := b.Build()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Running pipeline:")
	res, err := p.Run(ctx, true)
	if err != nil {
		return err
	}
	for _, e := range res.Errors {
		fmt.Fprintln(os.Stderr, "warning:", e)
	}
	if len(res.Files) > 0 {
		fmt.Fprintln(os.Stderr, "\nWrote:")
		for _, f := range res.Files {
			fmt.Fprintln(os.Stderr, "  "+f)
		}
	}
	return nil
}

func runBench(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	dir := fs.String("data", "data/raw", "directory with the raw CSV files")
	runs := fs.Int("runs", 3, "runs per configuration (best time is reported)")
	maxW := fs.Int("workers", runtime.NumCPU(), "largest worker count to test")
	_ = fs.Parse(args)
	counts := []int{1}
	for w := 2; w < *maxW; w *= 2 {
		counts = append(counts, w)
	}
	if *maxW > 1 {
		counts = append(counts, *maxW)
	}
	fmt.Printf("Benchmark: ingest + join + analyze (no rendering), best of %d runs, GOMAXPROCS=%d\n\n", *runs, runtime.GOMAXPROCS(0))
	fmt.Printf("  %-8s %10s %10s %10s %10s %9s\n", "workers", "ingest", "join", "analyze", "total", "speedup")
	var base time.Duration
	for _, w := range counts {
		best := map[string]time.Duration{}
		for r := 0; r < *runs; r++ {
			runtime.GC()
			p, err := pipeline.NewBuilder().DataDir(*dir).Workers(w).Quiet(true).Build()
			if err != nil {
				return err
			}
			res, err := p.Run(ctx, false)
			if err != nil {
				return err
			}
			var tot time.Duration
			for _, t := range res.Timings {
				tot += t.Duration
				if b, ok := best[t.Stage]; !ok || t.Duration < b {
					best[t.Stage] = t.Duration
				}
			}
			if b, ok := best["total"]; !ok || tot < b {
				best["total"] = tot
			}
		}
		if w == 1 {
			base = best["total"]
		}
		ms := func(d time.Duration) string { return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000) }
		fmt.Printf("  %-8d %10s %10s %10s %10s %8.2fx\n", w, ms(best["ingest"]), ms(best["join"]), ms(best["analyze"]),
			ms(best["total"]), float64(base)/float64(best["total"]))
	}
	return nil
}
