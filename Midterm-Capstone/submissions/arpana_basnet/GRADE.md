# Midterm Capstone Grade — Arpana Basnet

**Raw score: 75/100    Curved grade: 88/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: partial. Datasets found: 2. Questions answered: 5. Visualization: html. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 15 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 15 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 8 | 15 |
| **Raw Total** | **75** | 100 |
| **Curved Grade** | **88** | 100 |

## How this grade was reached

Go. 2 real datasets bundled (311 2025 931k, vacants 11k). Partial concurrency: four workers per dataset on parse, join and Q2-Q5 sequential. 5/5 questions use the join (notices vs 311, dumping, unresolved %, closure time) with honest caveats; separate visualizer builds HTML by scraping printed text. Flat single package, no tests. 1.8 s run.

### Datasets identified
- 311 Customer Service Requests 2025 (data/requests_2025.csv)
- Vacant Building Notices (data/Vacant_Building.csv)

### Concurrency primitives found
goroutines, buffered channels (cap 256) as bounded job queues, per-worker private maps merged by collector, sync.WaitGroup (join.go), sync.Once (service_data.go) for single 311 load; 4 workers per file, both files ingested concurrently

### Design patterns found
none named; de facto pipeline/worker-pool fan-out/fan-in; flat main package with one file per question

### Questions answered
1. Which neighborhoods have the most vacant-building notices, and how many 311 requests do they have?
2. Do neighborhoods with more notices tend to have more 311 requests? (Pearson r)
3. Among the top-10 notice neighborhoods, where are illegal-dumping requests concentrated?
4. How does the unresolved-request percentage compare between <100-notice and >=100-notice neighborhoods?
5. How do average recorded closure times compare between those groups?

Stakeholders named: Mayor's Office, Housing (HCD), Public Works, City Planning

### Overview
Flat single-package Go program (7 files) plus a separate visualize program that renders report.html by re-parsing the engine's printed text. Ingestion uses a reader goroutine feeding 4 aggregating workers per file over bounded channels, with both files loaded concurrently and only per-neighborhood summaries retained (13 MB RSS for 380 MB input). Code is clean and idiomatic with careful data-quality counters, but there are no packages, interfaces, tests, or named design patterns, and the text-scraping visualizer is brittle.

### Stakeholder value
All five questions join vacant notices to 311 requests by neighborhood and are addressed to Housing, Public Works, Planning and the Mayor's Office; the dumping-in-vacant-neighborhoods ranking and the finding that high-vacancy areas do not see slower 311 service are actionable. However Q1 is plain counting, Q4/Q5 collapse 213 neighborhoods into two arbitrary buckets using means on skewed data, and nothing is population-normalized, so insight is modest.

### Issues noted
- No unit tests and no run script; go.mod pins go 1.27.2 which forces a toolchain download
- Join stage is token concurrency (4 goroutines intersecting ~290 names); Q2-Q5 analyses run sequentially
- visualize/main.go parses human-readable results.txt by string prefix instead of structured data
- Q5 uses arithmetic means on heavy-tailed closure times; Q4/Q5 use a coarse two-bucket split
- No design pattern named or realized; question orchestration lives inside Q1's reportTopNotices

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

Detailed grader report with build logs and captured program output: `grading/arpana_basnet.md` in the course grading folder.
