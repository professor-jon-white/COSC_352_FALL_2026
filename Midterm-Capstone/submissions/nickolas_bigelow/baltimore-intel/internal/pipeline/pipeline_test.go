package pipeline

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"baltimore-intel/internal/sample"
)

// End-to-end: generate small sample data, run every stage, check outputs.
func TestEndToEnd(t *testing.T) {
	dir := t.TempDir()
	data, out := filepath.Join(dir, "raw"), filepath.Join(dir, "reports")
	if _, err := sample.Generate(sample.Options{Dir: data, Year: 2022, Seed: 7, Scale: 0.25}); err != nil {
		t.Fatal(err)
	}
	var term bytes.Buffer
	p, err := NewBuilder().DataDir(data).OutputDir(out).Workers(4).Year(2022).
		Formats("terminal,md,html,csv").Stdout(&term).Log(io.Discard).Build()
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("analysis errors: %v", res.Errors)
	}
	if len(res.Report.Sections) != 5 {
		t.Fatalf("%d sections, want 5", len(res.Report.Sections))
	}
	for _, s := range res.Report.Sections {
		if s.Finding == "" || len(s.Blocks) < 2 || len(s.Datasets) < 2 {
			t.Errorf("%s: incomplete section", s.ID)
		}
	}
	text := term.String()
	for _, want := range []string{"Q1", "Q5", "FINDING", "█", "SYNTHETIC SAMPLE DATA"} {
		if !strings.Contains(text, want) {
			t.Errorf("terminal output missing %q", want)
		}
	}
	for _, f := range []string{"report.md", "report.html", "q1_compound_distress.csv", "q4_agencies.csv"} {
		if st, err := os.Stat(filepath.Join(out, f)); err != nil || st.Size() == 0 {
			t.Errorf("missing output %s", f)
		}
	}
}

// The generator plants known relationships; the analyses must find them.
func TestAnalysesRecoverPlantedSignals(t *testing.T) {
	dir := t.TempDir()
	if _, err := sample.Generate(sample.Options{Dir: dir, Year: 2022, Seed: 11, Scale: 0.3}); err != nil {
		t.Fatal(err)
	}
	p, err := NewBuilder().DataDir(dir).Workers(3).Year(2022).Quiet(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	find := func(id string) string {
		for _, s := range res.Report.Sections {
			if s.ID == id {
				return s.Finding
			}
		}
		return ""
	}
	if f := find("q2"); !strings.Contains(f, "more violent crime") {
		t.Errorf("Q2 should find more violence near outlets: %s", f)
	}
	if f := find("q3"); !strings.Contains(f, "strong link") && !strings.Contains(f, "moderate link") {
		t.Errorf("Q3 should find vacancy linked to sanitation: %s", f)
	}
	if f := find("q4"); !strings.Contains(f, "wait longer") {
		t.Errorf("Q4 should find longer waits in high-crime areas: %s", f)
	}
}

// Results must not depend on the number of workers.
func TestWorkerCountDoesNotChangeResults(t *testing.T) {
	dir := t.TempDir()
	if _, err := sample.Generate(sample.Options{Dir: dir, Year: 2022, Seed: 3, Scale: 0.15}); err != nil {
		t.Fatal(err)
	}
	var findings []string
	for _, w := range []int{1, 8} {
		p, err := NewBuilder().DataDir(dir).Workers(w).Year(2022).Quiet(true).Build()
		if err != nil {
			t.Fatal(err)
		}
		res, err := p.Run(context.Background(), false)
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		for _, s := range res.Report.Sections {
			b.WriteString(s.Finding)
		}
		findings = append(findings, b.String())
	}
	if findings[0] != findings[1] {
		t.Fatalf("1 worker and 8 workers disagree:\n%s\n---\n%s", findings[0], findings[1])
	}
}

func TestBuilderValidation(t *testing.T) {
	_, err := NewBuilder().DataDir(t.TempDir()).Workers(0).Radius(1).Formats("pdf").Build()
	if err == nil {
		t.Fatal("want validation errors")
	}
	for _, want := range []string{"workers", "radius", "unknown format", "make sample"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}
