# Midterm Capstone Grade — Grace Joseph

**Raw score: 55/100    Curved grade: 81/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 8 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 12 | 15 |
| Project Structure | 5 | 15 |
| **Raw Total** | **55** | 100 |
| **Curved Grade** | **81** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (311 2026 943k, vacants 11k). Token concurrency (one goroutine per file). 5 questions, only 1 cross-dataset (Q4 comparison table). Output bug: 311 sections print empty before the file finishes loading, then everything prints again. No stakeholders named. Builds and runs in 1.1 s.

### Datasets identified
- 311 Customer Service Requests 2026 (Open Baltimore)
- Vacant Building Notices (Open Baltimore)

### Concurrency primitives found
2 goroutines (one analyzeCSV per file) + sync.WaitGroup + buffered results channel cap 2 with closer goroutine (main.go:242-268); parsing/aggregation inside each goroutine is sequential; comparison on main goroutine

### Design patterns found
None named/realized; single-file procedural decomposition (analyzeCSV, printTopCounts, printNeighborhoodComparison, printDataAnalysis, printNeighborhoodBarChart)

### Questions answered
1. What types of 311 service requests are reported most often?
2. Which neighborhoods have the highest number of vacant building notices?
3. Which neighborhoods have the highest number of 311 service requests?
4. Which neighborhoods appear in both datasets, and how do their numbers compare?
5. How many 311 records are missing neighborhood information, and how could that affect the results?

Stakeholders named: Baltimore City residents (generic), local government agencies (generic)

### Overview
Single-file, procedural Go program: two goroutines each stream one CSV into neighborhood/SRType count maps, then top-10 lists, one ASCII bar chart and a side-by-side comparison table are printed. Readable and race-free, but the print blocks sit inside the results-receive loop (so 311 sections print empty first and the vacant list prints twice), paths are hard-coded, sort code is duplicated, and there are no tests or named patterns.

### Stakeholder value
A Housing or DPW analyst could glance at the comparison table (e.g., Carrollton Ridge 744 notices vs 6,540 requests) and the 26.6% missing-neighborhood figure, but there is no rate, ratio, ranking or interpretation built on the join. Four of five questions are single-dataset top-10 counts and no specific office is named.

### Issues noted
- Output-loop bug (main.go:272-311): 311 sections printed empty on first iteration, vacant list printed twice
- Token concurrency: two goroutines, no pool, sequential join/aggregate
- Only Q4 is cross-dataset; stakeholders generic; raw counts only
- go.mod declares go 1.27.2 (toolchain auto-download, undocumented); Windows .exe bundled
- Hard-coded data paths, no tests, duplicated sort logic, README lacks dataset links

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

Detailed grader report with build logs and captured program output: `grading/grace_joseph.md` in the course grading folder.
