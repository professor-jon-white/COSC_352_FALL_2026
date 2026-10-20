# Baltimore City Data Intelligence Engine

## Project Overview

The Baltimore City Data Intelligence Engine is a project built using Go to analyze public data from Baltimore City. The goal is to look at 311 service requests and vacant building notices to better understand what is happening in different neighborhoods.

The program reads both datasets concurrently and summarizes the information so it is easier to understand. It also compares neighborhoods that appear in both datasets to see how their service request counts and vacant building notices compare.

## Programming Language

- Go (Golang)

## Datasets Used

This project uses two Baltimore City public datasets:

1. **311 Customer Service Requests 2026:** Includes information about service requests, request types, and neighborhoods.
2. **Vacant Building Notices:** Includes information about vacant building notices and neighborhoods.

Both CSV files should be placed inside the `data` folder for the program to run correctly.

## What the Program Does

- Reads and processes both CSV files concurrently using Go goroutines.
- Counts the total number of records in each dataset.
- Identifies records that are missing neighborhood information.
- Finds the most common types of 311 service requests.
- Shows which neighborhoods have the most vacant building notices.
- Lists neighborhoods with the most 311 service requests.
- Compares 311 service requests and vacant building notices in neighborhoods found in both datasets.
- Displays the results in the terminal.

## Research Questions

This project explores five questions that could be useful to Baltimore City residents and local government agencies:

1. What types of 311 service requests are reported most often?
2. Which neighborhoods have the highest number of vacant building notices?
3. Which neighborhoods have the highest number of 311 service requests?
4. Which neighborhoods appear in both datasets, and how do their numbers compare?
5. How many 311 records are missing neighborhood information, and how could that affect the results?

## Software Architecture

The program separates the main tasks of reading, processing, comparing, and displaying data. Each dataset is processed separately, and the results are used to create summaries and compare neighborhoods. This makes the code easier to follow and update.

## Concurrency and Resource Management

The program uses Go goroutines to process the two datasets concurrently. Each dataset is read and summarized separately before the results are used for the final output. The program uses Go's garbage collector to manage memory and checks for CSV reading errors.

Processing the records as they are read helps avoid keeping every raw CSV record in memory at the same time.

## How to Run the Program

### Requirements

- Go installed on your computer.
- Both CSV datasets saved in the `data` folder.
- A terminal or command prompt.

### Run the Application

Open a terminal in the project folder and run:

```bash
go run main.go
```

### Build the Application on macOS

Run:

```bash
go build -o baltimore-data-engine
```

Then execute the program:

```bash
./baltimore-data-engine
```

### Windows Instructions

On Windows, open PowerShell in the project folder and run:

```powershell
go run main.go
```

To build and run a Windows executable:

```powershell
go build -o baltimore-data-engine.exe
.\baltimore-data-engine.exe
```

## How to Verify the Application

After running the program, check that the output includes:

- Total records processed for both datasets.
- The number of records missing neighborhood information.
- The most common 311 service request types.
- The neighborhoods with the most vacant building notices.
- The neighborhoods with the most 311 service requests.
- A comparison of neighborhoods appearing in both datasets.

The program should finish by displaying `Initial data ingestion complete.`

## Current Results

Using the datasets included in the project during testing, the program processed:

- **943,283** 311 service request records.
- **11,445** vacant building notice records.
- **250,620** 311 records with missing neighborhood information.
- **213** neighborhoods appearing in both datasets.

These results are based on the files used during testing and may change if updated datasets are used.

## Limitations

One limitation is that the program only matches neighborhoods when their names are the same in both datasets. Some 311 records are also missing neighborhood information, which can affect the comparison. The results show patterns in the data, but they do not prove that vacant buildings cause more service requests.

## Future Improvements

If I continue working on this project, I would like to add date-based analysis to see how service requests change over time. I would also improve how the program handles neighborhood names and add charts or reports to make the results easier to understand.

## Author

Grace Joseph

COSC 352 Midterm Capstone Project