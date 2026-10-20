
# Baltimore City Intelligence Engine

## Project Overview

The Baltimore City Intelligence Engine is a concurrent
data-analysis application developed in Go.

It analyzes two Baltimore City public datasets to examine
patterns in municipal service requests and reported crime.

## Data Sources

1. Baltimore City 311 Service Requests
   - File: data/service_requests.csv
   - Important fields: Neighborhood, CreatedDate, SRType

2. Baltimore City NIBRS Group A Crime Data
   - File: data/crime.csv
   - Important fields: Neighborhood, CrimeDateTime

Source: Baltimore City Open Data.

## Research Questions

1. Which neighborhoods have the highest combined volume
   of 311 service requests and reported crime records?

2. What is the correlation between neighborhood-level
   311 request counts and crime record counts?

3. Which neighborhoods have above-average counts in
   both datasets?

4. What types of 311 service requests are most common
   in those high-activity neighborhoods?

5. How do 311 request and crime record counts change
   across months?

## Analysis Period

January 2026 through September 2026.

October 2026 is excluded because it is an incomplete
month at the time of analysis.

## Running the Application

Requirements:
- Go toolchain compatible with the version in go.mod
- Both CSV datasets in the data/ folder

On Windows:

    go run .

On macOS:

    bash run.sh

## Testing

Run:

    go test -count=1 ./...

## Concurrency Architecture

Two goroutines independently stream the 311 and crime
CSV datasets.

A results channel collects their outputs, while a
sync.WaitGroup coordinates completion.

The application also uses a bounded worker pool to
aggregate neighborhood and monthly statistics.

Each aggregation worker maintains local maps, avoiding
concurrent writes to shared maps. Results are merged
after all workers finish.

## Resource Management

- CSV files are read one record at a time.
- File handles are closed with defer.
- Channels coordinate processing.
- Worker counts are bounded.
- Aggregated counts are stored in maps rather than
  retaining all raw CSV rows.

## Findings

The application analyzes 279 neighborhoods during the
January-September 2026 reporting period.

The neighborhood-level Pearson correlation between
311 requests and crime records is approximately 0.777.

Belair-Edison has the highest combined event count in
the current report.

The most common 311 category in neighborhoods above
both dataset means is SW-RAT RUBOUT PROACTIVE.

## Limitations

- Correlation does not imply causation.
- Raw event counts are not adjusted for population.
- Differences in reporting practices may affect results.
- Neighborhood names are standardized by trimming
  whitespace and converting to uppercase.
- Results depend on the supplied CSV snapshots.
- Crime records are counted as CSV rows, rather than
  summing the Total_Incidents column.
- Monthly and neighborhood comparisons are based on
  aggregate counts, not individual event matching.
