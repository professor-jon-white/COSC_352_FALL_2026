# Midterm Capstone Grade — Amari Hall

**Raw score: 62/100    Curved grade: 84/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 9 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 8 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 8 | 15 |
| **Raw Total** | **62** | 100 |
| **Curved Grade** | **84** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (NIBRS 278k, 311 2026 943k). Weak concurrency: two file goroutines plus a pool that re-aggregates pre-aggregated keys; parsing sequential. 5 questions, 3 cross-dataset but all shallow counts (Report 4 misreads a city-initiated rat sweep as resident demand). No stakeholders. ASCII bars. Prebuilt binaries non-executable.

### Datasets identified
- NIBRS Group A Crime Data (data/crime.csv, 2022-Oct 2026)
- 311 Customer Service Requests 2026 (data/service_requests.csv)

### Concurrency primitives found
goroutines, buffered channels, sync.WaitGroup; one goroutine per dataset streams and aggregates its CSV (main.go:211-232); bounded worker pool (<=8) with local maps re-aggregates pre-aggregated neighborhood|month counts (analysis.go:29-148)

### Design patterns found
None named or realized; procedural split into ingest (main.go), aggregation + report functions (analysis.go), date parsing (dates.go)

### Questions answered
1. Which neighborhoods have the highest combined volume of 311 requests and crime records?
2. What is the Pearson correlation between neighborhood-level 311 counts and crime counts?
3. Which neighborhoods are above average in both datasets?
4. What 311 request types are most common in those high-activity neighborhoods?
5. How do monthly 311 and crime counts change Jan-Sep 2026?

Stakeholders named: (none named; 'municipal leaders' implied)

### Overview
Small single-package Go program in three files: two goroutines stream the two CSVs into count maps, a bounded worker pool merges pre-aggregated neighborhood-month counts, and five report functions print tables and ASCII bars. Readable and deterministic, with honest caveats printed, but dataset logic is dispatched on display-name strings, keys are 'a|b' strings, no design pattern is realized and there is one small test.

### Stakeholder value
The reports show that large neighborhoods have more 311 requests and more crime (r=0.777) and list the top ones, which is coincidental counting rather than insight; nothing is normalized for size. No stakeholders are named, and the headline 311 category is a city-run proactive rat sweep misread as resident demand, so a Mayor's Office or DPW would learn little that is actionable.

### Issues noted
- No stakeholders named; questions are generic count rankings and a size-driven correlation.
- Report 4 headline 'SW-RAT RUBOUT PROACTIVE' is a city-initiated program, not resident demand.
- Worker pool only re-aggregates ~15k pre-aggregated keys; CSV parsing is sequential per file.
- Prebuilt binaries shipped without execute permission (mode 0666) and an executable copy was SIGKILLed by macOS; could not be verified (source build graded).
- go.mod go 1.27.2 forces a toolchain download on older Go installs.

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

Detailed grader report with build logs and captured program output: `grading/amari_hall.md` in the course grading folder.
