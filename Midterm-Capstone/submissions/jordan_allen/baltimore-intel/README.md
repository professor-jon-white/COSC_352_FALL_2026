# Baltimore Crime and 311 Analysis (Go)

This is my project for the Open Baltimore assignment. It is a command-line program
written in Go that reads two datasets from the Baltimore City Open Data Portal
(crime incidents and 311 service requests), joins them by neighborhood, and answers
five questions that a city official might ask. The answers print in the terminal as
tables and text charts, and the program can also save them as a Markdown report.

It only uses Go's standard library, so there is nothing extra to install besides Go
itself.

## What is in this folder

| File or folder | What it is |
|---|---|
| `main.go` | Reads the command-line options and runs everything |
| `ingest.go` | Reads the CSV files with a pipeline of goroutines |
| `datasets.go` | Describes the two datasets and keeps the running totals |
| `join.go` | Matches crime and 311 data by neighborhood |
| `questions.go` | The five questions (Q1 to Q5) |
| `report.go`, `charts.go` | Formatting: tables, bar charts, the hourly histogram, Markdown |
| `stats.go`, `util.go` | Correlation, medians, ranks, small helpers |
| `main_test.go` | Automated tests |
| `data/` | Small pieces of the real Open Baltimore files (see "The data") |

## How to compile and run on macOS

I built this on a Windows laptop, so I want to be upfront that I have not been able
to try it on a Mac myself. I did compile it, run it and run the tests on Windows
(Go 1.27.2), and everything passed. Nothing in the code is specific to one operating
system, so it should behave the same on macOS.

**1. Install Go (version 1.21 or newer).** If you use Homebrew:

```bash
brew install go
go version
```

`go version` should print 1.21 or higher. If you do not use Homebrew, the installer
is at https://go.dev/dl/.

**2. Open Terminal and go to the project folder.** Use whatever path the folder is
in on your computer, for example:

```bash
cd ~/Downloads/baltimore-intel
```

The folder you want is the one that contains `main.go` and `go.mod`.

**3. Compile.**

```bash
go build -o baltimore-intel .
```

This makes a program called `baltimore-intel` in the same folder. It prints nothing
when it works.

**4. Run it on the real data I included.**

```bash
./baltimore-intel -crime data/crime_sample_week.csv -requests data/requests_sample_week.csv -report report.md
```

The report prints in the terminal and a copy is saved to `report.md`. It is long, so
you may need to scroll up to see Q1. You can also skip the compile step and use
`go run . -crime data/crime_sample_week.csv -requests data/requests_sample_week.csv`.

## How to verify that it worked

**Step 1: run the tests.**

```bash
go vet ./...
go test ./...
```

`go test` should print a line starting with `ok`. There is also
`go test -race ./...`, which checks for data races (it needs the Xcode command line
tools on a Mac). I did not run this one on Windows, so I have not seen its result myself.

**Step 2: check the numbers from the run in step 4 above.** I worked these out
separately from the program, so if yours match then the pipeline is doing what it
should:

| Where | What you should see |
|---|---|
| Header | Crime: 1137 rows read, 1137 accepted. 311: 1540 rows read, 1540 accepted. Join: 198 neighborhoods in both |
| Q1 | Pearson r = 0.46, Spearman rho = 0.40. Top row of the table is BELAIR EDISON (24 crimes, 48 requests) |
| Q2 | 554 requests in the top-crime quarter vs 862 elsewhere. Most over-represented type is SW-DIRTY ALLEY PROACTIVE at 6.22x |
| Q3 | Crime peaks at 22:00 (6.3%), 311 peaks at 12:00 (12.9%), profiles correlate at r = -0.28 |
| Q4 | Median days to close is 1.1 in the lowest-crime quarter and 0.8 in the highest, which is 0.68x |
| Q5 | r = 0.32 across 198 neighborhoods. The most disorder-heavy quarter averages 3.7 violent crimes per neighborhood vs 1.5 in the least |

## The data

I used two datasets from https://data.baltimorecity.gov/:

- **NIBRS Group A Crime Data**, which has one row per crime
- **311 Customer Service Requests 2025**, which has one row per request

The full downloads are very large (about 75 MB for crime and 380 MB for 311), so the
`data/` folder has small pieces of the **real** files instead. The columns and values
are unchanged.

| File | What is in it |
|---|---|
| `crime_sample_week.csv` | Every crime from 1 to 7 June 2025 that has a neighborhood (1,137 rows) |
| `requests_sample_week.csv` | Every 11th 311 request from the same week (1,540 rows) |
| `crime_first50.csv`, `requests_first50.csv` | The first 50 rows of each file. I used these first as a quick test while building. The dates do not line up (the crime rows are from October 2026 and the 311 rows from April and May 2025), so the statistics mean very little. The week files above are the ones to use |

I took the same week from both so that the two datasets cover the same time period.
Rows with an empty neighborhood were removed because they cannot be joined.

If you want to try the program on the full files, download the CSVs from the portal
into `data/` and run it with the same options. Use `-from` and `-to` (for example
`-from 2025-01-01 -to 2025-12-31`) so both datasets cover the same period. The
program finds the columns by their header names rather than their position, so it
should still work if the portal changes the column order. If a needed column is
missing it stops and prints the headers it did find.

## The five questions

| # | Who is asking | Question |
|---|---|---|
| Q1 | Mayor's Office | Which neighborhoods rank high on both crime and 311 requests, and how closely do the two go together? |
| Q2 | Public Works | Which kinds of 311 requests show up more than usual in the highest-crime neighborhoods? |
| Q3 | Police and Transportation | At what times of day do crimes and 311 requests peak, and how do the two patterns compare? |
| Q4 | Mayor's Office (fairness) | Do high-crime neighborhoods wait longer for 311 requests to be closed? |
| Q5 | City Planning | Do neighborhoods with more physical disorder requests (trash, graffiti, vacant buildings) also have more violent crime? |

Every question uses both datasets together. For example, Q2 and Q4 use crime counts
to sort neighborhoods into four groups and then look at the 311 data inside each
group, and Q5 does it the other way around. Each section starts with a few "Key
findings" sentences and then shows the table and chart behind them.

## How it is built

**Design patterns.**
- **Pipeline** (`ingest.go`): rows go from a reader, through a channel, to worker
  goroutines, and then get merged together.
- **Factory** (`datasets.go`): the crime and 311 datasets are created from a
  description of their columns, so the reading code is shared.
- **Strategy** (`datasets.go`, `questions.go`, `report.go`): the two kinds of
  aggregator share one interface, every question has the same `Question` interface,
  and there are two interchangeable output renderers (plain text and Markdown).
- **Builder** (`report.go`): a `ReportBuilder` puts the report together section by
  section.

The three jobs are kept apart on purpose. Reading the files does not know anything
about the questions. The questions only see the joined data. The formatting code only
sees a finished report.

**Concurrency.** Both datasets are read at the same time, each by its own pipeline.
In a pipeline, one goroutine reads the CSV and sends batches of rows (2048 by
default) into a channel. Several worker goroutines (the `-workers` option) take
batches from the channel, parse the dates and numbers, and count things. When the
file ends, the workers finish and their results are merged. After the join, the five
questions run as five goroutines at once. If one question crashes, the error is
caught and shown in its own section so the rest of the report still prints.

**Thread safety.** I avoided locks completely. A batch of rows belongs to the reader
until it is sent and to exactly one worker once it is received. Each worker only
changes its own private totals, and the totals are merged by a single goroutine after
the workers are done (`wg.Wait()`). The questions only read the joined data and never
change it. Since no two goroutines ever write to the same thing, there is nothing to
lock and no chance of a deadlock. `go test -race` is how I checked this.

**Memory.**
- The files are streamed. A whole file is never loaded into memory. For each row only
  the three or four columns that are needed are kept and the rest are thrown away.
- What gets stored is the totals (counts per neighborhood and category), not the
  rows, so memory use does not grow with the size of the file. The one exception is a
  `float32` per closed 311 request, which is needed to get exact medians.
- The channel has a limited size (`-queue` batches). If the workers fall behind, the
  reader waits. This means only a fixed number of rows can be in memory at once.
- A batch of rows is allocated in two big pieces instead of one per field, which
  cuts down on garbage for Go's garbage collector to clean up. Batches become
  garbage as soon as a worker finishes with them.
- The number of worker goroutines is set at the start instead of making one per row.
- At the end of the run, the program prints its heap usage, total bytes allocated and
  number of garbage collections (from `runtime.MemStats`), so the memory use can be
  seen and not only described.

## Options

`./baltimore-intel -h` lists everything. The ones I used most:

| Option | What it does |
|---|---|
| `-crime`, `-requests` | The CSV files to read (required). More than one file is allowed if they are separated by commas with no spaces |
| `-from`, `-to` | Only use records between two dates, written as `YYYY-MM-DD` |
| `-report` | Also write the report to a Markdown file |
| `-tz` | Convert times to a time zone, for example `America/New_York`. Without it the times are used as they are written in the files |
| `-workers`, `-batch`, `-queue` | Tuning for the pipeline |
| `-violent`, `-disorder` | The keywords used to decide what counts as a violent crime or a disorder request |

## Limits and assumptions

- **Joining by name.** Neighborhoods are matched by name after making them upper
  case and removing punctuation. A neighborhood that appears in only one dataset is
  left out of the questions, and the `Join:` line at the top lists them.
- **Violent and disorder are keyword guesses.** I decided which crimes count as
  violent (for example ASSAULT and ROBBERY) and which 311 requests count as disorder
  (for example TRASH and GRAFFITI) by searching the text. Other reasonable choices
  would change the Q5 numbers a little.
- **Raw counts.** I did not divide by population or neighborhood size, so bigger
  neighborhoods tend to rank higher.
- **Only one week of data.** The results are real but they come from a short window
  and a sample of the 311 requests, so they should not be treated as a finding about
  the whole city. The point is to show that the program works.
- **Q4 and open requests.** Only closed requests can be timed, so requests that are
  still open are not in the medians. The report shows what share of requests were
  closed next to the medians. The Q4 result (high-crime areas closing requests
  faster) is probably affected by which request types are common in each group, which
  is why the report also compares individual types.
- **Correlation is not cause.** Q1 and Q5 show that two things go together, not that
  one causes the other.

## How I made this

I used Claude to help write and debug the code. To make sure I understood it, I first
built a simpler single-file version of all five questions one step at a time, ran
each step myself, and checked the output against numbers calculated separately
before moving on. The multi-file version in this folder is the final one. I compiled
it, ran the tests and ran it on the included data on Windows, and the results matched
the table in the verify section.
