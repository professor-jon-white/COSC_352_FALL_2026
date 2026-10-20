# Amari Hall — COSC 352 Midterm Capstone Grading Report

**Summary: 71/100 (C)** — Go. Compiles and runs cleanly on two bundled real Open Baltimore datasets (NIBRS crime, 311 requests 2026) and prints five reports with ASCII bars. Concurrency is modest (one goroutine per file plus a worker pool that re-aggregates already-aggregated counts), the architecture is a single `main` package in three files, no stakeholders are named, and the questions are mostly count-based.

## Inventory

- Location: `submissions/amari_hall/BaltimoreCapstone/`
- Language: Go (`go.mod` module `baltimorecapstone`, `go 1.27.2` — go1.24 auto-downloaded the 1.27.2 toolchain)
- Source: `main.go` (272), `analysis.go` (421), `dates.go` (29), `dates_test.go` (39) — 761 LOC, 3 non-test files, single package `main`.
- README.md (~110 lines), `run.sh` (runs `go test`, `go build -o baltimore-engine .`, `./baltimore-engine`).
- Tests: `dates_test.go` (2 tests for `parseMonth`).
- Prebuilt binaries shipped: `baltimore-engine-mac-arm64` (2.6 MB), `baltimore-engine-mac-intel` (2.7 MB), `baltimore-engine.exe` (2.8 MB). All built with go1.27.2 (`go version -m`). Shipped with mode `-rw-rw-rw-` (no execute bit) and quarantine attributes.
- Data bundled: `data/crime.csv` 73 MB, `data/service_requests.csv` 382 MB.

## Datasets & Data Realness

| File | Rows (data) | Header / records |
|---|---|---|
| `data/crime.csv` | 278,239 | `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Shooting,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y`; e.g. `261003996,10/11/2026 5:00:00 AM,290,VANDALISM,O,...,5300 SINCLAIR LN,...,FRANKFORD,39.324417,-76.541309`. Byte-identical header and row count to the NIBRS file I downloaded from Open Baltimore today for another submission. Covers 2022–Oct 2026. |
| `data/service_requests.csv` | 943,283 | `X,Y,RowID,SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,...,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord`; e.g. `26-00004287,SW-Bulk Scheduled-Saturday,Phone,2026/01/03 14:44:56+00,...,"3954 DOLFIELD AVE, Baltimore City, 21215",21215,Dolfield,6,Northwestern`. This is the **311 Customer Service Requests 2026** layer (ArcGIS export layout with X,Y,RowID). Months Jan–Oct 2026 (Oct partial: 30,636 rows). 8,988 rows mention Sandtown. |

Both are real Open Baltimore datasets, bundled. No mock/sample generator exists. Data status: **real_bundled**.

## Build Results

```
$ go build -o baltimore-engine .     (output redirected to scratchpad)
exit=0   0.57 s
$ go vet ./...                       exit=0, no output
$ go build -race -o baltimore-engine_race .   exit=0
```
Clean build, no warnings.

## Test / Vet / Race Results

```
$ go test -count=1 ./...        ok  baltimorecapstone  0.325s
$ go test -race -count=1 ./...  ok  baltimorecapstone  1.243s
$ ./baltimore-engine_race  (full data)   exit=0, 0 "DATA RACE", 13.5 s
```
Tests: pass (2 small date-parsing tests). Race detector: **clean**.

## Run Results

Run from the student directory (program uses relative `data/` paths), binary from scratchpad:
```
$ /usr/bin/time -l baltimore-engine
exit=0   1.59 s real   max RSS 18 MiB (18,956,288 bytes)
```
Full output (`scratchpad/amari/run.txt`):
```
Baltimore City Intelligence Engine
Available CPUs: 10

Crime Incidents processed: 222948 records   Monthly groups: 12612   Invalid or missing dates: 0
311 Requests processed: 692663 records      Monthly groups: 2787    Invalid or missing dates: 0

Reporting Period Analysis: First shared month: 2026-01  Last shared month: 2026-10  Shared months: 10
ANALYSIS PERIOD: 2026-01 through 2026-09   Neighborhoods analyzed: 279   Months analyzed: 9

REPORT 1: Neighborhood Activity
Neighborhood                     311 Requests Crime Records
BELAIR-EDISON                           23989          818
COLDSTREAM HOMESTEAD MONTEBELLO         15803          401
DOWNTOWN                                12961         1746
CANTON                                  12719          717
BROOKLYN                                11452          711
CENTRAL PARK HEIGHTS                     9748          477
WASHINGTON VILLAGE/PIGTOWN               9214          475
FRANKFORD                                8919          741
BROADWAY EAST                            9218          407
HAMPDEN                                  8857          383

REPORT 2: Crime and 311 Association
Pearson correlation: 0.777

REPORT 3: High Activity in Both Datasets
Mean 311 requests: 2400.4   Mean crime records: 153.3   Neighborhoods above both means: 78
BELAIR-EDISON  Requests: 23989  Crimes: 818  ... (top 10 listed)

REPORT 4: 311 Categories in High-Activity Neighborhoods
SW-RAT RUBOUT PROACTIVE | ################################### 89066
TRS-PARKING COMPLAINT | #############                       31893
HCD-SANITATION PROPERTY | #########                           21785
SW-DIRTY ALLEY | #######                             17858
SW-HGW       | ######                              16371
... (10 bars)

REPORT 5: Monthly Activity Trends
311 SERVICE REQUESTS   2026-01 | ##### 66819 ... 2026-06 | ##### 79967 ... 2026-09 | 74947
CRIME RECORDS          2026-01 | 4298 ... 2026-08 | 5095 ... 2026-09 | 5028

Analysis completed successfully.
```
Prebuilt `baltimore-engine-mac-arm64`: not executable as shipped (mode 0666, exit 126). An executable copy in the scratchpad was SIGKILLed by macOS on launch (exit 137, quarantine/signature enforcement), so the prebuilt binaries could not be verified; the source build works, which is what was graded.

## Concurrency Analysis

Primitives: goroutines, buffered channels, `sync.WaitGroup`.
- `main.go:211-232`: two goroutines, one per dataset (`processDataset`), each streaming its CSV and building its own count maps; results sent on a buffered channel, closer goroutine after `wg.Wait()`.
- `analysis.go:29-148` `aggregateConcurrently`: a bounded pool of up to 8 workers reads `MonthRecord` jobs from a channel and keeps local maps, merged after `wg.Wait()`. However, the jobs are the already-aggregated `neighborhood|month` counts (12,612 + 2,787 keys), so the pool performs ~15 k trivial map updates — the expensive work (parsing 1.2 M CSV rows) is done sequentially inside each of the two file goroutines.
- Synchronization is correct (no shared writes; race detector clean). Report 4's type counts come straight from the ingest goroutine's `Types` map, not from the pool.
Level: **partial** — two real goroutines for parsing plus a correctly built but underused worker pool for aggregation; no parallel parsing within a file.

## Memory/Resource Analysis

- Streams CSV row by row (`csv.Reader.Read` loop, `main.go:106-160`), `defer file.Close()`, only aggregate maps retained (no raw rows), bounded worker count, buffered channels. README's "Resource Management" section lists these.
- Observed max RSS 18 MiB for 455 MB of CSV — genuinely frugal.
- Not done: `ReuseRecord`, string interning (keys are freshly concatenated `neighborhood|month` / `neighborhood|type` strings per row), no runtime memory measurement or discussion of GC. The `Types` map keys are built for every 311 row even outside the analysis period (filtered by month before insert, but the concatenation still allocates).
Some evidence, documented at a basic level: mid score.

## Architecture & Code Quality

Single `main` package split into `main.go` (ingest + wiring), `analysis.go` (aggregation + five report functions), `dates.go` (date parsing). Reasonable separation for a small program, readable names, deterministic sorting with tie-breaks, honest caveat lines printed under each report. Weaknesses: dataset-specific behavior is dispatched on the display name string (`if name == "Crime Incidents"` / `"311 Requests"`, `main.go:77,96`), which is brittle; reports print directly rather than returning data; no named design pattern is claimed or realized (the README does not mention one); one small test. Data flow is encoded as `"a|b"` string keys split back with `SplitN`, which works but is a weak substitute for struct keys. README findings (0.777, Belair-Edison, SW-RAT RUBOUT PROACTIVE) match the output exactly.

## Stakeholder Questions & Value Assessment

Stakeholders: none named — README says "municipal service requests and reported crime" and "municipal leaders" only implicitly.
1. Highest combined 311 + crime volume (sum of counts) — generic ranking.
2. Pearson correlation between neighborhood 311 and crime counts (0.777) — cross-dataset but shallow; largely reflects neighborhood size.
3. Neighborhoods above both means — cross-dataset selection, 78 of 279.
4. 311 types in those high-activity neighborhoods — the top category is `SW-RAT RUBOUT PROACTIVE` (89,066), a city-initiated sweep, which the README reports as the headline finding without noting it is not resident demand.
5. Monthly 311 vs crime counts side by side — two single-dataset trend charts.
The join is a neighborhood-level count join; there is no normalization for size, no rates, no per-type or per-time cross analysis beyond the above. A Mayor's Office would learn that big neighborhoods have more of everything. Visualization: ASCII `#` bars in Reports 4 and 5, tables elsewhere.

## Issues

- No stakeholders named; questions are count-based and mostly generic.
- Report 4's headline category is a proactive city program, misread as neighborhood demand.
- Worker pool operates on pre-aggregated keys; CSV parsing is not parallelized.
- Prebuilt binaries shipped without execute permission and could not be run (SIGKILL on an executable copy); not deducted since source builds.
- `go 1.27.2` in go.mod forces a toolchain download on older Go installs.
- README has no explicit "verify" step beyond running tests; `run.sh` works as documented.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | Clean first-try `go build`, vet clean, race build clean. |
| real_data | 15 | 15 | Two real Open Baltimore datasets (NIBRS crime 278,239 rows; 311 2026 943,283 rows) bundled and used; no mock path. |
| concurrency | 15 | 7 | Two goroutines parse the two files sequentially each; the bounded worker pool only re-aggregates ~15 k pre-aggregated keys. Correct sync, race clean, but no parallel parsing. |
| memory | 10 | 6 | Streaming reads, defer close, counts-only retention (18 MiB RSS), documented briefly; no ReuseRecord/interning/measurement. |
| architecture | 15 | 8 | Three-file single package with reasonable separation and one small test; string-dispatch on dataset name, `a|b` string keys, no named pattern realized. |
| stakeholder | 15 | 6 | Five questions but no stakeholders named, mostly raw-count rankings and a size-driven correlation; proactive sweeps misread as demand; ASCII bars present. |
| runs_results | 10 | 10 | Runs end-to-end on bundled data in 1.6 s and prints all five reports. |
| readme | 5 | 4 | macOS run via `bash run.sh`, build/test commands, findings match output; thin on verification. |
| **Total** | **100** | **71** | **C** |

## Final Grade Rationale

The program builds cleanly and runs end-to-end on two bundled real datasets, which secures the build, data and run points. It loses most of its points on depth: concurrency is limited to one goroutine per file plus a pool that does trivial work on already-aggregated maps; memory handling is sensible but undocumented beyond bullet points; the code is a small single-package program without a realized design pattern; and the five questions are count-based rankings without named stakeholders or size normalization, with one finding that mistakes a city-initiated program for resident demand. 71/100, C.
