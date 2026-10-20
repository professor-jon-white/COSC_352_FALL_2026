# Tayven Morton — COSC 352 Midterm Capstone Grading Report

**Summary: 93/100 — A** (Go; two real bundled Open Baltimore extracts with verified SHA-256 checksums and a provenance manifest; builds, 16 tests and race detector clean; deterministic end-to-end run in 2.5 s producing five cross-dataset answers.)

## Inventory

Location: `submissions/tayven_morton/baltimore-capstone/`

| Item | Value |
|---|---|
| Language | Go (`go.mod`: `module baltimore-capstone`, `go 1.22`, no dependencies) |
| Source files | 11 `.go` files, 1,332 LOC (980 non-test / 352 test): `cmd/analyzer/main.go` 96; `cmd/fetch/main.go` 222 (+87 test); `internal/model/model.go` 106 (+59); `internal/ingest/ingest.go` 172 (+111); `internal/analysis/analysis.go` 162 (+71); `internal/report/report.go` 209 (+37) |
| README | `README.md` — detailed: macOS quick start, verification, questions, provenance table, architecture, memory, matching rules, data-quality findings, assignment alignment |
| Other docs | `VERIFICATION.md` (record of checks), `CODE_WALKTHROUGH.md` (follow-one-record guide) |
| Run scripts | `run.sh` (build + exec), `verify.sh` (test, vet, build, `shasum -a 256 -c`, 1-vs-4 worker determinism, compare to bundled output) |
| Tests | 5 test files, 16 test functions |
| Data bundled | `data/requests.jsonl` 67,971,176 bytes / 243,620 lines; `data/vacancies.jsonl` 1,925,840 bytes / 11,481 lines; `data/manifest.json`; `data/SHA256SUMS` |
| Pre-computed output | `output/report.md`, `output/neighborhoods.csv` (byte-identical to the grader's fresh run) |

## Datasets & Data Realness

| Dataset | Source URL (manifest) | Filter | Rows | Fields |
|---|---|---|---|---|
| 311 Customer Service Requests 2025 | `services1.arcgis.com/UWYHeuuJISiGmgXx/.../311_Customer_Service_Requests_2025/FeatureServer/0` | `CreatedDate >= 2025-10-01 AND < 2026-01-01` | 243,620 (122 pages) | `ESRI_OID,ServiceRequestNum,CreatedDate,CloseDate,SRType,SRStatus,Agency,Neighborhood,IsDeleted` |
| Vacant Building Notice – Open | `egisdata.baltimorecity.gov/.../DHCD_Open_Baltimore_Datasets/FeatureServer/1` | `1=1` | 11,481 (12 pages) | `OBJECTID,NoticeNum,DateNotice,BLOCKLOT,Address,Neighborhood` |

Checksum verification by the grader: `shasum -a 256 -c data/SHA256SUMS` -> `data/requests.jsonl: OK`, `data/vacancies.jsonl: OK`; the hashes match `manifest.json` (`a82b81d7…8b8b0`, `4e376b76…5d58`). Manifest records download window 2026-10-16 01:11-01:13 UTC and `source_count_before == records == source_count_after` for both.

Sample records (raw ArcGIS JSON, one feature per line):
```
{"attributes":{"ESRI_OID":401261,"ServiceRequestNum":"25-00850497","CreatedDate":1759394332000,"CloseDate":1765967940000,"SRType":"TRS-48 Hour Parking/Abandoned Vehicle","SRStatus":"Closed","Agency":null,"Neighborhood":"Greektown","IsDeleted":0}}
{"attributes":{"OBJECTID":7,"NoticeNum":"2655955A","DateNotice":1779973980000,"Neighborhood":"Carrollton Ridge","BLOCKLOT":"0702 050","Address":"2113 WILKENS AVE"}}
```
Top 311 neighborhoods: Belair-Edison 7,237; Downtown 3,639; Canton 3,443; Coldstream Homestead Montebello 2,758; Brooklyn 2,547. Top SRTypes: `ECC-Information Request` 68,625; `SW-Rat Rubout Proactive` 33,585; `TRS-Parking Complaint` 15,139; `SW-Dirty Alley` 5,958; `HCD-Sanitation Property` 5,777. 75,051 requests have `"Neighborhood":null` (30.8%, mostly ECC phone information requests) — a known characteristic of the real feed. Vacant addresses: `1000 EASTERN AVE`, `309 S PULASKI ST`, `2310 W FAYETTE ST`, `2936 GREENMOUNT AVE`. Field names match the Open Baltimore ArcGIS schemas exactly. **These are real Open Baltimore records, bundled.** No mock or sample generator exists in the project. `cmd/fetch` re-downloads with pagination, duplicate-ID rejection, before/after count checks and SHA-256 (tested with a mock HTTP server). data_status = real_bundled.

## Build Results

```
$ time go build -trimpath -o bin/analyzer ./cmd/analyzer   -> exit 0, 4.1 s wall, no output
$ go build ./...                                          -> exit 0
$ go vet ./...                                            -> exit 0
```
Clean first try with the documented command (and via `sh run.sh`).

## Test/Vet/Race Results

```
$ go test ./...                      ok: cmd/fetch, internal/analysis, internal/ingest, internal/model, internal/report  (3.2 s)
$ go test -race -count=1 ./...       all ok, no DATA RACE (3.4 s)
$ sh verify.sh                       exit 0, 6.7 s: tests, vet, build, shasum OK x2, 1-worker vs 4-worker report.md and
                                     neighborhoods.csv byte-identical, and identical to bundled output/
```
Race-built analyzer on the full bundled data (`-workers 4`): exit 0, **0 DATA RACE** reports. Tests cover normalization, identity keys, period boundaries, closure eligibility, dedup across worker counts, invalid inputs, a 10,000-record no-deadlock stress fixture, join cohorts under concurrency, tied percentiles, medians, report sections/error propagation, CSV output, and downloader pagination/hash/duplicate/truncation handling.

## Run Results

```
$ /usr/bin/time -l sh run.sh -workers 4
  2.48 real  1.63 user  0.29 sys ; max RSS 37,666,816 bytes (36 MB) [includes go build]
  stderr: Wrote output/report.md and neighborhoods.csv (279 joined rows).
```
207 lines of Markdown to stdout (captured in `scratchpad/morton/run.txt`), byte-identical to the bundled `output/report.md`. Head and the five answers:

```
# Baltimore Neighborhood Service and Vacancy Analyzer
311 period: October 1 through December 31, 2025 (UTC). Vacancy data is a later open-notice snapshot ...
Accepted requests: 168569. Distinct vacancy properties with neighborhood: 11481.
Important coverage warning: 75051 of 243620 raw requests (30.81%) lack a neighborhood and are excluded ...
Analysis universe: 213 neighborhoods appearing in both datasets; 66 dataset-only neighborhoods excluded.

## Q1: Where are both request volume and vacancy counts concentrated?
  25 neighborhoods at/above the 75th percentile on both: BROADWAY EAST (2192 req / 719 vacant), CARROLLTON RIDGE (2175/743),
  SANDTOWN-WINCHESTER (2170/587), CENTRAL PARK HEIGHTS (2191/316), HARLEM PARK (1697/439), ... CHERRY HILL (1206/51)

## Q2: How does the mix of service needs differ?
  Low cohort (<=5 vacants): 75 hoods, 30,619 req; top types SW-RAT RUBOUT PROACTIVE 18.07%, TRS-PARKING COMPLAINT 10.59%,
  ECC-VEHICLE LOOK UP 4.44%, SW-BULK SCHEDULED-WEEKDAY 3.30%, SW-RECYCLING 2.55%
  High cohort (>26 vacants): 71 hoods, 79,202 req; SW-RAT RUBOUT PROACTIVE 20.99%, TRS-PARKING COMPLAINT 6.86%,
  SW-DIRTY ALLEY 5.25%, HCD-SANITATION PROPERTY 4.28%, SW-HGW 3.98%

## Q3: Does recorded closure time differ for comparable requests?
  TRS-PARKING COMPLAINT median 9.81 d (low) vs 22.16 d (high), +12.35 d; SW-DIRTY ALLEY 2.77 vs 2.26 (-0.52);
  HCD-SANITATION PROPERTY 4.29 vs 3.72; SW-HGW 8.29 vs 7.80; SW-CITY TRASH CAN 14.09 vs 13.68; others ~0 difference (10 types)

## Q4: How do monthly request volumes change by vacancy cohort?
  2025-10 Low 11,317 (150.9/hood) High 30,032 (423.0/hood); 2025-11 9,599 / 23,454; 2025-12 9,703 / 25,716
  ASCII bars; Oct->Dec change: Low -14.26%, High -14.37%

## Q5: Which neighborhoods should staff review first?
  BROADWAY EAST 97.89 (top type SW-HGW), CARROLLTON RIDGE 97.65 (SW-CLEANING), SANDTOWN-WINCHESTER 96.95 (SW-RAT RUBOUT PROACTIVE),
  CENTRAL PARK HEIGHTS 96.24, HARLEM PARK 96.01

## Data quality and limits
  311: raw=243620; accepted=168569; missing neighborhood=75051; unusable closure=3040. Vacancies: raw=11481; accepted=11481.
  + 66 dataset-only neighborhoods listed
```

## Concurrency Analysis

Correct, bounded and deterministic:
- Both datasets load concurrently (`cmd/analyzer/main.go:40-50`, goroutines + `WaitGroup`).
- Per file (`internal/ingest/ingest.go:40-158`): a scanner goroutine emits `job{seq,raw}` into `jobs` (cap `workers*4`) gated by a `credits` channel (cap `workers*8`) that bounds outstanding records including the reorder map; `workers` parser goroutines `json.Unmarshal` into `results`; a single dispatcher restores sequence order (`pending` map, `next` counter), validates, deduplicates, then hashes the neighborhood with FNV-1a to one of `workers` shard channels (cap 32); each shard goroutine exclusively owns its `map[string]*Stats` (no shared-map writes); maps are merged at the end.
- Join (`internal/analysis/analysis.go:21-65`): `workers` goroutines read the immutable maps from an unbuffered `jobs` channel and emit `Row`s; results sorted alphabetically so output is independent of scheduling.
- `runtime.GOMAXPROCS(workers)` is set explicitly. Downloader fetches both sources concurrently (`cmd/fetch/main.go:197-204`).
- Verified: `go test -race` clean, race-built binary clean on full data, 1-worker vs 4-worker outputs byte-identical. Report generation (`report.Write`) is sequential, which is appropriate.

## Memory/Resource Analysis

Documented in README "Memory and resource management" and evident in code: incremental line scanning with a 64 KiB initial / 4 MiB max buffer (`ingest.go:46-47`), bounded `jobs`/`results`/`credits`/shard channels for back-pressure, aggregate-before-join so no per-record joined rows are materialized, exact-dedup set and per-type duration slices acknowledged as the growing state, files/HTTP bodies closed with timeouts, output errors propagated to exit status. Honest about what is not constant-memory. Measured: 36 MB max RSS for a 68 MB input. No `runtime.MemStats` reporting (not required, but the other top submission has it).

## Architecture & Code Quality

Five packages with single responsibilities (`model` types/rules, `ingest` parse/validate/dedup/aggregate, `analysis` join/cohorts/percentiles, `report` presentation, `cmd/*` orchestration) — a Pipeline pattern realized through channels and stages. Code is gofmt'd, compact, idiomatic, with comments explaining the non-obvious (credit channel, owned maps, order restoration). Rules are explicit and tested (BLOCKLOT-then-address identity, midrank percentiles, tercile cutoffs). Weaknesses: `report.Write` is one 155-line function mixing analysis (Q1 thresholds, Q3 comparability filter) with formatting, so "presentation only" is slightly overstated; only one named pattern (Pipeline). The README and VERIFICATION.md contain sentences addressed to "the user" ("The user subsequently reported a successful native run…", "No Canvas submission has been performed. This package is a working project, not a guarantee of a grade", "review any course rules concerning outside assistance") that read as AI-assistant deliverable notes left in the submission; they do not contradict the code, which matches the documentation exactly.

## Stakeholder Questions & Value Assessment

Stakeholder: "a joint housing and city-services review team" (generic; DHCD/DPW/311 are implied by the request categories but no office is named per question).

| Q | Question | Cross-dataset mechanism |
|---|---|---|
| 1 | Where are request volume and vacancy counts both concentrated? | ≥75th midrank percentile on both measures |
| 2 | How does the request-type mix differ between low- and high-vacancy neighborhoods? | 311 types split by vacancy cohort |
| 3 | Do median closure times for the same request types differ between cohorts? | 311 durations split by vacancy cohort, ≥20 per side |
| 4 | How do monthly volumes change per cohort over the quarter? | 311 months split by vacancy cohort |
| 5 | Which five neighborhoods rank highest under a transparent 50/50 percentile score? | combined score of both datasets |

All five use the join, though Q2-Q4 are 311 measures stratified by a vacancy-derived cohort rather than two-sided joins. The results are real and partly actionable: the Q1/Q5 list (Broadway East, Carrollton Ridge, Sandtown-Winchester, Central Park Heights, Harlem Park) is credible, and Q3's finding that parking complaints take 22 days to close in high-vacancy neighborhoods versus 10 days in low-vacancy ones is a specific lead for DOT/Transportation. Q4 (both cohorts -14% Oct->Dec) is thin. Caveats (temporal mismatch, 30.8% missing neighborhoods, no denominators, closure ≠ resolution) are stated clearly. Visualization is minimal: Markdown tables and a single ASCII bar block in Q4; no HTML.

## Issues

- Only one ASCII chart; otherwise tables — visualization is the weakest rubric element.
- Stakeholders are a generic "joint housing and city-services review team"; no office named per question.
- Q2-Q4 are single-dataset measures stratified by the other dataset, not two-sided joins; Q4 is shallow.
- `report.Write` mixes analysis logic with formatting; single long function.
- README/VERIFICATION.md contain leftover assistant-style notes addressed to "the user" (e.g. "No Canvas submission has been performed", "review any course rules concerning outside assistance").
- 30.8% of 311 rows lack a neighborhood and are excluded (documented; inherent to the feed).

## Score Table

| # | Criterion | Pts | Score | Justification |
|---|---|---|---|---|
| 1 | build | 15 | 15 | Go; `go build -trimpath` / `sh run.sh` clean first try; vet clean. |
| 2 | real_data | 15 | 15 | Two real Open Baltimore extracts bundled (243,620 + 11,481 raw rows), SHA-256 verified, manifest provenance, no mock generator. |
| 3 | concurrency | 15 | 15 | Parsing pools, credit-bounded reorder, FNV-sharded owned-map aggregation, parallel join; race-clean; deterministic across worker counts. |
| 4 | memory | 10 | 9 | Streaming scanner, bounded channels, aggregate-before-join, documented growth points and limits; no runtime MemStats reporting. |
| 5 | architecture | 15 | 14 | Clean 5-package Pipeline, 16 tests incl. stress and determinism; report.Write mixes analysis with formatting, one named pattern. |
| 6 | stakeholder | 15 | 11 | Five questions all using the join with honest caveats and real actionable results (Q1/Q3/Q5), but generic stakeholder, Q2-Q4 are stratified single-dataset measures, and only one ASCII chart. |
| 7 | runs_results | 10 | 10 | Runs end to end in 2.5 s / 36 MB on bundled real data, prints all five answers, byte-identical to shipped output. |
| 8 | readme | 5 | 4 | Explicit, accurate macOS build/run/verify instructions and scripts; marred by leftover assistant notes addressed to "the user". |
| | **Total** | **100** | **93** | **A** |

## Final Grade Rationale

This submission meets every hard requirement of the specification: approved language, two real Open Baltimore datasets physically bundled with checksums and a reproducible downloader, native goroutine/channel concurrency in parsing, aggregation and joining that is both race-free and provably deterministic, explicit memory bounding, a clean multi-package pipeline with 16 tests, and a README whose instructions work exactly as written. Points were withheld only for the modest presentation (one ASCII chart, Markdown tables, no HTML), a generic rather than named stakeholder with three questions that are stratified 311 measures rather than two-sided joins, and documentation residue from an AI assistant. 93/100, A.
