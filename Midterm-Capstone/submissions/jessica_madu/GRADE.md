# Midterm Capstone Grade — Jessica Madu

**Raw score: 71/100    Curved grade: 87/100**

Language: Go. Builds: yes. Runs: yes. Data status: simulated. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 17 | 20 |
| Real Data | 5 | 15 |
| Five Questions | 17 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 12 | 15 |
| Project Structure | 12 | 15 |
| **Raw Total** | **71** | 100 |
| **Curved Grade** | **87** | 100 |

## How this grade was reached

Go. No real data shipped; default paths point at gen_sample.py output in the real schema with uniform random values, so findings are artifacts (r = 0.99). Strong pipeline: streaming, worker pools, MemStats, race-clean, 3 tests. 5/5 cross-dataset questions with named stakeholders and ASCII charts.

### Datasets identified
- Part 1 Crime Data (CSV) - declared, not shipped
- 311 Customer Service Requests (CSV) - declared, not shipped

### Concurrency primitives found
goroutine per dataset returning result channel (main.go:22-30); per-dataset worker pool of N parser goroutines over bounded jobs channel (ingest.go:71-91); sync.WaitGroup closes results (ingest.go:92); reader goroutine streams batches (ingest.go:95-116); per-worker private Agg merged by one goroutine (ingest.go:118-121, model.go:72-91)

### Design patterns found
Pipeline (Ingest reader->workers->merge), Strategy (Dataset interface, Question interface), Factory (NewCrimeDataset/New311Dataset)

### Questions answered
1. Mayor's Office: Which neighborhoods carry the most crime, and what does their 311 picture look like?
2. Public Works: Where is crime high but 311 engagement low (possible service gap / under-reporting)?
3. City Planning: Do neighborhoods with more crime also generate more 311 requests? (Pearson r, quartiles)
4. Transportation/DPW: Which 311 request types are over-represented in the highest-crime neighborhoods? (lift)
5. Operations: When do crime and 311 demand peak (by month and time of day)?

Stakeholders named: Mayor's Office, Public Works, City Planning, Transportation/DPW, Operations

### Overview
Single-package Go CLI with a clean reader->bounded-channel->worker-pool->merge pipeline, table-driven Dataset strategies with header aliases, and a Question strategy interface for five analyses rendered as ASCII tables/bars. Code is concise, well commented, consistent with its README, race-clean and covered by three ingestion tests. Memory design (streaming, confinement, O(groups) aggregates, MemStats) is explicit and verified.

### Stakeholder value
The question set (hotspots with 311 context, service-gap ratio, correlation, request-type lift, timing) is well framed and would be actionable on real data; report_real_data.txt hints at real findings (Inner Harbor under-reporting, curb-repair lift). But the submitted run uses uniformly random synthetic data, so every printed finding is an artefact (r=0.992, flat seasonality) and a stakeholder learns nothing from the shipped artefact.

### Issues noted
- No real Open Baltimore data in the submission; only scripts/gen_sample.py output (Location='X', District='CENTRAL', 20 neighborhoods).
- Synthetic sample files are the default -crime/-requests paths (rubric deduction).
- report_real_data.txt asserts real-data results but cannot be reproduced from the submission; program has no download step.
- Everything in one package; question logic has no tests.
- Crime (2024) and 311 (2025) cover different years (acknowledged); counts not per-capita.

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

Detailed grader report with build logs and captured program output: `grading/jessica_madu.md` in the course grading folder.
