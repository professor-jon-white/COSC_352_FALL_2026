package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Summary struct {
	WellBeingCount int
	GVTCount       int
	CVIMediations  int
	CVIEvents      int
}

func normalizeNeighborhood(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.Join(strings.Fields(name), " ")

	return name
}

func AnalysisOfData(wbRecords []WbRecord, gvtRecords []GVTRecord, cviRecords []CVIRecord) {
	fmt.Println("Combining Data together")

	wbChannel := make(chan map[string]int, 1)
	gvtChannel := make(chan map[string]int, 1)
	cviChannel := make(chan map[string]*Summary, 1)
	neighborhoodStats := make(map[string]*Summary)

	go func() {
		counts := make(map[string]int)
		for _, wb := range wbRecords {
			counts[normalizeNeighborhood(wb.Neighborhood)]++
		}
		wbChannel <- counts
	}()

	go func() {
		counts := make(map[string]int)
		for _, gvt := range gvtRecords {
			counts[normalizeNeighborhood(gvt.Neighborhood)]++
		}
		gvtChannel <- counts
	}()

	go func() {
		counts := make(map[string]*Summary)
		for _, cvi := range cviRecords {
			loc := normalizeNeighborhood(cvi.Neighborhood)
			if _, exists := counts[loc]; !exists {
				counts[loc] = &Summary{}
			}

			value, err := strconv.Atoi(cvi.Value)
			if err != nil {
				fmt.Printf("Error: incorrect CVI value %q for %s\n", cvi.Value, loc)
				continue
			}

			switch cvi.Metric {
			case "Mediations":
				counts[loc].CVIMediations += value
			case "Community Events":
				counts[loc].CVIEvents += value
			}
		}

		cviChannel <- counts
	}()

	wbCounts := <-wbChannel
	gvtCounts := <-gvtChannel
	cviCounts := <-cviChannel

	for loc, count := range wbCounts {
		if neighborhoodStats[loc] == nil {
			neighborhoodStats[loc] = &Summary{}
		}
		neighborhoodStats[loc].WellBeingCount = count
	}

	for loc, count := range gvtCounts {
		if neighborhoodStats[loc] == nil {
			neighborhoodStats[loc] = &Summary{}
		}
		neighborhoodStats[loc].GVTCount = count
	}

	for loc, stats := range cviCounts {
		if neighborhoodStats[loc] == nil {
			neighborhoodStats[loc] = &Summary{}
		}
		neighborhoodStats[loc].CVIMediations = stats.CVIMediations
		neighborhoodStats[loc].CVIEvents = stats.CVIEvents
	}

	fmt.Println("These are the neighborhoods and locations obtained from the datasets:", len(neighborhoodStats))
	fmt.Println("Here are some of the neighborhoods that were cross referenced:")

	fmt.Println("Here are the reports for the Community Violence InterVention:")

	neighborhoodNames := make([]string, 0, len(neighborhoodStats))

	for loc := range neighborhoodStats {
		neighborhoodNames = append(neighborhoodNames, loc)
	}
	sort.Strings(neighborhoodNames)

	count := 0
	for _, loc := range neighborhoodNames {
		stats := neighborhoodStats[loc]
		if loc != "" && count < 5 {
			fmt.Printf("Neighborhood: %s | Arrest Records: %d | Gun Offender Records: %d | CVI Mediations: %d | CVI Events: %d\n", loc,
				stats.WellBeingCount,
				stats.GVTCount,
				stats.CVIMediations,
				stats.CVIEvents,
			)
			count++
		}
	}
	fmt.Println("\nHere are the research Questions:")
	fmt.Println("\n1. How many unique neighborhoods appear across the datasets?")
	fmt.Println("Answer:", len(neighborhoodStats))

	fmt.Println("\n2. Which neighborhoods have both Arrest records and Gun Offender records?")
	found := false

	for _, loc := range neighborhoodNames {
		stats := neighborhoodStats[loc]
		if stats.WellBeingCount > 0 && stats.GVTCount > 0 {
			fmt.Println(loc)
			found = true
		}
	}
	if !found {
		fmt.Println("None")
	}

	fmt.Println("\n3. Which neighborhood has the highest CVI mediations?")

	highestMediations := -1
	highestNeighborhood := ""

	for _, loc := range neighborhoodNames {
		stats := neighborhoodStats[loc]
		if stats.CVIMediations > highestMediations {
			highestMediations = stats.CVIMediations
			highestNeighborhood = loc
		}
	}
	fmt.Println(highestNeighborhood, " ", highestMediations, "mediations")

	fmt.Println("\n4. Which neighborhood has the fewest CVI community events?")
	lowestEvents := -1
	lowestNeighborhood := ""

	for _, loc := range neighborhoodNames {
		stats := neighborhoodStats[loc]
		if stats.CVIEvents > 0 && (lowestEvents == -1 || stats.CVIEvents < lowestEvents) {
			lowestEvents = stats.CVIEvents
			lowestNeighborhood = loc
		}
	}

	if lowestNeighborhood != "" {
		fmt.Println(lowestNeighborhood, " ", lowestEvents, "events")
	} else {
		fmt.Println("No CVI community events found.")
	}

	fmt.Println("\n5. Which neighborhoods have CVI activity but no Gun Offender records?")
	found = false

	for _, loc := range neighborhoodNames {
		stats := neighborhoodStats[loc]
		if (stats.CVIMediations > 0 || stats.CVIEvents > 0) && stats.GVTCount == 0 {
			fmt.Println(loc)
			found = true
		}
	}

	if !found {
		fmt.Println("None")
	}
	fmt.Println("Done")
}
