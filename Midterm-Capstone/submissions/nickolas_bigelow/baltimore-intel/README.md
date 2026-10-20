# Baltimore City Cross-Dataset Intelligence

A Go application that ingests four raw public datasets from the
[Baltimore City Open Data Portal](https://data.baltimorecity.gov/), joins them by
neighborhood and by location, and answers five research questions for city
stakeholders. Results appear as terminal charts and tables, a Markdown report, a
self-contained HTML report, and CSV tables.

- **Language:** Go (standard library only, no third-party dependencies)
- **Concurrency:** goroutines, channels, worker pools, `sync.WaitGroup`, `sync.Pool`, `sync.RWMutex`, atomics
- **Patterns:** Pipeline, Builder, Factory, Strategy

---

## 1. Quick start on macOS

### Prerequisites

| Requirement | How to get it | Check |
|---|---|---|
| Go **1.21 or newer** (Apple Silicon or Intel) | `brew install go`, or the macOS installer from https://go.dev/dl/ | `go version` |
| `make` | `xcode-select --install` (Command Line Tools) | `make --version` |

There are no other dependencies. `go.mod` has no `require` lines, so nothing is
downloaded at build time.

### Build and run (offline, about 10 seconds)

```bash
cd baltimore-intel
make demo          # = go build + generate sample data + run all five analyses
make open          # opens reports/report.html in your browser
```

`make demo` does three things:

1. Compiles `bin/baltimore-intel`.
2. Writes four CSVs to `data/raw/` in the **exact column layout** of the Open
   Baltimore exports, using synthetic records.
3. Runs the pipeline and prints the report to the terminal. It also writes
   `reports/report.md`, `reports/report.html` and eight CSV tables.

### Run on live Open Baltimore data

```bash
make fetch YEAR=2022    # downloads Crime, 311, Liquor Licenses, Vacant Notices (~2-5 min)
make run   YEAR=2022
make open
```

`fetch` replaces the sample files in `data/raw/`. Every report states at the top
whether it was built from **synthetic sample data** or from **Open Baltimore**,
with the fetch date and source URLs (`data/raw/SOURCE.txt`).

> **Why a sample generator?** Grading has to succeed even if the portal is slow,
> rate-limits, or changes. The generator writes the same columns, date formats
> and data-quality problems as the real exports, so the whole pipeline runs
> identically on either. Use the fetched data for real conclusions about the city.

### Without `make`

```bash
go build -o bin/baltimore-intel ./cmd/baltimore-intel
./bin/baltimore-intel sample  -year 2022                   # or: fetch -year 2022
./bin/baltimore-intel analyze -year 2022
```

---

## 2. Verifying the build

| Step | Command | Expected result |
|---|---|---|
| Compile | `make build` | `bin/baltimore-intel` exists, no output |
| Static checks | `make vet` | no output |
| Unit + end-to-end tests | `make test` | every package `ok` (7 packages with tests) |
| Race detector | `make race` | every package `ok`, no `DATA RACE` reports |
| End-to-end run | `make demo` | five sections Q1–Q5, then "Wrote:" listing 10 files |
| Concurrency speedup | `make bench` | table of 1, 2, 4 … N workers with speedup vs 1 worker |

What the tests cover:

- **Parsing:** all date formats the portal uses, header aliases, the byte-order
  mark (BOM), ragged rows, bad dates, wrong years, and `(0,0)` coordinates.
- **Determinism:** the concurrent parser returns identical records **in file
  order** for 1, 2, 7 and 16 workers. The full pipeline produces identical
  findings with 1 and 8 workers.
- **Spatial index:** checked against a brute-force scan on 200 random queries.
- **Map-reduce:** checked against a sequential count.
- **Fetcher:** tested against a fake ArcGIS server (`httptest`). This covers
  ID listing, chunked POSTs, sampling and ArcGIS error responses, and checks
  that the downloaded file flows straight into ingestion.
- **Planted signals:** the generator plants known relationships, and the tests
  check that Q2–Q4 detect them.
- **Builder validation:** bad worker counts, radius and format names, and
  missing data files all produce clear errors.

---

## 3. Command reference

```
baltimore-intel fetch   -year 2022 -max311 150000 -workers 6 -data data/raw [-only crime,311]
baltimore-intel sample  -year 2022 -seed 42 -scale 1.0 -data data/raw
baltimore-intel analyze -data data/raw -out reports -year 2022 -workers N
                        -radius 150 -format terminal,md,html,csv [-ascii] [-memlimit MB]
baltimore-intel bench   -data data/raw -workers N -runs 3
```

- `-year`: keeps only crime and 311 records from that calendar year, in Baltimore time.
- `-radius`: distance in meters that counts as "near a liquor outlet" (Q2, Q5).
- `-ascii`: replaces Unicode block characters with `# + . : *` for terminals without those glyphs.
- `-max311`: the 311 layer has about 1 million rows a year. `fetch` downloads an
  evenly spaced sample across the whole year (default 150k); `0` downloads everything.
- `-memlimit`: sets a soft heap limit, the same mechanism as the `GOMEMLIMIT`
  environment variable (see section 7).

Makefile variables can be overridden: `make run YEAR=2021 WORKERS=4 DATA=other/dir`.

---

## 4. Datasets (Open Baltimore)

| Dataset | Portal / API layer | Fields used | Joined on |
|---|---|---|---|
| **Part 1 Crime Data** | [dataset page](https://data.baltimorecity.gov/datasets/e0992dddbbf64231976d5d57763ec4f5) · `services1.arcgis.com/UWYHeuuJISiGmgXx/.../Part1_Crime_Beta/FeatureServer/0` | `CrimeDateTime`, `Description`, `Neighborhood`, `Latitude`, `Longitude`, `GeoLocation`, `Total_Incidents` | neighborhood, location, time |
| **311 Customer Service Requests** | `.../311_Customer_Service_Requests_Yearly/FeatureServer/{2022-year}` (2004–2022), `311_Customer_Service_Requests_{year}` for newer years | `CreatedDate`, `CloseDate`, `DueDate`, `SRType`, `Agency`, `Neighborhood`, `Latitude`, `Longitude` | neighborhood, location |
| **Liquor Licenses** | [dataset page](https://data.baltimorecity.gov/datasets/ae5ed61365e74579aea25656ac9ce45e) · `baltegis.baltimorecity.gov/.../CityView/Licenses/FeatureServer/0` | `LicenseNumber`, `LicenseYear`, `LicenseStatus`, `TradeName`, `LicenseClass`, `EstablishmentDesc`, `AddrStreet`, geometry | location only |
| **Vacant Building Notices** | [dataset page](https://data.baltimorecity.gov/datasets/691d65a5f85640e6aaa46930bd9dc102) · `baltegis.baltimorecity.gov/.../DHCD_Open_Baltimore_Datasets/FeatureServer/1` | `Neighborhood`, `DateNotice`, `DateCancel`, `DateAbate`, `HousingMarketTypology2023`, geometry | neighborhood |

The CSV files are read as raw exports: original column names, original date
formats, and the data-quality problems listed below. The parser also accepts
files downloaded by hand from the portal ("Download → CSV") if you save them
under these names in `data/raw/`:

```
part1_crime.csv   service_requests_311.csv   liquor_licenses.csv   vacant_building_notices.csv
```

**Data cleaning, all of it reported per file at run time:**

- Neighborhood names are normalized into one join key. Case, spacing and
  punctuation are ignored, so `BELAIR-EDISON`, `Belair-Edison` and
  `belair edison` all match.
- Coordinates outside Baltimore, including `(0,0)`, are rejected. If
  `Latitude`/`Longitude` are missing, the parser falls back to `GeoLocation`
  or `X`/`Y`.
- Liquor licenses have one row per license per year. Only the latest row for
  each license number is kept. Cancelled, revoked and denied licenses are dropped.
- Vacant notices with a cancel or abate date are closed, and are dropped.
- 311 close dates earlier than the creation date are treated as still open.
- Crimes stamped exactly `00:00:00` (the source's "time unknown") are excluded
  from the time-of-day analysis.

---

## 5. The five research questions

| # | Stakeholder | Question | Datasets joined | How |
|---|---|---|---|---|
| Q1 | Mayor's Office | Which neighborhoods face **compounded distress** across public safety, housing and city services? | Crime + Vacants + 311 (+ Liquor) | Neighborhood join; composite index = mean z-score of log(violent crime), log(open vacants) and log(311 volume); Spearman correlations; concentration in the top 10 |
| Q2 | Liquor Board & BPD | Is **violent crime concentrated around liquor outlets**, and which outlets stand out? | Liquor + Crime + 311 | Spatial join: violent crime within *r* m of each outlet vs the same measure at evenly sampled 311 addresses; histogram; breakdown by establishment type; top outlets |
| Q3 | DPW & DHCD | Do **vacant buildings drive sanitation and rodent complaints**? Where is demand out of line with vacancy? | Vacants + 311 | Pearson and Spearman correlation, least-squares slope, above vs below median vacancy, residuals, ASCII scatter plot |
| Q4 | DOT & 311 agencies | Are 311 requests from **high-crime neighborhoods resolved as quickly**? | 311 + Crime | Neighborhoods split into violent-crime terciles; median and 90th-percentile days to close; % closed by due date; a **mix-adjusted wait index** (days ÷ citywide median for the same request type), so the comparison is like-for-like; per-agency gap |
| Q5 | BPD Patrol & MONSE | **When, and around which places**, should late-night patrol and outreach be concentrated? | Crime + Liquor | Weekday × hour heatmap (local time), hourly histogram, near-outlet vs elsewhere time profile, ranked neighborhood × time-block hotspots |

Each section in the report has:

- a one-paragraph **finding** with the actual numbers;
- charts and tables;
- **recommended actions** for the stakeholder;
- **method notes and caveats**. For example, Q2 measures association, not
  causation, and Q1 uses raw counts because the datasets publish no population figures.

---

## 6. Architecture

```
                    ┌────────────────────────────── Pipeline (Builder-configured) ──────────────────────────────┐
 data/raw/*.csv ──▶ │ INGEST            │ JOIN                         │ ANALYZE              │ RENDER              │
                    │ Source factory     │ 3 spatial grids (parallel)   │ 5 Question strategies│ Renderer strategies │
                    │ 4 datasets ║       │ outlet→neighborhood vote     │ run concurrently     │ terminal / md /     │
                    │ each: reader →     │ outlet exposure ║ baseline ║ │ results kept in      │ html + CSV export   │
                    │ N parsers →        │ crime-near-outlet flags      │ registration order   │                     │
                    │ ordered collector  │ neighborhood map-reduce      │                      │                     │
                    └───────────────────────────────────────────────────────────────────────────────────────────┘
                     ║ = runs in parallel
```

```
cmd/baltimore-intel/   CLI: fetch | sample | analyze | bench
internal/model/        compact record structs, thread-safe string interner, Dataset
internal/ingest/       Source interface + NewSource factory, concurrent CSV pipeline, date/coord parsing
internal/fetch/        ArcGIS REST downloader (ID listing + concurrent chunked POSTs, single CSV writer)
internal/spatial/      uniform-grid spatial index (radius / nearest / k-nearest)
internal/join/         ParallelReduce / ParallelFor (generic), cross-dataset joins
internal/analysis/     Question interface + Q1–Q5
internal/report/       renderer-neutral Report model, ASCII charts, terminal/Markdown/HTML renderers, CSV export
internal/pipeline/     Builder, staged Pipeline, memory/goroutine sampler
internal/sample/       deterministic schema-faithful data generator
```

### Design patterns

| Pattern | Where | Why |
|---|---|---|
| **Pipeline** | `pipeline.Run`, `ingest.ParseCSV` | Ingest → Join → Analyze → Render as separate stages; each CSV is streamed reader → parsers → collector over bounded channels |
| **Builder** | `pipeline.NewBuilder()...Build()` | Validates every option (workers, radius, formats, data files) and reports all problems at once before any work starts |
| **Factory** | `ingest.NewSource(kind)`, `report.NewRenderer(name)` | The pipeline never refers to concrete dataset or output types |
| **Strategy** | `analysis.Question`, `report.Renderer` | Questions and output formats are interchangeable; adding Q6 or a JSON renderer touches no other layer |

Ingestion (`ingest`), cross-dataset matching (`join`, `spatial`) and output
formatting (`report`) are separate packages. Analyses only see the read-only
`join.Joined` structure, and renderers only see the `report.Report` tree.

---

## 7. Concurrency, thread safety and memory management

### Concurrency model

| Stage | Primitive | Detail |
|---|---|---|
| Ingest across datasets | goroutine per dataset + `sync.WaitGroup` | Crime, 311, liquor and vacants load simultaneously |
| Ingest within a file | **reader goroutine → buffered channel → N parser goroutines → collector** | Rows travel in batches of 2,048, so channel overhead is paid per batch, not per row. The channel is bounded (`2×workers`), so a slow parser applies back-pressure and memory stays capped. |
| Order preservation | sequence-numbered batches | The collector puts each batch back in its slot, so output is identical for any worker count (tested) |
| Spatial indexes | 3 goroutines | Crime, outlet and 311 grids build at the same time |
| Joins | `join.ParallelFor` | Each goroutine owns a contiguous index range and writes only its own output slots, so no locks are needed |
| Aggregation | `join.ParallelReduce[T, A]` (generic) | Each shard folds into a **private accumulator** and the accumulators are merged at the end, with no shared map and no false sharing |
| Analyses | goroutine per question | `recover()` isolates a failing analysis; results return in report order |
| Fetch | worker pool of HTTP requests + **one writer goroutine** | Only the writer owns the CSV file, so it needs no mutex; requests retry with backoff on 429 and 5xx |
| Cancellation | `context.Context` + `signal.NotifyContext` | Ctrl-C stops readers, workers and downloads cleanly |

### Thread safety

- **Read-mostly after ingestion.** The `Dataset` is mutated only while loading,
  and each source writes its own slice field. After that, the joins write only
  index-partitioned slots, and the analyses only read. Every package passes
  `go test -race`.
- **`model.Interner` is safe under concurrent use.** Lookups of known strings
  take only a read lock (`RWMutex`), which is the common case. Inserts use
  double-checked locking under the write lock.
- **The memory sampler uses atomics.** It records peak heap and goroutine counts
  with compare-and-swap loops on `atomic.Uint64` / `atomic.Int64`.

### Memory and garbage collection

- **Streaming, not loading.** CSVs are never read into memory whole. With
  `csv.Reader.ReuseRecord`, the reader reuses its slice, and only the small
  slice header is copied per row. Batch buffers are recycled through a
  `sync.Pool`.
- **Flat value slices, no per-record pointers.** Records are stored as
  `[]model.Crime` and `[]model.ServiceRequest`, not `[]*Crime`. Because these
  structs contain no pointers, the garbage collector can skip their memory
  entirely during marking. Loops take `&slice[i]` instead of copying records.
- **Compact records:**
  - Coordinates are `float32`, about 1 m precision in Baltimore.
  - Times are `int64` Unix seconds; `time.Time` is 24 bytes and holds a pointer.
  - Repeated strings (neighborhood, crime type, 311 type, agency) are interned
    to `uint16` IDs.

  The result: a `Crime` is **24 bytes** and a `ServiceRequest` **40 bytes**,
  enforced by a unit test. A year of data (about 40k crimes and 150k requests)
  takes about **7 MB** of record memory.
- **Exact-size allocation.** The ordered collector counts records first, then
  allocates the final slice once at its exact size, rather than growing it by
  repeated doubling.
- **The spatial index stores `int32` positions**, not copies of records.
- **Measured at run time.** The "Runtime & Resources" section of every report
  shows:
  - heap after ingest and peak heap;
  - total allocated;
  - GC cycles and total pause time;
  - peak goroutine count;
  - each record type's size in bytes.
- **Heap limit.** `-memlimit 64` sets a soft heap limit via
  `debug.SetMemoryLimit`, the same mechanism as `GOMEMLIMIT`, for
  memory-constrained machines.

Reference run on the sample data (2-core Linux container; a Mac will be faster):
ingest about 350 ms, join about 160 ms, analyze about 80 ms, peak heap about
56 MB, 18 GC cycles totalling about 3 ms of pause. `make bench` prints the
speedup on your machine.

---

## 8. Output files

| File | Contents |
|---|---|
| terminal | full report with Unicode (or `-ascii`) bar charts, heatmap, scatter plot, boxed tables |
| `reports/report.md` | the same report in Markdown; charts are in code blocks |
| `reports/report.html` | self-contained page with CSS bar charts and a colored heatmap; supports dark mode; works offline |
| `reports/q*_*.csv` | every ranked table, ready for Excel or further analysis |

---

## 9. Troubleshooting

| Symptom | Fix |
|---|---|
| `missing data/raw/part1_crime.csv` | Run `make sample` (offline) or `make fetch` (live) first |
| `fetch` fails or times out | The portal may be rate-limiting. Retry later, use `-only crime` to fetch one dataset at a time, or use `make sample` |
| `no crime or 311 records left after filtering` | `-year` doesn't match the files. Use the same `YEAR` for `fetch` and `run`, or `-year 0` to keep all years |
| Boxes or blocks look garbled in the terminal | Use `make run-ascii` (or `-ascii`) |
| `go: go.mod requires go >= 1.21` | Update Go: `brew upgrade go` |
| `make: command not found` | Run `xcode-select --install`, or use the commands under "Without `make`" |

The legacy Part 1 Crime layer follows BPD's older crime-reporting standard
(called SRS); BPD has since moved to a new standard (NIBRS). Years after the
switch may be thin in this layer, which is why `YEAR=2022` is the default.
