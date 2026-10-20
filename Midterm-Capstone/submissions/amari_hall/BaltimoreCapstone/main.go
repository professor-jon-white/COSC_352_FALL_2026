package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const (
	startMonth = "2026-01"
	endMonth   = "2026-09"
)

type DatasetResult struct {
	Name          string
	Counts        map[string]int
	MonthlyCounts map[string]int
	Types         map[string]int
	Total         int
	InvalidDates  int
	Err           error
}

type MonthRecord struct {
	Neighborhood string
	Month        string
	Count        int
	IsCrime      bool
}

func processDataset(
	name, filename string,
	results chan<- DatasetResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	result := DatasetResult{
		Name:          name,
		Counts:        make(map[string]int),
		MonthlyCounts: make(map[string]int),
		Types:         make(map[string]int),
	}

	file, err := os.Open(filename)
	if err != nil {
		result.Err = err
		results <- result
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		result.Err = err
		results <- result
		return
	}

	columns := make(map[string]int)
	for i, header := range headers {
		header = strings.TrimSpace(
			strings.TrimPrefix(header, "\uFEFF"),
		)
		columns[header] = i
	}

	dateColumn := "CreatedDate"
	if name == "Crime Incidents" {
		dateColumn = "CrimeDateTime"
	}

	neighborhoodIndex, ok := columns["Neighborhood"]
	if !ok {
		result.Err = fmt.Errorf("missing Neighborhood in %s", filename)
		results <- result
		return
	}

	dateIndex, ok := columns[dateColumn]
	if !ok {
		result.Err = fmt.Errorf("missing %s in %s", dateColumn, filename)
		results <- result
		return
	}

	typeIndex := -1
	if name == "311 Requests" {
		var found bool
		typeIndex, found = columns["SRType"]
		if !found {
			result.Err = fmt.Errorf("missing SRType in %s", filename)
			results <- result
			return
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Err = err
			results <- result
			return
		}

		if neighborhoodIndex >= len(record) {
			continue
		}

		neighborhood := strings.ToUpper(
			strings.TrimSpace(record[neighborhoodIndex]),
		)
		if neighborhood == "" {
			continue
		}

		result.Counts[neighborhood]++
		result.Total++

		if dateIndex >= len(record) {
			result.InvalidDates++
			continue
		}

		month, err := parseMonth(record[dateIndex])
		if err != nil {
			result.InvalidDates++
			continue
		}

		key := neighborhood + "|" + month
		result.MonthlyCounts[key]++

		if typeIndex >= 0 &&
			typeIndex < len(record) &&
			month >= startMonth &&
			month <= endMonth {

			serviceType := strings.ToUpper(
				strings.TrimSpace(record[typeIndex]),
			)
			if serviceType == "" {
				serviceType = "UNSPECIFIED"
			}

			typeKey := neighborhood + "|" + serviceType
			result.Types[typeKey]++
		}
	}

	results <- result
}

func analyzeReportingPeriod(
	requests, crimes DatasetResult,
) {
	requestMonths := make(map[string]bool)
	crimeMonths := make(map[string]bool)

	for key := range requests.MonthlyCounts {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) == 2 {
			requestMonths[parts[1]] = true
		}
	}

	for key := range crimes.MonthlyCounts {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) == 2 {
			crimeMonths[parts[1]] = true
		}
	}

	var shared []string
	for month := range requestMonths {
		if crimeMonths[month] {
			shared = append(shared, month)
		}
	}
	sort.Strings(shared)

	fmt.Println("\nReporting Period Analysis")
	fmt.Println("=========================")

	if len(shared) == 0 {
		fmt.Println("No overlapping months found.")
		return
	}

	fmt.Println("First shared month:", shared[0])
	fmt.Println("Last shared month:", shared[len(shared)-1])
	fmt.Println("Shared months:", len(shared))
}

func main() {
	fmt.Println("Baltimore City Intelligence Engine")
	fmt.Println("==================================")
	fmt.Printf("Available CPUs: %d\n", runtime.NumCPU())

	results := make(chan DatasetResult, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go processDataset(
		"311 Requests",
		"data/service_requests.csv",
		results,
		&wg,
	)

	go processDataset(
		"Crime Incidents",
		"data/crime.csv",
		results,
		&wg,
	)

	go func() {
		wg.Wait()
		close(results)
	}()

	datasets := make(map[string]DatasetResult)
	hasError := false

	for result := range results {
		if result.Err != nil {
			fmt.Println("Error:", result.Err)
			hasError = true
			continue
		}

		datasets[result.Name] = result

		fmt.Printf(
			"\n%s processed: %d records\n",
			result.Name,
			result.Total,
		)
		fmt.Printf(
			"Monthly groups: %d\n",
			len(result.MonthlyCounts),
		)
		fmt.Printf(
			"Invalid or missing dates: %d\n",
			result.InvalidDates,
		)
	}

	if hasError {
		os.Exit(1)
	}

	requests := datasets["311 Requests"]
	crimes := datasets["Crime Incidents"]

	analyzeReportingPeriod(requests, crimes)
	runReports(requests, crimes)

	fmt.Println("\nAnalysis completed successfully.")
}
