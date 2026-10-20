# Abigail Wanjiku — COSC 352 Midterm Capstone Grading Report

**Summary: 95/100 (A)** — Go. Exceptionally engineered two-dataset (NIBRS crime x 311) pipeline with a chunked parallel CSV loader, sharded concurrent join, a compare mode that proves sequential == concurrent, extensive tests (race-clean), measured memory work, and five well-motivated cross-dataset questions. Only shortfall against the rubric: no real data is bundled; the user must download the two CSVs with the curl commands in the README (verified working).

## Inventory

- Location: `submissions/abigail_wanjiku/midterm-capstone/`
- Language: Go (`go.mod` module `capstone`, `go 1.27.2`; the installed go1.24 auto-downloaded the 1.27.2 toolchain via GOTOOLCHAIN=auto — transparent, but requires network on first build)
- Source files: 33 `.go` files, 3,390 LOC total (incl. tests). Non-test LOC ~2,300.
  - `cmd/capstone/` main.go (242), memstats.go (64), rss_unix.go (20), rss_windows.go (36), main_test.go (133), sample_test.go (72)
  - `internal/ingest/` ingest.go (263), concurrent.go (322) + tests; `ingest/crime/crime.go` (83); `ingest/requests311/requests311.go` (92)
  - `internal/match/match.go` (169), `internal/aggregate/aggregate.go` (59), `internal/model/model.go` (102), `internal/stats/stats.go` (115)
  - `internal/report/` strategy.go (161), report.go, format.go, ingest.go, categories.go, q1_sanitation.go (82), q2_distress.go (125), q3_closetime.go (134), q4_streetlights.go (107), q5_highdemand.go (102)
- README.md: 850 lines, very detailed. CI workflow `.github/workflows/ci.yml` (vet, build, test -race, sample run on macos-latest).
- Run script: none (README gives `go build -o capstone ./cmd/capstone` then `./capstone`).
- Tests: 11 test files covering every package.
- Data files bundled: only `testdata/crime_sample.csv` (101 lines, 28 KB) and `testdata/requests_311_sample.csv` (111 lines, 48 KB) — **synthetic**, explicitly labelled as such. Real data (`data/crime.csv`, `data/requests_311.csv`) is gitignored and must be downloaded.

## Datasets & Data Realness

| Dataset | Source | Bundled? | Rows (data) | Verified |
|---|---|---|---|---|
| NIBRS Group A Crime Data | Open Baltimore item `204beefe92a645d79fdf0969957bbdf8` | No — curl URL in README | 278,239 | Yes |
| 311 Customer Service Requests 2025 | Open Baltimore item `52c763f4496d4101ba0c20af33336a9c` | No — curl URL in README | 931,324 | Yes |

I ran the README's exact two `curl -L` commands into the scratchpad (`.../scratchpad/abigail/data/`). Both succeeded (75.1 MB and 379.4 MB). Headers match Open Baltimore schemas exactly:
- crime: `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Shooting,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents,x,y`
- 311: `SRRecordID,ServiceRequestNum,SRType,MethodReceived,CreatedDate,SRStatus,StatusDate,DueDate,CloseDate,Agency,...,Neighborhood,CouncilDistrict,PoliceDistrict,...`
- Sample records: `5300 SINCLAIR LN ... FRANKFORD`, `1713 N CHAPEL ST, Baltimore City, 21213 ... Broadway East`; 2,830 rows mention Sandtown in crime.csv. Row counts (278,239 / 931,324) match the README's documented counts exactly.

Mock/sample generator: `testdata/` synthetic samples exist and `main_test.go` generates random CSVs for tests. **Not the default path** — defaults are `data/crime.csv` / `data/requests_311.csv`; samples are only used with explicit `-crime/-requests` flags. The README is honest that the sample values are made up.

Data status: **real_fetched_runtime** in spirit (not bundled; fetched by README-provided curl commands rather than by the program).

## Build Results

```
$ go build -o capstone ./cmd/capstone      # (output to scratchpad)
exit=0   4.7 s wall (includes go1.27.2 toolchain download)
$ go version -m capstone  ->  go1.27.2
$ go vet ./...                              exit=0, no output
$ go build -race -o capstone_race ./cmd/capstone   exit=0
```
Clean build, no warnings. Note: `go.mod` pins `go 1.27.2`; a machine with an older Go and no network would fail to fetch the toolchain (README documents this behavior).

## Test / Vet / Race Results

```
$ go test ./...
ok  capstone/cmd/capstone  1.086s        ok  capstone/internal/aggregate
ok  capstone/internal/ingest             ok  capstone/internal/ingest/crime
ok  capstone/internal/ingest/requests311 ok  capstone/internal/match
ok  capstone/internal/model              ok  capstone/internal/report
ok  capstone/internal/stats
$ go test -race ./...     -> all 9 packages ok
$ ./capstone_race -crime data/crime.csv -requests data/requests_311.csv -reports=sanitation
   exit=0, 0 "DATA RACE" reports, 5.3 s wall
```
Race detector: **clean** on both the test suite and a full real-data run.

## Run Results

Real data (scratchpad copy), default concurrent mode, `-memstats`:
```
$ /usr/bin/time -l ./capstone -crime data/crime.csv -requests data/requests_311.csv -memstats
exit=0    2.37 s real    max RSS 285 MiB (298,942,464 bytes)
```
Captured output (`scratchpad/abigail/run_real.txt`, 175 lines) — first lines and the five answers:
```
concurrent: ingest 482ms, match 43ms

Ingest (2025)               crime        311
rows read                  278239     931324
rows kept                   54062     656827
rows skipped               224177     274497
  outside year             224160          0
  no neighborhood              17     274497
  malformed                     0          0
distinct neighborhoods        278        279
neighborhoods in both: 278
3336 neighborhood-month records across 279 neighborhoods (701 311 close times excluded as before creation)

Q1 [sanitation] Do neighborhoods with the most sanitation/dumping 311 requests also have the highest crime?
#   neighborhood                     sanitation 311                    crimes  crime rank
1   COLDSTREAM HOMESTEAD MONTEBELLO  3,308           ################  519     19 / 279    ####
2   BELAIR-EDISON                    2,933           ##############    1,115   3 / 279     #########
...
  sanitation 311 vs crime     +0.78 (very strong positive)
  all resident 311 vs crime   +0.87 (very strong positive)   (baseline)
Answer: Yes (rank correlation +0.78).
But it is not specific to sanitation: overall 311 volume tracks crime even more closely...

Q2 [distress] Top 10 neighborhoods by combined distress (crime + 311 volume), by quarter
Q1 2025 ... 1 DOWNTOWN 439 3,653 8.66 ################ ...
Q2 2025  note: 311 volume this quarter is 47% of the busiest quarter's, likely missing records; ...
Answer: 7 neighborhoods are in the top 10 all four quarters:
  BELAIR-EDISON, BROOKLYN, CANTON, DOWNTOWN, FRANKFORD, SANDTOWN-WINCHESTER, WASHINGTON VILLAGE/PIGTOWN.

Q3 [close-time] Are 311 requests closed more slowly in high-crime neighborhoods?
crime quartile  nbhds  crimes each  closed 311  avg days   mix-adjusted
Q1 (lowest)     70     0-60         33,882      16.3       1.00x
Q4 (highest)    69     243-2,024    350,857     12.8       0.98x
  raw avg days to close        -0.28 (weak negative)
  mix-adjusted close ratio     +0.01 (no meaningful)
Answer: No meaningful difference once request mix is accounted for (highest vs lowest crime quarter: -2%).

Q4 [streetlights] Do neighborhoods with many streetlight requests see more nighttime outdoor crime?
  night outdoor crimes (count)           +0.59 (strong positive)
  all crimes (count, baseline)           +0.59 (strong positive)
  night outdoor share of crime (256 nbhds) +0.11 (weak positive)
Answer: Somewhat. They have more night outdoor crime, but mostly because they have more crime overall ...
13.9% in the lowest streetlight quartile to 16.7% in the highest.

Q5 [high-311-low-crime] Which neighborhoods have high 311 volume but low crime?
Strict test: resident 311 requests >= 2,542 (75th percentile) and crimes <= 127 (median).
1  MOUNT WASHINGTON  2,887  102  28  SW-Recycling (6%)
2  ROLAND PARK       2,689  119  23  TRS-Parking Complaint (9%)
Answer: 2 of 279 neighborhoods pass the strict test (MOUNT WASHINGTON, ROLAND PARK).
By requests per crime, MOUNT WASHINGTON leads with 28 resident requests per crime, against 10 citywide.

Memory (concurrent mode)
  peak heap in use      231 MiB   peak process memory   285 MiB
  total allocated       717 MiB   GC cycles 14   GC pause total 5.2ms   GC CPU fraction 6.2%
```
Compare mode on real data (`run_compare.txt`): sequential ingest 1.323s / match 134ms vs concurrent 286ms / 59ms (4.2x total speedup, 10 CPUs); `Results identical: 54062 crimes, 656827 311 requests, ingest stats, 3336 neighborhood-month records`.

Documented sample path (no real data needed): `go run ./cmd/capstone -mode=compare -crime testdata/crime_sample.csv -requests testdata/requests_311_sample.csv` -> exit 0, `Results identical: 85 crimes, 105 311 requests ...`, 5 `Answer:` lines, exactly as the README predicts.

## Concurrency Analysis

Primitives: goroutines, channels (bounded), `sync.WaitGroup`, generics-based worker pool; no mutexes needed by design.
- `cmd/capstone/main.go:152-172` `loadConcurrent`: two goroutines load the two files concurrently, WaitGroup join, `errors.Join`.
- `internal/ingest/concurrent.go:57-147` `Concurrent[T]`: splitter goroutine reads 1 MiB chunks and cuts at CSV record boundaries honoring quoting rules (`lastRecordEnd`, lines 221-288, using vectorized `bytes.IndexByte`); `chunks` channel capacity 2 (backpressure, line 76); `free` channel recycles buffers (77); N workers each with private `csv.Reader`/`Scanner`/intern maps (89-107); closer goroutine closes `results` after `wg.Wait()` (110-113); collector reassembles by chunk index so output order and stats equal the sequential scan.
- `internal/match/match.go:60-98` `ByNeighborhoodMonthConcurrent`: sharding by FNV-1a hash of neighborhood; each worker builds a private `buckets` map and writes only `parts[w]`; concatenate + sort after `wg.Wait()`.
- Verification: `-mode=compare` deep-compares (`reflect.DeepEqual`) every record and summary between modes; tests exercise tiny chunk sizes and several worker counts under `-race`. Race detector clean on full data.
Level: **full** — parallel parsing and parallel join, correct synchronization, verified.

## Memory/Resource Analysis

Designed, documented (README section 7 with before/after measurements), and evident in code:
- Streaming: `csv.Reader.ReuseRecord = true` (ingest.go:140); never `ReadAll`. Concurrent mode bounds raw bytes to ~(workers+4) x 1 MiB via the 2-deep chunk channel and `free` buffer recycling.
- String interning: `Scanner.Intern`/`Neighborhood` (ingest.go:223-241) copy each distinct value once with `strings.Clone` so kept fields do not pin whole CSV lines.
- Slim structs: `model.Crime` 64 B, `model.ServiceRequest` 96 B, sizes pinned by `model_test.go`.
- Preallocation from row hints (`sc.RowsHint()`), exact-size merged slices, `parts[i].records = nil` to release chunks early (concurrent.go:139-143).
- Measurement: `memstats.go` samples `runtime/metrics` heap every 5 ms for a true peak, reports `runtime.MemStats` GC stats and `getrusage` max RSS; README documents GOGC trade-offs.
- `withFile` helper with `defer f.Close()` on all paths.
Observed: 285 MiB max RSS for 1.2 M rows / 455 MB of CSV, 14 GC cycles.

## Architecture & Code Quality

Multi-package layout with strict stage separation (`ingest` -> `match` -> `aggregate` -> `report`), stages depending only on `model`. Patterns actually realized: **Pipeline** (stages with shrinking typed hand-offs), **Strategy** (`report.Report` interface with five concrete question types, `report.Select`/`Run`), worker-pool/fan-out-fan-in and sharding. Naming is clear, comments explain *why* (e.g., midnight-as-unknown-time, clock skew on close times), every package has focused tests, and the CI workflow mirrors the README's verify steps. Code uses modern Go idioms correctly (generics, `slices`, `cmp.Or`, range-over-int). The README matches the code precisely — the row counts, timings and answers it documents reproduce exactly on a fresh download. The code and README are unusually polished for a midterm; whether AI-assisted or not, the student clearly understands it (the design rationale, measurements and limitations are coherent and specific to this code).

## Stakeholder Questions & Value Assessment

Stakeholders named: 311-responding agencies (DPW, DOT), BPD/public-safety planners, Mayor's Office & City Council, neighborhood associations, researchers.
1. Sanitation/dumping 311 vs crime — with a baseline (all resident 311 vs crime) to show sanitation is not a distinctive signal. Cross-dataset, honest.
2. Combined distress index by quarter, normalized to quarter averages; flags the April–May 2025 311 data gap. Cross-dataset; gives a stable shortlist (7 neighborhoods every quarter).
3. 311 close time vs crime quartile with a mix-adjusted ratio — the raw answer reverses once request mix is controlled. Genuinely insightful for an equity question.
4. Streetlight requests vs night outdoor crime, with share-of-crime control. Cross-dataset, DOT/BGE-relevant, honest "somewhat".
5. High 311 / low crime neighborhoods (Mount Washington, Roland Park) with top request type.
Each question includes an ASCII bar table and a one-line verdict. The joins are meaningful (neighborhood x month keys, 278 of 278 crime neighborhoods matched) and the analyses control for size, which is what makes them actionable rather than coincidental counting. Visualization: ASCII tables/bars in terminal only.

## Issues

- No real data bundled; relies on the user running two curl commands (verified working today). The program has no fetch mode.
- `go 1.27.2` in go.mod forces a toolchain download on machines with older Go (documented, worked here).
- Visualization is terminal-only (no HTML/web UI) — acceptable per spec.
- 29.5% of 311 rows lack a neighborhood and are dropped (documented in Limitations).

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | Clean first-try build with the documented command; vet clean; race build clean. |
| real_data | 15 | 10 | Two real Open Baltimore datasets, verified by download via README's exact curl URLs, but none bundled and no fetch script/program; synthetic samples are not the default path. |
| concurrency | 15 | 15 | Parallel chunked CSV parsing with worker pool + sharded parallel join, bounded channels/WaitGroup, compare mode proves equivalence, race detector clean. |
| memory | 10 | 10 | Streaming, interning, buffer recycling, pinned struct sizes, peak-heap sampler and RSS reporting, documented with measurements. |
| architecture | 15 | 15 | Clean multi-package pipeline, Strategy pattern realized, tests in every package, CI. |
| stakeholder | 15 | 15 | Five genuinely cross-dataset questions with baselines/controls, named stakeholders, clear verdicts and ASCII charts. |
| runs_results | 10 | 10 | Runs end-to-end on real data in 2.4 s, produces all five answers; sample path also works as documented. |
| readme | 5 | 5 | Explicit macOS install/download/build/run/verify/test instructions that match reality exactly. |
| **Total** | **100** | **95** | **A** |

## Final Grade Rationale

This submission meets or exceeds every engineering requirement: it compiles cleanly, uses real Open Baltimore data (verified), parallelizes both the expensive parse and the join with correct synchronization proven by a compare mode and the race detector, documents and measures its memory behavior, and separates stages into well-tested packages with a real Strategy pattern. The five questions are substantive cross-dataset analyses with baselines that keep the conclusions honest. The only rubric gap is that the real data is not shipped with the submission and the program does not fetch it itself; the user must run the documented curl commands, which costs 5 points on real_data. 95/100, A.
