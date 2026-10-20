# Midterm Capstone Grade — Jayden Reyes

**Raw score: 52/100    Curved grade: 80/100**

Language: Go. Builds: yes. Runs: yes. Data status: real_bundled. Concurrency level: token. Datasets found: 3. Questions answered: 5. Visualization: none. Tests: none. Race detector: clean.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | 7 | 20 |
| Real Data | 12 | 15 |
| Five Questions | 5 | 20 |
| Three Datasets | 10 | 15 |
| Runs Well + Documented | 11 | 15 |
| Project Structure | 7 | 15 |
| **Raw Total** | **52** | 100 |
| **Curved Grade** | **80** | 100 |

## How this grade was reached

Go. 3 sources: 2 real but reshaped Open Baltimore extracts (arrests, gun offenders) plus a 20-row CVI table hand-typed from a PDF. Token concurrency. 5 questions, 2 nominally cross-dataset, all trivial counts/sets; Q5 answer is a name-mismatch artifact. No stakeholder, no visualization, README minimal. Runs in 0.3 s.

### Datasets identified
- BPD Arrests (Open Baltimore; reshaped extract as wb_records.csv)
- Gun Offenders Registry (Open Baltimore; reshaped extract as 'gvt results.csv')
- Safe Streets CY2024 mediations/events (MONSE PDF transcription; not Open Baltimore)

### Concurrency primitives found
3 goroutines, one per CSV loader, each with a cap-1 result channel (concurrency.go:27-46); 3 goroutines counting per dataset into private maps via channels (analytics.go:30-78); sequential merge; no WaitGroup/Mutex/pool

### Design patterns found
None named; files split by role (revealer.go loaders, concurrency.go, analytics.go); normalizeNeighborhood join key

### Questions answered
1. How many unique neighborhoods appear across the datasets?
2. Which neighborhoods have both Arrest records and Gun Offender records?
3. Which neighborhood has the highest CVI mediations?
4. Which neighborhood has the fewest CVI community events?
5. Which neighborhoods have CVI activity but no Gun Offender records?

Stakeholders named: none

### Overview
Beginner Go program split into main/concurrency/revealer/analytics files: three goroutines load three CSVs fully into struct slices, three more count records per normalized neighborhood, and five questions are printed as plain text. Struct field names (OfficerName, SettlementAmount) contradict the actual gun-offender columns, there are no tests or flags, and the README is a first-person narrative. Canvas and direct_zip copies are identical except two README lines.

### Stakeholder value
No stakeholder is named and the questions are trivial: a neighborhood count, a ~230-line list of neighborhoods present in both arrests and registry, a max/min over 10 Safe Streets sites, and a set difference whose sole answer ('park heights') is a naming artifact. MONSE or BPD would get no rates, rankings or actionable comparison from this output.

### Issues noted
- Token concurrency (goroutine per file) and full in-memory loads; no streaming, pool, or concurrent join
- Questions trivial; Q3/Q4 single-dataset; Q5 answer is a name-mismatch artifact; no stakeholder; no visualization
- Data are reshaped/renamed extracts (synthetic RecordType/Count columns), not raw exports; third dataset hand-typed from a MONSE PDF; struct names contradict data
- scripts/gen_mock.go is broken (decimal perm bits 1304) and would overwrite real data if it worked; not the default path
- README narrative, claims make build runs the program, no macOS/verify section; go.mod go 1.27.2 forces toolchain download; direct_zip differs from Canvas only in README lines 19-21

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

Detailed grader report with build logs and captured program output: `grading/jayden_reyes.md` in the course grading folder.
