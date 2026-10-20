# Midterm Capstone Grade — Elijah Stewart

**Raw score: 38/100    Curved grade: 75/100**

Language: Go. Builds: yes. Runs: partial. Data status: sample_only. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 1 | 15 |
| Five Questions | 7 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 9 | 15 |
| Project Structure | 6 | 15 |
| **Raw Total** | **38** | 100 |
| **Curved Grade** | **75** | 100 |

## How this grade was reached

Go. 14 fabricated rows with invented headers; README says download real 311 and arrests but gives no URL; default run fails, only --offline works. Token concurrency (two goroutines). 5 questions, 1 cross-dataset (Q5 label overlap, case-sensitive). No stakeholders.

### Datasets identified
- 311 Customer Service Requests (claimed; not bundled)
- BPD Arrests (claimed; not bundled)

### Concurrency primitives found
sync.WaitGroup + 2 goroutines (main.go:142-152), one per CSV; no channels, no worker pool

### Design patterns found
None realized; streaming per-file summarize() helper, single main.go

### Questions answered
1. Which 311 service-request types appear most often?
2. Which neighborhoods have the most 311 requests?
3. Which offenses or charge descriptions appear most often in arrest records?
4. Which neighborhoods or districts appear most often in arrest records?
5. Which matching area labels have records in both datasets? (combined counts; exploratory)

Stakeholders named: none

### Overview
Terse single-file Go program: two goroutines each stream one CSV into type/area count maps, then five top-8 ASCII bar charts are printed. Code is tidy and race-free with two small tests, but there is no package structure, no realized design pattern, and the only cross-dataset step is a case-sensitive label intersection.

### Stakeholder value
No stakeholders are named and four of five questions are single-dataset frequency counts. The one join sums 311 and arrest counts per matching label, which the README itself calls exploratory and non-interpretable; on real data it matches a single label ('CARE') due to case mismatch. A city office would gain no actionable insight.

### Issues noted
- Sample CSVs are fabricated (headers crime_type/service_type do not match Open Baltimore); real data absent with no URL or fetch script
- Cross-dataset join (main.go:170-175) is case-sensitive and collapses to one match on real data
- Token concurrency: two goroutines, no pool, sequential join/aggregate
- Per-row map allocation and silent skipping of malformed rows
- README gives Windows instructions first and no dataset links; only --offline path runs

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

Detailed grader report with build logs and captured program output: `grading/elijah_stewart.md` in the course grading folder.
