# Jayden Reyes — COSC 352 Midterm Capstone Grading Report

**Summary: 56/100 — F.** Graded on the `direct_zip/baltimore-engine/` version (sent directly to the instructor). Four-file Go program that loads three CSVs with three goroutines, counts records per neighborhood, and prints five mostly trivial questions as plain text. Data realness was verified against Open Baltimore's live services: `wb_records.csv` is a reshaped extract of **BPD Arrests** (224,623 rows) and `gvt results.csv` a reshaped extract of the **Gun Offenders Registry** (3,476 rows) — real, but columns were renamed/synthesized and not in raw format; `cvi_records.csv` is 20 rows hand-transcribed from a MONSE PDF (not Open Baltimore). Concurrency is token, everything is loaded into memory, no stakeholder is named, no visualization.

## Canvas vs. direct_zip Comparison
`diff -r baltimore-engine direct_zip/baltimore-engine`: all `.go` files, Makefile, go.mod and all three data files are **byte-identical**. Only differences: Canvas copy has `.DS_Store` files; README lines 19–21 differ — Canvas: "2. Then type: go run ." ; direct_zip: "2. Then type: make build … 3. To execute, type: go run .". The direct_zip version is graded as final; it changes nothing about code or data.

## Inventory
Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/jayden_reyes/direct_zip/baltimore-engine/`

| File | LOC / size | Notes |
|---|---|---|
| main.go | 16 | entry |
| concurrency.go | 75 | 3 goroutines loading 3 files |
| revealer.go | 140 | CSV loaders into struct slices |
| analytics.go | 192 | counting + 5 questions |
| scripts/gen_mock.go | 26 | broken mock generator |
| Makefile | 21 lines | build/run/mock/clean |
| go.mod | module baltimore-engine, **go 1.27.2** | |
| README.md | 42 lines, narrative | |
| data/wb_records.csv | 13.6 MB, 224,624 lines | arrests extract |
| data/gvt results.csv | 150 KB, 3,477 lines | gun offender extract |
| data/cvi_records.csv | 4.8 KB, 21 lines | MONSE PDF transcription |

Language: Go. Source LOC 449 (5 files incl. script). No tests, no run script (Makefile).

## Datasets & Data Realness
```
wb_records.csv:  CrimeNum,Neighborhood,CrimeType,District,CrimeDate
                 26000003,Downtown,Unknown Offense,Central,2026-01-01
                 21095907,Charles North,4ECOMMON ASSAULT,Northern,2021-07-12
gvt results.csv: CaseID,RecordType,Neighborhood,Count,RecordYear
                 26-0505,Gun Offender,Frankford,1,2026
cvi_records.csv: Neighborhood,Program,ReportYear,Metric,Value,Unit,Source,SourceURL
                 Cherry Hill,Safe Streets Baltimore,2024,Mediations,429,count,"MONSE, Safe Streets ... Calendar Year 2024",https://monse.baltimorecity.gov/...pdf
```
Verification against live Open Baltimore ArcGIS services (network used by the grader, not by the program):
- **BPD Arrests** (`GeoSpatialized_Tables/Arrest/FeatureServer/0`, 399,895 records; fields RowID, IncidentNumber, ArrestNumber, Age, Gender, Race, ArrestDateTime, ArrestLocation, IncidentOffence, …, District, Post, Neighborhood). Live record `ArrestNumber=26000003, IncidentOffence=Unknown Offense, District=Central, Neighborhood=Downtown, 2026-01-01` is exactly row 1 of wb_records.csv. Values like `87NARCOTICS`, `4ECOMMON ASSAULT` are the real `IncidentOffence` codes with hyphens stripped; year distribution 2010 (40,953) declining to 2022 (3,034) and district split (Western 31k, Central 30k…) match the real dataset. 289 real neighborhoods. **Real, derived, reshaped** (5 of 18 columns, renamed; subset of 224,623 rows, likely those with a Neighborhood).
- **Gun Offenders Registry** (`OpenData/Gun_Offenders_Registry/FeatureServer/0`, 4,597 records; fields RowID, CaseNumber, CreatedDate, ModifiedDate, LastName, …, District, Post, Neighborhood). Live `CaseNumber` values `26-0520, 26-0519…` match the file's `CaseID` format (`26-0505`). `RecordType="Gun Offender"` and `Count=1` are **synthetic constant columns**; `RecordYear` presumably derived from a date. 232 real neighborhoods (Belair-Edison 157, Frankford 109…). **Real, derived, reshaped.**
- **cvi_records.csv**: 20 rows for 10 Safe Streets sites from the MONSE CY2024 annual report PDF. Real public source but **not an Open Baltimore dataset**, hand-transcribed.
- `scripts/gen_mock.go` (`make mock`) is **not the default path**. Run in a scratch copy it is broken: `os.MkdirAll(dataDir, 1304)` uses decimal 1304 (= mode 2430, owner read-only), so `os.Create` fails silently and no files are written; its intended output (3-column header, 5-column rows; GVT columns OfficerOrDetectiveName/SettlementCost) does not match either the real files or the loaders. If it ever worked it would overwrite the real data.
- Code/data mismatch: `GVTRecord` fields are `OfficerName, SettlementAmount, SettlementYear` (revealer.go:17-23) while the CSV columns are RecordType/Count/RecordYear — the loader reads by position so it still works, but the struct documents a different (earlier, fictional) schema.

Row counts: wb 224,623; gvt 3,476; cvi 20. **data_status: real_bundled** (two Open Baltimore datasets, derived/reshaped rather than raw; third dataset non-portal).

## Build Results
README: `cd baltimore-engine`, `make build`, `go run .`. `make -n build` -> `go build -o baltimore-engine .`
```
$ go build -o <scratch>/baltimore-engine .   -> exit 0 (0.5 s, first try; go 1.27.2 toolchain auto-selected per go.mod)
$ go build -race -o <scratch>/be-race .      -> exit 0
```
README claims step 2 (`make build`) "will run the program" — it only builds; `make run` would run it.

## Test / Vet / Race Results
```
$ go vet ./...               -> exit 0
$ go vet scripts/gen_mock.go -> exit 0
$ go test ./...              -> no test files (baltimore-engine, baltimore-engine/scripts)
```
Race-instrumented run: **0 DATA RACE** reports (2.06 s, 183 MB RSS).

## Run Results
`./baltimore-engine` from project dir: wall **0.30 s**, max RSS **50 MB**, exit 0. Output (scratchpad `jayden/run.txt`), abbreviated:
```
Processing data for program
Data is being process with concurrency, please wait
Time for execution:  57.4ms
The data has been loaded successfully! Records: 224623 / 3476 / 20
Combining Data together
These are the neighborhoods and locations obtained from the datasets: 288
Neighborhood: abell | Arrest Records: 350 | Gun Offender Records: 1 | CVI Mediations: 0 | CVI Events: 0
Neighborhood: allendale | Arrest Records: 983 | Gun Offender Records: 30 | ...   (5 alphabetical rows)

1. How many unique neighborhoods appear across the datasets?        Answer: 288
2. Which neighborhoods have both Arrest records and Gun Offender records?
   abell, allendale, arcadia, ... yale heights        (≈230 names, one per line)
3. Which neighborhood has the highest CVI mediations?               cherry hill  429 mediations
4. Which neighborhood has the fewest CVI community events?          belair edison  3 events
5. Which neighborhoods have CVI activity but no Gun Offender records?   park heights
Done ... Data Process Complete
```
Note on Q5: "park heights" appears only because the CVI file uses the Safe Streets site name "Park Heights" while the arrests/registry files use BPD neighborhoods ("Central Park Heights", etc.); the answer is a name-mismatch artifact, not a finding.

## Concurrency Analysis
- `concurrency.go:31-46`: three goroutines, one per CSV loader, each returning on its own buffered channel (cap 1); main blocks on the three receives. Correct.
- `analytics.go:35-78`: three more goroutines, each counting one dataset into its own map, returned via channels; merged sequentially (80-100).
- No worker pool, no parallel parsing within a file, no concurrent join. Each loader reads the whole file into a `[]struct`.
- Level: **token** (goroutine-per-file for load and for count).

## Memory / Resource Analysis
Loads all 224,623 arrest rows into `[]WbRecord` (three strings each) before any processing — the opposite of streaming; 50 MB RSS (183 MB under -race). Prints elapsed time only. No MemStats, no bounded channels beyond cap-1 result channels, no preallocation, no discussion in README beyond "goroutines and channels". `defer file.Close()` present. Minimal evidence.

## Architecture & Code Quality
Files are split by role (load / concurrency / analytics) which is a reasonable start, and `normalizeNeighborhood` is a sensible join key. But: struct field names contradict the data (`OfficerName`, `SettlementAmount` for a gun-offender registry); `Summary.WellBeingCount` means "arrest count"; file names ("revealer", "wb" = "well being") are idiosyncratic and undocumented; `fmt.Println` everywhere; Q2 dumps ~230 lines; no tests; no flags; `go.mod` says go 1.27.2. README is an honest first-person narrative ("first time using Golang", used Gemini/ChatGPT) rather than a build/verify document; no macOS/verify section. No named design pattern.

## Stakeholder Questions & Value Assessment
No stakeholder is named. Q1 (count of distinct neighborhoods) and Q2 (list neighborhoods present in both arrests and registry — nearly all of them) are trivial set operations; Q3/Q4 are single-dataset max/min over 10 Safe Streets sites; Q5 is a cross-dataset set difference whose only answer is a naming artifact. No per-neighborhood rates, no ranking, no chart or table beyond five sample rows. The idea — compare arrests, gun-offender residence and violence-intervention activity by neighborhood — could have been valuable to MONSE or BPD, but nothing is computed that they could act on. Visualization: none.

## Issues
1. Token concurrency (goroutine-per-file) and full in-memory loads; no streaming, pool, or concurrent join.
2. Questions are trivial counts/sets; Q3/Q4 single-dataset; Q5 answer is a name-mismatch artifact; no stakeholder; no visualization.
3. Data is reshaped/renamed rather than raw Open Baltimore exports (synthetic `RecordType`/`Count` columns); third dataset is a hand-typed PDF extract; struct names contradict the data.
4. `scripts/gen_mock.go` is broken (decimal permission bits) and would clobber real data if it worked; not default.
5. README is narrative, claims `make build` runs the program, lacks macOS/verify instructions; go.mod go 1.27.2 forces toolchain download; stray .DS_Store (Canvas copy).

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | `make build`/`go build` compile cleanly first try (toolchain auto-selected). |
| real_data | 12/15 | Two datasets verified as derived from real Open Baltimore Arrests and Gun Offenders Registry, bundled and used; deduct for reshaped/renamed non-raw format with synthetic constant columns, and a (broken, non-default) mock generator. |
| concurrency | 5/15 | Goroutine-per-file for load and count, cap-1 channels; no pool, no concurrent parse/join. |
| memory | 2/10 | Loads everything into slices; only an elapsed-time print; no design or documentation. |
| architecture | 7/15 | Multi-file role split and a normalization key, but misnamed structs, no tests, no pattern, no flags. |
| stakeholder | 4/15 | No stakeholder; trivial count/set questions, two single-dataset, Q5 an artifact; no visualization. |
| runs_results | 9/10 | Runs end-to-end in 0.3 s and prints all five answers; output is verbose and Q5 misleading. |
| readme | 2/5 | Narrative README with three steps, one of them mis-described; no macOS/verify section. |
| **Total** | **56/100** | **F** |

## Final Grade Rationale
The program builds and runs cleanly on bundled data that traces back to two real Open Baltimore datasets, which carries the grade through build, data and run credit. Everything else is thin: concurrency is one goroutine per file, all data is loaded into memory, the five questions are trivial counts and set memberships with no named stakeholder, visualization or actionable result, and the README is a reflection rather than build/verify documentation. The direct_zip version differs from Canvas only by two README lines. 56/100, an F.
