# Anaya Efferson — COSC 352 Midterm Capstone Grading Report

**Summary: 71/100 (C)** — Go. Builds and runs cleanly on two bundled real Open Baltimore datasets (311 requests 2025, Vacant Building Notices) with a per-dataset worker pool for counting and a neighborhood count join, printing five short answers and one ASCII chart. Two of the five questions are single-dataset top-5 lists, no stakeholders are named beyond "city officials", there are no tests, and the architecture is a three-file `main` package.

## Inventory

- Location: `submissions/anaya_efferson/BaltimoreDataEngine/`
- Language: Go (`go.mod` module `baltimoredataengine`, `go 1.27.2` — go1.24 auto-downloaded the 1.27.2 toolchain)
- Source: `main.go` (251), `ingestion.go` (68), `processing.go` (148) — 467 LOC, single package `main`. No tests.
- README.md (~130 lines), `run.sh` (`go build -o baltimore-engine .` then `./baltimore-engine`).
- Prebuilt binaries shipped: `baltimore-engine-macos` (arm64, 2.6 MB), `baltimoredataengine.exe` (Windows). Built with go1.27.2. Mode `-rw-rw-rw-` (no execute bit), quarantine attribute set.
- Data bundled: `data/service_requests.csv` 382 MB, `data/vacant_properties.csv` 1.4 MB.

## Datasets & Data Realness

| File | Rows (data) | Header / records |
|---|---|---|
| `data/service_requests.csv` | 931,324 | `SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,LastActivity,LastActivityDate,Outcome,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord,ESRI_OID,x,y`; e.g. `25-00360639,HCD-Illegal Dumping,Phone,4/28/2025 4:31:03 PM,Closed (Transferred),...,"1713 N CHAPEL ST, Baltimore City, 21213",21213,Broadway East,12,Eastern,323`. Byte-identical header and row count to the 311 Customer Service Requests 2025 file I downloaded from Open Baltimore today. |
| `data/vacant_properties.csv` | 11,445 | `X,Y,OBJECTID,NoticeNum,DateNotice,DateCancel,DateAbate,NT,OWNER_ABBR,HousingMarketTypology2023,Council_District,Neighborhood,BLOCKLOT,Address`; e.g. `2655955A,2026/05/28 13:13:00+00,,,Vacant,,J,9,Carrollton Ridge,0702 050,2113 WILKENS AVE`. Matches the DHCD Vacant Building Notices layer (same NoticeNum/Address as the GeoJSON in another submission). Top neighborhoods Carrollton Ridge 744, Broadway East 717, Sandtown-Winchester 587. |

Both real, bundled, no mock/sample generator. Data status: **real_bundled**.

## Build Results

```
$ go build -o baltimore-engine .     (output redirected to scratchpad)
exit=0   0.17 s
$ go vet ./...                       exit=0, no output
$ go build -race -o baltimore-engine_race .   exit=0
```
Clean build, no warnings.

## Test / Vet / Race Results

```
$ go test -count=1 ./...     ?   baltimoredataengine  [no test files]
$ ./baltimore-engine_race  (full data)   exit=0, 0 "DATA RACE", 12.5 s
```
Tests: none. Race detector: **clean** on a full run.

## Run Results

Run from the student directory (relative `data/` paths), binary from scratchpad:
```
$ /usr/bin/time -l baltimore-engine
exit=0   1.23 s real   max RSS 13 MiB (13,844,480 bytes)
```
Full output (`scratchpad/anaya/run.txt`):
```
Baltimore City Data Analysis Engine
Datasets processed successfully!
311 Records: 931324
Vacant Property Records: 11445

QUESTION 1: Which neighborhoods have the most 311 requests?
BELAIR-EDISON 23676 / DOWNTOWN 14273 / CANTON 12161 / COLDSTREAM HOMESTEAD MONTEBELLO 11025 / BROOKLYN 9478

QUESTION 2: Which neighborhoods have the most vacant properties?
CARROLLTON RIDGE 744 / BROADWAY EAST 717 / SANDTOWN-WINCHESTER 587 / HARLEM PARK 438 / MIDTOWN-EDMONDSON 336

QUESTION 3: What is the relationship between 311 requests and vacant properties?
Correlation coefficient: 0.451
Weak to moderate positive relationship

QUESTION 4: Which neighborhoods have the highest 311 requests per vacant property?
RIVERSIDE Ratio: 5963.00 / SOUTH BALTIMORE 3864.00 / BOLTON HILL 3196.00 / ROLAND PARK 2736.00 / WEST HILLS 2373.00

QUESTION 5: Which neighborhoods should city officials prioritize?
BROADWAY EAST 68.09 / CARROLLTON RIDGE 63.89 / SANDTOWN-WINCHESTER 58.11 / BELAIR-EDISON 54.97 / HARLEM PARK 41.77

ASCII CHART: Top 5 Neighborhoods by Vacant Properties
CARROLLTON RIDGE          | ############################## 744
BROADWAY EAST             | ############################   717
SANDTOWN-WINCHESTER       | #######################        587
HARLEM PARK               | #################              438
MIDTOWN-EDMONDSON         | #############                  336
```
Output matches the README's example results exactly (68.09, 63.89, 0.451). Prebuilt `baltimore-engine-macos`: not executable as shipped (exit 126); an executable copy was SIGKILLed by macOS (exit 137), so it could not be verified. Source build graded.

## Concurrency Analysis

Primitives: goroutines, bounded buffered channels, `sync.WaitGroup`.
- `main.go:129-140`: the two datasets are processed concurrently (one goroutine each), results via channels.
- `processing.go:10-74` `processDataset`: a reader goroutine (`streamNeighborhoods`, ingestion.go) parses CSV rows sequentially and sends each neighborhood string on a `workers*2`-bounded channel; `runtime.NumCPU()` workers each increment a private map; results merged after `wg.Wait()`. The pool is structurally correct but each worker only does `map[string]int++`; CSV decoding (the expensive part) is single-threaded per file, and one channel send per row (931 k sends) dominates.
- `processing.go:78-148` `combineDataParallel`: a worker pool over ~280 neighborhood names performing map lookups — token concurrency for the join.
- No shared writes; race detector clean.
Level: **partial** — a real fan-out pool in the ingest stage of both datasets, but doing trivial work; parsing itself not parallel.

## Memory/Resource Analysis

- Streams CSV one row at a time, `defer file.Close()`, bounded channels, only per-neighborhood counts retained (13 MiB max RSS for 383 MB of CSV). README mentions these and "Go's garbage collector manages heap memory automatically".
- Not done: `ReuseRecord`, interning, measurement; `strings.ToUpper(TrimSpace(...))` allocates a new string per row that is then sent through a channel. Some evidence, lightly documented.

## Architecture & Code Quality

Three files in one package: `ingestion.go` (CSV streaming), `processing.go` (worker pool and join), `main.go` (five questions computed inline in `main`, plus formatting helpers). README claims the Pipeline pattern; the flow stream -> workers -> merge -> join -> print is loosely pipeline-shaped but is not expressed as composable stages, and all analysis and output live in one 140-line `main`. Code is clean and readable with deterministic sorting of the joined slice. No tests. The README is internally consistent with the code and output.

## Stakeholder Questions & Value Assessment

Stakeholders: "municipal government", "city officials", "municipal leaders" — none specific.
1. Top 5 neighborhoods by 311 requests — single dataset.
2. Top 5 by vacant notices — single dataset.
3. Pearson correlation of 311 vs vacant counts (0.451) — cross-dataset, shallow.
4. 311 requests per vacant property — cross-dataset but degenerate: ranks neighborhoods with 1 vacant notice (Riverside 5963:1), which is not a useful signal.
5. Priority score = 50% normalized 311 + 50% normalized vacants — cross-dataset and at least produces a plausible shortlist (Broadway East, Carrollton Ridge, Sandtown-Winchester) that overlaps with other submissions' vacancy findings.
No size normalization, no request-type filtering (proactive sweeps included), no time dimension. One ASCII chart (top-5 vacants). A planning office would learn which neighborhoods have the most open vacant notices, which the source dataset already shows directly.

## Issues

- Two of five questions are single-dataset top-5 lists; Q4 ratio is dominated by neighborhoods with a single vacant notice.
- No stakeholders named; no tests.
- Worker pool does trivial counting; CSV parsing sequential per file; per-row channel sends.
- All five analyses inline in `main()`; README's claimed Pipeline pattern is only loosely realized.
- Prebuilt binary shipped without execute permission and SIGKILLed when copied/executed; not deducted (source builds).
- `go 1.27.2` in go.mod forces a toolchain download on older Go installs.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | Clean first-try `go build`, vet clean, race build clean. |
| real_data | 15 | 15 | Two real Open Baltimore datasets (311 2025, 931,324 rows; Vacant Building Notices, 11,445 rows) bundled and used; no mock path. |
| concurrency | 15 | 8 | Fan-out worker pool per dataset with correct WaitGroup/channel sync and race-clean, both datasets concurrent; but workers only count strings, parsing is sequential, join pool is token. |
| memory | 10 | 6 | Streaming reads, bounded channels, counts-only retention (13 MiB RSS), briefly documented; no ReuseRecord/interning/measurement. |
| architecture | 15 | 7 | Three-file single package with ingestion/processing separated, but all analysis and output inline in main, Pipeline claim only loosely realized, no tests. |
| stakeholder | 15 | 6 | Five questions but two single-dataset, Q4 degenerate, no named stakeholders, one ASCII chart; Q5 priority score is a reasonable cross-dataset shortlist. |
| runs_results | 10 | 10 | Runs end-to-end on bundled data in 1.2 s and prints all five answers matching the README. |
| readme | 5 | 4 | macOS `run.sh`/`go run .`, build verification and example results that match; verify/test guidance is thin ("if test files have been added"). |
| **Total** | **100** | **71** | **C** |

## Final Grade Rationale

The submission compiles cleanly, bundles two real Open Baltimore datasets, and runs to completion with output matching its README, which earns full marks on build, data and run. Depth is limited: the worker pools are correctly synchronized but do trivial work while CSV parsing stays sequential; memory handling is sensible but minimally documented; the code is a small single-package program with analyses inline in `main` and no tests; and the questions are mostly simple rankings, two of them single-dataset, without named stakeholders or size normalization. 71/100, C.
