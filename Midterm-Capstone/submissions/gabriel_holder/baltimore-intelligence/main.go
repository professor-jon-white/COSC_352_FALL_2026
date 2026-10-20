package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Metric struct {
	Area  string
	Value float64
}
type Result struct {
	Area     string  `json:"area"`
	Crime    float64 `json:"violent_crime_per_1000"`
	Business float64 `json:"businesses_per_1000"`
	Priority float64 `json:"priority_score"`
}
type Job struct{ row []string }
type ParseResult struct {
	values map[string]float64
	errors int
	err    error
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}
func loadCSV(path, areaColumn, valueColumn string, workers int) ParseResult {
	f, err := os.Open(path)
	if err != nil {
		return ParseResult{err: err}
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return ParseResult{err: err}
	}
	ai, vi := -1, -1
	for i, h := range header {
		if normalize(h) == normalize(areaColumn) {
			ai = i
		}
		if normalize(h) == normalize(valueColumn) {
			vi = i
		}
	}
	if ai < 0 || vi < 0 {
		return ParseResult{err: fmt.Errorf("%s: missing columns %q or %q; available: %v", path, areaColumn, valueColumn, header)}
	}
	jobs := make(chan Job, workers*4)
	parsed := make(chan Metric, workers*4)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				r := job.row
				if ai >= len(r) || vi >= len(r) {
					continue
				}
				name := normalize(r[ai])
				v, e := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r[vi]), ",", ""), 64)
				if name != "" && e == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 {
					parsed <- Metric{name, v}
				} else {
					parsed <- Metric{"", -1}
				}
			}
		}()
	}
	readErr := make(chan error, 1)
	go func() {
		defer close(jobs)
		for {
			r, e := reader.Read()
			if e != nil {
				if e != io.EOF {
					readErr <- e
				} else {
					readErr <- nil
				}
				return
			}
			jobs <- Job{r}
		}
	}()
	go func() { wg.Wait(); close(parsed) }()
	result := ParseResult{values: make(map[string]float64)}
	for p := range parsed {
		if p.Area == "" {
			result.errors++
			continue
		}
		result.values[p.Area] = p.Value
	}
	result.err = <-readErr
	return result
}
func join(a, b map[string]float64) []Result {
	out := []Result{}
	for name, crime := range a {
		if business, ok := b[name]; ok {
			out = append(out, Result{Area: name, Crime: crime, Business: business})
		}
	}
	return out
}
func correlation(rows []Result) float64 {
	n := float64(len(rows))
	if n < 2 {
		return math.NaN()
	}
	var sx, sy, sxx, syy, sxy float64
	for _, r := range rows {
		sx += r.Crime
		sy += r.Business
		sxx += r.Crime * r.Crime
		syy += r.Business * r.Business
		sxy += r.Crime * r.Business
	}
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return math.NaN()
	}
	return (n*sxy - sx*sy) / den
}
func score(rows []Result) {
	var mc, mb float64
	for _, r := range rows {
		mc = math.Max(mc, r.Crime)
		mb = math.Max(mb, r.Business)
	}
	for i := range rows {
		c, b := 0.0, 0.0
		if mc > 0 {
			c = rows[i].Crime / mc
		}
		if mb > 0 {
			b = 1 - rows[i].Business/mb
		}
		rows[i].Priority = math.Round((.7*c+.3*b)*1000) / 10
	}
}
func bar(v, max float64) string {
	n := 0
	if max > 0 {
		n = int(math.Round(25 * v / max))
	}
	return strings.Repeat("#", n)
}
func display(rows []Result, crimeN, businessN int) {
	fmt.Println("BALTIMORE NEIGHBORHOOD INTELLIGENCE | rates per 1,000 residents")
	fmt.Printf("Matched areas: %d | Crime-only: %d | Business-only: %d\n\n", len(rows), crimeN-len(rows), businessN-len(rows))
	if len(rows) == 0 {
		return
	}
	ordered := append([]Result(nil), rows...)
	show := func(title string, sortFn func(a, b Result) bool, value func(Result) float64) {
		sort.Slice(ordered, func(i, j int) bool { return sortFn(ordered[i], ordered[j]) })
		fmt.Println(title)
		fmt.Printf("%-39s %10s  %s\n", "COMMUNITY STATISTICAL AREA", "VALUE", "CHART")
		max := 0.0
		for _, r := range ordered {
			max = math.Max(max, value(r))
		}
		for i, r := range ordered {
			if i >= 10 {
				break
			}
			fmt.Printf("%-39.39s %10.2f  %s\n", r.Area, value(r), bar(value(r), max))
		}
		fmt.Println()
	}
	show("Q1. Highest violent crime rate", func(a, b Result) bool { return a.Crime > b.Crime }, func(r Result) float64 { return r.Crime })
	show("Q2. Lowest business density", func(a, b Result) bool { return a.Business < b.Business }, func(r Result) float64 { return r.Business })
	fmt.Println("Q3. Relationship between crime and business density")
	c := correlation(rows)
	if math.IsNaN(c) {
		fmt.Println("Pearson correlation: unavailable (insufficient variation)")
	} else {
		fmt.Printf("Pearson correlation: %.3f (association, not causation)\n", c)
	}
	fmt.Println()
	show("Q4. Highest crime with lower business density (priority score)", func(a, b Result) bool { return a.Priority > b.Priority }, func(r Result) float64 { return r.Priority })
	fmt.Println("Q5. Largest differences between crime and business density")
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Crime-ordered[i].Business > ordered[j].Crime-ordered[j].Business
	})
	fmt.Printf("%-39s %12s\n", "AREA", "RATE GAP")
	for i, r := range ordered {
		if i >= 10 {
			break
		}
		fmt.Printf("%-39.39s %12.2f\n", r.Area, r.Crime-r.Business)
	}
	fmt.Println("\nInterpretation: The two rates describe different activities; subtracting them is an exploratory comparison, not a policy impact estimate.")
}
func main() {
	crime := flag.String("crime", "data/crime.csv", "crime CSV path")
	business := flag.String("business", "data/business.csv", "business CSV path")
	area := flag.String("area", "CSA", "geographic join column")
	crimeCol := flag.String("crime-col", "2023", "crime metric column")
	businessCol := flag.String("business-col", "2023", "business metric column")
	workers := flag.Int("workers", 4, "parser goroutines per dataset")
	out := flag.String("out", "output/report.json", "JSON report path")
	flag.Parse()
	if *workers < 1 || *workers > 128 {
		fmt.Fprintln(os.Stderr, "workers must be 1..128")
		os.Exit(1)
	}
	var a, b ParseResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a = loadCSV(*crime, *area, *crimeCol, *workers) }()
	go func() { defer wg.Done(); b = loadCSV(*business, *area, *businessCol, *workers) }()
	wg.Wait()
	if a.err != nil || b.err != nil {
		fmt.Fprintln(os.Stderr, "ingestion errors:", a.err, b.err)
		os.Exit(1)
	}
	rows := join(a.values, b.values)
	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "no matched areas; inspect geographic names and CSV headers")
		os.Exit(1)
	}
	score(rows)
	display(rows, len(a.values), len(b.values))
	fmt.Printf("\nSkipped invalid records: crime=%d, businesses=%d\n", a.errors, b.errors)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Priority > rows[j].Priority })
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		panic(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		panic(err)
	}
	fmt.Println("JSON report:", *out)
}
