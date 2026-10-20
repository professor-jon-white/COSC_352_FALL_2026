# Midterm Capstone Grade — Wariz Rasheed

**Raw score: 99/100    Curved grade: 99/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 3. Questions answered: 5. Visualization: html. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 19 | 20 |
| Three Datasets | 15 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 15 | 15 |
| **Raw Total** | **99** | 100 |
| **Curved Grade** | **99** | 100 |

## How this grade was reached

Go. 3 real datasets bundled (311 2026 943k, NIBRS 278k, liquor 31k). Worker-pool parse, fan-out/fan-in block-level join, 5 analyses run concurrently, race-clean on full data. 5/5 cross-dataset questions with ASCII and HTML charts; fresh run reproduces shipped report byte for byte. 15 tests. 1.6 s run.

### Datasets identified
- 311 Customer Service Requests 2026 (Open Baltimore)
- NIBRS Group A Crime Data (Open Baltimore)
- Liquor Licenses (Open Baltimore / BLLC)

### Concurrency primitives found
Per-file streaming reader goroutine -> batches channel (cap workers*2) -> N parser goroutines with private slices/interners -> results channel, WaitGroup (internal/ingest/csv.go:117-205); 3 files loaded concurrently (pipeline.go:118-134); generic fold fan-out/fan-in with private accumulators for index and join, 3 dataset folds in parallel (pipeline.go:166-246); 5 analyses in 5 goroutines writing own result slots (analysis.go:36-49)

### Design patterns found
Pipeline (4 explicit stages), Builder (pipeline.NewBuilder().Crime().Requests().Licenses().Workers().Window().Build()), Strategy (analysis.Analysis interface, 5 implementations), Factory (report.New("terminal"|"html") Renderer); generics for Load[T] and fold[T,A]

### Questions answered
1. Which 311 request types track violent crime across neighborhoods after adjusting for total 311 volume?
2. Do neighborhoods with more active liquor licenses see more night-time (21:00-03:59) violent crime?
3. Do high-crime neighborhoods wait longer for 311 requests to be closed?
4. Are street blocks with a liquor license more violent than blocks without one, and which license types stand out?
5. Which neighborhoods need combined intervention first (weighted priority index over violent crime, 311 demand, slow response, license density)?

Stakeholders named: Department of Public Works / Department of Transportation, Board of Liquor License Commissioners / Baltimore Police Department, Mayor's Office (equity of city services), City Planning / Board of Liquor License Commissioners, Mayor's Office

### Overview
Professional multi-package Go (internal/model, geo, ingest, pipeline, analysis, report, stats) with a streaming worker-pool CSV loader, a generic fan-out/fan-in fold, a normalized street-block join key with a majority-vote block-to-neighborhood index, Builder/Strategy/Factory patterns that actually exist in code, deterministic output, and 15 passing tests including an end-to-end run on real data slices. Highly maintainable; README, code and runtime behavior are fully consistent.

### Stakeholder value
Each question names its stakeholder and joins at least two datasets (Q4 and Q5 use all three at block level). Q3 refutes an equity hypothesis with data, Q4's 5.7x licensed-block finding and Q5's transparent weighted priority index are directly usable by Planning and the Mayor's Office. Q1's headline correlate is a noisy low-volume request type, though the dirty-street/illegal-dumping signals below it are meaningful.

### Issues noted
- Q1 headline picks a low-volume oddly specific 311 type (HLTH-Animal ... Standby) as the top size-adjusted correlate
- Q4 unlicensed-block baseline is biased upward because event-free blocks are absent from the data (acknowledged in report)
- Counts not population-adjusted (acknowledged)
- run.sh builds into bin/ inside the project folder

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

Detailed grader report with build logs and captured program output: `grading/wariz_rasheed.md` in the course grading folder.
