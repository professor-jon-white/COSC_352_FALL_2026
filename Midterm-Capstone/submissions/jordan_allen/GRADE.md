# Midterm Capstone Grade — Jordan Allen

**Raw score: 88/100    Curved grade: 93/100**    (submitted LATE; noted, not deducted)

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 20 | 20 |
| Real Data | 13 | 15 |
| Five Questions | 19 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 15 | 15 |
| Project Structure | 13 | 15 |
| **Raw Total** | **88** | 100 |
| **Curved Grade** | **93** | 100 |

## How this grade was reached

Go, LATE. 2 real datasets (NIBRS + 311) but only a one-week June 2025 excerpt (1,137 crimes / 1,540 requests). Full concurrency: worker pools per dataset, 5 questions run as goroutines, streaming with column projection, 14 tests. 5/5 cross-dataset questions with named stakeholders and dual histograms. Every README number reproduced. Lost points only on dataset count and data scale.

### Datasets identified
- NIBRS Group A Crime Data (CSV excerpt, 1-7 June 2025, all rows with neighborhood)
- 311 Customer Service Requests 2025 (CSV excerpt, every 11th request, same week)

### Concurrency primitives found
two dataset goroutines + sync.WaitGroup (main.go:162-176); per dataset: reader goroutine -> bounded batches channel (-queue) -> N worker goroutines each owning a private Aggregator, WaitGroup then Merge, reader error via buffered errc (ingest.go:194-245); five Question goroutines over read-only Joined with per-goroutine recover (main.go:196-213); no locks (confinement)

### Design patterns found
Pipeline (runPipeline), Factory (crimeSpec/requestSpec/newCrimeAgg/newRequestAgg), Strategy (Aggregator, Question, Renderer interfaces each with 2+ implementations), Builder (ReportBuilder)

### Questions answered
1. Mayor's Office: Which neighborhoods rank highest on BOTH crime and 311 service demand, and how closely do the two track each other city-wide?
2. Department of Public Works: Which 311 request types are over-represented in the highest-crime neighborhoods compared with the rest of the city?
3. Police Department and Transportation (staffing): How do the hour-of-day profiles of crime and 311 requests compare, and where do they diverge?
4. Mayor's Office (service equity): Is the time to close a 311 request longer in the highest-crime neighborhoods than in the lowest-crime ones?
5. City Planning: Do neighborhoods with more physical-disorder 311 requests also have more violent crime?

Stakeholders named: Mayor's Office, Department of Public Works, Police Department and Transportation, Mayor's Office (service equity), City Planning

### Overview
Single-package Go CLI in ten single-responsibility files: streaming CSV reader with column projection and two-allocation batches feeding a bounded worker pool per dataset, per-worker aggregators merged after wg.Wait, a quartile-labelled neighborhood join, five Question strategies run concurrently, and a ReportBuilder rendered by Text and Markdown renderers. Careful statistics (tied-rank Spearman, NaN handling), 14 tests, race-clean, MemStats reporting; README is detailed and candid (Windows-only testing, Claude assistance).

### Stakeholder value
Each question stratifies one dataset by the other (crime quartiles vs 311 type lift and closure time; disorder quartiles vs violent crime; joint ranks; hourly profiles) and ends with plain-language key findings, so the Mayor's Office, DPW, Police/DOT and Planning would each get something actionable. The bundled data is a single week with a 1/11 sample of 311, so the specific numbers are illustrative; the program is ready for the full downloads but they are not included.

### Issues noted
- LATE submission (flagged per rubric, not deducted).
- Bundled data is a one-week excerpt (1,137 crimes, 1,540 311 requests); small-n findings such as a 6.22x lift on 8 requests are fragile; full files must be downloaded for citywide conclusions.
- Student could not test on macOS (Windows only); it did build and run identically here.
- Single Go package; questions.go is 544 lines.
- Violent/disorder categories are keyword heuristics (configurable and disclosed).
- README discloses Claude assistance in writing and debugging.

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

Detailed grader report with build logs and captured program output: `grading/jordan_allen.md` in the course grading folder.
