package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func findColumn(headers []string, name string) int {
	for i, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), name) {
			return i
		}
	}
	return -1
}

type NeighborhoodData struct {
	Name     string
	Requests int
	Vacant   int
}

func showTopFive(data []NeighborhoodData, byVacant bool) {
	sorted := append([]NeighborhoodData(nil), data...)

	sort.Slice(sorted, func(i, j int) bool {
		if byVacant {
			return sorted[i].Vacant > sorted[j].Vacant
		}
		return sorted[i].Requests > sorted[j].Requests
	})

	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}

	for i := 0; i < limit; i++ {
		count := sorted[i].Requests
		if byVacant {
			count = sorted[i].Vacant
		}

		fmt.Printf("%-25s %8d\n", sorted[i].Name, count)
	}
}

func showChart(data []NeighborhoodData) {
	sorted := append([]NeighborhoodData(nil), data...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Vacant > sorted[j].Vacant
	})

	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}

	if limit == 0 {
		return
	}

	max := sorted[0].Vacant

	for i := 0; i < limit; i++ {
		bars := 0
		if max > 0 {
			bars = sorted[i].Vacant * 30 / max
		}

		fmt.Printf("%-25s | %-30s %d\n",
			sorted[i].Name,
			strings.Repeat("#", bars),
			sorted[i].Vacant)
	}
}

func calculateCorrelation(data []NeighborhoodData) float64 {
	n := float64(len(data))
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for _, item := range data {
		x := float64(item.Requests)
		y := float64(item.Vacant)

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := n*sumXY - sumX*sumY

	denominator := math.Sqrt(
		(n*sumX2 - sumX*sumX) *
			(n*sumY2 - sumY*sumY),
	)

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func main() {
	fmt.Println("===================================")
	fmt.Println("Baltimore City Data Analysis Engine")
	fmt.Println("===================================")

	type datasetResult struct {
		counts map[string]int
		total  int
		err    error
	}

	requestChannel := make(chan datasetResult, 1)
	propertyChannel := make(chan datasetResult, 1)

	// Process both datasets concurrently.
	go func() {
		counts, total, err := processDataset("data/service_requests.csv")
		requestChannel <- datasetResult{counts, total, err}
	}()

	go func() {
		counts, total, err := processDataset("data/vacant_properties.csv")
		propertyChannel <- datasetResult{counts, total, err}
	}()

	requestResult := <-requestChannel
	propertyResult := <-propertyChannel

	if requestResult.err != nil {
		fmt.Println("311 processing error:", requestResult.err)
		return
	}

	if propertyResult.err != nil {
		fmt.Println("Property processing error:", propertyResult.err)
		return
	}

	combined := combineDataParallel(
		requestResult.counts,
		propertyResult.counts,
	)

	fmt.Println("\nDatasets processed successfully!")
	fmt.Println("311 Records:", requestResult.total)
	fmt.Println("Vacant Property Records:", propertyResult.total)

	fmt.Println("\nQUESTION 1: Which neighborhoods have the most 311 requests?")
	showTopFive(combined, false)

	fmt.Println("\nQUESTION 2: Which neighborhoods have the most vacant properties?")
	showTopFive(combined, true)

	fmt.Println("\nQUESTION 3: What is the relationship between 311 requests and vacant properties?")

	correlation := calculateCorrelation(combined)

	fmt.Printf("Correlation coefficient: %.3f\n", correlation)

	if correlation > 0.5 {
		fmt.Println("Strong positive relationship")
	} else if correlation > 0.2 {
		fmt.Println("Weak to moderate positive relationship")
	} else if correlation < -0.5 {
		fmt.Println("Strong negative relationship")
	} else if correlation < -0.2 {
		fmt.Println("Weak to moderate negative relationship")
	} else {
		fmt.Println("Little or no linear relationship")
	}

	fmt.Println("\nQUESTION 4: Which neighborhoods have the highest 311 requests per vacant property?")

	ratioData := []NeighborhoodData{}

	for _, item := range combined {
		if item.Vacant > 0 {
			ratioData = append(ratioData, item)
		}
	}

	sort.Slice(ratioData, func(i, j int) bool {
		ratioI := float64(ratioData[i].Requests) / float64(ratioData[i].Vacant)
		ratioJ := float64(ratioData[j].Requests) / float64(ratioData[j].Vacant)
		return ratioI > ratioJ
	})

	for i := 0; i < 5 && i < len(ratioData); i++ {
		item := ratioData[i]
		ratio := float64(item.Requests) / float64(item.Vacant)

		fmt.Printf("%-25s Ratio: %.2f\n", item.Name, ratio)
	}

	fmt.Println("\nQUESTION 5: Which neighborhoods should city officials prioritize?")

	priorityData := append([]NeighborhoodData(nil), combined...)

	maxRequests := 0
	maxVacant := 0

	for _, item := range priorityData {
		if item.Requests > maxRequests {
			maxRequests = item.Requests
		}
		if item.Vacant > maxVacant {
			maxVacant = item.Vacant
		}
	}

	priorityScore := func(item NeighborhoodData) float64 {
		requestScore := 0.0
		vacantScore := 0.0

		if maxRequests > 0 {
			requestScore = float64(item.Requests) / float64(maxRequests)
		}

		if maxVacant > 0 {
			vacantScore = float64(item.Vacant) / float64(maxVacant)
		}

		return (requestScore*0.5 + vacantScore*0.5) * 100
	}

	sort.Slice(priorityData, func(i, j int) bool {
		return priorityScore(priorityData[i]) > priorityScore(priorityData[j])
	})

	for i := 0; i < 5 && i < len(priorityData); i++ {
		item := priorityData[i]

		fmt.Printf("%-30s Priority Score: %.2f\n",
			item.Name, priorityScore(item))
	}
	fmt.Println("\nASCII CHART: Top 5 Neighborhoods by Vacant Properties")
	showChart(combined)
}
