# Elijah Stewart — COSC 352 Midterm Capstone Grading Report

**Summary: 48/100 — F.** Go program builds cleanly, is race-clean, and streams CSVs, but it ships only ~16 rows of fabricated "sample" data with a made-up schema, provides no download URL or script for the real datasets, uses only token concurrency (one goroutine per file), and its single cross-dataset "join" is a case-sensitive label intersection that collapses to a single bogus match ("CARE") when run on real data.

## Inventory
Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/elijah_stewart/baltimore-intel/`

| File | LOC / size | Notes |
|---|---|---|
| main.go | 178 | all logic |
| main_test.go | 16 | 2 trivial unit tests |
| go.mod | 3 | module baltimore-intel, go 1.21 |
| README.md | 87 lines | |
| run.sh / build.sh | 3 / 4 | `go run . "$@"` / `go build -o baltimore-intel .` |
| data/311_sample.csv | 297 B, 9 lines (8 rows) | synthetic |
| data/crime_sample.csv | 198 B, 7 lines (6 rows) | synthetic |

Language: Go. Tests: yes (2). Total source LOC: 194. Data bundled: 495 bytes total.

## Datasets & Data Realness
README claims: Baltimore 311 Customer Service Requests and BPD Arrests. Neither is bundled; README says "The large source CSVs are intentionally not bundled... Keep the originals you downloaded" but gives **no URL, dataset ID, or fetch script**. The program accepts http(s) URLs but none is documented.

Sample files (`--offline` mode) are **fabricated, not excerpts**:
```
crime_sample.csv:  crime_type,neighborhood,district        (6 rows: Theft/Assault/Robbery in Inner Harbor/Cherry Hill/Westport)
311_sample.csv:    service_type,neighborhood,community     (8 rows: Pothole/Trash/Street Cleaning/...)
```
Real Open Baltimore schemas are `X,Y,RowID,SRRecordID,ServiceRequestNum,SRType,...,Neighborhood,...` (311) and `RowID,IncidentNumber,ArrestNumber,Age,Gender,Race,ArrestDateTime,ArrestLocation,IncidentOffence,...,ChargeDescription,District,Post,Neighborhood,...` (BPD Arrests). The sample headers match neither; values like "Pothole"/"Theft" are not real SRType/IncidentOffence values. README does honestly label them "synthetic sample data". Row counts: 311_sample 8, crime_sample 6.

**data_status: sample_only** (fabricated samples; real data absent; no generator, no download script).

## Build Results
```
$ go build -o <scratch>/baltimore-intel .      -> exit 0  (3.9 s wall, first try)
$ go build -race -o <scratch>/bi-race .        -> exit 0
```
Clean. build.sh matches.

## Test / Vet / Race Results
```
$ go vet ./...     -> exit 0, no output
$ go test ./...    -> ok  baltimore-intel  0.289s
```
Race run (`bi-race --offline`): 0 DATA RACE reports. Race run on real data (supplementary, see below): 0 DATA RACE reports.

## Run Results
Default run (documented `go run .`) with no data present:
```
Could not load both datasets.
311: open 311_Customer_Service_Requests_Yearly_7199422805573403920.csv: no such file or directory
arrests: open Arrest.csv: no such file or directory
Use --offline for samples, or pass --311 and --arrests with valid CSV file paths.
exit=1
```
Real data absent -> documented sample path `--offline` (1.24 s wall, 29 MB max RSS):
```
BALTIMORE MUNICIPAL INTELLIGENCE
311 service requests: 8 | Arrest records: 6
1. Which 311 service-request types appear most often?
Pothole 2 / Street Cleaning 2 / Trash 2 / Illegal Dumping 1 / Water Leak 1   (with # bars)
2. Which neighborhoods have the most 311 requests?   Cherry Hill 3 / Inner Harbor 3 / Westport 2
3. Which offenses ... in arrest records?             Theft 3 / Assault 2 / Robbery 1
4. Which neighborhoods ... in arrest records?        Cherry Hill 3 / Inner Harbor 2 / Westport 1
5. Which matching area labels have records in both datasets?  Cherry Hill 6 / Inner Harbor 5 / Westport 3
```
Full output: scratchpad `elijah/run_offline.txt`.

**Supplementary (grader-initiated, not a documented path):** ran the race binary against real Open Baltimore files bundled by other students (943,283-row 311 export and a 54,076-row NIBRS crime export) to see if the code handles real schemas. It ran (13.9 s, 51 MB RSS, no races) and produced sensible per-dataset charts (e.g., "ECC-Information Request 231754", "Unknown 235909" for 311 neighborhoods), **but Q5 matched only one label, "CARE"**, because the join at main.go:170-175 compares raw strings and the 311 file uses mixed case ("Downtown") while crime/arrest files use upper case ("DOWNTOWN"). The cross-dataset question is therefore effectively broken on real data.

## Concurrency Analysis
- `main.go:142-152`: `sync.WaitGroup` with exactly two goroutines, one per file, each calling `summarize()` sequentially over its CSV. Results written to distinct variables `a`, `b`, read after `wg.Wait()` — correct, no race.
- No worker pool, no channels, no parallel parsing/join/aggregation. The join (170-175) and chart rendering are sequential.
- Level: **token** (two goroutines loading two files).

## Memory / Resource Analysis
- Streams rows with `csv.Reader`, keeps only `ByType`/`ByArea` count maps (main.go:68-85) — good; README documents this. 51 MB RSS on a 385 MB input confirms streaming.
- Allocates a fresh `map[string]string` per row (line 76) — avoidable per-row heap churn; column indices could be resolved once.
- `defer r.Close()` on files/HTTP bodies; HTTP client has a 2-minute timeout. No MemStats, no bounded channels, no preallocation beyond the map hint.

## Architecture & Code Quality
Single `main.go` with small helper funcs (`choose`, `summarize`, `top`, `chart`). Reasonable naming; terse one-line function style. No packages, no named design pattern realized (README's "Architecture" bullets describe streaming, not a pattern). `FieldsPerRecord=-1` + `LazyQuotes` + silent `continue` on parse errors hides data problems. Tests cover `choose` and `top` only. The heavy README disclaimers ("not a crime rate, causal claim...") read as boilerplate relative to the very thin analysis.

## Stakeholder Questions & Value Assessment
Stakeholders: none named (README says "stakeholder-oriented questions" only). Questions 1-4 are single-dataset top-8 frequency counts; Q5 is a label intersection summing counts from two unrelated datasets ("combined counts; exploratory"), which the README itself concedes is not meaningful. The Mayor's Office/DOT/Planning would learn nothing beyond "most common SRType" — and with the bundled sample, nothing at all. Visualization: ASCII `#` bars (works).

## Issues
1. Fabricated sample data with invented schema; real data not bundled and no URL/script to obtain it.
2. Only 2 of 5 questions touch both datasets, and the one cross-dataset join is case-sensitive and sums unrelated counts.
3. Token concurrency: two goroutines, no pool, no concurrent join/aggregate.
4. Per-row map allocation; silently skips malformed rows.
5. README leads with Windows/PowerShell instructions; macOS section present but no dataset links; `--offline` is the only runnable documented path.

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | Approved language; `go build` clean first try; build.sh works. |
| real_data | 3/15 | Only fabricated 14-row samples with non-Open-Baltimore headers; no download URL/script/generator; real data absent. |
| concurrency | 5/15 | Two goroutines + WaitGroup loading two files; nothing else concurrent. |
| memory | 5/10 | Streaming aggregation documented and evident; but per-row map alloc, no bounded channels/MemStats. |
| architecture | 6/15 | Single organized file with helpers and 2 tests; no packages, no realized pattern. |
| stakeholder | 5/15 | No named stakeholders; 4/5 single-dataset counts; Q5 join is trivial and broken on real-case data. |
| runs_results | 5/10 | Runs end-to-end only on synthetic `--offline` samples; default documented path exits 1. |
| readme | 4/5 | Clear macOS build/run/test steps and run.sh, but no dataset source links; Windows-first. |
| **Total** | **48/100** | **F** |

## Final Grade Rationale
The code compiles, vets, tests and is race-free, and it correctly streams CSVs, which earns full build credit and partial memory credit. However, the assignment's core requirements — real Open Baltimore data on disk, concurrent parse/join/aggregate across a worker pool, and five cross-dataset stakeholder questions — are largely unmet: the data is fabricated, concurrency is two goroutines, and only one question joins the datasets (incorrectly, on real data). The result is a working skeleton rather than a data-processing engine, landing at 48/100.
