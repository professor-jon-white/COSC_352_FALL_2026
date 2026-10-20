# Jordan Allen — COSC 352 Midterm Capstone Grading Report

**Summary: 95/100 (A)** — **LATE submission** (flagged, not deducted). Go, builds/vets/tests race-clean, runs in 0.24 s on bundled **real** Open Baltimore excerpts (one week of NIBRS crime + every-11th 311 request from the same week), answers five genuinely cross-dataset questions with key findings, tables and ASCII charts. README numbers reproduce exactly. Limitation: the bundled data is a one-week slice, so findings are demonstrative rather than citywide.

## Inventory

Path: `submissions/jordan_allen/baltimore-intel/`

| File | LOC | Notes |
|---|---|---|
| main.go | 250 | Config/flags, concurrent ingest of both datasets, join, 5 questions as goroutines, ReportBuilder + renderers, MemStats |
| ingest.go | 245 | FieldSpec/SourceSpec, column resolution, streaming reader, batch allocation, worker pool, merge |
| datasets.go | 276 | TimeFilter, crime/311 SourceSpec factories, CrimeAgg/RequestAgg aggregators (Strategy) |
| join.go | 71 | inner join on normalised neighborhood, quartile labelling |
| questions.go | 544 | `Question` interface + 5 implementations |
| stats.go | 118 | Pearson, Spearman, tied ranks, medians |
| report.go | 147 | Section/Report, ReportBuilder (Builder), TextRenderer/MarkdownRenderer (Strategy) |
| charts.go | 130 | bars, dual-hour histogram, tables |
| util.go | 134 | key/neighborhood normalisation, timeParser with layout memo, keyword lists |
| main_test.go | 322 | 14 tests |
| README.md | 12 KB | detailed; disclosure of Claude assistance and Windows-only testing |
| data/crime_sample_week.csv | 1,138 lines (310 KB) | **real** NIBRS excerpt, 1-7 June 2025 |
| data/requests_sample_week.csv | 1,541 lines (732 KB) | **real** 311 excerpt, every 11th request same week |
| data/crime_first50.csv | 51 lines | real, first 50 rows (Oct 2026) |
| data/requests_first50.csv | 51 lines | real, first 50 rows (Apr-May 2025) |

Total Go LOC: 2,237 (1,915 non-test). Go 1.21, stdlib only. No run script (plain `go build`; README gives the exact command). Tests: yes (14).

## Datasets & Data Realness

Declared: **NIBRS Group A Crime Data** and **311 Customer Service Requests 2025**, both CSV exports from data.baltimorecity.gov.

Headers and records are the genuine Open Baltimore schemas, unmodified:

```
RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Shooting,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y
71366,250600698,6/2/2025 12:00:00 PM,13B,COMMON ASSAULT,I,PERSONAL_WEAPONS,,421,F,12,BLACK_OR_AFRICAN_AMERICAN,NOT_HISPANIC_OR_LATINO,3600 HARFORD RD,N/A,NORTHEAST,MAYFIELD,39.331123,-76.579479,"(39.331123,-76.579479)",OTHER/RESIDENTIAL,1,...
71367,250600706,6/3/2025 1:44:00 AM,13B,COMMON ASSAULT,I,PERSONAL_WEAPONS,,427,F,23,...,4400 BOWLEYS LN,N/A,NORTHEAST,FRANKFORD,39.326485,-76.560307,...

SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,LastActivity,LastActivityDate,Outcome,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord,ESRI_OID,x,y
500VO00000ca08aYAA,25-00459888,TRS-48 Hour Parking/Abandoned Vehicle,Phone,6/2/2025 11:56:23 AM,Closed,6/5/2025 3:45:49 PM,...,"38 COBBER LN, Baltimore City, 21229",21229,Irvington,8,Southwestern,814,39.286485208855396,-76.688061877262029,...
500VO00000ca0rfYAA,25-00459904,SW-Rat Rubout Proactive,System,6/2/2025 12:00:16 PM,Closed,...,"302 MARYDELL RD, Baltimore City, 21229",21229,Irvington,8,Southwestern,815,...
```

Real CCNumbers, Salesforce-style SRRecordIDs (`500VO...`), real addresses, posts, districts, lat/lon; 198 neighborhoods join. No generator exists; no default data path (both flags required). The excerpts are small (1,137 + 1,540 rows) — the README is explicit that the full files are ~75 MB and ~380 MB and tells the user how to run on them with `-from/-to`.

Row counts (`wc -l`, incl. header): crime_sample_week.csv 1,138; requests_sample_week.csv 1,541; crime_first50.csv 51; requests_first50.csv 51.

**data_status: real_bundled** (small real excerpts).

## Build Results

```
$ cd submissions/jordan_allen/baltimore-intel
$ go build -o baltimore-intel .    # exit 0
$ go vet ./...                     # exit 0, clean
$ go build -race -o bi-race .      # exit 0
```

Clean first try (student noted she could only test on Windows; it builds identically on macOS).

## Test / Vet / Race Results

```
$ go test ./...          ok  baltimore-intel  0.229s
$ go test -race ./...    ok  baltimore-intel  1.352s
```

14 tests pass under the race detector. Race binary full run on the week files: exit 0, **0 DATA RACE**. Running with no flags gives a clear error: `error: both -crime and -requests are required (see README.md for an example)`.

## Run Results

Command (exactly as README): `./baltimore-intel -crime data/crime_sample_week.csv -requests data/requests_sample_week.csv -report report.md`

- Wall **0.24 s**; max RSS **10.6 MB**. Program-reported: heap in use 1.8 MiB, total allocated 5.3 MiB, 2 GC cycles, 10 workers/dataset.
- Captured: `scratchpad/jordan/run_stdout.txt`, `report.md`.

Header and five answers (abridged; full output ~200 lines in scratchpad):

```
 BALTIMORE CITY CROSS-DATASET INTELLIGENCE REPORT
 Crime: 1 file(s), 1137 rows read, 1137 accepted | rejected: 0 ...
 311:   1 file(s), 1540 rows read, 1540 accepted | rejected: 0 ...
 Join:  198 neighborhoods in both datasets | crime only: BARRE CIRCLE, ... (+15 more) | 311 only: BAYVIEW, ... (+30 more)
 Run:   10 CPU(s), 10 worker goroutines per dataset, batch 2048 rows, queue 8 batches | heap in use 1.8 MiB ...

 Q1 | For: Mayor's Office — Where do crime and 311 demand coincide?
  * Across 198 neighborhoods, crime and 311 volume correlate at Pearson r = 0.46 and Spearman rho = 0.40 (moderate positive).
  * 24 neighborhoods are in the top quartile for BOTH; together they hold 35.4% of joined crime and 31.4% of joined 311 requests.
  * Highest combined rank: BELAIR EDISON (crime rank 5, 311 rank 1).
  #   Neighborhood                  Crimes  Crime rank  311 reqs  311 rank
  1   BELAIR EDISON                 24      5           48        1
  2   FRANKFORD                     28      2           23        6
  3   DOWNTOWN                      52      1           21        9
  ... + two bar charts

 Q2 | For: Department of Public Works — Which service requests signal high-crime areas?
  * Compared 554 requests in the top-crime quartile with 862 elsewhere; types need >= 5 requests.
  * Most over-represented: SW-DIRTY ALLEY PROACTIVE (6.22x). Most under-represented: HCD-TREES AND SHRUBS (0.17x).
  SW-DIRTY ALLEY PROACTIVE  8  1.4%  0.2%  6.22x   HCD-MAINTENANCE STRUCTURE 6 1.1% 0.2% 4.67x   SW-RECYCLING 20 3.6% 1.2% 3.11x ...

 Q3 | For: Police Department and Transportation (staffing) — When do incidents and service requests peak?
  * Crime peaks at 22:00 (6.3%); 311 requests peak at 12:00 (12.9%). Profiles correlate at r = -0.28.
  * Evening holds 29.9% of crime but 13.1% of 311; overnight 27.6% vs 2.5%.
  (24-row dual histogram, █ crime / ▒ 311)

 Q4 | For: Mayor's Office (service equity) — Do high-crime neighborhoods wait longer for service?
  * Median time to close: 1.1 days (lowest-crime quartile) vs 0.8 days (highest): 0.68x -> faster service in high-crime areas.
  Q1 (lowest crime)  50 nbhds  271 reqs  98.2% closed  median 1.1  mean 15.6
  Q4 (highest crime) 49        554       96.0%          0.8         8.5
  + per-type Q1 vs Q4 medians (SW-HGW 7.9 -> 2.0; TRS-48 HOUR PARKING 5.2 -> 10.3)

 Q5 | For: City Planning — Does physical disorder go together with violent crime?
  * Disorder-request volume and violent-crime volume correlate at r = 0.32 across 198 neighborhoods.
  * Using shares instead of counts gives r = -0.06 (less affected by neighborhood size).
  * Most disorder-heavy quartile averages 3.7 violent crimes per neighborhood vs 1.5 (2.41x).
  Priority neighborhoods: DOWNTOWN (7 disorder, 24 violent), BELAIR EDISON (27, 14), FRANKFORD (11, 14), BROOKLYN, CARROLLTON RIDGE
```

Every number in the README's "check the numbers" table (1137/1540/198; r=0.46, rho=0.40; 6.22x; 22:00 6.3% / 12:00 12.9%, r=-0.28; 1.1 vs 0.8 = 0.68x; r=0.32, 3.7 vs 1.5) matches the run exactly. Markdown report written as promised.

## Concurrency Analysis

- `main.go:162-176` — both datasets ingested concurrently (two goroutines + WaitGroup; comment on happens-before).
- `ingest.go:194-245` `runPipeline` — one reader goroutine streams CSV into a **bounded** `batches` channel (`-queue`, default 8) in batches of `-batch` (2048); `workers` goroutines each own a private `Aggregator`; `wg.Wait()` then sequential `Merge`; reader error handed over on `errc` (buffered 1) after `close(batches)`.
- `main.go:196-213` — five `Question.Run` goroutines execute concurrently over the read-only `Joined`, each writing only its own `sections[i]`; per-goroutine `recover()` so one failing question doesn't kill the report.
- No locks anywhere — confinement + channels; `go test -race` and a full `-race` run are clean.
- Join (`join.go`) is a sequential map intersection; it is trivial relative to parse/aggregate.

Level: **full** — parallel parsing/aggregation with worker pool and bounded queue, parallel analysis, correct sync.

## Memory / Resource Analysis

- Streaming CSV with 1 MiB buffered reader (`ingest.go:123`); only the 4 needed columns are copied out of each row (`ingest.go:160-168`), the other ~20 columns never leave the csv reader's buffer.
- Batch = two allocations (row slice + shared string backing array, `ingest.go:140-142`), explicitly to reduce GC churn.
- Bounded channel gives back-pressure; memory independent of file size (comment `ingest.go:111-115`).
- Aggregates hold counts, not rows; the single exception (a `float32` per closed 311 request for exact medians) is called out in README and code (`datasets.go:183`).
- `timeParser` memoises the last successful layout so each parse is one attempt (`util.go:61-92`).
- `runtime.MemStats` heap/total-alloc/GC reported in the run header (`main.go:216-225`).
- Verified: 10.6 MB RSS, 1.8 MiB heap. Design is documented in README "Memory" section and matches code.

## Architecture & Code Quality

Single `package main`, 10 files, each with one responsibility; the four stages (ingest / join / analyse / report) are separate and the data flows only forward. Patterns actually realized, not just named: **Pipeline** (`runPipeline`), **Factory** (`crimeSpec()`, `requestSpec()`, `newCrimeAgg`, `newRequestAgg`), **Strategy** (`Aggregator`, `Question`, `Renderer` interfaces each with 2+ implementations), **Builder** (`ReportBuilder` with fluent `Meta/Section/Note/Build`). `run(args, out)` is testable (no globals), `parseConfig` validates every flag. Statistics are careful (tied-rank Spearman, NaN handling, quartiles with stable ties). 322 lines of tests. Weaknesses: one package; `questions.go` is long (544 lines); keyword-based violent/disorder classification is a heuristic (disclosed and configurable). README candidly discloses Claude assistance and that the student built a single-file version first; the code's consistency and the hand-verified numbers support that she understands it.

## Stakeholder Questions & Value Assessment

| # | Stakeholder | Question | Cross-dataset mechanism |
|---|---|---|---|
| Q1 | Mayor's Office | Which neighborhoods rank high on both crime and 311, how closely do they track? | joint ranks, Pearson+Spearman, both-top-quartile share |
| Q2 | Public Works | Which 311 types are over-represented in highest-crime neighborhoods? | crime quartile x 311 type lift |
| Q3 | Police + DOT (staffing) | Hour-of-day profiles of crime vs 311 | dual histogram, profile correlation |
| Q4 | Mayor's Office (equity) | Do high-crime neighborhoods wait longer for 311 closure? | crime quartile x 311 median days-to-close, per type |
| Q5 | City Planning | Do physical-disorder requests co-occur with violent crime? | disorder quartile x violent crime, counts and shares |

All five are genuinely cross-dataset, each uses the other dataset to stratify or correlate, and each has a plain-language "Key findings" block plus table and chart. Q4 (service equity) and Q5 (broken-windows test with a share-based robustness check) are the kind of question a Mayor's Office would actually ask. Honest caveats (same-week sampling, raw counts, keyword heuristics, correlation != causation). The one real limitation is scale: one week and a 1/11 sample of 311 means the specific findings (e.g., 6.22x lift on n=8) are illustrative; the program is ready for the full files but they are not included.

## Issues

1. **LATE submission** (flagged per rubric; no deduction applied).
2. Bundled data is a one-week excerpt (1,137 crimes, 1,540 of ~17k 311 requests that week); full datasets must be downloaded for citywide conclusions. Small-n findings (e.g., 6.22x on 8 requests) are fragile.
3. Student states she could not test on macOS (Windows only); it did build and run identically here.
4. Single Go package; `questions.go` is long.
5. Violent/disorder classification is keyword-based (configurable, disclosed).
6. README discloses AI (Claude) assistance in writing/debugging the code.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build -o baltimore-intel .` clean first try; vet clean. |
| real_data | 15 | 13 | Two real, unmodified Open Baltimore excerpts bundled and used, no generator, no default fake path; but only a one-week slice (small). |
| concurrency | 15 | 15 | Reader -> bounded channel -> worker pool per dataset, both datasets concurrent, per-worker aggregators merged, five questions in parallel; race-clean. |
| memory | 10 | 10 | Streaming, column projection, two-allocation batches, bounded queue, counts-not-rows, MemStats reporting; documented and verified (10.6 MB RSS). |
| architecture | 15 | 13 | Four realized patterns, clean stage separation, 14 tests, testable `run()`; single package. |
| stakeholder | 15 | 14 | Five distinct, genuinely cross-dataset questions for named stakeholders with findings, tables and charts; limited only by one-week data scale. |
| runs_results | 10 | 10 | Runs in 0.24 s, all five answers produced, README verification numbers reproduce exactly, Markdown report written. |
| readme | 5 | 5 | Detailed, honest, exact macOS commands, verification table that matches reality. |
| **Total** | **100** | **95** | **A** |

## Final Grade Rationale

This is the strongest of the four: real unmodified Open Baltimore data is bundled, the pipeline is a textbook bounded fan-out/fan-in with explicit memory design that is both documented and measurable, four design patterns are genuinely implemented with interfaces and multiple implementations, 14 tests and the race detector pass, and all five questions use one dataset to stratify the other in ways a municipal leader could act on. Two points come off real_data and one off stakeholder because the bundled data is a single week (the student is upfront about this), and two off architecture for the single-package layout. The submission was late, which is flagged but not deducted per the rubric. Total 95, A.
