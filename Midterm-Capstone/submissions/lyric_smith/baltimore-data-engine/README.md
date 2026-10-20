# Baltimore Multi-Paradigmatic Data Processing Engine

**Course:** Programming Languages Capstone
**Language:** Go (Golang)

## Overview

This program ingests two Baltimore City Open Data Portal datasets --
311 Customer Service Requests and Crime Incident reports -- and
produces a cross-dataset intelligence report for fictional municipal
stakeholders (Mayor's Office, Department of Public Works, Department
of Transportation, and City Planning).

## Architecture

The program is split across several files, each with a focused
responsibility:

- models.go -- the ServiceRequest and CrimeIncident data structures,
  plus the normalizeKey join-key helper.
- parsers.go -- the Strategy pattern: a RecordParser interface with
  two concrete implementations, ServiceRequestParser and CrimeParser.
- loader.go -- concurrent CSV ingestion using goroutines and
  sync.WaitGroup, with a sync.Mutex protecting shared results.
- joiner.go -- joins the two datasets on a normalized Neighborhood
  key using a shared, mutex-protected Index built concurrently.
- analysis.go -- the five stakeholder research questions.
- main.go -- wires the pipeline together: load, join, analyze, report.

## Design Patterns Used

- Strategy -- RecordParser interface with two interchangeable
  parser implementations.
- Factory -- NewServiceRequestParser() and NewCrimeParser() hide
  parser construction.
- Pipeline -- main.go runs an explicit, ordered sequence of
  stages: ingest, join, analyze, report.

## Concurrency and Memory Management

- Two goroutines load the 311 and crime CSV files at the same time
  in loader.go, synchronized with sync.WaitGroup.
- Two goroutines build the joined neighborhood index at the same
  time in joiner.go, with a sync.Mutex protecting the shared map.
- CSV rows are streamed one at a time rather than loading the whole
  file into memory at once.
- A malformed row is skipped and counted rather than crashing the
  whole load.

## The Five Stakeholder Questions

1. Neighborhood Burden Ranking (Mayor's Office)
2. Crime vs 311 Correlation (City Planning and Housing)
3. Monthly Trend (Department of Transportation)
4. 311 Complaint Mix (Department of Public Works)
5. Reporting Outliers (Mayor's Office, equity)

## How to Build and Run (macOS)

### Prerequisites

Go 1.19 or newer. Check with:

go version

Install via Homebrew if needed:

brew install go

### Build and run

cd baltimore-data-engine
go run .

This compiles and runs the program in one step, using the sample
data already included in data/311_sample.csv and
data/crime_sample.csv.

To build a standalone binary instead:

go build -o baltimore-data-engine .
./baltimore-data-engine

### Verifying it works

A successful run prints progress for each pipeline stage, followed
by a full report with five numbered sections, and exits normally.

## Data

The included sample CSVs are small, hand-built examples that mirror
the real Open Baltimore column names so the program can be verified
quickly. Real, larger exports from data.baltimorecity.gov use the
same column names and would work with this same code.
