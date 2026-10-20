# Daniel Dada — COSC 352 Midterm Capstone Grading Report

**Summary: 88/100 (B).** **LATE submission** (flagged, not deducted). Go, multi-package (`cmd`, `internal/{ingest,join,analysis,normalize,model,report}`, `tools/mksample`) with a parallel batch-ingest pipeline and a parallel chunked join that is verified equal to a sequential reference. Three real Open Baltimore CSVs are bundled (~950 MB: Part 1 Crime 644k rows, 311 2024 1.08M rows, 311 2025 931k rows) plus a real-data sample subset. Builds and vets clean, tests and `-race` tests pass, and the **full-data run completes in 2.0 s** (830 MB RSS). Weaker on memory discipline (all records retained with un-interned strings; 2.8 GB total allocation) and on cross-dataset depth (Q4/Q5 are mostly side-by-side single-dataset views).

## Inventory

Submission root: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/daniel_dada/baltimore-intel/` (original filename contained `_LATE_`).

| File | LOC | Purpose |
|---|---:|---|
| `cmd/main.go` | 118 | flags (`-data -year -mode -verify`), sample fallback, 3-stage orchestration, MemStats summary |
| `internal/ingest/ingest.go` | 150 | `streamCSV` (row-at-a-time, bufio 1 MB), sequential `ReadCrime`/`ReadRequests`, `Stats` |
| `internal/ingest/concurrent.go` | 141 | generic `parallel[T]` reader -> batch channel -> NumCPU workers -> merge |
| `internal/ingest/ingest_test.go` | 62 | sequential vs parallel ingest stats, missing-column |
| `internal/join/join.go` | 134 | `Classify` 311 types, `IsViolent`, per-neighborhood `Stats`, sequential `Build` |
| `internal/join/concurrent.go` | 73 | `BuildParallel`: chunked goroutines with private `Dataset`, merged once |
| `internal/join/join_test.go` | 106 | known-answer test, parallel==sequential on 5k rows, empty |
| `internal/analysis/analysis.go` | 88 | `Analyzer` Strategy interface, `All()`, ranking/Pearson helpers |
| `internal/analysis/q1_q2.go` | 96 | Q1 TopBoth, Q2 SanitationVsViolence |
| `internal/analysis/q3_q5.go` | 213 | Q3 Underreporting (z-score), Q4 HourOfDay, Q5 MonthlyTrend |
| `internal/analysis/analysis_test.go` | 18 | Pearson |
| `internal/normalize/normalize.go` + `_test.go` | 44 | neighborhood key normalization |
| `internal/model/model.go` | 16 | `CrimeRecord`, `ServiceRequest` |
| `internal/report/report.go` | 35 | `Section`, ASCII `Bar` |
| `tools/mksample/main.go` | 108 | builds `data/sample/` by taking every Nth real 2024 row |
| **Total** | **1402** | 16 Go files (11 source + 5 test) |

Other: `README.md`, `run.sh`, `go.mod` (`go 1.21`, no deps), `report.txt` (saved full-data output), `.github/workflows/test.yml` (macOS CI: build, test, test -race, run, -verify, run.sh), `.gitignore`, `.gitattributes`.

Data (bundled): `data/crime.csv` 166 MB (644,738 lines), `data/311_2024.csv` 425 MB (1,081,869 lines), `data/311_2025.csv` 379 MB (931,325 lines), `data/sample/crime.csv` 1.3 MB (4,906 lines), `data/sample/311_2024.csv` 2.1 MB (5,411 lines). Total 955 MB.

## Datasets & Data Realness

- **Part 1 Crime Data (Legacy SRS)** — header `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y`; rows e.g. `14I10336, 8/29/2014 12:00:00 PM, 5A, BURGLARY, ... 2700 W FAIRMOUNT AVE, SOUTHWEST, SHIPLEY HILL, 39.288, -76.660`. 644,737 data rows spanning 2014+. **Real.**
- **311 Customer Service Requests 2024** — header `RowID,SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,...,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,x,y`; rows `24-00000007 TRS-48 Hour Parking/Abandoned Vehicle ... 5242 FAIRLAWN AVE ... Woodmere`, `24-00000008 SW-Graffiti Removal ... 1909 N FULTON AVE ... Penn North`. 1,081,868 data rows. **Real.**
- **311 Customer Service Requests 2025** — same export as other students' copies (931,324 rows). **Real**, but **not used by the default run** (`-year 2024`); using it requires `-year 2025` and it would be paired with the same crime file.
- `data/sample/` — every 9th 2024 crime row and every 200th 2024 311 row of the real files (`tools/mksample`), i.e. real records, not synthetic. The sample is only used if `./data/crime.csv` is absent (`cmd/main.go:29-32`); with the bundled data present, the **real full data is the default path**.
- Data status: **real_bundled**.

## Build Results

```
$ go build -o baltimore-intel ./cmd     # exit 0, 0.5 s
$ go vet ./...                          # exit 0, clean
$ go build -race -o baltimore-intel-race ./cmd   # exit 0
```
Clean build first try with the documented command (binaries written to the scratchpad to avoid touching the submission).

## Test/Vet/Race Results

```
$ go test ./...
ok  baltimore-intel/internal/analysis   0.236s
ok  baltimore-intel/internal/ingest     0.478s
ok  baltimore-intel/internal/join       0.658s
ok  baltimore-intel/internal/normalize  0.824s
$ go test -race ./...     -> all ok, 0 races
$ ./baltimore-intel-race -data data/sample   -> exit 0, 1.3 s, 0 DATA RACE reports
```
(The race binary was run on the bundled real-data sample rather than the 950 MB full set, since the race runtime multiplies memory of an 830 MB-RSS run.)
```
$ ./baltimore-intel -data <full data> -verify   -> "VERIFY: parallel join is identical to sequential join"
```

## Run Results

Run from a scratchpad working directory with `-data` pointing at the submission's data (the program writes `report.txt` to cwd). Full bundled data, within the 10-minute budget by a wide margin:
```
$ /usr/bin/time -l ./baltimore-intel -data .../daniel_dada/baltimore-intel/data
        2.03 real         3.12 user         0.61 sys
           829587456  maximum resident set size      (830 MB)
```
`bash run.sh` (scratchpad copy with `data/` symlinked): build + `go test -race ./...` + run + `-verify` all succeeded in 5.7 s total.

Captured output (scratchpad `daniel/run_full.txt`), first ~60 lines plus Q4/Q5 answers:
```
Mode:  parallel
Crime: read=644737 kept=43803 dropped(bad date=0, other year=600598, blank neighborhood=336, malformed=0)
311:   read=1081868 kept=747207 dropped(bad date=0, other year=0, blank neighborhood=334661, malformed=0)
Joined 289 neighborhoods (205234 non-complaint 311 rows excluded)

=== Q1: Neighborhoods high in BOTH crime and 311 demand ===
For: Mayor's Office

Ranked by crime rank + 311 rank. Bars show crime count.

1. DOWNTOWN                    ██████████████████████████████ 2087   | 311 requests: 15489
2. BELAIR EDISON               █████████████ 938   | 311 requests: 14335
3. BROOKLYN                    ████████████ 882   | 311 requests: 8772
4. CANTON                      ██████████ 720   | 311 requests: 9412
5. FRANKFORD                   ███████████████ 1069   | 311 requests: 7630
6. SANDTOWN WINCHESTER         ████████ 583   | 311 requests: 8438
7. PATTERSON PARK NEIGHBORHOO  ████████ 589   | 311 requests: 5848
8. HAMPDEN                     ████████ 616   | 311 requests: 5786
9. CENTRAL PARK HEIGHTS        ██████ 448   | 311 requests: 7829
10. WASHINGTON VILLAGE PIGTOW  ███████ 510   | 311 requests: 6495

Note: raw counts favor large neighborhoods; population data would be needed to normalize.

=== Q2: Do sanitation/blight requests track violent crime? ===
For: Department of Public Works

Pearson r = 0.684 across 289 neighborhoods (moderate correlation)
Correlation is not causation: both may simply reflect neighborhood size or density.

Top 10 by sanitation requests (bar), with violent crimes:
CARROLLTON RIDGE               ██████████████████████████████ 4332   | violent crimes: 127
BELAIR EDISON                  █████████████████████████ 3717   | violent crimes: 228
BROOKLYN                       ███████████████████████ 3380   | violent crimes: 174
SANDTOWN WINCHESTER            ████████████████████ 2989   | violent crimes: 188
BROADWAY EAST                  ████████████████████ 2950   | violent crimes: 88
PENN NORTH                     █████████████████ 2552   | violent crimes: 115
EAST BALTIMORE MIDWAY          █████████████████ 2491   | violent crimes: 113
CENTRAL PARK HEIGHTS           ████████████████ 2379   | violent crimes: 130
MCELDERRY PARK                 ████████████████ 2378   | violent crimes: 114
DOWNTOWN                       ███████████████ 2231   | violent crimes: 519

=== Q3: Neighborhoods with high crime but low 311 use (possible under-reporting) ===
For: Baltimore Police Department

Metric: crimes per 311 request (121 neighborhoods with 109+ crimes and 311 matches; mean 0.11, sd 0.10).
A high z-score means unusually many crimes relative to 311 activity.

INNER HARBOR                   ██████████████████████████████ 30   z=7.19  crimes=772  311=932
DUNBAR BROADWAY                ███████████████ 15   z=3.58  crimes=246  311=527
CANTON INDUSTRIAL AREA         ██████████████ 14   z=3.25  crimes=411  311=947
REISTERSTOWN STATION           ███████████ 11   z=2.55  crimes=314  311=862
PULASKI INDUSTRIAL AREA        ██████████ 10   z=2.32  crimes=436  311=1276
DOWNTOWN WEST                  █████████ 9   z=2.09  crimes=174  311=546
UNIVERSITY OF MARYLAND         █████████ 9   z=2.09  crimes=172  311=540
GAY STREET                     ███████ 7   z=1.68  crimes=313  311=1128
OLDTOWN                        █████ 5   z=1.11  crimes=301  311=1367
ARMISTEAD GARDENS              ████ 4   z=1.02  crimes=191  311=904

Excluded (enough crimes but zero 311 rows; likely a neighborhood-name mismatch between datasets):
  GLENHAM BELHAR, NEW SOUTHWEST MOUNT CLARE

Caveat: about 31% of 311 rows lack a neighborhood, so 311 counts are undercounted.

...
=== Q4: Crime vs. lighting and transportation requests by hour ===
For: Department of Transportation

Peak hours: crime 17:00, lighting requests 21:00, transportation requests 08:00
Each column is scaled to its own maximum.

Hour   Crime                  Lighting               Transport             
... (24-row hourly chart) ...
=== Q5: Monthly trends and neighborhoods that diverge from the city ===
For: Department of Planning

Month Crime                  311 requests          
Jan   ████████████████       ███████████████         (3594 / 45853)
Feb   ███████████████        █████████████           (3482 / 40611)
Mar   ████████████████       █████████████           (3642 / 41446)
Apr   ███████████████        ███████████████         (3512 / 47541)
May   ████████████████       ████████████████        (3696 / 51576)
Jun   █████████████████      ███████████████         (3811 / 48664)
Jul   █████████████████      ████████████████        (3981 / 50519)
Aug   █████████████████      ██████████████████      (3869 / 54943)
Sep   ██████████████████     ██████████████          (4026 / 44248)
Oct   ████████████████       ██████████████          (3798 / 45506)
Nov   ███████████████        ███████████             (3472 / 35893)
Dec   █████████████          ███████████             (2920 / 35173)

Neighborhoods whose monthly crime pattern differs most from the citywide pattern
(lowest correlation with the city curve; neighborhoods with 300+ crimes):
  BROOKLYN                       r=-0.27  (882 crimes)
  BALTIMORE HIGHLANDS            r=-0.10  (339 crimes)
  CHARLES VILLAGE                r=-0.09  (322 crimes)
  BELAIR EDISON                  r=-0.08  (938 crimes)
  CENTRAL PARK HEIGHTS           r=-0.00  (448 crimes)

--- Performance and memory ---
Mode:                parallel
CPU threads used:    10
Load time:           1.731226458s
Join time:           38.447375ms
Total elapsed:       1.771170917s
Heap in use:         537.3 MB
Total allocated:     2803.8 MB
Memory from OS:      884.9 MB
GC cycles:           33
```

**The five answers (2024 data):**
1. Q1 — Highest in both crime and 311 demand (rank-sum): Downtown (2,087 crimes / 15,489 requests), Belair-Edison (938 / 14,335), Brooklyn, Canton, Frankford, Sandtown-Winchester, Patterson Park, Hampden, Central Park Heights, Washington Village/Pigtown.
2. Q2 — Sanitation/blight requests vs violent crime: Pearson r = 0.684 across 289 neighborhoods (moderate); Carrollton Ridge leads sanitation requests (4,332) with 127 violent crimes; Downtown 2,231 / 519.
3. Q3 — High crime, low 311 use (z-score of crimes-per-request among 121 neighborhoods with 109+ crimes): Inner Harbor z=7.19 (772 crimes, 932 requests), Dunbar-Broadway 3.58, Canton Industrial Area 3.25, Reisterstown Station 2.55, Pulaski Industrial Area 2.32.
4. Q4 — Peak hours: crime 17:00, lighting requests 21:00, transportation requests 08:00 (24-row three-column ASCII chart).
5. Q5 — Monthly: crime peaks Sep (4,026) and troughs Dec (2,920); 311 peaks Aug (54,943). Neighborhoods whose monthly crime curve diverges most from the city: Brooklyn r=-0.27, Baltimore Highlands -0.10, Charles Village -0.09, Belair-Edison -0.08, Central Park Heights 0.00.

Output is sensible and matches the student's saved `report.txt`. Ingest stats were printed honestly: crime read=644,737 kept=43,803 (600,598 other-year rows dropped), 311 read=1,081,868 kept=747,207 (334,661 blank neighborhood), 205,234 non-complaint 311 rows excluded.

## Concurrency Analysis

- `cmd/main.go:48-60` — the two files are ingested in parallel (`sync.WaitGroup`, two goroutines); `-mode sequential` switches to the reference path.
- `internal/ingest/concurrent.go:52-105` — generic `parallel[T]`: the calling goroutine streams rows (`streamCSV`) into 2,000-row batches on `batches := make(chan []row, workers*2)`; `runtime.NumCPU()` worker goroutines parse dates, filter by year, normalize keys into a worker-local `partial[T]`; a closer goroutine `wg.Wait(); close(results)`; the main goroutine merges. Bounded, no shared mutable state.
- `internal/join/concurrent.go:27-72` — `BuildParallel` splits both slices into NumCPU chunks, each goroutine fills a private `Dataset` (`newDataset()`), results flow over a channel sized `2*workers` so no sender blocks, and `merge` folds them once. Equivalence to `Build` is asserted by tests (`join_test.go:63-87`) and at runtime (`-verify`, `reflect.DeepEqual`).
- Analyses (`analysis.All()`) run sequentially (`cmd/main.go:95-98`); they are cheap (289 neighborhoods).
- Race detector clean in tests and program run. **Level: full** — parallel parse and parallel join/aggregate with worker pools and correct sync; verification of determinism is a nice touch.

## Memory/Resource Analysis

Some genuine thought: `streamCSV` reads one record at a time through a 1 MB `bufio.Reader` (comment: "never ReadAll, so memory stays flat"), bounded batch channels, worker-local partials, and a `runtime.MemStats` summary printed at the end (heap, total alloc, sys, GC cycles). But every kept record is retained in memory with **un-interned strings** (`Description`, `Type`, `Agency`) that are substrings of the full CSV line (`encoding/csv` returns fields sharing one backing string per line — so each kept record pins its whole ~400-byte row). Result on full data: heap 537 MB, total allocated 2.8 GB, **830 MB max RSS** for 791k kept records, versus 13 MB (Basnet A.) and 217 MB (Eze) on comparable input. Date parsing with `time.Parse` on 1.7M rows is also the dominant cost. The README has no memory section; `runtime.MemStats` is reported but not acted upon. Partial credit.

## Architecture & Code Quality

- Clear package separation matching the assignment's three concerns: `ingest` (parse), `join` (cross-dataset matching + classification), `analysis` (questions), `report` (output), with `normalize` and `model` as leaves.
- `Analyzer` is a real Strategy interface with five implementations registered in `All()`; sequential `Build`/`ReadCrime` serve as reference implementations for the parallel versions — a good testing discipline. No Builder/Factory/Pipeline named, though the main is an explicit 3-stage pipeline.
- Tests in four packages (known-answer join, parallel==sequential at scale, ingest stats, normalization edge cases, Pearson). A macOS GitHub Actions workflow exercises build/test/race/run/verify/run.sh.
- Report layer is minimal (`Section` + one `Bar` function); output is plain text only, with a sentinel 29-character label truncation that chops names (`PATTERSON PARK NEIGHBORHOO`, `WASHINGTON VILLAGE PIGTOW`).
- Code is compact and readable; comments explain thread-safety reasoning. README states it was "Developed with Go 1.27.2 on Windows" with macOS verified via CI; it also ends mid-sentence ("a run takes a few"). Nothing reads as unbacked boilerplate.

## Stakeholder Questions & Value Assessment

| # | Stakeholder | Question | Cross-dataset? |
|---|---|---|---|
| 1 | Mayor's Office | Neighborhoods high in both crime and 311 demand (rank-sum) | yes |
| 2 | DPW | Do sanitation/blight requests track violent crime? (r, top-10) | yes |
| 3 | BPD | High crime but low 311 use — z-score of crimes per request | yes |
| 4 | DOT | Crime vs lighting vs transportation requests by hour | side-by-side histograms, no join |
| 5 | Planning | Monthly crime & 311 trend; neighborhoods diverging from city crime curve | monthly both; divergence is crime-only |

Q3 is the most insightful: it filters tiny neighborhoods, separates join-key gaps (Glenham-Belhar, New Southwest/Mount Clare) from genuine under-reporting, and flags Inner Harbor / Dunbar-Broadway / Canton Industrial Area — a BPD or 311 outreach team could use that. Q2's r=0.68 blight–violence association for DPW is useful context. Q1 is a rank-sum of raw counts (acknowledged as size-biased). Q4 answers "when do lighting complaints come in vs crime" without actually relating them per neighborhood or per hour statistically, and Q5's divergence ranking uses only the crime dataset. Visualization is ASCII bars only, no HTML. Honest caveats (default-time inflation at 00:00/12:00, 31% missing neighborhoods) add credibility.

## Issues

1. **LATE submission** (filename contained `_LATE_`); flagged per rubric, no deduction.
2. Memory: all kept records retained with un-interned strings pinning full CSV lines; 830 MB RSS / 2.8 GB total allocation; no memory design section in README.
3. Q4 and Q5 are largely single-dataset views placed side by side rather than cross-dataset joins.
4. `data/311_2025.csv` (379 MB) is bundled but unused by the default `-year 2024` run; README download instructions mention only 2024.
5. Analyses run sequentially; report labels are truncated at 29 chars; text-only output (no HTML/web UI); README ends mid-sentence.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---:|---:|---|
| build | 15 | 15 | `go build`, `go vet`, `-race` build all clean with the documented command. |
| real_data | 15 | 15 | Three real Open Baltimore CSVs bundled (644k + 1.08M + 931k rows), schemas verified, full data is the default path; sample is a real-row subset. |
| concurrency | 15 | 14 | Parallel batch-ingest pool and parallel chunked join with private partials, WaitGroup/channels, race-clean, verified equal to sequential; analyses sequential. |
| memory | 10 | 6 | Streaming reader, bounded channels, MemStats report; but all records retained with un-interned strings (830 MB RSS, 2.8 GB allocated) and no documented memory design. |
| architecture | 15 | 14 | Clean multi-package layout, Strategy interface realized, reference implementations, tests in 4 packages, CI; report layer thin and no other patterns. |
| stakeholder | 15 | 10 | Five questions for named stakeholders with ASCII charts; Q1–Q3 genuinely cross-dataset (Q3 strong), Q4/Q5 mostly single-dataset side-by-side. |
| runs_results | 10 | 10 | Runs on the full 950 MB data in 2.0 s, verify passes, five sensible answers, report.txt written. |
| readme | 5 | 4 | Explicit macOS build/run/verify, run.sh, flag table, CI; truncated final sentence and no memory section. |
| **Total** | **100** | **88** | **B** |

## Final Grade Rationale

A solid, well-tested Go pipeline on genuinely large real data: it builds and runs cleanly, the parallel ingest and join are correct (proven equal to a sequential reference under the race detector), and the full-data run takes two seconds. Points are lost for memory management — the design retains every record with un-interned CSV substrings and shows no documented memory thinking beyond streaming reads — and for stakeholder depth, where two of the five questions compare datasets side by side rather than joining them, with text-only visualization. Late submission flagged without deduction. 88/100, B.
