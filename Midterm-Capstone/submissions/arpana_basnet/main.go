package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type DatasetResult struct {
	Counts  map[string]int
	Total   int
	Missing int
	Err     error
}

// Each worker owns its neighborhood-count map.
func countWorker(
	jobs <-chan string,
	results chan<- DatasetResult,
) {
	result := DatasetResult{
		Counts: make(map[string]int),
	}

	for rawNeighborhood := range jobs {
		result.Total++

		name := strings.ToUpper(strings.TrimSpace(rawNeighborhood))
		if name == "" {
			result.Missing++
			continue
		}

		if result.Counts[name] == 0 {
			name = strings.Clone(name)
		}

		result.Counts[name]++
	}

	results <- result
}

// Used to ingest the vacant-building notice dataset.
func countNeighborhoods(path string) (map[string]int, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, 0, 0, err
	}

	neighborhoodIndex := -1
	for index, name := range headers {
		if strings.TrimSpace(name) == "Neighborhood" {
			neighborhoodIndex = index
			break
		}
	}

	if neighborhoodIndex == -1 {
		return nil, 0, 0, fmt.Errorf(
			"Neighborhood column missing in %s", path,
		)
	}

	const workerCount = 4

	jobs := make(chan string, 256)
	results := make(chan DatasetResult, workerCount)

	for i := 0; i < workerCount; i++ {
		go countWorker(jobs, results)
	}

	var readErr error
	recordNumber := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			readErr = fmt.Errorf("%s: %w", path, err)
			break
		}

		recordNumber++

		if len(record) != len(headers) {
			readErr = fmt.Errorf(
				"%s: unexpected field count in record %d",
				path, recordNumber,
			)
			break
		}

		jobs <- record[neighborhoodIndex]
	}

	close(jobs)

	counts := make(map[string]int)
	total, missing := 0, 0

	for i := 0; i < workerCount; i++ {
		result := <-results
		total += result.Total
		missing += result.Missing

		for name, count := range result.Counts {
			counts[name] += count
		}
	}

	if readErr != nil {
		return nil, 0, 0, readErr
	}

	return counts, total, missing, nil
}

func loadDataset(path string, results chan<- DatasetResult) {
	counts, total, missing, err := countNeighborhoods(path)

	results <- DatasetResult{
		Counts:  counts,
		Total:   total,
		Missing: missing,
		Err:     err,
	}
}

// Reuse the same 311 summary that supplies Questions 3–5.
func loadRequestCounts(results chan<- DatasetResult) {
	data, err := getServiceData()
	if err != nil {
		results <- DatasetResult{Err: err}
		return
	}

	counts := make(map[string]int)

	for name, metrics := range data.Neighborhoods {
		if name != "" {
			counts[name] = metrics.Records
		}
	}

	results <- DatasetResult{
		Counts:  counts,
		Total:   data.Totals.Records,
		Missing: data.Neighborhoods[""].Records,
	}
}

func main() {
	requestChannel := make(chan DatasetResult, 1)
	noticeChannel := make(chan DatasetResult, 1)

	fmt.Println("Processing both datasets with four workers each...")

	// Read the two datasets concurrently.
	go loadRequestCounts(requestChannel)
	go loadDataset("data/Vacant_Building.csv", noticeChannel)

	requestResult := <-requestChannel
	noticeResult := <-noticeChannel

	if requestResult.Err != nil {
		fmt.Fprintln(os.Stderr, "311 dataset error:", requestResult.Err)
		os.Exit(1)
	}

	if noticeResult.Err != nil {
		fmt.Fprintln(os.Stderr, "Vacant-building dataset error:", noticeResult.Err)
		os.Exit(1)
	}

	requests := requestResult.Counts
	notices := noticeResult.Counts

	fmt.Println("311 records:", requestResult.Total)
	fmt.Println("311 records missing neighborhood:", requestResult.Missing)
	fmt.Println("Vacant-building notice records:", noticeResult.Total)
	fmt.Println("Notice records missing neighborhood:", noticeResult.Missing)

	matched := matchNeighborhoods(requests, notices)
	sort.Strings(matched)

	reportTopNotices(matched, requests, notices)

	fmt.Println("\nNeighborhoods found in both datasets:", len(matched))
	fmt.Printf(
		"%-45s %12s %12s\n",
		"Neighborhood", "311 Records", "Notices",
	)

	for _, neighborhood := range matched {
		fmt.Printf(
			"%-45s %12d %12d\n",
			neighborhood,
			requests[neighborhood],
			notices[neighborhood],
		)
	}
}
