# Midterm Capstone Grade — Xavier Jones

**Raw score: 76/100    Curved grade: 88/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 15 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 13 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 10 | 15 |
| **Raw Total** | **76** | 100 |
| **Curved Grade** | **88** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (NIBRS 12k + arrests 4k, Jan-Mar 2025) fetched by its own ArcGIS client with manifest. Partial concurrency: parallel fetch/parse and one aggregation pool; cross-tabs sequential. 5/5 questions compare the two datasets but stakeholders are generic purposes. Markdown report reproduces shipped output. 1 s run.

### Datasets identified
- NIBRS Group A Crime Data (Open Baltimore ArcGIS FeatureServer, 2025-01-01 to 2025-03-31, data/live/crime.json)
- BPD Arrests (Open Baltimore / baltegis GeoSpatialized_Tables/Arrest FeatureServer, same window, data/live/arrests.json)

### Concurrency primitives found
2 goroutines fetch both ArcGIS sources with context cancellation (ingest.go:150-170); 2 goroutines decode both JSON files (main.go:60-70); crime aggregation worker pool: bounded jobs channel (2*workers), N workers with private partial Result maps, partials channel, feeder goroutine + WaitGroup, sequential reducer (analysis.go:36-83); arrest pre-pass, cross-tabs and incident index are sequential

### Design patterns found
Pipeline (ingest -> normalize -> analyze -> report, one file per stage) with map-reduce style worker pool; no Builder/Strategy/Factory

### Questions answered
1. Which neighborhoods have the largest recorded crime workload, and how does arrest activity compare?
2. How do monthly crime and arrest workloads change?
3. What offense mix is recorded in the five neighborhoods with the most arrest records?
4. Where are crime-record and arrest-record shares most different by neighborhood?
5. What weapon profile and arrest workload occur in the five neighborhoods with the most crime records?

Stakeholders named: citywide coordination (unnamed), staffing review (unnamed), investigative resource planning (unnamed), geographic workload distribution review (unnamed), violence-prevention program planning (unnamed)

### Overview
Compact single-package Go organized by pipeline stage (ingest, normalize, analysis, report) with its own paginated ArcGIS fetcher, provenance manifest, schema checks and httptest-backed tests. Rows are loosely typed map[string]any decoded fully into memory; the report is one dense function and variable names are terse, but the README is careful, accurate and heavily caveated.

### Stakeholder value
All five questions are aggregate crime-by-arrest joins by neighborhood or month, honestly qualified by the fact that 69% of arrest rows lack a neighborhood. Q4 (Downtown holds 3.6% of crime but 8.3% of geocoded arrests) is the one finding a leader would act on; Q2 is a plain monthly count and Q3/Q5 mix differences are small. Purposes are stated but no municipal office is named, and visualization is one ASCII chart plus tables.

### Issues noted
- Stakeholders are generic purposes ('for staffing review') rather than named municipal offices
- Concurrency covers I/O and one aggregation stage only; arrest aggregation, cross-tabs and incident index are sequential passes
- Rows are map[string]any and both JSON files are decoded fully into memory (98 MB RSS for 12 MB input)
- Only one ASCII chart; remaining output is Markdown tables
- 69% of arrest rows have no neighborhood so most answers describe a 31% subset (disclosed)
- README is Windows-first (macOS section present and correct); run.sh writes binary and report into the project folder

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

Detailed grader report with build logs and captured program output: `grading/xavier_jones.md` in the course grading folder.
