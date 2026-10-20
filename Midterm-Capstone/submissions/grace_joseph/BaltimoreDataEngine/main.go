package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

type DatasetSummary struct {
	Name                string
	TotalRecords        int
	MissingNeighborhood int
	NeighborhoodCounts  map[string]int
	RequestTypeCounts   map[string]int
	Error               error
}

func findColumn(headers []string, name string) int {
	for i, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), name) {
			return i
		}
	}
	return -1
}

func getField(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func analyzeCSV(path, datasetName string, is311 bool) DatasetSummary {
	summary := DatasetSummary{
		Name:               datasetName,
		NeighborhoodCounts: make(map[string]int),
		RequestTypeCounts:  make(map[string]int),
	}

	file, err := os.Open(path)
	if err != nil {
		summary.Error = err
		return summary
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		summary.Error = err
		return summary
	}

	neighborhoodIndex := findColumn(headers, "Neighborhood")
	requestTypeIndex := -1

	if is311 {
		requestTypeIndex = findColumn(headers, "SRType")
	}

	for {
		record, err := reader.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			summary.Error = err
			return summary
		}

		summary.TotalRecords++

		neighborhood := getField(record, neighborhoodIndex)
		if neighborhood == "" {
			summary.MissingNeighborhood++
		} else {
			summary.NeighborhoodCounts[neighborhood]++
		}

		if is311 {
			requestType := getField(record, requestTypeIndex)
			if requestType != "" {
				summary.RequestTypeCounts[requestType]++
			}
		}
	}

	return summary
}

func printTopCounts(title string, counts map[string]int, limit int) {
	type item struct {
		name  string
		count int
	}

	items := make([]item, 0, len(counts))

	for name, count := range counts {
		items = append(items, item{name: name, count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].name < items[j].name
		}
		return items[i].count > items[j].count
	})

	fmt.Println("\n" + title)

	if len(items) == 0 {
		fmt.Println("  No records found.")
		return
	}

	if limit > len(items) {
		limit = len(items)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("  %-45s %d\n", items[i].name, items[i].count)
	}
}

func printNeighborhoodComparison(
	requests map[string]int,
	vacant map[string]int,
) {
	type comparison struct {
		name     string
		requests int
		vacant   int
	}

	var results []comparison

	for neighborhood, vacantCount := range vacant {
		requestCount := requests[neighborhood]

		if requestCount > 0 {
			results = append(results, comparison{
				name:     neighborhood,
				requests: requestCount,
				vacant:   vacantCount,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].vacant == results[j].vacant {
			return results[i].name < results[j].name
		}
		return results[i].vacant > results[j].vacant
	})

	fmt.Println("\nNEIGHBORHOOD COMPARISON")
	fmt.Println("Sorted by vacant building notices")
	fmt.Printf("%-35s %15s %15s\n",
		"Neighborhood", "311 Requests", "Vacant Notices")

	limit := 10
	if len(results) < limit {
		limit = len(results)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("%-35s %15d %15d\n",
			results[i].name,
			results[i].requests,
			results[i].vacant,
		)
	}

	fmt.Printf("\nNeighborhoods found in both datasets: %d\n",
		len(results))
}

func printDataAnalysis(requests, vacant DatasetSummary) {
	fmt.Println("\nDATA ANALYSIS AND KEY FINDINGS")

	// Find the neighborhood with the most 311 requests.
	var topRequestNeighborhood string
	maxRequests := 0

	for neighborhood, count := range requests.NeighborhoodCounts {
		if count > maxRequests {
			maxRequests = count
			topRequestNeighborhood = neighborhood
		}
	}

	// Find the neighborhood with the most vacant building notices.
	var topVacantNeighborhood string
	maxVacant := 0

	for neighborhood, count := range vacant.NeighborhoodCounts {
		if count > maxVacant {
			maxVacant = count
			topVacantNeighborhood = neighborhood
		}
	}

	fmt.Printf("Highest 311 request count: %s (%d)\n",
		topRequestNeighborhood, maxRequests)

	fmt.Printf("Highest vacant building notice count: %s (%d)\n",
		topVacantNeighborhood, maxVacant)

	// Calculate the percentage of 311 records missing neighborhood data.
	if requests.TotalRecords > 0 {
		missingPercentage := float64(requests.MissingNeighborhood) /
			float64(requests.TotalRecords) * 100

		fmt.Printf("311 records missing neighborhood information: %.1f%%\n",
			missingPercentage)
	}

	fmt.Println("\nInterpretation:")
	fmt.Println("- The neighborhood with the most 311 requests is not necessarily")
	fmt.Println("  the neighborhood with the most vacant building notices.")
	fmt.Println("- Missing neighborhood information limits how accurately")
	fmt.Println("  the two datasets can be compared.")
	fmt.Println("- These results show patterns in the data, but do not prove")
	fmt.Println("  that vacant buildings cause service requests.")
}

func main() {
	fmt.Println("========================================")
	fmt.Println(" Baltimore City Data Intelligence Engine")
	fmt.Println("========================================")
	fmt.Println("Reading both datasets concurrently...")

	results := make(chan DatasetSummary, 2)
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		results <- analyzeCSV(
			"data/311_Customer_Service_Requests_2026.csv",
			"311 Service Requests",
			true,
		)
	}()

	go func() {
		defer wg.Done()
		results <- analyzeCSV(
			"data/Vacant_Building_Notices.csv",
			"Vacant Building Notices",
			false,
		)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var serviceRequests, vacantBuildings DatasetSummary

	for result := range results {
		if result.Error != nil {
			fmt.Printf("\nError reading %s: %v\n", result.Name, result.Error)
			os.Exit(1)
		}

		fmt.Printf("\nFinished reading: %s\n", result.Name)
		fmt.Printf("Total records: %d\n", result.TotalRecords)
		fmt.Printf("Missing neighborhood: %d\n", result.MissingNeighborhood)

		if result.Name == "311 Service Requests" {
			serviceRequests = result
		} else if result.Name == "Vacant Building Notices" {
			vacantBuildings = result
		}

		printNeighborhoodBarChart(
			"311 REQUESTS BY NEIGHBORHOOD (BAR CHART)",
			serviceRequests.NeighborhoodCounts,
			10,
		)

		printTopCounts(
			"TOP 10 311 REQUEST TYPES",
			serviceRequests.RequestTypeCounts,
			10,
		)

		printTopCounts(
			"TOP 10 311 SERVICE REQUEST NEIGHBORHOODS",
			serviceRequests.NeighborhoodCounts,
			10,
		)

		printTopCounts(
			"TOP 10 VACANT BUILDING NEIGHBORHOODS",
			vacantBuildings.NeighborhoodCounts,
			10,
		)
	}

	printNeighborhoodComparison(
		serviceRequests.NeighborhoodCounts,
		vacantBuildings.NeighborhoodCounts,
	)

	printDataAnalysis(serviceRequests, vacantBuildings)
	fmt.Println("\nInitial data ingestion complete.")
}

func printNeighborhoodBarChart(title string, counts map[string]int, limit int) {
	type item struct {
		name  string
		count int
	}

	items := make([]item, 0, len(counts))

	for name, count := range counts {
		items = append(items, item{name: name, count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].name < items[j].name
		}
		return items[i].count > items[j].count
	})

	if limit > len(items) {
		limit = len(items)
	}

	fmt.Println("\n" + title)

	if limit == 0 {
		fmt.Println("No data available.")
		return
	}

	maxCount := items[0].count
	const maxBarWidth = 40

	for i := 0; i < limit; i++ {
		barLength := 0
		if maxCount > 0 {
			barLength = items[i].count * maxBarWidth / maxCount
		}

		bar := strings.Repeat("#", barLength)

		fmt.Printf("%-32s | %-40s %d\n",
			items[i].name, bar, items[i].count)
	}
}
