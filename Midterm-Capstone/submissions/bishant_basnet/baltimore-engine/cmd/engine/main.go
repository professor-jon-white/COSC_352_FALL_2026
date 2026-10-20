// Command baltimore-engine ingests Open Baltimore crime and 311 data
// concurrently, joins them by neighborhood, and answers five stakeholder
// research questions with terminal charts and a Markdown report.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/bishant/baltimore-engine/internal/analysis"
	"github.com/bishant/baltimore-engine/internal/ingest"
	"github.com/bishant/baltimore-engine/internal/model"
	"github.com/bishant/baltimore-engine/internal/report"
)

func main() {
	crimePath := flag.String("crime", "data/crime.csv", "path to Part 1 Crime Data CSV")
	reqPath := flag.String("requests", "data/311.csv", "path to 311 Customer Service Requests CSV")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	out := flag.String("out", "report.md", "Markdown report output path (empty to skip)")
	top := flag.Int("top", 10, "rows to show in top-N charts")
	flag.Parse()
	if *workers < 1 {
		*workers = 1
	}
	if *top < 1 {
		*top = 10
	}

	start := time.Now()
	in := ingest.NewInterner()

	// Both files are ingested at the same time, each with its own worker pool.
	var (
		wg             sync.WaitGroup
		crimes         []model.CrimeRecord
		reqs           []model.ServiceRequest
		cStats, rStats ingest.Stats
		cErr, rErr     error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		crimes, cStats, cErr = ingest.Load[model.CrimeRecord](*crimePath, ingest.CrimeParser{In: in}, *workers)
	}()
	go func() {
		defer wg.Done()
		reqs, rStats, rErr = ingest.Load[model.ServiceRequest](*reqPath, ingest.ServiceRequestParser{In: in}, *workers)
	}()
	wg.Wait()
	for _, err := range []error{cErr, rErr} {
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
	ingestDur := time.Since(start)
	var afterIngest runtime.MemStats
	runtime.ReadMemStats(&afterIngest)

	joinStart := time.Now()
	ctx := analysis.BuildContext(crimes, reqs, *workers, *top)
	joinDur := time.Since(joinStart)

	// The joined Context holds only aggregates, so release the raw records
	// and let the GC reclaim them before analysis.
	crimes, reqs = nil, nil
	runtime.GC()
	var afterJoin runtime.MemStats
	runtime.ReadMemStats(&afterJoin)

	qStart := time.Now()
	results := analysis.RunAll(ctx, analysis.DefaultQuestions())
	qDur := time.Since(qStart)

	rep := buildReport(results, []ingest.Stats{cStats, rStats}, ctx, *workers,
		ingestDur, joinDur, qDur, afterIngest, afterJoin)

	text, _ := report.NewRenderer("text")
	fmt.Println(text.Render(rep))

	if *out != "" {
		md, _ := report.NewRenderer("markdown")
		if err := os.WriteFile(*out, []byte(md.Render(rep)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing report:", err)
			os.Exit(1)
		}
		fmt.Printf("\nMarkdown report written to %s\n", *out)
	}
}

func mb(b uint64) string { return fmt.Sprintf("%.1f MB", float64(b)/(1<<20)) }

func buildReport(results []analysis.Result, stats []ingest.Stats, ctx *analysis.Context, workers int,
	ingestDur, joinDur, qDur time.Duration, mIngest, mJoin runtime.MemStats) *report.Report {

	b := report.NewBuilder("Baltimore Municipal Intelligence Report")
	b.Meta("Generated: " + time.Now().Format("Mon Jan 2 2006 15:04 MST"))
	b.Meta("Sources: Open Baltimore - Part 1 Crime Data; 311 Customer Service Requests (joined on Neighborhood)")
	b.Meta(fmt.Sprintf("Engine: Go %s, %d worker goroutines per dataset, %d CPUs", runtime.Version(), workers, runtime.NumCPU()))

	b.Section("Ingestion & Runtime Summary", "")
	tb := report.Table{Headers: []string{"Dataset", "File size", "Rows read", "Rows kept", "Dropped", "Malformed", "Time"}}
	for _, s := range stats {
		tb.Rows = append(tb.Rows, []string{s.Source, mb(uint64(s.Bytes)), report.Commas(s.RowsRead),
			report.Commas(s.RowsKept), report.Commas(s.RowsDropped), report.Commas(s.Malformed),
			s.Duration.Round(time.Millisecond).String()})
	}
	b.Table(tb)
	b.Bullet(fmt.Sprintf("Concurrent ingest: %s | join/aggregate: %s | 5 questions: %s",
		ingestDur.Round(time.Millisecond), joinDur.Round(time.Millisecond), qDur.Round(time.Millisecond)))
	b.Bullet(fmt.Sprintf("Neighborhoods: %d total, %d present in both datasets (join coverage: %.1f%% of crimes, %.1f%% of 311 requests)",
		len(ctx.Hoods), len(ctx.Joined),
		100*float64(ctx.JoinedCrimes)/float64(max(ctx.TotalCrimes, 1)),
		100*float64(ctx.JoinedRequests)/float64(max(ctx.TotalRequests, 1))))
	b.Bullet(fmt.Sprintf("Heap after ingest: %s | heap after join + GC of raw records: %s | GC cycles: %d",
		mb(mIngest.HeapAlloc), mb(mJoin.HeapAlloc), mJoin.NumGC))

	for _, r := range results {
		b.Section(fmt.Sprintf("Q%d. %s", r.ID, r.Title), "Stakeholder: "+r.Stakeholder)
		b.Para("Research question: " + r.Question)
		for _, f := range r.Findings {
			b.Bullet(f)
		}
		for _, c := range r.Charts {
			b.Chart(c)
		}
		for _, t := range r.Tables {
			b.Table(t)
		}
	}
	return b.Build()
}
