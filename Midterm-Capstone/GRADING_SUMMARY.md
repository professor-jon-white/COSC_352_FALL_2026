# COSC 352 Fall 2026 — Midterm Capstone Grading Summary

Assignment: Multi-Paradigmatic Data Processing Engine (Go/Scala/Clojure/Rust) ingesting multiple Baltimore City Open Data datasets with native concurrency and answering five stakeholder research questions. Graded 2026-10-19/20 by building every submission from source on macOS, running it unmodified, and reading all source files.

## Folder layout

- `submissions/<firstname_lastname>/` — each student's extracted submission plus a `GRADE.md` with their scores, curve, evidence and column explanations
- `grading/<student>.md` and `.json` — full grader report per student (build logs, run output, race detector, per-criterion justification)
- `grading/GRADES_v3.md`, `grading/grades_v3.csv` — final grade table with raw and curved scores (this summary mirrors it)
- `grading/GRADES_v2.md` — re-weighted rubric before the curve; `grading/GRADES.md` — original 8-criterion rubric for reference
- `grading/RUBRIC.md` — criterion definitions used by the graders
- `_raw/` — untouched Canvas zip contents

## High-level statistics

| Metric | Value |
|---|---|
| Roster | 36 students (plus Canvas test account) |
| Submitted | 29 |
| No submission | 7 — Montrell Martin, Oluwatobi Olajide, Abimbola Olorogun, Gladys Qualls, Nyriq Shareef, Imran Tejan, Drake Wilson |
| Late submissions | 4 — Jordan Allen, Daniel Dada, Nicholas Ighade, Vernon Williams (flagged, not deducted) |
| Language | Rust 2, Go 26, Python 1 |
| Builds cleanly | yes 27, partial 1, no 1 |
| Runs end to end | yes 24, partial 4, no 1 |
| Data status | real_bundled 17, real_fetched_runtime 4, sample_only 2, simulated 6 |
| Concurrency level | full 14, partial 7, token 8 |
| Students with 3+ real datasets | 4 (Eze 5, James 4, Bigelow 4, Rasheed 3) |
| Students with tests | 18 |
| Race detector clean (where run) | 26 of 26 |
| Raw score mean / median / sd | 67.7 / 71 / 21.7 (range 11–100) |
| Curved mean / median | 86.1 / 87 (range 70–100) |
| Raw letters | A 3, B 7, C 5, D 4, F 10 |
| Curved letters | A 10, B 13, C 6 |
| Mean per criterion | Concurrency 14.1/20, Real Data 9.9/15, Five Questions 12.1/20, Three Datasets 8.7/15, Runs Well + Documented 12.3/15, Project Structure 10.6/15 |
| Total data processed across submissions | ~5.3 GB of Open Baltimore exports (largest: Dada 950 MB, Nwokelo 812 MB) |

## Grading criteria (100 raw points)

| Criterion | Pts | How it was scored |
|---|---|---|
| Concurrency | 20 | Native primitives used across parse, join and aggregate with correct synchronization. Full multi-stage worker pools 20; one real stage (usually ingest) ~13; token (one goroutine per file) ~7; none 0. Verified by code reading and Go race detector. |
| Real Data | 15 | Two or more genuine Open Baltimore datasets on disk and used by the documented run. Bundled 15; fetched at runtime only 10; synthetic in exact portal schema 3–9; invented schema or hand-typed rows 0–1. Headers and records checked against the live portal. |
| Five Questions | 20 | Five distinct cross-dataset research questions for named municipal stakeholders with formatted summaries or visuals. Deductions for single-dataset questions, trivial counts, unnamed stakeholders, degenerate answers, no visualization. |
| Three Datasets | 15 | Distinct Open Baltimore datasets actually ingested. 3+ = 15; 2 = 8; mixed sources (one not from portal) 5–10; none 0. |
| Runs Well + Documented | 15 | Run result (10): builds and executes end to end on included/downloaded data with sensible output, no code changed by grader. README (5): explicit macOS build/run/verify steps that match reality. |
| Project Structure | 15 | Separation of ingest / join / analysis / output, design patterns realized in code, package layout, tests, maintainability. |

**Curve.** Piecewise-linear on the raw total with a 70 floor. Anchors: 70 printout only · 75 minimal synthetic data · 80 weak concurrency and weak questions · 85 real data but minimal concurrency/questions · 88–93 real data with full concurrency on two datasets · 100 top projects with 3+ real datasets. Non-submitters remain 0.

## Grades

| # | Student | Lang | Conc /20 | Data /15 | 5 Qs /20 | 3 Sets /15 | Runs+Docs /15 | Struct /15 | Raw | **Curved** | Summary |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | Eze, Ebubechukwu | Rust | 20 | 15 | 20 | 15 | 15 | 15 | 100 | **100** | 5 real Open Baltimore datasets bundled (NIBRS crime 276k, 311 2025 931k, vacant notices 11k, liquor 31k, CitiWatch 861) plus fetch script. |
| 2 | James, Aidan | Go | 20 | 15 | 20 | 15 | 15 | 15 | 100 | **100** | 4 real datasets bundled (crime, 311, liquor, vacant GeoJSON). |
| 3 | Rasheed, Wariz | Go | 20 | 15 | 19 | 15 | 15 | 15 | 99 | **99** | 3 real datasets bundled (311 2026 943k, NIBRS 278k, liquor 31k). |
| 4 | Allen, Jordan (LATE) | Go | 20 | 13 | 19 | 8 | 15 | 13 | 88 | **93** | 2 real datasets (NIBRS + 311) but only a one-week June 2025 excerpt (1,137 crimes / 1,540 requests). |
| 5 | Bigelow, Nickolas | Go | 20 | 9 | 16 | 15 | 13 | 15 | 88 | **93** | 4 datasets fetched via `make fetch` (crime, 311, liquor, vacants) but nothing bundled; documented `make demo` and the submitted report run on a synthetic generator in exact Open Baltimore schemas. |
| 6 | Wanjiku, Abigail | Go | 20 | 10 | 20 | 8 | 15 | 15 | 88 | **93** | 2 real datasets (NIBRS 278k, 311 931k) fetched by the README's curl commands, not bundled. |
| 7 | Stewart, Elizabeth | Go | 20 | 15 | 17 | 8 | 14 | 13 | 87 | **93** | 2 real datasets bundled (NIBRS 54k, vacant notices 11k). |
| 8 | Morton, Tayven | Go | 20 | 15 | 15 | 8 | 14 | 14 | 86 | **92** | 2 real datasets bundled (311 Oct-Dec 2025 244k, vacants 11k) with SHA-256 manifest and reproducible fetch tool. |
| 9 | Dada, Daniel (LATE) | Go | 19 | 15 | 13 | 10 | 14 | 14 | 85 | **92** | 3 real files bundled (crime 644k, 311 2024 1.08M, 311 2025 931k) but the 2025 file is unused and the two 311 files are one dataset split by year. |
| 10 | Curbeam, Michael | Go | 20 | 10 | 17 | 8 | 15 | 14 | 84 | **91** | 2 real datasets fetched live at runtime (20k most-recent 311, 11.5k vacants), nothing bundled. |
| 11 | Basnet, Bishant | Go | 20 | 6 | 16 | 8 | 13 | 15 | 78 | **89** | Declares 2 real datasets but ships none; run.sh silently generates synthetic data in the real schema. |
| 12 | Jones, Xavier | Go | 15 | 15 | 13 | 8 | 15 | 10 | 76 | **88** | 2 real datasets bundled (NIBRS 12k + arrests 4k, Jan-Mar 2025) fetched by its own ArcGIS client with manifest. |
| 13 | Basnet, Arpana | Go | 15 | 15 | 15 | 8 | 14 | 8 | 75 | **88** | 2 real datasets bundled (311 2025 931k, vacants 11k). |
| 14 | Nwokelo, John | Go | 16 | 12 | 9 | 8 | 15 | 14 | 74 | **88** | 2 real datasets (812 MB 311 2026 GeoJSON + NSA boundaries), uploaded separately; bundled samples are synthetic and the default. |
| 15 | Madu, Jessica | Go | 17 | 5 | 17 | 8 | 12 | 12 | 71 | **87** | No real data shipped; default paths point at gen_sample.py output in the real schema with uniform random values, so findings are artifacts (r = 0.99). |
| 16 | Eboh, Kingsley | Go | 13 | 10 | 12 | 8 | 15 | 9 | 67 | **86** | 2 real datasets fetched live by default (707k 311 2026 + 12.6k vacant rehabs, 6m47s). |
| 17 | Lewis, Mekhi | Go | 13 | 12 | 9 | 8 | 14 | 10 | 66 | **86** | 2 real datasets bundled (arrests 400k, 311 943k) but shipped incidentally, unmentioned in README; run.sh defaults to 3-row fixtures. |
| 18 | Efferson, Anaya | Go | 11 | 15 | 8 | 8 | 14 | 7 | 63 | **85** | 2 real datasets bundled (311 2025 931k, vacants 11k). |
| 19 | Hall, Amari | Go | 9 | 15 | 8 | 8 | 14 | 8 | 62 | **84** | 2 real datasets bundled (NIBRS 278k, 311 2026 943k). |
| 20 | Joseph, Grace | Go | 7 | 15 | 8 | 8 | 12 | 5 | 55 | **81** | 2 real datasets bundled (311 2026 943k, vacants 11k). |
| 21 | Arulogun, Obaloluwa | Go | 16 | 3 | 9 | 8 | 6 | 10 | 52 | **80** | Both CSVs are generated by internal/seed/generator.go yet README calls them real exports. |
| 22 | Reyes, Jayden | Go | 7 | 12 | 5 | 10 | 11 | 7 | 52 | **80** | 3 sources: 2 real but reshaped Open Baltimore extracts (arrests, gun offenders) plus a 20-row CVI table hand-typed from a PDF. |
| 23 | Thomas, Tyriqe | Go | 7 | 7 | 8 | 5 | 15 | 10 | 52 | **80** | 1 Open Baltimore dataset (BNIA dropout rate, one citywide row) plus a 9-row CompStat table hand-typed from a BPD PDF; the two cannot be joined, which the student admits. |
| 24 | Williams, Vernon (LATE) | Go | 7 | 12 | 7 | 8 | 8 | 5 | 47 | **78** | 2 real but pre-aggregated 55-row BNIA indicators (violent crime rate, poverty %). |
| 25 | Holder, Gabriel | Go | 13 | 1 | 7 | 8 | 9 | 6 | 44 | **77** | 10 hand-typed synthetic rows per file (admitted). |
| 26 | Smith, Lyric | Go | 7 | 0 | 7 | 8 | 9 | 8 | 39 | **75** | 6 + 6 hand-written rows with invented schemas (README admits 'hand-built'). |
| 27 | Stewart, Elijah | Go | 7 | 1 | 7 | 8 | 9 | 6 | 38 | **75** | 14 fabricated rows with invented headers; README says download real 311 and arrests but gives no URL; default run fails, only --offline works. |
| 28 | Brown, Mikayla | Rust | 7 | 0 | 5 | 8 | 6 | 11 | 37 | **74** | 'GTTF settlements' and officer incidents are not Open Baltimore datasets; 3 + 3 synthetic rows and the program itself prints 'inputs are synthetic'. |
| 29 | Ighade, Nicholas (LATE) | Python | 4 | 0 | 4 | 0 | 0 | 3 | 11 | **70** | Submitted only a 7-page PDF printout of a Colab notebook. |
| 30 | Martin, Montrell | — | | | | | | | 0 | **0** | No submission. |
| 31 | Olajide, Oluwatobi | — | | | | | | | 0 | **0** | No submission. |
| 32 | Olorogun, Abimbola | — | | | | | | | 0 | **0** | No submission. |
| 33 | Qualls, Gladys | — | | | | | | | 0 | **0** | No submission. |
| 34 | Shareef, Nyriq | — | | | | | | | 0 | **0** | No submission. |
| 35 | Tejan, Imran | — | | | | | | | 0 | **0** | No submission. |
| 36 | Wilson, Drake | — | | | | | | | 0 | **0** | No submission. |

Full narrative per student: `grading/GRADES_v3.md` and each student's `submissions/<name>/GRADE.md`.

## Notable questions (novelty ranking)

1. **Ebubechukwu Eze** — CitiWatch camera coverage gaps: 41.5% of violent crime and 50.7% of shootings since 2022 happened more than 150 m from any camera, with a map of where new cameras would see the most. Also the only block-level liquor analysis that controls for neighborhood (outlet blocks are 5.5% of blocks but 17.6% of violent crime).
2. **Aidan James** — Tested whether liquor-block crime is a closing-hours problem and found it is not (21.4% vs 21.3% late-night share): "the excess is about place, not time." Two-hop council-district capital-budget scorecard (District 9 highest).
3. **Abigail Wanjiku** — Streetlight requests vs nighttime outdoor crime with a share-based control, and the inverted question "high 311, low crime" that surfaced Mount Washington and Roland Park at 28 requests per crime vs 10 citywide.
4. **Nickolas Bigelow** — Mix-adjusted 311 wait index by agency across crime tiers (widest gap Recreation & Parks, +14.7 days) and day-of-week x hour patrol timing near liquor outlets.
5. **Wariz Rasheed** — Volume-adjusted search for which 311 types track violent crime, and night-time (21:00–03:59) violent crime by license-density tier (15.8 vs 3.9 per neighborhood).
6. **Daniel Dada** — Under-reporting z-score (crimes per 311 request) flagging Inner Harbor at z = 7.2, and neighborhoods whose monthly crime curve diverges from the city's (Brooklyn r = −0.27).
7. **Jordan Allen** — Lift analysis of 311 types in top-crime neighborhoods (SW-Dirty Alley Proactive 6.2x) and crime-vs-311 hour-of-day divergence for staffing.
8. **Tayven Morton** — Same-request-type closure comparison across vacancy cohorts, isolating parking complaints at +12.4 days in high-vacancy areas.
9. **Michael Curbeam** — Category-level correlation with vacancy (sanitation r = +0.70 vs transportation +0.26) and ZIP-level code-enforcement targeting (21217, 21223, 21213).
10. **Xavier Jones** — Crime-share vs arrest-share gap by neighborhood (Downtown 3.6% of crime, 8.3% of arrests).

Honorable mentions for novel ideas undermined by data: Obaloluwa Arulogun's intake-channel "digital divide" by vacancy tier (synthetic data) and Tyriqe Thomas's leave-one-out predictive test of district gun incidents (hand-typed 9-row table).

## Findings corroborated across independent submissions

- Belair-Edison is the top 311 neighborhood in 2026 at ~24,800 requests (Nwokelo, Hall, Joseph, Eboh).
- Carrollton Ridge has the most open vacant-building notices at 743–744 (Efferson, E. Stewart, A. Basnet, Joseph, Curbeam).
- Liquor-licensed blocks carry roughly 4x–6x the violent crime of unlicensed blocks (James 4.0x, Rasheed 5.7x, Eze 2.1x–3.2x block-level).
- High-crime neighborhoods do not wait longer for 311 closure once request mix is considered (Wanjiku, Rasheed, Allen, Bigelow, A. Basnet, B. Basnet, James).
- Vacancy and violent crime rank-correlate at about 0.74 (James and Eze independently).

## Cross-cutting issues

- Six go.mod files pin Go 1.27.2 (Wanjiku, Curbeam, Joseph, Reyes, A. Basnet, Williams); they built only because GOTOOLCHAIN=auto downloaded it. Offline on Go 1.24 they fail.
- Eight submissions had no real data in the box; four more ship real data but default to synthetic fixtures (Bigelow, Nwokelo, Lewis, Madu). Arulogun's README mislabels generated data as real exports.
- Six students shipped prebuilt binaries that were wrong-platform, non-executable or killed by macOS (Hall, Efferson, Joseph, Thomas, Williams, Brown). All built from source instead.
- Nicholas Ighade submitted a PDF printout of a Python notebook — not an approved language and nothing runnable.
- AI assistance signals: Allen discloses it; Morton's README contains assistant notes addressed to "the user"; James's 5.4K LOC scope is unusually large but code and README match in detail.