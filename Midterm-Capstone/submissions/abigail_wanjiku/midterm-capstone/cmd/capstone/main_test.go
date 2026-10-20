package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeSample writes small crime and 311 CSVs with the real column names and
// the quirks of the real files: short ungeocoded rows, quoted fields with
// commas, mixed-case neighborhoods, UTC crime times, open requests, and a
// close time far before creation.
func writeSample(t *testing.T, dir string) (crimePath, requestsPath string) {
	t.Helper()
	rng := rand.New(rand.NewPCG(1, 2))
	hoods := []string{"Abell", "BARCLAY", " Canton ", "Downtown", "Frankford", "Hampden", "Upton", "Belair-Edison"}
	at := func() time.Time {
		return time.Date(2025, time.Month(rng.IntN(12)+1), rng.IntN(28)+1, rng.IntN(24), rng.IntN(60), 0, 0, time.UTC)
	}
	const layout = "1/2/2006 3:04:05 PM"

	var c strings.Builder
	c.WriteString("RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Neighborhood,Location,x,y\n")
	for i := range 1500 {
		when := at()
		if i%50 == 0 {
			when = when.AddDate(-1, 0, 0) // out of year
		}
		desc := []string{"COMMON ASSAULT", "BURGLARY", "LARCENY FROM AUTO", "ROBBERY"}[rng.IntN(4)]
		inOut := "IO"[rng.IntN(2)]
		hood := hoods[rng.IntN(len(hoods))]
		if i%97 == 0 {
			fmt.Fprintf(&c, "%d,%d,%s,4E,%s,%c,,%s,\"%d N CHARLES ST, BALTIMORE\"\n", i, 250000+i, when.Format(layout), desc, inOut, hood, i) // short row
			continue
		}
		fmt.Fprintf(&c, "%d,%d,%s,4E,%s,%c,,%s,\"%d N CHARLES ST, BALTIMORE\",-76.6,39.3\n", i, 250000+i, when.Format(layout), desc, inOut, hood, i)
	}

	types := []string{"HCD-Illegal Dumping", "SW-Dirty Alley", "BGE-StLight(s) Out", "TRM-Street Light Out",
		"SW-Rat Rubout Proactive", "TRS-Parking Complaint", "SW-Recycling", "FCPF-Trial Request-Red Light"}
	var r strings.Builder
	r.WriteString("SRRecordID,ServiceRequestNum,SRType,CreatedDate,SRStatus,CloseDate,Agency,Address,Neighborhood,x,y\n")
	for i := range 4000 {
		created := at()
		closed := ""
		switch i % 4 {
		case 1, 2:
			closed = created.Add(time.Duration(rng.IntN(30*24*60)) * time.Minute).Format(layout)
		case 3:
			if i%40 == 3 {
				closed = created.AddDate(0, -2, 0).Format(layout)
			}
		}
		fmt.Fprintf(&r, "id%d,25-%08d,%s,%s,Closed,%s,Solid Waste      ,\"%d MAIN ST, Baltimore City, 21201\",%s,-76.6,39.3\n",
			i, i, types[rng.IntN(len(types))], created.Format(layout), closed, i, hoods[rng.IntN(len(hoods))])
	}

	crimePath = filepath.Join(dir, "crime.csv")
	requestsPath = filepath.Join(dir, "requests_311.csv")
	if err := os.WriteFile(crimePath, []byte(c.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestsPath, []byte(r.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return crimePath, requestsPath
}

// TestPipelineCompare runs the whole program in compare mode on a small
// sample: sequential and concurrent ingest and match both run, and compare
// fails if any output differs. A tiny chunk size forces the ingest splitter
// to produce many chunks, so the worker pool and the match shards all have
// work to do at once. Run with -race on a machine with cgo to check for
// data races:
//
//	go test -race ./cmd/capstone
func TestPipelineCompare(t *testing.T) {
	crimePath, requestsPath := writeSample(t, t.TempDir())
	cfg := config{
		crimePath:    crimePath,
		requestsPath: requestsPath,
		year:         2025,
		workers:      4,
		chunkSize:    1024,
		reports:      "all",
	}

	var out bytes.Buffer
	if err := run(&out, cfg, "compare"); err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"Results identical",
		"Q1 [sanitation]", "Q2 [distress]", "Q3 [close-time]", "Q4 [streetlights]", "Q5 [high-311-low-crime]",
		"Answer:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n%s", want, got)
		}
	}
}

// TestPipelineConcurrentRepeatable runs the concurrent pipeline several times
// with different worker counts; every run must equal the sequential result.
func TestPipelineConcurrentRepeatable(t *testing.T) {
	crimePath, requestsPath := writeSample(t, t.TempDir())
	base := config{crimePath: crimePath, requestsPath: requestsPath, year: 2025, chunkSize: 700}

	want, err := pipeline(base, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.crimes) == 0 || len(want.requests) == 0 || len(want.summaries) == 0 {
		t.Fatalf("sample produced no data: %d crimes, %d requests", len(want.crimes), len(want.requests))
	}
	for _, workers := range []int{1, 2, 3, 8} {
		cfg := base
		cfg.workers = workers
		got, err := pipeline(cfg, true)
		if err != nil {
			t.Fatal(err)
		}
		if diffs := differences(want, got); len(diffs) > 0 {
			t.Errorf("workers=%d: concurrent differs from sequential: %v", workers, diffs)
		}
	}
}
