package main

import (
	"fmt"
	"sort"
)

type Finding struct {
	Title string
	Body  string
}

func analyzeBurden(buckets []*Bucket) Finding {
	type row struct {
		name     string
		combined int
	}
	var rows []row
	for _, b := range buckets {
		rows = append(rows, row{b.Display, len(b.Requests) + len(b.Crimes)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].combined > rows[j].combined })

	body := "Top neighborhoods by combined 311 + crime volume:\n"
	n := 5
	if n > len(rows) {
		n = len(rows)
	}
	for i := 0; i < n; i++ {
		body += barChart(rows[i].name, rows[i].combined, rows[0].combined) + "\n"
	}
	return Finding{Title: "1. Neighborhood Burden Ranking", Body: body}
}

func analyzeCorrelation(buckets []*Bucket) Finding {
	burglaryByNeighborhood := map[string]int{}
	vacantByNeighborhood := map[string]int{}

	for _, b := range buckets {
		for _, c := range b.Crimes {
			if c.CrimeCode == "BURGLARY" {
				burglaryByNeighborhood[b.Display]++
			}
		}
		for _, r := range b.Requests {
			if r.SRType == "VACANT" {
				vacantByNeighborhood[b.Display]++
			}
		}
	}

	body := "Neighborhoods with both burglaries and vacant property reports:\n"
	for name, count := range burglaryByNeighborhood {
		if vacantByNeighborhood[name] > 0 {
			body += fmt.Sprintf("  %-20s burglaries=%d vacant=%d\n", name, count, vacantByNeighborhood[name])
		}
	}
	return Finding{Title: "2. Crime vs 311 Correlation", Body: body}
}


func analyzeMonthlyTrend(buckets []*Bucket) Finding {
	crimeByMonth := map[string]int{}
	srByMonth := map[string]int{}

	for _, b := range buckets {
		for _, c := range b.Crimes {
			crimeByMonth[c.CrimeDate.Format("2006-01")]++
		}
		for _, r := range b.Requests {
			srByMonth[r.CreatedDate.Format("2006-01")]++
		}
	}

	seen := map[string]bool{}
	var months []string
	for m := range crimeByMonth {
		if !seen[m] {
			months = append(months, m)
			seen[m] = true
		}
	}
	for m := range srByMonth {
		if !seen[m] {
			months = append(months, m)
			seen[m] = true
		}
	}
	sort.Strings(months)

	body := "Monthly volume, crime vs 311:\n"
	for _, m := range months {
		body += fmt.Sprintf("  %s crime=%d 311=%d\n", m, crimeByMonth[m], srByMonth[m])
	}
	return Finding{Title: "3. Monthly Trend", Body: body}
}

func analyzeTypesByDistrict(buckets []*Bucket) Finding {
	typeCounts := map[string]int{}
	for _, b := range buckets {
		for _, r := range b.Requests {
			typeCounts[r.SRType]++
		}
	}

	type row struct {
		name  string
		count int
	}
	var rows []row
	for name, count := range typeCounts {
		rows = append(rows, row{name, count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })

	body := "Most common 311 complaint types:\n"
	for _, r := range rows {
		body += fmt.Sprintf("  %-20s %d\n", r.name, r.count)
	}
	return Finding{Title: "4. 311 Complaint Mix", Body: body}
}

func analyzeOutliers(buckets []*Bucket) Finding {
	type row struct {
		name   string
		ratio  float64
		crimes int
		srs    int
	}
	var rows []row
	for _, b := range buckets {
		if len(b.Crimes) < 3 {
			continue
		}
		ratio := float64(len(b.Requests)) / float64(len(b.Crimes))
		rows = append(rows, row{b.Display, ratio, len(b.Crimes), len(b.Requests)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ratio < rows[j].ratio })

	body := "Neighborhoods with lowest 311-per-crime ratio (possible under-reporting):\n"
	n := 5
	if n > len(rows) {
		n = len(rows)
	}
	for i := 0; i < n; i++ {
		body += fmt.Sprintf("  %-20s crimes=%d 311s=%d ratio=%.2f\n", rows[i].name, rows[i].crimes, rows[i].srs, rows[i].ratio)
	}
	return Finding{Title: "5. Reporting Outliers", Body: body}
}

// barChart renders a simple ASCII horizontal bar chart.
func barChart(label string, value, max int) string {
	width := 30
	filled := 0
	if max > 0 {
		filled = (value * width) / max
	}
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "#"
	}
	for i := filled; i < width; i++ {
		bar += "."
	}
	return fmt.Sprintf("  %-15s [%s] %d", label, bar, value)
}
