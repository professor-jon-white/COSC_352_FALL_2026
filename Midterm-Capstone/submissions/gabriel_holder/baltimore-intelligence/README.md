# Baltimore Neighborhood Intelligence (Go)

## Purpose
A terminal-based Go pipeline that joins two Baltimore City Open Data indicators by Community Statistical Area (CSA):
- Violent Crime Rate per 1,000 Residents: https://data.baltimorecity.gov/maps/ab03385abf3b4f50aec0b090caa8877a
- Neighborhood Businesses per 1,000 Residents: https://data.baltimorecity.gov/maps/1af8ce39ea9448c78948db77e286bdeb/about

**Important:** `data/crime.csv` and `data/business.csv` contain synthetic demonstration values, not real Baltimore data. Replace both before making city findings. The original portal datasets may need a CSV export of the CSA layer and a column rename. Both must use the same year and geographic units.

## Run
Install Go 1.21 or newer. From this folder:

```sh
go run main.go
```

To run compiled code:
```sh
go build -o baltimore main.go
./baltimore
```

To run against downloaded CSV exports, use explicit columns matching their exact headers:
```sh
go run main.go -crime path/to/crime.csv -business path/to/business.csv -area "Community Statistical Area (CSA)" -crime-col 2023 -business-col 2023 -workers 8 -out output/real-report.json
```

The column header may differ between exports. Inspect the first CSV line and set `-area`, `-crime-col`, and `-business-col`. If the geographic headers differ across files, rename them to `CSA` in each CSV. Only CSA-level rows should be included. If your downloaded layer includes a citywide summary row, remove it to avoid treating the city as a neighborhood.

## Questions answered
1. Which CSAs have the highest violent crime rate?
2. Which CSAs have the lowest business density?
3. How strongly are the two rates associated (Pearson correlation)?
4. Which CSAs score highest on an exploratory crime-plus-low-business priority index?
5. Which CSAs have the largest numerical difference between the two rates?

## Architecture
- CSV readers stream rows, avoiding loading the raw CSVs in memory.
- Two datasets ingest simultaneously using goroutines and a `sync.WaitGroup`.
- Each CSV uses a bounded worker pool with channels to parse records concurrently.
- A single collector goroutine owns each dataset map. This avoids concurrent map writes and data races.
- An inner join matches normalized CSA names.
- Aggregation calculates rates, a Pearson correlation, and a weighted index (70% normalized crime + 30% inverse normalized business density).
- Output renders ASCII tables and bars and writes structured JSON.

## Resource management
- Worker count is bounded by `-workers` (1 to 128).
- Channels are buffered with `4 * workers` slots for backpressure.
- Only parsed area-to-rate maps are retained. Raw CSV records are streamed.
- Go's garbage collector manages heap allocations. `defer Close()` closes files.
- Each goroutine has its own stack, which Go grows dynamically.

## Interpretation and limitations
- The source indicators represent rates per 1,000 residents, not counts of incidents or businesses.
- Different years must not be joined as if contemporaneous.
- Correlation is not causation. This analysis does not establish that businesses change crime.
- Priority weights are illustrative and require stakeholder validation.
- A rate difference is not a causal or economic measure.
- A production deployment should add geographic ID matching, data provenance, duplicate detection, validation, and statistical uncertainty.
