# Midterm Capstone Grade — Nicholas Ighade

**Raw score: 11/100    Curved grade: 70/100**    (submitted LATE; noted, not deducted)

Language: Python. Builds: no. Runs: no. Data status: simulated. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: not_run.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 4 | 20 |
| Real Data | 0 | 15 |
| Five Questions | 4 | 20 |
| Three Datasets | 0 | 15 |
| Runs Well + Documented | 0 | 15 |
| Project Structure | 3 | 15 |
| **Raw Total** | **11** | 100 |
| **Curved Grade** | **70** | 100 |

## How this grade was reached

Python, LATE. Submitted only a 7-page PDF printout of a Colab notebook. Not an approved language, nothing compilable or runnable. Data URLs are empty strings so it ran on 16 + 7 inline fake rows. 5 questions declared, output truncated before Q4/Q5. No README.

### Datasets identified
- Inline 16-row synthetic '311' CSV (ServiceType,ZipCode,Neighborhood,CreatedDate,ClosedDate)
- Inline 7-row synthetic 'liquor' CSV (Name,ZipCode,Neighborhood)

### Concurrency primitives found
Python concurrent.futures.ThreadPoolExecutor: one future per CSV row in parse_csv_worker_pool, ex.map over 311 rows in join_by_zip, one future per analysis strategy; two-worker pool to parse the two files concurrently

### Design patterns found
Loader base class with FileLoader/HTTPLoader + new_loader() factory; list of analysis functions as strategies; sectioned single-notebook pipeline

### Questions answered
1. Top 10 ZIP codes by 311 request volume
2. Slowest ZIP codes for 311 closure (avg hours)
3. Noise complaints per liquor license by ZIP
4. 311 volume by month (last 12)
5. Avg 311 closure time: low vs high liquor-density ZIPs

Stakeholders named: none

### Overview
A single Python Colab notebook, submitted only as a 7-page PDF printout, organized into model/loader/pipeline/join/analysis/render sections with a small Factory and Strategy. It reads two inline hand-typed CSV strings (16 and 7 rows), joins on ZIP, and prints five ASCII bar charts. There is no repository, README, test, or data file, and the language is not approved.

### Stakeholder value
None. No stakeholders are named, three of five questions use the 311 sample alone, and all answers are computed over 23 invented rows, so nothing can be learned about Baltimore.

### Issues noted
- LATE submission (flagged, not deducted)
- Wrong language: Python, not Go/Scala/Clojure/Rust
- Submitted as a PDF print of a Colab notebook; no source code, cannot be built or run
- No Open Baltimore data: URL_311 and URL_LIQUOR are empty strings; falls back to inline synthetic CSVs whose schemas do not match Open Baltimore
- No README, run script, or tests
- PDF output truncated before Q4/Q5 results
- Three of five questions are single-dataset; no stakeholders named

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

Detailed grader report with build logs and captured program output: `grading/nicholas_ighade.md` in the course grading folder.
