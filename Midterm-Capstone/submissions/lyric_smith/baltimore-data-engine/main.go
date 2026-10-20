package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Loading datasets concurrently...")

	results := loadAll(map[string]Job{
		"311":   {Path: "data/311_sample.csv", Parser: NewServiceRequestParser()},
		"crime": {Path: "data/crime_sample.csv", Parser: NewCrimeParser()},
	})

	srResult := results["311"]
	crimeResult := results["crime"]

	if srResult.Err != nil {
		fmt.Println("error loading 311 data:", srResult.Err)
		os.Exit(1)
	}
	if crimeResult.Err != nil {
		fmt.Println("error loading crime data:", crimeResult.Err)
		os.Exit(1)
	}

	var requests []ServiceRequest
	for _, r := range srResult.Records {
		requests = append(requests, r.(ServiceRequest))
	}

	var crimes []CrimeIncident
	for _, c := range crimeResult.Records {
		crimes = append(crimes, c.(CrimeIncident))
	}

	fmt.Printf("Loaded %d 311 requests (%d skipped)\n", len(requests), srResult.Skipped)
	fmt.Printf("Loaded %d crime records (%d skipped)\n", len(crimes), crimeResult.Skipped)

	fmt.Println("Joining datasets...")
	idx := buildIndex(requests, crimes)
	buckets := idx.all()
	fmt.Printf("Joined into %d neighborhoods\n", len(buckets))

	fmt.Println("Running analyses...")
	findings := []Finding{
		analyzeBurden(buckets),
		analyzeCorrelation(buckets),
		analyzeMonthlyTrend(buckets),
		analyzeTypesByDistrict(buckets),
		analyzeOutliers(buckets),
	}

	fmt.Println()
	fmt.Println("=== BALTIMORE DATA ENGINE REPORT ===")
	for _, f := range findings {
		fmt.Println()
		fmt.Println(f.Title)
		fmt.Println(f.Body)
	}
}
