# Midterm Capstone Grade — Elizabeth Stewart

**Raw score: 87/100    Curved grade: 93/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 17 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 13 | 15 |
| **Raw Total** | **87** | 100 |
| **Curved Grade** | **93** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (NIBRS 54k, vacant notices 11k). Full concurrency: parsing pools + FNV-partitioned join/aggregate workers, output invariant to worker count, race-clean. 5/5 questions use the join, though Q3/Q4 are stratified single-dataset views. 1 s run, 4 tests. Analysis and reporting share one file.

### Datasets identified
- NIBRS Group A Crime Data (Open Baltimore)
- Vacant Building Notices (Open Baltimore)

### Concurrency primitives found
2 reader goroutines + sync.WaitGroup (analysis.go:57-64); per-dataset parsing worker pool over bounded chan (cap 256) with sync.Mutex error capture (159-215); N hash-partitioned join/aggregate goroutines each owning a bounded shard chan, FNV-32a routing (66-81,105-134); results merged via channel

### Design patterns found
Pipeline (read -> parse pool -> partitioned aggregate -> merge -> Report); cmd/internal package layout

### Questions answered
1. Which neighborhoods have the most vacancy notices, and how many crime records and distinct cases are reported there?
2. How concentrated are notices in the top 10/25/50/100% of vacancy neighborhoods, and what share of distinct crime cases occurs there?
3. What crime types are reported in neighborhoods with versus without notices?
4. How do monthly crime record counts vary in neighborhoods with notices?
5. Which neighborhoods rank highest on a transparent combined planning score?

Stakeholders named: Baltimore City Planning team

### Overview
A disciplined Go pipeline split into cmd/baltimore (flags) and internal/analysis (Load/Report). Two streaming CSV readers feed bounded channels into parsing worker pools and hash-partitioned join/aggregation workers that exclusively own their maps, with strict input validation and four tests including a serial-vs-parallel equivalence check. Maintainable and race-clean, though analysis and all report rendering share one 350-line file.

### Stakeholder value
A City Planning office would learn that the top 10% of vacancy neighborhoods hold 60% of notices but only 14% of distinct crime cases, and that Carrollton Ridge, Broadway East and Sandtown-Winchester rank highest on a combined vacancy/crime score while Downtown is crime-heavy but vacancy-light. The joins are meaningful per-neighborhood aggregations, not coincidental counting, though Q3/Q4 are unnormalized counts.

### Issues noted
- README says 'cd data-processing-engine' but no such folder exists in the submission
- Join/aggregation and all output formatting live in one file; no separate output package
- Q3/Q4 use raw counts without normalizing for group size
- fnv.New32a() allocated per event in dispatcher loop
- No run script; commands only in README

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

Detailed grader report with build logs and captured program output: `grading/elizabeth_stewart.md` in the course grading folder.
