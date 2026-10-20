# Midterm Capstone Grade — Mikayla Brown

**Raw score: 37/100    Curved grade: 74/100**

Language: Rust. Builds: yes. Runs: yes. Data status: simulated. Concurrency level: token. Datasets found: 2. Questions answered: 5. Visualization: ascii. Tests: none. Race detector: not_run.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 0 | 15 |
| Five Questions | 5 | 20 |
| Three Datasets | 8 | 15 |
| Runs Well + Documented | 6 | 15 |
| Project Structure | 11 | 15 |
| **Raw Total** | **37** | 100 |
| **Curved Grade** | **74** | 100 |

## How this grade was reached

Rust. 'GTTF settlements' and officer incidents are not Open Baltimore datasets; 3 + 3 synthetic rows and the program itself prints 'inputs are synthetic'. Token concurrency. 5 cross-dataset questions on fake data. Clean idiomatic Rust with Builder/Strategy, but README truncated mid code block, 192 MB target/ committed, static mock dashboard.

### Datasets identified
- 'Gun Trace Task Force settlements' (claimed Open Baltimore; actually 3 synthetic rows, dataset does not exist on Open Baltimore)
- 'BPD GIS incidents' with officer_id/shift/unit columns (claimed Open Baltimore; actually 3 synthetic rows, no such schema exists)

### Concurrency primitives found
rayon::join for the two CSV loads (ingestion.rs:78-81); rayon par_iter().map().collect() over settlements in the join (pipeline.rs:24-39 via concurrency.rs:5-10); single tokio spawn_blocking task wrapping the whole pipeline (main.rs:41-53); analyses sequential

### Design patterns found
Builder (QueryConfig::builder().district().build()), Strategy (trait AnalysisStrategy with 5 unit-struct implementors via &dyn), explicit pipeline in main; zero-copy borrowed JoinedRecord<'a>

### Questions answered
1. Q1 Settlement cost by district alongside incident counts
2. Q2 Units/shifts in officer_id-matched settlement-incident pairs
3. Q3 Misconduct reports vs disciplinary actions per year (settlements only)
4. Q4 Recurring officer identifiers across both files with cost vs mean
5. Q5 District risk index = 0.6*normalized cost + 0.4*normalized incidents

Stakeholders named: City leaders (unnamed)

### Overview
Idiomatic Rust split into seven small modules with a real Builder, a Strategy trait dispatched via &dyn, clap CLI, anyhow errors, and a zero-copy borrowed join; compiles warning-free. No tests, a 10-line rayon wrapper and a tokio runtime used for a single blocking task pad out the concurrency story, serde_json is unused, and the companion index.html dashboard renders hard-coded mock numbers rather than program output. README reads as generated marketing copy and is truncated.

### Stakeholder value
None in practice: the data is six invented rows about a dataset Open Baltimore does not publish, so every answer (Western district costs $200K, OFF-001 recurs) is fiction, and the dashboard shows an 'Eastern District' and 'OFF-042' that the engine never computed. Only Q2 is a genuine cross-dataset join; the others are side-by-side single-dataset counts. The misconduct-liability theme would be interesting with real settlement data, but the officer-level incident schema it assumes is not public.

### Issues noted
- Fully synthetic data: 3+3 rows with John Doe/Jane Smith/OFF-001; claimed 'GTTF settlements' and officer-linked incidents are not Open Baltimore datasets; README's 'official Open Baltimore datasets' claim is false
- No path to real data: URL flags expect the same invented schema; no generator; no source URLs
- README has no macOS build/execute/verify instructions and is truncated mid code block; only run.sh documents the build
- 192 MB target/ directory (989 build-artifact files incl. 7.3 MB debug binary from nightly rustc 1.99) committed and shipped; shipped binary is SIGKILLed on launch; no .gitignore
- index.html dashboard is static mockData unconnected to the Rust program, with districts/officers that exist nowhere else
- Token concurrency: rayon::join of two loads + par_iter over three rows; analyses sequential; tokio runtime wraps one spawn_blocking
- No tests; unused serde_json dependency; empty data/data/ directory; .vscode/ shipped
- Q1 bar chart renders $75K and $200K as identical full-width bars (sqrt + clamp)

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

Detailed grader report with build logs and captured program output: `grading/mikayla_brown.md` in the course grading folder.
