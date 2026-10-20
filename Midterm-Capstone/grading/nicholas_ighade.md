# Nicholas Ighade — COSC 352 Midterm Capstone Grading Report

**Summary: 10/100 — F** (LATE submission; Python/Colab notebook submitted as a PDF printout; no runnable code, no Open Baltimore data, wrong language)

## Inventory

| Item | Value |
|---|---|
| Submitted file | `submissions/nicholas_ighade/ighadenicholas_LATE_4579_8488982_Untitled8.ipynb - Colab mid t.pdf` (72,594 bytes, 7 pages, printed from Chrome/Edge on Windows, created 2026-10-16 02:01 EDT) |
| Source files | 0 (the only artifact is a PDF screenshot/print of a Google Colab notebook; no `.ipynb`, no `.py`, no repo) |
| Language | Python 3 (imports `os, io, csv, time, math, urllib.request, concurrent.futures`, `collections`, `datetime`). **Not an approved language** (Go / Scala / Clojure / Rust). |
| LOC (as printed) | ~440 lines of Python code + ~55 lines of captured notebook output (extracted with `pdftotext -layout`, 497 lines total) |
| README / run script / tests | None |
| Data files | None. Two inline CSV string literals: `SAMPLE_311_CSV` (16 rows, columns `ServiceType,ZipCode,Neighborhood,CreatedDate,ClosedDate`) and `SAMPLE_LIQUOR_CSV` (7 rows, `Name,ZipCode,Neighborhood`). |
| Late flag | Filename contains `_LATE_` — flagged, not deducted. |

## Datasets & Data Realness

- `URL_311 = ""` and `URL_LIQUOR = ""` (PDF lines 15-16). The notebook comments say "put real Open Baltimore CSV URLs here if you have them. If the download fails, the script falls back to synthetic sample data."
- The captured output in the PDF shows exactly that fallback: `[loader] no URL configured — using sample data` (twice), `[parse] 311 rows: 16 | liquor rows: 7`, `[join] joined records: 24`.
- The inline sample columns (`ServiceType, ZipCode, Neighborhood, CreatedDate, ClosedDate`) do **not** match the Open Baltimore 311 schema (`SRRecordID, SRType, CreatedDate, Agency, Neighborhood, ...`), and the liquor sample (`Name, ZipCode, Neighborhood`) does not match the Liquor Licenses layer (`LicenseNumber, TradeName, LicenseClass, AddrStreet, ...`). Rows like "The Federal Tap, 21230, Federal Hill" are invented.
- Row counts: 16 and 7. **No real Open Baltimore data was downloaded or bundled.** data_status = simulated.

## Build Results

Not applicable. There is no source file to compile; the submission is a PDF image/print of a notebook. Even if transcribed, the language is Python, which is not approved by the assignment. No build was attempted on student code (no code exists on disk). The `pdftotext` extraction also shows mangled indentation in several blocks (e.g. the `parse_csv_worker_pool` body and `fetch()` helper have inconsistent 3/4-space indentation in the print), so the printed text is not even guaranteed to be valid Python as-is.

Command used to inspect: `pdftotext -layout "<pdf>" scratchpad/ighade.txt` (497 lines).

## Test/Vet/Race Results

None possible. tests = none, race_detector = not_run.

## Run Results

Not run by the grader (nothing runnable). The notebook's own captured output (from Colab) is included in the PDF and shows:

```
============================================================
 Baltimore Civic Insights Engine (BCIE)
============================================================
[loader] no URL configured — using sample data
[loader] no URL configured — using sample data
[parse] decoding with 2 workers...
[parse] 311 rows: 16 | liquor rows: 7
[join] joined records: 24

 Q1: Top 10 ZIP Codes by 311 Request Volume
  21230 |████████████████████████████████████████ 8.00
  21201 |████████████████████████████████████████ 8.00
  21205 |███████████████ 3.00 ...
 Q2: Slowest ZIP Codes for 311 Closure (hours)
  21207 469.5 h (2 requests), 21205 215.0, 21201 213.2, 21230 119.9, 21202 31.5
 Q3: Noise Complaints per Liquor License (Top 10)
  21202 0.67, 21230 0.50, 21205 0.33, 21201 0.25, 21207 0.00
 Q4 / Q5: truncated in the PDF (the print ends mid-Q3 table on page 7)
```

The 5 "answers" are computed over 16 fabricated rows; they carry no information about Baltimore.

## Concurrency Analysis

Python `concurrent.futures.ThreadPoolExecutor` is used in three places (PDF lines 133, 165, 411, 426): per-row decode futures in `parse_csv_worker_pool`, `ex.map(probe, svc_list)` in `join_by_zip`, and one future per analysis strategy. This is not a native-concurrency primitive of an approved language, is subject to the Python GIL, and operates on 16 rows. Rows are fully materialized (`rows = list(reader)`) before being submitted one-future-per-row, which is the opposite of streaming. Token at best.

## Memory/Resource Analysis

`__slots__` on the three record classes is the only memory consideration. Entire CSV text is read into a string, then into a list of rows, then one future per row. No bounded queues, no streaming, no discussion of GC or runtime overhead.

## Architecture & Code Quality

Single notebook cell organized into sections (MODEL / UTILITIES / LOADER / PIPELINE / JOIN / ANALYSIS / RENDER / SAMPLE DATA / MAIN). `Loader` base class with `FileLoader`/`HTTPLoader` and `new_loader()` is a small Factory; the five `q*_` functions passed as a list is a Strategy-ish dispatch. Reasonable for a notebook, but it is a single file in the wrong language with no packaging, no README, no tests. The structure and comments ("Factory + Strategy pattern", "worker-pool CSV parser") read like a Go design translated to Python ("Python / Colab Edition" is in the header), which suggests the design was adapted from boilerplate rather than engineered for the problem.

## Stakeholder Questions & Value Assessment

Questions (from the code):
1. Top 10 ZIP codes by 311 request volume (311 only)
2. Slowest ZIP codes for 311 closure (311 only)
3. Noise complaints per liquor license by ZIP (cross-dataset)
4. 311 volume by month (311 only)
5. Avg 311 closure time, low vs high liquor-density ZIPs (cross-dataset)

Stakeholders: none named (narratives say "allocate intake capacity", "investigate service equity" without naming an office). Three of five questions are single-dataset; the join key is ZIP code, which for real data is a weak proxy. With 16 synthetic rows the output has zero value to any municipal stakeholder.

## Issues

- LATE submission (flagged, not deducted).
- Wrong language: Python, not Go/Scala/Clojure/Rust.
- Submission is a PDF print of a Colab notebook, not source code or a repository; cannot be built or run.
- No Open Baltimore data; URLs left blank; inline 16-row / 7-row fabricated CSVs with non-Open-Baltimore schemas.
- No README, no build/run/verify instructions, no tests.
- Q4 and Q5 output not even visible in the PDF (print truncated).
- Three of five questions are single-dataset; no stakeholders named.

## Score Table

| # | Criterion | Pts | Score | Justification |
|---|---|---|---|---|
| 1 | build | 15 | 0 | Python (not approved) and no compilable source — only a PDF printout. |
| 2 | real_data | 15 | 0 | No Open Baltimore data; blank URLs; 16-row and 7-row invented inline CSVs. |
| 3 | concurrency | 15 | 3 | Python ThreadPoolExecutor used for parse/join/analyses — token, non-native, GIL-bound, 16 rows. |
| 4 | memory | 10 | 1 | `__slots__` only; reads entire text, materializes all rows, one future per row. |
| 5 | architecture | 15 | 3 | Single notebook, sectioned, small Factory/Strategy; no packages, no tests, wrong language. |
| 6 | stakeholder | 15 | 3 | 5 questions but 3 single-dataset, no named stakeholders, answers over fake data; ASCII bars present. |
| 7 | runs_results | 10 | 0 | Nothing runnable was submitted; the captured Colab output is on fabricated data and truncated. |
| 8 | readme | 5 | 0 | No README. |
| | **Total** | **100** | **10** | **F** |

## Final Grade Rationale

The assignment requires an approved language, two real Open Baltimore datasets, native concurrency, and macOS build/run instructions, graded strictly on compile/run success. This submission is a late PDF print of a Python Colab notebook that never downloads data (URLs are empty strings) and runs on 23 hand-typed rows. Nothing can be compiled or executed from what was submitted, and even the notebook's own printed output is truncated before Q4. The few points awarded recognize that the notebook is organized into recognizable pipeline/strategy sections, uses thread pools, and poses five questions, two of which are cross-dataset.
