// Command baltimore-engine runs a four-stage pipeline:
// ingest -> processor -> analytics -> ui.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"baltimore-engine/analytics"
	"baltimore-engine/ingest"
	"baltimore-engine/processor"
	"baltimore-engine/ui"
)

// pipelineTimeout bounds the whole run, dominated by the live API fetches.
const pipelineTimeout = 3 * time.Minute

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Ctrl-C cancels in-flight fetches; the timeout bounds the whole pipeline.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, pipelineTimeout)
	defer cancel()

	// Stage 1: concurrently fetch both datasets.
	start := time.Now()
	progress("ingest: fetching 311 requests and vacant buildings...")
	data, err := ingest.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}
	progress("ingest: %d 311 requests, %d vacant notices (%s)",
		len(data.ServiceRequests), len(data.VacantBuildings), since(start))

	// Stage 2: join datasets concurrently.
	start = time.Now()
	joined := processor.Join(ctx, data.ServiceRequests, data.VacantBuildings)
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("processor: %w", err)
	}
	progress("processor: joined %d neighborhoods, %d ZIP codes (%s)",
		len(joined.Neighborhoods), len(joined.Zips), since(start))

	// Stage 3: answer the stakeholder questions concurrently.
	start = time.Now()
	results := analytics.RunAll(joined)
	progress("analytics: answered %d questions (%s)", len(results), since(start))

	// Stage 4: render results. Progress goes to stderr, results to stdout.
	fmt.Fprintln(os.Stdout)
	ui.RenderAll(os.Stdout, results)
	return nil
}

func progress(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func since(t time.Time) string { return time.Since(t).Round(time.Millisecond).String() }
