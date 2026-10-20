# Lyric Smith — COSC 352 Midterm Capstone Grading Report

**Summary: 45/100 (F).** Clean, small Go program that builds and runs first try, but it processes only 12 hand-written rows of invented data whose columns do not match any Open Baltimore schema; concurrency is token (two goroutines per stage) and three of the five "analyses" would return nothing on real data because they hard-code values that do not exist in the real datasets.

## Inventory
- Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/lyric_smith/baltimore-data-engine/`
- Language: Go (go.mod `go 1.19`, module `baltimore-data-engine`, single `package main`)
- Source files (469 LOC): `analysis.go` 166, `joiner.go` 78, `loader.go` 73, `main.go` 62, `parsers.go` 59, `models.go` 31
- README: yes (`README.md`). Run script: none (README says `go run .`). Tests: none. Late: no.
- Data: `data/311_sample.csv` (7 lines, 292 B), `data/crime_sample.csv` (7 lines, 371 B), `requests.csv` at repo root (4 lines, 88 B — not referenced by any source file; dead file).

## Datasets & Data Realness
- Claimed: "311 Customer Service Requests" and "Crime Incident reports" from Open Baltimore.
- Actual: `data/311_sample.csv` header `SRNumber,SRType,Neighborhood,District,CreatedDate`; `data/crime_sample.csv` header `CrimeCode,Description,Neighborhood,District,CrimeDate`. Six records each, e.g. `SR1,POTHOLE,Canton,Southeastern,2024-01-15` and `BURGLARY,Breaking and entering,Sandtown,Western,2024-01-10`.
- Neither header matches Open Baltimore. Real 311 uses `SRRecordID, ServiceRequestNum, SRType, CreatedDate, SRStatus, Agency, Neighborhood, ...` with SRTypes like `SW-Dirty Alley`; real crime/NIBRS uses `RowID/CCNO, CrimeDateTime, CrimeCode (e.g. "4C"), Description, District, Neighborhood, Latitude, Longitude`. `CrimeCode=BURGLARY` and `SRType=VACANT` are invented values. Neighborhood "Sandtown" (real name is "Sandtown-Winchester") and generic 2024 dates.
- README openly states: "The included sample CSVs are small, hand-built examples." No download script, no URLs, no generator. Row counts: 6 + 6 data rows.
- **data_status: simulated.** Zero real Open Baltimore records are present or fetched.

## Build Results
```
cd .../lyric_smith/baltimore-data-engine
go build -o <scratch>/lyric/bin .      -> exit 0, no output
go vet ./...                          -> exit 0, no output
go build -race -o <scratch>/lyric/bin_race . -> exit 0
```
Clean build first try with the documented command (`go run .` / `go build`).

## Test/Vet/Race Results
- `go test ./...` -> `? baltimore-data-engine [no test files]` (no tests).
- `go vet ./...` clean.
- Race binary run on the included data: 0 `DATA RACE` warnings (**race_detector: clean**; trivially small data).

## Run Results
Command (README): `go run .` run from the project directory. Wall time 0.63 s for `go run` (compile included), max RSS 61 MB; prebuilt binary 0.21 s, max RSS 4.7 MB. Exit 0. Captured output (`<scratch>/lyric/run.txt`), complete:
```
Loading datasets concurrently...
Loaded 6 311 requests (0 skipped)
Loaded 6 crime records (0 skipped)
Joining datasets...
Joined into 3 neighborhoods
Running analyses...

=== BALTIMORE DATA ENGINE REPORT ===

1. Neighborhood Burden Ranking
Top neighborhoods by combined 311 + crime volume:
  Sandtown        [##############################] 6
  Canton          [####################..........] 4
  Hampden         [##########....................] 2

2. Crime vs 311 Correlation
Neighborhoods with both burglaries and vacant property reports:
  Sandtown             burglaries=3 vacant=3

3. Monthly Trend
Monthly volume, crime vs 311:
  2024-01 crime=2 311=2
  2024-02 crime=2 311=2
  2024-03 crime=2 311=2

4. 311 Complaint Mix
Most common 311 complaint types:
  VACANT               3
  POTHOLE              2
  RODENT               1

5. Reporting Outliers
Neighborhoods with lowest 311-per-crime ratio (possible under-reporting):
  Sandtown             crimes=3 311s=3 ratio=1.00
```
The five answers are produced, but on 12 fabricated rows they carry no information.

## Concurrency Analysis
- `loader.go:55-73` `loadAll`: one goroutine per file (2 total), `sync.WaitGroup`, `sync.Mutex` guarding the results map. Parsing within a file is sequential.
- `joiner.go:48-70` `buildIndex`: exactly two goroutines (one appends requests, one appends crimes) sharing one `Index` whose single `sync.Mutex` serializes every append (`joiner.go:21-46`), so the "concurrent" join is effectively serialized under a lock.
- Analysis stage (`analysis.go`) is fully sequential. No worker pool, no channels, no pipelining.
- Correctness: lock discipline is right (race detector clean). Minor: `Index.all()` reads the map without the lock, safe only because it is called after `wg.Wait()`.
- Level: **token** — matches the rubric's "two goroutines loading two files, nothing else" example.

## Memory/Resource Analysis
- `loader.go:29-50` streams rows with `csv.Reader.Read()` rather than `ReadAll` (README claims this as the memory feature), but every parsed record is boxed into `[]interface{}` and then copied again into typed slices in `main.go:28-36`, doubling allocations.
- `analysis.go` builds output with string `+=` concatenation in loops; `barChart` concatenates characters one at a time. No preallocation, no MemStats, no bounded channels. Rubric tier: minimal evidence.

## Architecture & Code Quality
- Single `package main` split into six focused files; naming is clear and gofmt-clean. `RecordParser` interface (`parsers.go:10-13`) is a genuine Strategy; the "Factory" is two one-line constructors; "Pipeline" is just the ordered calls in `main()`.
- Weaknesses: `interface{}` records with type assertions (`main.go:30,35`) instead of generics or typed loaders; parsers index columns positionally (`row[0..4]`) so they only work on the hand-built files, not on a real export with 20-30 columns in a different order; no tests; dead `requests.csv` at root.
- Originality: modest, straightforward student code; README matches the code accurately.

## Stakeholder Questions & Value Assessment
Stakeholders named: Mayor's Office, Department of Public Works, Department of Transportation, City Planning and Housing.
1. Neighborhood Burden Ranking (combined 311+crime counts) — cross-dataset but simple co-counting.
2. Crime vs 311 Correlation — hard-codes `CrimeCode == "BURGLARY"` and `SRType == "VACANT"` (`analysis.go:41,46`); on real data these literals never occur, so the section would be empty. Not a correlation, just two counts.
3. Monthly Trend — side-by-side monthly counts; no relationship computed.
4. 311 Complaint Mix — single-dataset (311 only); assigned to DOT but is just a type histogram.
5. Reporting Outliers (311-per-crime ratio) — a reasonable cross-dataset idea, the strongest question here.
Value: with six synthetic rows per side, no stakeholder could learn anything; the design would also break on real exports (positional columns, invented category literals). ASCII bar chart only for Q1.

## Issues
- Fully simulated data; no Open Baltimore records bundled or fetched; column names do not match the real schemas despite README claim that they "mirror the real Open Baltimore column names."
- Positional column parsing and hard-coded category literals mean the code would not work on real exports without modification.
- Token concurrency; join is serialized by a single mutex; analysis sequential.
- No tests; `interface{}` boxing; unused `requests.csv` at repository root.
- Q4 is single-dataset.

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | `go build`/`go vet` clean on first try with the documented command. |
| real_data | 0/15 | Both files are hand-built, non-schema-conformant toy data; nothing real, nothing fetched, no generator/downloader. |
| concurrency | 5/15 | Two goroutines per stage with WaitGroup/Mutex; no worker pool, join serialized under one lock, analysis sequential. |
| memory | 3/10 | Streams CSV rows but boxes into `[]interface{}` and copies; string concatenation; no other evidence. |
| architecture | 8/15 | Six well-named files and a real Strategy interface, but single package, no tests, positional parsing, interface{} records. |
| stakeholder | 5/15 | Five questions with named stakeholders, but Q4 single-dataset, Q2 uses nonexistent literals, all answers vacuous on 12 fake rows. |
| runs_results | 5/10 | Runs to completion and prints five sections, but only on synthetic data; output is not meaningful. |
| readme | 4/5 | Clear macOS build/run/verify steps that match reality; honest about data; no run script. |
| **Total** | **45/100** | **F** |

## Final Grade Rationale
The program compiles and runs cleanly and is tidy for its size, which earns full build credit and partial architecture credit. However, the assignment's core requirements — real Open Baltimore data, meaningful concurrent parse/join/aggregate, and cross-dataset insight — are essentially unmet: the data is twelve invented rows with invented schemas, concurrency is two goroutines per stage, and several analyses depend on literals that do not exist in real data. The result is a well-organized skeleton rather than a data processing engine, which places it at 45/100.
