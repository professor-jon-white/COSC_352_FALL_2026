# Baltimore Intel

A Go command-line tool that ingests two Open Baltimore datasets (Part 1 Crime Data and 311 Customer Service Requests), joins them by neighborhood, and answers five research questions for municipal stakeholders using concurrent processing.

Developed with Go 1.27.2 on Windows. macOS verification:Build, `go test ./...`, `go test -race ./...`, the sample run, `-verify`, and `run.sh` also pass on macOS (macos-latest) via GitHub Actions; see `.github/workflows/test.yml`.

## 1. macOS setup, build, run, verify

Prerequisite: Go 1.21 or newer (the code uses the built-in `min` function and generics). No third-party dependencies are used.

```bash
brew install go
go version
```

### Quick start (one command)

From the project folder:

```bash
bash run.sh
```

`run.sh` checks that Go is installed, builds the program, runs `go test -race ./...`, runs the analysis, and checks that the parallel join equals the sequential join. Use `bash run.sh` rather than `./run.sh`, because a zip file may not preserve the executable bit.

### Manual commands

```bash
# Build
go build -o baltimore-intel ./cmd

# Run on the bundled sample (works offline, finishes in well under a second)
./baltimore-intel

# Verify: tests, race detector, and parallel-vs-sequential join check
go test ./...
go test -race ./...
./baltimore-intel -verify
```

Expected: every test package prints `ok`, and the run prints
`VERIFY: parallel join is identical to sequential join`.

A full report is also written to `report.txt` in the current folder.

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `-data` | `./data` | Folder holding `crime.csv` and `311_<year>.csv`. If `./data/crime.csv` is missing, the bundled `./data/sample` is used. |
| `-year` | `2024` | Year to analyze. |
| `-mode` | `parallel` | `parallel` or `sequential`. |
| `-verify` | off | Check that the parallel join equals the sequential join. |

### Running on the full data (optional)

Download from https://data.baltimorecity.gov/ :
- "Part 1 Crime Data (Legacy SRS)" as CSV, saved as `data/crime.csv`
- "311 Customer Service Requests 2024" as CSV, saved as `data/311_2024.csv`

Then run `./baltimore-intel`. The full 311 file is about 320 MB, and a run takes a few