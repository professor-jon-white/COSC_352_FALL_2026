# Baltimore Municipal Data Processing Engine

**Author:** Arpana Basnet  
**Language:** Go  
**Platform verified:** macOS on Apple Silicon

## Project overview

This application combines Baltimore 311 service-request records with vacant-building notice records to support fictional municipal stakeholders: the Mayor’s Office, Housing, Public Works, and City Planning.

It reads two raw CSV datasets, normalizes neighborhood names, joins their neighborhood summaries, and answers five research questions. Results are available as formatted terminal output and an HTML report with charts.

## Datasets

Source portal: https://data.baltimorecity.gov/

- [311 Customer Service Requests 2025](https://data.baltimorecity.gov/datasets/52c763f4496d4101ba0c20af33336a9c_0/explore)
- [Vacant Building Notices](https://data.baltimorecity.gov/datasets/691d65a5f85640e6aaa46930bd9dc102_1/explore)

Downloaded for this project on October 13, 2026.

| Local file | Description | Records in the project snapshot |
|---|---|---:|
| `data/requests_2025.csv` | Baltimore 311 service-request export for 2025 | 931,324 |
| `data/Vacant_Building.csv` | Baltimore vacant-building notice export | 11,445 |

The application processes local CSV files rather than downloading data during execution. Keep the supplied files unchanged to reproduce the recorded results. New portal exports may produce different results.

The datasets may represent different time periods. The 311 file contains 2025 requests, while its status fields can reflect later updates.

### Fields used

- Both datasets: `Neighborhood`
- 311 dataset: `SRType`, `SRStatus`, `CreatedDate`, and `CloseDate`

Neighborhood names are trimmed and converted to uppercase before matching.

## Five research questions

1. Which neighborhoods have the most vacant-building notice records, and how many 311 request records do those neighborhoods have?
2. Do neighborhoods with more notices tend to have more 311 requests?
3. Among the ten neighborhoods with the most notices, where are illegal-dumping requests concentrated?
4. How does the unresolved-request percentage compare between neighborhoods with fewer than 100 notices and those with at least 100?
5. How do average recorded closure times compare between those groups?

## macOS requirements

- Go compatible with the version declared in `go.mod`
- Terminal or the VS Code integrated terminal
- A browser to view `report.html`
- Both CSV files in the `data` folder

Check the installed Go version and module requirement:

```bash
go version
cat go.mod
```

No third-party Go packages are required.

All commands below must run from the project folder. If the project is on the Desktop:

```bash
cd ~/Desktop/BaltimoreDataProject
```

## Compile

```bash
mkdir -p bin
go build -o bin/baltimore .
go build -o bin/visualize ./visualize
```

Successful builds normally print nothing. Compile on the target Mac rather than relying on binaries built for another operating system or processor architecture.

## Execute

Generate the analysis results:

```bash
./bin/baltimore > results.txt
```

Generate the HTML report:

```bash
./bin/visualize
```

Open the report on macOS:

```bash
open report.html
```

The report generator prints `Created report.html` when it succeeds.

Alternatively, execute directly from source:

```bash
go run . > results.txt
go run ./visualize
open report.html
```

The HTML report is self-contained and does not require a web server or an internet connection.

## Verify

Check the source for common programming problems:

```bash
go vet ./...
```

Run the data-processing application with Go’s race detector:

```bash
go run -race . > verification_results.txt
```

Compare the output with the saved baseline:

```bash
diff -u baseline_results.txt verification_results.txt
```

No output from `diff` means the files match. A race warning, runtime error, or unexpected difference requires investigation.

The baseline comparison checks that refactoring preserved the recorded results. It is not independent proof that every analysis is correct. The race detector checks memory accesses exercised during that execution; it does not prove that every possible execution is race-free.

### Expected results for the supplied snapshot

| Check | Expected result |
|---|---:|
| 311 records | 931,324 |
| 311 records missing neighborhood | 274,497 |
| Notice records | 11,445 |
| Notice records missing neighborhood | 0 |
| Matched neighborhoods | 213 |
| Pearson correlation | 0.404 |
| Illegal-dumping records | 11,888 |
| Illegal-dumping records missing neighborhood | 4 |
| Unresolved percentage: below 100 notices | 1.81% |
| Unresolved percentage: at least 100 notices | 1.26% |
| Average closure time: below 100 notices | 13.65 days |
| Average closure time: at least 100 notices | 11.65 days |

These checks apply to the supplied files, not all future exports.

## Software architecture

The application follows a Pipeline pattern with distinct stages:

1. Ingest raw CSV records using `encoding/csv`.
2. Normalize neighborhood names and parse request fields.
3. Aggregate records into worker-owned neighborhood summaries.
4. Join the dataset summaries by normalized neighborhood name.
5. Compute and print the five analyses.
6. Reformat the saved results into an HTML report.

| File | Responsibility |
|---|---|
| `main.go` | Coordinate ingestion and output; ingest notice records |
| `service_data.go` | Parse and aggregate 311 request fields |
| `join.go` | Match neighborhoods using a worker pool |
| `analysis.go` | Coordinate analyses; notice ranking and correlation |
| `dumping.go` | Illegal-dumping analysis |
| `backlog.go` | Unresolved-request comparison |
| `closure.go` | Recorded closure-time comparison |
| `visualize/main.go` | Convert `results.txt` into HTML charts |
| `go.mod` | Declare the Go module and toolchain requirement |

## Concurrency and thread safety

The two datasets are ingested concurrently.

Each ingestion pipeline uses four goroutine workers. The CSV reader preserves record boundaries and sends extracted fields through a channel with capacity 256. Workers concurrently normalize fields, parse request dates and statuses, and aggregate records.

CSV record tokenization itself is sequential within each file. Worker processing is concurrent, and the two files have separate readers.

Each worker owns its maps. After workers finish, the collector combines their summaries. Workers do not concurrently modify a shared map.

Neighborhood matching uses another four-worker pool. It reads the completed maps and sends matched names through a channel. Names are sorted before reporting.

`sync.Once` ensures that the 311 summary is loaded once per process and reused by the analyses. Each input dataset is read once per application run.

Go’s runtime schedules goroutines on operating-system threads. Goroutines are not individual dedicated threads, and concurrency does not guarantee a speed improvement for every workload.

## Memory and resource management

- CSV input is streamed; the entire dataset is not loaded into a slice.
- Bounded ingestion channels limit queued records.
- Persistent memory stores neighborhood summaries rather than individual service requests.
- `strings.Clone` prevents newly stored neighborhood keys from retaining larger CSV backing strings.
- Go manages heap memory through garbage collection.
- A fixed worker count limits goroutine overhead.
- Input files are closed using `defer`.
- Workers are collected even when CSV reading fails.
- Structural CSV errors stop processing rather than silently accepting unreliable records.

The 311 parser accepts the observed 28-field export and its 26-field records, which omit the trailing coordinate fields. Other field counts are rejected. Notice records must match their header field count.

The matching queues hold neighborhood names rather than full CSV records. The report generator reads the much smaller summary file, `results.txt`.

## Analysis definitions

- Counts refer to records, not necessarily unique requests or buildings.
- Illegal dumping means the exact request type `HCD-Illegal Dumping`, compared without case sensitivity. `SW-Dumpster Collection` is excluded.
- Unresolved means `New`, `Open`, `Hold`, or `Pending` in the export.
- Question 4 excludes records whose status contains `Duplicate`. Unknown statuses are also excluded and counted.
- `Closed (Transferred)` is classified as recorded closed for Question 4.
- Question 5 includes only status exactly `Closed`, compared without case sensitivity.
- Closure time is `CloseDate - CreatedDate`, measured in elapsed days. Dates are interpreted in `America/New_York`.
- Missing dates, invalid dates, and negative closure times are excluded from closure-time averages and reported separately.
- The 100-notice threshold is a project choice, not an official city classification.
- Group percentages pool eligible request records within each group.
- Correlation uses raw counts across matched neighborhoods.

## Findings and municipal use

- Carrollton Ridge has the most notice records among matched neighborhoods: 744. Housing could use the ranking to prioritize further neighborhood reviews.
- The correlation of 0.404 suggests a positive association between notice and request counts. Planning could investigate whether population, neighborhood size, or service mix explains it.
- Broadway East has 284 illegal-dumping requests, the highest count within the selected ten neighborhoods. Housing and Public Works could review these locations for coordinated follow-up.
- The higher-notice group has a lower recorded unresolved percentage. The Mayor’s Office could investigate differences in request categories before drawing conclusions about service performance.
- The higher-notice group also has a shorter average recorded closure time. Further analysis should compare similar request categories.

## Limitations and data quality

- Neighborhood matching does not establish that requests occurred at vacant-building addresses.
- Correlation does not establish causation.
- Counts are not adjusted for population or neighborhood area.
- Missing neighborhoods and unmatched names reduce analysis coverage.
- Recorded closure does not verify that an issue was resolved.
- The datasets may represent different time periods.
- Different request categories can affect unresolved percentages and closure-time averages.
- Averages can be influenced by unusually long durations.
- The supplied snapshot contains 8,122 matched closed records whose closure date precedes creation; these are excluded from Question 5.
- Duplicate-status filtering is not full record-level deduplication.
- Results describe the supplied export snapshots rather than live municipal conditions.

## Troubleshooting

- **CSV file not found:** run from the project folder and check the filenames inside `data`.
- **Undefined function:** keep the main application’s Go files together in the project root.
- **Report data missing:** regenerate `results.txt` with the analysis application before running the report generator.
- **No terminal output:** output redirected with `>` is written to a file.
- **Different results after downloading new data:** the baseline belongs to the original supplied snapshot.

## Submission contents

Include the application source files, `go.mod`, this README, the two raw CSV files, `baseline_results.txt`, `results.txt`, and `report.html`.

The `bin` folder contains locally compiled executables and is not required because the grader can rebuild them using the instructions above.