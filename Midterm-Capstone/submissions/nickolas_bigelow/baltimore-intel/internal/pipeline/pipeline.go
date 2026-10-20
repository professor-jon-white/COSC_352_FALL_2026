// Package pipeline wires the stages together:
//
//	Ingest (4 datasets in parallel, each a worker pool)
//	  → Join (spatial indexes + neighborhood map-reduce, in parallel)
//	  → Analyze (5 questions in parallel)
//	  → Render (terminal / Markdown / HTML / CSV)
//
// A Pipeline is configured with a Builder so every option is validated in
// one place before any work starts.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"baltimore-intel/internal/analysis"
	"baltimore-intel/internal/ingest"
	"baltimore-intel/internal/join"
	"baltimore-intel/internal/model"
	"baltimore-intel/internal/report"
)

// Pipeline is an immutable, validated run configuration.
type Pipeline struct {
	dataDir   string
	outDir    string
	workers   int
	year      int
	radius    float64
	formats   []string
	ascii     bool
	exportCSV bool
	quiet     bool
	sources   []ingest.Source
	questions []analysis.Question
	stdout    io.Writer
	log       io.Writer
}

// Builder constructs a Pipeline.
type Builder struct {
	p    Pipeline
	errs []error
}

func NewBuilder() *Builder {
	return &Builder{p: Pipeline{
		dataDir: "data/raw", outDir: "reports", workers: runtime.NumCPU(), year: 0, radius: 150,
		formats: []string{"terminal", "md", "html"}, exportCSV: true, stdout: os.Stdout, log: os.Stderr,
	}}
}

func (b *Builder) DataDir(d string) *Builder   { b.p.dataDir = d; return b }
func (b *Builder) OutputDir(d string) *Builder { b.p.outDir = d; return b }
func (b *Builder) Year(y int) *Builder         { b.p.year = y; return b }
func (b *Builder) ASCII(on bool) *Builder      { b.p.ascii = on; return b }
func (b *Builder) CSV(on bool) *Builder        { b.p.exportCSV = on; return b }
func (b *Builder) Quiet(on bool) *Builder      { b.p.quiet = on; return b }
func (b *Builder) Stdout(w io.Writer) *Builder { b.p.stdout = w; return b }
func (b *Builder) Log(w io.Writer) *Builder    { b.p.log = w; return b }

func (b *Builder) Workers(n int) *Builder {
	if n < 1 || n > 512 {
		b.errs = append(b.errs, fmt.Errorf("workers must be 1-512, got %d", n))
	}
	b.p.workers = n
	return b
}

func (b *Builder) Radius(m float64) *Builder {
	if m < 10 || m > 2000 {
		b.errs = append(b.errs, fmt.Errorf("radius must be 10-2000 meters, got %.0f", m))
	}
	b.p.radius = m
	return b
}

func (b *Builder) Formats(list string) *Builder {
	b.p.formats = nil
	for _, f := range strings.Split(list, ",") {
		if f = strings.TrimSpace(strings.ToLower(f)); f == "" {
			continue
		}
		if f == "csv" {
			b.p.exportCSV = true
			continue
		}
		if _, err := report.NewRenderer(f, false); err != nil {
			b.errs = append(b.errs, err)
		}
		b.p.formats = append(b.p.formats, f)
	}
	return b
}

func (b *Builder) Build() (*Pipeline, error) {
	p := b.p
	for _, k := range ingest.Kinds {
		s, err := ingest.NewSource(k)
		if err != nil {
			b.errs = append(b.errs, err)
			continue
		}
		path := filepath.Join(p.dataDir, s.FileName())
		if _, err := os.Stat(path); err != nil {
			b.errs = append(b.errs, fmt.Errorf("missing %s (%s). Run `make sample` for bundled demo data or `make fetch` for live Open Baltimore data", path, s.Name()))
		}
		p.sources = append(p.sources, s)
	}
	p.questions = analysis.All()
	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}
	return &p, nil
}

// Timing records one stage.
type Timing struct {
	Stage    string
	Duration time.Duration
}

// Result is everything a run produced.
type Result struct {
	Report  *report.Report
	Ingest  []ingest.Stats
	Timings []Timing
	Files   []string
	Errors  []error
}

func (p *Pipeline) logf(f string, a ...any) {
	if !p.quiet {
		fmt.Fprintf(p.log, f+"\n", a...)
	}
}

// Run executes every stage. Rendering is skipped when render=false (benchmarks).
func (p *Pipeline) Run(ctx context.Context, render bool) (*Result, error) {
	res := &Result{}
	stage := func(name string, start time.Time) {
		d := time.Since(start)
		res.Timings = append(res.Timings, Timing{name, d})
		p.logf("  %-8s %8.1f ms", name, float64(d.Microseconds())/1000)
	}
	mem := newMemSampler()
	defer mem.stop()
	runStart := time.Now()

	// ---- Ingest: all datasets concurrently ----------------------------------
	t := time.Now()
	ds := model.NewDataset()
	ds.Year, ds.RadiusM = p.year, p.radius
	res.Ingest = make([]ingest.Stats, len(p.sources))
	errs := make([]error, len(p.sources))
	var wg sync.WaitGroup
	for i, s := range p.sources {
		wg.Add(1)
		go func(i int, s ingest.Source) {
			defer wg.Done()
			res.Ingest[i], errs[i] = s.Load(ctx, p.dataDir, p.workers, ds)
		}(i, s)
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	stage("ingest", t)
	for _, st := range res.Ingest {
		p.logf("    %s", st)
	}
	if len(ds.Crimes) == 0 || len(ds.Requests) == 0 {
		return nil, fmt.Errorf("no crime or 311 records left after filtering (year=%d). Check -year matches the data", p.year)
	}
	heapAfterIngest := mem.snapshot()

	// ---- Join -------------------------------------------------------------
	t = time.Now()
	j := join.Build(ds, p.workers)
	stage("join", t)

	// ---- Analyze: questions concurrently ---------------------------------
	t = time.Now()
	results := analysis.RunAll(j, p.questions)
	stage("analyze", t)

	rep := &report.Report{
		Title:     "Baltimore City Cross-Dataset Intelligence Report",
		Generated: time.Now(),
		DataNote:  dataNote(p.dataDir),
	}
	yearLabel := "all years in files"
	if p.year > 0 {
		yearLabel = fmt.Sprint(p.year)
	}
	rep.Inputs = [][2]string{
		{"Analysis year", yearLabel},
		{"Part 1 crimes", report.Int(len(ds.Crimes))},
		{"311 service requests", report.Int(len(ds.Requests))},
		{"Liquor outlets", report.Int(len(ds.Outlets))},
		{"Open vacant building notices", report.Int(len(ds.Vacants))},
		{"Neighborhoods (join key)", report.Int(ds.Hoods.Len() - 1)},
		{"Outlet radius", fmt.Sprintf("%.0f m", p.radius)},
	}
	for i, r := range results {
		if r.Err != nil {
			res.Errors = append(res.Errors, r.Err)
			rep.Sections = append(rep.Sections, &report.Section{ID: p.questions[i].ID(),
				Question: "Analysis unavailable: " + p.questions[i].ID(), Finding: r.Err.Error()})
			continue
		}
		rep.Sections = append(rep.Sections, r.Section)
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	recBytes := uintptr(len(ds.Crimes))*unsafe.Sizeof(model.Crime{}) +
		uintptr(len(ds.Requests))*unsafe.Sizeof(model.ServiceRequest{}) +
		uintptr(len(ds.Vacants))*unsafe.Sizeof(model.VacantNotice{})
	rep.Runtime = [][2]string{
		{"Worker goroutines / stage", fmt.Sprint(p.workers)},
		{"GOMAXPROCS / CPUs", fmt.Sprintf("%d / %d", runtime.GOMAXPROCS(0), runtime.NumCPU())},
		{"Stage times", timingLine(res.Timings)},
		{"Wall time (excl. render)", time.Since(runStart).Round(time.Millisecond).String()},
		{"Record memory (flat slices)", fmt.Sprintf("%s (Crime %dB, 311 %dB, Vacant %dB each)", mb(uint64(recBytes)),
			unsafe.Sizeof(model.Crime{}), unsafe.Sizeof(model.ServiceRequest{}), unsafe.Sizeof(model.VacantNotice{}))},
		{"Heap in use after ingest", mb(heapAfterIngest)},
		{"Peak heap observed", mb(mem.peak())},
		{"Total allocated", mb(ms.TotalAlloc)},
		{"GC cycles / total pause", fmt.Sprintf("%d / %s", ms.NumGC, time.Duration(ms.PauseTotalNs).Round(time.Microsecond))},
		{"Peak goroutines observed", fmt.Sprint(mem.peakG.Load())},
	}
	for i, r := range results {
		rep.Runtime = append(rep.Runtime, [2]string{fmt.Sprintf("Q%d time", i+1), r.Duration.Round(time.Microsecond).String()})
	}
	res.Report = rep
	if !render {
		return res, nil
	}

	// ---- Render ------------------------------------------------------------
	t = time.Now()
	if err := os.MkdirAll(p.outDir, 0o755); err != nil {
		return res, err
	}
	for _, f := range p.formats {
		r, _ := report.NewRenderer(f, p.ascii)
		if r.Name() == "terminal" {
			if err := r.Render(p.stdout, rep); err != nil {
				return res, err
			}
			continue
		}
		path := filepath.Join(p.outDir, "report."+r.Name())
		out, err := os.Create(path)
		if err != nil {
			return res, err
		}
		err = r.Render(out, rep)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return res, err
		}
		res.Files = append(res.Files, path)
	}
	if p.exportCSV {
		files, err := report.ExportCSV(p.outDir, rep)
		if err != nil {
			return res, err
		}
		res.Files = append(res.Files, files...)
	}
	stage("render", t)
	return res, nil
}

func timingLine(ts []Timing) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = fmt.Sprintf("%s %.0fms", t.Stage, float64(t.Duration.Microseconds())/1000)
	}
	return strings.Join(parts, ", ")
}

func mb(b uint64) string { return fmt.Sprintf("%.1f MB", float64(b)/(1<<20)) }

func dataNote(dir string) string {
	if b, err := os.ReadFile(filepath.Join(dir, "SAMPLE_DATA.txt")); err == nil {
		_ = b
		return "SYNTHETIC SAMPLE DATA: generated in the exact Open Baltimore schemas for offline demos. " +
			"Run `make fetch` to analyze live Open Baltimore data; findings below describe the sample, not the city."
	}
	if b, err := os.ReadFile(filepath.Join(dir, "SOURCE.txt")); err == nil {
		first := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)[0]
		return "Source: Open Baltimore (data.baltimorecity.gov). " + first
	}
	return "Source: raw files in " + dir
}

// memSampler tracks peak heap and goroutine counts during the run.
type memSampler struct {
	peakHeap atomic.Uint64
	peakG    atomic.Int64
	done     chan struct{}
	stopped  sync.Once
}

func newMemSampler() *memSampler {
	m := &memSampler{done: make(chan struct{})}
	go func() {
		tk := time.NewTicker(20 * time.Millisecond)
		defer tk.Stop()
		for {
			m.observe()
			select {
			case <-tk.C:
			case <-m.done:
				return
			}
		}
	}()
	return m
}

func (m *memSampler) observe() uint64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	for {
		cur := m.peakHeap.Load()
		if ms.HeapAlloc <= cur || m.peakHeap.CompareAndSwap(cur, ms.HeapAlloc) {
			break
		}
	}
	g := int64(runtime.NumGoroutine())
	for {
		cur := m.peakG.Load()
		if g <= cur || m.peakG.CompareAndSwap(cur, g) {
			break
		}
	}
	return ms.HeapAlloc
}

func (m *memSampler) snapshot() uint64 { return m.observe() }
func (m *memSampler) peak() uint64     { m.observe(); return m.peakHeap.Load() }
func (m *memSampler) stop()            { m.stopped.Do(func() { close(m.done) }) }

// SetMemoryLimit applies a soft heap limit (Go 1.19+ GOMEMLIMIT semantics).
func SetMemoryLimit(mbLimit int) {
	if mbLimit > 0 {
		debug.SetMemoryLimit(int64(mbLimit) << 20)
	}
}
