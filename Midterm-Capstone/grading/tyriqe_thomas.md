# Tyriqe Thomas — COSC 352 Midterm Capstone Grading Report

**Summary: 62/100 (D).** Go. Builds, vets, tests and runs cleanly (race detector clean), with unusually careful validation, statistics and tests — but the project misses the core spec: only one of the two datasets is from Open Baltimore (a one-row citywide dropout indicator), the other is hand-transcribed from a BPD CompStat PDF, the two cannot be joined (student admits this), and concurrency is token (two loader goroutines, three writer goroutines over ~20 rows of data).

## Inventory

Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/tyriqe_thomas/baltimore_etl/`

| File | LOC | Role |
|---|---|---|
| main.go | 316 | CSV helpers, loaders/validation, atomic file writes, `run()` pipeline, console report |
| analysis.go | 345 | Five "research question" sections written to Markdown |
| stats.go | 137 | Pearson/Spearman, t-based p-values (incomplete beta), OLS |
| report.go | 163 | Self-contained HTML report with hand-built SVG bar charts |
| concurrent.go | 50 | `group` (WaitGroup + 1-slot error channel + panic recovery) |
| analysis_test.go / run_test.go / main_test.go | 187 / 145 / 147 | Unit + end-to-end tests |
| README.md, go.mod (`go 1.21`), .gitignore | | |
| data/compstat_districts.csv | 10 lines (9 districts) | hand-typed from BPD CompStat PDF |
| data/High_School_Dropout_Withdrawl_Rate.csv | 2 lines (1 data row) | BNIA/Open Baltimore ArcGIS export |
| baltimoreetl | 3.1 MB | **prebuilt Linux x86-64 ELF binary** — will not run on macOS |

Total source LOC 1,490 (1,011 non-test). Tests: yes. Run script: none (README commands only). Standard library only.

## Datasets & Data Realness

1. `High_School_Dropout_Withdrawl_Rate.csv` — header `OBJECTID,City,drop10,...,drop17,drop19,drop20,Shape__Area,Shape__Length,drop21`, one row `Baltimore City`. This matches the BNIA-JFI Vital Signs layer "High School Dropout/Withdrawal Rate – City" that Open Baltimore catalogs (ArcGIS Hub export; `Shape__Area/Shape__Length` fields are the giveaway). **Real, from Open Baltimore, but it is a single pre-aggregated citywide row (11 values), not a raw record dataset.**
2. `compstat_districts.csv` — `district,homicide_2021_ytd,...,gun_arrests_2020_ytd`, 9 rows. README states these were "transcribed by hand from the BPD CompStat PDF (week ending 07/17/2021)". **Not an Open Baltimore dataset**; it is a manually typed summary table. Values are plausible (541 gun incidents, 709 gun arrests citywide YTD) and the README cross-checks totals against the PDF.

No mock/sample generator exists. Row counts: compstat 9 data rows, dropout 1 data row. Net: ≥2 datasets in the loose sense, but only one is from the portal and neither is "raw"; total input is ~20 numbers.

## Build Results

```
cd submissions/tyriqe_thomas/baltimore_etl
go build -o <scratch>/baltimoreetl .     # exit 0, 0.82 s, no output
go vet ./...                             # exit 0, no output
go build -race -o baltimoreetl_race .    # exit 0
```
Clean first try. The shipped `baltimoreetl` binary is a Linux ELF (`file` output: "ELF 64-bit LSB executable, x86-64, statically linked") and cannot be executed on macOS; README instructs the user to build, so this is harmless but sloppy.

## Test/Vet/Race Results

```
go test ./...          ok  baltimoreetl  0.237s
go test -race ./...    ok  baltimoreetl  1.289s
./baltimoreetl_race -data data -out out   # exit 0, 0 "DATA RACE" reports
```
Tests are genuinely good: wide→long reshape, BOM handling, validation errors, deterministic output across 30 runs, NaN/Inf guards, p-value against textbook critical value, end-to-end run writing all three outputs.

## Run Results

```
/usr/bin/time -l ./baltimoreetl -data data -out <scratch>/tyriqe/output
real 0.23 s, max RSS 5.6 MB, exit 0
```
Console output (complete):
```
Loaded 11 dropout years for Baltimore City, 9 CompStat districts

COMPSTAT 2021 YTD (gun-related incidents and gun arrests)
district   incidents  arrests
Central    62         83
Eastern    72         62
Northeast  39         85
Northern   29         71
Northwest  63         103
Southeast  45         80
Southern   71         55
Southwest  72         73
Western    88         97

Wrote output files (dropout_long.csv, research_answers.md, report.html) to .../output
```
`research_answers.md` (the five answers, abridged):
1. **Descriptive** — 541 gun-related incidents, 188 homicides, 379 NFS, 709 gun arrests, 11,624 Part I crimes (9 districts, 1/1–7/17/2021). Most: Western (88, 16.3%); fewest: Northern (29). Dropout rate 2010–2021 mean 3.23%, high 4.16% (2011), low 2.05% (2014), latest 3.02% (2021); 2018 missing.
2. **Exploratory** — 5 districts rose, 4 fell; largest rise Central (+77.1%), largest fall Northeast (−22.0%); incidents up while arrests fell in Central, Eastern, Southeast, Southern; arrests/incident 0.77 (Southern) to 2.45 (Northern); largest one-year dropout drop 2.00 pts (2012→2013).
3. **Explanatory** — 2020 vs 2021 gun incidents r = 0.69 (p = 0.04) significant; Part I vs gun incidents r = −0.48 (p = 0.19) n.s.; arrests vs incidents r = 0.06 n.s. "The dropout rate is citywide, so it cannot be tested against districts."
4. **Evaluative** — citywide gun incidents 503→541 (+7.6%), gun arrests 632→709 (+12.2%); change-in-arrests vs change-in-incidents r = −0.20 (p = 0.61) n.s.
5. **Predictive** — leave-one-out: fitted line error 14.8, mean-of-others 16.8, repeat-2020 10.9 → "repeating the district's 2020 count" is best.

`report.html` contains two SVG bar charts (district incidents/arrests; dropout by year) and a table. Output is deterministic and atomically written.

## Concurrency Analysis

- `concurrent.go:16-50` — `group`: `sync.WaitGroup` + one-slot buffered `chan error`, non-blocking `report()` via `select/default`, `recover()` converts panics to errors.
- `main.go:261-272` — stage 1: two goroutines, one per input file (`load.Go(...)`).
- `main.go:281-287` — stage 2: three goroutines, one per output file.
- No worker pool, no parallel parsing/join/aggregate; nothing to parallelize with 10 input rows. The helper is correct and race-clean, but this is **token concurrency** by the rubric's definition ("two goroutines loading two files, nothing else").

## Memory/Resource Analysis

- Atomic writes via temp file + rename (`main.go:76-98`); `defer f.Close()`; bounded 1-slot error channel; outputs built in `strings.Builder`/`bytes.Buffer`.
- `readCSV` loads all rows into `[]map[string]string` (fine for 10 rows, not a streaming design).
- No MemStats reporting, no discussion of memory in README/comments. Some evidence of resource care, little memory thinking.

## Architecture & Code Quality

Single `package main`, but files are cleanly split by concern (ingest/validate → analysis → stats → report), idiomatic Go, good naming, well-commented, deterministic ordering everywhere (sorted district lists), excellent input validation (consistency checks such as `NFS ≤ gun-related ≤ homicides + NFS`), and real tests. No named design pattern is claimed or realized beyond an implicit two-stage pipeline. Maintainability is high. The statistics (incomplete-beta p-values, Spearman with tie ranks, LOO comparison) are more rigorous than most submissions and are unit-tested against known values. The README is honest about limitations. Nothing reads as copy-pasted boilerplate; code and README agree.

## Stakeholder Questions & Value Assessment

The five "questions" are organized by analysis type (descriptive / exploratory / explanatory / evaluative / predictive), not by stakeholder; **no municipal stakeholder is named**. Only Q1 and Q2 touch both datasets, and they merely list dropout and CompStat facts side by side — the student's own caveat states the datasets cannot be joined. Q3–Q5 are CompStat-only. With 9 district points and a single citywide dropout series, a Mayor's Office would learn nothing actionable it could not read directly off the CompStat PDF. The HTML/SVG report is clean and the honesty about significance is commendable, but cross-dataset insight is absent by construction.

## Issues

- Only one dataset (dropout) comes from Open Baltimore, and it is a single aggregated citywide row; CompStat table is hand-transcribed from a PDF, not a portal dataset.
- Datasets are not joined and cannot be (different geography/time); the "cross-dataset" requirement is unmet.
- Concurrency is token: 2 loader + 3 writer goroutines over ~20 numbers; no worker pool, no parallel parse/aggregate.
- No stakeholders named for the questions.
- Shipped prebuilt binary is a Linux ELF, not a macOS executable.
- No run script (README commands only) — minor.

## Score Table

| Criterion | Pts | Score | Justification |
|---|---|---|---|
| build | 15 | 15 | `go build` clean first try, `go vet` clean, documented command works on macOS. |
| real_data | 15 | 7 | One real Open Baltimore dataset (1-row BNIA indicator); second is hand-typed from a BPD PDF (real numbers, not portal data); nothing raw, no join possible. |
| concurrency | 15 | 5 | Correct WaitGroup/channel helper but only 2 loader + 3 writer goroutines; no pool, no parallel parse/join/aggregate. |
| memory | 10 | 4 | Atomic writes, bounded error channel, deferred closes; no streaming/memory design or discussion. |
| architecture | 15 | 10 | Single package but clean file separation, strong validation, deterministic, well-tested; no named pattern realized. |
| stakeholder | 15 | 6 | 5 answered questions with real statistics and SVG charts, but no stakeholders named, Q3–Q5 single-dataset, Q1–Q2 only juxtapose. |
| runs_results | 10 | 10 | Runs in 0.23 s, all outputs produced, numbers sensible and reproducible. |
| readme | 5 | 5 | Thorough macOS compile/run/verify steps with expected outputs and troubleshooting; matches reality. |
| **Total** | **100** | **62** | **D** |

## Final Grade Rationale

This is well-engineered, well-tested Go, and everything the README promises works exactly as described. It loses most of its points on the assignment's substance rather than its craft: the data requirement (≥2 real raw Open Baltimore datasets, joined) is largely unmet — one tiny indicator row from the portal plus a hand-typed PDF table that cannot be joined to it — and the concurrency requirement is satisfied only nominally because there is nothing to parallelize. The stakeholder analysis is statistically honest but delivers no cross-dataset insight and names no stakeholder. 62/100, D.
