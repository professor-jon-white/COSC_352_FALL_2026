# Midterm Capstone Grade — Daniel Dada

**Raw score: 85/100    Curved grade: 92/100**    (submitted LATE; noted, not deducted)

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: full. Datasets found: 4. Questions answered: 5. Visualization: ascii. Tests: pass. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 19 | 20 |
| Real Data | 15 | 15 |
| Five Questions | 13 | 20 |
| Three Datasets | 10 | 15 |
| Runs Well + Documented | 14 | 15 |
| Project Structure | 14 | 15 |
| **Raw Total** | **85** | 100 |
| **Curved Grade** | **92** | 100 |

## How this grade was reached

Go, LATE. 3 real files bundled (crime 644k, 311 2024 1.08M, 311 2025 931k) but the 2025 file is unused and the two 311 files are one dataset split by year. Full concurrency, race-clean, 950 MB processed in 2 s at 830 MB RSS. 5 questions, 3 cross-dataset; Q4/Q5 are side-by-side single-dataset views. ASCII charts, named stakeholders.

### Datasets identified
- Part 1 Crime Data Legacy SRS (data/crime.csv)
- 311 Customer Service Requests 2024 (data/311_2024.csv)
- 311 Customer Service Requests 2025 (data/311_2025.csv, bundled but unused by default -year 2024)
- data/sample/*.csv real-row subsets built by tools/mksample

### Concurrency primitives found
goroutines; two files ingested concurrently (sync.WaitGroup); generic parallel[T] ingest with reader -> bounded batch channel (2000 rows, cap workers*2) -> NumCPU workers with worker-local partials -> merge; BuildParallel join splits slices into NumCPU chunks each filling a private Dataset, merged once via channel; -verify asserts parallel join == sequential join with reflect.DeepEqual; analyses sequential

### Design patterns found
Strategy (analysis.Analyzer interface with 5 implementations, All() registry); explicit 3-stage pipeline in main; sequential reference implementations (Build, ReadCrime) verified against parallel versions; no Builder/Factory

### Questions answered
1. Which neighborhoods rank high in BOTH Part 1 crime and 311 demand? (Mayor's Office)
2. Do sanitation/blight 311 requests track violent crime across neighborhoods? (DPW)
3. Which neighborhoods have high crime but low 311 use, suggesting under-reporting? (BPD)
4. How do crime, street-lighting requests and transportation requests vary by hour of day? (DOT)
5. What are the monthly crime and 311 trends, and which neighborhoods diverge from the citywide crime pattern? (Planning)

Stakeholders named: Mayor's Office, Department of Public Works, Baltimore Police Department, Department of Transportation, Department of Planning

### Overview
Compact multi-package Go module (cmd, internal/{ingest,join,analysis,normalize,model,report}, tools/mksample) with a generic batch-ingest worker pool and a chunked parallel join, each paired with a sequential reference implementation that tests and a -verify flag prove equivalent. An Analyzer Strategy interface drives five plain-text ASCII reports. Code is readable with good thread-safety comments, tests in four packages and a macOS CI workflow, but memory handling is naive (all records retained with un-interned strings; 830 MB RSS) and the report layer is minimal.

### Stakeholder value
Q3's under-reporting z-score (Inner Harbor, Dunbar-Broadway, Canton Industrial Area) with join-gap neighborhoods separated out is genuinely actionable for BPD/311 outreach, and Q2's r=0.68 blight-violence association is useful context for DPW. Q1 is a size-biased rank-sum of raw counts, while Q4 (hourly histograms) and Q5 (monthly trend, crime-only divergence) place the datasets side by side rather than joining them, so about half the questions deliver real cross-dataset insight.

### Issues noted
- LATE submission (filename contained _LATE_); flagged, not deducted
- Memory: every kept record retained with un-interned CSV substrings; 830 MB RSS and 2.8 GB total allocation on full data; no memory design documented
- Q4 and Q5 are largely single-dataset views presented side by side rather than cross-dataset joins
- data/311_2025.csv (379 MB) is bundled but never used by the default run
- Analyses run sequentially; ASCII-only output with 29-char label truncation; README ends mid-sentence

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

Detailed grader report with build logs and captured program output: `grading/daniel_dada.md` in the course grading folder.
