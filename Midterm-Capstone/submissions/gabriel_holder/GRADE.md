# Midterm Capstone Grade — Gabriel Holder

**Raw score: 44/100    Curved grade: 77/100**

Language: Go. Builds: yes. Runs: yes. Data status: simulated. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 13 | 20 |
| Real Data | 1 | 15 |
| Five Questions | 7 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 9 | 15 |
| Project Structure | 6 | 15 |
| **Raw Total** | **44** | 100 |
| **Curved Grade** | **77** | 100 |

## How this grade was reached

Go. 10 hand-typed synthetic rows per file (admitted). Parse-only worker pool. 5/5 cross-dataset questions, but Pearson r = 0.834 on 10 fake rows is meaningless. No go.mod so `go build ./...` fails; `go build main.go` works. JSON report.

### Datasets identified
- Violent Crime Rate per 1,000 Residents by CSA (BNIA Vital Signs; synthetic stand-in)
- Neighborhood Businesses per 1,000 Residents by CSA (BNIA Vital Signs; synthetic stand-in)

### Concurrency primitives found
2 goroutines + sync.WaitGroup per dataset (main.go:224-229); per-CSV bounded worker pool: jobs/parsed channels cap 4*workers, N parser goroutines, reader goroutine, single collector owning map (main.go:62-110)

### Design patterns found
Implicit pipeline (loadCSV -> join -> score -> display/JSON); single file, no named pattern

### Questions answered
1. Which CSAs have the highest violent crime rate?
2. Which CSAs have the lowest business density?
3. How strongly are the two rates associated (Pearson correlation)?
4. Which CSAs score highest on an exploratory crime-plus-low-business priority index?
5. Which CSAs have the largest numerical difference between the two rates?

Stakeholders named: none

### Overview
Single-file Go program with clean functional decomposition (loadCSV worker pool, join, correlation, score, display, JSON writer) and flag-driven column mapping. The parsing stage uses a correct bounded worker pool with channels and a single collector; join and scoring are sequential. No go.mod, no tests, no packages.

### Stakeholder value
No stakeholder is named, and all five answers (Pearson r=0.834, priority index, rate gaps) are computed from ten admittedly invented values per file, so nothing can be learned about Baltimore. The priority-index idea (Q4) would be useful on real Vital Signs exports; Q5 subtracts incommensurable rates.

### Issues noted
- Both CSVs are synthetic 10-row files (README admits); no fetch script or generator; real data absent
- No go.mod: go build ./..., go vet ./..., go test ./... all fail
- Worker pool only in parsing; join/aggregate sequential
- No tests, no named stakeholder, Q1/Q2 single-dataset
- Default run overwrites output/report.json; stray .DS_Store

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

Detailed grader report with build logs and captured program output: `grading/gabriel_holder.md` in the course grading folder.
