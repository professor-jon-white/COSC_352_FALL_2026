# Baltimore City Data Analysis Engine

## Project Overview

This is a program for data processing implemented in Go programming language. The program is designed to analyze two public data sets provided by the Baltimore City Open Data Portal to aid the municipal government to evaluate the state of neighborhoods.

The application makes use of concurrency, CSV stream processing, pool of workers and neighborhood data joining techniques. The results are shown on the terminal via summary statements and ASCII graphs.

## Programming Language

Go (GoLang)

## Datasets

Source: https://data.baltimorecity.gov/

1. **311 Service Requests** (`data/service_requests.csv`) — contains records of service requests submitted to Baltimore City.
2. **Vacant Property Records** (`data/vacant_properties.csv`) — contains vacant building notice records.

Both datasets are matched using the `Neighborhood` column. Neighborhood names are standardized by trimming whitespace and converting them to uppercase.

## Research Questions

1. Which neighborhoods have the highest number of 311 service requests?
2. Which neighborhoods have the most vacant property records?
3. What is the relationship between service requests and vacant property records?
4. Which neighborhoods have the highest ratio of service requests to vacant property records?
5. Which neighborhoods should municipal leaders prioritize based on both datasets?

## Software Architecture

The project uses the Pipeline design pattern to separate ingestion, processing, joining, and output.

- `main.go`: Application entry point, research question calculations, and output formatting.
- `ingestion.go`: Streams CSV records and extracts neighborhood information.
- `processing.go`: Implements concurrent aggregation and dataset joining using worker pools.
- `run.sh`: Builds and executes the application on macOS.

## Concurrency and Memory Management

The application uses Go goroutines, channels, and synchronization with `sync.WaitGroup`.

One row is read from CSV files at a time rather than loading the entire file to the memory. Bounded channels regulate the movement of rows between processors.

The aggregation workers have their own maps and there is no case of concurrent writes to the shared maps. In the joining process, the workers only read from the finished maps.

Go's garbage collector manages heap memory automatically.

## Priority Score

The application calculates a neighborhood priority score using:

Priority Score = 50 × (Neighborhood 311 Requests / Maximum Neighborhood 311 Requests) + 50 × (Neighborhood Vacant Records / Maximum Neighborhood Vacant Records)

Both datasets contribute equally to the score.

The score is intended as a preliminary screening tool and does not establish causation or replace population-adjusted analysis.

## Requirements

- Go installed
- macOS or Windows
- Both CSV datasets included in the `data` directory

No external Go libraries are required.

## Running on macOS

Open Terminal, navigate to the project folder, and run:

```bash
chmod +x run.sh
./run.sh
```

Or, run:

```bash
go run .
```

## Running on Windows

Open the project folder in VS Code, open the terminal, and run:

```powershell
go run .
```

## Build Verification

To verify the project compiles successfully:

```bash
go build .
```

To run the project's tests, if test files have been added:

```bash
go test ./...
```

A successful build should complete without compilation errors.

## Example Results

Using the supplied dataset snapshots, the preliminary priority ranking was:

| Rank | Neighborhood | Priority Score |
|---|---|---|
| 1 | Broadway East | 68.09 |
| 2 | Carrollton Ridge | 63.89 |
| 3 | Sandtown-Winchester | 58.11 |
| 4 | Belair-Edison | 54.97 |
| 5 | Harlem Park | 41.77 |

The correlation coefficient between neighborhood-level 311 request counts and vacant property counts was approximately 0.451.

## Limitations

The analysis relies on record counts for each neighborhood. Variation between neighborhood populations, time of reporting, level of completeness, and request frequency may impact any comparisons made.

Vacant property records represent notices in the dataset and should not automatically be interpreted as unique currently vacant buildings.

The scores indicate patterns in the data; however, they cannot be used as evidence to prove that vacant properties lead to service requests.

## Conclusion

The Baltimore City Data Analysis Engine is one such example where Go language can be used for concurrent processing of several public datasets and gaining valuable insights for municipal planning. This tool gives five different research analysis, terminal output in text format, and ASCII visualization with the help of a modular approach to data processing.