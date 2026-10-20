package main

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type AreaStats struct {
	Name     string
	Requests int
	Crimes   int
}

type MonthlyStats struct {
	Month    string
	Requests int
	Crimes   int
}

type CategoryCount struct {
	Name  string
	Count int
}

func aggregateConcurrently(
	requests, crimes DatasetResult,
) ([]AreaStats, []MonthlyStats) {
	jobs := make(chan MonthRecord, 256)

	workerCount := runtime.NumCPU()
	if workerCount > 8 {
		workerCount = 8
	}
	if workerCount < 1 {
		workerCount = 1
	}

	type Partial struct {
		Areas  map[string]AreaStats
		Months map[string]MonthlyStats
	}

	partials := make(chan Partial, workerCount)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			local := Partial{
				Areas:  make(map[string]AreaStats),
				Months: make(map[string]MonthlyStats),
			}

			for job := range jobs {
				area := local.Areas[job.Neighborhood]
				area.Name = job.Neighborhood

				month := local.Months[job.Month]
				month.Month = job.Month

				if job.IsCrime {
					area.Crimes += job.Count
					month.Crimes += job.Count
				} else {
					area.Requests += job.Count
					month.Requests += job.Count
				}

				local.Areas[job.Neighborhood] = area
				local.Months[job.Month] = month
			}

			partials <- local
		}()
	}

	sendJobs := func(counts map[string]int, crime bool) {
		for key, count := range counts {
			parts := strings.SplitN(key, "|", 2)
			if len(parts) != 2 {
				continue
			}

			neighborhood, month := parts[0], parts[1]
			if month < startMonth || month > endMonth {
				continue
			}

			jobs <- MonthRecord{
				Neighborhood: neighborhood,
				Month:        month,
				Count:        count,
				IsCrime:      crime,
			}
		}
	}

	sendJobs(requests.MonthlyCounts, false)
	sendJobs(crimes.MonthlyCounts, true)
	close(jobs)

	wg.Wait()
	close(partials)

	areaTotals := make(map[string]AreaStats)
	monthTotals := make(map[string]MonthlyStats)

	for partial := range partials {
		for name, area := range partial.Areas {
			total := areaTotals[name]
			total.Name = name
			total.Requests += area.Requests
			total.Crimes += area.Crimes
			areaTotals[name] = total
		}

		for month, item := range partial.Months {
			total := monthTotals[month]
			total.Month = month
			total.Requests += item.Requests
			total.Crimes += item.Crimes
			monthTotals[month] = total
		}
	}

	areas := make([]AreaStats, 0, len(areaTotals))
	for _, area := range areaTotals {
		areas = append(areas, area)
	}

	months := make([]MonthlyStats, 0, len(monthTotals))
	for _, month := range monthTotals {
		months = append(months, month)
	}

	sort.Slice(months, func(i, j int) bool {
		return months[i].Month < months[j].Month
	})

	return areas, months
}

func sortAreas(
	areas []AreaStats,
	score func(AreaStats) int,
) []AreaStats {
	result := append([]AreaStats(nil), areas...)

	sort.Slice(result, func(i, j int) bool {
		a := score(result[i])
		b := score(result[j])

		if a == b {
			return result[i].Name < result[j].Name
		}
		return a > b
	})

	return result
}

func drawBar(label string, value, maximum int) {
	width := 35
	length := 0

	if maximum > 0 {
		length = int(math.Round(
			float64(value) / float64(maximum) * float64(width),
		))
	}

	if length > width {
		length = width
	}

	fmt.Printf(
		"%-12s | %-35s %d\n",
		label,
		strings.Repeat("#", length),
		value,
	)
}

func reportOne(areas []AreaStats) {
	fmt.Println("\nREPORT 1: Neighborhood Activity")
	fmt.Println("================================")

	sorted := sortAreas(areas, func(a AreaStats) int {
		return a.Requests + a.Crimes
	})

	fmt.Printf(
		"%-32s %12s %12s\n",
		"Neighborhood",
		"311 Requests",
		"Crime Records",
	)

	for i, a := range sorted {
		if i >= 10 {
			break
		}
		fmt.Printf(
			"%-32s %12d %12d\n",
			a.Name, a.Requests, a.Crimes,
		)
	}

	fmt.Println(
		"\nRanking uses combined event counts, not population-adjusted rates.",
	)
}

func reportTwo(areas []AreaStats) {
	fmt.Println("\nREPORT 2: Crime and 311 Association")
	fmt.Println("===================================")

	n := float64(len(areas))
	if n < 2 {
		fmt.Println("Insufficient neighborhoods.")
		return
	}

	var sx, sy, sxx, syy, sxy float64

	for _, area := range areas {
		x := float64(area.Requests)
		y := float64(area.Crimes)

		sx += x
		sy += y
		sxx += x * x
		syy += y * y
		sxy += x * y
	}

	denominator := math.Sqrt(
		(n*sxx - sx*sx) * (n*syy - sy*sy),
	)

	if denominator <= 0 {
		fmt.Println("Correlation cannot be calculated.")
		return
	}

	r := (n*sxy - sx*sy) / denominator

	fmt.Printf("Pearson correlation: %.3f\n", r)
	fmt.Println("Positive values indicate a positive association.")
	fmt.Println("Correlation does not establish causation.")
	fmt.Println("Counts are not adjusted for neighborhood population.")
}

func reportThree(areas []AreaStats) []AreaStats {
	fmt.Println("\nREPORT 3: High Activity in Both Datasets")
	fmt.Println("========================================")

	if len(areas) == 0 {
		return nil
	}

	var sumRequests, sumCrimes float64

	for _, a := range areas {
		sumRequests += float64(a.Requests)
		sumCrimes += float64(a.Crimes)
	}

	averageRequests := sumRequests / float64(len(areas))
	averageCrimes := sumCrimes / float64(len(areas))

	var selected []AreaStats

	for _, area := range areas {
		if float64(area.Requests) > averageRequests &&
			float64(area.Crimes) > averageCrimes {
			selected = append(selected, area)
		}
	}

	selected = sortAreas(selected, func(a AreaStats) int {
		return a.Requests + a.Crimes
	})

	fmt.Printf("Mean 311 requests: %.1f\n", averageRequests)
	fmt.Printf("Mean crime records: %.1f\n", averageCrimes)
	fmt.Printf("Neighborhoods above both means: %d\n", len(selected))

	for i, a := range selected {
		if i >= 10 {
			break
		}

		fmt.Printf(
			"%-32s Requests: %-7d Crimes: %d\n",
			a.Name, a.Requests, a.Crimes,
		)
	}

	return selected
}

func reportFour(
	requests DatasetResult,
	highActivity []AreaStats,
) {
	fmt.Println("\nREPORT 4: 311 Categories in High-Activity Neighborhoods")
	fmt.Println("=======================================================")

	selected := make(map[string]bool)

	for _, area := range highActivity {
		selected[area.Name] = true
	}

	categoryTotals := make(map[string]int)

	for key, count := range requests.Types {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}

		if selected[parts[0]] {
			categoryTotals[parts[1]] += count
		}
	}

	var categories []CategoryCount

	for name, count := range categoryTotals {
		categories = append(categories, CategoryCount{
			Name:  name,
			Count: count,
		})
	}

	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Count == categories[j].Count {
			return categories[i].Name < categories[j].Name
		}
		return categories[i].Count > categories[j].Count
	})

	if len(categories) == 0 {
		fmt.Println("No matching categories found.")
		return
	}

	maximum := categories[0].Count

	for i, category := range categories {
		if i >= 10 {
			break
		}
		drawBar(category.Name, category.Count, maximum)
	}

	fmt.Println(
		"\nCategories are aggregated across neighborhoods above both dataset means.",
	)
}

func reportFive(months []MonthlyStats) {
	fmt.Println("\nREPORT 5: Monthly Activity Trends")
	fmt.Println("=================================")

	maxRequests := 0
	maxCrimes := 0

	for _, month := range months {
		if month.Requests > maxRequests {
			maxRequests = month.Requests
		}
		if month.Crimes > maxCrimes {
			maxCrimes = month.Crimes
		}
	}

	fmt.Println("\n311 SERVICE REQUESTS")
	for _, month := range months {
		drawBar(month.Month, month.Requests, maxRequests)
	}

	fmt.Println("\nCRIME RECORDS")
	for _, month := range months {
		drawBar(month.Month, month.Crimes, maxCrimes)
	}

	fmt.Println(
		"\nEach chart uses its own scale; compare trends, not bar lengths across charts.",
	)
}

func runReports(
	requests, crimes DatasetResult,
) {
	fmt.Printf(
		"\nANALYSIS PERIOD: %s through %s\n",
		startMonth, endMonth,
	)

	areas, months := aggregateConcurrently(requests, crimes)

	fmt.Printf("Neighborhoods analyzed: %d\n", len(areas))
	fmt.Printf("Months analyzed: %d\n", len(months))

	reportOne(areas)
	reportTwo(areas)

	highActivity := reportThree(areas)
	reportFour(requests, highActivity)
	reportFive(months)
}
