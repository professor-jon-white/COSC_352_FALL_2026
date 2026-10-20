# Midterm Capstone Grade — Anaya Efferson

**Raw score: 63/100    Curved grade: 85/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 11 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 8 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 7 | 15 |
| **Raw Total** | **63** | 100 |
| **Curved Grade** | **85** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (311 2025 931k, vacants 11k). Partial concurrency: worker pools do trivial per-row counting, parsing sequential. 5 questions, 3 cross-dataset (Q1/Q2 single-dataset top-5 lists; Q4 degenerate ratio 5963:1). No stakeholders, no tests, analysis inline in main(). One ASCII chart. 1.2 s run.

### Datasets identified
- 311 Customer Service Requests 2025 (data/service_requests.csv)
- Vacant Building Notices (data/vacant_properties.csv)

### Concurrency primitives found
goroutines, bounded buffered channels, sync.WaitGroup; both datasets processed concurrently (main.go:129-140); per-dataset reader goroutine streams rows into a workers*2 channel consumed by NumCPU workers with private count maps, merged after wg.Wait (processing.go:10-74); worker pool over ~280 neighborhood names for the join (processing.go:78-148)

### Design patterns found
README claims Pipeline; realized only loosely as stream -> worker pool -> merge -> join -> print, with all five analyses inline in main()

### Questions answered
1. Which neighborhoods have the most 311 service requests? (single dataset)
2. Which neighborhoods have the most vacant property records? (single dataset)
3. What is the correlation between neighborhood 311 counts and vacant property counts?
4. Which neighborhoods have the highest ratio of 311 requests to vacant property records?
5. Which neighborhoods should city officials prioritize (50/50 normalized 311 + vacant score)?

Stakeholders named: (none specific; 'city officials' / 'municipal leaders')

### Overview
Three-file single-package Go program: ingestion.go streams CSV neighborhoods into a channel, processing.go runs a worker pool that counts them and a second pool that joins the two count maps, and main.go computes and prints five questions inline. Clean and readable with deterministic ordering, but no tests, no realized design pattern beyond a loose pipeline, and analysis mixed with output.

### Stakeholder value
Only the correlation and the 50/50 priority score genuinely combine the datasets; the top-5 lists restate what each source dataset already shows and the requests-per-vacant ratio is dominated by neighborhoods with a single notice. The priority shortlist (Broadway East, Carrollton Ridge, Sandtown-Winchester) is plausible but unnormalized and unfiltered, so a planning office gains little beyond what the vacant-notice counts alone provide.

### Issues noted
- Two of five questions are single-dataset top-5 lists; Q4 ratio is degenerate (Riverside 5963:1 from one vacant notice).
- No stakeholders named; no tests.
- Worker pool does trivial per-row counting; CSV parsing is sequential per file; join pool is token concurrency.
- All five analyses and output live inline in main(); README's Pipeline pattern is only loosely realized.
- Prebuilt macOS binary shipped without execute permission and an executable copy was SIGKILLed by macOS; not verifiable (source build graded).
- go.mod go 1.27.2 forces a toolchain download on older Go installs.

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

Detailed grader report with build logs and captured program output: `grading/anaya_efferson.md` in the course grading folder.
