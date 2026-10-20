# Baltimore GodEye

A Rust program that ingests five raw datasets from
[Open Baltimore](https://data.baltimorecity.gov/), joins them by neighborhood,
street block and distance, and answers five research questions for city
leaders. It prints a terminal report (tables, ASCII bar charts and two ASCII
city maps) and writes the same report as HTML and JSON. On an Apple-silicon Mac
it processes 1.25 million rows (465 MB) in about half a second.

## Build and run on macOS

```bash
brew install rust              # skip if `cargo --version` works (Rust 1.82+)
cargo run --release            # builds, runs, prints the report
```

The five raw datasets are included in `data/`, so no download is needed. To
fetch fresh copies from the portal instead, delete `data/` and run
`sh scripts/fetch_data.sh` (~465 MB).

The report is also written to `output/report.html` (open with
`open output/report.html`), `output/report.txt` and `output/report.json`.

Options (`cargo run --release -- --help`):

```text
--data-dir <DIR>     input directory              [default: data]
--out-dir <DIR>      output directory             [default: output]
--threads <N>        worker threads               [default: all cores]
--camera-radius <M>  camera coverage in metres    [default: 150]
--top <N>            rows per ranking table       [default: 10]
--question <ID>      run only q1..q5 (repeatable)
--serve [PORT]       serve the HTML report on http://127.0.0.1:PORT [default: 8080]
--no-color, --quiet
```

## How to verify

1. **Tests** (28, no data needed): `cargo test --release`. They cover the
   parallel CSV reader, address normalisation, the spatial grid, matching,
   statistics, the maps, all three renderers, and all five questions on a small
   hand-built city with known answers. `cargo clippy --release --all-targets`
   reports no warnings.
2. **Row counts.** The report's first table lists rows read per file; each is
   one less than `wc -l data/*.csv` (the header).
3. **Concurrency is deterministic.** One thread and all cores give identical
   results; only the timings differ:

```bash
cargo run --release -- --quiet --threads 1 --out-dir /tmp/one
cargo run --release -- --quiet --out-dir /tmp/all
diff <(grep -v -e Pipeline -e threads -e '\.csv' -e geojson /tmp/one/report.txt) \
     <(grep -v -e Pipeline -e threads -e '\.csv' -e geojson /tmp/all/report.txt) && echo "identical results"
```

On a 10-core Mac: 1.3 s with one thread, 0.46 s with all cores.

## Datasets

| Dataset | File | Rows | Join keys |
|---|---|---|---|
| NIBRS Group A Crime (2022–present) | `crime_nibrs.csv` | 276,007 | neighborhood, block address, lat/lon |
| 311 Service Requests 2025 | `service_requests_311_2025.csv` | 931,324 | neighborhood, address, lat/lon |
| Vacant Building Notices | `vacant_building_notices.csv` | 11,445 | neighborhood, address |
| Liquor Licenses | `liquor_licenses.csv` | 30,968 | address only |
| CitiWatch CCTV Cameras | `citiwatch_cameras.geojson` | 861 | lon/lat only |

Counts are from 12 October 2026; the portal updates weekly. No two datasets
share a clean key, so three joins are used: **by name** (normalised
neighborhood names), **by address** (`1601-03 PRATT STREET EAST` and
`1611 E PRATT ST, Baltimore City` both become the block `1600 E PRATT ST`),
and **by distance** (coordinates projected to metres in a spatial grid).

## Research questions

| | Question | Stakeholder | Joins | Headline result |
|---|---|---|---|---|
| Q1 | Where do vacant buildings and violent crime overlap? | Mayor's Office, Housing | vacants + crime | Spearman 0.74 across 290 neighborhoods; blocks with a vacant have 2.1x the violent crime |
| Q2 | Do blocks with liquor outlets see more violent crime? | Liquor Board, Police | liquor + crime | 5.5% of blocks, 17.6% of violent crime; higher in 74 of 79 neighborhoods |
| Q3 | How much violent crime is beyond camera range, and where should new cameras go? | Police, MONSE | cameras + crime | 58.5% is out of range; map and table of ten candidate sites |
| Q4 | Where is blight reported, and do high-vacancy areas wait longer? | Public Works, Housing | 311 + vacants | Spearman 0.90 with vacancy, but no slower service (4.2 vs 5.0 days) |
| Q5 | Which neighborhoods need coordinated investment first? | Mayor's Office, Planning | all five | Downtown, Broadway East, Sandtown-Winchester, Carrollton Ridge, Upton |

Q3 and Q5 include a **map of the city** drawn in the terminal and the HTML
report: Q3 shades each 400 m square by out-of-range violent crime and numbers
the candidate camera sites; Q5 shades each square by its neighborhood's
priority score and numbers the top five. No boundary file is used: each
square's neighborhood is the majority vote of the geocoded records inside it,
so the city's real outline emerges from the data.

Excerpt of the Q5 map around central Baltimore (north is up, each square is
400 m):

```text
    ░░░░▒▒▓▓▒▒  ░░▓▓▓▓████████████████▓▓▓▓▓▓██▓▓▓▓██████ 2████████░░▒▒▓▓▓▓  ▓▓▓▓▓▓▓▓
    ░░░░░░░░▓▓▓▓░░░░██████████████████▓▓▓▓▓▓██████████████████████░░▒▒▓▓▓▓▓▓▓▓▓▓▓▓
    ░░░░░░░░▓▓████▓▓████▓▓▓▓▓▓████ 3██████▓▓██████████▓▓▓▓▓▓▓▓▓▓░░░░▒▒▒▒▓▓▓▓▓▓▓▓▓▓▓▓
    ░░░░▒▒▒▒▒▒▒▒██▓▓▓▓████▓▓▓▓████████ 5██▓▓████▒▒██████▓▓▓▓██▓▓██▒▒▒▒▒▒▒▒▓▓▓▓▓▓▓▓▓▓
    ░░░░▒▒▒▒▒▒▒▒██▓▓▓▓████▓▓▓▓██████████▒▒▓▓████▒▒██████▓▓▓▓████████▒▒▒▒▓▓▓▓▓▓▒▒▒▒▓▓
    ▒▒░░▒▒▒▒▒▒████████████▓▓████████████████ 1████▓▓▓▓▓▓▓▓▒▒████████▒▒▓▓▓▓▒▒▓▓▓▓▒▒▒▒
    ▒▒░░░░░░▒▒▒▒  ████▓▓████████████████▒▒████████▓▓▓▓██▓▓▓▓▒▒▒▒██████▓▓▓▓▒▒▓▓▓▓▓▓▒▒
    ▒▒▒▒░░▒▒▒▒████████▓▓▓▓██████████████▒▒▒▒▒▒▓▓▓▓▓▓▓▓████████████▓▓▓▓▓▓▓▓▓▓▓▓▓▓▒▒▒▒
    ▒▒▒▒▒▒▒▒██████████▓▓▓▓████ 4██████████░░░░████▓▓░░██████████████▓▓▓▓▓▓▒▒▒▒▒▒▒▒▒▒

    ░░ bottom quarter   ▒▒ second   ▓▓ third   ██ top quarter of the Priority Index
     1 Downtown   2 Broadway East   3 Sandtown-Winchester   4 Carrollton Ridge   5 Upton
```

Run `cargo run --release -- --question q3 --question q5` to see both full maps.

Limits, stated in the report: figures are counts, not per-resident rates;
correlation is not causation; 311 "closed" is the agency's own status.

## Architecture

```text
Stage 1  ingest     5 files at once; per file: reader ─▶ bounded channel ─▶ N parsers
Stage 2  match      MatchChain fills in each record's neighborhood
Stage 3  aggregate  per-neighborhood and per-block tables
Stage 4  analyse    5 Question objects run in parallel
Report   Terminal / HTML / JSON renderers
```

| Path | Responsibility |
|---|---|
| `src/main.rs`, `src/cli.rs` | entry point, arguments, writing files |
| `src/pipeline.rs` | `PipelineBuilder` and the four stages |
| `src/ingest/` | parallel CSV reader, one schema per dataset, GeoJSON |
| `src/normalize.rs`, `src/geo.rs` | address/time reformatting; projection and spatial grid |
| `src/matching.rs` | `MatchStrategy` implementations and `MatchChain` |
| `src/analysis/` | aggregation, the five questions, the city map grid |
| `src/report/` | report model and the three renderers |

**Design patterns.**
- **Pipeline:** four stages, each taking the previous one's output by value.
- **Builder:** `PipelineBuilder` validates options once, so an invalid
  `Pipeline` cannot exist.
- **Strategy:** `MatchStrategy` (name, spatial cell, block address), `Question`
  and `Renderer`.
- **Factory:** `report::renderer(format)` and `analysis::questions(ids)`
  return boxed trait objects.

## Concurrency

| Where | Primitive |
|---|---|
| Five files loaded at once | `std::thread::scope` (borrows without `Arc`) |
| Reader to parsers | bounded `crossbeam_channel`, so a slow parser blocks the reader instead of buffering the file |
| Shared vocabularies | `RwLock<HashMap>` interner; the write lock is only taken for new keys |
| Matching, aggregation | rayon `par_iter_mut`, `fold`/`reduce`, `join` |
| Block counters | `Vec<AtomicU32>` with lock-free `fetch_add` |
| Map neighborhood vote | `par_sort_unstable` on packed `u32` keys |
| The five questions | `par_iter` over `Box<dyn Question>` sharing `&Context` |

Results are deterministic: ties break by name, and sums run in a fixed order.

## Memory and resource management

- **Compact records:** a ~400-byte 311 row becomes a 24-byte `Copy` struct
  (interned ids, `u32` timestamps, `f32` coordinates) in one contiguous `Vec`.
- **Bounded memory:** files stream in 4 MiB chunks, and buffers are recycled.
  Peak memory is about 180 MB for 465 MB of input.
- **No per-row allocation:** reused `ByteRecord`s and borrowed `&str` fields.
- **Zero-cost abstractions:** generic `ingest_csv::<S: CsvSchema>` is
  monomorphised. `dyn` is used only where calls are rare.
- **Ownership instead of GC:** each stage frees its inputs at a known point;
  the final `DataStore` is shared immutably by `&`.
- **Safety:** no `unsafe`; `Send`/`Sync` rule out data races; errors are
  `Result`s, and bad rows are counted, not panicked on.

Dependencies: `csv`, `serde`/`serde_json`, `rayon`, `crossbeam-channel`.
Everything else (timestamps, spatial index, statistics, HTML, HTTP server) uses
only the standard library.

## Packaging

The submission zip includes `data/` (the datasets compress to about 82 MB) and
leaves out build products and generated reports. From the folder above the
project:

```bash
zip -r -9 baltimore-godeye.zip baltimore-godeye -x 'baltimore-godeye/target/*' 'baltimore-godeye/output/*' '*.DS_Store'
```
