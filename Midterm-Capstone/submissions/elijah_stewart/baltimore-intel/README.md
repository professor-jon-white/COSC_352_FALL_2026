# Baltimore Municipal Intelligence

A Go command-line project that reads two real Baltimore datasets, processes them concurrently, and prints five stakeholder-oriented questions with terminal bar charts. It uses streaming CSV reads, so it does not need to keep hundreds of megabytes of raw CSV rows in memory.

## Datasets used

1. **Baltimore 311 Customer Service Requests** — uses `SRType` and `Neighborhood` (with fallback location fields if neighborhood is blank).
2. **BPD Arrests** — uses `IncidentOffence` / `ChargeDescription` and `Neighborhood` (with district/location fallbacks).

The arrest file records arrests, not every crime reported or committed. The 311 dataset and arrest dataset may cover different date ranges. The combined chart is exploratory and is not a crime rate, causal claim, or apples-to-apples comparison.

## Requirements

- Go 1.21 or newer
- The two CSV files you downloaded from Baltimore's open-data sources
- No external Go packages are required

## Put the CSV files where the program can find them

The simplest option is to put these files in the project root (the same folder as `main.go`):

- `311_Customer_Service_Requests_Yearly_7199422805573403920.csv`
- `Arrest.csv`

The large source CSVs are intentionally **not bundled** in this project ZIP, because together they are hundreds of megabytes. Keep the originals you downloaded.

Alternatively, provide their paths with `--311` and `--arrests`.

## Run on Windows (PowerShell / VS Code terminal)

From the `baltimore-intel` project folder, run:

```powershell
go test ./...
go vet ./...
go run . --311 "C:\path\to\311_Customer_Service_Requests_Yearly_7199422805573403920.csv" --arrests "C:\path\to\Arrest.csv"
```

Replace the example paths with the actual paths on your computer. If you put both CSVs in the project root, simply run:

```powershell
go run .
```

## Run on macOS or Linux

From this project folder:

```sh
go test ./...
go vet ./...
go run . --311 "/path/to/311_Customer_Service_Requests_Yearly_7199422805573403920.csv" --arrests "/path/to/Arrest.csv"
```

If both files are in the project root, run `go run .`.

## Offline sample mode

To verify that the program works without the large source CSVs:

```sh
go test ./...
go run . --offline
```

The offline output uses small **synthetic sample data**, not the real Baltimore datasets. Use the real CSV command above for your analysis.

## Five stakeholder questions

1. Which 311 service-request types appear most often?
2. Which neighborhoods have the most 311 requests?
3. Which offenses or charge descriptions appear most often in arrest records?
4. Which neighborhoods or districts appear most often in arrest records?
5. Which matching area labels have records in both datasets? (combined counts; exploratory)

## Architecture and concurrency

- Each CSV is read and summarized row-by-row, keeping aggregate counts rather than storing every raw row in memory.
- Two goroutines process the datasets concurrently, coordinated with a `sync.WaitGroup`.
- CSV readers tolerate variable-width records and skip malformed rows.
- Files and HTTP response bodies are closed after processing. File paths are recommended for the downloaded datasets; direct HTTP CSV URLs are also accepted.
- The program prints a count and up to eight rows for each chart.

## Interpretation and limitations

Counts depend on the selected dataset versions and date ranges. Blank area fields may fall back to another field, so the fifth chart is only an exploratory comparison of labels and should be treated cautiously. The charts are descriptive, do not establish causation, and should not be interpreted as rates or evidence that service requests cause arrests.
