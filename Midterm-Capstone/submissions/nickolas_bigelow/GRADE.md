# Midterm Capstone Grade — Nickolas Bigelow

**Raw score: 88/100    Curved grade: 93/100**

Language: Go. Builds: yes. Runs: partial. Data status: real_fetched_runtime. Concurrency level: full. Datasets found: 4. Questions answered: 5. Visualization: html. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 9 | 15 |
| Five Questions | 16 | 20 |
| Three Datasets | 15 | 15 |
| Runs Well + Documented | 13 | 15 |
| Project Structure | 15 | 15 |
| **Raw Total** | **88** | 100 |
| **Curved Grade** | **93** | 100 |

## How this grade was reached

Go. 4 datasets fetched via `make fetch` (crime, 311, liquor, vacants) but nothing bundled; documented `make demo` and the submitted report run on a synthetic generator in exact Open Baltimore schemas. Full concurrency, 11 packages, 25 tests, race-clean. 5/5 cross-dataset questions; on real data 3 answered well, Q2 unavailable (license layer had no coordinates), Q5 half-degenerate.

### Datasets identified
- Part 1 Crime Data (Part1_Crime_Beta/FeatureServer/0)
- 311 Customer Service Requests 2022 (311_Customer_Service_Requests_Yearly/FeatureServer/0)
- Liquor Licenses (CityView/Licenses/FeatureServer/0)
- Vacant Building Notices (DHCD_Open_Baltimore_Datasets/FeatureServer/1)

### Concurrency primitives found
goroutine per dataset + WaitGroup; reader->bounded batch channel->N parser goroutines->ordered collector (ingest/csvstream.go:129-222); generic ParallelReduce/ParallelFor with private accumulators (join/parallel.go); 3 spatial grids built in parallel and 3 joins in parallel (join/join.go:46-125); 5 analyses concurrent with recover (analysis.go:36-55); fetch worker pool + single writer goroutine with context cancellation (fetch.go:191-273); RWMutex interner; atomic memory sampler; sync.Pool

### Design patterns found
Pipeline (pipeline.Run stages), Builder (pipeline.Builder with validation), Factory (ingest.NewSource, report.NewRenderer), Strategy (analysis.Question, report.Renderer)

### Questions answered
1. Which neighborhoods face compounded distress across public safety, housing and city services? (Crime+Vacants+311+Liquor)
2. Is violent crime concentrated around liquor outlets, and which outlets stand out? (Liquor+Crime+311 baseline, spatial)
3. Do vacant buildings drive trash, dumping and rodent complaints, and where is demand out of line with vacancy? (Vacants+311)
4. Are 311 requests from high-crime neighborhoods resolved as quickly as elsewhere? (311+Crime, mix-adjusted)
5. When, and around which places, should late-night patrol and outreach be concentrated? (Crime+Liquor)

Stakeholders named: Mayor's Office, Board of Liquor License Commissioners & Baltimore Police Department, Department of Public Works & Department of Housing and Community Development, Department of Transportation & 311 service agencies, BPD Patrol Command & Mayor's Office of Neighborhood Safety and Engagement

### Overview
A polished standard-library Go pipeline in 11 internal packages (ingest, join, spatial, stats, analysis, report, pipeline, fetch, sample, model) driven by a Makefile and a Builder-validated CLI. Parsing streams CSV through batched worker pools with ordered collection, joins use a spatial grid and generic sharded reducers, five Question strategies run concurrently, and a renderer-neutral report tree is emitted to terminal, Markdown, HTML and CSV. Code is idiomatic, well-commented and covered by 25 tests including determinism and end-to-end checks; the README matches the code exactly.

### Stakeholder value
On the synthetic default the findings merely recover the generator's planted relationships. On fetched real 2022 data, Q1 (composite distress), Q3 (vacancy vs sanitation residuals) and Q4 (mix-adjusted 311 wait by crime tercile, per-agency gaps) give the Mayor's Office, DPW/DHCD and DOT genuinely actionable rankings; Q2 fails and Q5's outlet comparison is empty because the live Liquor Licenses layer carries no coordinates through this fetcher.

### Issues noted
- No real data bundled; documented Quick Start (make demo) and the submitted report.html use a synthetic generator with planted relationships (clearly labeled)
- Live fetch of Liquor Licenses returns blank X/Y (and 12,499 of 30,953 IDs), so all licenses are dropped as no_location; Q2 returns an error section and Q5's near-outlet share is 0.0% on real data
- Real-data run yields 3 of 5 answers cleanly
- 311 is sampled to 150k of ~1M rows by default (documented)
- No unit tests for analysis, report, sample packages

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

Detailed grader report with build logs and captured program output: `grading/nickolas_bigelow.md` in the course grading folder.
