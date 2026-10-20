# Jessica Madu — COSC 352 Midterm Capstone Grading Report

**Summary: 77/100 (C)** — Go, builds and runs cleanly, well-engineered streaming pipeline with per-worker aggregation; but the only data shipped is a synthetic generator's output and the synthetic path is the default. A `report_real_data.txt` shows she did run against the real files locally, but no real data is in the submission and the program does not fetch any.

## Inventory

Path: `submissions/jessica_madu/baltimore-intel/`

| File | LOC | Notes |
|---|---|---|
| main.go | 114 | flags, concurrent start of both pipelines, report wiring, MemStats |
| datasets.go | 152 | `Dataset` Strategy interface, `spec` table-driven impl, factories, header binding, date layouts |
| ingest.go | 128 | reader -> bounded jobs channel -> N workers -> merge |
| model.go | 97 | `Event` value type, `Agg` per-worker aggregate, `Merge` |
| analysis.go | 279 | `Question` Strategy interface, five question types |
| render.go | 60 | topN, ASCII bars, truncation, headings |
| main_test.go | 44 | 3 tests (ingest crime, 311 open status, missing columns) |
| scripts/gen_sample.py | 27 | **synthetic data generator** |
| README.md | — | detailed macOS build/run/verify |
| report_real_data.txt | — | output of a prior run against real downloads (not reproducible here) |
| data/sample/crime_sample.csv | 6,002 lines (392 KB) | synthetic |
| data/sample/311_sample.csv | 8,001 lines (603 KB) | synthetic |

Total Go LOC: 874 (830 non-test). Language: Go 1.21, stdlib only. Tests: yes. Run script: none (plain `go build`). Total source files: 7 Go + 1 Python.

## Datasets & Data Realness

Declared datasets: Open Baltimore **Part 1 Crime Data** (CSV) and **311 Customer Service Requests** (CSV).

What is actually shipped: two CSVs produced by `scripts/gen_sample.py` (random.seed, 20 hand-listed neighborhoods, weighted random crime/SR types). Evidence they are synthetic:

```
RowID,CrimeDateTime,Location,Neighborhood,Description,Weapon,District
0,2024/07/22 19:21:00+00,X,Brooklyn,AUTO THEFT,NA,CENTRAL
1,2024/06/13 12:10:00+00,X,Park Heights,COMMON ASSAULT,NA,CENTRAL

ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,Agency,Neighborhood
0,SW-Bulk Pickup,Phone,11/18/2025 12:39:00 AM,Closed,,DPW,Roland Park
1,SW-Bulk Pickup,Phone,12/11/2025 03:54:00 PM,Closed,,DPW,Edmondson Village
```

- `Location` is always `X`, `District` always `CENTRAL`, `Weapon` always `NA`, sequential integer IDs, only 20 neighborhoods, no CCNumber/SRRecordID/lat-lon. The real Part 1 schema has CCNumber, Inside_Outside, Post, Latitude/Longitude, etc.; the real 311 schema has SRRecordID, CloseDate, Address, ZipCode, lat/lon. These are a reduced imitation, not excerpts.
- README states plainly: "`data/sample/` contains **synthetic** files".
- The generator is **the default path**: `-crime` defaults to `data/sample/crime_sample.csv`, `-requests` to `data/sample/311_sample.csv`.
- No real data anywhere in the submission folder (`find -iname '*.csv'` returns only the two synthetic files). Program has no download step.
- `report_real_data.txt` (generated 2026-10-14) is plausible real output (43,803 2024 crimes / 656,827 2025 requests, 288/279 neighborhoods, `SW-RAT RUBOUT PROACTIVE` 17.9% citywide) and shows the code works on the real files, but it is not verifiable here.

Row counts (`wc -l`): crime_sample.csv 6,002; 311_sample.csv 8,001.

**data_status: sample_only (simulated)**.

## Build Results

```
$ cd submissions/jessica_madu/baltimore-intel
$ go build -o baltimore-intel .      # exit 0, no output
$ go vet ./...                       # exit 0, clean
$ go build -race -o bi-race .        # exit 0
```

Clean first-try build with the documented command.

## Test / Vet / Race Results

```
$ go test -race ./...
ok  	baltimore-intel	1.227s
```

3 tests pass under the race detector. Running the `-race` binary on the sample data: exit 0, **0 DATA RACE reports**. README's 1-vs-8-worker determinism check: outputs identical ("SAME").

## Run Results

Command (README default path, sample data): `./baltimore-intel -out report.txt`

- Wall: **0.21 s** real; max RSS **17.6 MB** (`/usr/bin/time -l`). Program-reported: heap in use 1.0 MB, 4 GC cycles, 10 workers/dataset.
- Captured: `scratchpad/jessica/run_stdout.txt`, `run_stderr.txt`, `report.txt`.

Output (abridged; full in scratchpad):

```
BALTIMORE CIVIC INTELLIGENCE REPORT
Crime dataset : 6000     rows used (2024), 1 skipped, 20 neighborhoods  (7ms)
311 dataset   : 8000     rows used (2025), 0 skipped, 20 neighborhoods  (8ms)
Join (neighborhood): 20 neighborhoods appear in both datasets

Q1. Mayor's Office: Which neighborhoods carry the most crime, and what does their 311 picture look like?
Neighborhood                 Crime  Viol%     311  Open%  Crime volume
SANDTOWN-WINCHESTER            684  21.9%     690  30.3%  ██████████████████████████████
DOWNTOWN                       580  24.7%     644  30.7%  █████████████████████████
HARLEM PARK                    526  23.2%     584  31.2%  ███████████████████████
...
Takeaway: SANDTOWN-WINCHESTER has the most incidents (684).

Q2. Public Works: Where is crime high but 311 engagement low?
SANDTOWN-WINCHESTER            684     690        0.99  ██████████████████████████████
HARLEM PARK                    526     584        0.90  ███████████████████████████
...
Takeaway: SANDTOWN-WINCHESTER has 1.0 crimes per 311 request; consider outreach to boost 311 awareness there.

Q3. City Planning: Do neighborhoods with more crime also generate more 311 requests?
Pearson r across 20 neighborhoods = 0.992 (strong positive relationship)
Q1 (lowest crime)           105.0      230.0  ███████████
Q4 (highest crime)          540.4      607.4  ██████████████████████████████

Q4. Transportation/DPW: Which 311 request types are over-represented in the highest-crime neighborhoods?
SW-DIRTY ALLEY                   1115    21.3%    20.4%  1.04x
TRM-POTHOLE                       746    14.3%    14.2%  1.00x
HCD-RAT RODENT COMPLAINT          789    15.1%    15.5%  0.98x
Takeaway: 'SW-DIRTY ALLEY' requests are 1.04x more common in high-crime areas than citywide.

Q5. Operations: When do crime and 311 demand peak (by month and time of day)?
Jan        546 ██████████████████████     716 █████████████████████
... (flat; synthetic uniform distribution)
Overnight 00-05      25.3%   24.5%   Morning 25.3% 25.0%   Afternoon 25.1% 25.6%   Evening 24.3% 24.9%

Runtime: 10 CPUs, 10 workers/dataset, total 9ms, heap in use 1.0 MB, 4 GC cycles
```

The five answers are produced, but because the data is uniformly random, the "findings" are artefacts (r = 0.992 because the generator uses the same weights for both files; flat seasonality; all lifts ~1.0).

## Concurrency Analysis

- `main.go:22-30, 55-57` — `start()` launches one ingest pipeline per dataset in its own goroutine, returning a result channel (future style); both datasets ingested concurrently.
- `ingest.go:71-92` — bounded `jobs` channel (`workers*2` batches of 2,000 rows), `workers` parser goroutines each owning a private `*Agg`, `sync.WaitGroup` closes `results`.
- `ingest.go:94-116` — reader goroutine streams CSV, batches, closes `jobs`.
- `ingest.go:118-121` — single merger folds per-worker Aggs (`model.go:72-91`).
- `readErr` is written by the reader before `close(jobs)` and read after `results` closes; the happens-before chain is sound (commented at `ingest.go:122`).
- Questions run sequentially (`main.go:91-94`); join is a map intersection (`main.go:83-88`).
- Race detector: clean on tests and on a full sample run.

Level: **full for parse/aggregate** (worker pool, confinement, merge), join/analysis sequential but trivial. Scored 13/15.

## Memory / Resource Analysis

- CSV streamed via `bufio.NewReaderSize(f, 1<<20)` + `encoding/csv` (`ingest.go:57`); rows never all in memory.
- Back-pressure via bounded channel (`ingest.go:71`); batch slices become garbage after processing.
- Aggregates are O(neighborhoods x categories) (`model.go:18-24` comments); `Event` is a value type.
- `runtime.MemStats` read and reported (`main.go:67-69, 96-98`).
- README section "Memory & thread safety" matches the code. Verified 17.6 MB RSS on the sample; the prior real-data report shows 10.3 MB heap on 1.3 M rows, consistent with the design.

## Architecture & Code Quality

Single `package main`, seven cohesive files. Patterns actually realized: **Pipeline** (`Ingest`), **Strategy** (`Dataset` and `Question` interfaces), **Factory** (`NewCrimeDataset`, `New311Dataset`). Header binding is table-driven with alias lists and BOM handling; multiple date layouts. Clear naming, concise comments that match the code, tests cover ingestion/binding. Minor: everything in one package; Q5 month folding by re-parsing "YYYY-MM" strings is slightly roundabout; no tests for the question logic. Code reads as hand-written and consistent with the README; nothing contradictory.

## Stakeholder Questions & Value Assessment

| # | Stakeholder | Question | Cross-dataset? |
|---|---|---|---|
| 1 | Mayor's Office | Top-crime neighborhoods and their 311 volume / open % | yes (join) |
| 2 | Public Works | High crime but low 311 engagement (crime/311 ratio) | yes |
| 3 | City Planning | Pearson r + crime-quartile table of 311 volume | yes |
| 4 | Transportation/DPW | 311 types over-represented in top-crime areas (lift) | yes |
| 5 | Operations | Month and time-of-day peaks, side by side | both datasets, compared not joined |

Questions are well motivated and distinct; the Q2 "service gap" ratio and Q4 lift are genuinely useful framings. Output is formatted tables + ASCII bars. Caveats: crime 2024 vs 311 2025 (acknowledged), raw counts not per-capita, and — decisive here — the shipped run answers them on fake data, so a stakeholder learns nothing from the submitted artefact; `report_real_data.txt` does contain actionable-looking results (e.g., Inner Harbor 0.51 crimes per 311 request, curb repair 2.55x lift) but cannot be reproduced from the submission.

## Issues

1. No real Open Baltimore data present; only generator output. Program needs user to download ~75 MB + ~380 MB CSVs.
2. Synthetic sample is the default flag path (rubric deduction).
3. `report_real_data.txt` cannot be verified; real-data correctness is asserted, not demonstrated.
4. Single package; question logic untested.
5. Q3 r = 0.992 and flat Q5 on synthetic data illustrate that the shipped run is not informative.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build` clean first try; vet clean. |
| real_data | 15 | 2 | Only synthetic generator output shipped (5), minus 3 because the synthetic files are the default path; no fetch. |
| concurrency | 15 | 13 | Per-dataset worker pools with bounded channels, per-worker aggregates merged, both datasets concurrent, race-clean; join/questions sequential. |
| memory | 10 | 10 | Streaming CSV, bounded back-pressure, O(groups) aggregates, MemStats reported, documented and evident. |
| architecture | 15 | 12 | Clear Pipeline/Strategy/Factory in well-separated files with tests, but single package. |
| stakeholder | 15 | 13 | Five distinct, mostly joined questions for named stakeholders with tables and bars; Q5 is side-by-side rather than joined; not per-capita. |
| runs_results | 10 | 7 | Runs to completion and prints all five answers, but only on synthetic data so results are not meaningful. |
| readme | 5 | 5 | Explicit macOS build/run/verify steps that all work as written. |
| **Total** | **100** | **77** | **C** |

## Final Grade Rationale

Engineering quality is high: the pipeline, confinement-based thread safety, streaming design and documentation are all solid, and everything builds, vets, tests and runs race-free on the first try. The grade is pulled down by the data requirement, which the rubric weights heavily: the submission contains no Open Baltimore data at all, the synthetic generator's output is the default input, and the one artefact showing real results is a text file that cannot be reproduced from what was submitted. With the real CSVs bundled (or fetched), this would be a solid A/B project.
