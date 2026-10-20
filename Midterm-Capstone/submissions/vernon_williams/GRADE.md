# Midterm Capstone Grade — Vernon Williams

**Raw score: 47/100    Curved grade: 78/100**    (submitted LATE; noted, not deducted)

Language: Go. Builds: yes. Runs: partial. Data status: real_bundled. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 12 | 15 |
| Five Questions | 7 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 8 | 15 |
| Project Structure | 5 | 15 |
| **Raw Total** | **47** | 100 |
| **Curved Grade** | **78** | 100 |

## How this grade was reached

Go, LATE. 2 real but pre-aggregated 55-row BNIA indicators (violent crime rate, poverty %). Token concurrency (worker pool over 55 rows). 5 questions, 2 cross-dataset (above/below averages). fmt.Print bug prints literal format strings for all 20 Q4 lines; README describes a Strategy pattern that does not exist; builds only via Go 1.27.2 auto-download, go vet fails.

### Datasets identified
- Violent Crime Rate per 1,000 Residents - Community Statistical Area (BNIA Vital Signs via Open Baltimore; data/crime.csv, column viol23)
- Percent of Households Living Below the Poverty Line - Community Statistical Area (BNIA Vital Signs via Open Baltimore, ACS-derived; data/poverty.csv, column hhpov23)

### Concurrency primitives found
2 goroutines load the two CSVs and send Result on a buffered channel (main.go:4-6, concurrent.go); 3-goroutine worker pool with jobs/results channels (cap 55) and sync.WaitGroup parses two floats per community (workers.go:11-33); all analysis sequential

### Design patterns found
README claims Strategy (RankingStrategy/CrimeStrategy/PovertyStrategy) but no such code exists; actual structure is a flat procedural split across 6 files with a worker-pool helper

### Questions answered
1. Which Baltimore community has the highest violent crime rate?
2. Which Baltimore community has the highest household poverty percentage?
3. What are the average violent crime rate and household poverty percentage across communities?
4. Which communities have both violent crime and poverty above their averages?
5. Which communities have violent crime above average but poverty below average?

Stakeholders named: none

### Overview
Short procedural Go split across six files (load, join, workers, analysis, visualization). Data is parsed into typed structs by a worker pool and then converted back to strings so each of five analysis functions can re-parse it; the README describes a Strategy pattern that is not in the code. No tests, a go vet failure, typos and inconsistent formatting; reads as a first draft.

### Stakeholder value
No stakeholders are named. Q1-Q3 are single-dataset maxima and averages; Q4/Q5 (above-average crime vs above/below-average poverty) are the only cross-dataset questions and Q5's high-crime/low-poverty flag has some planning value, but Q4's output is garbled by a Printf bug and everything is threshold counting on 55 pre-aggregated rows.

### Issues noted
- LATE submission
- go.mod requires go 1.27.2; builds only via undocumented automatic toolchain download, fails with GOTOOLCHAIN=local
- go vet fails: analysis.go:87 fmt.Print with format directives; RQ4 output prints literal '%s | Crime: %.2f ...' strings
- README claims a Strategy pattern (RankingStrategy/CrimeStrategy/PovertyStrategy) that does not exist in the code
- Ships a Windows .exe, not a macOS build
- Token concurrency: 3-worker pool parses 110 floats, analyses sequential, results converted back to strings
- Datasets are pre-aggregated 55-row BNIA indicator tables, not raw records; one column used from each
- Nondeterministic output order and detail lines printed before their question headers
- No tests, no run script

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

Detailed grader report with build logs and captured program output: `grading/vernon_williams.md` in the course grading folder.
