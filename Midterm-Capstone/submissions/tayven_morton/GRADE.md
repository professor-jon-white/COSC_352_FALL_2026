# Midterm Capstone Grade — Tayven Morton

**Raw score: 86/100    Curved grade: 92/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 15 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 14 | 15 |
| **Raw Total** | **86** | 100 |
| **Curved Grade** | **92** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (311 Oct-Dec 2025 244k, vacants 11k) with SHA-256 manifest and reproducible fetch tool. Full concurrency, verify.sh proves 1 vs 4 workers byte-identical. 5/5 questions use the join but Q2-Q4 are 311 measures stratified by vacancy cohort; one ASCII chart, otherwise Markdown tables. README carries leftover AI-assistant notes.

### Datasets identified
- 311 Customer Service Requests 2025 (services1.arcgis.com/.../311_Customer_Service_Requests_2025/FeatureServer/0), Oct 1 - Dec 31 2025
- Vacant Building Notice - Open (egisdata.baltimorecity.gov/.../DHCD_Open_Baltimore_Datasets/FeatureServer/1)

### Concurrency primitives found
Both datasets loaded in parallel goroutines + WaitGroup (cmd/analyzer/main.go:40-50); per file: scanner goroutine -> bounded jobs chan gated by a credits chan (workers*8) -> N json.Unmarshal parser goroutines -> sequence-restoring dispatcher -> FNV-hashed shard channels (cap 32) each owned by one aggregation goroutine with a private map, merged at end (internal/ingest/ingest.go:40-158); join worker pool over immutable maps with sorted output (internal/analysis/analysis.go:21-65); runtime.GOMAXPROCS(workers); concurrent downloads in cmd/fetch

### Design patterns found
Pipeline (bounded channel stages: read -> parse -> ordered validate/dedup -> sharded aggregate -> parallel join -> analyses -> report) across cmd/fetch, cmd/analyzer, internal/model, internal/ingest, internal/analysis, internal/report

### Questions answered
1. Which neighborhoods have high counts on both vacancy properties and 311 requests (>=75th midrank percentile on both)?
2. How does the mix of request types differ between low- and high-vacancy-count neighborhoods?
3. How do median closure times for the same request types differ between those cohorts?
4. How do monthly requests change across the selected quarter in each cohort?
5. Which five neighborhoods rank highest under a transparent 50/50 percentile score, and what is their leading request category?

Stakeholders named: Joint housing and city-services review team (generic; DHCD / DPW / 311 agencies implied)

### Overview
A compact, idiomatic Go pipeline in five packages that streams two bundled raw ArcGIS JSONL extracts through credit-bounded parsing pools, an order-restoring dispatcher that validates and deduplicates, FNV-sharded owned-map aggregation, and a parallel neighborhood join, then writes a Markdown report and CSV. Everything is deterministic across worker counts (verified byte-for-byte by verify.sh), covered by 16 tests including a 10k-record stress fixture and a mocked downloader, and documented honestly, with explicit data-quality limits. The only structural weakness is a long report.Write that mixes some analysis thresholds with formatting.

### Stakeholder value
The results are real and partly actionable: a credible priority list (Broadway East, Carrollton Ridge, Sandtown-Winchester, Central Park Heights, Harlem Park) and a specific lead that parking complaints take 22 days to close in high-vacancy neighborhoods versus 10 days in low-vacancy ones. Value is limited by the generic stakeholder, three questions that are 311 measures stratified by vacancy cohort rather than two-sided joins, a shallow Q4, and minimal visualization (one ASCII bar block, otherwise tables).

### Issues noted
- Visualization is minimal: Markdown tables and one ASCII bar chart; no HTML or web UI
- Stakeholder is a generic 'joint housing and city-services review team'; no office named per question
- Q2-Q4 are single-dataset 311 measures stratified by a vacancy-derived cohort; Q4 (both cohorts -14% Oct->Dec) is shallow
- report.Write is a single 155-line function that mixes analysis thresholds with formatting
- README and VERIFICATION.md contain leftover assistant-style notes addressed to 'the user' (e.g. 'No Canvas submission has been performed', 'review any course rules concerning outside assistance')
- 30.8% of raw 311 rows lack a neighborhood and are excluded (documented; inherent to the source)

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

Detailed grader report with build logs and captured program output: `grading/tayven_morton.md` in the course grading folder.
