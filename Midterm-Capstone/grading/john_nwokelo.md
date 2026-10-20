# John Nwokelo — COSC 352 Midterm Capstone Grading Report

**Summary: 80/100 (B)** — Go, multi-package, builds/tests/vets clean, runs end-to-end on the real 812 MB 311 GeoJSON + NSA boundary GeoJSON that he submitted alongside the code (20 s, 2.26 GB RSS). Spatial point-in-polygon join is real, but all five questions are 311-only statistics grouped by neighborhood, the whole file is loaded into memory, and the bundled default path is a 45-record synthetic sample.

## Inventory

Path: `submissions/john_nwokelo/baltimore-intelligence/` plus two separately-uploaded data files at `submissions/john_nwokelo/`.

| File | LOC | Notes |
|---|---|---|
| cmd/engine/main.go | 56 | flags, two loader goroutines, Join, Analyze, Report |
| internal/models/models.go | 20 | Point, Neighborhood (polygons + bbox), Request, Joined |
| internal/ingestion/loader.go | 125 | `os.ReadFile` + `json.Unmarshal` of FeatureCollections; property key aliases |
| internal/ingestion/dates.go | 70 | epoch ms/s and string date parsing |
| internal/ingestion/dates_test.go | 51 | 3 tests |
| internal/processing/joiner.go | 93 | ray-casting point-in-polygon with holes; worker-pool fan-out/fan-in |
| internal/processing/joiner_test.go | 16 | 1 test |
| internal/analysis/analyzer.go | 141 | segmented parallel aggregation + merge; priority score |
| internal/analysis/analyzer_test.go | 42 | 2 tests |
| internal/output/reporter.go | 95 | five Q sections, `#` bar chart, warnings |
| internal/output/reporter_test.go | 39 | 1 test |
| run.sh | 4 | `go run ./cmd/engine "$@"` |
| run-windows.cmd | — | Windows helper |
| README.md | — | build/run/verify |
| data/sample_requests.geojson | 16 KB, 45 features | **synthetic** ("SYNTH-001", "Synthetic West") |
| data/sample_neighborhoods.geojson | 1.7 KB, 3 features | **synthetic** rectangles |
| nwokelojohn_94012_8488239_requests.geojson | **812 MB**, 943,290 lines | **real** `311_Customer_Service_Requests_2026` |
| nwokelojohn_94012_8488216_neighborhoods.geojson | 5.7 MB, 279 features | **real** `Neighborhood_NSA` |

Total Go LOC: 748 (600 non-test). Go 1.21, stdlib only. Tests in 4 packages. Run script: `run.sh`.

## Datasets & Data Realness

Declared: Open Baltimore **311 Customer Service Requests** (GeoJSON) and **Neighborhood Statistical Area (NSA) Boundaries** (GeoJSON). README says to save them as `data/requests.geojson` and `data/neighborhoods.geojson`; the two large files were uploaded to Canvas separately rather than inside the zip.

Real requests file head (genuine Open Baltimore schema: RowID, SRRecordID, ServiceRequestNum, SRType, MethodReceived, CreatedDate, SRStatus, StatusDate, DueDate, CloseDate, Agency, Address, ZipCode, Neighborhood, CouncilDistrict, PoliceDistrict, PolicePost, Latitude, Longitude, GeoLocation ...):

```
"name": "311_Customer_Service_Requests_2026",
{"type":"Feature","properties":{"RowID":2,"SRRecordID":"500VO00000ta054YAA","ServiceRequestNum":"26-00004287","SRType":"SW-Bulk Scheduled-Saturday","MethodReceived":"Phone","CreatedDate":"2026-01-03T14:44:56Z","SRStatus":"Closed",...,"Address":"3954 DOLFIELD AVE, Baltimore City, 21215","ZipCode":"21215","Neighborhood":"Dolfield","CouncilDistrict":"6","PoliceDistrict":"Northwestern",...},"geometry":{"type":"Point","coordinates":[-76.68284346,39.3403914800001]}}
{"type":"Feature","properties":{"RowID":3,...,"SRType":"FOR-Tree Maintenance",...,"Address":"434 E FRANKLIN ST, Baltimore City, 21202","Neighborhood":"Mount Vernon",...}}
```

Real neighborhoods file head: `"name": "Neighborhood_NSA"`, properties `OBJECTID, Name ("Beechfield"), Population, White, Blk_AfAm, ..., Total_Units, Occ_Vacant, ...` with Polygon/MultiPolygon geometry — the Open Baltimore NSA layer with 2010 census attributes. The loader accepts `nbrdesc`/`Name`/`NAME`/`name` (`loader.go:90`), so it binds.

Bundled samples are explicitly synthetic ("SYNTH-001", "Synthetic West/Central/East" rectangles) and are the **default** `-requests`/`-boundaries` flag values (`main.go:18-19`). README is honest: "The included sample files contain made-up data".

Row counts: real requests ~943,285 features (943,290 lines; program kept 707,579 with valid Point geometry — 235k have `"geometry":null`); real neighborhoods 279 features; sample_requests 45; sample_neighborhoods 3.

**data_status: real_bundled** (submitted with the assignment, outside the zip; synthetic sample is the default path -> rubric deduction).

## Build Results

```
$ cd submissions/john_nwokelo/baltimore-intelligence
$ go build -o baltimore-engine ./cmd/engine     # exit 0
$ go vet ./...                                  # exit 0, clean
$ go build -race -o engine-race ./cmd/engine    # exit 0
```

## Test / Vet / Race Results

```
$ go test ./...
ok  	baltimore-intelligence/internal/analysis	1.423s
ok  	baltimore-intelligence/internal/ingestion	1.611s
ok  	baltimore-intelligence/internal/output	1.243s
ok  	baltimore-intelligence/internal/processing	1.782s
$ go test -race ./...    # all ok
```

7 tests pass, also under `-race`. Race binary on the bundled sample: exit 0, **0 DATA RACE**. Race run on the 812 MB file skipped (plain run already used 2.26 GB RSS; race instrumentation would risk the 16 GB machine).

## Run Results

**(a) Default/sample path** (`sh run.sh`): 0.40 s, 70 MB RSS. Output: three "Synthetic" neighborhoods with 15 requests each — see `scratchpad/john/sample_stdout.txt`.

**(b) Real data** — code tree copied to scratchpad, the two uploaded GeoJSONs symlinked to `data/requests.geojson` and `data/neighborhoods.geojson` as the README directs (no code change):

```
$ ./baltimore-engine -requests data/requests.geojson -boundaries data/neighborhoods.geojson
```

- Wall: **20.27 s** real (14.5 user, 7.3 sys); max RSS **2,260 MB**.
- Captured: `scratchpad/john/real_stdout.txt`, `real_stderr.txt`.

Full output:

```
BALTIMORE CITY SERVICE REPORT
Requests processed: 707579 | Geographically unmatched: 14935 | Neighborhoods with matches: 279

Q1. WHICH NEIGHBORHOODS REPORT THE MOST PROBLEMS?
Belair-Edison                  24783 ####################################
Coldstream Homestead Montebell 16428 ########################
Downtown                       13476 ####################
Canton                         13119 ###################
Brooklyn                       11878 #################
Central Park Heights           10000 ###############
Washington Village/Pigtown      9543 ##############
Broadway East                   9453 ##############
Frankford                       9165 #############
Sandtown-Winchester             9063 #############

Q2. WHAT PROBLEMS ARE MOST COMMON IN EACH NEIGHBORHOOD?
Belair-Edison                  SW-Rat Rubout Proactive             10200
Coldstream Homestead Montebell SW-Rat Rubout Proactive             5099
Downtown                       FCPF-Trial Request-Parking          1184
Canton                         SW-Rat Rubout Proactive             1614
Brooklyn                       SW-Rat Rubout Proactive             3551
Central Park Heights           SW-Rat Rubout Proactive             2203
Washington Village/Pigtown     SW-Rat Rubout Proactive             2832
Broadway East                  SW-HGW                              1964
Frankford                      SW-Rat Rubout Proactive             1060
Sandtown-Winchester            SW-Rat Rubout Proactive             1466

Q3. WHERE IS THE HIGHEST PERCENTAGE OF PROBLEMS STILL UNRESOLVED?
Johns Hopkins Homewood           35.2% (177/503)
Morgan State University          30.6% (171/559)
Inner Harbor                     29.8% (308/1032)
Saint Paul                       29.2% (73/250)
Jones Falls Area                 28.2% (223/791)
University of Maryland           27.2% (184/677)
Patterson Park                   26.3% (177/674)
Locust Point Industrial Area     26.2% (84/320)
Sabina-Mattfeldt                 25.8% (109/422)
Middle Branch/Reedbird Parks     24.8% (79/318)

Q4. WHERE DOES IT TAKE THE LONGEST TO FIX PROBLEMS?
Closed requests with valid creation and closure dates: 607194 / 618896
Curtis Bay Industrial Area        25.3 days (n=117)
Johns Hopkins Homewood            20.6 days (n=309)
Carroll Park                      19.3 days (n=194)
Clifton Park                      19.1 days (n=259)
Perring Loch                      19.0 days (n=1945)
Dickeyville                       19.0 days (n=174)
Mount Winans                      18.9 days (n=525)
Overlea                           18.2 days (n=507)
Hopkins Bayview                   18.2 days (n=250)
North Harford Road                17.3 days (n=3459)

Q5. WHICH NEIGHBORHOODS NEED THE MOST ATTENTION?
Higher scores suggest which areas to review first (not proof of greater need).
Belair-Edison                   49.4 / 100
Coldstream Homestead Montebell  38.5 / 100
Canton                          35.0 / 100
Downtown                        35.0 / 100
Johns Hopkins Homewood          31.2 / 100
Curtis Bay Industrial Area      30.2 / 100
Frankford                       29.9 / 100
Brooklyn                        29.4 / 100
Hampden                         27.4 / 100
Washington Village/Pigtown      27.1 / 100

Priority = 40% relative volume + 40% unresolved rate + 20% relative mean closure time.
WARNING: Some closed requests lack valid dates, so resolution-time rankings may be biased.
WARNING: Scores are descriptive, not causal; reporting access and population differ by area.
```

Results are sensible and consistent with the other students' independent runs of the same 2026 311 data (Belair-Edison ~24.8k, Coldstream ~16.4k, Downtown ~13.5k).

## Concurrency Analysis

- `cmd/engine/main.go:30-34` — two goroutines load the two files concurrently with a `sync.WaitGroup` (each load is a single `os.ReadFile` + `json.Unmarshal`; parsing itself is not parallel).
- `internal/processing/joiner.go:46-92` — `Join`: bounded `jobs`/`results` channels (`workers*2`), `workers` goroutines run bbox + ray-casting point-in-polygon per request, `context` cancellation, WaitGroup closes results, single collector. This is the CPU-heavy stage and it is genuinely parallel (14.5 s user over 20 s wall).
- `internal/analysis/analyzer.go:46-111` — `Analyze` splits the joined slice into `workers` contiguous segments, each goroutine builds a private `map[string]*Stats`, results merged by the main goroutine via a channel.
- Neighborhood slice is read-only during Join; no shared mutable state; race-clean on tests/sample.

Level: parallel join and aggregate with worker pools; parsing only "two goroutines for two files". Scored 12/15.

## Memory / Resource Analysis

- `loader.go:47-51` reads the **entire 812 MB file into memory** and unmarshals it into `map[string]json.RawMessage` per feature -> 2.26 GB RSS. No streaming decoder (`json.Decoder` token loop would have avoided this).
- Some evidence of thought: timestamps stored as `int64` ms rather than `time.Time` (`models.go:14`), slices preallocated with capacity (`loader.go:65,88`; `joiner.go:84`), bounding-box prefilter before polygon test (`joiner.go:24`), bounded channels, per-worker maps.
- README: "Go manages memory automatically. Limited worker pools and separate worker data help avoid unsafe shared access. Large datasets may use substantial memory." Honest but thin.

Scored 5/10.

## Architecture & Code Quality

Proper Go layout: `cmd/engine` + `internal/{models,ingestion,processing,analysis,output}`, each stage in its own package with its own tests. Pipeline is clear (load -> join -> analyze -> report). No named GoF pattern beyond pipeline/fan-out-fan-in, and no interfaces (Strategy/Factory) — the README does not claim them. Code is compact and readable; `Stats` methods (`Rate`, `AvgDays`, `Common`) are tidy; reporter has a generic `rank(less)` helper. Minor smells: one-line struct literal at `loader.go:79` is 300+ chars; `run-windows.cmd` text ("Copy your 311 download from the old project's data folder") is a leftover from another project; priority weights hard-coded. Tests are meaningful (date formats, priority fallback, headings regression).

## Stakeholder Questions & Value Assessment

| # | Question | Datasets actually used |
|---|---|---|
| 1 | Which neighborhoods report the most problems? | 311 counts by (spatially-assigned) neighborhood |
| 2 | Most common problem per neighborhood | 311 only |
| 3 | Highest % unresolved | 311 only |
| 4 | Longest average time to close | 311 only |
| 5 | Composite priority score (40/40/20) | 311 only |

The NSA boundary dataset contributes only the grouping key via point-in-polygon — and the 311 file already carries a `Neighborhood` attribute, so the second dataset adds nothing analytically (it does validate/normalize geography, and 14,935 points fell outside any NSA). None of the questions combines an attribute from the second dataset (e.g., the NSA layer's `Population`, `Occ_Vacant`, `Total_Units` were right there and would have enabled per-capita rates or vacancy-vs-311 insight). Stakeholders are "city officials" — not named. Q1-Q3 are basic counts; Q4/Q5 are a bit more interesting (Curtis Bay Industrial 25 days to close; campus neighborhoods have the highest open rates). Output is tables + `#` bars; good caveat warnings.

## Issues

1. All five questions are single-dataset 311 aggregates; the boundary layer is only a lookup key already present in the 311 data; no named stakeholders.
2. Whole 812 MB file loaded into memory (`os.ReadFile` + full unmarshal): 2.26 GB RSS; no streaming.
3. Synthetic sample is the default path; real data was uploaded separately rather than placed where the README expects (grader symlinked it).
4. Parsing is not parallel (one goroutine per file).
5. ~235k 311 records with null geometry are silently dropped (no count reported), and `lon==0||lat==0` filter is fine but undocumented.
6. Leftover Windows script text referencing "the old project".

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build ./cmd/engine` clean first try; vet clean. |
| real_data | 15 | 12 | Two real Open Baltimore GeoJSONs submitted and used (15), minus 3 because a synthetic sample is the default path. |
| concurrency | 15 | 12 | Worker-pool spatial join and segmented parallel aggregation with correct sync; parsing is just two goroutines. |
| memory | 10 | 5 | Bounded channels, bbox prefilter, int64 timestamps, preallocation; but entire 812 MB file read and unmarshalled at once (2.26 GB RSS). |
| architecture | 15 | 14 | Clean cmd/internal multi-package layout with tests per package; no explicit Strategy/Factory abstractions. |
| stakeholder | 15 | 7 | Five questions but all are 311-only statistics; boundary dataset only supplies the group key; stakeholders unnamed. |
| runs_results | 10 | 10 | Runs end-to-end on the real data in 20 s with sensible, cross-validated results. |
| readme | 5 | 5 | Clear macOS steps, run.sh, verify commands; matches reality. |
| **Total** | **100** | **80** | **B** |

## Final Grade Rationale

This is a well-structured, tested, multi-package Go program that actually processes the real 2026 311 dataset (700k geocoded requests) against real NSA boundaries with a parallel point-in-polygon join and finishes in 20 seconds. It loses points for memory (naive read-all of an 812 MB file), for a synthetic default path, and most of all for the analysis: every question is a 311-only count/rate grouped by neighborhood, so the "cross-dataset" requirement is met only mechanically by the spatial lookup, and the stakeholders are generic. Total 80, B.
