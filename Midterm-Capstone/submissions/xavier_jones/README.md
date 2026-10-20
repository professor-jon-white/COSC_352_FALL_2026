# Baltimore Public Safety Data Processing Engine

A Go capstone inspired by the [Baltimore Public Safety Accountability Dashboard](https://baltimore-psad.github.io/). It analyzes raw Open Baltimore crime and arrest data and produces five municipal workload analyses with tables and an ASCII chart. It is an independent class project; definitions and totals are not those of the official Power BI dashboard.

## Quick start on Windows

Install Go 1.22 or newer from https://go.dev/dl/. Extract this folder, open PowerShell **in this folder**, and run:

```powershell
go run .
notepad public-safety-report.md
```

The default run analyzes the **included real-data snapshot**, with no network required. To run tests:

```powershell
go test ./...
```

Or use `powershell -File run.ps1`, which tests, builds and runs the program. To run an explicitly synthetic demo:

```powershell
go run . -mode demo -out demo-report.md
```

The demo automatically uses `data/demo` unless you supply another data folder. Do not describe its invented records as Baltimore findings.

## Compile and run on macOS

Install Go 1.22 or newer, open Terminal in this folder, and run:

```sh
sh run.sh
```

The script tests the code, builds `public-safety`, analyzes the included raw snapshot and writes `public-safety-report.md`. No third-party Go modules, database, C compiler or credentials are required for this normal build. To build and execute manually:

```sh
go test ./...
go vet ./...
go build -o public-safety .
./public-safety -mode analyze -data data/live -workers 4 -out report.md
```

## Fetch a new public snapshot

The included data covers the source query period January 1, 2025 inclusive through April 1, 2025 exclusive. Both endpoints use their own date fields. Use a **new snapshot folder** when downloading:

```sh
go run . -mode fetch -data data/new-snapshot -start 2025-01-01 -end 2025-04-01 -limit 0 -out new-report.md
```

`-limit 0` requests all available rows in the selected interval, not all years. The default limit is 20,000 per dataset; if used, it selects a non-random source-ID prefix, which can bias every comparison. Manifest and report state available/downloaded counts and truncation. The program refuses to fetch over an existing manifest so a preserved snapshot is not silently replaced. If a fetch fails, choose a fresh directory for your next attempt; do not analyze incomplete files.

To reanalyze offline:

```sh
go run . -mode analyze -data data/new-snapshot -workers 4 -out report.md
```

`-start` and `-end` affect fetching only. Analysis reads the saved manifest. `-workers` must be 1�256. Source schemas or service URLs can change; incompatible fields and HTTP/API errors produce an error exit.

## Five stakeholder questions

1. **Where is recorded crime workload largest, alongside arrest activity?** Full outer join of normalized neighborhood aggregates for geographic coordination.
2. **How do monthly crime and arrest workloads change?** Full outer join of Eastern calendar-month aggregates for staffing review.
3. **What offense mix occurs in the five neighborhoods with the most arrest records?** Select from arrest aggregates, join crime records by neighborhood, then compare source Description proportions with other known neighborhoods for resource planning.
4. **Where do geographic shares of crime records and arrest records differ most?** Joined neighborhood shares with explicit denominators and a signed ASCII chart for workload-distribution review.
5. **What weapon profile and arrest workload occur in the five neighborhoods with the most crime records?** Select crime neighborhoods, join arrest totals and compare source Weapon proportions elsewhere for violence-prevention planning.

These are descriptive analyses, not causal evaluations, clearance rates, crime risk estimates, or policing performance scores. The same geography does not establish a case-level link. Category rows are source offense/victim records, not counts of unique incidents. Arrest rows may include charge records; distinct arrest numbers are also reported for context. Unknown neighborhoods remain in overall tables but are excluded from geographic selection and share denominators. A selected group with no records has zero displayed mix, not an estimated underlying rate.

## Data sources and snapshot provenance

The linked website embeds a Power BI report. This implementation uses public raw data rather than scraping its rendered totals:

- [NIBRS Group A Crime Data catalog](https://www.arcgis.com/home/item.html?id=204beefe92a645d79fdf0969957bbdf8), maintained by Baltimore City.
- Crime API: https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/NIBRS_GroupA_Crime_Data/FeatureServer/0
- [BPD Arrests catalog](https://www.arcgis.com/home/item.html?id=619ec10c14b346f784a5a07bad4c43cd), maintained by Baltimore City.
- Arrest API: https://baltegis.baltimorecity.gov/mapping/rest/services/GeoSpatialized_Tables/Arrest/FeatureServer/0
- [BPD source definitions and reporting caveat](https://www.baltimorepolice.org/crime-stats/open-data).

`data/live/crime.json` and `arrests.json` preserve raw feature attributes returned by the two APIs; no derived dashboard totals are ingested. Geometry is omitted. `manifest.json` records retrieval time, URLs, fields, source SQL filters, row counts and truncation. Download pages are ordered by RowID; missing or repeated object IDs stop a download. A separate count query helps identify incomplete coverage, but records can change between queries: this is not a transactional source snapshot.

The included snapshot has **12,142 crime rows and 4,183 arrest rows**, matching source count queries at retrieval. It contains 10,648 distinct nonempty crime incident IDs and 4,010 distinct nonempty arrest numbers. This is complete for the API query at that retrieval, not a guarantee that all real-world events were reported or correctly exported.

## Interpretation and known data limits

The current snapshots do not support trustworthy case-level linking: crime and arrest incident-number conventions differ, and 3,509 of the 4,183 arrest rows have no usable incident number. Exact normalized ID overlap is a quality diagnostic only; no conversion between identifier systems is invented. All five answers instead use aggregate geographic/time joins.

Arrest neighborhoods are missing in **2,881 rows**, compared with 3 crime rows. Consequently geographic arrest comparisons describe a substantially incomplete subset. The report displays this coverage before the questions. Never describe the neighborhood share gap as a clearance rate or a measure of police adequacy. Arrest location and crime location refer to different events, and arrests can relate to crimes outside the chosen interval.

Neighborhood names are case/whitespace normalized without fuzzy spelling or spatial inference. Month labels use America/New_York, with Go's embedded timezone database for portability. ArcGIS SQL timestamp filters follow source service semantics; preserve the manifest and inspect date boundaries before comparing with another system. BPD documents reporting-system translation issues beginning May 2021. NIBRS and legacy Part 1 definitions also differ; do not combine them without reconciling definitions. No victim or arrestee identities, addresses or demographics are printed in the report.

## Architecture and engineering requirements

This implements a **Pipeline** with explicit stages:

- `ingest.go`: concurrent source downloads, raw JSON parsing, schema checks and snapshot metadata.
- `normalize.go`: source value normalization, calendar grouping and shared helpers.
- `analysis.go`: concurrent aggregation, hash-index diagnostics and reduction.
- `report.go`: cross-dataset aggregate joins, selections, comparisons and Markdown/ASCII output.
- `main.go`: CLI orchestration; `types.go`: data model; `demo.go`: synthetic fixtures.

Two goroutines download the datasets concurrently, with HTTP timeouts and cancellation if either fails. Two goroutines parse raw datasets. A configurable worker pool processes crime records through a bounded channel; workers own separate maps, then one reducer merges them. The arrest-ID index is read-only while workers run. Neighborhood matching and report selections use hash maps; deterministic sorts break ties by name. No concurrent writes target shared worker maps. Arrest aggregation and the final neighborhood/category cross-tabs are sequential stages.

Time is O(C+A) expected for ingestion, indexing and aggregation, plus sorting G groups and output categories. Raw JSON is held in memory; this is not an unlimited-size streaming engine. Memory is O(C+A+W*G+K), where W is workers, G is grouping cardinality and K is distinct incident/arrest IDs and neighborhood/category combinations. The bounded channel limits queued work, not total source storage. Go garbage collection owns heap allocations, including decoded interface values and maps; extra workers add maps, goroutine stacks and GC pressure. Choose a shorter interval, a labeled limit, or fewer workers on memory-constrained machines. Runtime output reports analysis time and current Go heap, not peak RSS or download time. More workers need not improve small workloads.

## Verification

```sh
go test ./...
go vet ./...
go build .
```

Tests check worker-count consistency, normalized identifiers, duplicate arrest numbers, missing IDs, source pagination/filter propagation, repeated-page rejection, schema errors, neighborhood selection/profile denominators and Eastern month boundaries. On a platform with a C compiler, optionally run `go test -race ./...`. The regular build does not need that compiler.

The included live report, synthetic demo and source snapshots make execution reviewable offline. macOS cross-compilation checks both Intel and Apple Silicon; a cross-build is not a substitute for executing on a Mac. Review the generated report, explain the coverage limits, and run on macOS before submission if available.

## Submission

Submit the source, README, run scripts, tests, raw snapshots and reports through Canvas. This folder replaces the earlier housing/311 topic; that original project remains separately available. Run `go run .` in this new project folder, not the original `baltimore-engine` folder.
