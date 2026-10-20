# Baltimore Cross-Dataset Intelligence (Go)

A Go program that ingests three public datasets from the
[Open Baltimore portal](https://data.baltimorecity.gov/), joins them, and
answers five research questions for municipal stakeholders. It prints a
terminal report with ASCII charts and writes the same report as an HTML page.

| Dataset | Used for |
|---|---|
| NIBRS Group A Crime Data | violent crime, time of day, location |
| 311 Customer Service Requests | neighborhood service demand and response times |
| Liquor Licenses | licensed establishments by street address |

## Requirements

* macOS (the code is portable and has no OS-specific parts)
* Go **1.21 or newer**: `brew install go`, or the installer from https://go.dev/dl/
* No third-party packages and no network access are needed to build or run.

Check your install with `go version`.

## 1. Data

The three CSV files this project was built and tested with are expected in the
`data/` folder:

* `NIBRS_GroupA_Crime_Data_1100291637803527672.csv` (NIBRS Group A Crime Data)
* `311_Customer_Service_Requests_2026.csv` (311 Customer Service Requests)
* `Licenses.csv` (Liquor Licenses)

**If the files are already in `data/`** (they ship with the submission), there
is nothing to do. If they arrive zipped, unzip them first so that `data/`
contains plain `.csv` files.

**If `data/` is empty**, download each dataset from
https://data.baltimorecity.gov/ (open the dataset, then **Export > CSV**) and
put the three files in `data/`. File names do not matter as long as they
contain `NIBRS`, `311`, and `icense` respectively; the program finds them by
those patterns. Keep exactly one file per dataset in `data/`, and remove any
extra copies or samples, or the program will stop and list the matches.
To use other locations, pass `-crime`, `-requests`, `-licenses` (see Flags).

## 2. Build and run

From the project folder (open Terminal, then `cd` into it):

```bash
go run .
```

This compiles and runs the program in one step. It was tested on macOS
(Intel) and Windows, and produced identical results on both. `go run` does not
leave an executable behind. To build one instead:

```bash
go build -o bin/baltimore-intel .
./bin/baltimore-intel
```

`./run.sh` does the same build-and-run in one command. If it reports
"permission denied" after unzipping, run `bash run.sh` or use `go run .`.

Progress messages go to stderr, the report goes to stdout, and the HTML copy
is written to `output/report.html` (open it in any browser). The report footer
prints stage timings, heap use and GC cycles for the run.

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `-data` | `data` | folder holding the three CSVs |
| `-crime`, `-requests`, `-licenses` | auto-detected | explicit file paths |
| `-out` | `output` | folder for `report.html` |
| `-workers` | number of CPUs | worker goroutines per stage |
| `-from`, `-to` | span of the 311 file | analysis window, `YYYY-MM-DD` |

## 3. Verify

```bash
go vet ./...          # static checks
go test ./...         # unit and end-to-end tests
go test -race ./...   # optional: same tests with Go's data-race detector
```

`go vet` and `go test` should finish without errors (`ok` for each package).
The `-race` run is optional: it needs a C compiler (on macOS, the Xcode command
line tools: `xcode-select --install`) and is not available on a stock Windows
install. The tests use
small slices of the real exports stored in `internal/pipeline/testdata/`:

* address normalisation, neighborhood index, statistics, CSV parsing;
* an end-to-end run that checks exact join coverage counts;
* a check that results are identical with 1 worker and with 8 workers, which
  together with `-race` demonstrates that the concurrent stages are correct;
* both report formats render all five questions.

To verify a full run, check the "Datasets and join coverage" table at the top
of the report (rows read, rows parsed, rows placed in a neighborhood) and that
all five questions print a finding, a chart, and tables.

## The five research questions

| # | Question | Datasets | Stakeholder |
|---|---|---|---|
| 1 | Which 311 request types track violent crime across neighborhoods, after adjusting for each neighborhood's overall 311 volume? | 311 + crime | Public Works / DOT |
| 2 | Do neighborhoods with more liquor licenses see more night-time violent crime? | licenses + crime | Liquor Board / Police |
| 3 | Do high-crime neighborhoods wait longer for 311 service? | 311 + crime | Mayor's Office (equity) |
| 4 | Are street blocks with a liquor license more violent than blocks without one, and which license types stand out? | all three, joined by block | City Planning / Liquor Board |
| 5 | Which neighborhoods need combined intervention first? (weighted priority index) | all three | Mayor's Office |

## How the datasets are joined

The datasets share no ID, and the license file has no neighborhood or
coordinates, only a street address. So every address in every dataset is
normalised to a **street block**: house number rounded down to the hundred, with
street types and directions standardised
(`4410 NORTHERN PARKWAY WEST` and `4400 W NORTHERN PKWY` become the same key;
`1724-26 CHARLES STREET NORTH` becomes `1700 N CHARLES ST`).

Crime and 311 rows carry a neighborhood, so each one votes for "this block is in
neighborhood X". The majority vote builds a block-to-neighborhood index, which
places licenses (and any crime or 311 row with a blank neighborhood) into a
neighborhood. Question 4 joins directly at block level. The "join coverage"
table in the report shows how many rows could be placed.

## Architecture

```
main.go                      flags, wiring, output
internal/
  model/      shared types and the per-worker accumulator
  geo/        address normalisation, block -> neighborhood index
  ingest/     streaming CSV reader + worker pool, one parser per dataset
  pipeline/   Builder, stages (ingest, window, index, join/aggregate)
  analysis/   one Strategy per research question + RunAll
  report/     renderer Factory: terminal (ASCII charts) and HTML
  stats/      Spearman/Pearson, median, percentile ranks
```

Design patterns:

* **Pipeline**: ingest, window, index, join/aggregate, analyse, report. Each
  stage only depends on the output of the previous one.
* **Builder**: `pipeline.NewBuilder().Crime(...).Requests(...).Licenses(...).Workers(n).Build()`.
* **Strategy**: every research question implements `analysis.Analysis`, so
  questions can be added or removed without touching ingestion or output.
* **Factory**: `report.New("terminal" | "html")` returns a `Renderer`.

## Concurrency

* **Ingest**: three files load concurrently. Within a file, one goroutine reads
  the CSV in batches of 2,048 records and sends them over a buffered channel to
  N worker goroutines that parse them. Workers fan their results back in over
  a second channel.
* **Index and join**: the parsed rows are split into one chunk per worker. Each
  worker aggregates into its **own private maps**, and the maps are merged
  once at the end. No worker writes to shared state, so there are no locks and
  nothing to contend for.
* **Analysis**: the five analyses run as five goroutines over the finished,
  read-only dataset, each writing to its own slot of a result slice.
* Everything shared is either immutable after construction (the block index,
  the final dataset) or owned by one goroutine. `go test -race` checks this.

## Memory and resource management

* **Streaming**: CSVs are read record by record through a buffered reader; a
  file is never loaded whole. Only compact parsed rows are kept, and the raw rows become garbage once aggregated.
* **String interning**: repeated values (neighborhoods, offense names, request
  types, block keys) are stored once per worker. This also matters because
  `encoding/csv` returns fields that point into a larger per-row string;
  interning copies the few bytes needed so the full row can be freed.
* **Garbage collector**: Go's concurrent GC reclaims the raw slices after the
  join. `debug.SetGCPercent(200)` lets the heap grow further between
  collections, trading memory for fewer GC cycles; this suits a short batch
  job. The report footer prints heap use and GC cycles for each run.
* **Bounded buffers**: channels are small and buffered, so the reader cannot
  run far ahead of the workers.
* Generics keep one loader for all three datasets with no interface boxing.

## Assumptions and limitations

* **Analysis window**: the span of the 311 file (a 2026 export), so all three
  datasets are compared over the same period. Override with `-from`/`-to`.
* **Time zones**: crime timestamps are published in UTC (converted, the daily
  curve peaks in the evening and bottoms out around 6 AM). They are converted to
  America/New_York. "Night" is 21:00-03:59. About 3% of crime rows sit at
  exactly local midnight, which looks like a "time unknown" default, so those
  are excluded from night-time counts only.
* **Violent crime** = homicide, rape, robbery (all types), aggravated assault.
* **Active license** = not "Closed" and overlapping the window, counted once
  per license number (the file has one row per licensee per year).
* **311 rows from the call centre** (`ECC-` types, mostly information requests)
  are excluded: they have no address and close instantly.
* **Response time** counts completed requests only; the share still open is
  reported next to it.
* Counts are not adjusted for population, and all findings are associations,
  not proof of cause. Each question states its own caveats in the report.
