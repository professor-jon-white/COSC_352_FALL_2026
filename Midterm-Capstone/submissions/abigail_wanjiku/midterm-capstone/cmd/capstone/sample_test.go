package main

import (
	"bytes"
	"strings"
	"testing"

	"capstone/internal/ingest"
)

// The sample files in testdata/ are small synthetic extracts with the exact
// columns and quirks of the real Open Baltimore downloads: a UTF-8 byte-order
// mark, CRLF line endings in the 311 file, UTC crime times, space-padded
// fields, ungeocoded rows with no neighborhood or trailing x,y, rows outside
// 2025, open requests, and close times before creation. CI runs the program
// on them, since the real data is not in the repository.
const (
	sampleCrime    = "../../testdata/crime_sample.csv"
	sampleRequests = "../../testdata/requests_311_sample.csv"
)

func sampleConfig() config {
	return config{crimePath: sampleCrime, requestsPath: sampleRequests, year: 2025, workers: 3, chunkSize: 512, reports: "all"}
}

// TestSampleData checks the exact ingest and match results for the sample
// files, in both modes. A 512-byte chunk size splits each file into dozens of
// chunks, so concurrent ingest exercises the worker pool fully.
func TestSampleData(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		res, err := pipeline(sampleConfig(), concurrent)
		if err != nil {
			t.Fatal(err)
		}
		check := func(name string, got ingest.Stats, read, kept, outOfYear, noHood, hoods int) {
			t.Helper()
			if got.Read != read || got.Kept != kept || got.OutOfYear != outOfYear || got.NoNeighborhood != noHood ||
				got.Malformed != 0 || got.Neighborhoods() != hoods {
				t.Errorf("concurrent=%v %s stats = %+v (neighborhoods %d)", concurrent, name, got, got.Neighborhoods())
			}
		}
		check("crime", res.crimeStats, 100, 85, 12, 3, 9)
		check("311", res.requestStats, 110, 105, 0, 5, 9)

		bad := 0
		for _, s := range res.summaries {
			bad += s.BadCloseTimes
		}
		if len(res.summaries) != 79 || bad != 2 {
			t.Errorf("concurrent=%v: %d summaries with %d bad close times, want 79 and 2", concurrent, len(res.summaries), bad)
		}
	}
}

// TestSampleRun runs the whole program on the sample files in compare mode,
// the same way the CI workflow does, and checks every report answers.
func TestSampleRun(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, sampleConfig(), "compare"); err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "Results identical") {
		t.Errorf("compare mode did not report identical results:\n%s", got)
	}
	if n := strings.Count(got, "Answer:"); n != 5 {
		t.Errorf("got %d answers, want 5:\n%s", n, got)
	}
	if strings.Contains(got, "Not enough data") {
		t.Errorf("a report had too little data:\n%s", got)
	}
}
