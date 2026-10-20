package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"
)

const (
	requestsURL = "https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0/query"
	vacantURL   = "https://data.baltimorecity.gov/api/download/v1/items/4db6d1e54e714a3e8125990a09d4623d/geojson?layers=2"
)

func main() {
	workers := flag.Int("workers", defaultWorkers(), "bounded parsing worker count")
	rawDir := flag.String("raw-dir", "data/raw", "directory for saved raw GeoJSON pages")
	reportDir := flag.String("out", "reports", "directory for generated report and CSV")
	offline := flag.Bool("offline", false, "analyze the bundled raw example files instead of downloading live data")
	fromRaw := flag.String("from-raw", "", "replay previously saved GeoJSON pages from this directory")
	flag.Parse()
	if *workers < 1 {
		log.Fatal("-workers must be at least 1")
	}
	if *offline && *fromRaw != "" {
		log.Fatal("use either -offline or -from-raw, not both")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	start := time.Now()
	var sources []Source
	if *offline {
		sources = []Source{
			{Name: "311 requests", Key: "requests", LocalPath: "data/samples/311_sample.geojson"},
			{Name: "vacant building rehabs", Key: "vacant", LocalPath: "data/samples/vacant_rehabs_sample.geojson"},
		}
	} else if *fromRaw != "" {
		sources = []Source{
			{Name: "saved 311 request pages", Key: "requests", LocalPath: *fromRaw},
			{Name: "saved vacant-building rehab snapshot", Key: "vacant", LocalPath: *fromRaw},
		}
	} else {
		sources = []Source{
			{Name: "311 requests 2026", Key: "requests", URL: requestsURL, Where: "Neighborhood IS NOT NULL", PageSize: 2000, SortField: "RowID"},
			{Name: "vacant building rehabs", Key: "vacant", URL: vacantURL},
		}
	}

	result, err := RunPipeline(ctx, sources, *workers, *rawDir)
	if err != nil {
		log.Fatal(err)
	}
	result.GeneratedAt = time.Now().Format(time.RFC3339)
	result.Workers = *workers
	result.OfflineExample = *offline
	result.RawReplay = *fromRaw != ""
	result.Elapsed = time.Since(start).Round(time.Millisecond).String()
	if err := WriteOutputs(*reportDir, result); err != nil {
		log.Fatal(err)
	}
	printDashboard(result)
	fmt.Printf("\nWrote %s and %s\n", filepath.Join(*reportDir, "analysis.md"), filepath.Join(*reportDir, "neighborhoods.csv"))
	if *offline {
		fmt.Println("Offline example mode: this small example demonstrates the pipeline; use the default live mode for current citywide analysis.")
	}
}

func defaultWorkers() int {
	n := runtime.NumCPU()
	if n > 8 {
		n = 8
	}
	if n < 1 {
		return 1
	}
	return n
}

func printDashboard(r Analysis) {
	fmt.Println("BALTIMORE CIVIC PULSE — 311 × VACANT BUILDING REHABS")
	fmt.Printf("Requests: %s | Rehab locations: %s | Matched neighborhoods: %d | Workers: %d | Elapsed: %s\n",
		comma(r.RequestCount), comma(r.RehabCount), r.MatchedNeighborhoods, r.Workers, r.Elapsed)
	fmt.Println("\nTop neighborhoods by joint indicator (311 requests + rehab locations)")
	fmt.Printf("%-30s %10s %10s %10s\n", "NEIGHBORHOOD", "311", "REHABS", "NON-CLOSED")
	for _, n := range firstN(r.Neighborhoods, 10) {
		fmt.Printf("%-30s %10s %10s %10s\n", clip(n.Name, 30), comma(n.Requests), comma(n.RehabSites), comma(n.Unresolved))
	}
	fmt.Printf("\nNeighborhood rank correlation (Spearman): %s\n", r.SpearmanLabel)
	fmt.Println("Interpret counts as reported activity, not a measure of need or causation.")
}

func firstN[T any](xs []T, n int) []T {
	if len(xs) < n {
		n = len(xs)
	}
	return xs[:n]
}
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 0 {
		return s
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Source is one raw ArcGIS GeoJSON stream or one bundled raw fixture.
type Source struct {
	Name      string
	Key       string
	URL       string
	Where     string
	LocalPath string
	PageSize  int
	SortField string
}

type rawFeatureCollection struct {
	Type       string `json:"type"`
	Properties struct {
		Exceeded bool `json:"exceededTransferLimit"`
	} `json:"properties"`
	Features []json.RawMessage `json:"features"`
}

func validateSources(src []Source) error {
	seen := map[string]bool{}
	for _, s := range src {
		if s.Key != "requests" && s.Key != "vacant" {
			return fmt.Errorf("unknown source key %q", s.Key)
		}
		seen[s.Key] = true
	}
	if !seen["requests"] || !seen["vacant"] {
		return errors.New("both the 311 and vacant-building datasets are required")
	}
	return nil
}

func cleanNeighborhood(s string) string { return s }
