# Vernon Williams — COSC 352 Midterm Capstone Grading Report

**Summary: 46/100 (F).** Go. **LATE submission.** Builds only because Go auto-downloads the `go 1.27.2` toolchain demanded by go.mod; `go vet` fails on a `fmt.Print` with format directives, and that same bug garbles Research Question 4's output at runtime. Two real (but pre-aggregated, 55-row) BNIA Vital Signs CSA indicator tables from Open Baltimore are bundled. Concurrency is token, the README describes a Strategy pattern that does not exist in the code, and a Windows `.exe` is shipped instead of a macOS binary.

## Inventory

Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/vernon_williams/Baltimore City Insight/`

| File | LOC | Role |
|---|---|---|
| main.go | 49 | orchestration: concurrent load, join, calls analyses and charts |
| concurrent.go | 13 | `Result` struct + `loadConcurrent` wrapper |
| workers.go | 60 | 3-goroutine worker pool that parses two floats per community |
| join.go | 48 | `loadData` (csv.ReadAll into map[CSA2020]value), `convertNumber` |
| analysis.go | 115 | five research-question functions |
| visualization.go | 62 | two ASCII top-5 bar charts |
| README.md, go.mod (`go 1.27.2`) | | |
| data/crime.csv | 56 lines (55 rows) | BNIA "Violent Crime Rate per 1,000 Residents – CSA" |
| data/poverty.csv | 56 lines (55 rows) | BNIA "Percent of Households Living Below the Poverty Line – CSA" |
| `baltimore-city-insight.exe (NOT FOR macOS).exe` | 2.6 MB | PE32+ Windows executable |

Total LOC 347 across 6 files. No tests, no run script.

## Datasets & Data Realness

- `data/crime.csv`: header `OBJECTID,CSA2010,viol10..viol19,CSA2020,viol20..viol23,Shape__Area,Shape__Length`; rows such as `Allendale/Irvington/S. Hilton`, `Washington Village/Pigtown`, `Westport/Mount Winans/Lakeland`. This is the BNIA-JFI Vital Signs "Violent Crime Rate per 1,000 Residents – Community Statistical Area" layer (source BPD) as exported from the Open Baltimore / ArcGIS Hub catalog. Real; 55 CSAs.
- `data/poverty.csv`: header `OBJECTID,CSA2010,hhpov15..hhpov19,CSA2020,hhpov21,hhpov22,hhpov23,Shape__Area,Shape__Length`. This is the BNIA "Percent of Households Living Below the Poverty Line – CSA" indicator. The underlying numbers are American Community Survey estimates, but the file itself is the BNIA layer that the Open Baltimore catalog publishes (same ArcGIS export signature), so it counts as an Open Baltimore download, not a raw Census pull. Real; 55 CSAs.
- Both are **pre-aggregated indicator tables, not raw records** (the assignment asks for raw-format datasets). Program uses only `viol23` and `hhpov23` columns. No mock generator. Join key CSA2020 matches 55/55.

## Build Results

```
cd "submissions/vernon_williams/Baltimore City Insight"
go build -o <scratch>/baltimore-city-insight .   # exit 0, 1.4 s
go version                                       # go1.27.2 darwin/arm64 (auto-switched toolchain)
GOTOOLCHAIN=local go build .                     # FAILS: go: go.mod requires go >= 1.27.2 (running go 1.24.0; GOTOOLCHAIN=local)
```
The build succeeds only because `GOTOOLCHAIN=auto` downloaded Go 1.27.2 (network required, undocumented). The go.mod pins a far newer toolchain than the code needs (it uses nothing beyond Go 1.0 features).

## Test/Vet/Race Results

```
go vet ./...
analysis.go:87:4: fmt.Print call has possible Printf formatting directive %s     # exit 1
```
No tests (`has_tests: false`). `go build -race` succeeded; running the race binary produced 0 "DATA RACE" reports (there is barely any shared state to race on).

## Run Results

```
/usr/bin/time -l ./baltimore-city-insight     # run from project dir (relative data/ paths)
real 0.23 s, max RSS 4.8 MB, exit 0
```
Output begins with 55 lines like `Hamilton Hills Crime: 10.0303951367781 Poverty: 3.245132`, then:
```
Total joined communities: 55

 Worker pool finished processing.
Jobs Submitted: 55
Valid records processed: 55
Communites ready for analysis: 55

Research Question 1:
Which community has the highest violent crime rate?
Downtown/Seton Hill: 62.24 crimes per 1000 residents

Research Question 2:
Which community has the highest poverty percentage?
Poppleton/The Terraces/Hollins Market: 53.20%

Research Question 3:
What are the average crime and poverty rates?
Average crime rate : 19.45 per 1000 ResidentsAverage poverty: 16.68%
%s | Crime: %.2f | Poverty: %.2f%%
Downtown/Seton Hill62.235394 21.74863%s | Crime: %.2f | Poverty: %.2f%%
The Waverlies19.865697 25.07962%s | Crime: %.2f | Poverty: %.2f%%
... (20 garbled lines) ...
Research Question 4:
Which communities have above average crime and poverty?
Total communities above bothe averages:
 20
Morrell Park/Violetville | Crime: 20.33 | Poverty: 9.55%
Harbor East/Little Italy | Crime: 34.22 | Poverty: 15.19%
Highlandtown | Crime: 20.86 | Poverty: 9.79%

Research Question 5:
Which communities have a high crime rate but low poverty rate?
Total communities meeting both conditions: 3

 Top 5 Communities by Violent Crime Rate
-----------------------------------------------------------------
Downtown/Seton Hill                    | ~~~~~~~~~~~~~~~~~~~~~~~~~ 62.24
Southwest Baltimore                    | ~~~~~~~~~~~~~~~           38.82
Harbor East/Little Italy               | ~~~~~~~~~~~~~             34.22
Pigtown/Carroll Park                   | ~~~~~~~~~~~~~             34.11
Upton/Druid Heights                    | ~~~~~~~~~~~~~             34.10

Top 5 Communities by Poverty Percentage
-----------------------------------------------------------------
Poppleton/The Terraces/Hollins Market  | ~~~~~~~~~~~~~~~~~~~~~~~~~ 53.20%
Cherry Hill                            | ~~~~~~~~~~~~~~~~~~~       42.30%
Oldtown/Middle East                    | ~~~~~~~~~~~~~~~~~~        39.89%
Upton/Druid Heights                    | ~~~~~~~~~~~~~~~~~         36.51%
Westport/Mount Winans/Lakeland         | ~~~~~~~~~~~~~~~           33.19%
```
The five answers: Q1 Downtown/Seton Hill 62.24/1,000; Q2 Poppleton/The Terraces/Hollins Market 53.20%; Q3 averages 19.45 and 16.68%; Q4 20 communities above both averages (list garbled by the `fmt.Print` bug); Q5 3 communities (Morrell Park/Violetville, Harbor East/Little Italy, Highlandtown). Q4/Q5 detail lines print *before* their question headers, and Q3 is missing a newline. Map iteration makes list order nondeterministic between runs.

## Concurrency Analysis

- `main.go:4-6` — two goroutines call `loadConcurrent` and send a `Result` on a buffered channel (cap 2).
- `workers.go:11-24, 26-33` — 3 worker goroutines read `Job` from a channel (cap 55), parse two floats, send `CommunityData`; `sync.WaitGroup` to join; channels closed correctly.
- Nothing else is concurrent; the five analyses and charts are sequential. The worker pool does microseconds of work (two `strconv.ParseFloat` per row for 55 rows) and its results are immediately converted *back* to strings in `main.go:35-38` so that the analysis functions can re-parse them. This is **token** concurrency.
- Hard-coded channel capacity 55 equals the row count; with more rows the producer would still not deadlock (workers drain), but it shows no general design.

## Memory/Resource Analysis

- `join.go:19` uses `csv.ReadAll` (whole file in memory); no streaming, no preallocation, no MemStats, no README discussion. Floats are parsed, formatted to strings, and parsed again in every analysis function (`analysis.go` repeats `convertNumber` loops five times). No memory thinking evident.

## Architecture & Code Quality

Six small files give a nominal separation (load / join / workers / analysis / visualization), which is the one structural positive. However:
- The README claims "The application uses a Strategy design pattern primarily. The RankingStrategy interface ... CrimeStrategy ... PovertyStrategy". **No such interface or types exist anywhere in the code** (`grep -r Strategy` finds only the README). README does not match code.
- Data flows `map[string]string` → worker pool → `[]CommunityData` (typed) → back to `map[string]string` → re-parsed in each analysis. Each of the five analysis functions re-implements the same join+parse loop.
- `fmt.Print` with format directives (`analysis.go:87`) produces garbage output; mixed indentation; typos ("Communites", "bothe", "communtity").
- No tests, no error-exit code on failure (prints and returns).
Maintainability is low; the code reads as a first draft.

## Stakeholder Questions & Value Assessment

No stakeholders are named. Q1 and Q2 are single-dataset max lookups; Q3 is two averages; Q4 and Q5 (above-average crime with above/below-average poverty) are the only genuinely cross-dataset questions and are the most useful idea in the project (a planner could use Q5 — high crime, low poverty — to flag commercial/nightlife CSAs such as Harbor East and Highlandtown). But Q4's list is unreadable due to the bug, and all questions are threshold counts on 55 pre-aggregated rows; the ASCII charts are just top-5 lists of each single dataset. Limited actionable value.

## Issues

- **LATE** submission (flagged, not deducted).
- go.mod requires `go 1.27.2`; builds only via automatic toolchain download (network) — undocumented; fails with `GOTOOLCHAIN=local`.
- `go vet` fails: `analysis.go:87` `fmt.Print` with `%s`/`%.2f` directives; RQ4's 20 result lines print as literal format strings.
- README claims a Strategy pattern (`RankingStrategy`, `CrimeStrategy`, `PovertyStrategy`) that is not in the code.
- Ships a Windows `.exe`, no macOS binary (README does say to build).
- Token concurrency; worker pool does trivial work and results are converted back to strings.
- Datasets are pre-aggregated 55-row BNIA indicators, not raw records; only one year column used from each.
- Output ordering nondeterministic (map iteration); Q4/Q5 details print before their headers; missing newline in Q3.
- No tests, no run script.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 10 | Builds with the documented command, but only through an undocumented automatic download of Go 1.27.2; `go vet` reports an error. |
| real_data | 15 | 12 | Two real, bundled Open Baltimore (BNIA Vital Signs) CSA tables with real community names, joined 55/55; deducted because they are pre-aggregated indicators, not raw records, and only one column of each is used. |
| concurrency | 15 | 5 | Two loader goroutines plus a 3-worker pool that only parses 110 floats; analyses sequential; correct WaitGroup/channel use but token. |
| memory | 10 | 1 | `csv.ReadAll`, repeated string↔float round-trips, hard-coded channel sizes, no discussion. |
| architecture | 15 | 5 | Files are split by concern, but README's Strategy pattern does not exist, analyses duplicate the join loop five times, no tests. |
| stakeholder | 15 | 5 | Five questions, no stakeholders named, Q1–Q3 trivial single-dataset/averages, Q4 output garbled; ASCII top-5 charts are single-dataset. |
| runs_results | 10 | 5 | Runs to completion in 0.23 s but Q4 output is nonsensical (format strings printed literally) and ordering is broken. |
| readme | 5 | 3 | Has macOS steps that work, but misdescribes the design (nonexistent Strategy pattern) and ships a Windows exe. |
| **Total** | **100** | **46** | **F** |

## Final Grade Rationale

The submission does load two real Open Baltimore community indicator tables and join them on CSA2020, which earns most of the data credit, and it executes. But almost every engineering requirement is met only nominally: the concurrency does trivial work, there is no memory thinking, the claimed design pattern is absent from the code, there are no tests, `go vet` fails, and that same bug corrupts one of the five answers at runtime. The go.mod toolchain pin means the documented build does not actually work offline on a stock Go 1.24 install. 46/100, F (late submission noted, not deducted).
