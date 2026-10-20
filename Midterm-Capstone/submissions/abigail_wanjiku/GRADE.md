# Midterm Capstone Grade — Abigail Wanjiku

**Raw score: 88/100    Curved grade: 93/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_fetched_runtime. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 10 | 15 |
| Five Questions | 20 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 15 | 15 |
| **Raw Total** | **88** | 100 |
| **Curved Grade** | **93** | 100 |

## How this grade was reached

Go. 2 real datasets (NIBRS 278k, 311 931k) fetched by the README's curl commands, not bundled. Full concurrency with compare mode proving sequential == concurrent (4.2x speedup), race-clean, tests in every package. 5/5 cross-dataset questions with mix-adjusted controls and ASCII charts. 2.4 s run. Lost points on dataset count and bundling.

### Datasets identified
- NIBRS Group A Crime Data (Open Baltimore 204beefe92a645d79fdf0969957bbdf8)
- 311 Customer Service Requests 2025 (Open Baltimore 52c763f4496d4101ba0c20af33336a9c)

### Concurrency primitives found
goroutines, bounded channels, sync.WaitGroup; generic chunked worker pool with splitter/workers/closer/collector (internal/ingest/concurrent.go), buffer recycling via free channel, FNV-1a sharded concurrent join (internal/match/match.go), two-file parallel load (cmd/capstone/main.go); compare mode deep-equals sequential vs concurrent

### Design patterns found
Pipeline (ingest->match->aggregate->report, stages depend only on model), Strategy (report.Report interface, 5 question types), worker pool / fan-out fan-in, sharding by key

### Questions answered
1. Do neighborhoods with the most sanitation/dumping 311 requests also have the highest crime? (with all-311 baseline)
2. Top 10 neighborhoods by combined distress index (crime + resident 311 volume), by quarter
3. Are 311 requests closed more slowly in high-crime neighborhoods? (raw and mix-adjusted)
4. Do neighborhoods with many streetlight requests see more nighttime outdoor crime? (count and share)
5. Which neighborhoods have high 311 volume but low crime?

Stakeholders named: Department of Public Works, Department of Transportation, Baltimore Police Department / public-safety planners, Mayor's Office, City Council, Neighborhood associations and residents

### Overview
Multi-package Go pipeline (ingest/match/aggregate/report over a shared model) with a generic chunked parallel CSV loader, sharded concurrent join, Strategy-pattern reports, and a compare mode that proves sequential and concurrent results are identical. Code is idiomatic modern Go, thoroughly commented, tested in every package, with CI; the README documents design, measurements and limitations and matches the code exactly.

### Stakeholder value
The five questions are real cross-dataset analyses that control for neighborhood size (baselines, mix-adjusted close ratios, shares), so the Mayor's Office, DPW/DOT and BPD get honest, actionable answers (e.g., no service-speed penalty in high-crime areas once request mix is controlled; a stable 7-neighborhood distress shortlist). Joins on (neighborhood, month) are meaningful, not coincidental counting.

### Issues noted
- No real data bundled; user must run two documented curl commands (verified working today). Program has no fetch mode.
- go.mod requires go 1.27.2, forcing a toolchain download on older Go installs (documented).
- Terminal-only visualization (ASCII tables/bars), no HTML or web UI.
- 29.5% of 311 rows lack a neighborhood and are dropped (documented).

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

Detailed grader report with build logs and captured program output: `grading/abigail_wanjiku.md` in the course grading folder.
