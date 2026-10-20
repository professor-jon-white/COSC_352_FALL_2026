# COSC 352 Midterm Capstone — Grading Rubric (100 pts)

Assignment: Multi-Paradigmatic Data Processing Engine. Approved languages: Go, Scala, Clojure, Rust.
Must ingest >=2 REAL datasets from the Baltimore City Open Data Portal (Open Baltimore), use native
concurrency (goroutines/channels, Rust threads/rayon, Clojure futures/agents, Scala futures), show memory/resource
management thinking, clean architecture (Pipeline/Builder/Strategy/Factory), answer 5 distinct cross-dataset
stakeholder research questions with formatted output / visualizations, and include a README with macOS
build/run/verify instructions. Graded strictly on spec and compile/run success.

| # | Criterion | Pts | Guidance |
|---|-----------|-----|----------|
| 1 | build | 15 | Approved language AND compiles cleanly on macOS with documented command. 15 = clean build first try. 10 = builds with warnings or minor undocumented step. 5 = builds only after trivial env fix (not a code change). 0 = does not compile / wrong language / no code. |
| 2 | real_data | 15 | >=2 datasets, REAL Open Baltimore data, downloaded onto the computer (bundled in submission or fetched by provided script/program). Verify by inspecting headers/records against known Open Baltimore schemas (e.g., 311: SRRecordID, SRType, CreatedDate, Agency, Neighborhood...; NIBRS/Part1 crime: CCNO/RowID, CrimeDateTime, Description, District, Neighborhood, Latitude/Longitude; Vacant Building Notices: ReferenceID, BuildingAddress, NoticeDate, Neighborhood...). 15 = >=2 real datasets present and used. 10 = real data but only fetched at runtime (none bundled) or partial/only one real. 5 = only sample/simulated data present (program needs real data user must download) and generator included. 0 = fully simulated/fake data or <2 datasets. Deduct 3 if a mock generator is the default path. |
| 3 | concurrency | 15 | Native concurrency primitives actually used for parse/join/aggregate across a pool of workers. 15 = parallel parsing AND parallel join/aggregate with worker pool, correct sync (WaitGroup/Mutex/channels, Arc/Mutex/rayon, etc.), no races visible. 10 = concurrency used in at least one real stage. 5 = token concurrency (e.g., two goroutines loading two files, nothing else). 0 = sequential. Deduct for data races / unsafe shared state. |
| 4 | memory | 10 | Explicit consideration of runtime overhead, thread safety, memory (streaming CSV instead of read-all, bounded channels, string interning, preallocation, runtime.MemStats reporting, Rust ownership/zero-copy, documented in README/comments). 10 = designed & documented & evident in code. 5 = some evidence. 0 = none / loads everything naively with no thought. |
| 5 | architecture | 15 | Separation of ingestion / matching-join / analysis / output; named design pattern(s) actually realized in code; package layout; naming; tests; maintainability. 15 = clean multi-package layout, clear patterns, tests. 10 = reasonable separation in few files. 5 = single big file but organized. 0 = spaghetti. |
| 6 | stakeholder | 15 | 5 distinct, major, CROSS-DATASET research questions for named municipal stakeholders; uniqueness/insight of the questions and answers; formatted summaries + visualization (ASCII charts, HTML, web UI). 15 = 5 genuinely cross-dataset, well-motivated, unique questions with clear actionable answers + visuals. 10 = 5 questions but some are single-dataset or shallow. 5 = <5 or trivially generic (counts per district). 0 = none. |
| 7 | runs_results | 10 | Actually executes end-to-end on the included/downloaded data and produces the 5 answers. 10 = runs to completion, output sensible. 5 = runs but crashes partway/produces partial or nonsensical output/needs undocumented flags. 0 = does not run. |
| 8 | readme | 5 | README with explicit macOS compile/execute/verify instructions, run script, matches reality. |

Total = sum (max 100). Letter: A 90+, B 80-89, C 70-79, D 60-69, F <60.

Rules for the grader:
- DO NOT modify any student code or data. If it does not build/run, report the exact error. You MAY create files in the scratchpad or run commands with different flags/env vars documented in the README.
- Use timeouts: builds <= 5 min, runs <= 10 min. If a run exceeds that, note it and score runs_results at most 5.
- Note whether data was bundled vs fetched at runtime, and whether any mock/sample generator exists and whether it is the default path.
- Flag LATE submissions (filename contained _LATE_) in issues but do not deduct.
- Record actual program output (first ~60 lines plus the 5 answers) in the report.
