# Midterm Capstone Grade — Aidan James

**Raw score: 100/100    Curved grade: 100/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 4. Questions answered: 5. Visualization: web_ui. Tests: pass. Race detector: clean.

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

Go. 4 real datasets bundled (crime, 311, liquor, vacant GeoJSON). Chunked parallel CSV decode, sharded map-reduce join, parallel analysis strategies, race-clean. 5/5 cross-dataset questions incl. a two-hop council-district scorecard; ASCII + HTML + token-protected web UI verified. 3.3 s run. Only nit: prebuilt .app would not launch from sandbox.

### Datasets identified
- NIBRS Group A Crime Data 2025 (crime_2025.csv.gz)
- 311 Customer Service Requests 2025 (311_requests_2025.csv.gz)
- Liquor Licenses (liquor_licenses.csv.gz)
- Vacant Building Notices (vacant_building_notices.geojson.gz, GeoJSON)

### Concurrency primitives found
goroutines, bounded channels (workers*2), sync.WaitGroup, sync.Pool, sync.RWMutex (Interner), sync/atomic, context cancellation; 4 datasets loaded in parallel (ingest/load.go), chunked parallel CSV decode with quote-parity splitter and ordered collector (ingest/pipeline.go), row-batch pipeline for GeoJSON, sharded map-reduce join index (index/index.go), 5 analyses run concurrently (analysis/analysis.go), concurrent paged HTTP fetcher (fetch/fetch.go)

### Design patterns found
Pipeline (ingest->index->analyze->render; channel pipeline inside ingest), Factory (ingest.Open for CSV/GeoJSON/gzip, report.NewRenderer), Strategy (ingest.Source, ParseFunc, analysis.Analysis, report.Renderer), Builder (engine.NewBuilder().DataDir().Workers().Output().Build())

### Questions answered
1. Do neighborhoods/blocks with more open vacant-building notices experience more violent crime; which neighborhoods should be first for coordinated demolition + public-safety investment? (Vacants x Crime)
2. Do liquor-licensed blocks see more crime; which license types/outlets; is it a late-night problem? (Liquor x Crime by hundred-block)
3. Are residents in the highest-crime neighborhoods getting slower or less reliable 311 service? (311 x Crime)
4. How much resident-reported blight comes from blocks with vacant buildings; is there a dose-response? (311 x Vacants by block)
5. Which council districts carry the heaviest combined burden for the next capital budget? (Crime x Vacants x 311 x Liquor, two-hop joins)

Stakeholders named: Mayor's Office, Dept. of Housing and Community Development (DHCD), Board of Liquor License Commissioners, Baltimore Police Department, Mayor's Office of Performance & Innovation (CitiStat), Dept. of Public Works (Solid Waste), DHCD Code Enforcement, City Council, Dept. of Planning

### Overview
Nine-package Go engine with a Source factory (CSV/GeoJSON/gzip), chunked parallel CSV decoding, sharded map-reduce join on normalized neighborhood and hundred-block keys, Strategy-based analyses returning a format-neutral report model, and terminal/plain/HTML renderers plus a token-protected local web dashboard and an optional cgo macOS app. Code is idiomatic, well-commented, tested (incl. determinism and chunked-vs-sequential equivalence), and the README matches it in detail; scope is far beyond a typical midterm.

### Stakeholder value
The five questions join datasets at block and neighborhood level and produce decision-relevant findings for named agencies: a vacant-building shortlist for DHCD/BPD, a liquor-outlet watch-list with evidence that closing hours are not the lever, an honest mixed-signal 311 equity result for CitiStat, a block-level blight dose-response for DPW crews, and a four-dataset council-district scorecard with caveats. Joins are substantive (address normalization, majority-vote district assignment), not coincidental counting.

### Issues noted
- Prebuilt macOS .app (second submission folder) could not be verified from the grading shell: quarantined GUI bundle produced no window or listening port; source-built equivalent and embedded UI server work.
- go build ./... prints harmless ld warnings for the cgo native-app package (macOS 14.4 objects linked for 13.0 target).
- Gzipped 311 ingest is single-stream DEFLATE-bound (2.0 s of 3.3 s); README documents ./run.sh unpack to remove the bottleneck.
- Only 66% of 311 rows yield a block key (no address on call-center requests); documented.
- Binary must be run from repo root or with -data (run.sh handles it).

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

Detailed grader report with build logs and captured program output: `grading/aidan_james.md` in the course grading folder.
