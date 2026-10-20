Name: Jessica Madu
Course: COSC352
Date: 15 October 2026
# Baltimore Civic Intelligence (Go)

A command-line tool that ingests two public datasets from the
[Open Baltimore portal](https://data.baltimorecity.gov/), joins them by
**neighborhood**, and answers five research questions for fictional city
stakeholders (Mayor's Office, Public Works, City Planning, Transportation).

| Dataset | Portal name | Used fields |
|---|---|---|
| 1 | **Part 1 Crime Data** (CSV) | `Neighborhood`, `Description`, `CrimeDateTime` |
| 2 | **311 Customer Service Requests** (CSV, any year) | `Neighborhood`, `SRType`, `CreatedDate`, `SRStatus` |

* Language: **Go** (standard library only, no dependencies to install)
* Output: formatted tables + ASCII bar charts in the terminal, also saved to `report.txt`

---

## 1. Compile & run on macOS

### Install Go (one time)
```bash
brew install go          # or download the installer from https://go.dev/dl/
go version               # should print go1.21 or newer
```

### Build and run with the bundled sample data (works offline)
```bash
cd baltimore-intel
go build -o baltimore-intel .
./baltimore-intel
```

> `data/sample/` contains **synthetic** files with the same column layout as the
> real datasets, so the program can be graded without a download.

### Run on the real data
1. On https://data.baltimorecity.gov/ search for **"Part 1 Crime Data"** → *Export* → **CSV**.
2. Search for **"311 Customer Service Requests"** (e.g. the 2025 file) → *Export* → **CSV**.
3. Run:
```bash
./baltimore-intel \
  -crime ~/Downloads/Part1_Crime_Beta.csv \
  -requests ~/Downloads/311_Customer_Service_Requests_2025.csv \
  -crime-years 2024 -requests-years 2025
```
The crime file spans 2011-2024 while the 311 file is 2025 only, so the program
filters each file to the year(s) you choose (defaults: crime 2024, 311 2025 = the
latest full year of each). Q5 compares seasonality by month-of-year for this reason.
A real-data result from the files above is saved in `report_real_data.txt`.

### Flags
| Flag | Default | Meaning |
|---|---|---|
| `-crime` | `data/sample/crime_sample.csv` | crime CSV path |
| `-requests` | `data/sample/311_sample.csv` | 311 CSV path |
| `-crime-years` | `2024` | year, range like `2023-2024`, or `all` |
| `-requests-years` | `2025` | same, for 311 |
| `-workers` | number of CPUs | parser goroutines per dataset |
| `-top` | 10 | rows per ranking |
| `-out` | `report.txt` | report file |

## 2. Verify

```bash
go vet ./...
go test -race ./...      # unit tests + Go race detector (proves thread safety)
./baltimore-intel -workers 1 -out /tmp/a.txt
./baltimore-intel -workers 8 -out /tmp/b.txt
diff <(grep -v -E 'Generated|Runtime|\(.*ms\)' /tmp/a.txt) <(grep -v -E 'Generated|Runtime|\(.*ms\)' /tmp/b.txt) && echo "Same results with 1 or 8 workers"
```
The report header shows rows used / skipped and how many neighborhoods joined.
If a real CSV uses unexpected column names, the program exits with a message
listing the header it found (column matching is case/punctuation-insensitive and
accepts several known names per field; edit the candidate lists in `datasets.go`).

## 3. The five questions

| # | Stakeholder | Question | Cross-dataset logic |
|---|---|---|---|
| 1 | Mayor's Office | Which neighborhoods carry the most crime and what is their 311 picture? | crime rank + violent % joined with 311 volume and % still open |
| 2 | Public Works | Where is crime high but 311 engagement low? | crime ÷ 311 ratio for above-average-crime areas |
| 3 | City Planning | Do high-crime neighborhoods also produce more 311 requests? | Pearson *r* + crime-quartile table |
| 4 | Transportation / DPW | Which 311 request types are over-represented in the top-crime areas? | "lift" = share in hot areas ÷ citywide share |
| 5 | Operations | When do crime and 311 demand peak? | monthly trend + time-of-day blocks side by side |

## 4. Architecture

```
main.go      wiring: start two pipelines concurrently, then run questions
datasets.go  Dataset interface (Strategy) + Crime/311 factories, header binding, date parsing
ingest.go    Pipeline: reader -> worker pool -> merge
model.go     Event (value type) and Agg (per-worker aggregate)
analysis.go  Question interface (Strategy): five independent analyses
render.go    ASCII bars, sorting and formatting helpers
```

**Design patterns**
* **Pipeline** – `Ingest` is reader → N parsers → merger, connected by channels.
* **Strategy** – `Dataset` (how to read a file) and `Question` (how to analyze) are interfaces; new datasets/questions plug in without touching the pipeline.
* **Factory** – `NewCrimeDataset()` / `New311Dataset()` hide column details.

**Concurrency**
* Both datasets are ingested at the same time (one goroutine each, results returned over channels).
* Inside each, a pool of `-workers` goroutines parse batches of 2,000 rows.
* A `sync.WaitGroup` closes the results channel once all workers finish.

**Memory & thread safety**
* The CSV is **streamed**; raw rows are never all in memory. Memory is
  O(neighborhoods × categories), not O(rows). Final heap size is printed in the report.
* The job channel is **bounded** (`workers*2` batches) which gives back-pressure:
  a slow parser makes the reader wait instead of growing the heap.
* Each worker owns a **private `Agg`** (confinement), so the hot path needs no
  mutex; the partial results are merged by a single goroutine. `go test -race`
  verifies there are no data races.
* `Event` is a small value type (no pointer allocation per row); Go's
  garbage collector frees each processed batch automatically.
* The `RowParser` closure is read-only after creation, so sharing it between
  goroutines is safe.

## 5. Assumptions / limitations
* Join key is the upper-cased, trimmed `Neighborhood` name; rows without one, with a bad date, or outside the selected years are counted as "skipped" (about 29% of 2025 311 rows have no neighborhood).
* Crime (2024) and 311 (2025) are different years, so the join is a *neighborhood-level structural comparison*, not a same-period comparison.
* Q4 ignores finance/administrative 311 types (prefix `FC`) and very rare types.
* "Violent" = description contains HOMICIDE, SHOOTING, ROBBERY, AGG. ASSAULT or RAPE.
* A 311 request is "open" if its status contains OPEN, NEW, IN PROGRESS or PENDING.
* Correlation is not causation; Q3 reports association only.
