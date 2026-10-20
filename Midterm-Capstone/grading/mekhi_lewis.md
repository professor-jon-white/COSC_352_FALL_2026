# Mekhi Lewis — COSC 352 Midterm Capstone Grading Report

**Summary: 75/100 (C).** Solid Go project with a real producer/worker-pool CSV pipeline over bounded channels, a test, a race-clean build, and two genuine Open Baltimore datasets bundled (399K BPD arrest rows, 943K 311 rows). It is held back by the default run path using 3-row synthetic fixtures, a sequential aggregate stage, three of five questions being single-dataset, no named stakeholders, and a README whose question list does not match what the program prints.

## Inventory
- Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/mekhi_lewis/baltimore-intelligence-engine/`
- Language: Go (go.mod `go 1.22`, module `baltimore-intelligence-engine`, no external deps)
- Source files (940 LOC): `internal/engine/engine.go` 593, `internal/ui/menu.go` 181, `cmd/bcie/main.go` 134, `internal/engine/engine_test.go` 32
- README: yes. Run script: `run.sh`. Tests: 1 (`TestPipeline`). Late: no. Also `reports/summary.txt` (pre-generated output), `.gitignore`, two `.DS_Store` files.
- Data (`data/`): `arrests.csv` 399,793 lines / 68 MB; `requests_test.csv` 943,284 lines / 386 MB; `requests_small.csv` 10,001 lines / 4 MB; `arrests_sample.csv` 4 lines; `requests_sample.csv` 4 lines.

## Datasets & Data Realness
- **BPD Arrests (2026)** — `data/arrests.csv` header: `X,Y,RowID,IncidentNumber,ArrestNumber,Age,Gender,Race,ArrestDateTime,ArrestLocation,IncidentOffence,IncidentLocation,Charge,ChargeDescription,District,Post,Neighborhood,Latitude,Longitude,GeoLocation,Shape`. Sample rows: `... 26000003,37,M,B,2026/01/01 03:07:00+00,1 HOWARD ST,Unknown Offense,1 HOWARD ST,1 1119,CDS: POSS W/I DIST: NARC,Central,112,Downtown,39.2894,-76.6194 ...`; `1300 NORTH AVE ... Western,728,Penn North`; `1400 GOUGH ST ... Central,126,Perkins`. This is the genuine Open Baltimore ArcGIS export schema. 399,792 data rows; 232,871 carry coordinates. 8,247 rows mention Sandtown.
- **311 Customer Service Requests (2026)** — `data/requests_test.csv` / `requests_small.csv` header: `X,Y,RowID,SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,LastActivity,LastActivityDate,Outcome,Address,ZipCode,Neighborhood,CouncilDistrict,PoliceDistrict,PolicePost,Latitude,Longitude,GeoLocation,NeedsSync,IsDeleted,HashedRecord`. Rows like `500VO00000ta054YAA,26-00004287,SW-Bulk Scheduled-Saturday,Phone,2026/01/03 14:44:56+00,Closed,... "3954 DOLFIELD AVE, Baltimore City, 21215",21215,Dolfield,6,Northwestern,624,...` and `903 S DECKER AVE ... Canton,1,Southeastern`. Genuine schema (Salesforce-style SRRecordID, real SRTypes, real neighborhoods). 943,283 rows spanning Jan–Oct 2026; `requests_small.csv` is the first 10,000 rows of the same export.
- Synthetic fixtures: `arrests_sample.csv` (`Latitude,Longitude,ChargeDescription,ArrestDate`, 3 rows) and `requests_sample.csv` (3 rows). **`run.sh` with no arguments uses these synthetic fixtures — the mock path is the default (-3 deduction).** Real data is reached via `./run.sh --real` (downloads both ArcGIS exports; verified URLs in `main.go:17-18`) or by passing `-arrests`/`-requests` paths.
- The README never mentions that the real CSVs are bundled (its checklist says to "exclude large downloaded files" and `.gitignore` lists `data/*.csv`), so the real files appear to have shipped incidentally; `reports/summary.txt` was nevertheless produced from `arrests.csv` + `requests_small.csv` (I reproduced it byte-for-byte in the header lines).
- **data_status: real_bundled** (two real datasets present and usable; default path synthetic).

## Build Results
```
go build -o <scratch>/mekhi/bcie ./cmd/bcie      -> exit 0, no output
go vet ./...                                    -> exit 0, no output
go build -race -o <scratch>/mekhi/bcie_race ./cmd/bcie -> exit 0
```
Clean build first try using the README command (`go build -o bcie ./cmd/bcie`).

## Test/Vet/Race Results
```
go test ./...        -> ok  baltimore-intelligence-engine/internal/engine  0.313s  (cmd, ui: no test files)
go test -race ./...  -> ok  baltimore-intelligence-engine/internal/engine  1.261s
```
Race-instrumented binary run on real data (`arrests.csv` + `requests_small.csv`): exit 0, **0 DATA RACE warnings**. race_detector: clean.

## Run Results
Program ends in an interactive menu (`internal/ui/menu.go`); it exits cleanly on EOF or `0`, so I fed stdin (`printf '6\n\n0\n'` to view the full report, `printf '0\n'` to exit). `-report` was pointed at the scratchpad so the student's `reports/summary.txt` was not overwritten (verified unchanged).

1. Default documented path `./run.sh` (synthetic 3-row fixtures): exit 0, 1.33 s wall (includes `go run` compile), max RSS 126 MB. Output: `Arrests: 3 | 311 Requests: 3 | Skipped Records: 0 | Grid Cells: 2`, all five Q sections print. Captured in `<scratch>/mekhi/run_sample.txt`.
2. Real bundled data `bcie -arrests data/arrests.csv -requests data/requests_small.csv`: exit 0, **1.75 s wall, max RSS 223 MB**. Output header matches the shipped `reports/summary.txt`:
```
B-MORE INSIGHT
Baltimore City Intelligence Report
=================================================================
Arrests: 224726 | 311 Requests: 7069
Skipped Records: 177997 | Grid Cells: 357

Q1: Highest Arrest Activity
-----------------------------------------------------------------
39.30,-76.64    7562 ##############################
  Arrests: 7562 | 311: 96 | Infrastructure: 19 | Open: 0
39.29,-76.62    7045 ###########################
  Arrests: 7045 | 311: 114 | Infrastructure: 14 | Open: 5
39.30,-76.65    6946 ###########################
  Arrests: 6946 | 311: 84 | Infrastructure: 23 | Open: 2
...
Q2: Highest 311 Request Volume
39.29,-76.58     172 ##############################   Arrests: 5072 | 311: 172 | Infrastructure: 40 | Open: 1
...
Q3: Arrest and Infrastructure Overlap
39.31,-76.61      49 ##############################   Arrests: 3371 | 311: 106 | Infrastructure: 49 | Open: 2
...
Q4: Combined Municipal Activity
39.30,-76.64    7658 ##############################   Arrests: 7562 | 311: 96 | Infrastructure: 19 | Open: 0
...
Q5: Unresolved 311 Requests
39.29,-76.62       5 ##############################   Arrests: 7045 | 311: 114 | Infrastructure: 14 | Open: 5
39.29,-76.67       5 ##############################   Arrests: 2361 | 311: 56 | Infrastructure: 15 | Open: 5
...
Grid cells are approximate, not official neighborhoods. / Counts are not population-normalized. / Arrests do not represent all crime incidents. / No causal relationship is implied.
```
3. Full bundled data `bcie -arrests data/arrests.csv -requests data/requests_test.csv` (386 MB): exit 0, **4.56 s wall, max RSS 948 MB**. `Arrests: 224726 | 311 Requests: 707522 | Skipped Records: 410827 | Grid Cells: 886`. Top cells: Q1 arrests 39.30,-76.64 = 7,562 (downtown/west); Q2 311 volume 39.29,-76.58 = 15,886; Q3 infrastructure overlap 39.29,-76.59 = 2,754; Q5 open 311 39.29,-76.62 = 692. Full report at `<scratch>/mekhi/report_real_full.txt`.

## Concurrency Analysis
- `cmd/bcie/main.go:65-96`: one goroutine per dataset (fetch + parse), `sync.WaitGroup`, results written to distinct slice slots (no shared-write race).
- `internal/engine/engine.go:147-329` `ParseFile`: a producer goroutine streams `csv.Reader` rows into a bounded channel `jobs` (cap `workers*4`, `engine.go:189`); `workers = runtime.NumCPU()` goroutines (`engine.go:195-253`) validate coordinates, bucket into 0.01° grid cells and emit `Record`s on a bounded `out` channel; a closer goroutine (`engine.go:255-258`) waits on the WaitGroup then closes `out`; the caller drains `out`. `select` on `ctx.Done()` for cancellation. This is a correct fan-out/fan-in worker pool for the parse stage.
- Aggregation `Analyze` (`engine.go:344-400`) and reporting are **sequential** on the main goroutine (README: "Aggregation occurs after workers finish, avoiding shared mutable map writes"). No parallel join/aggregate.
- Race detector clean in tests and on real data. Level: **partial** — real worker-pool concurrency in parsing only.

## Memory/Resource Analysis
- Streaming CSV reads (no `ReadAll`), bounded channels sized `workers*4` (back-pressure), `ReuseRecord=false` noted deliberately, 150 MiB download cap via `io.LimitReader` (`engine.go:63-80`), temp download removed with `defer os.Remove` (`main.go:78`), HTTP client timeout 120 s. README has a short memory paragraph.
- Weaknesses: all normalized `Record`s are retained in a slice (948 MB RSS on the 386 MB file); `ParseFile` re-opens and re-reads the entire CSV a second time just to count total rows for the "skipped" figure (`engine.go:303-326`), doubling I/O; `fmt.Sprintf` per row for the cell key allocates heavily. Reasonable, documented design with some inefficiencies.

## Architecture & Code Quality
- Standard `cmd/` + `internal/` layout with `engine` and `ui` packages; one deterministic pipeline test; `run.sh`; flags for workers/report path.
- `engine.go` is a 593-line file that mixes HTTP download, CSV parsing, normalization, aggregation, and report rendering — the ingestion/analysis/output separation the assignment asks for is by function, not by package. The five analyses are a table-driven map of `{title, score, keep}` closures (`engine.go:451-497`), a lightweight Strategy that is actually used. Header normalization and field-alias lookup (`norm`, `field`) make the parser schema-tolerant across both datasets — a nice touch.
- README does not name any design pattern; "Architecture" section is a file list. The UI menu re-parses the rendered report text by `Q<n>:` prefixes (`menu.go:139-173`) instead of using the structured `Result`, a brittle design.
- Mismatch: README's "Five stakeholder analyses" (1 combined volume, 2 infrastructure+arrests, 3 resource-attention ranking, 4 arrest concentrations, 5 unresolved 311) are in a different order and wording from the code/menu (Q1 arrests, Q2 311 volume, Q3 overlap, Q4 combined, Q5 unresolved). Code comments are sparse; a few stray ones ("//fake end", "//Detect func").

## Stakeholder Questions & Value Assessment
No stakeholders are named anywhere (README says "municipal leaders" only in the title tagline; the assignment asks for named stakeholders such as Mayor's Office/DOT/Planning).
1. Highest Arrest Activity by grid cell — single-dataset.
2. Highest 311 Request Volume by grid cell — single-dataset.
3. Arrest and Infrastructure Overlap — cells with arrests AND infrastructure-type 311 requests (keyword match on pothole/street/light/trash/...); cross-dataset co-location count.
4. Combined Municipal Activity — arrests + 311 count per cell; cross-dataset sum.
5. Unresolved 311 Requests — open-status 311 per cell; single-dataset.
Value: the join key is a 0.01° lat/lon bin (~1.1 km × 0.85 km), not neighborhoods, so a Mayor's Office reader gets coordinates like `39.29,-76.62` instead of "Downtown/Westside." Q3/Q4 are coincidental co-counting rather than a relationship or rate. The caveats printed under every section are honest and appropriate. The output is a ranked ASCII bar chart plus a colored terminal menu; the real-data run does surface real hotspots (downtown west cells dominate both arrests and 311), so there is some descriptive value, but little that is actionable.

## Issues
- Default `./run.sh` path runs on 3-row synthetic fixtures; real data requires `--real` (network) or manual flags (-3 per rubric).
- README does not mention the bundled real CSVs (`.gitignore` even excludes `data/*.csv`); the 386 MB `requests_test.csv` appears to have shipped unintentionally.
- README's list of the five analyses does not match the titles/order the program prints.
- No stakeholders named; Q1, Q2, Q5 are single-dataset; Q3/Q4 are co-location counts in coarse grid cells, not neighborhoods.
- Aggregation and reporting are sequential; only parsing uses the worker pool.
- `ParseFile` re-reads the whole file to compute the skipped count; all records retained (948 MB RSS on full data).
- `engine.go` is a single 593-line file spanning fetch/parse/analyze/render; UI re-parses report text instead of using structured results.
- Program ends in an interactive menu (handled with piped stdin; exits on EOF, so it is scriptable).

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | `go build`, `go vet`, `go build -race` all clean first try with the documented command. |
| real_data | 12/15 | Two genuine Open Baltimore datasets bundled (399K arrests, 943K 311) with correct schemas and used successfully; -3 because the default run path is the synthetic fixtures. |
| concurrency | 10/15 | Correct bounded-channel producer/worker-pool parse stage with WaitGroup and ctx cancellation, race-clean; aggregation/reporting sequential. |
| memory | 7/10 | Streaming reads, bounded channels, download cap, temp cleanup, documented; but all records retained, file read twice, heavy per-row Sprintf. |
| architecture | 10/15 | cmd/internal layout, a test, table-driven analyses; but one 593-line file does everything, no named patterns, brittle text-parsing UI. |
| stakeholder | 7/15 | Five questions with honest caveats and ASCII bars, but no named stakeholders, three single-dataset, cross-dataset ones are grid-cell co-counts; README list mismatches output. |
| runs_results | 10/10 | Runs end-to-end on the real bundled data (1.8 s small, 4.6 s full) with documented flags and sensible, reproducible output. |
| readme | 4/5 | Clear macOS build/run/test/verify steps and run.sh that work; but omits bundled real data and misdescribes the five analyses. |
| **Total** | **75/100** | **C** |

## Final Grade Rationale
This is a working engine: it builds and tests cleanly, is race-free, and chews through 1.3 million real Open Baltimore rows in under five seconds using a genuine bounded worker pool. Points are lost where the assignment's analytical and documentation requirements are concerned: the shipped default path is synthetic, the aggregate stage is sequential, the five questions are mostly single-dataset counts over anonymous grid cells with no named stakeholder, and the README both hides the real data and describes a different question list than the program prints. 75/100, C.
