// Package pipeline wires the stages together:
//
//	1. Ingest     three files are streamed and parsed concurrently
//	2. Window     the shared analysis period is derived (or taken from flags)
//	3. Index      a block -> neighborhood index is built from crime + 311
//	4. Aggregate  every dataset is joined into neighborhoods and street blocks
//	              by fan-out workers whose private results are merged (fan-in)
//
// Analysis and reporting are separate packages that only see the finished,
// read-only model.Dataset.
package pipeline

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"baltimoreintel/internal/geo"
	"baltimoreintel/internal/ingest"
	"baltimoreintel/internal/model"
)

// Config is the complete, validated set of pipeline options.
type Config struct {
	CrimePath    string
	RequestsPath string
	LicensesPath string
	Workers      int
	From, To     time.Time // zero means "derive from the 311 data"
}

// Builder assembles a Config step by step (Builder pattern).
type Builder struct {
	cfg Config
	log func(format string, args ...any)
}

// NewBuilder returns a Builder with one worker per CPU.
func NewBuilder() *Builder {
	return &Builder{
		cfg: Config{Workers: runtime.NumCPU()},
		log: func(string, ...any) {},
	}
}

// Crime sets the NIBRS crime CSV path.
func (b *Builder) Crime(path string) *Builder { b.cfg.CrimePath = path; return b }

// Requests sets the 311 requests CSV path.
func (b *Builder) Requests(path string) *Builder { b.cfg.RequestsPath = path; return b }

// Licenses sets the liquor licenses CSV path.
func (b *Builder) Licenses(path string) *Builder { b.cfg.LicensesPath = path; return b }

// Workers sets the number of worker goroutines used by each stage.
func (b *Builder) Workers(n int) *Builder { b.cfg.Workers = n; return b }

// Window fixes the analysis period. Zero values mean "derive automatically".
func (b *Builder) Window(from, to time.Time) *Builder {
	b.cfg.From, b.cfg.To = from, to
	return b
}

// Logger sets a progress logger.
func (b *Builder) Logger(fn func(format string, args ...any)) *Builder { b.log = fn; return b }

// Build validates the configuration and returns a runnable Pipeline.
func (b *Builder) Build() (*Pipeline, error) {
	if b.cfg.CrimePath == "" || b.cfg.RequestsPath == "" || b.cfg.LicensesPath == "" {
		return nil, errors.New("crime, 311 and license file paths are all required")
	}
	if b.cfg.Workers < 1 {
		b.cfg.Workers = 1
	}
	return &Pipeline{cfg: b.cfg, log: b.log}, nil
}

// Pipeline runs the ingest -> join stages.
type Pipeline struct {
	cfg Config
	log func(format string, args ...any)
}

// StageTime records how long a stage took.
type StageTime struct {
	Name string
	Took time.Duration
}

// Output is everything Run produces.
type Output struct {
	Data   *model.Dataset
	Loads  []ingest.Stats // crime, 311, licenses (in that order)
	Stages []StageTime
}

// Run executes all stages.
func (p *Pipeline) Run() (*Output, error) {
	out := &Output{}
	w := p.cfg.Workers
	mark := func(name string, start time.Time) {
		out.Stages = append(out.Stages, StageTime{Name: name, Took: time.Since(start)})
	}

	// Stage 1: ingest the three files concurrently.
	t0 := time.Now()
	p.log("[1/4] Ingesting 3 datasets (%d workers per file)...", w)
	var (
		crimes     []model.Crime
		reqs       []model.Request
		lics       []model.License
		cst, rst   ingest.Stats
		lst        ingest.Stats
		cerr, rerr error
		lerr       error
	)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		crimes, cst, cerr = ingest.Load[model.Crime](p.cfg.CrimePath, w, ingest.CrimeParser)
	}()
	go func() {
		defer wg.Done()
		reqs, rst, rerr = ingest.Load[model.Request](p.cfg.RequestsPath, w, ingest.RequestParser)
	}()
	go func() {
		defer wg.Done()
		lics, lst, lerr = ingest.Load[model.License](p.cfg.LicensesPath, w, ingest.LicenseParser)
	}()
	wg.Wait()
	for _, e := range []error{cerr, rerr, lerr} {
		if e != nil {
			return nil, e
		}
	}
	out.Loads = []ingest.Stats{cst, rst, lst}
	mark("ingest", t0)

	// Stage 2: the shared analysis window.
	from, to, err := resolveWindow(reqs, p.cfg.From, p.cfg.To)
	if err != nil {
		return nil, err
	}
	p.log("[2/4] Analysis window: %s to %s", from.Format("2006-01-02"), to.Format("2006-01-02"))

	// Stage 3: block -> neighborhood index. Crime and 311 both report a street
	// address and a neighborhood, so each row casts a vote for its block.
	t0 = time.Now()
	p.log("[3/4] Building block -> neighborhood index...")
	mergeVotes := func(dst, src geo.Votes) { dst.Merge(src) }
	votes := fold[model.Crime, geo.Votes](crimes, w, geo.NewVotes,
		func(v geo.Votes, c model.Crime) { v.Add(c.Block, c.Hood) }, mergeVotes)
	reqVotes := fold[model.Request, geo.Votes](reqs, w, geo.NewVotes,
		func(v geo.Votes, r model.Request) { v.Add(r.Block, r.Hood) }, mergeVotes)
	votes.Merge(reqVotes)
	ix := geo.BuildIndex(votes)
	mark("index", t0)

	// Stage 4: join + aggregate. The three datasets are folded concurrently,
	// and each fold is itself split across worker goroutines.
	t0 = time.Now()
	p.log("[4/4] Joining datasets by neighborhood and street block...")
	activeLics := activeLicenses(lics, from, to)
	mergeAcc := func(dst, src *model.Acc) { dst.Merge(src) }
	accs := make([]*model.Acc, 3)
	wg.Add(3)
	go func() {
		defer wg.Done()
		accs[0] = fold[model.Crime, *model.Acc](crimes, w, model.NewAcc, addCrime(ix, from, to), mergeAcc)
	}()
	go func() {
		defer wg.Done()
		accs[1] = fold[model.Request, *model.Acc](reqs, w, model.NewAcc, addRequest(ix, from, to), mergeAcc)
	}()
	go func() {
		defer wg.Done()
		accs[2] = fold[model.License, *model.Acc](activeLics, w, model.NewAcc, addLicense(ix), mergeAcc)
	}()
	wg.Wait()
	total := model.NewAcc()
	for _, a := range accs {
		total.Merge(a)
	}
	mark("join", t0)

	out.Data = &model.Dataset{
		From:        from,
		To:          to,
		Hoods:       total.Hoods,
		Blocks:      total.Blocks,
		Quality:     total.Q,
		IndexBlocks: ix.Len(),
	}
	// crimes, reqs and lics go out of scope here, so the garbage collector can
	// reclaim the raw rows; only the compact aggregates survive in out.Data.
	return out, nil
}

// fold splits items into one contiguous chunk per worker. Each worker folds
// its chunk into a private accumulator (no shared mutable state, so no locks),
// and the accumulators are merged on the calling goroutine (fan-in).
func fold[T, A any](items []T, workers int, newAcc func() A, add func(acc A, item T), merge func(dst, src A)) A {
	if workers < 1 {
		workers = 1
	}
	total := newAcc()
	if len(items) == 0 {
		return total
	}
	size := (len(items) + workers - 1) / workers
	out := make(chan A, workers)
	var wg sync.WaitGroup
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		wg.Add(1)
		go func(part []T) {
			defer wg.Done()
			acc := newAcc()
			for _, it := range part {
				add(acc, it)
			}
			out <- acc
		}(items[start:end])
	}
	wg.Wait()
	close(out)
	for acc := range out {
		merge(total, acc)
	}
	return total
}

// resolveWindow returns the analysis period. Unless both bounds are given, the
// window is the span of the 311 data (the shortest of the three datasets).
func resolveWindow(reqs []model.Request, from, to time.Time) (time.Time, time.Time, error) {
	if from.IsZero() || to.IsZero() {
		if len(reqs) == 0 {
			return from, to, errors.New("the 311 file has no usable rows, so the analysis window cannot be derived (use -from and -to)")
		}
		lo, hi := reqs[0].Created, reqs[0].Created
		for _, r := range reqs {
			if r.Created.Before(lo) {
				lo = r.Created
			}
			if r.Created.After(hi) {
				hi = r.Created
			}
		}
		if from.IsZero() {
			from = lo
		}
		if to.IsZero() {
			to = hi
		}
	}
	if to.Before(from) {
		return from, to, fmt.Errorf("analysis window is empty: %s is after %s",
			from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	return from, to, nil
}

// activeLicenses keeps one row per license (LLKey) that is not closed and
// overlaps the window. The raw file has one row per licensee per license year.
func activeLicenses(lics []model.License, from, to time.Time) []model.License {
	best := make(map[string]model.License)
	for _, l := range lics {
		if !l.Active || l.Start.After(to) || l.End.Before(from) {
			continue
		}
		cur, ok := best[l.Key]
		if !ok || l.End.After(cur.End) {
			best[l.Key] = l
		}
	}
	out := make([]model.License, 0, len(best))
	for _, l := range best {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func addCrime(ix *geo.Index, from, to time.Time) func(*model.Acc, model.Crime) {
	return func(a *model.Acc, c model.Crime) {
		if c.When.Before(from) || c.When.After(to) {
			return
		}
		a.Q.Crime.Rows++
		night := c.Violent && model.IsNight(c.LocalHour)
		if c.Block != "" {
			b := a.Block(c.Block)
			b.Crimes++
			if c.Violent {
				b.Violent++
			}
			if night {
				b.NightViolent++
			}
		}
		hood := c.Hood
		if hood == "" {
			hood = ix.Lookup(c.Block)
		}
		if hood == "" {
			return
		}
		a.Q.Crime.Located++
		h := a.Hood(hood)
		h.Crimes++
		if c.Shooting {
			h.Shootings++
		}
		if c.Violent {
			h.Violent++
			if c.LocalHour >= 0 {
				h.TimedViolent++
			}
			if night {
				h.NightViolent++
			}
		}
	}
}

func addRequest(ix *geo.Index, from, to time.Time) func(*model.Acc, model.Request) {
	return func(a *model.Acc, r model.Request) {
		if r.Created.Before(from) || r.Created.After(to) {
			return
		}
		if r.Prefix == "ECC" {
			// ECC rows are call-centre information requests with no field work
			// or address; they would distort response times and coverage.
			a.Q.Requests.Ignored++
			return
		}
		a.Q.Requests.Rows++
		if r.Block != "" {
			a.Block(r.Block).Requests++
		}
		hood := r.Hood
		if hood == "" {
			hood = ix.Lookup(r.Block)
		}
		if hood == "" {
			return
		}
		a.Q.Requests.Located++
		h := a.Hood(hood)
		h.Requests++
		h.ReqByType[r.Type]++
		if r.Closed.IsZero() {
			h.Open++
		}
		if r.Completed {
			h.CloseHours = append(h.CloseHours, r.Closed.Sub(r.Created).Hours())
		}
	}
}

func addLicense(ix *geo.Index) func(*model.Acc, model.License) {
	return func(a *model.Acc, l model.License) {
		a.Q.Licenses.Rows++
		if l.Block != "" {
			b := a.Block(l.Block)
			b.Kinds = append(b.Kinds, l.Kind)
		}
		hood := ix.Lookup(l.Block)
		if hood == "" {
			return
		}
		a.Q.Licenses.Located++
		h := a.Hood(hood)
		h.Licenses++
		h.LicByKind[l.Kind]++
	}
}
