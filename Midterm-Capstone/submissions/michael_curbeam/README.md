# Midterm Capstone: Baltimore Data Engine

A multi-paradigm data processing engine written in Go. It pulls two live Baltimore City open datasets, **311 Customer Service Requests** and **Vacant Building Notices**, joins them by neighborhood and ZIP code, answers five municipal stakeholder questions, and prints the results as ASCII tables and histograms.

Mid-Term Capstone: Multi-Paradigmatic Data Processing Engine.

---

## 1. Overview and Architecture

The engine is a **four-stage pipeline**. Each stage is its own Go package and passes plain data structures to the next.

```
            ┌──────────┐    ┌────────────┐    ┌────────────┐    ┌──────┐
 ArcGIS ──▶ │  ingest  │──▶ │ processor  │──▶ │ analytics  │──▶ │  ui  │ ──▶ terminal
 REST APIs  └──────────┘    └────────────┘    └────────────┘    └──────┘
            2 goroutines    worker pool +      5 goroutines     tables +
            + channels      channel merge      (one per Q)      histograms
```

| Stage | Package | Responsibility |
|-------|---------|----------------|
| 1. Ingest | [ingest/](ingest/) | Fetches both datasets **in parallel** from live ArcGIS FeatureServer JSON APIs, paging through results. |
| 2. Process | [processor/](processor/) | Aggregates and joins the datasets by neighborhood and ZIP code using a bounded pool of worker goroutines. |
| 3. Analyze | [analytics/](analytics/) | Computes the five stakeholder questions concurrently. |
| 4. Present | [ui/](ui/) | Renders results as aligned ASCII tables and `#` histograms. |

[main.go](main.go) is the orchestrator. It creates the context (Ctrl-C handling plus a 3-minute timeout), runs the stages in order, and writes progress to **stderr** and results to **stdout**.

### Paradigms used

- **Concurrent / CSP:** goroutines and channels for fetching, joining and analyzing.
- **Pipeline / dataflow:** each stage consumes the previous stage's output and shares no hidden state.
- **Generic and higher-order programming:** `fetchLayer[T]` pages any ArcGIS layer into any struct type, and `analytics.RunAll` takes a slice of question functions.
- **Procedural / data-oriented:** plain structs and pure functions (for example `pearson`, `IsClosed`, `Category`) transform the data.

### Data sources

| Dataset | Endpoint |
|---------|----------|
| 311 Customer Service Requests | `https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0` |
| Vacant Building Notices (Open) | `https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1` |

Baltimore's open data is served through ArcGIS REST, not Socrata. The 311 layer has about 935,000 rows, so the engine fetches only the **newest 20,000 records that have a neighborhood** (`maxServiceRequests` in [ingest/service_requests.go](ingest/service_requests.go)). All open vacant notices (about 11,500) are fetched.

### Project layout

```
main.go                      pipeline orchestration
ingest/
  ingest.go                  FetchAll: parallel fetch via goroutines + channels
  client.go                  generic paged ArcGIS client (fetchLayer[T])
  service_requests.go        311 fetcher
  vacant_buildings.go        vacant buildings fetcher
  models.go                  ServiceRequest, VacantBuilding, Datasets
  probe_test.go              live integration test (skipped with -short)
processor/
  processor.go               concurrent join and aggregation
  classify.go                status / category classification rules
analytics/
  analytics.go               Result type, RunAll, shared helpers
  questions.go               Q1-Q5
ui/table.go                  ASCII tables and histograms
```

---

## 2. Build, Run and Verify (macOS)

### Prerequisites

- macOS with a terminal (Terminal.app or iTerm)
- **Go 1.27 or newer.** `go.mod` declares `go 1.27.2`, and older toolchains may refuse to build it.
- An **internet connection**, because the engine calls the live Baltimore APIs.

Check your Go installation:

```bash
go version
```

If Go is missing or too old, install it with Homebrew (`brew install go`) or download it from <https://go.dev/dl/>.

### Step 1: Open the project directory

```bash
cd /path/to/baltimore-data-engine
ls
```

You should see `go.mod`, `main.go`, `README.md` and the `ingest`, `processor`, `analytics` and `ui` folders.

### Step 2: Compile and static checks (no network needed)

```bash
gofmt -l .          # prints nothing if all files are formatted
go build ./...      # compiles every package; prints nothing on success
go vet ./...        # static analysis; prints nothing on success
```

A silent run of all three means **zero build or syntax errors**.

### Step 3: Build and run the executable

```bash
go build -o baltimore-engine .
./baltimore-engine
```

Or run without producing a binary:

```bash
go run .
```

The run takes about **30-40 seconds**, mostly downloading data. Progress lines appear on stderr as each stage finishes:

```
ingest: fetching 311 requests and vacant buildings...
ingest: 20000 311 requests, 11481 vacant notices (28.5s)
processor: joined 277 neighborhoods, 32 ZIP codes (72ms)
analytics: answered 5 questions (1ms)
```

After that the banner and five question sections print to stdout. To save a clean report without the progress lines:

```bash
./baltimore-engine > report.txt
```

Counts will differ slightly from run to run, because the datasets are live.

### Step 4: Verify correctness and concurrency safety

Run with Go's **race detector** (it makes the run slower, but it should report nothing):

```bash
go run -race .
```

The output must contain **no** `WARNING: DATA RACE` text.

Run the live ingestion test, which asserts that 20,000 311 rows and more than 10,000 vacant notices are fetched:

```bash
go test ./ingest -run Live -v
```

To run all tests while skipping the network test:

```bash
go test -short ./...
```

### Verification checklist

- [ ] `go build ./...` and `go vet ./...` print nothing.
- [ ] The run prints four progress lines and then the banner.
- [ ] Sections **Q1 through Q5** each print a table. Q1 and Q4 also print a histogram.
- [ ] `go run -race .` shows no data-race warnings.
- [ ] The process exits with status 0 (`echo $?` prints `0`).

### Troubleshooting

| Symptom | Fix |
|---------|-----|
| `go: command not found` | Install Go (`brew install go`) and open a new terminal. |
| `go.mod requires go >= 1.27.2` | Upgrade Go, or lower the `go` line in `go.mod` to your installed version. |
| `ingest: ... no such host` or a timeout | Check your internet connection and retry. The run is cancelled after 3 minutes. |
| `ingest: arcgis error ...` | The city API returned an error. Wait a few minutes and retry. |
| Want faster runs | Lower `maxServiceRequests` in [ingest/service_requests.go](ingest/service_requests.go). |

---

## 3. Stakeholder Research Questions

All questions are answered in [analytics/questions.go](analytics/questions.go).

| # | Question | Method |
|---|----------|--------|
| **Q1** | **Which 5 neighborhoods have the highest combined vacancies and 311 issues?** | Ranks neighborhoods by `vacant notices + 311 requests`. Prints a table and a histogram. |
| **Q2** | **How strongly does vacancy density correlate with specific 311 request types (e.g. sanitation/dumping)?** | Pearson correlation `r` across neighborhoods between vacancy count and request counts for five categories: Sanitation/Dumping, Housing Code, Transportation/Streets, Water/Sewer and Other. |
| **Q3** | **What is the Open vs. Closed 311 status breakdown in heavy-vacancy areas?** | "Heavy vacancy" means the top quartile of neighborhoods by vacant notices. Compares their open/closed shares with all other neighborhoods. |
| **Q4** | **Which 3 ZIP codes need immediate code-enforcement intervention?** | Scores each ZIP as `open code-enforcement 311 requests + vacant notices` and ranks them. Prints a table and a histogram. |
| **Q5** | **What does overall data density look like across neighborhoods, for resource planning?** | Total, mean, median and max per neighborhood for 311 requests, open 311 requests and vacant notices, plus the densest neighborhood. |

### Definitions and assumptions

- **Join key:** neighborhood name, uppercased and trimmed. The vacancy dataset has no ZIP code, so each neighborhood's vacancies are attributed to the **most common ZIP** among its 311 requests. This is how Q4 gets vacancies per ZIP.
- **Open vs. closed:** any status starting with "Closed" is closed. Open means everything else (Open, New, Pending, Hold).
- **Sanitation/dumping:** request types matching dirty, dumping, mixed refuse, sanitation, cleaning, SIU, high grass and weeds (HGW), or trash.
- **Code enforcement:** all Housing and Community Development (`HCD-*`) request types plus sanitation/dumping.
- **Vacancy density:** the raw count of open vacant notices per neighborhood. The data has no area or parcel counts to normalize against.
- **Recency skew:** the 311 side is the most recent 20,000 records, while vacancies are the full open list. Each result prints a note where this matters.

The classification rules live in [processor/classify.go](processor/classify.go) and are easy to adjust.

---

## 4. Concurrency and Memory Safety

### Concurrency (goroutines and channels)

| Where | Technique |
|-------|-----------|
| [ingest/ingest.go](ingest/ingest.go) | `FetchAll` starts one goroutine per dataset. Each sends its result on its own channel, and `FetchAll` receives from both with `select`. The first error cancels the other fetch through `context`. |
| [processor/processor.go](processor/processor.go) | Both datasets are split into 2,000-record chunks. A worker pool bounded by `runtime.NumCPU()` (a semaphore channel) aggregates chunks into private partial results and sends them on a channel. A single goroutine merges them. |
| [analytics/analytics.go](analytics/analytics.go) | `RunAll` launches one goroutine per question and waits with `sync.WaitGroup`. |
| [main.go](main.go) | Ctrl-C and a 3-minute timeout propagate through `context` to the fetchers and the processor. |

### Memory safety

- **No shared mutable state without ownership.** Workers build private maps, and only the merging goroutine writes the final result, so the join needs **no mutexes**. Each analytics goroutine writes to its own slot in the results slice (`results[i]`), and the joined data is read-only by then.
- **Verified with the race detector.** `go run -race .` reports no data races.
- **No goroutine leaks.** Channels are buffered or drained, `close` is called from a single owner, and every goroutine exits when work completes or the context is cancelled.
- **Bounded resource use.** The worker pool is bounded, the 311 download is capped, and HTTP requests have a 30-second client timeout.
- **Language-level safety.** The code uses no `unsafe` and no cgo. Go provides bounds-checked slices and garbage collection, and values from the API decode into typed structs. Null JSON fields become safe zero values.
- **Defensive error handling.** HTTP status codes, ArcGIS error bodies and JSON decode errors are all checked and returned up the pipeline instead of being ignored. `defer` guarantees that response bodies are closed.

---

## Author

Michael Curbeam, COSC 352.001
