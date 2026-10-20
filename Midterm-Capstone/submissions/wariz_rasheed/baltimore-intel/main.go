// Command baltimore-intel ingests three Open Baltimore datasets (NIBRS crime,
// 311 service requests, liquor licenses), joins them, and answers five
// research questions for municipal stakeholders.
//
// Usage:
//
//	go run .                      # reads CSVs from ./data
//	go run . -data /path/to/csvs  # or point at another folder
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"baltimoreintel/internal/analysis"
	"baltimoreintel/internal/model"
	"baltimoreintel/internal/pipeline"
	"baltimoreintel/internal/report"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := flag.String("data", "data", "folder containing the three CSV files")
	crimePath := flag.String("crime", "", "NIBRS crime CSV (default: *NIBRS*.csv in -data)")
	reqPath := flag.String("requests", "", "311 requests CSV (default: *311*.csv in -data)")
	licPath := flag.String("licenses", "", "liquor licenses CSV (default: *icense*.csv in -data)")
	outDir := flag.String("out", "output", "folder for the HTML report")
	workers := flag.Int("workers", runtime.NumCPU(), "worker goroutines per stage")
	fromStr := flag.String("from", "", "window start, YYYY-MM-DD (default: first 311 request)")
	toStr := flag.String("to", "", "window end, YYYY-MM-DD (default: last 311 request)")
	flag.Parse()

	start := time.Now()

	// The pipeline holds hundreds of thousands of short-lived parsed rows.
	// A higher GC target trades some memory for fewer collection cycles.
	debug.SetGCPercent(200)

	crime, err := resolve(*crimePath, *dataDir, "*NIBRS*.csv", "NIBRS crime")
	if err != nil {
		return err
	}
	requests, err := resolve(*reqPath, *dataDir, "*311*.csv", "311 requests")
	if err != nil {
		return err
	}
	licenses, err := resolve(*licPath, *dataDir, "*icense*.csv", "liquor licenses")
	if err != nil {
		return err
	}

	var from, to time.Time
	if *fromStr != "" {
		if from, err = time.Parse("2006-01-02", *fromStr); err != nil {
			return fmt.Errorf("-from must look like 2026-01-31: %w", err)
		}
	}
	if *toStr != "" {
		if to, err = time.Parse("2006-01-02", *toStr); err != nil {
			return fmt.Errorf("-to must look like 2026-01-31: %w", err)
		}
		to = to.Add(24*time.Hour - time.Second) // include the whole end day
	}

	progress := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	p, err := pipeline.NewBuilder().
		Crime(crime).Requests(requests).Licenses(licenses).
		Workers(*workers).Window(from, to).Logger(progress).
		Build()
	if err != nil {
		return err
	}
	out, err := p.Run()
	if err != nil {
		return err
	}

	progress("Running 5 analyses concurrently...")
	results := analysis.RunAll(out.Data, analysis.All())

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	summary := report.Summary{
		Title: "Baltimore City Cross-Dataset Intelligence Report",
		Subtitle: fmt.Sprintf("Analysis window: %s to %s (UTC). Datasets: NIBRS Group A crime, 311 service requests, liquor licenses.",
			out.Data.From.Format("2006-01-02"), out.Data.To.Format("2006-01-02")),
		Overview: overview(out),
		Results:  results,
		Footer:   footer(out, *workers, time.Since(start), &ms),
	}

	term, err := report.New("terminal")
	if err != nil {
		return err
	}
	if err := term.Render(os.Stdout, summary); err != nil {
		return err
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	htmlPath := filepath.Join(*outDir, "report.html")
	f, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	defer f.Close()
	hr, err := report.New("html")
	if err != nil {
		return err
	}
	if err := hr.Render(f, summary); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "\nHTML report written to %s\n", htmlPath)
	return nil
}

// resolve returns an explicit path, or finds the single matching file in dir.
func resolve(explicit, dir, pattern, label string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no %s file found: put a CSV matching %q in %q (or pass its path with a flag)", label, pattern, dir)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("several files match %q in %q for %s: %v; pass the one you want with a flag", pattern, dir, label, matches)
	}
	return matches[0], nil
}

func overview(out *pipeline.Output) report.Table {
	q := out.Data.Quality
	t := report.Table{
		Title: "Datasets and join coverage",
		Headers: []string{"Dataset", "File", "Rows read", "Rows parsed", "In window", "Placed in a neighborhood"},
	}
	cover := []model.Coverage{q.Crime, q.Requests, q.Licenses}
	names := []string{"NIBRS crime", "311 requests", "Liquor licenses (active)"}
	for i, ld := range out.Loads {
		t.Rows = append(t.Rows, []string{
			names[i], filepath.Base(ld.Path), count(ld.Read), count(ld.Kept),
			count(cover[i].Rows), fmt.Sprintf("%s (%.1f%%)", count(cover[i].Located), percent(cover[i].Located, cover[i].Rows)),
		})
	}
	return t
}

func footer(out *pipeline.Output, workers int, total time.Duration, ms *runtime.MemStats) []string {
	lines := []string{
		fmt.Sprintf("Join: %s street blocks indexed; %s 311 call-centre information rows (ECC) excluded from analysis.",
			count(out.Data.IndexBlocks), count(out.Data.Quality.Requests.Ignored)),
	}
	stages := "Stage timings:"
	for _, s := range out.Stages {
		stages += fmt.Sprintf(" %s %s;", s.Name, s.Took.Round(time.Millisecond))
	}
	lines = append(lines, stages)
	lines = append(lines, fmt.Sprintf("Run: %s total, %d workers per stage, heap in use %d MB, %d GC cycles.",
		total.Round(time.Millisecond), workers, ms.HeapAlloc/(1<<20), ms.NumGC))
	return lines
}

func percent(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

// count formats an integer with thousands separators.
func count(n int) string {
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}
