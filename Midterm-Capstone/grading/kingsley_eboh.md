# Kingsley Eboh — COSC 352 Midterm Capstone Grading Report

**Summary: 78/100 (C)** — Go, builds and vets clean, and is the only submission that **fetches real Open Baltimore data at runtime**: the default run paged 706,938 2026 311 requests (354 GeoJSON pages) plus the 12,647-record Vacant Building Rehabs snapshot from ArcGIS, in 6 min 47 s with only 53 MB RSS, and wrote Markdown/CSV/JSON reports. Bundled `data/samples/*.geojson` are real 250-feature excerpts (not fake) but too small to analyse. Weaknesses: no tests, concurrency limited to the parse stage, two of five questions are single-dataset, and a nonsensical "requests per 100 rehab locations" ratio.

## Inventory

Path: `submissions/kingsley_eboh/MidTermCapstone/`

| File | LOC | Notes |
|---|---|---|
| main.go | 154 | flags, Source definitions (live / `-offline` / `-from-raw`), terminal dashboard |
| pipeline.go | 256 | concurrent fetchers (paged ArcGIS query + snapshot download), bounded parser worker pool, single collector |
| analysis.go | 335 | accumulator, Spearman, z-score joint index, enrichment, CSV/JSON writers |
| report.go | 96 | Markdown report with tables and ASCII bars |
| run-macos.sh | 5 | `go build -o civicpulse . && exec ./civicpulse "$@"` |
| README.md | 5.5 KB | build/run/verify, data sources, architecture, limits |
| data/samples/311_sample.geojson | 220 KB, 250 features | **real** excerpt (RowID 2.., SRRecordID 500VO...) |
| data/samples/vacant_rehabs_sample.geojson | 111 KB, 250 features | **real** excerpt (`ob_Rehabbed_VBN_Pt_flat`) |
| data/samples/README.md | — | provenance of excerpts (retrieved 2026-10-11) |
| reports/analysis.md, neighborhoods.csv, summary.json | 7 KB / 13 KB / 86 KB | outputs of a prior real run (2026-10-11, 692,663 requests) |

Total Go LOC: 841. Go 1.22, stdlib only. **No tests.** Run script: `run-macos.sh`.

## Datasets & Data Realness

Declared and used:
1. **311 Customer Service Requests 2026** — ArcGIS FeatureServer `311_Customer_Service_Requests_current/FeatureServer/0/query`, paged 2,000 records at a time, `where=Neighborhood IS NOT NULL`, `f=geojson` (`main.go:16`, `pipeline.go:160-223`).
2. **Vacant Building Rehabs** — `data.baltimorecity.gov/api/download/v1/items/4db6d1e54e714a3e8125990a09d4623d/geojson?layers=2` (`main.go:17`).

Bundled sample heads (genuine schemas and records):

```
311_sample.geojson:  {"RowID":2,"SRRecordID":"500VO00000ta054YAA","ServiceRequestNum":"26-00004287","SRType":"SW-Bulk Scheduled-Saturday","CreatedDate":1767451496000,"SRStatus":"Closed",...,"Address":"3954 DOLFIELD AVE, Baltimore City, 21215","Neighborhood":"Dolfield","CouncilDistrict":"6","PoliceDistrict":"Northwestern",...}
vacant_rehabs_sample.geojson: "name":"ob_Rehabbed_VBN_Pt_flat" ... {"OBJECTID":7,"Block":"0702","Lot":"050","PermitNum":"BUSE-25-004083","VBN":"Y","DateIssue":"2025-12-22T00:00:00Z","ExistingUse":"Single Family Dwelling","HousingMarketTypology2017":"J","Council_District":"9","Neighborhood":"Carrollton Ridge","Address":"2113 WILKENS AVE"}
```

These match John Nwokelo's and Jordan Allen's independently obtained 311 records (same SRRecordID `500VO00000ta054YAA` for RowID 2), confirming realness. Samples are 250 features each — a format demo; `-offline` is **not** the default. The default path is the live fetch, and the program saved 603 MB of raw pages to `data/raw/` during grading (gitignored in the submission).

Row counts: 311_sample.geojson 250 features; vacant_rehabs_sample.geojson 250 features; live fetch: 706,938 311 requests (354 pages) and 12,647 rehab locations.

**data_status: real_fetched_runtime** (real excerpts bundled but tiny; full data fetched by the program).

## Build Results

```
$ cd submissions/kingsley_eboh/MidTermCapstone
$ go build -o civicpulse .        # exit 0
$ go vet ./...                    # exit 0, clean
$ go build -race -o cp-race .     # exit 0
$ chmod +x run-macos.sh           # required, and documented in README
```

Clean first try.

## Test / Vet / Race Results

```
$ go test ./...
?   	baltimore-civic-pulse	[no test files]
```

**No tests.** Race binary runs: `-offline` -> 0 DATA RACE; `-from-raw data/raw` (replaying the 603 MB of real pages fetched during grading, 706,938 records) -> exit 0, **0 DATA RACE**.

## Run Results

**(a) Live default path** (as README: `./run-macos.sh -workers 4`), network access used:

- Wall **407.1 s** (6 min 47 s; 34.6 s user) — within the 10-minute limit; max RSS **53.2 MB**.
- Wrote `data/raw/requests_page_00001..00354.geojson` + `vacant_snapshot.geojson` (603 MB), `reports/analysis.md`, `reports/neighborhoods.csv`, `reports/summary.json`.
- Captured: `scratchpad/kingsley/live_stdout.txt`, `live_stderr.txt`, `work/reports/`.

Terminal output:

```
BALTIMORE CIVIC PULSE — 311 × VACANT BUILDING REHABS
Requests: 706,938 | Rehab locations: 12,647 | Matched neighborhoods: 234 | Workers: 4 | Elapsed: 6m45.983s

Top neighborhoods by joint indicator (311 requests + rehab locations)
NEIGHBORHOOD                          311     REHABS NON-CLOSED
Belair-Edison                      25,248        373        684
Coldstream Homestead Montebel…     16,760        367        428
Broadway East                       9,641        445        292
McElderry Park                      9,047        382        261
Brooklyn                           12,169        253        398
Oliver                              7,648        337        250
Central Park Heights               10,184        265        274
Washington Village/Pigtown          9,734        248        299
Sandtown-Winchester                 9,167        221        304
Canton                             13,281         82        789

Neighborhood rank correlation (Spearman): strong positive association
Interpret counts as reported activity, not a measure of need or causation.

Wrote reports/analysis.md and reports/neighborhoods.csv
```

Five answers from `reports/analysis.md` (live run):

```
Executive summary: 706,938 311 requests with a neighborhood label and 12,647 vacant-building rehab locations; 234 matched neighborhoods; Spearman = 0.770 (strong positive association).

1. Strongest combined signal (z-score sum of 311 + rehabs):
| Belair-Edison | 25,248 | 373 | 684 | 6768.9 |
| Coldstream Homestead Montebello | 16,760 | 367 | 428 | 4566.8 |
| Broadway East | 9,641 | 445 | 292 | 2166.5 |
| McElderry Park | 9,047 | 382 | 261 | 2368.3 |
| Brooklyn | 12,169 | 253 | 398 | 4809.9 |

2. Largest concentrations of rehab locations: Broadway East 445, McElderry Park 382, Belair-Edison 373, Coldstream 367, Oliver 337.

3. Most non-closed 311 requests: Downtown 863, Canton 789, Belair-Edison 684, Frankford 501, Coldstream 428.

4. 311 types enriched in high-rehab neighborhoods (cutoff 69 rehab locations):
| SW-Boarding | 3,070 | 0.9% | 3.82x |
| SW-Cleaning | 4,434 | 1.3% | 3.72x |
| HCD-Vacant Building Squatter Encounter (Internal BCFD USE Only) | 204 | 0.1% | 3.50x |
| HCD-Vacant Building | 4,832 | 1.4% | 2.66x |
| SW-Dirty Alley | 17,884 | 5.2% | 2.52x |

5. Spearman rank correlation = 0.770 (strong positive association) over 234 neighborhoods.

Visual summaries: two ASCII bar charts (top-10 311 by neighborhood; top-10 rehab locations).
```

**(b) `-offline`** (bundled 250+250 excerpts): 0.24 s, 16.7 MB RSS; 17 matched neighborhoods, "moderate negative association" — demonstrates the pipeline only.

**(c) `-from-raw data/raw`** replay of the fetched pages: 8.7 s, 45 MB RSS, identical totals to the live run.

## Concurrency Analysis

- `pipeline.go:38-47` — one fetcher goroutine per source (`fetchWG`), so the 354-page 311 download and the rehab snapshot download overlap; `close(jobs)` after both finish.
- `pipeline.go:34-35, 48-61` — bounded `jobs`/`parsed` channels (`workers*8`), `workers` parser goroutines (`parseFeature` unmarshals each feature's properties), WaitGroup closes `parsed`.
- `pipeline.go:62-65` — single collector goroutine (main) does all aggregation into `accumulator` (no sharing, no locks).
- `context.WithTimeout(20 min)` and `select` on `ctx.Done()` in every send (`pipeline.go:110-114, 152-156, 213-217`).
- Aggregation (`accumulator.add`) and analysis (`finish`) are sequential. Join is a map by normalised neighborhood name.
- Race detector clean on offline and on the 706k-record replay.

Level: concurrency genuinely used for fetch + parse with correct sync; aggregate/join sequential. Wall time is network-bound (34 s CPU over 407 s). Scored 10/15.

## Memory / Resource Analysis

- Paged fetching (2,000 records/page) with `io.LimitReader(resp.Body, 64<<20)` and a 90 s HTTP timeout (`pipeline.go:122, 183`); each page is decoded, written to disk, and its features handed off — so memory stays around one page plus channel buffers.
- Each feature copied to an owned `json.RawMessage` before sending (`pipeline.go:109, 151, 212`) so the page buffer can be released.
- Records are reduced to four strings (`record{Source, Neighborhood, RequestType, Status}`) and then to counters per neighborhood; rows are not retained.
- Measured: **53 MB RSS while processing 706,938 records**, 45 MB on replay — the best memory profile of the four.
- README's "Memory and resource management" bullets match the code. No `runtime.MemStats` reporting; `-from-raw`/`-offline` replay the saved raw pages one file at a time (`os.ReadFile` per page is fine at 2,000 records).

Scored 10/10.

## Architecture & Code Quality

Single `package main`, four files: fetch/parse (`pipeline.go`), aggregation + stats + CSV/JSON writers (`analysis.go`), Markdown (`report.go`), CLI + dashboard + `Source` type (`main.go`). Producer/worker/collector pipeline is clear and the `Source` struct cleanly switches between live, offline and replay modes. No interfaces or named Strategy/Factory/Builder; README claims only "pipeline" and "source configuration separates providers from ingestion", which is accurate. **No tests.** Dead code: `cleanNeighborhood` is the identity function (`main.go:155`). `analysis.go` mixes statistics with file output. Property lookup is a linear scan per key per feature (`pipeline.go:236`) — fine here. Generics used (`firstN[T]`). Formatting is dense (one-line struct literals, 300-char `Fprintf`). Code is consistent with README; nothing looks pasted from elsewhere.

## Stakeholder Questions & Value Assessment

Stakeholders named: Mayor's Office and Department of Housing and Community Development.

| # | Question | Cross-dataset? |
|---|---|---|
| 1 | Strongest combined signal of 311 volume and rehab locations (z-score sum) | yes |
| 2 | Largest concentrations of rehab locations | no (rehabs only) |
| 3 | Neighborhoods with most non-closed 311 requests | no (311 only) |
| 4 | 311 types relatively more common in high-rehab neighborhoods (enrichment) | yes |
| 5 | Do the two indicators rank neighborhoods similarly? (Spearman) | yes, but a one-number restatement of the executive summary |

Q4 is the genuinely useful result: SW-Boarding (3.8x), SW-Cleaning (3.7x), HCD-Vacant Building (2.7x) and Dirty Alley (2.5x) are enriched where rehab permits cluster — a plausible DHCD signal that rehab activity and blight complaints co-locate in East Baltimore (Broadway East, McElderry Park, Oliver). Q1 identifies the same neighborhoods. Weaknesses: Q2/Q3 are single-dataset top-5 lists; the "Requests per 100 rehab locations" column is meaningless where rehabs are few (Downtown 41,781.8); no per-capita or per-housing-unit normalisation although the rehab dataset's typology field was available; the two datasets cover different periods (2026 vs multi-year inventory), acknowledged. Visuals are Markdown tables plus two ASCII bar charts and a terminal table.

## Issues

1. No tests (`go test` reports no test files).
2. Concurrency confined to fetch/parse; aggregation and analysis are sequential.
3. Q2 and Q3 are single-dataset; Q5 duplicates the executive-summary statistic; "requests per 100 rehab locations" is not a meaningful ratio.
4. Default live run takes ~7 minutes and downloads ~600 MB (network-bound); README does not warn about duration/size. Bundled excerpts (250 features) are too small to analyse.
5. Dead code (`cleanNeighborhood` identity), dense formatting, stats and file I/O mixed in `analysis.go`.
6. `run-macos.sh` is not executable in the archive (README does say `chmod +x`).
7. `reports/` in the submission are outputs from the student's 2026-10-11 run; they are regenerated (and overwritten) on each run.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build -o civicpulse .` clean first try; vet clean. |
| real_data | 15 | 10 | Two real Open Baltimore datasets fetched by the program at runtime (706,938 + 12,647 records); bundled excerpts are real but only 250 features each. |
| concurrency | 15 | 10 | Concurrent fetchers plus bounded parser worker pool with correct sync and context cancellation; aggregation and analysis sequential. |
| memory | 10 | 10 | Paged streaming fetch, 64 MiB page cap, owned RawMessage copies, reduce-to-counters; documented in README; 53 MB RSS on 707k records. |
| architecture | 15 | 9 | Clear producer/worker/collector separation in four files, but single package, no interfaces/patterns beyond pipeline, no tests, dead code. |
| stakeholder | 15 | 9 | Named stakeholders and a strong Q4 enrichment result; but Q2/Q3 single-dataset, Q5 redundant, one meaningless ratio, no normalisation. |
| runs_results | 10 | 10 | Live default run completed in 6m47s (within limit) with sensible, cross-validated output and all report files; offline and replay modes also work. |
| readme | 5 | 5 | Explicit macOS compile/execute/verify steps, run script, modes documented; matches reality. |
| **Total** | **100** | **78** | **C** |

## Final Grade Rationale

Kingsley's program is the only one of the four that actually acquires the full real datasets itself, and it does so with a genuinely streaming, memory-frugal pipeline (53 MB for 707k records) that is race-clean — full marks for build, memory, run and README. The rubric caps runtime-only data at 10, concurrency is limited to the fetch/parse stage, and the architecture, while clear, is a single untested package with no realized design-pattern abstractions. The analysis has one strong cross-dataset finding (vacancy-related 311 types enriched in high-rehab neighborhoods) but two of the five questions are single-dataset lists. Total 78, C.
