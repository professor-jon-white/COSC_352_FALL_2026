# Grace Joseph — COSC 352 Midterm Capstone Grading Report

**Summary: 63/100 — D.** Single-file Go program over two **real, bundled** Open Baltimore datasets (311 Customer Service Requests 2026, 943,283 rows / 386 MB; Vacant Building Notices, 11,445 rows). It builds from source, streams both files in ~1 s with 12 MB RSS, and is race-clean. But concurrency is only two goroutines (one per file), there is no join beyond an exact-name neighborhood comparison table, four of five "questions" are single-dataset counts, no stakeholder is named, and an output-loop bug prints the 311 sections twice (first time empty).

## Inventory
Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/grace_joseph/BaltimoreDataEngine/`

| File | LOC / size | Notes |
|---|---|---|
| main.go | 366 | all logic |
| go.mod | `module baltimore-data-engine`, **go 1.27.2** | |
| README.md | 134 lines | |
| baltimore-data-engine.exe | 2.6 MB | PE32+ Windows binary (unusable on macOS; ignored, built from source) |
| data/311_Customer_Service_Requests_2026.csv | 385.7 MB, 943,284 lines | real |
| data/Vacant_Building_Notices.csv | 1.5 MB, 11,446 lines | real |

Language: Go. LOC 366 (1 file). No tests, no run script.

## Datasets & Data Realness
```
311 header: X,Y,RowID,SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,LastActivity,LastActivityDate,Outcome,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord
row: -76.68,39.34,2,500VO00000ta054YAA,26-00004287,SW-Bulk Scheduled-Saturday,Phone,2026/01/03 14:44:56+00,Closed,...,"3954 DOLFIELD AVE, Baltimore City, 21215",21215,Dolfield,6,Northwestern,624,...
Vacant header: X,Y,OBJECTID,NoticeNum,DateNotice,DateCancel,DateAbate,NT,OWNER_ABBR,HousingMarketTypology2023,Council_District,Neighborhood,BLOCKLOT,Address
row: 1411570.99,588061.84,7,2655955A,2026/05/28 13:13:00+00,,,Vacant,,J,9,Carrollton Ridge,0702 050,2113 WILKENS AVE
```
Both match Open Baltimore schemas exactly (SRRecordID/ServiceRequestNum/SRType/Agency; NoticeNum/DateNotice/BLOCKLOT). Real addresses and neighborhoods (Dolfield, Carrollton Ridge, 21 Canton vacant rows). Row counts: 943,283 and 11,445 (README's "Current Results" match). No mock generator. README does not link the portal pages.
**data_status: real_bundled.**

## Build Results
README (macOS): `go build -o baltimore-data-engine` then `./baltimore-data-engine`.
```
$ go build -o <scratch>/baltimore-data-engine .   -> exit 0 (0.4 s, first try)
$ go build -race -o <scratch>/bde-race .          -> exit 0
```
Note: go.mod declares `go 1.27.2`; with local go1.24.0 and `GOTOOLCHAIN=auto` the 1.27.2 toolchain was auto-selected (already cached here; on a fresh machine Go would download it — works transparently with network, but README does not mention the version requirement).

## Test / Vet / Race Results
```
$ go vet ./...   -> exit 0
$ go test ./...  -> ?  baltimore-data-engine  [no test files]
```
Race-instrumented full run over 386 MB: **0 DATA RACE** reports (10.9 s, 43 MB RSS).

## Run Results
`./baltimore-data-engine` from project dir: wall **1.11 s**, max RSS **12.3 MB**, exit 0. Output (scratchpad `grace/run.txt`), abbreviated:
```
 Baltimore City Data Intelligence Engine
Reading both datasets concurrently...
Finished reading: Vacant Building Notices / Total records: 11445 / Missing neighborhood: 0
311 REQUESTS BY NEIGHBORHOOD (BAR CHART)
No data available.                      <-- BUG: 311 sections printed before 311 finished
TOP 10 311 REQUEST TYPES / No records found.
TOP 10 311 SERVICE REQUEST NEIGHBORHOODS / No records found.
TOP 10 VACANT BUILDING NEIGHBORHOODS
  Carrollton Ridge 744 / Broadway East 717 / Sandtown-Winchester 587 / Harlem Park 438 / ...
Finished reading: 311 Service Requests / Total records: 943283 / Missing neighborhood: 250620
311 REQUESTS BY NEIGHBORHOOD (BAR CHART)
Belair-Edison                    | ######################################## 24775
Coldstream Homestead Montebello  | ##########################               16414
Downtown                         | #####################                    13502
Canton                           | #####################                    13051
... (10)
TOP 10 311 REQUEST TYPES
  ECC-Information Request 231754 / SW-Rat Rubout Proactive 139454 / TRS-Parking Complaint 52980 / HCD-Sanitation Property 35494 / ...
TOP 10 311 SERVICE REQUEST NEIGHBORHOODS   (same as chart)
TOP 10 VACANT BUILDING NEIGHBORHOODS       (printed again)
NEIGHBORHOOD COMPARISON  (sorted by vacant notices)
Carrollton Ridge      6540 requests   744 notices
Broadway East         9466            717
Sandtown-Winchester   8988            587
Harlem Park           6102            438
... (10)   Neighborhoods found in both datasets: 213
DATA ANALYSIS AND KEY FINDINGS
Highest 311 request count: Belair-Edison (24775)
Highest vacant building notice count: Carrollton Ridge (744)
311 records missing neighborhood information: 26.6%
Initial data ingestion complete.
```
Mapping to the README's five questions: Q1 top SRTypes (answered), Q2 top vacant neighborhoods (answered), Q3 top 311 neighborhoods (answered), Q4 comparison table (answered, 213 shared), Q5 missing-neighborhood share 26.6% (answered). Ordering of the two "Finished reading" blocks is nondeterministic (whichever goroutine finishes first).

## Concurrency Analysis
- `main.go:242-268`: two goroutines, one `analyzeCSV` per file, sending `DatasetSummary` on a buffered `results` channel (cap 2); `sync.WaitGroup` + closer goroutine. Correct, race-clean.
- Each file is parsed and aggregated sequentially inside its goroutine; no worker pool; comparison/aggregation across datasets (`printNeighborhoodComparison`, 134-185) is sequential on the main goroutine.
- Level: **token** (two goroutines loading two files).

## Memory / Resource Analysis
Streams with `csv.Reader`, retains only count maps; 12 MB RSS over 386 MB of CSV is genuinely good. `defer file.Close()`; `FieldsPerRecord=-1`. README has a short paragraph ("Processing the records as they are read helps avoid keeping every raw CSV record in memory"). No bounded channels (none needed), no MemStats, no worker bounds. Some evidence, light documentation.

## Architecture & Code Quality
One file, but readable: `analyzeCSV`, `printTopCounts`, `printNeighborhoodComparison`, `printDataAnalysis`, `printNeighborhoodBarChart`. Dataset paths are hard-coded (no flags). `printTopCounts` and `printNeighborhoodBarChart` duplicate the same sort logic. **Bug:** all four print blocks sit inside `for result := range results` (lines 272-311), so they execute once per finished dataset — the first iteration prints empty 311 sections and the vacant list is printed twice. Result routing by string-compare on `result.Name` is brittle. No named design pattern; README's "Software Architecture" section is generic. The shipped Windows .exe is irrelevant on macOS. README is candid and student-written (first-person "Future Improvements").

## Stakeholder Questions & Value Assessment
Stakeholders: "Baltimore City residents and local government agencies" (generic, none specific). Q1–Q3 are single-dataset top-10 counts; Q5 is a data-quality statistic about one dataset; only Q4 (side-by-side 311 vs. vacant counts for shared neighborhoods) is cross-dataset, and it is raw counts without any rate, ratio, or ranking insight. The comparison table does show something a Housing/DPW stakeholder might glance at (Carrollton Ridge: 744 notices / 6,540 requests), but there is no analysis of it. Visualization: one ASCII bar chart (311 by neighborhood) and plain tables.

## Issues
1. Output loop bug prints 311 sections empty, then everything again (main.go:272-311).
2. Token concurrency: two goroutines; no pool; no concurrent join/aggregate.
3. Only 1 of 5 questions is cross-dataset; no specific stakeholder; shallow counts.
4. go.mod `go 1.27.2` forces toolchain auto-download, undocumented; Windows .exe bundled.
5. Hard-coded data paths; no tests; duplicated sort code; README lacks dataset links.

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | Builds cleanly from source first try with the documented macOS command (toolchain auto-selected). |
| real_data | 15/15 | Two real Open Baltimore datasets bundled (schemas verified), 943k + 11k rows, used directly. |
| concurrency | 5/15 | Exactly two goroutines loading two files via WaitGroup/channel; nothing else concurrent. |
| memory | 5/10 | Streaming aggregation evident (12 MB RSS on 386 MB) and briefly documented; no further design. |
| architecture | 5/15 | Single file, organized into functions, but output-loop bug, duplication, hard-coded paths, no tests/pattern. |
| stakeholder | 6/15 | 5 questions but 4 single-dataset/data-quality; generic stakeholders; one chart. |
| runs_results | 8/10 | Runs end-to-end fast on real data with sensible numbers; duplicated/empty sections mar the output. |
| readme | 4/5 | Explicit macOS build/run/verify steps and expected counts; no dataset links or Go-version note. |
| **Total** | **63/100** | **D** |

## Final Grade Rationale
Grace earns full credit for building cleanly and for bundling and correctly streaming two large real Open Baltimore datasets — the run is fast and memory-frugal. The grade is pulled down by the thin treatment of the other requirements: concurrency stops at one goroutine per file, the architecture is a single file with a visible print-loop bug, and the research questions are mostly single-dataset counts with no named stakeholder or analytic join. 63/100, a D.
