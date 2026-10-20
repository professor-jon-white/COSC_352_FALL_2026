# Baltimore Civic Pulse

**Mid-Term Capstone Project: Multi-Paradigmatic Data Processing Engine**  
Implementation language: **Go** (Go 1.22 or newer; standard library only)

## What it does

This command-line application fetches two raw GeoJSON sources from Open Baltimore, parses their records with a bounded Go worker pool, joins them by neighborhood, computes five descriptive stakeholder analyses, and writes a Markdown report, CSV table, and JSON summary. It also draws ASCII bar charts in the terminal and report.

The selected stakeholders are the Baltimore Mayor's Office and the Department of Housing and Community Development. The questions are:

1. Which neighborhoods have the strongest combined signal of 2026 311 request volume and vacant-building rehab locations?
2. Where are the largest concentrations of vacant-building rehab locations?
3. Which neighborhoods have the most 311 requests whose recorded status is not closed, complete, resolved, or cancelled?
4. Which 311 service request types are relatively more common in high-rehab neighborhoods?
5. Do neighborhoods rank similarly by 311 request volume and vacant-building rehab locations?

## Data sources

1. [311 Customer Service Requests 2026](https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0). Baltimore City describes this as the current calendar year's requests and says the data is updated daily. The app retrieves all pages available at run time.
2. [Vacant Building Rehabs](https://data.baltimorecity.gov/datasets/baltimore::vacant-building-rehabs). This Baltimore City dataset records locations of vacant buildings that have been rehabilitated and is distributed as GeoJSON.

The program requests the 311 features as GeoJSON with all available fields and geometry (filtering out null neighborhood labels), and downloads the rehab GeoJSON snapshot. It saves the unmodified source responses under `data/raw/` for reproducibility. The records are joined at the neighborhood level using normalized neighborhood labels. This is a geographic group-by join, not a parcel match.

## Architecture and engineering notes

- **Pipeline:** two source producers fetch the feeds concurrently; a fixed number of parser workers process features; a single collector updates the neighborhood aggregates; report writers produce Markdown, CSV, and JSON.
- **Concurrency:** goroutines and bounded channels provide backpressure. `-workers` controls the parser pool (default: the smaller of CPU count and 8). No global mutable state is shared between parser workers.
- **Memory and resource management:** the HTTP client has a request timeout; responses are capped at 64 MiB per page; raw data is paginated; a bounded job queue limits in-flight work; rows are reduced into neighborhood counters rather than retained after parsing. The raw source pages are intentionally kept on disk.
- **Patterns:** source configuration separates providers from ingestion; the producer/worker/collector pipeline separates fetching, parsing, and aggregation; the report writer separates presentation from analysis.
- **Dependencies:** Go standard library only; no external modules, API key, or secrets.

## macOS compile, execute, and verify

Install Go 1.22 or newer from [go.dev/dl](https://go.dev/dl/), then open Terminal and change directory to this project.

```sh
go version
go build -o civicpulse .
./civicpulse -workers 4
```

Or use the included build-and-run script:

```sh
chmod +x run-macos.sh
./run-macos.sh -workers 4
```

The program writes these outputs:

- `reports/analysis.md` — five stakeholder answers, source/method notes, and ASCII charts
- `reports/neighborhoods.csv` — all joined neighborhood measures
- `reports/summary.json` — machine-readable totals and rankings
- `data/raw/requests_page_*.geojson` and `data/raw/vacant_snapshot.geojson` — raw source responses fetched in that run

Verify the build and generated outputs:

```sh
test -x ./civicpulse
test -s reports/analysis.md
test -s reports/neighborhoods.csv
test -s reports/summary.json
find data/raw -name '*.geojson' -type f
```

To replay a previous full run without downloading the sources again, keep its `data/raw/` directory and run:

```sh
go run . -from-raw data/raw
```

To use the bundled raw GeoJSON excerpts without a network connection:

```sh
go run . -offline
```

The bundled excerpts demonstrate the same raw input format and pipeline but are intentionally small. Use the default live run for citywide findings. The live datasets change, so exact totals and rankings may differ across run dates.

## Interpretation limits

Counts of reported 311 requests are not a direct measure of community need; reporting access, awareness, and service practices affect them. The two feeds cover different time windows (calendar-year 2026 versus a rehab inventory), are not normalized by population or housing stock, and may have missing or inconsistent neighborhood labels. “Non-closed” is a broad approximation based on the recorded status value. Correlation and co-location do not establish causation. Use the output to frame follow-up analysis, not as a stand-alone basis for enforcement or resource-allocation decisions.

## Attribution

Data are supplied by the City of Baltimore through Open Baltimore. See the city source pages linked above for dataset descriptions, updates, and current terms. Preserve the source attribution when reusing these records.
