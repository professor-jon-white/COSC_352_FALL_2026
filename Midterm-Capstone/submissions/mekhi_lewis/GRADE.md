# Midterm Capstone Grade — Mekhi Lewis

**Raw score: 66/100    Curved grade: 86/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 13 | 20 |
| Real Data | 12 | 15 |
| Five Questions | 9 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 10 | 15 |
| **Raw Total** | **66** | 100 |
| **Curved Grade** | **86** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (arrests 400k, 311 943k) but shipped incidentally, unmentioned in README; run.sh defaults to 3-row fixtures. Partial concurrency: correct bounded-channel parser, sequential aggregation. 5 questions, 2 cross-dataset (Q1/Q2/Q5 single); grid cells instead of neighborhoods, no named stakeholders. Interactive menu.

### Datasets identified
- BPD Arrests 2026 (Open Baltimore ArcGIS export, data/arrests.csv)
- 311 Customer Service Requests 2026 (Open Baltimore ArcGIS export, data/requests_test.csv and 10K-row data/requests_small.csv)

### Concurrency primitives found
goroutine per dataset + sync.WaitGroup (main.go:65-96); per-file producer goroutine -> bounded chan []string (workers*4) -> runtime.NumCPU() worker goroutines -> bounded chan Record, closer goroutine, select on ctx.Done() (engine.go:147-329); Analyze/Report sequential

### Design patterns found
Pipeline (fetch -> parse -> analyze -> report), fan-out/fan-in worker pool, table-driven Strategy map of {title,score,keep} closures for the five analyses; no patterns named in README

### Questions answered
1. Q1 Highest arrest activity by 0.01-degree grid cell (single-dataset)
2. Q2 Highest 311 request volume by grid cell (single-dataset)
3. Q3 Grid cells with both arrests and infrastructure-type 311 requests
4. Q4 Combined arrests + 311 activity by grid cell
5. Q5 Unresolved (open-status) 311 requests by grid cell (single-dataset)

Stakeholders named: none

### Overview
Standard cmd/internal Go layout with a schema-tolerant CSV parser driven by a bounded-channel producer/worker-pool, a deterministic test, run.sh, and a colored interactive terminal menu. Nearly all logic (download, parse, aggregate, render) lives in one 593-line engine.go, aggregation is sequential, and the UI re-parses rendered report text by 'Q<n>:' prefixes. Code is clean and race-free but lightly commented.

### Stakeholder value
Limited: no stakeholders are named, three of five questions are single-dataset rankings, and the two cross-dataset questions are co-location counts in anonymous 0.01-degree lat/lon cells rather than neighborhoods or rates. The honest caveats printed under each section are a plus, and the real-data run does surface genuine downtown-west hotspots, but a Mayor's Office reader would get coordinates, not actionable guidance.

### Issues noted
- Default ./run.sh runs on 3-row synthetic fixtures; real data needs --real (network) or manual -arrests/-requests flags
- README never mentions the bundled real CSVs (.gitignore excludes data/*.csv); 386 MB requests_test.csv appears to have shipped unintentionally
- README's five-analysis list does not match the titles/order the program prints
- No named stakeholders; Q1/Q2/Q5 single-dataset; Q3/Q4 are grid-cell co-counts
- Aggregation and reporting are sequential; only parsing is parallel
- ParseFile re-reads the entire CSV to count skipped rows; all records retained (948 MB RSS on full data)
- engine.go is a single 593-line file spanning all stages; menu parses report text instead of structured results
- Ends in interactive menu (exits cleanly on EOF, so scriptable)

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

Detailed grader report with build logs and captured program output: `grading/mekhi_lewis.md` in the course grading folder.
