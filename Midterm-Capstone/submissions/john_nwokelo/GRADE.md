# Midterm Capstone Grade — John Nwokelo

**Raw score: 74/100    Curved grade: 88/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 16 | 20 |
| Real Data | 12 | 15 |
| Five Questions | 9 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 14 | 15 |
| **Raw Total** | **74** | 100 |
| **Curved Grade** | **88** | 100 |

## How this grade was reached

Go. 2 real datasets (812 MB 311 2026 GeoJSON + NSA boundaries), uploaded separately; bundled samples are synthetic and the default. Full concurrency: parallel point-in-polygon join and aggregation. 5 questions but all five are 311-only statistics; boundaries serve only as a lookup key. 20 s run at 2.26 GB RSS (whole-file unmarshal). Clean multi-package layout, 7 tests.

### Datasets identified
- 311 Customer Service Requests 2026 (GeoJSON, 812 MB, submitted separately)
- Neighborhood Statistical Area (NSA) Boundaries (GeoJSON, 279 polygons, submitted separately)

### Concurrency primitives found
two loader goroutines + sync.WaitGroup (cmd/engine/main.go:30-34); worker-pool point-in-polygon join over bounded jobs/results channels with context cancellation and WaitGroup-closed results (internal/processing/joiner.go:46-92); segmented parallel aggregation with per-goroutine maps merged via channel (internal/analysis/analyzer.go:56-111)

### Design patterns found
Pipeline (load -> spatial join -> analyze -> report) across internal packages; fan-out/fan-in worker pool; no Strategy/Factory interfaces

### Questions answered
1. Which neighborhoods report the most problems?
2. What problems are most common in each neighborhood?
3. Where is the highest percentage of problems still unresolved?
4. Where does it take the longest to fix problems?
5. Which neighborhoods need the most attention? (40% volume + 40% unresolved rate + 20% mean closure time)

Stakeholders named: city officials (unnamed)

### Overview
Idiomatic multi-package Go (cmd/engine + internal/ingestion, processing, analysis, output, models) with tests in every stage package. Loads both GeoJSONs concurrently, assigns 311 points to NSA polygons with a bounded worker pool (ray casting with holes, bbox prefilter), aggregates in parallel segments, and prints five ranked tables with # bars. Compact and readable, but the loader reads and unmarshals the whole 812 MB file at once.

### Stakeholder value
On the real data it produces plausible, cross-validated rankings (Belair-Edison 24.8k requests, campus neighborhoods with the highest open rates, Curtis Bay Industrial 25-day closures). However every question is a 311-only statistic; the boundary dataset merely supplies a grouping key the 311 file already has, and the NSA layer's population/vacancy attributes go unused, so there is no genuine cross-dataset insight and stakeholders are not named.

### Issues noted
- All five questions are single-dataset 311 aggregates; NSA boundaries used only as a spatial lookup key; no named stakeholders.
- Entire 812 MB GeoJSON read with os.ReadFile and fully unmarshalled: 2.26 GB RSS, no streaming decoder.
- Synthetic sample GeoJSON is the default flag path (rubric -3); real data was uploaded separately and had to be symlinked into data/ per README.
- Parsing is not parallel (one goroutine per file); ~235k null-geometry 311 records dropped without being reported.
- run-windows.cmd contains leftover text referencing 'the old project'.
- Race detector not run on the full dataset (memory risk); clean on tests and sample.

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

Detailed grader report with build logs and captured program output: `grading/john_nwokelo.md` in the course grading folder.
