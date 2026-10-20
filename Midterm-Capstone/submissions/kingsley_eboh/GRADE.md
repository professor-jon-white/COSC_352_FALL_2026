# Midterm Capstone Grade — Kingsley Eboh

**Raw score: 67/100    Curved grade: 86/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_fetched_runtime. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 13 | 20 |
| Real Data | 10 | 15 |
| Five Questions | 12 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 9 | 15 |
| **Raw Total** | **67** | 100 |
| **Curved Grade** | **86** | 100 |

## How this grade was reached

Go. 2 real datasets fetched live by default (707k 311 2026 + 12.6k vacant rehabs, 6m47s). Concurrency only in fetch/parse; aggregation sequential. 5 questions, 3 cross-dataset (Q2/Q3 single-dataset, Q5 restates the Spearman). Single untested package. Two ASCII charts, Markdown report.

### Datasets identified
- 311 Customer Service Requests 2026 (ArcGIS FeatureServer, paged GeoJSON, fetched at runtime: 706,938 records)
- Vacant Building Rehabs (Open Baltimore GeoJSON download, fetched at runtime: 12,647 records)

### Concurrency primitives found
one fetcher goroutine per source with sync.WaitGroup (pipeline.go:38-47); bounded jobs/parsed channels (workers*8) feeding N parser goroutines, WaitGroup closes parsed (pipeline.go:34-61); single collector goroutine aggregates (pipeline.go:62-65); context.WithTimeout + select on ctx.Done() for every send (pipeline.go:110-114,152-156,213-217)

### Design patterns found
Producer/worker/collector Pipeline; Source configuration object selecting live / -offline / -from-raw providers; no Strategy/Factory/Builder interfaces

### Questions answered
1. Which neighborhoods have the strongest combined signal of 2026 311 request volume and vacant-building rehab locations?
2. Where are the largest concentrations of vacant-building rehab locations?
3. Which neighborhoods have the most 311 requests whose recorded status is not closed/complete/resolved/cancelled?
4. Which 311 service request types are relatively more common in high-rehab neighborhoods?
5. Do neighborhoods rank similarly by 311 request volume and vacant-building rehab locations? (Spearman)

Stakeholders named: Mayor's Office, Department of Housing and Community Development

### Overview
Four-file single-package Go CLI whose default mode pages the live ArcGIS 311 FeatureServer (2,000 records/page) and downloads the Vacant Building Rehabs GeoJSON concurrently, saves raw pages to data/raw, parses features in a bounded worker pool, reduces them to per-neighborhood counters in one collector, and writes Markdown/CSV/JSON reports plus a terminal dashboard. Streaming design is excellent (53 MB RSS for 707k records) and race-clean, but there are no tests, no interfaces, a dead identity function, and dense one-line formatting.

### Stakeholder value
Q4 is a real cross-dataset finding DHCD could use: SW-Boarding (3.8x), SW-Cleaning (3.7x), HCD-Vacant Building (2.7x) and Dirty Alley (2.5x) requests are enriched in the top-quartile rehab neighborhoods (Broadway East, McElderry Park, Oliver, Belair-Edison), and Spearman 0.77 shows rehab permits and 311 load co-locate. Q2 and Q3 are single-dataset top-5 lists, Q5 restates the summary statistic, and the 'requests per 100 rehab locations' column is meaningless where rehabs are rare (Downtown 41,781.8); nothing is normalised per capita.

### Issues noted
- No tests (go test: no test files).
- Concurrency limited to fetch and parse; aggregation and analysis are sequential in one collector.
- Q2 and Q3 are single-dataset; Q5 duplicates the executive summary; 'requests per 100 rehab locations' ratio is not meaningful.
- Default live run took 6m47s and downloaded ~600 MB of raw pages; README does not warn about duration or size; bundled excerpts (250 features each) are too small to analyse.
- Dead code (cleanNeighborhood identity), stats and file I/O mixed in analysis.go, dense formatting.
- run-macos.sh not executable in archive (README does instruct chmod +x).
- Bundled reports/ are stale outputs from the student's 2026-10-11 run and are overwritten on each run.

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

Detailed grader report with build logs and captured program output: `grading/kingsley_eboh.md` in the course grading folder.
