# Midterm Capstone Grade — Michael Curbeam

**Raw score: 84/100    Curved grade: 91/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_fetched_runtime. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 10 | 15 |
| Five Questions | 17 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 14 | 15 |
| **Raw Total** | **84** | 100 |
| **Curved Grade** | **91** | 100 |

## How this grade was reached

Go. 2 real datasets fetched live at runtime (20k most-recent 311, 11.5k vacants), nothing bundled. Full concurrency: parallel fetch, bounded semaphore pool, parallel analytics. 5/5 cross-dataset questions (Pearson by category, ZIP targeting); tables + ASCII. No unit tests, stakeholders implied not named. go.mod pins Go 1.27.2.

### Datasets identified
- 311 Customer Service Requests (current year) - Open Baltimore ArcGIS FeatureServer, newest 20,000 with neighborhood, fetched at runtime
- Vacant Building Notices (Open) - DHCD Open Baltimore ArcGIS FeatureServer, all ~11,481 rows, fetched at runtime

### Concurrency primitives found
goroutine per dataset + buffered result channels + select + context cancel (ingest/ingest.go:13-50); bounded semaphore channel (runtime.NumCPU()) worker pool over 2000-record chunks producing private partial aggregates sent on a channel to a single merging goroutine, WaitGroup + closer goroutine (processor/processor.go:78-115); one goroutine per question writing its own slot + WaitGroup (analytics/analytics.go:25-39); signal.NotifyContext + WithTimeout (main.go:30-33)

### Design patterns found
Pipeline (ingest -> processor -> analytics -> ui as separate packages passing plain structs), generic Template/Strategy paged ArcGIS client fetchLayer[T], fan-out/fan-in worker pool with semaphore and single-owner merge, higher-order RunAll over question functions, generic Result abstraction for rendering

### Questions answered
1. Q1 Top 5 neighborhoods by combined vacant notices + 311 requests
2. Q2 Pearson correlation of vacancy counts with 311 request categories across neighborhoods
3. Q3 Open vs closed 311 share in top-quartile-vacancy neighborhoods vs all others
4. Q4 Top 3 ZIP codes by open code-enforcement 311 requests + vacant notices
5. Q5 Data density (total/mean/median/max per neighborhood) for 311, open 311 and vacancies

Stakeholders named: code enforcement / DHCD (implied by Q4), resource planning (implied by Q5), generic 'municipal stakeholders' - no office explicitly named

### Overview
Idiomatic four-package Go pipeline (ingest/processor/analytics/ui) with a thin main orchestrator, Go generics for a paged ArcGIS client, a generic Result type rendered by a reusable table/histogram package, deterministic sorting, and context-based cancellation throughout. Code is gofmt/vet clean, well commented, and the detailed README matches it exactly; the only maintainability gap is the lack of unit tests for the pure join and statistics functions.

### Stakeholder value
Real value: the Pearson analysis shows sanitation/dumping 311 requests track vacancy counts (r=+0.70) far more than transportation requests (r=+0.26), which tells DHCD/DPW where to co-target cleanup with abatement; the ZIP ranking (21217, 21223, 21213) is directly actionable for code enforcement; and the open-rate comparison shows high-vacancy areas are not being closed out more slowly. Q5 is merely descriptive and stakeholders are implied rather than named.

### Issues noted
- No data bundled; every run requires live network access to Baltimore ArcGIS endpoints, with no offline or cached fallback
- go.mod requires go 1.27.2; built on the Go 1.24 host only because GOTOOLCHAIN=auto downloaded the 1.27.2 toolchain over the network
- Only one test and it is a live network integration test; no unit tests for pearson/IsClosed/Category/merge/finalize
- Stakeholders not explicitly named; Q5 is single-dataset descriptive statistics
- 311 side limited to the newest 20,000 records (~2% of the layer), so counts are recency-biased (disclosed)
- Vacancies attributed to ZIP via each neighborhood's dominant 311 ZIP, a heuristic that can misallocate
- Q4 histogram bar labels show rank numbers instead of ZIP codes

## What each column means

| Column | Max | How it was scored |
|---|---|---|
| Concurrency | 20 | Native concurrency primitives (goroutines/channels, Rust threads/rayon, etc.) used across parse, join and aggregate stages with correct synchronization. Full multi-stage worker pools = 20. Concurrency in one real stage (usually ingest only) = about 13. Token concurrency (one goroutine per file) = about 7. Verified by reading the code and by running the Go race detector where feasible. |
| Real Data | 15 | At least two genuine Open Baltimore datasets downloaded onto the computer and used by the documented run. Real data bundled = 15. Real data fetched at runtime but not bundled = 10. Synthetic data earns partial credit only if it uses the exact Open Baltimore schema; invented schemas or hand-typed rows = 0 to 1. Verified by inspecting headers and records against the live portal. |
| Five Questions | 20 | Five distinct research questions for named municipal stakeholders that genuinely cross datasets, with formatted summaries or visualizations. Points are lost for single-dataset questions, trivial counts, unnamed stakeholders, degenerate answers, or no visualization. |
| Three Datasets | 15 | Number of distinct Open Baltimore datasets actually ingested in the documented run. Three or more = 15. Two = 8. Mixed sources where one is not from the portal = 5 to 10. None = 0. |
| Runs Well + Documented | 15 | Run result (10): builds and executes end to end on the included or downloaded data with sensible output. README (5): explicit macOS compile, execute and verify instructions that match reality. Nothing in the submission was modified to make it run. |
| Project Structure | 15 | Separation of ingestion, join, analysis and output; design patterns actually realized in code; package layout; tests; maintainability. |
| Raw Total | 100 | Sum of the six columns. |
| Curved Grade | 100 | Class curve with a 70 floor, piecewise-linear on the raw total. Anchors: 70 printout only, 75 minimal synthetic data, 80 weak concurrency and weak questions, 85 real data but minimal concurrency or questions, 88 to 93 real data with full concurrency on two datasets, 100 for the top projects with three or more real datasets. |

The grader built each project from source exactly as the README described, ran it with timeouts, captured output, and read every source file. Full build logs, run output, race-detector results and per-criterion justifications are in the detailed report referenced below.

Detailed grader report with build logs and captured program output: `grading/michael_curbeam.md` in the course grading folder.
