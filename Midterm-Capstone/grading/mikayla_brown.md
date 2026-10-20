# Mikayla Brown — COSC 352 Midterm Capstone Grading Report

**Summary: 46/100 (F).** Clean, idiomatic Rust with a real Builder and Strategy trait and a zero-copy borrowed join, but it runs on two three-row synthetic CSVs ("John Doe", "OFF-001") describing a dataset ("Gun Trace Task Force settlements" with per-officer incident IDs) that does not exist on Open Baltimore. The web dashboard is a static page with hard-coded mock numbers unconnected to the program, the README has no build/run/verify instructions and is truncated mid-sentence, and a 192 MB `target/` build directory with a debug binary was committed.

## Inventory
- Path: `/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/mikayla_brown/baltimore_intelligence/`
- Language: Rust (edition 2021; crate `baltimore-municipal-intelligence-engine`, binary `bmie`; deps anyhow, chrono, clap, csv, rayon, serde, serde_json, tokio)
- Source files (612 LOC): `src/analysis.rs` 305, `src/ingestion.rs` 99, `src/main.rs` 57, `src/models.rs` 50, `src/ui.rs` 49, `src/pipeline.rs` 42, `src/concurrency.rs` 10
- README: yes, 47 lines, truncated (ends inside an unclosed code block at `└── ui.rs`). Run script: `run.sh` (`cargo build --release && exec ./target/release/bmie`). Tests: none (`cargo test` → 0 tests). Late: no.
- Other: `index.html` (18 KB static dashboard), `Cargo.lock`, `.vscode/settings.json`, `.git/` (one commit), **`target/` directory, 192 MB, including `target/debug/bmie` (7.3 MB Mach-O arm64 built with rustc 1.99.0 nightly per `.rustc_info.json`)** — 989 of the 1,004 git-tracked files are build artifacts. No `.gitignore`.
- Data: `data/gttf.csv` (4 lines, 332 B), `data/incidents.csv` (4 lines, 310 B), plus an empty nested `data/data/` directory.

## Datasets & Data Realness
- README claims "two official Open Baltimore datasets (Gun Trace Task Force settlements and BPD GIS incidents)."
- `data/gttf.csv` header `officer_id,officer_name,amount_usd,settlement_date,unit,district,case_type,report_year,discipline_year`; rows: `OFF-001,John Doe,150000.0,2021-05-12,Western,Western,Civil Rights,2020,2022`, `OFF-002,Jane Smith,75000.0,...`, `OFF-001,John Doe,50000.0,...`.
- `data/incidents.csv` header `incident_id,officer_id,incident_date,district,unit,shift,crime_type,latitude,longitude`; rows `INC-101,OFF-001,2020-03-15,Western,Western,Day,Assault,39.2904,-76.6122` etc.
- Assessment: Open Baltimore publishes no "GTTF settlements" dataset, and no Open Baltimore crime/incident layer carries an `officer_id`, `shift`, or `unit` column (the real Part 1/NIBRS schema is `RowID/CCNO, CrimeDateTime, CrimeCode, Description, District, Neighborhood, Latitude, Longitude`). Placeholder names (John Doe / Jane Smith), sequential synthetic IDs, three rows each. The program's own output says "Demonstration inputs are synthetic" (`analysis.rs:303`); `Cargo.toml` describes the crate as an "Offline-first municipal data analytics demonstration"; `index.html` footer says "Demonstration inputs are synthetic."
- No downloader for real data: `--settlements-url/--incidents-url` flags shell out to `curl` but expect the same invented "normalized" columns, and the README gives no URL. No generator either.
- **data_status: simulated.** Row counts: 3 + 3.

## Build Results
Built with `cargo build --release` exactly as `run.sh` does, but with `CARGO_TARGET_DIR` redirected to the scratchpad so the student's committed `target/` was not modified.
```
cargo 1.88.0 / rustc 1.88.0
cargo build --release   -> Finished `release` profile [optimized] target(s) in 11.70s   (exit 0, no warnings)
cargo check             -> Finished `dev` profile in 5.98s (exit 0, no warnings)
cargo clippy --release  -> 1 warning (this `map_or` can be simplified, analysis.rs:21)
```
Clean build first try. The README itself contains no build command; only `run.sh` does.

## Test/Vet/Race Results
- `cargo test --release` -> `running 0 tests ... test result: ok. 0 passed` — **no tests exist.**
- `cargo check` clean; clippy one style nit. Rust ownership model + rayon make data races a compile error; no `unsafe`. race_detector: not_run (not applicable; no dynamic race tool used).
- The shipped prebuilt `target/debug/bmie` is killed by macOS on launch (exit 137 / SIGKILL — unsigned binary from another machine); it is not usable as shipped. (Note: after my launch attempt the file was found missing from the working tree; I restored it unchanged from the student's own git HEAD, verified same size 7,302,952 bytes as in the submitted zip. No student file was otherwise modified.)

## Run Results
Command: equivalent of `./run.sh` — `target/release/bmie` executed from the project directory (default `--data-dir data`). Exit 0, **0.19 s wall, max RSS 3.2 MB**. Complete stdout (`<scratch>/mikayla/run.txt`):
```
Baltimore Municipal Intelligence Engine
Records: 3 settlements | 3 incidents
Settlement source: local data/gttf.csv
Incident source: local data/incidents.csv

Q1 · Settlement cost by district
  Central                    $75000  ██████████████████████████████
  $75000 settlements; 1 incidents
  Western                   $200000  ██████████████████████████████
  $200000 settlements; 2 incidents

Q2 · Units/shifts in matched records (association, not correlation)
  Western / Day                 2.0  █
  2 matched pairs, 1 distinct incidents, $200000 associated payout
  Western / Night               2.0  █
  2 matched pairs, 1 distinct incidents, $200000 associated payout
  Central / Evening             1.0  █
  1 matched pairs, 1 distinct incidents, $75000 associated payout

Q3 · Reporting and discipline timeline
  2020                          1.0  █
  1 modeled misconduct reports; 0 modeled disciplinary actions
  2021                          1.0  █
  1 modeled misconduct reports; 0 modeled disciplinary actions
  2022                          1.0  █
  1 modeled misconduct reports; 1 modeled disciplinary actions
  2023                          0.0  █
  0 modeled misconduct reports; 2 modeled disciplinary actions

Q4 · Recurring officer identifiers
  OFF-001                       4.0  ██
  4 source records; $200000 modeled settlement cost; 1.5x mean officer cost
  OFF-002                       2.0  █
  2 source records; $75000 modeled settlement cost; 0.5x mean officer cost

Q5 · District resource allocation risk index
  Western                     100.0  ██████████
  100.0/100; $200000 liability, 2 incidents
  Central                      42.5  ███████
  42.5/100; $75000 liability, 1 incidents

Data limitations
Demonstration inputs are synthetic. Statistics are descriptive and do not establish causal relationships. ...
```
`--district Western` filter also works. Q1 bar widths are both full (30 blocks) because the bar is `sqrt(value)` clamped to 30, so $75K and $200K render identically — the chart does not convey the ratio. `index.html` was opened for inspection: it renders only the hard-coded `mockData` object (`index.html:141-209`) including an "Eastern District" and officer "OFF-042" that appear in neither CSV nor program output; the Rust program never writes anything the page reads.

## Concurrency Analysis
- `ingestion.rs:78-81`: `rayon::join` runs the two CSV loads in parallel (two closures).
- `pipeline.rs:24-39` via `concurrency.rs:5-10`: `par_iter().map().collect()` over the settlements slice (3 items) to look up matching incidents in a pre-built `HashMap`; the index itself is built sequentially.
- `main.rs:41-53`: the whole pipeline runs in a single `tokio::task::spawn_blocking` task under `#[tokio::main]` — tokio adds nothing (one blocking task, no async I/O); the README's "Tokio to handle background tasks" is overstated.
- All five analyses (`analysis.rs:283-305`) run sequentially; no parallel aggregation.
- Correctness: sound by construction (shared data borrowed immutably). Level: **token** — rayon join of two loads plus a par_iter over three rows.

## Memory/Resource Analysis
- Good: `JoinedRecord<'a>` holds `&Settlement`/`&Incident` references (`models.rs:30-33`) so the join is zero-copy; `HashMap<&str, ...>` keys borrow; strategies take slices; `.max(1.0)` guards divide-by-zero. No `unsafe`.
- Weak: `fs::read` loads whole files into memory then parses from the byte buffer (fine at 300 B, not streaming); `Settlement`/`Incident` derive `Clone` and own `String`s; the 1.5–2 MB tokio runtime is spun up for no reason; `Command::new("curl")` spawns an external process rather than using an HTTP client. README offers only buzzwords ("crazy fast, memory-safe"). Some evidence: 5/10.

## Architecture & Code Quality
- Seven small modules with clear responsibilities (ingestion / pipeline / analysis / ui / models / concurrency). `QueryConfig::builder().district(..).build()` is a real (if tiny) Builder (`analysis.rs:4-36`); `trait AnalysisStrategy` with five unit-struct implementors dispatched through `&dyn` (`analysis.rs:38-305`) is a real Strategy; the pipeline sequence in `main.rs` is explicit. Idiomatic Rust (iterators, `BTreeMap` for stable ordering, `anyhow` contexts, clap derive). Clippy nearly clean.
- Weaknesses: no tests; `concurrency.rs` is a 10-line wrapper around `par_iter`; `run_all` indexes `strategies[0..4]` by hand instead of iterating; everything in one crate with no lib target; committed `target/` (192 MB) and `.vscode/`; empty `data/data/` dir; `index.html` duplicates the analysis in hard-coded JS rather than consuming program output (the program depends on `serde_json` but never emits JSON).
- README style (" super fast", "crazy fast", "cool ASCII charts", LaTeX `$\rightarrow$` in Markdown) reads like generated marketing text and is truncated; it does not describe how to build, run, or verify anything.

## Stakeholder Questions & Value Assessment
Stakeholders: "City leaders" only; no office named.
1. Settlement cost by district — settlements summed per district, incidents counted per district; two single-dataset aggregates printed side by side.
2. Units/shifts in matched records — the only true cross-dataset question (join on `officer_id`), but the "matched pairs" double-count because every settlement × incident for the same officer is a pair.
3. Reporting vs discipline timeline — settlements only (single-dataset).
4. Recurring officer identifiers — counts records per officer across both files (cross-dataset count, trivially).
5. District risk index — 0.6×cost + 0.4×incidents normalized; both datasets, but a made-up weighting over invented numbers.
Value: none for actual municipal stakeholders — the data is fictional, the dataset itself does not exist on Open Baltimore, and three rows per side cannot support any of the stated conclusions. The topic (misconduct liability by district) would be interesting if real settlement data were paired with a real BPD dataset, but the schema assumed (officer-level incidents) is not public. The companion dashboard shows numbers the program never computed.

## Issues
- Fully synthetic data (3 + 3 rows, John Doe/Jane Smith, OFF-001); the claimed "Gun Trace Task Force settlements" and officer-linked "BPD GIS incidents" are not Open Baltimore datasets; README's claim that they are "official Open Baltimore datasets" is false.
- No way to obtain real data: URL flags expect the same invented schema; no generator; README gives no source.
- README has no macOS build/execute/verify instructions and is truncated mid-code-block; only `run.sh` documents the build.
- 192 MB `target/` directory (989 build-artifact files, 7.3 MB debug binary built with a nightly rustc 1.99) committed and shipped; the shipped binary is SIGKILLed on launch. No `.gitignore`.
- `index.html` dashboard is static mock data (`mockData`) unconnected to the Rust program; includes an "Eastern District" and "OFF-042" that exist nowhere else.
- Token concurrency: `rayon::join` of two loads + `par_iter` over three rows; analyses sequential; tokio runtime used for a single `spawn_blocking`.
- No tests; `serde_json` dependency unused; empty `data/data/` directory; stray `.vscode/` settings.
- Q1 bar chart renders $75K and $200K with identical full-width bars (sqrt + clamp).

## Score Table
| Criterion | Pts | Justification |
|---|---|---|
| build | 15/15 | `cargo build --release` (the run.sh command) compiles cleanly with zero warnings on first try. |
| real_data | 0/15 | Two three-row synthetic CSVs with placeholder names describing a dataset that does not exist on Open Baltimore; no downloader or generator for real data. |
| concurrency | 5/15 | `rayon::join` for two file loads and a `par_iter` over three settlement rows; analysis sequential; tokio is a no-op wrapper. |
| memory | 5/10 | Borrowed zero-copy `JoinedRecord<'a>` and `&str` map keys show ownership thinking; whole-file reads, needless tokio runtime, no documentation beyond slogans. |
| architecture | 11/15 | Seven clean idiomatic modules with a real Builder and Strategy trait; no tests, hand-indexed strategies, committed build artifacts, dashboard disconnected from program. |
| stakeholder | 4/15 | Five questions but only Q2 is a true join; the rest are side-by-side counts; "city leaders" unnamed; answers over fictional 3-row data; dashboard shows fabricated numbers. |
| runs_results | 5/10 | Runs to completion in 0.19 s with five sections, but only on synthetic data; Q1 chart uninformative. |
| readme | 1/5 | Present but truncated, no compile/execute/verify steps, inaccurate dataset claims; run.sh exists. |
| **Total** | **46/100** | **F** |

## Final Grade Rationale
The Rust itself is competent — it compiles warning-free, uses genuine Builder and Strategy patterns, and borrows rather than copies in the join — which earns full build credit and a solid architecture score. But the project does not meet the assignment's substance: there is no real Open Baltimore data (the named dataset does not exist and the files are six invented rows), concurrency is a rayon join over two tiny loads, the web dashboard is a static mock with numbers the engine never produced, the README gives no build/run/verify instructions, and 192 MB of build artifacts were shipped. 46/100, F.
