# Baltimore Crime and 311 Analysis Engine

A Go command-line program that joins Baltimore's 2025 crime records with its
2025 311 service requests, neighborhood by neighborhood and month by month,
and answers five questions about how the two relate. It runs as a pipeline of
stages (ingest, match, aggregate, report), can run each stage sequentially or
concurrently, and checks that both modes produce identical results.

## Contents

1. [Overview](#1-overview)
2. [Requirements](#2-requirements)
3. [Getting the data](#3-getting-the-data)
4. [Build, run, and verify](#4-build-run-and-verify)
5. [Architecture](#5-architecture)
6. [Concurrency](#6-concurrency)
7. [Memory and resource management](#7-memory-and-resource-management)
8. [Research questions and findings](#8-research-questions-and-findings)
9. [Limitations](#9-limitations)

---

## 1. Overview

### What the engine does

1. **Ingests** two large CSV files from Open Baltimore, streaming them row by
   row, keeping only the columns it needs, and filtering to calendar year 2025
   in Baltimore local time. Malformed rows are counted and skipped, never fatal.
2. **Matches** the datasets on a shared key: *(neighborhood, month)*. Each key
   becomes one combined record with crime counts by type, 311 counts by request
   type, nighttime outdoor crime, and time-to-close totals for 311 requests.
3. **Aggregates** months into per-neighborhood and per-quarter views.
4. **Reports** answers to five research questions as terminal tables with ASCII
   bar charts and a one-line verdict each.

### Datasets

Both come from [Open Baltimore](https://data.baltimorecity.gov), the City of
Baltimore's open data portal.

| Dataset | Rows (as downloaded) | Covers | Used for |
|---|---|---|---|
| **NIBRS Group A Crime Data** | 278,239 | 1 Jan 2022 to present | crime counts, crime type, indoor/outdoor, time of day |
| **311 Customer Service Requests 2025** | 931,324 | calendar year 2025 | request volume, request type, status, time to close |

The join columns are `Neighborhood` (both files) plus the month of
`CrimeDateTime` (crime) or `CreatedDate` (311). Neighborhood names use the same
city neighborhood list in both files; after normalizing case and spacing, all
278 crime neighborhoods appear in the 311 data.

### Stakeholders

- **City agencies that answer 311 requests**, such as the Department of Public
  Works and the Department of Transportation: to see where demand is heaviest
  and whether some neighborhoods wait longer for service.
- **The Baltimore Police Department and public-safety planners**: to see
  whether physical-environment complaints (dumping, broken streetlights) cluster
  where crime does.
- **The Mayor's Office and City Council members**: for neighborhood-level
  prioritization and for checking whether service is equitable.
- **Neighborhood associations and residents**: to compare their neighborhood
  with others on both crime and city responsiveness.
- **Researchers and students**: as a reproducible example of joining
  administrative datasets.

---

## 2. Requirements

- **Go 1.27.2 or newer** (the version in `go.mod`).
- About **460 MB of disk** for the two CSV files and about **300 MB of free
  memory** while running.
- macOS, Linux, or Windows. The instructions below are for macOS.
- Optional, for the race detector: Xcode Command Line Tools (provides the C
  compiler that `go test -race` needs).

### Installing Go on macOS

**Option A, Homebrew** (if you have [Homebrew](https://brew.sh)):

```bash
brew install go
```

**Option B, official installer**: download the macOS `.pkg` for your Mac from
<https://go.dev/dl/> (choose **ARM64** for Apple silicon M-series Macs, **x86-64**
for Intel Macs), open it, and follow the prompts. It installs to
`/usr/local/go` and adds `/usr/local/go/bin` to your `PATH`. Open a new
Terminal window afterwards.

**Check the installation:**

```bash
go version
```

If this prints a version older than 1.27.2 but at least 1.21, that is fine: Go
reads `go.mod` and downloads the right toolchain automatically the first time
you build.

**For the race detector** (optional), install the command line tools once:

```bash
xcode-select --install
```

---

## 3. Getting the data

The CSV files are not in the repository (they are 75 MB and 380 MB, and
`/data/` is in `.gitignore`). Download them into a `data/` folder at the root of
the project with exactly these names:

| Dataset on Open Baltimore | Save as |
|---|---|
| NIBRS Group A Crime Data | `data/crime.csv` |
| 311 Customer Service Requests 2025 | `data/requests_311.csv` |

### Option A: download from the terminal (recommended)

From the project folder:

```bash
mkdir -p data
```

```bash
curl -L -o data/crime.csv "https://data.baltimorecity.gov/api/download/v1/items/204beefe92a645d79fdf0969957bbdf8/csv?layers=0"
```

```bash
curl -L -o data/requests_311.csv "https://data.baltimorecity.gov/api/download/v1/items/52c763f4496d4101ba0c20af33336a9c/csv?layers=0"
```

These are the portal's own CSV download links for the two datasets (the
32-character IDs identify each dataset). `-L` follows the portal's redirect to
the file. The 311 file takes a minute or two on a typical connection.

### Option B: download in a browser

1. Go to <https://data.baltimorecity.gov> and search for the dataset name,
   **NIBRS Group A Crime Data** (then repeat for **311 Customer Service Requests
   2025**; make sure you pick the 2025 one, as there is one per year).
2. Open the dataset and click **Download**.
3. Make sure **Toggle Filters** is off, so you get every record, then click
   **Download** under **CSV**.
4. The browser saves the files to `~/Downloads` with names like
   `NIBRS_GroupA_Crime_Data_1100291637803527672.csv` and
   `311_Customer_Service_Requests_2025_9198839567159193201.csv` (the number may
   differ). Move and rename them:

```bash
mkdir -p data
```

```bash
mv ~/Downloads/NIBRS_GroupA_Crime_Data_*.csv data/crime.csv
```

```bash
mv ~/Downloads/311_Customer_Service_Requests_2025_*.csv data/requests_311.csv
```

### Check the files

```bash
ls -lh data/
```

```bash
head -1 data/crime.csv
```

You should see two files of roughly 72 MB and 362 MB, and the crime header
should begin `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,...`. The 311
header begins `SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,...`.

> **The source data changes over time.** The crime dataset is updated weekly and
> the 311 dataset periodically, so a later download can have slightly different
> row counts than the ones in this README (which come from files downloaded in
> October 2026). Late-reported or corrected 2025 records can shift results by a
> few rows; the conclusions should not change.

---

## 4. Build, run, and verify

All commands are run from the project folder (the one containing `go.mod`).

### Build

```bash
go build -o capstone ./cmd/capstone
```

This creates an executable named `capstone` in the project folder (it is
ignored by git).

### Run

```bash
./capstone
```

This runs the default: concurrent mode, all five reports. It took about 1.5
seconds on the 12-CPU development machine; expect a similar order of magnitude
on a recent Mac.

You can also run without building:

```bash
go run ./cmd/capstone
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `-mode` | `concurrent` | `sequential`: one goroutine for everything. `concurrent`: worker pools for ingest and sharded matching. `compare`: run both, time each, and fail if any result differs. |
| `-workers` | number of CPUs | Workers per file during ingest, and number of match shards, in concurrent mode. |
| `-reports` | `all` | Which reports to print: `all`, or a comma-separated list of `sanitation`, `distress`, `close-time`, `streetlights`, `high-311-low-crime`. |
| `-memstats` | off | Print peak memory and garbage collection statistics at the end. Run one mode at a time for clean numbers. |
| `-chunk-size` | `0` (1 MiB) | Ingest chunk size in bytes for concurrent mode. |
| `-year` | `2025` | Keep only records from this year. The 311 file only covers 2025, so other years produce empty 311 data. |
| `-crime` | `data/crime.csv` | Path to the crime CSV. |
| `-requests` | `data/requests_311.csv` | Path to the 311 CSV. |

Examples:

```bash
./capstone -mode=compare
```

```bash
./capstone -mode=sequential -reports=close-time,streetlights
```

```bash
./capstone -mode=concurrent -workers=4 -memstats
```

```bash
./capstone -h
```

### What correct output looks like

The run starts with a line of timings and an ingest summary. With the data
described above, the counts are:

```
concurrent: ingest 1.152s, match 293ms

Ingest (2025)               crime        311
rows read                  278239     931324
rows kept                   54062     656827
rows skipped               224177     274497
  outside year             224160          0
  no neighborhood              17     274497
  malformed                     0          0
distinct neighborhoods        278        279

neighborhoods in both: 278

3336 neighborhood-month records across 279 neighborhoods (701 311 close times excluded as before creation)
```

Then come five reports, each with a heading, a table with `#` bar charts, and
an `Answer:` line. For example, the start of question 3:

```
Q3 [close-time] Are 311 requests closed more slowly in high-crime neighborhoods?
================================================================================

crime quartile  nbhds  crimes each  closed 311  avg days                    mix-adjusted
--------------  -----  -----------  ----------  --------                    ------------
Q1 (lowest)     70     0-60         33,880      16.3      ################  1.00x         ###############
Q2              70     63-127       96,906      14.5      ##############    1.04x         ################
Q3              70     129-243      162,166     13.3      #############     1.03x         ################
Q4 (highest)    69     243-2,024    350,812     12.7      ############      0.98x         ###############
```

Things to check:

- `rows read` matches the number of data rows in each file (`wc -l` minus 1
  for the header).
- `malformed` is 0 for both files.
- `rows read = rows kept + outside year + no neighborhood + malformed`.
- There are five `Answer:` lines.

Timings depend on the machine; counts and report values do not.

### Verify that sequential and concurrent modes agree

```bash
./capstone -mode=compare -reports=sanitation
```

The first lines should look like this (times vary):

```
Timing (concurrent: 12 ingest workers per file, 12 match shards, 12 CPUs)
  mode             ingest      match      total
  sequential       6.691s     1.241s     7.932s
  concurrent       1.152s      293ms     1.444s
  speedup            5.8x       4.2x       5.5x

Results identical: 54062 crimes, 656827 311 requests, ingest stats, 3336 neighborhood-month records
```

`Results identical` is printed only after a deep comparison
(`reflect.DeepEqual`) of every crime record, every 311 record (in order), both
sets of ingest statistics, and every neighborhood-month record. If anything
differed, the program would print which outputs differ and exit with status 1.

### Run the tests

```bash
go test ./...
```

Every package should print `ok` (packages without tests print `no test files`).
The tests do not need the downloaded data: they generate their own small CSVs
or use the samples in `testdata/` (see below).

To check the concurrent code for data races (needs the Xcode Command Line
Tools, see [Requirements](#2-requirements)):

```bash
go test -race ./...
```

The end-to-end tests in `cmd/capstone/main_test.go` run concurrent ingest and
sharded matching on a generated sample with a tiny chunk size and several
worker counts, so the race detector sees all the goroutines working at once.

### Try it without downloading the data

`testdata/` holds two small **synthetic** files with exactly the same columns
and formats as the real downloads, about 100 rows each. The values are made
up; no real records are included. They copy the real files' quirks: a UTF-8
byte-order mark, Windows (CRLF) line endings in the 311 file, crime times in
UTC, space-padded fields, rows with no location, rows outside 2025, open
requests, and close times before creation.

```bash
go run ./cmd/capstone -mode=compare -crime testdata/crime_sample.csv -requests testdata/requests_311_sample.csv
```

Expected: `Results identical: 85 crimes, 105 311 requests, ingest stats, 79
neighborhood-month records`, then five reports. Because the sample is tiny, the
answers say nothing about Baltimore, and concurrent mode is slower than
sequential (starting goroutines costs more than parsing 100 rows).
`cmd/capstone/sample_test.go` checks these exact counts in both modes.

### Continuous integration

`.github/workflows/ci.yml` runs on every push, on GitHub's `macos-latest`
runner, with the Go version from `go.mod`:

1. `go vet ./...`
2. `go build -o capstone ./cmd/capstone`
3. `go test -race ./...` (the race detector runs here, since macOS runners
   have a C compiler)
4. The program in compare mode on the `testdata/` samples, failing unless it
   prints `Results identical` and five answers.

Results appear on the repository's **Actions** tab.

---

## 5. Architecture

### Pipeline

```
 data/crime.csv ────────┐
                        ├─▶ ingest ──▶ match ──────────▶ aggregate ─────────▶ report
 data/requests_311.csv ─┘   slim        one record per    per neighborhood    5 strategies,
                            structs     neighborhood-     and per quarter     tables + bars
                                        month
```

| Package | Stage | Responsibility |
|---|---|---|
| `cmd/capstone` | wiring | Flags, opening files, running the stages, timing, compare mode, `-memstats`. |
| `internal/ingest` | ingest | Shared CSV `Scanner` (column selection by header name, malformed-row counting, string interning), row statistics, and the concurrent chunked loader. |
| `internal/ingest/crime` | ingest | Converts crime rows to `model.Crime`: UTC to local time, year filter, neighborhood normalization. |
| `internal/ingest/requests311` | ingest | Converts 311 rows to `model.ServiceRequest`, including the close date. |
| `internal/match` | match | Groups both datasets by (neighborhood, month) into `model.Summary` records; sequential and sharded concurrent versions. |
| `internal/aggregate` | aggregate | Rolls months up per neighborhood. |
| `internal/report` | report | The five report strategies, the runner, and table and bar-chart formatting. |
| `internal/stats` | helper | Spearman rank correlation, percentiles, strength labels. |
| `internal/model` | shared | The record and key types every stage uses. |

### Design patterns

**Pipeline.** Each stage takes the previous stage's output and produces a
simpler, smaller form: about 1.2 million CSV rows become 710,889 slim structs,
then 3,336 neighborhood-month records, then 279 neighborhood totals, then five
reports.

- Stages depend only on `internal/model`, never on each other, so each one can
  be tested on its own with hand-built inputs (every stage has its own tests).
- A stage can change its internals, as ingest and match did when they became
  concurrent, without the other stages noticing.
- The data shrinks early: ingest drops unused columns and out-of-year rows
  before anything is stored, so later stages work on small data.

**Strategy.** Each research question is a separate type implementing one
interface:

```go
type Report interface {
    ID() string       // name for the -reports flag
    Question() string // the question it answers
    Render(w io.Writer, in *Input) error
}
```

- `report.Run` executes any list of reports without knowing how each computes
  its answer, and `-reports` picks which ones run.
- Adding a sixth question means writing one new type and adding it to
  `report.All()`; nothing else changes.
- Each question's method lives in its own file (`q1_sanitation.go` to
  `q5_highdemand.go`), with its calculation documented next to its code.

**Supporting patterns.** Concurrent mode also uses a **worker pool** with
**fan-out/fan-in** over channels (ingest) and **sharding**, or partitioning by
key (matching). These are described in the next section.

### Key data decisions

- **Time zones.** The crime CSV stores times in UTC (the portal notes this, and
  the hour-of-day distribution confirms it), while 311 times are local. Both
  are converted to `America/New_York` before filtering by year and grouping by
  month, so a crime at 10 PM on 31 December stays in December. The time-zone
  database is embedded (`time/tzdata`), so this works on any machine.
- **Neighborhood names** are trimmed, have runs of spaces collapsed, and are
  uppercased, so ` Belair-Edison` and `BELAIR-EDISON` match.
- **Malformed rows** (bad quoting, too few fields, unparseable dates) are
  counted and skipped, with the first five reported by line number. Only I/O
  errors stop the program.

---

## 6. Concurrency

`-mode=concurrent` parallelizes the two expensive stages, ingest and match.
`-mode=sequential` runs the identical row-processing code on one goroutine, and
`-mode=compare` proves the two give identical results.

### Ingest: worker pool over channels

Both files load at the same time, each in its own goroutine, and each file has
its own worker pool:

```
main ─┬─ goroutine: crime.LoadConcurrent ──┐  joined with sync.WaitGroup
      └─ goroutine: 311.LoadConcurrent ────┘

 Per file:
   splitter ──chunks──▶ worker × N ──results──▶ collector ──▶ reassemble in order
 (1 goroutine)   ▲          │          (calling goroutine)
                 └──free────┘  (finished buffers recycled)
```

1. **Splitter goroutine.** The only reader of the file. It reads about 1 MiB at
   a time, cuts the block at the last complete record, and sends
   `chunk{index, firstLine, lines, data}` on the `chunks` channel. The partial
   record at the end is copied into the next buffer.
   - The cut point follows `encoding/csv`'s quoting rules, so a quoted field
     containing a comma or newline is never split. It finds quotes and newlines
     with `bytes.IndexByte`, which is vectorized; the first byte-by-byte version
     limited the whole speedup to 2.5×.
2. **Workers.** Each takes chunks from `chunks`, parses its chunk with its own
   `csv.Reader` and `Scanner`, converts rows with the same `scan` function the
   sequential loader uses, sends a `result` on `results`, and returns the
   buffer on the `free` channel.
3. **Closer goroutine.** Waits for all workers, then closes `results`.
4. **Collector.** The calling goroutine stores each result at
   `parts[res.index]` (results arrive in any order), then concatenates them in
   chunk order and merges statistics. Records and statistics therefore come out
   exactly as a single sequential scan would produce them, including line
   numbers in malformed-row messages.

The `chunks` channel holds only two chunks, so a fast splitter waits for
workers instead of reading the whole file into memory (backpressure). Every
path closes its channel, and the collector always reads until `results` is
closed, so a blocked send always has a receiver: the design cannot deadlock.

### Match: sharding by neighborhood

```
            crimes, requests (read-only, shared)
          ┌──────────────┼──────────────┐
      worker 0       worker 1   ...  worker N-1
   (neighborhoods  (neighborhoods    (neighborhoods
    hashing to 0)   hashing to 1)     hashing to N-1)
          │              │              │
      parts[0]       parts[1]        parts[N-1]
          └──────────────┼──────────────┘
              concatenate, sort ─▶ []Summary
```

Each worker owns the neighborhoods whose FNV-1a hash, modulo N, equals its
number. It scans both input slices, skips records outside its shard, and builds
its own map of neighborhood-month records. After `wg.Wait()`, the parts are
concatenated and sorted.

### Why there are no race conditions

A data race needs two goroutines touching the same memory at the same time
with at least one writing. The design rules that out everywhere:

- **Shared input is read-only.** All match workers read `crimes` and
  `requests`, but nothing writes to them after ingest.
- **Each goroutine owns what it writes.** Every ingest worker has its own
  scanner, CSV reader, intern maps, statistics, and result slice. Every match
  worker has its own map. No mutexes are needed because nothing mutable is
  shared.
- **Each key belongs to exactly one worker.** All months of a neighborhood hash
  to the same shard, so no two workers ever update the same record. This is the
  core idea of sharding: split the work by the grouping key instead of by rows.
- **Ownership is handed off through channels.** A chunk's bytes belong to the
  worker that received it; a result belongs to the collector once sent. Go's
  memory model guarantees the receiver sees everything written before the send.
- **Distinct slice elements.** Match worker `w` writes only `parts[w]`, and the
  ingest collector is the only writer of its `parts`.
- **`sync.WaitGroup` orders the hand-back.** Writes made by workers (and by the
  two file goroutines in `main`) are visible after `wg.Wait()` returns.

Results are identical, not just close: each match worker processes records in
the original file order, so even floating-point sums are added in the same
order as in sequential mode.

### Timing

Measured on a 12-CPU Windows laptop with `-mode=compare` (three runs; middle
run shown, range in brackets). Absolute times on a Mac will differ; the ratios
are what matter.

| Mode | Ingest | Match | Total |
|---|---|---|---|
| Sequential | 6.69 s | 1.24 s | 7.93 s |
| Concurrent (12 workers) | 1.15 s | 0.29 s | 1.44 s |
| **Speedup** | **5.8×** (5.5 to 6.7) | **4.2×** (3.3 to 4.5) | **5.5×** (5.0 to 6.3) |

### Worker scaling

Concurrent mode with different `-workers` values (average of two runs):

| Workers | 1 | 2 | 4 | 8 | 12 |
|---|---|---|---|---|---|
| Ingest | 4.66 s | 2.49 s | 1.53 s | 1.12 s | 1.19 s |
| Match | 1.24 s | 0.66 s | 0.43 s | 0.28 s | 0.30 s |

Both scale almost linearly up to 4 workers and level off around 8. Ingest with
1 worker is already faster than sequential (4.66 s vs 6.69 s) because the two
files load in parallel and the splitter runs on its own goroutine.

### Why ingest speeds up more than matching

- **Ingest is heavy, independent work per byte.** Parsing CSV and timestamps
  costs microseconds per row, and every chunk is independent, so adding workers
  divides the work almost perfectly. The only serial parts, the splitter and
  the final merge, are fast.
- **Matching is light work with overhead that grows with workers.** Each row
  costs only a hash and a map update. Every shard worker scans *all* 710,889
  records to find its own, so total scanning work grows with the number of
  workers even though the useful work does not.
- **Shards are uneven.** Neighborhoods differ a lot in size (Belair-Edison has
  23,676 311 requests; the median neighborhood has about 1,300 resident
  requests), and the hash does not balance by size, so the worker holding the
  biggest neighborhoods finishes last.
- **Map updates are limited by memory, not CPU.** Random access into hash maps
  spends much of its time waiting on memory, which more cores do not speed up
  as much.
- **The final sort is serial**, though with 3,336 records it is cheap.

---

## 7. Memory and resource management

### Choices

**Files are always closed.** Every file is opened in one helper,
`withFile`, which calls `defer f.Close()`. The file is closed on every path,
including errors and early returns.

**CSVs are streamed, never loaded whole.**

- Sequential mode reads one row at a time. `csv.Reader.ReuseRecord` reuses the
  slice of fields for each row, and the `Scanner` reuses its slice of selected
  columns. Nothing in the program calls `ReadAll` or `os.ReadFile`.
- Concurrent mode streams 1 MiB chunks through a queue that holds at most two.
  Workers return finished buffers on a `free` channel and the splitter reuses
  them, so a handful of buffers serve the whole 380 MB file. Raw bytes in memory
  stay at roughly (workers + 4) × 1 MiB per file regardless of file size.
- Chunk size trades memory for speed: 4 MiB chunks used about 300 MiB of peak
  heap, 1 MiB about 240 MiB, and 256 KiB about 238 MiB but ingested about 10%
  slower. 1 MiB is the default; `-chunk-size` changes it.

**Kept strings do not hold on to whole CSV lines.** `encoding/csv` returns
every field of a row as a substring of one string holding the entire line.
Keeping even a short field such as `"Closed"` kept the whole ~400-byte line
alive: about 280 MB across 711,000 rows. The `Scanner` now *interns* kept
values: `Scanner.Intern` copies each distinct value once with `strings.Clone`
and hands out that copy for every later row. There are only about 305 request
types, 12 statuses, and 279 neighborhoods. `Scanner.Neighborhood` also caches
normalization, so each raw spelling is uppercased once rather than per row.
Each scanner has its own intern maps, so workers share nothing.

**Structs keep only the fields we use.** Of 24 crime columns and 28 311
columns, the program stores four and five:

| Struct | Fields | Size |
|---|---|---|
| `model.Crime` | time, neighborhood, crime type, outdoor flag | 64 bytes |
| `model.ServiceRequest` | created time, neighborhood, request type, status, closed time | 96 bytes |

A test (`internal/model/model_test.go`) pins these sizes, so adding a field has
to be a deliberate change. The combined neighborhood-month records keep counts
and totals, not lists of rows.

**Slices and maps are pre-sized where the size is known.**

- In concurrent ingest, the splitter counts each chunk's lines, an upper bound
  on its rows, and each worker sizes its result slice to that.
- The merged result slice is sized exactly from the sum of the chunks, so it is
  allocated once.
- Match's merged output, its per-worker output, report tables, header column
  maps, and statistics merging are sized exactly or by a known upper bound.
- Sequential ingest is **not** pre-sized, because the number of kept rows isn't
  known until the file is read (only 19% of crime rows are from 2025); `append`
  doubles capacity as needed. Match's grouping map is not pre-sized for the
  same reason.

**Merged results are released as they are copied.** After each chunk's records
are copied into the final slice, `parts[i].records = nil` lets the garbage
collector reclaim it instead of keeping every chunk alive until the function
returns.

### Measurements

Measured with `-memstats`, each mode in its own process (three runs; typical
values). "Before" is the code before the memory work; "after" is the current
code.

| Mode | Peak heap | Peak process memory | Total allocated | Heap objects | GC cycles | GC pause total | GC CPU |
|---|---|---|---|---|---|---|---|
| Sequential, before | 518 MiB | 549 MiB | 893 MiB | 4.3 M | 20 | 2 to 7 ms | 0.3% |
| **Sequential, after** | **245 MiB** | **270 MiB** | 858 MiB | **2.5 M** | 45 | 1 to 11 ms | 1.0% |
| Concurrent, before | 670 MiB | 709 MiB | 1,302 MiB | 4.3 M | 10 | 11 to 21 ms | 4.7% |
| **Concurrent, after** | **234 MiB** | **275 MiB** | **718 MiB** | **3.0 M** | 14 | 18 to 28 ms | 6.3% |

- **Peak heap** is sampled every 5 ms from `runtime/metrics`, because Go does
  not record a peak itself.
- **Peak process memory** is the operating system's figure: peak working set
  on Windows, maximum resident set size (`getrusage`) on macOS and Linux.
- **GC figures** come from `runtime.ReadMemStats`.

To reproduce:

```bash
./capstone -mode=sequential -memstats -reports=sanitation
```

```bash
./capstone -mode=concurrent -memstats -reports=sanitation
```

### Reading the GC numbers

By default (`GOGC=100`), Go lets the heap grow to about twice the live data
before collecting. That is why peak heap (about 240 MiB) is more than twice the
roughly 70 MiB of rows the program keeps.

After the fixes, sequential mode runs *more* GC cycles (20 to 45): the live
heap is smaller, so the collector starts sooner, but each cycle has less to
scan and GC still uses about 1% of CPU. Concurrent mode shows a higher GC CPU
share only because the same allocation happens in about a sixth of the time.

`GOGC` trades memory for GC work:

| `GOGC` | Sequential peak heap | Sequential GC cycles | Concurrent peak heap | Concurrent GC CPU |
|---|---|---|---|---|
| 50 | 167 MiB | 87 | 202 MiB | 9.8% |
| 100 (default) | about 230 MiB | about 45 | about 240 MiB | about 6% |
| 200 | 213 MiB | 28 | 258 MiB | 4.7% |

For example: `GOGC=50 ./capstone -memstats`. `GOMEMLIMIT` sets a hard ceiling
instead.

---

## 8. Research questions and findings

Shared definitions:

- **Resident 311 volume** excludes city-initiated request types whose name
  contains "Proactive" (for example `SW-Rat Rubout Proactive`, the largest
  single type with 117,723 requests). These reflect city crews' sweeps, not what
  residents report.
- **Correlations** are Spearman rank correlations across the 279 neighborhoods.
  They measure whether one value tends to rise with the other without assuming
  a straight line, so a few very large neighborhoods cannot dominate.
- **Size matters.** Bigger, busier neighborhoods have more of everything, so
  each report includes a baseline or a rate that controls for size.

### Q1. Do neighborhoods with the most sanitation/dumping 311 requests also have the highest crime? (`sanitation`)

**Method.** Sanitation requests are a fixed list of resident-reported types:
illegal dumping, sanitation property, dirty alley, dirty street, cleaning, and
rat complaints (scheduled pickups and proactive work excluded). The report
lists the top 10 sanitation neighborhoods with their crime rank and compares
the sanitation-crime correlation with a baseline: all resident 311 requests
versus crime.

**Finding: yes, but it is not specific to sanitation.** All 10 of the top
sanitation neighborhoods are in the top quarter for crime, and the rank
correlation is strong (+0.78). But total resident 311 volume tracks crime even
more closely (+0.87), so the pattern mostly reflects busy neighborhoods having
more of both, not dumping in particular.

### Q2. Top 10 neighborhoods by combined distress, by quarter (`distress`)

**Method.** Crime and 311 counts are on very different scales (about ten
resident requests per crime), so adding them would just rank 311. Each count is
divided by that quarter's average across all neighborhoods:

`index = (crimes / average crimes + 311 / average 311) / 2`

An index of 1.0 is a typical neighborhood; 10 means ten times the typical level.

**Finding.** Downtown ranks first every quarter (index 8.3 to 10.1). Seven
neighborhoods are in the top 10 all four quarters: Belair-Edison, Brooklyn,
Canton, Downtown, Frankford, Sandtown-Winchester, and Washington
Village/Pigtown.

### Q3. Are 311 requests closed more slowly in high-crime neighborhoods? (`close-time`)

**Method.** Neighborhoods are split into four equal groups by crime. Raw
average days to close is misleading because request types close at very
different speeds and neighborhoods file different mixes, so the report also
computes a **mix-adjusted ratio**: actual total days divided by the days the
same requests would take at citywide per-type averages. 1.0 means "as fast as
the city handles the same kinds of requests."

**Finding: no.** Raw averages suggest high-crime neighborhoods are served
*faster* (12.7 days vs 16.3 days), but only because they file more
quick-to-close request types. Comparing like with like, the highest-crime
quarter is within 2% of the lowest (0.98× vs 1.00×), and the adjusted rank
correlation with crime is +0.01.

### Q4. Do neighborhoods with many streetlight requests see more nighttime outdoor crime? (`streetlights`)

**Method.** Streetlight requests are types containing "StLight" or "Street
Light" (red-light cameras do not match). Night outdoor crime means
`Inside_Outside = O` between 8 PM and 6 AM local time, excluding times of
exactly midnight (00:00 appears far more often than any other minute, which
suggests it often means "time unknown"). Because raw counts mostly measure
size, the report also checks the *share* of each neighborhood's crime that is
outdoors at night.

**Finding: somewhat.** Neighborhoods with more streetlight requests have more
night outdoor crime (correlation +0.59), but that is exactly the same as their
correlation with all crime (+0.59). The night outdoor share rises only slightly,
from 13.9% in the lowest streetlight quartile to 16.7% in the highest (share
correlation +0.11, weak).

### Q5. Which neighborhoods have high 311 volume but low crime? (`high-311-low-crime`)

**Method.** The strict test is resident 311 volume at or above the 75th
percentile and crime at or below the median. Because 311 and crime rise
together so strongly, few neighborhoods pass, so the report also ranks busy
neighborhoods (at least median 311 volume) by resident requests per crime.

**Finding.** Only **Mount Washington** and **Roland Park** pass the strict
test. By requests per crime, Mount Washington leads with 28 resident requests
per crime against 10 citywide, followed by Perring Loch, Idlewood, and Roland
Park, where parking complaints are the most common request.

---

## 9. Limitations

**Rows dropped for missing neighborhoods.** 274,497 of 931,324 311 requests
(**29.5%**) have no neighborhood and are dropped. Nearly all of them (259,577)
have no location at all, such as information calls to the city's call center
that are not tied to an address. For crime, only 17 of 54,079 rows from 2025 (0.03%) lack a
neighborhood. Results therefore describe the 70.5% of 311 requests that can be
placed in a neighborhood.

**2025 only.** The 311 file covers only calendar year 2025. In addition, the
Baltimore Police Department changed how it collects and reports crime data
effective 1 January 2025, moving to the FBI's National Incident-Based
Reporting System (NIBRS), according to the dataset's description on Open
Baltimore. The crime dataset includes records back to 2022, but those predate
the change in collection, so comparing them with 2025 risks measuring the
reporting change rather than a real change in crime. The engine keeps only 2025
from both files (`-year` defaults to 2025).

**Gap in the 311 data in April and May 2025.** Those months have about 10,700
and 23,500 requests, against about 60,000 in a typical month, which suggests
missing records rather than a quiet period. Q2's index compares neighborhoods
within each quarter, so its rankings still hold, and the Q2 report flags the
quarter. Totals for April to June undercount 311 activity.

**Crime times.** Crime timestamps are downloaded in UTC and converted to local
time. Exactly-midnight timestamps are excluded from "night" because they appear
to be placeholders for unknown times, which may undercount some genuine
midnight crimes.

**Close times.** 701 closed requests have a close time well before their
creation time and are excluded from time-to-close figures. Close times up to an
hour before creation are treated as zero days, because many close times are
rounded down to the minute. Average days to close is skewed by a few very slow
requests (median about 1.5 hours, mean about 9.7 days), so the mix-adjusted
ratio is the more reliable measure.

**Category definitions are judgment calls.** Which request types count as
sanitation or streetlights, and the exclusion of "Proactive" types, are
explicit lists in `internal/report/categories.go` and can be changed.

**No population or area adjustment.** The datasets do not include neighborhood
population or size, so raw counts partly measure how big a neighborhood is.
The reports use baselines and rates to reduce this, but per-capita figures
would need another dataset.

**Correlation, not causation.** All five questions describe patterns across
neighborhoods. They do not show that one thing causes another, and patterns
across neighborhoods may not hold for individual streets or people.

**Reported data only.** Both datasets record what was reported. Crimes not
reported to police, and problems residents do not call 311 about, are not
included, and willingness to report may differ by neighborhood.

**Source data changes.** Both datasets are updated over time, so a fresh
download may give slightly different counts than this README.

**Performance numbers are from one machine.** Timings were measured on a
12-CPU Windows laptop and varied between runs (up to roughly 1.5× between
consecutive runs); a Mac will give different absolute times. The race detector
was not available on the Windows development machine (no C compiler); it runs
in continuous integration on macOS, and locally on a Mac with the Xcode Command
Line Tools via `go test -race ./...`.
