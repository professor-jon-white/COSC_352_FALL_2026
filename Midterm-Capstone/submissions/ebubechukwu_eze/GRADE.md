# Midterm Capstone Grade — Ebubechukwu Eze

**Raw score: 100/100    Curved grade: 100/100**

Language: Rust. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 5. Questions answered: 5. Visualization: web_ui. Tests: pass. Race detector: not_run.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 20 | 20 |
| Three Datasets | 15 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 15 | 15 |
| **Raw Total** | **100** | 100 |
| **Curved Grade** | **100** | 100 |

## How this grade was reached

Rust. 5 real Open Baltimore datasets bundled (NIBRS crime 276k, 311 2025 931k, vacant notices 11k, liquor 31k, CitiWatch 861) plus fetch script. Full multi-stage concurrency: threaded ingest, parallel match/aggregate/analyze, deterministic across thread counts. 5/5 cross-dataset questions, all answered with ASCII, HTML, JSON and a web server. 28 tests, clippy clean, 0.9 s run. Top of class.

### Datasets identified
- NIBRS Group A Crime 2022-present (data/crime_nibrs.csv)
- 311 Customer Service Requests 2025 (data/service_requests_311_2025.csv)
- Vacant Building Notices (data/vacant_building_notices.csv)
- Liquor Licenses (data/liquor_licenses.csv)
- CitiWatch CCTV Cameras GeoJSON (data/citiwatch_cameras.geojson)

### Concurrency primitives found
std::thread::scope (5 files concurrently, borrowed interners); per-file reader thread -> crossbeam_channel::bounded(parsers*2) -> N parser threads with buffer-pool return channel; RwLock<HashMap> interner; rayon par_iter_mut/fold/reduce/join for matching and vote tallies; rayon par_chunks map-reduce for neighborhood aggregates; Vec<AtomicU32> fetch_add for block counters; par_sort_unstable; five questions via par_iter over Box<dyn Question> in a private rayon ThreadPool; deterministic output verified 1 vs 10 threads; no unsafe

### Design patterns found
Builder (PipelineBuilder::build validates once), Pipeline (4 stages passing ownership), Strategy (MatchStrategy, Question, Renderer, CsvSchema traits), Factory (report::renderer(format), analysis::questions(ids)); all realized in code

### Questions answered
1. Where do vacant buildings and violent crime overlap, and how strongly does one track the other? (Mayor's Office, HCD; neighborhood + block joins)
2. Do blocks with liquor outlets see more violent crime than comparable blocks, and which licensed blocks stand out? (Liquor Board, BPD; block-address join)
3. How much violent crime happens beyond the reach of the CitiWatch camera network, and where would new cameras see the most? (BPD, MONSE; 150 m spatial join + city map)
4. Where is resident-reported blight concentrated, and do high-vacancy neighborhoods wait longer for it to be resolved? (DPW, HCD; 311 x vacants)
5. Taking violence, vacancy, blight, camera coverage and liquor outlets together, which neighborhoods need coordinated investment first? (Mayor's Office, Planning; all five datasets)

Stakeholders named: Mayor's Office, Department of Housing & Community Development, Board of Liquor License Commissioners, Baltimore Police Department, Mayor's Office of Neighborhood Safety and Engagement, Department of Public Works, Department of Planning

### Overview
A 4,330-line Rust crate organized as a four-stage pipeline (ingest -> match -> aggregate -> analyse) with a validated PipelineBuilder, Strategy traits for schemas, matching, questions and renderers, and factories for the latter two. Ingestion streams 4 MiB quote-aware chunks through bounded crossbeam channels into parser threads that emit compact Copy records with interned ids; matching recovers missing neighborhoods via spatial cells and hundred-block address keys; rayon map-reduce and atomics build the aggregates; questions run in parallel and render to terminal, HTML (SVG maps), JSON and an optional localhost server. Clippy-clean, no unsafe, 28 tests including an end-to-end hand-built city, and a README whose every claim verified.

### Stakeholder value
The five questions are genuinely cross-dataset and use three different join grains (neighborhood name, hundred-block address, 150 m spatial radius), yielding specific, actionable outputs: outlet blocks carry 17.6% of violence on 5.5% of blocks even within the same neighborhood, 41.5% of violent crime is out of camera range with ten named candidate sites, high-vacancy neighborhoods are not served slower on blight requests (an equity check with a clear answer), and a weighted priority index assigns a lead agency per neighborhood. Methods and limits (raw counts, correlation not causation, mismatched data periods) are stated in each section.

### Issues noted
- Datasets cover different periods (crime 2022-2026, 311 2025 only, vacant notices back to 2010) and nothing is per-resident normalized; disclosed but affects Q1/Q5 comparability
- Hand-rolled DST/calendar and map-projection code instead of established crates; well-tested but more to maintain
- --serve is a one-line HTTP parser serving a single in-memory page (fine for the assignment, not production)
- First cargo build requires network to fetch crates; no run.sh (README relies on cargo run --release)
- Q5 index weights are an undocumented-sensitivity judgment call

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

Detailed grader report with build logs and captured program output: `grading/ebubechukwu_eze.md` in the course grading folder.
