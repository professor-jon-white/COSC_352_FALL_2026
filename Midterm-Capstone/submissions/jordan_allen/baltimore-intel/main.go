// Command baltimore-intel ingests two Open Baltimore datasets (crime incidents
// and 311 service requests), joins them by neighborhood, and answers five
// research questions for municipal stakeholders.
//
// Architecture (see README.md for the full picture):
//
//	ingest   reader goroutine -> bounded channel -> N parser/aggregator workers   (Pipeline)
//	join     per-neighborhood inner join of the two aggregates
//	analyse  five Question strategies run concurrently over the read-only join
//	report   ReportBuilder -> Renderer strategies (terminal text, Markdown file)
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// Keyword lists are matched as substrings of the upper-cased text.
	defaultViolent  = "HOMICIDE,MURDER,SHOOT,ASSAULT,ROBBERY,RAPE,KIDNAP,STABB"
	defaultDisorder = "TRASH,LITTER,DUMP,GRAFFITI,VACANT,RODENT,RAT RUBOUT,HGW,WEED,LIGHT,ABANDON,SANITATION,DIRTY,CLEAN,BOARDING"
)

// Config holds every setting parsed from the command line.
type Config struct {
	CrimeFiles   []string
	RequestFiles []string
	Workers      int
	Batch        int
	Queue        int
	Filter       TimeFilter
	Loc          *time.Location
	Violent      []string
	Disorder     []string
	ReportPath   string
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func parseConfig(args []string) (Config, error) {
	fs := flag.NewFlagSet("baltimore-intel", flag.ContinueOnError)
	crime := fs.String("crime", "", "comma-separated crime CSV file(s) (required)")
	reqs := fs.String("requests", "", "comma-separated 311 CSV file(s) (required)")
	workers := fs.Int("workers", runtime.NumCPU(), "parser/aggregator goroutines per dataset")
	batch := fs.Int("batch", 2048, "rows per batch sent from the reader to the workers")
	queue := fs.Int("queue", 8, "maximum batches buffered between reader and workers")
	from := fs.String("from", "", "only use records on/after this date (YYYY-MM-DD)")
	to := fs.String("to", "", "only use records on/before this date (YYYY-MM-DD)")
	tz := fs.String("tz", "", "convert timestamps to this IANA zone, e.g. America/New_York (default: use as written)")
	violent := fs.String("violent", defaultViolent, "comma-separated keywords marking violent crime descriptions")
	disorder := fs.String("disorder", defaultDisorder, "comma-separated keywords marking physical-disorder 311 request types")
	report := fs.String("report", "", "also write a Markdown report to this path")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		CrimeFiles:   splitFiles(*crime),
		RequestFiles: splitFiles(*reqs),
		Workers:      *workers,
		Batch:        *batch,
		Queue:        *queue,
		Violent:      splitList(*violent),
		Disorder:     splitList(*disorder),
		ReportPath:   *report,
	}
	if len(cfg.CrimeFiles) == 0 || len(cfg.RequestFiles) == 0 {
		return Config{}, errors.New("both -crime and -requests are required (see README.md for an example)")
	}
	all := append(append([]string{}, cfg.CrimeFiles...), cfg.RequestFiles...)
	for _, p := range all {
		if _, err := os.Stat(p); err != nil {
			return Config{}, fmt.Errorf("cannot read input file: %w", err)
		}
	}
	if cfg.Workers < 1 || cfg.Batch < 1 || cfg.Queue < 1 {
		return Config{}, errors.New("-workers, -batch and -queue must all be at least 1")
	}

	loc := time.UTC
	if *tz != "" {
		l, err := time.LoadLocation(*tz)
		if err != nil {
			return Config{}, fmt.Errorf("invalid -tz %q: %w", *tz, err)
		}
		cfg.Loc = l
		loc = l
	}
	if *from != "" {
		t, err := time.ParseInLocation("2006-01-02", *from, loc)
		if err != nil {
			return Config{}, fmt.Errorf("invalid -from (want YYYY-MM-DD): %w", err)
		}
		cfg.Filter.From = t
	}
	if *to != "" {
		t, err := time.ParseInLocation("2006-01-02", *to, loc)
		if err != nil {
			return Config{}, fmt.Errorf("invalid -to (want YYYY-MM-DD): %w", err)
		}
		cfg.Filter.To = t.AddDate(0, 0, 1) // inclusive of the whole "to" day
	}
	return cfg, nil
}

// ingestResult is what one dataset pipeline hands back.
type ingestResult struct {
	agg     Aggregator
	stats   IngestStats
	err     error
	elapsed time.Duration
}

func ingest(files []string, spec SourceSpec, newAgg func() Aggregator, cfg Config) ingestResult {
	t0 := time.Now()
	agg, st, err := runPipeline(files, spec, newAgg, cfg.Workers, cfg.Batch, cfg.Queue)
	return ingestResult{agg: agg, stats: st, err: err, elapsed: time.Since(t0)}
}

func describeIngest(label string, r ingestResult) string {
	st := r.stats
	return fmt.Sprintf("%-6s %d file(s), %d rows read, %d accepted | rejected: %d no neighborhood, %d bad date, %d outside -from/-to, %d malformed lines | %.2fs",
		label, st.Files, st.RowsRead, st.Rows.Accepted,
		st.Rows.NoNeighborhood, st.Rows.BadDate, st.Rows.OutOfRange, st.BadCSV, r.elapsed.Seconds())
}

func previewList(names []string, limit int) string {
	if len(names) == 0 {
		return "none"
	}
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + fmt.Sprintf(", ... (+%d more)", len(names)-limit)
}

func run(args []string, out io.Writer) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	start := time.Now()

	// Stage 1: ingest both datasets at the same time. Each dataset has its own
	// reader goroutine and its own pool of parser workers.
	var crimeRes, reqRes ingestResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		crimeRes = ingest(cfg.CrimeFiles, crimeSpec(), func() Aggregator {
			return newCrimeAgg(cfg.Filter, cfg.Loc, cfg.Violent)
		}, cfg)
	}()
	go func() {
		defer wg.Done()
		reqRes = ingest(cfg.RequestFiles, requestSpec(), func() Aggregator {
			return newRequestAgg(cfg.Filter, cfg.Loc, cfg.Disorder)
		}, cfg)
	}()
	wg.Wait() // happens-before: the reads below see both goroutines' writes
	if crimeRes.err != nil {
		return fmt.Errorf("crime dataset: %w", crimeRes.err)
	}
	if reqRes.err != nil {
		return fmt.Errorf("311 dataset: %w", reqRes.err)
	}
	crimeAgg := crimeRes.agg.(*CrimeAgg)
	reqAgg := reqRes.agg.(*RequestAgg)

	// Stage 2: join on neighborhood.
	joined := joinDatasets(crimeAgg, reqAgg)
	if len(joined.Rows) == 0 {
		return fmt.Errorf("the datasets share no neighborhoods (crime: %d, 311: %d); check the date filters and that both files cover Baltimore",
			len(crimeAgg.byNeigh), len(reqAgg.byNeigh))
	}

	// Stage 3: answer the five questions concurrently. joined is read-only from
	// here on; each goroutine writes only its own slot of sections.
	questions := allQuestions()
	sections := make([]Section, len(questions))
	var qwg sync.WaitGroup
	for i, q := range questions {
		qwg.Add(1)
		go func(i int, q Question) {
			defer qwg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					sections[i] = Section{
						ID: q.ID(), Stakeholder: q.Stakeholder(), Title: q.Title(), Question: q.Ask(),
						Findings: []string{fmt.Sprintf("analysis failed: %v", rec)},
					}
				}
			}()
			sections[i] = q.Run(joined)
		}(i, q)
	}
	qwg.Wait()

	// Stage 4: build and render the report.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	rb := NewReportBuilder("BALTIMORE CITY CROSS-DATASET INTELLIGENCE REPORT")
	rb.Meta(describeIngest("Crime:", crimeRes))
	rb.Meta(describeIngest("311:", reqRes))
	rb.Meta(fmt.Sprintf("Join:  %d neighborhoods in both datasets | crime only: %s | 311 only: %s",
		len(joined.Rows), previewList(joined.CrimeOnly, 5), previewList(joined.RequestOnly, 5)))
	rb.Meta(fmt.Sprintf("Run:   %d CPU(s), %d worker goroutines per dataset, batch %d rows, queue %d batches | heap in use %.1f MiB, total allocated %.1f MiB, %d GC cycles | wall time %.2fs",
		runtime.NumCPU(), cfg.Workers, cfg.Batch, cfg.Queue,
		float64(ms.HeapAlloc)/(1<<20), float64(ms.TotalAlloc)/(1<<20), ms.NumGC, time.Since(start).Seconds()))
	for _, s := range sections {
		rb.Section(s)
	}
	rb.Note("Neighborhoods are joined by normalised name (upper-case, punctuation removed); neighborhoods present in only one dataset are excluded from the questions.")
	rb.Note("Violent crime and physical disorder are keyword classifications (see -violent and -disorder); counts are raw, not per capita.")
	rb.Note("Quartiles rank the joined neighborhoods; ties are broken alphabetically. Correlation does not imply causation.")
	rb.Note("Use -from/-to so both datasets cover the same period.")
	rep := rb.Build()

	if err := (TextRenderer{}).Render(out, rep); err != nil {
		return err
	}
	if cfg.ReportPath != "" {
		f, err := os.Create(cfg.ReportPath)
		if err != nil {
			return fmt.Errorf("cannot write report: %w", err)
		}
		defer f.Close()
		if err := (MarkdownRenderer{}).Render(f, rep); err != nil {
			return fmt.Errorf("cannot write report: %w", err)
		}
		fmt.Fprintf(out, "Markdown report written to %s\n", cfg.ReportPath)
	}
	return nil
}
