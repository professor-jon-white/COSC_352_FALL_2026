# Baltimore Multi-Paradigm Data Processing Engine (Go)

A concurrent Go application that ingests two raw public datasets from the
[Baltimore City Open Data Portal](https://data.baltimorecity.gov/), joins them
by neighborhood, and answers **five research questions for municipal
stakeholders** with terminal ASCII charts, tables, and a generated Markdown
report (`report.md`).

**Datasets (Open Baltimore, raw CSV):**

1. **NIBRS Group A / Part 1 Crime Data** (columns used: `CrimeDateTime`, `Neighborhood`, `Description`, `Weapon`, `New_District`)
2. **311 Customer Service Requests** (columns used: `CreatedDate`, `CloseDate`, `SRType`, `Agency`, `Neighborhood`)

The join key is `Neighborhood`, normalized (upper-cased, whitespace collapsed) on both sides.

---

## 1. Requirements (macOS)

- Go **1.21 or newer**. Install with `brew install go` or from https://go.dev/dl/
- No third-party packages; only the Go standard library is used.

Check with:
```bash
go version
```

## 2. Get the data

1. Go to https://data.baltimorecity.gov/ and search **"NIBRS Group A Crime Data"**
   (2022 to present). The older **"Part 1 Crime Data (Legacy SRS)"** (through 2024)
   also works, because both use the same columns (`CrimeDateTime`, `Neighborhood`, ...).
   Choose **Download, then CSV**, and save it as `data/crime.csv`.
2. Search **"311 Customer Service Requests 2024"** (Open Baltimore publishes one 311
   dataset per year). Download the CSV and save it as `data/311.csv`.
   Use the **same year** for both files so the comparison is fair. For example, filter
   the crime data to 2024, or just accept that the crime file covers more years.

Column names are matched case- and punctuation-insensitively with aliases, so
minor naming differences between exports still work. If a required column cannot
be found, the program stops and prints the columns it *did* see.

> **No data downloaded?** `run.sh` automatically generates a **synthetic** sample
> (`data/sample/`) with the same column layout so the build can be verified.
> The synthetic numbers are random and are **not** real Baltimore statistics.

## 3. Compile and run

**One command (build plus run):**
```bash
chmod +x run.sh
./run.sh
```

**Manually:**
```bash
go build -o bin/baltimore-engine ./cmd/engine
./bin/baltimore-engine -crime data/crime.csv -requests data/311.csv -out report.md
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-crime` | `data/crime.csv` | Part 1 Crime CSV |
| `-requests` | `data/311.csv` | 311 requests CSV |
| `-workers` | number of CPUs | worker goroutines per dataset |
| `-top` | `10` | rows in top-N charts |
| `-out` | `report.md` | Markdown report path (`""` to skip) |

Custom file locations with the script: `CRIME=path/a.csv REQUESTS=path/b.csv ./run.sh`

## 4. Verify

```bash
go vet ./...          # static analysis
go test ./...         # unit tests (parsing, schema aliases, concurrency, stats)
go test -race ./...   # proves the concurrent code has no data races
./run.sh              # end-to-end: prints the report and writes report.md
```

Expected result: all tests `ok`, then the terminal shows the *Ingestion & Runtime
Summary* followed by sections **Q1 to Q5**, each with findings, an ASCII chart,
and a table. `report.md` contains the same report in Markdown.

---

## 5. Architecture

```
cmd/engine/        CLI entry point: wires the pipeline together
cmd/gensample/     synthetic data generator (verification only)
internal/model/    typed records (CrimeRecord, ServiceRequest)
internal/ingest/   concurrent CSV pipeline, schema resolution, time parsing, string interning
internal/analysis/ cross-dataset join (map-reduce) + the 5 research questions
internal/report/   report model (Builder) + renderers (text / Markdown)
```

Data flow:

```
crime.csv --> [reader] --> [parser pool] --+
                                           +--> BuildContext (parallel map-reduce join)
311.csv   --> [reader] --> [parser pool] --+        |
                                                    v
                                     5 questions run concurrently
                                                    |
                                       Report Builder --> Text / Markdown renderer
```

The three concerns the assignment names are kept separate: **ingestion**
(`ingest`), **cross-dataset matching logic** (`analysis.BuildContext`), and
**output formatting** (`report`).

### Design patterns

| Pattern | Where | Purpose |
| --- | --- | --- |
| **Pipeline** | `ingest.Load` | reader, then parser workers, then collector, connected by bounded channels |
| **Strategy** | `ingest.RowParser[T]` (`CrimeParser`, `ServiceRequestParser`); `analysis.Question`; `report.Renderer` | swap dataset parsers, research questions, or output formats without touching the pipeline |
| **Factory** | `analysis.NewQuestion(id)`, `report.NewRenderer(format)` | create strategies by name/id |
| **Builder** | `report.Builder` | assemble the report incrementally, independent of format |

Generics (`Load[T any]`) let one pipeline serve both datasets with full type safety.

### Multithreading and concurrency

- **Both datasets are ingested at the same time**, each in its own goroutine.
- **Within each file**, one reader goroutine owns the file (`encoding/csv` is not
  thread-safe) and sends batches of 4,096 rows to a **worker pool of goroutines**
  that parse, validate, normalize, and convert timestamps in parallel.
- **Bounded (buffered) channels** of size `2 x workers` give back-pressure: the
  reader cannot outrun the workers, so memory stays flat regardless of file size.
- **The join is a lock-free map-reduce**: each worker aggregates a disjoint slice
  of records into its *own* map (no shared writes, so no mutex contention), and
  partial results are merged afterward.
- **The five questions run concurrently**, one goroutine each. They read a
  `Context` that is immutable after construction, and each writes to its own
  slice index, so no locks are needed. Code that needs sorting (`quartiles`)
  sorts a *copy*, never the shared data.
- Counters shared across workers use `sync/atomic`; coordination uses `sync.WaitGroup`.
- `go test -race` verifies there are no data races.

### Memory and resource management

- **Streaming input:** files are read through a 1 MB `bufio` buffer in batches,
  never loaded whole into memory.
- **String interning (`ingest.Interner`):** `encoding/csv` stores each line as one
  string and returns fields as substrings of it, so keeping any field would keep
  the *whole line* alive. Every kept field is cloned once into a shared, thread-safe
  intern table. Raw lines become garbage immediately, and repeated values
  (about 280 neighborhoods, a few dozen SR types) are stored once instead of
  millions of times.
- **Pre-sized allocations:** batches and per-batch output slices are allocated
  with exact capacity to avoid repeated slice growth.
- **Compact aggregates:** resolution times are stored as `float32` (half the
  memory of `float64`, ample precision for hours); records are iterated by
  pointer (`&crimes[i]`) to avoid copying structs.
- **Explicit release:** after the join, raw record slices are set to `nil` and
  `runtime.GC()` reclaims them. The engine reports heap usage before and after
  (for example, about 40 MB after ingest down to under 1 MB after the join on
  the 230k-row sample).
- **Garbage collection:** Go uses a concurrent tri-color mark-and-sweep GC.
  The design minimizes GC pressure (interning, pre-sizing, batching) rather than
  fighting the collector, and `runtime.MemStats` is printed for transparency.
- **Time parsing:** the last successful timestamp layout is cached atomically,
  so typical rows are parsed on the first attempt instead of trying about 15 layouts.

### Robustness

- Rows without a neighborhood are counted as *dropped* (they cannot be joined).
- CSV syntax errors are counted as *malformed* and skipped, not fatal.
- Crime CSV timestamps without a zone are treated as **UTC** (the portal states the
  NIBRS export downloads in UTC) and converted to Baltimore time; 311 timestamps are
  treated as local Eastern time.
- Timestamps are parsed in several formats (ArcGIS `2024/01/02 13:00:00+00`,
  ISO 8601, US `1/2/2024 1:00 PM`, epoch ms) and converted to Baltimore local
  time (`America/New_York`, embedded with `time/tzdata`).
- Blight classification uses whole-word matching, so `OPERATIONS` does not
  match `RAT`.

---

## 6. Research questions

| # | Stakeholder | Question | Method |
| --- | --- | --- | --- |
| 1 | Mayor's Office | Where is the combined crime + 311 burden highest? | Composite index (crime and 311 each scaled 0-50); top-N and concentration share |
| 2 | Office of Neighborhood Engagement | Which high-crime neighborhoods under-use 311? | Pearson r (crime vs 311); top-quartile-crime neighborhoods ranked by 311 requests per crime |
| 3 | Police Dept. & DOT | When do crime and 311 demand peak during the day? | 24-hour histograms of both datasets; peak hours, night share, business-hours share |
| 4 | Planning / Housing & Community Dev. | Is blight associated with crime? | Neighborhoods split into blight-request quartiles; average crime per quartile; Pearson r |
| 5 | Public Works / Office of Equity | Are 311 requests resolved equitably? | Median and 90th-percentile close time by neighborhood crime quartile |

Each answer includes findings in plain language, an ASCII chart, a table, and a
recommendation. Correlations are descriptive only, not causal.
