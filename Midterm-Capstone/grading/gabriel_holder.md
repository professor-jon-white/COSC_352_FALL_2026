# Gabriel Holder — COSC 352 Midterm Capstone Grading Report

**Summary: 52/100 — F.** Competently structured single-file Go program with a real per-CSV worker pool and bounded channels, but it operates on **two 10-row CSVs of admittedly synthetic values** (README: "synthetic demonstration values, not real Baltimore data"). No real Open Baltimore data is bundled, fetched, or generated, so every "answer" (Pearson r = 0.834, priority ranking) is meaningless. No go.mod.

## Inventory
Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/gabriel_holder/baltimore-intelligence/`

| File | LOC / size | Notes |
|---|---|---|
| main.go | 257 | all logic |
| README.md | 59 lines | |
| data/crime.csv | 232 B, 11 lines (10 rows) | synthetic |
| data/business.csv | 235 B, 11 lines (10 rows) | synthetic |
| output/report.json | 1.4 KB | prior run output |
| .DS_Store | | stray |

No go.mod, no tests, no run script. Language: Go. LOC 257 (1 file).

## Datasets & Data Realness
README names two BNIA Vital Signs indicators hosted on Open Baltimore: "Violent Crime Rate per 1,000 Residents" and "Neighborhood Businesses per 1,000 Residents" (both by Community Statistical Area), with portal links.
```
data/crime.csv:     CSA,2023 / Canton,12.62 / Downtown/Seton Hill,62.24 / Cherry Hill,18.41 ... (10 rows)
data/business.csv:  CSA,2023 / Canton,25.97 / Downtown/Seton Hill,249.68 / Cherry Hill,16.142 ... (10 rows)
```
The CSA names are real BNIA CSA names, but the README states plainly these are **synthetic demonstration values**; the real Vital Signs CSV exports have ~55 CSAs plus a Baltimore City row and many year columns (`CSA2010`, `vcrime23`, `neibus23`, ...), not a two-column `CSA,2023` layout. There is no fetch script and no generator. Row counts: 10 and 10.
**data_status: simulated.**

## Build Results
README: `go run main.go` and `go build -o baltimore main.go`.
```
$ go build -o <scratch>/baltimore main.go     -> exit 0 (0.3 s)
$ go build -race -o <scratch>/baltimore-race main.go -> exit 0
$ go build ./...   -> FAIL: "pattern ./...: directory prefix . does not contain main module" (no go.mod)
```
Builds cleanly with the documented file-based command; module-style commands fail for lack of go.mod.

## Test / Vet / Race Results
```
$ go vet main.go   -> exit 0
$ go test ./...    -> FAIL ./... [setup failed] (no go.mod; also no tests exist)
```
Race-instrumented run: **0 DATA RACE** reports.

## Run Results
Ran the binary with `-out <scratch>/report.json` to avoid overwriting the student's `output/report.json` (the produced JSON is byte-identical to the submitted one, md5 3784eafb…). Wall **1.23 s** (race binary), max RSS **24 MB**, exit 0. Output (scratchpad `gabriel/run.txt`):
```
BALTIMORE NEIGHBORHOOD INTELLIGENCE | rates per 1,000 residents
Matched areas: 10 | Crime-only: 0 | Business-only: 0

Q1. Highest violent crime rate
downtown/seton hill   62.24  #########################
greater mondawmin     30.22  ############
clifton-berea         23.23  #########
... (10)
Q2. Lowest business density
belair-edison          7.49  #
greenmount east       10.78  #
... downtown/seton hill 249.68 #########################
Q3. Relationship between crime and business density
Pearson correlation: 0.834 (association, not causation)
Q4. Highest crime with lower business density (priority score)
downtown/seton hill   70.00 ... greater roland park/poplar hill 27.70
Q5. Largest differences between crime and business density
greater mondawmin 13.78 ... downtown/seton hill -187.44
Skipped invalid records: crime=0, businesses=0
JSON report: <path>
```
`go run main.go` as documented also works (verified) but would overwrite `output/report.json` by default.

## Concurrency Analysis
- `main.go:224-229`: two goroutines + `sync.WaitGroup`, one `loadCSV` per dataset.
- `main.go:62-110` (`loadCSV`): per-file **bounded worker pool** — `jobs`/`parsed` channels with capacity `4*workers`; `workers` parser goroutines; a reader goroutine feeding `jobs` and closing it; `wg.Wait()` closer for `parsed`; a single collector loop owning the result map (no shared map writes). Correct and race-clean.
- Join (`join`, 112-120), correlation, scoring, display: sequential. With 10 rows the pool is decorative, but the mechanism is real.
- Level: **partial** (concurrency in parsing stage only; join/aggregate sequential).

## Memory / Resource Analysis
Streams rows; retains only `map[string]float64` per dataset; bounded channels for backpressure (`workers*4`); worker bound 1–128; `defer f.Close()`; README has a "Resource management" section covering these points and GC/stack growth. Reasonable design, but trivially exercised by 20 rows. No MemStats.

## Architecture & Code Quality
Single file with clear functional decomposition: `normalize`, `loadCSV`, `join`, `correlation`, `score`, `bar`, `display`, `main`. Reasonable naming, JSON output via struct tags, flag-driven configuration (`-area`, `-crime-col`, `-business-col`) that anticipates the real exports' headers. No named pattern beyond an implicit pipeline; no packages; no tests; no go.mod (so `go vet ./...`/`go test ./...` cannot run). `panic` on output errors. The README's limitations section is thoughtful but largely boilerplate relative to a 10-row demo.

## Stakeholder Questions & Value Assessment
No stakeholder is named. Q1/Q2 are single-indicator rankings; Q3 (correlation), Q4 (weighted priority index), Q5 (rate difference) do combine the two indicators, and Q4 is a reasonable targeting idea. However, with fabricated inputs none of the answers is real: "r = 0.834" and "Downtown/Seton Hill priority 70" describe invented numbers. Q5 subtracts two incommensurable rates, which the program itself labels non-interpretable. Visualization: ASCII bars plus JSON report. A city stakeholder learns nothing until real exports are supplied and columns renamed by hand.

## Issues
1. Data is fully synthetic (10 rows each), explicitly acknowledged; no download script or generator; real data absent.
2. No go.mod — `go build ./...`, `go vet ./...`, `go test ./...` fail; only `go run main.go` works.
3. Join/aggregate/score sequential; worker pool only in parsing.
4. No tests; no named stakeholder; Q1/Q2 single-dataset, Q5 subtracts unrelated rates.
5. Default run overwrites `output/report.json`; stray .DS_Store.

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 13/15 | Documented `go build -o baltimore main.go` works cleanly; missing go.mod breaks standard `./...` commands. |
| real_data | 2/15 | Admittedly synthetic 10-row files, no generator, no fetch; credit only for correct portal links and a documented real-export path. |
| concurrency | 10/15 | Real bounded worker pool with channels/WaitGroup for parsing (race-clean); join/aggregate sequential. |
| memory | 7/10 | Streaming, bounded channels, bounded workers, documented; not meaningfully exercised, no MemStats. |
| architecture | 6/15 | Single well-decomposed file; no packages, tests, go.mod, or explicitly realized pattern. |
| stakeholder | 5/15 | No stakeholder named; 3 of 5 cross-indicator but answers computed on invented values; ASCII+JSON output. |
| runs_results | 5/10 | Runs end-to-end but output is derived entirely from fabricated data, so results are not meaningful. |
| readme | 4/5 | Clear macOS run/build steps, honest about data, real-export instructions; no verify step. |
| **Total** | **52/100** | **F** |

## Final Grade Rationale
The engineering is decent for its size — a correct bounded worker pool, streaming parse, flag-driven schema mapping, JSON output — and it builds and runs cleanly. But the assignment is about ingesting real Open Baltimore data, and this submission contains none: both inputs are ten invented values, with no mechanism to obtain the real ones. That empties the stakeholder analysis and run results of meaning, and together with a single-file layout lacking go.mod and tests, yields 52/100.
