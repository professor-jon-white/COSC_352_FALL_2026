# Midterm Capstone Grade — Tyriqe Thomas

**Raw score: 52/100    Curved grade: 80/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: html. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 7 | 15 |
| Five Questions | 8 | 20 |
| Three Datasets | 5 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 10 | 15 |
| **Raw Total** | **52** | 100 |
| **Curved Grade** | **80** | 100 |

## How this grade was reached

Go. 1 Open Baltimore dataset (BNIA dropout rate, one citywide row) plus a 9-row CompStat table hand-typed from a BPD PDF; the two cannot be joined, which the student admits. Token concurrency. 5 questions on gun incidents by district, 0 cross-dataset. HTML with SVG charts, clean tests, good README. Runs in 0.2 s.

### Datasets identified
- High School Dropout/Withdrawal Rate - City (BNIA Vital Signs via Open Baltimore, 1 citywide row)
- BPD CompStat district figures week ending 2021-07-17 (hand-transcribed from PDF, not Open Baltimore)

### Concurrency primitives found
sync.WaitGroup + 1-slot buffered error channel in a custom group helper (concurrent.go); 2 goroutines load the two CSVs (main.go:261-272), 3 goroutines write the three outputs (main.go:281-287); no worker pool, no parallel parse/join/aggregate

### Design patterns found
Implicit two-stage pipeline (load -> write); errgroup-style helper; no named pattern claimed or realized

### Questions answered
1. Descriptive: how much gun violence is there, and how high is the dropout rate?
2. Exploratory: what patterns show up (district change 2020->2021, arrests per incident, dropout year-over-year)?
3. Explanatory: what correlates with a district's gun-related incidents?
4. Evaluative: did more gun arrests go with fewer gun incidents?
5. Predictive: can a district's 2020 gun incidents predict its 2021 count (leave-one-out)?

Stakeholders named: none

### Overview
Idiomatic, well-commented single-package Go split cleanly into ingest/validate, analysis, stats, HTML report and a small concurrency helper, with strong unit and end-to-end tests, deterministic atomic output and rigorous statistics (p-values, Spearman, leave-one-out). Highly maintainable and honest about limitations, but engineered around ~20 input numbers.

### Stakeholder value
No stakeholder is named and the two datasets cannot be joined (one citywide dropout row vs nine police districts), so the report offers no cross-dataset insight; Q3-Q5 are CompStat-only and could be read off the source PDF. The statistical caveats are sound but a Mayor's Office would learn nothing actionable.

### Issues noted
- Only one dataset is from Open Baltimore and it is a single aggregated citywide row; CompStat table is hand-typed from a BPD PDF
- Datasets are never joined; cross-dataset requirement unmet by construction
- Token concurrency: 2 loader + 3 writer goroutines, no worker pool or parallel aggregation
- No municipal stakeholders named
- Shipped prebuilt binary is a Linux x86-64 ELF, not runnable on macOS
- No run script (README commands only)

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

Detailed grader report with build logs and captured program output: `grading/tyriqe_thomas.md` in the course grading folder.
