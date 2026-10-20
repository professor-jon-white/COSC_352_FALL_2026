# Michael Curbeam — COSC 352 Midterm Capstone Grading Report

**Summary: 90/100 (A).** A genuinely well-engineered four-package Go pipeline that fetches two live Open Baltimore ArcGIS layers in parallel, joins them with a bounded worker pool using private partial aggregates (no mutexes), answers five mostly cross-dataset questions (including a Pearson correlation) with ASCII tables and histograms, and is race-clean. The main deductions: no data is bundled (everything is fetched at runtime), `go.mod` demands Go 1.27.2 (satisfied here only by an automatic toolchain download), there are no unit tests (one live network test), and stakeholders are implied rather than named.

## Inventory
- Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/michael_curbeam/`
- Language: Go (go.mod `go 1.27.2`, module `baltimore-engine`, stdlib only)
- Source files (994 LOC, 12 files): `main.go` 69; `ingest/client.go` 93, `ingest.go` 50, `models.go` 48, `service_requests.go` 31, `vacant_buildings.go` 16, `probe_test.go` 25; `processor/processor.go` 202, `classify.go` 44; `analytics/analytics.go` 91, `questions.go` 183; `ui/table.go` 142
- README: yes (detailed). Run script: `run.sh`. Tests: 1 live integration test (`TestFetchAllLive`, skipped with `-short`). Late: no.
- Data files: **none bundled.** No `data/` directory.

## Datasets & Data Realness
- **311 Customer Service Requests (current year)** — ArcGIS FeatureServer `services1.arcgis.com/UWYHeuuJISiGmgXx/.../311_Customer_Service_Requests_current/FeatureServer/0`, fetched as JSON attributes with `where=Neighborhood IS NOT NULL`, `orderBy CreatedDate DESC`, capped at 20,000 newest records (`ingest/service_requests.go:13`). Struct fields `RowID, ServiceRequestNum, SRType, MethodReceived, CreatedDate, SRStatus, Agency, Address, ZipCode, Neighborhood, CouncilDistrict, PoliceDistrict, Latitude, Longitude` match the Open Baltimore 311 schema exactly.
- **Vacant Building Notices (Open)** — DHCD layer `egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1`, all rows (~11.5K). Fields `NoticeNum, DateNotice, NT, OWNER_ABBR, HousingMarketTypology2023, Council_District, Neighborhood, BLOCKLOT, Address`.
- Verified live at grading time: the run fetched **20,000 311 requests and 11,481 vacant notices**; the live test logged a real record: `ServiceRequestNum:26-00979750 SRType:ECC-Miscellaneous Request ... Address:1300 BLK WALKER-GITTINGS, Baltimore City, 21239 ZipCode:21239 Neighborhood:Idlewood PoliceDistrict:Northeastern`. Output neighborhoods (Carrollton Ridge, Broadway East, Sandtown-Winchester, Belair-Edison, Coldstream Homestead Montebello) and ZIPs (21217, 21223, 21213) are real Baltimore places with plausible magnitudes.
- No mock/sample generator exists; there is no offline path at all. **data_status: real_fetched_runtime** — nothing bundled, so a grader without network cannot run or verify it.

## Build Results
Toolchain note: the machine's base Go is 1.24.0 (`/opt/homebrew/bin/go`), but `go.mod` declares `go 1.27.2`. With `GOTOOLCHAIN=auto` (default) Go transparently downloaded `golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64` over the network and built with it. The README documents the 1.27 requirement and a troubleshooting row for it; without network the build would fail with `go: go.mod requires go >= 1.27.2`.
```
gofmt -l .        -> no output (all formatted)
go build ./...    -> exit 0, no output (4.2 s incl. toolchain)
go vet ./...      -> exit 0, no output
go build -o <scratch>/michael/baltimore-engine . -> exit 0
```
Clean build with the documented commands.

## Test/Vet/Race Results
- `go test -short ./...` -> `ok baltimore-engine/ingest 0.331s` (live test skipped); other packages: no test files. No unit tests for processor/analytics/ui.
- `go test ./ingest -run Live -v` -> `--- PASS: TestFetchAllLive (25.19s)` fetching 20,000 + 11,481 records.
- `go run -race .` (README step 4) -> full pipeline completed, **0 `DATA RACE` warnings**, 32.2 s wall, 167 MB RSS. race_detector: clean.

## Run Results
Command: `./baltimore-engine` (built per README; equivalent to `./run.sh` / `go run .`). Exit 0. **Wall 20.54 s** (20.27 s of that is the two network fetches), **max RSS 44.7 MB**. Progress lines on stderr:
```
ingest: fetching 311 requests and vacant buildings...
ingest: 20000 311 requests, 11481 vacant notices (20.269s)
processor: joined 277 neighborhoods, 31 ZIP codes (2ms)
analytics: answered 5 questions (0s)
```
Stdout (`<scratch>/michael/run_stdout.txt`), complete:
```
==============================================
BALTIMORE DATA ENGINE - 311 x VACANT BUILDINGS
==============================================

Q1: Top 5 neighborhoods by combined vacant notices + 311 requests
+------+---------------------------------+----------------+--------------+----------+
| Rank | Neighborhood                    | Vacant Notices | 311 Requests | Combined |
+------+---------------------------------+----------------+--------------+----------+
|    1 | CARROLLTON RIDGE                |            743 |          231 |      974 |
|    2 | BROADWAY EAST                   |            719 |          226 |      945 |
|    3 | SANDTOWN-WINCHESTER             |            587 |          294 |      881 |
|    4 | BELAIR-EDISON                   |             75 |          672 |      747 |
|    5 | COLDSTREAM HOMESTEAD MONTEBELLO |            234 |          487 |      721 |
+------+---------------------------------+----------------+--------------+----------+
  Combined
  CARROLLTON RIDGE                | ######################################## 974
  BROADWAY EAST                   | ###################################### 945
  SANDTOWN-WINCHESTER             | #################################### 881
  BELAIR-EDISON                   | ############################## 747
  COLDSTREAM HOMESTEAD MONTEBELLO | ############################# 721
Note: 311 counts cover only the most recent records ingested.

Q2: Correlation of vacancies with 311 request types (Pearson r across neighborhoods)
+------------------------+----------+-----------+------------+
| 311 Category           | Requests | Pearson r | Strength   |
+------------------------+----------+-----------+------------+
| Sanitation/Dumping     |     5191 |    +0.696 | moderate   |
| Housing Code           |      818 |    +0.559 | moderate   |
| Transportation/Streets |     3623 |    +0.262 | weak       |
| Water/Sewer            |      844 |    +0.290 | weak       |
| Other                  |     9524 |    +0.173 | negligible |
+------------------------+----------+-----------+------------+
Note: r near +1 = categories rise with vacancy counts; near 0 = no linear relationship.

Q3: Open vs. closed 311 requests in heavy-vacancy areas
+---------------+---------------+----------+------+--------+--------+----------+
| Area          | Neighborhoods | Requests | Open | Closed | % Open | % Closed |
+---------------+---------------+----------+------+--------+--------+----------+
| Heavy vacancy |            69 |     9231 | 3375 |   5856 |  36.6% |    63.4% |
| All others    |           208 |    10769 | 3777 |   6992 |  35.1% |    64.9% |
+---------------+---------------+----------+------+--------+--------+----------+
Note: Heavy vacancy = top quartile of neighborhoods (>= 28 vacant notices). 'Open' includes New/Pending/Hold.

Q4: Top 3 ZIP codes needing code-enforcement intervention
+------+-------+----------------------+----------------+-------+
| Rank | ZIP   | Open Enforcement 311 | Vacant Notices | Score |
+------+-------+----------------------+----------------+-------+
|    1 | 21217 |                  300 |           2605 |  2905 |
|    2 | 21223 |                  285 |           2378 |  2663 |
|    3 | 21213 |                  285 |           1538 |  1823 |
+------+-------+----------------------+----------------+-------+
  Score
  1 | ######################################## 2905
  2 | #################################### 2663
  3 | ######################### 1823
Note: Score = open code-enforcement requests (HCD, sanitation, dumping) + vacant notices. Vacancies mapped to ZIP via each neighborhood's dominant 311 ZIP.

Q5: Data density across 277 neighborhoods (resource planning)
+----------------+-------+-----------+--------+-----+---------------+------------------+
| Metric         | Total | Mean/Hood | Median | Max | Hoods w/ Data | Densest          |
+----------------+-------+-----------+--------+-----+---------------+------------------+
| 311 requests   | 20000 |      72.2 |   49.0 | 672 |           277 | BELAIR-EDISON    |
| Open 311       |  7152 |      25.8 |   18.0 | 215 |           272 | BELAIR-EDISON    |
| Vacant notices | 11481 |      41.4 |    5.0 | 743 |           213 | CARROLLTON RIDGE |
+----------------+-------+-----------+--------+-----+---------------+------------------+
```
Minor rendering nit: the Q4 histogram labels bars "1/2/3" (rank) because `labelFor` picks the first non-numeric cell and ZIP codes are numeric.

## Concurrency Analysis
- Ingest (`ingest/ingest.go:13-50`): generic `start[T]` launches one goroutine per dataset returning a buffered result channel; `FetchAll` multiplexes both with `select`, and the first error cancels the sibling via `context.WithCancel`. Paging within each fetcher is sequential (bounded by the API).
- Join/aggregate (`processor/processor.go:78-115`): both datasets are sliced into 2,000-record chunks; each chunk's goroutine acquires a semaphore channel sized `runtime.NumCPU()` (bounded pool), builds a **private** partial `Joined`, and sends it on `partials`; a closer goroutine does `wg.Wait(); close(partials)`; the caller is the sole merger. Context checked before work. No shared mutable state, hence no mutexes — correct by construction.
- Analytics (`analytics/analytics.go:25-39`): five goroutines, one per question, each writing its own slot `results[i]` over read-only joined data; `sync.WaitGroup`.
- `main.go:30-33`: `signal.NotifyContext` + 3-minute `WithTimeout` propagate cancellation through every stage.
- Race detector: clean on a full live run. Level: **full** — parallel ingest, bounded worker-pool join/aggregate, parallel analysis, correct synchronization.

## Memory/Resource Analysis
- Bounded worker pool; private partial maps merged by a single owner (documented in README §4); buffered/drained channels and single-owner `close`; `defer resp.Body.Close()`; 30 s HTTP client timeout; 20,000-record cap on the 311 pull (`maxServiceRequests`) to bound memory and time; `json.NewDecoder` streams the HTTP body rather than buffering it.
- Observed: 44.7 MB max RSS for ~31.5K records; 167 MB under `-race`.
- Not done: no `runtime.MemStats` reporting, records are fully materialized (acceptable at this scale), latitude/longitude kept as strings. Clearly designed and documented; 9/10.

## Architecture & Code Quality
- Four packages exactly mirroring the required separation: `ingest` (fetch) → `processor` (join/aggregate + `classify.go` rules) → `analytics` (questions over a generic `Result{Question, Headers, Rows, Note, BarCol}`) → `ui` (table/histogram renderer taking `io.Writer`). `main.go` is a thin orchestrator writing progress to stderr and results to stdout.
- Patterns realized: Pipeline (explicit, with plain data handed between stages), generic paged client `fetchLayer[T]` (Template/Strategy for any ArcGIS layer), fan-out/fan-in worker pool with semaphore, higher-order `RunAll([]func(Joined) Result)`. Deterministic ordering (tie-breaks by name) and `sortedHoods` helper show care.
- Weaknesses: no unit tests for the pure functions (`pearson`, `IsClosed`, `Category`, `merge`, `finalize`) that would be easy to test; `go.mod` pinned to 1.27.2 makes the build toolchain-download dependent; vacancies attributed to a neighborhood's "dominant 311 ZIP" is a heuristic that can misassign across ZIP boundaries (disclosed in output note).
- Originality/consistency: README is detailed and matches the code line-for-line (file names, counts, timings "30-40 s" vs observed 20 s). Nothing reads as unrelated boilerplate.

## Stakeholder Questions & Value Assessment
Stakeholders are implied (code enforcement / DHCD for Q4, "resource planning" for Q5, the README speaks of "municipal stakeholder questions") but no office is explicitly named in code or output — a small gap against the spec.
1. Top 5 neighborhoods by combined vacant notices + 311 requests — cross-dataset ranking; Carrollton Ridge, Broadway East, Sandtown-Winchester lead — consistent with known West/East Baltimore vacancy belts.
2. Pearson r between vacancy counts and 311 category counts across 277 neighborhoods — the strongest question: sanitation/dumping r=+0.70 and housing-code r=+0.56 vs transportation r=+0.26 is an actionable, non-obvious cross-dataset finding (DPW/DHCD should co-target dumping cleanup with vacancy abatement).
3. Open vs closed 311 share in top-quartile-vacancy neighborhoods vs the rest (36.6% vs 35.1% open) — cross-dataset; the finding that service closure rates are not worse in high-vacancy areas is itself informative for an equity question.
4. Top 3 ZIPs by open enforcement requests + vacant notices (21217, 21223, 21213) — cross-dataset, directly actionable for code enforcement, though vacancy-to-ZIP mapping is heuristic.
5. Data density summary (totals/mean/median/max per neighborhood for both datasets) — the weakest; descriptive statistics on each dataset rather than a cross-dataset insight.
Caveats are printed per section (recency skew of the 20K newest 311 records). Visualization: aligned ASCII tables plus `#` histograms for Q1 and Q4.

## Issues
- No data bundled; the program requires live network access to the Baltimore ArcGIS endpoints every run (rubric caps real_data at 10 for runtime-only fetch). No offline/sample fallback or cached snapshot.
- `go.mod` requires Go 1.27.2; the installed Go 1.24 only built because `GOTOOLCHAIN=auto` downloaded the newer toolchain over the network. Offline, the documented `go build ./...` would fail.
- Only one test, and it is a live network integration test; no unit tests for the pure join/stat/classify functions.
- Stakeholders not explicitly named; Q5 is descriptive rather than cross-dataset.
- 311 side is only the 20,000 most recent records (about 2% of the layer), so Q1/Q3/Q4 counts are recency-biased (disclosed in notes).
- Vacancies attributed to ZIP via each neighborhood's dominant 311 ZIP — a heuristic that can misallocate.
- Q4 histogram labels show rank numbers instead of ZIP codes because ZIPs parse as numeric.

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 14/15 | gofmt/build/vet clean first try with documented commands, but only via an automatic network download of the Go 1.27.2 toolchain that go.mod demands (documented in README). |
| real_data | 10/15 | Two genuine Open Baltimore datasets with exact schemas, fetched and verified live (20,000 + 11,481 records); nothing bundled, so runtime-fetch-only per rubric. |
| concurrency | 15/15 | Parallel fetch with select/ctx cancel, bounded semaphore worker pool over chunks with private partials and single-owner merge, parallel analytics, WaitGroups; race detector clean. |
| memory | 9/10 | Bounded pool, no shared mutable state, capped download, timeouts, streaming JSON decode, documented in README; no MemStats, records fully materialized. |
| architecture | 14/15 | Four packages matching ingest/join/analysis/output, generics, clean Result abstraction, deterministic output; lacks unit tests (one live test only). |
| stakeholder | 13/15 | Four of five questions genuinely cross-dataset with a real statistical finding (Pearson r) and actionable ZIP ranking, tables + histograms; stakeholders implied not named, Q5 descriptive. |
| runs_results | 10/10 | Runs end-to-end in 20.5 s, exit 0, 44.7 MB RSS, all five answers sensible and consistent with known Baltimore geography. |
| readme | 5/5 | Thorough, accurate macOS build/run/verify/troubleshooting instructions plus run.sh; matches the code exactly. |
| **Total** | **90/100** | **A** |

## Final Grade Rationale
This submission meets the engineering core of the assignment better than most: real Open Baltimore data with the correct schemas, a correct and idiomatic concurrent pipeline (parallel fetch, bounded worker-pool join with mutex-free partial aggregates, parallel analytics) verified race-free, a clean four-package architecture, and five questions of which four are genuinely cross-dataset and one (the vacancy/311-category correlation) produces a finding a DHCD or DPW director could act on. The grade is held to a low A by the rubric's cap for runtime-only data (nothing bundled, so it cannot run or be verified offline), the Go 1.27.2 toolchain requirement that only resolved via network download, the absence of unit tests, and stakeholders being implied rather than named. 90/100, A.
