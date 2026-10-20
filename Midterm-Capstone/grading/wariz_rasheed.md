# Wariz Rasheed — COSC 352 Midterm Capstone Grading Report

**Summary: 99/100 (A).** Go. Three real Open Baltimore datasets (943k 311 rows, 278k NIBRS crime rows, 31k liquor-license rows; ~458 MB) are bundled and processed by a genuinely concurrent, multi-package pipeline (streaming CSV → worker-pool parse → fan-out/fan-in join on a normalized street-block key) in about 1.1 s, with Builder/Strategy/Factory/Pipeline patterns actually realized, 15 passing tests (race-clean), and five well-motivated cross-dataset questions answered with ASCII and HTML charts. Everything the README says is true.

## Inventory

Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/wariz_rasheed/baltimore-intel/`

| File | LOC | Role |
|---|---|---|
| main.go | 201 | flags, file resolution, Builder wiring, report factory, MemStats footer |
| internal/model/model.go | 198 | compact row types, per-worker `Acc`, `Dataset`, merge logic |
| internal/geo/geo.go (+_test 79) | 161 | address → 100-block key normalisation, block→neighborhood majority-vote index |
| internal/ingest/csv.go (+parsers.go 166, _test 143) | 209 | generic streaming `Load[T]` (reader goroutine → batches → N workers), string interner, 3 parsers |
| internal/pipeline/pipeline.go (+_test 155) | 384 | Builder, 4 stages, generic `fold` fan-out/fan-in |
| internal/analysis/*.go (5 questions + analysis.go, _test 108) | 851 | Strategy interface `Analysis`, `RunAll` concurrent |
| internal/report/{report,terminal,html}.go | 371 | Renderer Factory: ASCII bars/tables, HTML template |
| internal/stats/stats.go (+_test 94) | 148 | Spearman/Pearson/partial Spearman, median, percentile ranks |
| README.md (12 KB), run.sh, go.mod (`go 1.21`), data/README.txt | | |
| data/311_Customer_Service_Requests_2026.csv | 379 MB, 943,284 lines | |
| data/NIBRS_GroupA_Crime_Data_1100291637803527672.csv | 72 MB, 278,240 lines | |
| data/Licenses.csv | 6.8 MB, 30,969 lines | |
| internal/pipeline/testdata/{crime,licenses,requests}.csv | 2,283 / 909 / 300 lines | real slices for tests |
| output/report.html | 27 KB | pre-generated HTML report |

Total 3,259 LOC in 21 Go files (2,580 non-test). Tests: 6 test files. Run script: `run.sh`. Standard library only.

## Datasets & Data Realness

All three headers match Open Baltimore schemas exactly:
- 311: `X,Y,RowID,SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,...,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord`. Sample row: `26-00004287, SW-Bulk Scheduled-Saturday, 2026/01/03 14:44:56+00, Closed, Solid Waste, "3954 DOLFIELD AVE, Baltimore City, 21215", 21215, Dolfield, 6, Northwestern, 624, 39.3403..., -76.6828...`. Real Salesforce-style SR IDs, real addresses, real neighborhoods.
- NIBRS: `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Shooting,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y`. Rows span 2022-01 through 2026-10 (e.g. `261003876, 10/11/2026 6:00:00 AM, 13B, COMMON ASSAULT, 800 N CHARLES ST, CENTRAL, MOUNT VERNON`). Real.
- Licenses: `LLKey,CorpKey,LicenseClass,SubClass,LicenseNumber,LicenseDate,LicenseEndDate,LicenseYear,...,LicenseStatus,LicenseeFirstName,LicenseeLastName,TradeName,CorpName,EstablishmentDesc,...,AddrStreet,AddrZip,ESRI_OID`. Rows such as `RED ROOSTER, INC., Tavern, 6412-16 FRANKFORD AVENUE` and `HARBOR EAST DELICATESSEN, 1006 ALICEANNA STREET`. Real BLLC liquor-license export.

Row counts (wc -l minus header): 943,283 / 278,239 / 30,968 — matching the program's "Rows read" table exactly. No mock or sample generator exists; the only small files are real slices used by tests. `data_status: real_bundled`.

## Build Results

```
cd submissions/wariz_rasheed/baltimore-intel
go build -o <scratch>/baltimore-intel .    # exit 0, 1.3 s, no output
go vet ./...                               # exit 0, no output
go build -race -o baltimore-intel_race .   # exit 0
```
Clean first try. (`./run.sh` would do `go build -o bin/baltimore-intel . && exec ./bin/baltimore-intel`; I built to the scratchpad instead to avoid writing into the submission.)

## Test/Vet/Race Results

```
go test ./...
ok  baltimoreintel/internal/analysis  0.930s
ok  baltimoreintel/internal/geo       0.410s
ok  baltimoreintel/internal/ingest    0.228s
ok  baltimoreintel/internal/pipeline  1.345s
ok  baltimoreintel/internal/stats     1.098s
go test -race ./...     all ok
./baltimore-intel_race (full 458 MB data)   exit 0, 12.0 s, max RSS 1.53 GB, 0 "DATA RACE" reports
```
Tests include an end-to-end run on real slices asserting exact join-coverage counts (2282 crime rows read / 2279 located, 829 active licenses / 237 located) and a 1-worker vs 8-worker equivalence check.

## Run Results

```
/usr/bin/time -l ./baltimore-intel -out <scratch>/wariz/output      # run from project dir
real 1.59 s, user 2.97 s, max RSS 574 MB, exit 0
stderr: [1/4] Ingesting 3 datasets (10 workers per file)... [2/4] Analysis window: 2026-01-01 to 2026-10-11 ... Running 5 analyses concurrently...
```
Captured stdout (279 lines) is in the scratchpad `wariz/run.txt`; the HTML written is byte-identical to the shipped `output/report.html` except the stage-timing footer. Opening lines:
```
==============================================================================
Baltimore City Cross-Dataset Intelligence Report
Analysis window: 2026-01-01 to 2026-10-11 (UTC). Datasets: NIBRS Group A crime, 311 service requests, liquor licenses.
==============================================================================
  Datasets and join coverage
  Dataset                   File                                             Rows read  Rows parsed  In window  Placed in a neighborhood
  NIBRS crime               NIBRS_GroupA_Crime_Data_1100291637803527672.csv  278,239    278,239      44,485     44,480 (100.0%)
  311 requests              311_Customer_Service_Requests_2026.csv           943,283    943,283      655,510    643,404 (98.2%)
  Liquor licenses (active)  Licenses.csv                                     30,968     30,968       2,233      2,091 (93.6%)
```
The five findings:
1. **Q1 (311 × crime, for DPW/DOT)** — "Across 268 neighborhoods, total 311 volume has a very strong positive relationship with violent crime (rho = 0.76), largely because busy neighborhoods report more of everything. After removing that size effect, the request type that tracks violent crime most closely is 'HLTH-Animal Police/Fire/Sheriff/City Work Crew Standby' (rho = 0.44, 596 requests)." Followed by top-10 bar chart (SW-Dirty Street Proactive 0.38, SW-Cleaning 0.34, TRM-Pickup Pothole 0.33, HCD-Illegal Dumping 0.32 ...).
2. **Q2 (licenses × crime, for Liquor Board/BPD)** — rho = 0.46; "6+ licenses" tier averages 15.8 night-time violent crimes vs 3.9 in "0 licenses" tier; night share 35.6% vs 29.6%. Table of top license neighborhoods (Fells Point 153 licenses, 59.5% of violent crime at night; Canton 149; Downtown 140, 372 violent).
3. **Q3 (311 × crime, for Mayor's Office equity)** — highest-crime quartile closes 311 requests in median 1.3 days vs 2.2 days in lowest quartile; rho = −0.11 (weak negative) — i.e. no evidence of slower service in high-crime areas; open-request share 4.1% vs 6.8%.
4. **Q4 (all three, block-level, for Planning/Liquor Board)** — "The 805 street blocks with a liquor license average 1.42 violent crimes per block, versus 0.25 on the 22,650 blocks without one (5.7x as many). The license type with the highest average is 'Arena' (4.33)."
5. **Q5 (all three, for Mayor's Office)** — weighted priority index: "Of 268 neighborhoods ranked, Downtown scores highest (95.9 of 100), followed by Carrollton Ridge and Frankford." Top-15 chart with the factors driving each.
Footer: `Join: 19,566 street blocks indexed; 287,773 311 call-centre information rows (ECC) excluded. Stage timings: ingest 837ms; index 91ms; join 52ms. Run: 1.053s total, 10 workers per stage, heap in use 335 MB, 16 GC cycles.`

## Concurrency Analysis

- **Ingest** `internal/ingest/csv.go:117-205`: per file, the calling goroutine streams `csv.Reader` records into batches of 2,048 and sends them on `batches` (cap `workers*2`); N worker goroutines parse into private `[]T` with a private `Interner`, send on `results` (cap `workers`); `sync.WaitGroup` + channel close; fan-in concatenation. Three files are loaded concurrently (`pipeline.go:118-134`).
- **Index & join** `pipeline.go:166-200` + generic `fold` `pipeline.go:216-246`: items split into one contiguous chunk per worker, each worker fills a private accumulator (`geo.Votes` / `*model.Acc`), merged on the caller — no shared mutable state, no locks. The three dataset folds themselves run in three goroutines.
- **Analysis** `internal/analysis/analysis.go:36-49`: five Strategies run in five goroutines over the immutable `Dataset`, each writing its own slot of the result slice.
- Correct sync everywhere; `go test -race` and a full-data `-race` run are clean. This is parallel parse **and** parallel join/aggregate with worker pools: full marks.

## Memory/Resource Analysis

- Streaming CSV through `bufio.NewReaderSize(f, 1<<20)`; file never loaded whole (`csv.go:133`).
- Compact typed rows (`model.go`) keep only needed fields; per-worker string interning with `strings.Clone` to avoid pinning whole CSV rows (`csv.go:52-72`, well explained).
- Bounded channels; `debug.SetGCPercent(200)` with a comment on the trade-off (`main.go:47`); `runtime.ReadMemStats` reported in the footer; raw slices allowed to go out of scope after the join with an explicit comment (`pipeline.go:209`).
- Measured: 574 MB max RSS for 458 MB of input; 335 MB heap in use at the end. README has a dedicated "Memory and resource management" section that matches the code. Full marks.

## Architecture & Code Quality

Multi-package layout under `internal/` with clean dependency direction (model ← geo/ingest ← pipeline ← analysis/report ← main). Patterns are real, not decorative: `pipeline.Builder` (`pipeline.go:36-80`), `analysis.Analysis` Strategy interface with five implementations and `All()`, `report.New("terminal"|"html")` Renderer Factory (`report.go:49-58`), and an explicit four-stage Pipeline. Generics (`Load[T]`, `fold[T,A]`) remove duplication across datasets. Naming, comments and doc strings are professional; output is deterministic (sorted names, alphabetical tie-breaks). The join design — normalizing every address to a 100-block key and building a majority-vote block→neighborhood index so licenses (which have no neighborhood field) can be placed — is the most thoughtful join among the submissions and is unit-tested across all three address styles. The code is polished enough that AI assistance is plausible, but README, tests and code are fully consistent with each other and with the data, and every claim was verified at runtime.

## Stakeholder Questions & Value Assessment

Each question names a stakeholder (DPW/DOT; Liquor Board/BPD; Mayor's Office equity; City Planning/Liquor Board; Mayor's Office), is genuinely cross-dataset (Q4 and Q5 use all three), and ships with a headline finding, chart, tables and method caveats. Q3 is actionable precisely because it refutes an intuitive equity hypothesis with data (high-crime neighborhoods are *not* waiting longer). Q4's block-level 5.7x finding and Q5's priority index are directly usable by a planning office, and the weights are exposed as a policy choice. Q1's headline ("Animal ... Standby") is a noisy pick even after size adjustment — the dirty-street / illegal-dumping signals right below it are the more meaningful story — and Q4 acknowledges its unlicensed-block baseline bias. Minor deduction for that; otherwise this is exactly what the assignment asked for.

## Issues

- Q1's top size-adjusted correlate is a low-volume, oddly specific request type; the finding sentence would be stronger leading with the cleaner "SW-Dirty Street"/"HCD-Illegal Dumping" signals.
- Q4 unlicensed-block average is biased upward (blocks with zero events are absent), as the report itself notes.
- Neighborhood counts not population-adjusted (acknowledged).
- `run.sh` builds into `bin/` inside the project (fine, just noted).
- Shipped `output/report.html` is a pre-generated artifact; it matches a fresh run exactly.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build` and `go vet` clean first try with the documented command; stdlib only. |
| real_data | 15 | 15 | Three real Open Baltimore exports bundled (943k/278k/31k rows), schemas and records verified; no mock path. |
| concurrency | 15 | 15 | Worker-pool streaming parse, fan-out/fan-in join with private accumulators, concurrent analyses; race detector clean on full data. |
| memory | 10 | 10 | Streaming, interning, compact rows, bounded channels, GC tuning, MemStats footer; documented and measured (574 MB RSS for 458 MB input). |
| architecture | 15 | 15 | Clean `internal/` packages, Builder/Strategy/Factory/Pipeline realized, generics, 15 tests incl. end-to-end on real slices. |
| stakeholder | 15 | 14 | Five named-stakeholder, genuinely cross-dataset questions with charts/tables/caveats; Q1 headline is a noisy pick. |
| runs_results | 10 | 10 | Runs end-to-end in 1.6 s, output sensible and identical to the shipped report. |
| readme | 5 | 5 | Detailed macOS build/run/verify, flags, data instructions, architecture and memory sections — all accurate. |
| **Total** | **100** | **99** | **A** |

## Final Grade Rationale

This submission meets every requirement of the specification with real evidence: three large real Open Baltimore datasets bundled and verified; parallel parsing and parallel join/aggregate with correct synchronization that passes the race detector on the full data; explicit, measured memory management; a multi-package architecture in which the named patterns actually exist; five cross-dataset questions for named stakeholders with ASCII and HTML visualizations; and a README whose every instruction and claim checked out. One point withheld on stakeholder insight for a noisy Q1 headline. 99/100, A.
