# Elizabeth Stewart — COSC 352 Midterm Capstone Grading Report

**Summary: 95/100 — A.** Well-engineered Go pipeline over two real, bundled Open Baltimore datasets (NIBRS Group A crime, 54,076 rows; Vacant Building Notices, 11,445 rows). Concurrent CSV readers feed bounded channels into parsing worker pools and hash-partitioned join/aggregation workers; race-clean, deterministic across worker counts, tested, and documented. Five genuinely cross-dataset questions for a City Planning stakeholder with tables and an ASCII monthly chart.

## Inventory
Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/elizabeth_stewart/`

| File | LOC / size | Notes |
|---|---|---|
| cmd/baltimore/main.go | 28 | flags, calls analysis.Load/Report |
| internal/analysis/analysis.go | 350 | pipeline + reports |
| internal/analysis/analysis_test.go | 109 | 4 tests incl. serial-vs-parallel equivalence |
| go.mod | module baltimore-insights, go 1.22 | |
| README.md | 7.7 KB | |
| .gitignore | /bin/, /reports/ | |
| data/crime.csv | 14.7 MB, 54,077 lines | real |
| data/vacant.csv | 1.5 MB, 11,446 lines | real |

Language: Go. Source LOC 487 (3 files). Tests: yes. No run script (commands documented).

## Datasets & Data Realness
```
crime.csv header:  RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Shooting,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y
row: 37174,251209209,12/31/2025 2:50:00 PM,120,ROBBERY,O,FIREARM,,217,F,38,WHITE,NOT_HISPANIC_OR_LATINO,1400 DUNDALK AVE,N/A,SOUTHEAST,O'DONNELL HEIGHTS,39.277261,-76.533707,...
vacant.csv header: X,Y,OBJECTID,NoticeNum,DateNotice,DateCancel,DateAbate,NT,OWNER_ABBR,HousingMarketTypology2023,Council_District,Neighborhood,BLOCKLOT,Address
row: 1411570.99,588061.84,7,2655955A,2026/05/28 13:13:00+00,,,Vacant,,J,9,Carrollton Ridge,0702 050,2113 WILKENS AVE
```
Both match the Open Baltimore NIBRS Group A Crime and Vacant Building Notices schemas exactly (CCNumber, CrimeDateTime, New_District, PremiseType; NoticeNum, DateNotice, BLOCKLOT). Real addresses and neighborhoods (O'Donnell Heights, Oakenshawe, Carrollton Ridge, 1,396 Canton crime rows, 587 Sandtown vacant rows). Row counts: crime 54,076; vacant 11,445 (matches README). README links both portal dataset pages. No mock generator exists; test fixtures are built in-memory in tests only.
**data_status: real_bundled.**

## Build Results
```
$ go build -o <scratch>/bin/baltimore ./cmd/baltimore   -> exit 0 (1.2 s, first try)
$ go build -race -o <scratch>/bin/baltimore-race ./cmd/baltimore -> exit 0
```

## Test / Vet / Race Results
```
$ go vet ./...            -> exit 0
$ go test -race ./...     -> ok  baltimore-insights/internal/analysis  1.588s   (cmd: no test files)
```
Race-instrumented full run on real data: **0 DATA RACE** reports (1.9 s, 141 MB RSS). `-workers 1` vs `-workers 4` outputs: **IDENTICAL** (diff empty), as the README predicts.

## Run Results
Documented command: `./bin/baltimore -year 2025 -workers 4 -top 10 > reports/2025.txt`. Wall **1.05 s**, max RSS **37 MB**, exit 0. Output (scratchpad `elizabeth/2025.txt`):
```
BALTIMORE CRIME AND VACANCY REPORT — 2025
Crime rows read: 54076 | in year: 54068 | missing neighborhood: 17
Vacant notice rows: 11445 | dated after analysis year: 1549

1. Where are the most notices, and how many crime records occur there?
CARROLLTON RIDGE     744 notices |  471 records |  408 cases
BROADWAY EAST        717 notices |  387 records |  352 cases
SANDTOWN-WINCHESTER  587 notices |  701 records |  610 cases
HARLEM PARK          438 notices |  415 records |  349 cases
... (10 rows)

2. How concentrated are notices, and what share of crime cases occurs in those areas?
Top  10% ( 22 of 213)   59.8%   14.2%
Top  25% ( 54 of 213)   85.3%   34.9%
Top  50% (107 of 213)   96.2%   62.0%
Top 100% (213 of 213)  100.0%   91.9%

3. What crime types are reported in neighborhoods with versus without notices?
TYPE                 WITH NOTICES  WITHOUT NOTICES
AGG. ASSAULT         4664          249
COMMON ASSAULT       8752          651
... (27 types)

4. How do monthly crime records vary in neighborhoods with notices?
January  3666 #################################
...
September 4420 ########################################
December 4080 ####################################

5. Which areas rank highest on a combined planning score?
CARROLLTON RIDGE   61.6 / 100 | 744 notices | 408 cases
BROADWAY EAST      58.2 / 100 | 717 notices | 352 cases
SANDTOWN-WINCHESTER 56.8 / 100 | 587 notices | 610 cases
DOWNTOWN           53.4 / 100 | 50 notices | 1757 cases
... (10 rows)
```

## Concurrency Analysis
- `analysis.go:57-64`: two reader goroutines (one per dataset) under a `sync.WaitGroup`; a closer goroutine closes `out`/`errs` after both finish.
- `analysis.go:159-215` (`read`): per-dataset **parsing worker pool** of `workers` goroutines consuming a bounded `jobs` channel (cap 256); first-error captured under `sync.Mutex`; `wg.Wait()` before return.
- `analysis.go:66-81`: `workers` **join/aggregation goroutines**, each owning a bounded shard channel (cap 256); dispatcher routes events by FNV-32a hash of normalized neighborhood so both datasets' rows for a neighborhood land on the same worker (`aggregate`, 105-134) — a proper partitioned hash join with no shared mutable maps.
- Results merged via `partials` channel (ownership transfer). Race detector clean; deterministic output across worker counts verified by test `TestPartitionedJoinMatchesSerial` and by diff.
- Level: **full** — parallel parse and parallel join/aggregate with correct synchronization.

## Memory / Resource Analysis
- Streaming `csv.Reader`; raw rows never retained; bounded channels (256) provide backpressure (documented README "Architecture, concurrency and memory").
- Aggregates only per-neighborhood counts, type maps, `[12]int` months, and distinct-case sets; 37 MB RSS over 16 MB of CSV.
- Worker count bounded 1–64, default `min(NumCPU, 8)`; `defer f.Close()`. BOM stripped on header. README discusses GC, heap growth with distinct cases, and goroutine overhead.
- Minor: the FNV hasher is re-created per event (`fnv.New32a()` in a hot loop), a small allocation cost.

## Architecture & Code Quality
Pipeline pattern explicitly realized: read -> parse pool -> partitioned aggregate -> merge -> Report. `cmd/` vs `internal/` layout, exported `Load`/`Report`/`Normalize`. Strict input validation (required columns, date layouts, worker/year bounds) with fail-fast errors and no partial report. Tests cover pipeline correctness, bad inputs, concentration math, and serial/parallel equivalence. Weaknesses: `analysis.go` mixes join, aggregation and all five report renderers in one 350-line file (an `output` package would be cleaner); dense one-line struct declarations; `Report` closures are long. README is thorough but slightly over-hedged; `cd data-processing-engine` does not match the submitted folder name.

## Stakeholder Questions & Value Assessment
Stakeholder: fictional Baltimore City Planning team. All five questions join notices and crime by normalized neighborhood: (1) notices vs. crime records/cases per neighborhood, (2) concentration analysis — top 10% of vacancy neighborhoods hold 60% of notices but only 14% of distinct crime cases, (3) crime-type mix in notice vs. non-notice neighborhoods, (4) monthly crime seasonality restricted to notice neighborhoods, (5) a transparent combined planning score. Q2 and Q5 are genuinely actionable for targeting (e.g., Carrollton Ridge, Broadway East, Sandtown-Winchester rank top on both metrics; Downtown is crime-heavy but vacancy-light). Q3/Q4 are weaker (raw counts not normalized by group size, and Q4 is effectively single-dataset once filtered). Visualization: formatted tables plus one ASCII bar chart; a second chart for Q1/Q5 would help.

## Issues
1. README says `cd data-processing-engine`; the submission root has no such folder (trivial path mismatch).
2. All analysis + reporting in one file; no separate output package.
3. Q3/Q4 comparisons are unnormalized raw counts (README acknowledges).
4. Per-event `fnv.New32a()` allocation in dispatcher loop.
5. No run script (commands are in README only).

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | Clean `go build` first try with documented command. |
| real_data | 15/15 | Two real Open Baltimore datasets bundled (schemas verified), portal links given, no mock path. |
| concurrency | 15/15 | Parallel parse pools + hash-partitioned parallel join/aggregate with bounded channels, WaitGroup, Mutex; race-clean, deterministic. |
| memory | 10/10 | Streaming, bounded channels, aggregate-only retention, bounded workers; designed, documented, evident (37 MB RSS). |
| architecture | 13/15 | cmd/internal layout, realized Pipeline, strong tests; deduct for single large analysis.go mixing join and output. |
| stakeholder | 13/15 | 5 cross-dataset questions for a named stakeholder with concentration and scoring insight; Q3/Q4 shallower, only one chart. |
| runs_results | 10/10 | Runs end-to-end in ~1 s on bundled data, sensible output, identical across worker counts. |
| readme | 4/5 | Explicit macOS build/test/run/verify with expected row counts; minor folder-name mismatch. |
| **Total** | **95/100** | **A** |

## Final Grade Rationale
This submission meets every core requirement on real data: it compiles cleanly, uses real bounded worker pools for both parsing and a partitioned hash join, is race-detector clean with a test proving serial/parallel equivalence, documents memory behavior that is borne out at runtime, and answers five cross-dataset questions that a planning office could act on. Points were withheld only for packaging (analysis and output in one file), two shallower questions with a single visualization, and a trivial README path mismatch.
