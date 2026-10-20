# Baltimore Neighborhood Service and Vacancy Analyzer

A Go command-line capstone that combines two Baltimore public datasets to answer five city-services research questions. It includes raw data extracts, concurrent processing, a neighborhood join, formatted reports, ASCII charts, tests, and offline execution.

## Quick start on macOS

Install **Go 1.22 or newer** using the appropriate macOS installer from the [official Go download page](https://go.dev/dl/). Choose Apple Silicon/ARM64 for an M-series Mac or Intel/AMD64 for an Intel Mac, then reopen Terminal.

Unzip the project, open Terminal, and change into its folder. For example:

```sh
cd ~/Downloads/baltimore-capstone
go version
sh run.sh -workers 4
```

No Python, Node.js, database, third-party Go modules, API credentials, or internet connection is required to run the analyzer after Go is installed. The real data is bundled.

The program prints the five answer sections and writes:

- `output/report.md`: complete findings, tables, ASCII chart, data-quality counts, and caveats.
- `output/neighborhoods.csv`: machine-readable neighborhood results.

To build and run manually:

```sh
mkdir -p bin
go build -trimpath -o bin/analyzer ./cmd/analyzer
./bin/analyzer -data data -out output -workers 4
```

Use `-workers 1` through `-workers 64`; four is a reasonable starting point. Each processing stage has its own pool, so this flag is not the total number of goroutines. `runtime.GOMAXPROCS` also sets the maximum number of OS threads simultaneously executing Go code.

## Verify the project

```sh
sh verify.sh
```

This runs tests, `go vet`, build checks, SHA-256 verification of both raw files, and byte-for-byte comparisons between one-worker, four-worker, and bundled results. Run the following optional race check when a C compiler is available:

```sh
# If needed on macOS: xcode-select --install
go test -race -count=1 ./...
```

The project was executed and tested on Linux with Go 1.27.2. The analyzer was also cross-compiled for both macOS ARM64 and AMD64. The user subsequently reported a successful native run on an Apple Silicon Mac with `go version go1.27.2 darwin/arm64`: verification completed without errors, the application printed the report, and both output files were present. This native Mac result is user-reported, not an independently captured execution log; native Intel Mac execution has not been tested.

## Stakeholder and research questions

The fictional audience is a joint housing and city-services review team. This is a neighborhood screening exercise, not an automatic funding or enforcement system.

- **Concentration:** Which neighborhoods have high counts on both vacancy properties and 311 requests?
- **Service needs:** How does the mix of request types differ between low- and high-vacancy-count neighborhoods?
- **Recorded closure time:** How do median closure times for the same request types differ between those groups?
- **Time patterns:** How do monthly requests change across the selected quarter in each group?
- **Coordination priorities:** Which five neighborhoods rank highest under a transparent score, and what is their leading request category?

All five answers use the joined datasets. A question about 311 categories alone is not treated as a cross-dataset result: categories are compared within groups defined using the vacancy input.

## Data provenance and scope

The two inputs are selected raw record-level attributes from Baltimore's public ArcGIS services, not precomputed summary statistics.

| Input | Scope | Raw rows | Download pages |
|---|---|---:|---:|
| [311 Customer Service Requests 2025](https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/311_Customer_Service_Requests_2025/FeatureServer/0) | Created on or after October 1, 2025 and before January 1, 2026, UTC | 243,620 | 122 |
| [Vacant Building Notice – Open](https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1) | All records in the open-notice layer at extraction | 11,481 | 12 |

The extracts were downloaded October 15, 2026, approximately 9:11–9:13 p.m. Eastern (October 16, 01:11–01:13 UTC). `data/manifest.json` records exact timestamps, source URLs, field selections, SQL filters, pre/post download counts, page counts, and file hashes.

The downloader removes geometry by asking the server for attributes only, requests selected columns, and serializes each returned raw feature as one JSONL line. It does not alter attribute values or aggregate the input. The Go analyzer performs parsing, cleaning, grouping, and joining locally.

### Refreshing the extracts

Keep the bundled snapshot for reproducibility. If desired, download into a new directory:

```sh
go run ./cmd/fetch -out data-refresh
go run ./cmd/analyzer -data data-refresh -out output-refresh -workers 4
```

This requires internet access and can produce different results because source data changes. The original `verify.sh` deliberately checks the bundled `data/` and `output/`, not refreshed files.

The downloader processes both datasets concurrently, paginates using source object-ID order, rejects duplicate object IDs, retries failed requests, and requires downloaded counts to match counts checked before and after retrieval. It stages each dataset as `.partial` before renaming; on download failure, do not use that directory as a completed snapshot.

Equal counts do not guarantee an atomic historical snapshot: live attributes could change without changing row counts. The manifest and local hashes establish what was actually downloaded.

## Architecture and concurrency

The design pattern is **Pipeline**, with separate responsibilities:

```text
raw JSONL files
    |
bounded line readers and parsing worker pools
    |
ordered validation, normalization and deduplication
    |
neighborhood-sharded aggregation worker pools
    |
parallel neighborhood matching
    |
five research analyses
    |
Markdown tables, terminal ASCII chart, CSV
```

| Component | Responsibility |
|---|---|
| `cmd/fetch` | Reproducible, paginated raw-data acquisition |
| `internal/model` | Shared types, dates, identity rules, normalization |
| `internal/ingest` | Concurrent parsing, validation, deduplication, aggregation |
| `internal/analysis` | Concurrent neighborhood join, cohorts, percentiles, medians |
| `internal/report` | Presentation only: tables, ASCII bars, CSV |
| `cmd/analyzer` | CLI flags and orchestration |

Each input has a parsing pool. A sequential dispatcher restores source order and deduplicates records deterministically. A bounded credit channel prevents its reorder buffer from growing without limit.

The dispatcher hashes neighborhood names to aggregation workers. Each worker exclusively owns its neighborhood maps, eliminating shared-map write races. Both datasets load concurrently; once aggregation completes, join workers read immutable maps and build joined rows. A final alphabetical sort makes output independent of scheduling.

Go's runtime schedules goroutines over OS threads; this implementation uses bounded worker pools, not one goroutine per record. It combines imperative data cleaning, typed record modeling, and channel-based concurrent programming.

### Memory and resource management

- Raw JSON is read incrementally rather than unmarshalling a whole file.
- Channels have bounded capacity, providing backpressure.
- At most `8 × workers` records per input can be outstanding in the parse/reorder stage.
- Aggregation channels hold at most 32 records each.
- Exact deduplication requires an in-memory set of unique identities: memory grows with unique input records.
- Exact medians retain eligible closure durations; this also grows with input size.
- Most other retained state scales with neighborhood, type, and month counts.
- Each line has a 4 MiB scanner limit; oversized or malformed input produces an error.
- Go garbage collection manages heap allocations; this is not a constant-memory or zero-allocation design.
- Files and HTTP bodies are closed, HTTP calls have timeouts, and output errors propagate to a nonzero exit status.

Avoid claiming linear speedup. Parsing, deduplication, scheduling overhead, available CPU cores, and input size all affect performance.

## Matching and statistical rules

Neighborhood normalization uses uppercase, trimmed ends, and collapsed internal whitespace. There is no fuzzy matching or spatial inference.

Requests are deduplicated by service-request number. Vacancy properties are deduplicated by `BLOCKLOT`, falling back to normalized address; missing both identifiers makes a vacancy record unusable. Within eligible duplicates, the first source-ordered record wins. Notice IDs are not treated as unique property IDs.

Each dataset is aggregated before joining. This avoids multiplying request counts by the number of vacancy notices in a neighborhood.

All neighborhood names are retained in the CSV. Research comparisons and scoring include only neighborhoods represented in both datasets; dataset-only rows are labeled `unmatched`, their score is blank, and the report lists them.

For matched neighborhoods, sort vacancy counts ascending. With zero-based indexing, set the lower cutoff to `counts[(n-1)/3]` and upper cutoff to `counts[2*(n-1)/3]`. Low contains counts at or below the lower cutoff; high contains counts strictly above the upper cutoff; the rest are middle. Equal counts remain together; group sizes may differ. If there are fewer than three neighborhoods or the cutoffs coincide, no low/high comparison is manufactured.

Question 1 requires both empirical midrank percentiles to be at least 75. The midrank percentile is:

```text
100 × (number strictly below + 0.5 × number equal) / neighborhood count
```

Question 5 averages the vacancy and request midrank percentiles with equal weights. Ties receive equal scores; names break display-order ties only.

Closure time is `(CloseDate - CreatedDate)` in days, only when status begins `CLOSED` and dates are present and ordered correctly. This includes transferred closures. For each compared type, each cohort must have at least 20 eligible closures; up to ten types with the largest combined eligible sample are shown. Medians are descriptive, not statistical tests.

## Important findings about data quality

The bundled 311 extract contains **75,051 records with missing neighborhoods**, about **30.8%** of its raw rows; 168,569 requests remain eligible for neighborhood aggregation, and all 11,481 vacancy records have usable identifiers and neighborhoods in this snapshot ([311 input](https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/311_Customer_Service_Requests_2025/FeatureServer/0), [vacancy input](https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1)).

There are 213 neighborhoods shared by the inputs and 66 dataset-only neighborhoods in the bundled output; comparisons therefore do not represent all Baltimore neighborhoods or all requests ([311 input](https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/311_Customer_Service_Requests_2025/FeatureServer/0), [vacancy input](https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1)).

Use these limitations when explaining the results:

- **Temporal mismatch:** Current open vacancy notices do not reconstruct vacancy status during the 2025 request period.
- **Missing locations:** Excluding unlocated requests may introduce nonrandom selection bias.
- **Selection of neighborhoods:** Common-neighborhood comparisons exclude areas with no matched vacancy records, including potentially genuine zero-notice areas.
- **Meaning of requests:** Records can include proactive agency activity, not only resident complaints; request volume is not a direct measure of unmet need.
- **No denominator:** Counts are not adjusted for population, total buildings, land area, or reporting access.
- **Closure bias:** Open cases are not in closure-time medians, and closed/transferred does not necessarily mean resolved.
- **No causality:** Associations cannot establish that vacant buildings cause complaints or slower service.
- **No automated policy:** The equal-weight score is a student-designed screening choice that requires real-world validation.

## Assignment alignment

The [capstone assignment](https://morganstate.instructure.com/courses/53088/assignments/1140854) requires an approved language, multiple public datasets, native concurrency, memory/resource considerations, clean architecture, five cross-dataset questions, and macOS compile/run instructions.

| Requirement | Implementation |
|---|---|
| Approved language | Go application and Go downloader |
| At least two raw public datasets | Bundled requests and vacancy JSONL extracts |
| Concurrent parsing, joining and aggregation | Parsing pools, sharded reducer pools, join pool |
| Memory and thread safety | Bounded queues, owned maps, explicit retention discussion |
| Design pattern | Pipeline with separate packages |
| Five stakeholder questions | Five generated report sections |
| Formatted outputs | Markdown tables, ASCII bars, CSV |
| Execution and verification | `run.sh`, `verify.sh`, tests, documented commands |
| macOS support | ARM64/AMD64 cross-build checks; successful native ARM64 verification and execution reported by the user |

## Before submitting

Read the code and results so you can explain the data cleaning, channels, join, and five findings. Run `sh verify.sh` and `sh run.sh -workers 4` on a Mac, review any course rules concerning outside assistance, and submit the project files/archive through Canvas yourself.

No Canvas submission has been performed. This package is a working project, not a guarantee of a grade.
