# Bishant Basnet — COSC 352 Midterm Capstone Grading Report

**Summary: 85/100 (B).** Go, well-architected multi-package engine (`cmd/engine`, `internal/{ingest,analysis,report,model}`) with generic worker-pool ingestion, lock-free map-reduce join, concurrent questions, string interning, tests, and a Builder/Strategy/Factory report layer. The major weakness is **data: the `data/` directory is empty** (only `.gitkeep`). The README tells the user to download two Open Baltimore CSVs manually; with none present, `run.sh` silently falls back to a bundled **synthetic data generator**, so the default out-of-the-box run analyzes random numbers. The student's saved `report.md` shows they did run it on real 2024 data (1,081,868 311 rows), and I confirmed the engine works on real Open Baltimore files borrowed from other submissions, but no real data was shipped and no fetch script is provided.

## Inventory

Submission root: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/bishant_basnet/baltimore-engine/`

| File | LOC | Purpose |
|---|---:|---|
| `cmd/engine/main.go` | 137 | CLI flags, concurrent ingest of both files, join, RunAll, report build/render |
| `cmd/gensample/main.go` | 140 | **synthetic** CSV generator (40 fake neighborhoods, random dates) |
| `internal/ingest/loader.go` | 135 | generic `Load[T]` pipeline: reader -> batches -> N parser workers -> collector |
| `internal/ingest/parsers.go` | 82 | `RowParser[T]` Strategy; `CrimeParser`, `ServiceRequestParser` |
| `internal/ingest/schema.go` | 71 | alias-based, case/punctuation-insensitive column resolution |
| `internal/ingest/timeparse.go` | 88 | 15 timestamp layouts, atomic last-layout cache, tz conversion |
| `internal/ingest/interner.go` | 36 | `sync.Map` string interner + name normalization |
| `internal/ingest/ingest_test.go` | 62 | tests: time formats, schema aliases, concurrent load |
| `internal/model/model.go` | 28 | `CrimeRecord`, `ServiceRequest` |
| `internal/analysis/context.go` | 161 | `BuildContext` lock-free map-reduce join; `IsBlight` |
| `internal/analysis/questions.go` | 348 | `Question` Strategy, `NewQuestion` Factory, Q1–Q5, `RunAll` |
| `internal/analysis/stats.go` | 88 | Pearson, percentile, quartiles |
| `internal/analysis/analysis_test.go` | 45 | tests: Pearson, blight, parallel==serial for 1/3/8 workers |
| `internal/report/report.go` | 72 | Report model + `Builder` |
| `internal/report/render.go` | 232 | `Renderer` Strategy, `NewRenderer` Factory, text + markdown, ASCII charts/tables |
| **Total** | **1725** | 15 Go files (13 source + 2 test) |

Other: `README.md` (detailed), `run.sh`, `go.mod` (`go 1.21`, no deps), `report.md` (saved output from a real-data run), `data/.gitkeep`. **No CSV data files of any kind.** Submission size 136 KB.

## Datasets & Data Realness

- README documents: Open Baltimore **NIBRS Group A / Part 1 Crime Data** -> `data/crime.csv`, and **311 Customer Service Requests 2024** -> `data/311.csv`, to be downloaded manually from data.baltimorecity.gov. No fetch script, no URLs to the specific items.
- **Data present in the submission: none.** `ls data/` -> `.gitkeep` only.
- `run.sh` lines 18-24: if `data/crime.csv`/`data/311.csv` are missing it runs `./bin/gensample -out data/sample` and analyzes `crime_sample.csv` (80,000 rows) + `311_sample.csv` (150,000 rows). These are random: 40 hard-coded neighborhood names, addresses like `679 SAMPLE ST`, request numbers `24-00000001`, synthetic hour/risk weights. **The synthetic path is what runs by default on a fresh submission.** The generator prints a clear warning and the README says the numbers are not real.
- Evidence the student used real data: saved `report.md` lists `Part 1 Crime Data 71.7 MB 278,239 rows` and `311 Customer Service Requests 405.0 MB 1,081,868 rows` (the real 2024 311 export has exactly 1,081,868 rows — matches Daniel Dada's bundled `311_2024.csv`), 290 neighborhoods, 279 joined, with real names (Downtown, Belair-Edison, Inner Harbor, Canton Industrial Area).
- Verification on real data (grader-supplied, not part of the submission): `CRIME=<Eze's crime_nibrs.csv> REQUESTS=<Basnet A.'s requests_2025.csv> ./run.sh` ran in 2.0 s, 276,007 + 931,324 rows read, 279 neighborhoods joined, sensible results (see below). So the engine's schema aliasing and parsing work against genuine Open Baltimore exports.
- Data status: **sample_only** (synthetic generator present, real data must be downloaded by user).

## Build Results

Run in a scratchpad copy of the project so the student's folder was not modified (run.sh writes `data/sample/`, `bin/`, `report.md`).
```
$ go build -o bin/baltimore-engine ./cmd/engine   # exit 0, 0.4 s
$ go build -o bin/gensample ./cmd/gensample        # exit 0
$ go vet ./...                                     # exit 0, clean
```
Clean build first try.

## Test/Vet/Race Results

```
$ go test ./...
ok  github.com/bishant/baltimore-engine/internal/analysis 0.395s
ok  github.com/bishant/baltimore-engine/internal/ingest   0.214s
$ go test -race ./...
ok  internal/analysis 1.422s / ok internal/ingest 1.228s      (0 races)
$ go build -race -o bin/engine_race ./cmd/engine && ./bin/engine_race -crime data/sample/crime_sample.csv -requests data/sample/311_sample.csv
exit 0, 2.8 s, 0 DATA RACE reports
```
Tests pass; race detector clean in tests and in a full program run on the sample.

## Run Results

**Documented path (`./run.sh`), real data absent -> synthetic fallback:**
```
$ /usr/bin/time -l ./run.sh
        1.36 real   0.87 user   0.50 sys      95191040 max RSS (95 MB, includes generating 27 MB of CSV)
```
Captured output (scratchpad `bishant/run_sample.txt`), first 70 lines:
```
==> Building (go version go1.24.0 darwin/arm64)
==> Real Open Baltimore files not found at data/crime.csv and data/311.csv
    Generating a SYNTHETIC sample so the engine can be verified (see README).
Synthetic sample written to data/sample (80000 crime rows, 150000 311 rows)
==> Running engine on data/sample/crime_sample.csv + data/sample/311_sample.csv
Baltimore Municipal Intelligence Report
=======================================
Generated: Mon Oct 19 2026 15:45 EDT
Sources: Open Baltimore - Part 1 Crime Data; 311 Customer Service Requests (joined on Neighborhood)
Engine: Go go1.24.0, 10 worker goroutines per dataset, 10 CPUs


Ingestion & Runtime Summary
---------------------------

  +-------------------------------+-----------+-----------+-----------+---------+-----------+-------+
  | Dataset                       | File size | Rows read | Rows kept | Dropped | Malformed | Time  |
  +-------------------------------+-----------+-----------+-----------+---------+-----------+-------+
  | Part 1 Crime Data             | 6.8 MB    | 80,000    | 79,197    | 803     | 0         | 69ms  |
  | 311 Customer Service Requests | 18.7 MB   | 150,000   | 150,000   | 0       | 0         | 112ms |
  +-------------------------------+-----------+-----------+-----------+---------+-----------+-------+
  * Concurrent ingest: 113ms | join/aggregate: 7ms | 5 questions: 20ms
  * Neighborhoods: 40 total, 40 present in both datasets (join coverage: 100.0% of crimes, 100.0% of 311 requests)
  * Heap after ingest: 54.5 MB | heap after join + GC of raw records: 0.8 MB | GC cycles: 13


Q1. Where is combined public-safety and service burden highest?
---------------------------------------------------------------
Stakeholder: Mayor's Office

Research question: Which neighborhoods carry the heaviest combined load of Part 1 crime AND 311 service demand, and how concentrated is it?
  * 40 neighborhoods were matched across both datasets.
  * Highest combined burden: DRUID HEIGHTS (index 97.7 of 100).
  * The top 10 neighborhoods (25.0% of matched neighborhoods) generate 37.2% of crime and 34.7% of 311 demand.
  * Recommendation: prioritize these neighborhoods for coordinated multi-agency investment rather than single-agency responses.

Top 10 neighborhoods by composite burden index (0-100)
  DRUID HEIGHTS                  | ████████████████████████████████████████ 97.7
  LAURAVILLE                     | ██████████████████████████████████████ 91.8
  CHARLES VILLAGE                | █████████████████████████████████████ 90.9
  PIMLICO GOOD NEIGHBORS         | █████████████████████████████████ 81.3
  FEDERAL HILL                   | █████████████████████████████████ 81.0
  COLDSTREAM HOMESTEAD MONTEB... | █████████████████████████████████ 79.8
  PARK HEIGHTS                   | ████████████████████████████████ 79.3
  BELAIR-EDISON                  | ████████████████████████████████ 78.6
  HIGHLANDTOWN                   | ███████████████████████████████ 75.7
  EDMONDSON VILLAGE              | ██████████████████████████████ 73.5

  +------+---------------------------------+---------------+--------------+--------------+
  | Rank | Neighborhood                    | Part 1 crimes | 311 requests | Burden index |
  +------+---------------------------------+---------------+--------------+--------------+
  | 1    | DRUID HEIGHTS                   | 3,482         | 6,096        | 97.7         |
  | 2    | LAURAVILLE                      | 3,328         | 5,640        | 91.8         |
  | 3    | CHARLES VILLAGE                 | 3,652         | 4,994        | 90.9         |
  | 4    | PIMLICO GOOD NEIGHBORS          | 3,109         | 4,727        | 81.3         |
  | 5    | FEDERAL HILL                    | 2,924         | 5,002        | 81.0         |
  | 6    | COLDSTREAM HOMESTEAD MONTEBELLO | 2,610         | 5,378        | 79.8         |
  | 7    | PARK HEIGHTS                    | 2,858         | 4,905        | 79.3         |
  | 8    | BELAIR-EDISON                   | 2,501         | 5,415        | 78.6         |
  | 9    | HIGHLANDTOWN                    | 2,494         | 5,066        | 75.7         |
  | 10   | EDMONDSON VILLAGE               | 2,512         | 4,764        | 73.5         |
  +------+---------------------------------+---------------+--------------+--------------+


Q2. Which high-crime neighborhoods are under-using 311?
-------------------------------------------------------
Stakeholder: Mayor's Office of Neighborhood Engagement

Research question: Does 311 demand track crime, and which high-crime neighborhoods file unusually few 311 requests per crime (possible under-reporting)?
  * Crime and 311 volume across neighborhoods have a weak positive correlation (Pearson r = 0.32).
```
Five answers on the **synthetic** sample (meaningless as municipal facts, but demonstrates the pipeline): Q1 Druid Heights index 97.7; Q2 r=0.32, lowest engagement Mondawmin 0.62 req/crime; Q3 crime peaks 21:00, 311 peaks 10:00; Q4 r=0.70, top blight quartile 4.0x crime; Q5 median close 80.1 h (high-crime) vs 27.6 h (low-crime).

**Real-data run (grader-supplied Open Baltimore files, not bundled by the student):**
```
$ CRIME=.../ebubechukwu_eze/.../crime_nibrs.csv REQUESTS=.../arpana_basnet/data/requests_2025.csv /usr/bin/time -l ./run.sh
        2.04 real   2.73 user   0.61 sys     382451712 max RSS (382 MB)
| Part 1 Crime Data             | 71.1 MB  | 276,007   | 220,714 | 55,293  | 0 | 602ms  |
| 311 Customer Service Requests | 361.8 MB | 931,324   | 656,827 | 274,497 | 0 | 1.617s |
Neighborhoods: 290 total, 279 present in both datasets; Heap after ingest: 250.7 MB | after join + GC: 3.1 MB
```
Real-data answers: Q1 Downtown highest burden (80.1/100), top 10 neighborhoods = 17.9% of crime and 17.4% of 311; Q2 r=0.82, Canton Industrial Area lowest 311 engagement (0.49 req/crime, 16% of city rate); Q3 crime peaks 17:00, 311 at 13:00, 37.8% of crime 20:00–05:00; Q4 46.8% of 311 is blight-related, r=0.65, top blight quartile has 6.0x the crime; Q5 median close 24.5 h (highest-crime quartile) vs 25.2 h (lowest) — essentially equal. These are plausible and consistent with other students' results on the same files.

## Concurrency Analysis

- `cmd/engine/main.go:45-57` — both datasets ingested concurrently (`sync.WaitGroup`, two goroutines).
- `internal/ingest/loader.go:64-120` — per-file pipeline: one reader goroutine (`:66-93`) batches 4,096 rows into `batches := make(chan [][]string, workers*2)`; `workers` (= NumCPU) parser goroutines (`:96-112`) parse/normalize/intern into per-batch slices; `atomic.Int64` counters for read/dropped/malformed; `wg.Wait(); close(results)` closer goroutine; collector appends. Correct and bounded.
- `internal/analysis/context.go:83-125` — `BuildContext` join: `workers` goroutines each aggregate a disjoint `chunk` of both slices into a **private** `map[string]*HoodStats` (`parts[w]`), merged after `wg.Wait()`. Lock-free by design; verified parallel == serial for 1/3/8 workers in `analysis_test.go`.
- `internal/analysis/questions.go:75-86` — `RunAll` runs the five questions concurrently, each writing to its own slice index over an immutable `Context`; `quartiles` sorts a copy (`stats.go:69`).
- `internal/ingest/interner.go:17-29` — `sync.Map` interner shared across workers; `timeparse.go:49` atomic layout hint.
- Race detector clean. **Level: full** — parallel parse, parallel join/aggregate, parallel analysis, correct sync.

## Memory/Resource Analysis

Designed and documented: 1 MB `bufio` reader, bounded batch channels (`workers*2`) for back-pressure, string interning so kept fields do not pin whole CSV lines (explained in `interner.go` comments), pre-sized batch/out slices, `float32` resolution times, pointer iteration (`&crimes[i]`), explicit `crimes, reqs = nil, nil; runtime.GC()` after the join (`main.go:65-69`), `runtime.MemStats` heap-before/after printed in the report (real data: 250.7 MB after ingest -> 3.1 MB after join). The design still materializes all records before joining (382 MB RSS on 433 MB input) — a streaming aggregate would be flatter — but the thinking is explicit, accurate, and measured.

## Architecture & Code Quality

- Clean layout: `cmd/` vs `internal/{model,ingest,analysis,report}`; ingestion, matching (`BuildContext`), analysis and output are in separate packages exactly as the assignment asks.
- Patterns are real, not decorative: `RowParser[T]` and `Question` and `Renderer` are Strategy interfaces with multiple implementations; `NewQuestion(id)` / `NewRenderer(format)` are Factories; `report.Builder` builds a format-neutral `Report`; `ingest.Load` is a channel Pipeline. Generics (`Load[T any]`) keep one pipeline for both schemas.
- Robustness: alias-based schema resolution with helpful missing-column errors, 15 timestamp layouts, UTC-vs-local handling per dataset, `LazyQuotes`, malformed-row counting, whole-word blight matching.
- Tests cover parsing, schema aliases, concurrent load, stats, and parallel/serial equivalence. No tests for `report` or the questions' numeric outputs.
- Naming and comments are good; the README architecture section matches the code precisely. Nothing reads as unbacked boilerplate — if anything the engineering outstrips the data story.

## Stakeholder Questions & Value Assessment

| # | Stakeholder | Question | Cross-dataset? |
|---|---|---|---|
| 1 | Mayor's Office | Composite crime+311 burden index, concentration in top-N | yes |
| 2 | Office of Neighborhood Engagement | r(crime,311); high-crime neighborhoods with lowest 311 per crime (under-reporting) | yes |
| 3 | BPD & DOT | Hour-of-day peaks of crime vs 311 intake | parallel histograms, not a join |
| 4 | Planning / HCD | Blight-request quartiles vs average crime; r | yes |
| 5 | DPW / Office of Equity | Median & p90 311 close time by neighborhood crime quartile | yes |

Framing is strong: each answer has plain-language findings, an ASCII chart, a table, and a recommendation, and the engine also writes `report.md`. Q2 (Inner Harbor / Canton Industrial Area / Dunbar-Broadway under-use 311 relative to crime) and Q5 (equity check on resolution time) are the kind of thing a Neighborhood Engagement office or DPW could act on. Q3 is two independent histograms placed side by side, and Q1's 0–50/0–50 index is a simple rescale. Caveat: because no real data ships, a grader following the README sees only synthetic results; the real-data outputs above required files from elsewhere.

## Issues

1. **No real data bundled** (empty `data/`), no fetch script; README requires a manual portal download. The documented `./run.sh` on the submission as delivered analyzes synthetic random data.
2. Synthetic generator is the automatic fallback — a user could easily mistake its output for real findings despite the warning line.
3. Q3 is not truly cross-dataset (two single-dataset hourly histograms compared visually).
4. All records are materialized before the join (382 MB RSS on real data) despite the "memory stays flat" framing; flat applies to the ingestion buffers, not the retained records.
5. No tests for the report renderers or for the questions' actual numeric output.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---:|---:|---|
| build | 15 | 15 | `go build` and `go vet` clean on first try with documented commands. |
| real_data | 15 | 5 | Zero data files shipped; only a synthetic generator plus manual-download instructions (rubric: "only sample/simulated data present... generator included" = 5). Engine verified to work on real files supplied by the grader. |
| concurrency | 15 | 15 | Parallel parse pool, lock-free map-reduce join, concurrent questions, atomics/WaitGroup, race-clean, parallel==serial tested. |
| memory | 10 | 10 | Interning, bounded channels, bufio, pre-sizing, float32, explicit GC + MemStats before/after; documented and measured. |
| architecture | 15 | 15 | Clean cmd/internal layout; Strategy/Factory/Builder/Pipeline genuinely realized; generics; tests. |
| stakeholder | 15 | 12 | Five well-motivated questions with findings, charts, tables, recommendations; Q3 single-dataset; results as delivered are synthetic. |
| runs_results | 10 | 8 | Runs end-to-end and produces five coherent answers, but on synthetic data by default; real-data run required grader-supplied files. |
| readme | 5 | 5 | Explicit macOS build/run/verify, run.sh, flag table, architecture; matches reality and is honest about the synthetic fallback. |
| **Total** | **100** | **85** | **B** |

## Final Grade Rationale

Engineering-wise this is one of the strongest Go submissions: genuine multi-stage concurrency, careful memory design with measurement, realized design patterns, tests, and a polished report layer, all of which build and run cleanly. It is held to a B by the data requirement: the assignment asks for real Open Baltimore data downloaded onto the computer, and the submission ships none — only a synthetic generator that runs automatically when the real files are absent. The saved `report.md` and a grader-supplied real-data run prove the engine handles real exports correctly, which is why runs_results and stakeholder are not penalized further, but real_data is scored at the rubric's sample-only level. 85/100, B.


## Lead-grader normalization
real_data adjusted 5 -> 2 (rubric: -3 when a mock generator is the default path; run.sh auto-generates synthetic data). Total 85 -> 82 (B). Applied for consistency with Jessica Madu, whose situation is identical.
