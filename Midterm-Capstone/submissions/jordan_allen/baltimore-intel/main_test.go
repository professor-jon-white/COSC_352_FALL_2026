package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNormalizeNeighborhood(t *testing.T) {
	cases := map[string]string{
		"  Mount  Vernon ":           "MOUNT VERNON",
		"Mt. Washington":             "MT WASHINGTON",
		"Sandtown-Winchester":        "SANDTOWN WINCHESTER",
		"Hamilton & Lauraville":      "HAMILTON AND LAURAVILLE",
		"":                           "",
		"   ":                        "",
		"O'Donnell Heights":          "O DONNELL HEIGHTS",
		"downtown":                   "DOWNTOWN",
	}
	for in, want := range cases {
		if got := normalizeNeighborhood(in); got != want {
			t.Errorf("normalizeNeighborhood(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	p := &timeParser{}
	cases := []struct {
		in   string
		hour int
		ok   bool
	}{
		{"2025/03/04 17:30:00+00", 17, true},
		{"2025-03-04T09:15:00.000Z", 9, true},
		{"3/4/2025 5:30:00 PM", 17, true},
		{"2025-03-04", 0, true},
		{"not-a-date", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := p.parse(c.in)
		if ok != c.ok {
			t.Errorf("parse(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && got.Hour() != c.hour {
			t.Errorf("parse(%q) hour = %d, want %d", c.in, got.Hour(), c.hour)
		}
	}
}

func TestStatistics(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	if r := pearson(x, []float64{2, 4, 6, 8, 10}); math.Abs(r-1) > 1e-9 {
		t.Errorf("pearson perfect positive = %v", r)
	}
	if r := pearson(x, []float64{10, 8, 6, 4, 2}); math.Abs(r+1) > 1e-9 {
		t.Errorf("pearson perfect negative = %v", r)
	}
	if r := spearman(x, []float64{1, 4, 9, 16, 25}); math.Abs(r-1) > 1e-9 {
		t.Errorf("spearman monotonic = %v", r)
	}
	if r := pearson(x, []float64{3, 3, 3, 3, 3}); !math.IsNaN(r) {
		t.Errorf("pearson with zero variance = %v, want NaN", r)
	}
	if m := medianF32([]float32{5, 1, 3}); m != 3 {
		t.Errorf("median odd = %v", m)
	}
	if m := medianF32([]float32{4, 1, 3, 2}); m != 2.5 {
		t.Errorf("median even = %v", m)
	}
	rk := ranks([]float64{10, 20, 20, 30})
	want := []float64{1, 2.5, 2.5, 4}
	for i := range want {
		if rk[i] != want[i] {
			t.Errorf("ranks[%d] = %v, want %v", i, rk[i], want[i])
		}
	}
}

func TestQuartiles(t *testing.T) {
	q := quartiles(8, func(i int) float64 { return float64(i) })
	want := []int{1, 1, 2, 2, 3, 3, 4, 4}
	for i := range want {
		if q[i] != want[i] {
			t.Fatalf("quartiles = %v, want %v", q, want)
		}
	}
}

func TestResolveColumnsAliasesAndMissing(t *testing.T) {
	header := []string{"X", "crime_date_time", "DESCRIPTION", "neighborhood", "Total Incidents"}
	spec := crimeSpec()
	// "crime_date_time" normalises to "crimedatetime", which matches the CrimeDateTime alias.
	cols, err := resolveColumns(header, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cols[crimeWhen] != 1 || cols[crimeDesc] != 2 || cols[crimeNeigh] != 3 || cols[crimeCount] != 4 {
		t.Errorf("unexpected column mapping: %v", cols)
	}
	if _, err := resolveColumns([]string{"foo", "bar"}, spec); err == nil {
		t.Error("expected an error for missing required columns")
	}
}

const testCrimeCSV = `CrimeDateTime,Description,Neighborhood,Total_Incidents
2025/01/01 10:00:00+00,ROBBERY,Alpha,1
2025/01/01 11:00:00+00,LARCENY,Alpha,1
2025/01/01 23:00:00+00,HOMICIDE,Beta,1
2025/01/02 01:00:00+00,LARCENY,Gamma,2
2025/01/02 02:00:00+00,LARCENY,,1
not-a-date,LARCENY,Alpha,1
`

const testRequestCSV = `CreatedDate,CloseDate,SRType,Neighborhood
2025/01/01 09:00:00+00,2025/01/03 09:00:00+00,Street Light Out,Alpha
2025/01/01 10:00:00+00,,Rodent Complaint,Alpha
2025/01/01 11:00:00+00,2025/01/02 11:00:00+00,Pothole,Beta
2025/01/01 12:00:00+00,,Pothole,Delta
`

func ingestTestData(t *testing.T) (*CrimeAgg, *RequestAgg, IngestStats) {
	t.Helper()
	crimePath := writeTempFile(t, "crime.csv", testCrimeCSV)
	reqPath := writeTempFile(t, "req.csv", testRequestCSV)

	// batch size 2 and 3 workers force several batches and real concurrency.
	ca, cst, err := runPipeline([]string{crimePath}, crimeSpec(), func() Aggregator {
		return newCrimeAgg(TimeFilter{}, nil, splitList(defaultViolent))
	}, 3, 2, 2)
	if err != nil {
		t.Fatalf("crime pipeline: %v", err)
	}
	ra, _, err := runPipeline([]string{reqPath}, requestSpec(), func() Aggregator {
		return newRequestAgg(TimeFilter{}, nil, splitList(defaultDisorder))
	}, 3, 2, 2)
	if err != nil {
		t.Fatalf("311 pipeline: %v", err)
	}
	return ca.(*CrimeAgg), ra.(*RequestAgg), cst
}

func TestPipelineAggregation(t *testing.T) {
	ca, ra, cst := ingestTestData(t)

	if cst.RowsRead != 6 {
		t.Errorf("RowsRead = %d, want 6", cst.RowsRead)
	}
	if cst.Rows.Accepted != 4 || cst.Rows.NoNeighborhood != 1 || cst.Rows.BadDate != 1 {
		t.Errorf("crime row stats = %+v", cst.Rows)
	}
	if a := ca.byNeigh["ALPHA"]; a == nil || a.Total != 2 || a.Violent != 1 {
		t.Errorf("ALPHA crime stats = %+v", a)
	}
	if b := ca.byNeigh["BETA"]; b == nil || b.Total != 1 || b.Violent != 1 {
		t.Errorf("BETA crime stats = %+v", b)
	}
	if g := ca.byNeigh["GAMMA"]; g == nil || g.Total != 2 || g.Violent != 0 {
		t.Errorf("GAMMA crime stats = %+v", g)
	}

	a := ra.byNeigh["ALPHA"]
	if a == nil || a.Total != 2 || a.Disorder != 2 || a.Closed != 1 {
		t.Fatalf("ALPHA 311 stats = %+v", a)
	}
	d := a.Durations["STREET LIGHT OUT"]
	if len(d) != 1 || math.Abs(float64(d[0])-2) > 1e-6 {
		t.Errorf("ALPHA street-light durations = %v, want [2]", d)
	}
}

func TestJoin(t *testing.T) {
	ca, ra, _ := ingestTestData(t)
	j := joinDatasets(ca, ra)
	if len(j.Rows) != 2 || j.Rows[0].Name != "ALPHA" || j.Rows[1].Name != "BETA" {
		t.Fatalf("joined rows = %v", j.Rows)
	}
	if len(j.CrimeOnly) != 1 || j.CrimeOnly[0] != "GAMMA" {
		t.Errorf("CrimeOnly = %v", j.CrimeOnly)
	}
	if len(j.RequestOnly) != 1 || j.RequestOnly[0] != "DELTA" {
		t.Errorf("RequestOnly = %v", j.RequestOnly)
	}
}

func TestFilterExcludesOutOfRange(t *testing.T) {
	crimePath := writeTempFile(t, "crime.csv", testCrimeCSV)
	cfg, err := parseConfig([]string{"-crime", crimePath, "-requests", crimePath, "-from", "2025-01-02", "-to", "2025-01-02"})
	if err != nil {
		t.Fatal(err)
	}
	ca, st, err := runPipeline([]string{crimePath}, crimeSpec(), func() Aggregator {
		return newCrimeAgg(cfg.Filter, cfg.Loc, cfg.Violent)
	}, 2, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Only the 2025-01-02 01:00 GAMMA row passes; the 02:00 row has no neighborhood.
	if st.Rows.Accepted != 1 || st.Rows.OutOfRange != 3 {
		t.Errorf("row stats = %+v, want 1 accepted and 3 out of range", st.Rows)
	}
	if g := ca.(*CrimeAgg).byNeigh["GAMMA"]; g == nil || g.Total != 2 {
		t.Errorf("GAMMA = %+v", g)
	}
}

func syntheticJoined() *Joined {
	c := newCrimeAgg(TimeFilter{}, nil, splitList(defaultViolent))
	r := newRequestAgg(TimeFilter{}, nil, splitList(defaultDisorder))
	for i := 1; i <= 12; i++ {
		name := fmt.Sprintf("N%02d", i)
		c.byNeigh[name] = &CrimeStats{
			Total: i * 10, Violent: i * 2,
			ByCat: map[string]int{"LARCENY": i * 8, "ROBBERY": i * 2},
		}
		r.byNeigh[name] = &RequestStats{
			Total: i * 20, Disorder: i * 5, Closed: i * 15,
			ByType:    map[string]int{"POTHOLE": i * 10, "TRASH": i * 10},
			Durations: map[string][]float32{"POTHOLE": {1, 2, 3}, "TRASH": {2, 4}},
		}
	}
	c.byHour[18] = 50
	c.byHour[3] = 10
	r.byHour[10] = 40
	r.byHour[18] = 5
	return joinDatasets(c, r)
}

func TestAllQuestionsRun(t *testing.T) {
	j := syntheticJoined()
	qs := allQuestions()
	if len(qs) != 5 {
		t.Fatalf("expected 5 questions, got %d", len(qs))
	}
	for _, q := range qs {
		s := q.Run(j)
		if s.ID != q.ID() || len(s.Findings) == 0 || s.Body == "" {
			t.Errorf("%s produced an incomplete section: %+v", q.ID(), s)
		}
	}
	// Crime and 311 volumes are perfectly proportional in the synthetic data.
	s1 := qs[0].Run(j)
	if !strings.Contains(s1.Findings[0], "r = 1.00") {
		t.Errorf("Q1 first finding = %q", s1.Findings[0])
	}
}

func TestReportRenderers(t *testing.T) {
	j := syntheticJoined()
	rb := NewReportBuilder("TEST REPORT")
	rb.Meta("meta line")
	for _, q := range allQuestions() {
		rb.Section(q.Run(j))
	}
	rb.Note("a note")
	rep := rb.Build()

	var txt, md bytes.Buffer
	if err := (TextRenderer{}).Render(&txt, rep); err != nil {
		t.Fatal(err)
	}
	if err := (MarkdownRenderer{}).Render(&md, rep); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TEST REPORT", "Q1", "Q5", "Key findings", "a note"} {
		if !strings.Contains(txt.String(), want) {
			t.Errorf("text report is missing %q", want)
		}
	}
	if !strings.HasPrefix(md.String(), "# TEST REPORT") || !strings.Contains(md.String(), "```text") {
		t.Errorf("markdown report has unexpected shape: %.60q", md.String())
	}
}

func TestRunEndToEndWithSampleFiles(t *testing.T) {
	crimePath := writeTempFile(t, "crime.csv", testCrimeCSV)
	reqPath := writeTempFile(t, "req.csv", testRequestCSV)
	var out bytes.Buffer
	if err := run([]string{"-crime", crimePath, "-requests", reqPath, "-workers", "2"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "BALTIMORE CITY CROSS-DATASET INTELLIGENCE REPORT") {
		t.Errorf("unexpected output: %.200q", out.String())
	}
}

// TestRealDataExtract runs the whole program on the small extract of the real
// Open Baltimore files that ships in data/. It is skipped if the files are absent.
func TestRealDataExtract(t *testing.T) {
	crimePath := filepath.Join("data", "crime_sample_week.csv")
	reqPath := filepath.Join("data", "requests_sample_week.csv")
	if _, err := os.Stat(crimePath); err != nil {
		t.Skip("real-data extract not present")
	}
	if _, err := os.Stat(reqPath); err != nil {
		t.Skip("real-data extract not present")
	}
	var out bytes.Buffer
	if err := run([]string{"-crime", crimePath, "-requests", reqPath}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"1137 rows read, 1137 accepted", "1540 rows read, 1540 accepted"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q", want)
		}
	}
}
