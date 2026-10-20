# Midterm Capstone Grade — Lyric Smith

**Raw score: 39/100    Curved grade: 75/100**

Language: Go. Builds: yes. Runs: yes. Data status: simulated. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 0 | 15 |
| Five Questions | 7 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 9 | 15 |
| Project Structure | 8 | 15 |
| **Raw Total** | **39** | 100 |
| **Curved Grade** | **75** | 100 |

## How this grade was reached

Go. 6 + 6 hand-written rows with invented schemas (README admits 'hand-built'). Token concurrency (two goroutines per stage). 5 questions, 4 cross-dataset, but Q2 hard-codes literals that never occur in real data. Named stakeholders in README only. Runs in 0.2 s.

### Datasets identified
- 311 Customer Service Requests (claimed; hand-built sample)
- Crime Incidents (claimed; hand-built sample)

### Concurrency primitives found
goroutines + sync.WaitGroup + sync.Mutex: 2 goroutines load 2 CSVs (loader.go:55-73); 2 goroutines append into a single mutex-guarded index (joiner.go:48-70); analysis sequential; no channels, no worker pool

### Design patterns found
Strategy (RecordParser interface with two parsers), trivial Factory constructors, ordered pipeline in main()

### Questions answered
1. Neighborhood burden ranking by combined 311 + crime volume
2. Neighborhoods with both burglaries and vacant-property 311 reports
3. Monthly crime vs 311 volume trend
4. Most common 311 complaint types (single-dataset)
5. Lowest 311-per-crime ratio neighborhoods (under-reporting outliers)

Stakeholders named: Mayor's Office, Department of Public Works, Department of Transportation, City Planning and Housing

### Overview
Small, tidy single-package Go program split into six focused files with a genuine Strategy interface for parsers and a mutex-guarded neighborhood index. Parsing is positional and records are boxed as interface{}, so it only works on the two hand-built CSVs it ships with. No tests; README is accurate and honest about the data.

### Stakeholder value
No actionable value: all five answers are computed over twelve invented rows, and Q2 depends on literal values (CrimeCode 'BURGLARY', SRType 'VACANT') that never occur in real Open Baltimore data. Q5 (311-per-crime ratio) is a reasonable idea that could be informative on real data; Q4 is single-dataset.

### Issues noted
- Fully simulated data: 6+6 hand-built rows; headers (SRNumber, CrimeCode=BURGLARY) do not match Open Baltimore schemas despite README claim
- No download script, URL, or generator for real data
- Positional column parsing and hard-coded category literals would break on real exports
- Token concurrency; join serialized under a single mutex; analysis sequential
- No tests; interface{} boxing with type assertions; unused requests.csv at repo root
- Q4 (311 complaint mix) is single-dataset

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

Detailed grader report with build logs and captured program output: `grading/lyric_smith.md` in the course grading folder.
