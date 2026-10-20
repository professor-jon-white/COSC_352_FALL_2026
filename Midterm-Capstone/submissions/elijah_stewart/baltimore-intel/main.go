package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Summary struct {
	Name   string
	Count  int64
	ByType map[string]int64
	ByArea map[string]int64
	Err    error
}

func clean(s string) string { return strings.TrimSpace(s) }
func key(s string) string   { return strings.ToLower(strings.TrimSpace(s)) }
func choose(rec map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := clean(rec[key(k)]); v != "" {
			return v
		}
	}
	return "Unknown"
}
func openSource(pathOrURL string) (io.ReadCloser, error) {
	if strings.HasPrefix(strings.ToLower(pathOrURL), "https://") || strings.HasPrefix(strings.ToLower(pathOrURL), "http://") {
		client := &http.Client{Timeout: 2 * time.Minute}
		resp, err := client.Get(pathOrURL)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %s", resp.Status)
		}
		return resp.Body, nil
	}
	return os.Open(pathOrURL)
}
func summarize(name, path string, typeKeys, areaKeys []string) Summary {
	s := Summary{Name: name, ByType: map[string]int64{}, ByArea: map[string]int64{}}
	r, err := openSource(path)
	if err != nil {
		s.Err = err
		return s
	}
	defer r.Close()
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	headers, err := cr.Read()
	if err != nil {
		s.Err = err
		return s
	}
	for i, h := range headers {
		headers[i] = key(h)
	}
	for {
		row, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			continue
		}
		rec := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				rec[h] = row[i]
			}
		}
		s.Count++
		s.ByType[choose(rec, typeKeys...)]++
		s.ByArea[choose(rec, areaKeys...)]++
	}
	return s
}

type pair struct {
	Name  string
	Count int64
}

func top(m map[string]int64, n int) []pair {
	a := make([]pair, 0, len(m))
	for k, v := range m {
		a = append(a, pair{k, v})
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].Count == a[j].Count {
			return a[i].Name < a[j].Name
		}
		return a[i].Count > a[j].Count
	})
	if len(a) > n {
		a = a[:n]
	}
	return a
}
func chart(title string, m map[string]int64) {
	fmt.Println("\n" + title)
	ps := top(m, 8)
	var max int64 = 1
	for _, p := range ps {
		if p.Count > max {
			max = p.Count
		}
	}
	if len(ps) == 0 {
		fmt.Println("  No matching records.")
		return
	}
	for _, p := range ps {
		bars := int(p.Count * 30 / max)
		if bars < 1 {
			bars = 1
		}
		fmt.Printf("%-28.28s %8d %s\n", p.Name, p.Count, strings.Repeat("#", bars))
	}
}
func main() {
	offline := flag.Bool("offline", false, "use the small included sample datasets")
	requests := flag.String("311", "311_Customer_Service_Requests_Yearly_7199422805573403920.csv", "path or direct URL to the Baltimore 311 CSV")
	arrests := flag.String("arrests", "Arrest.csv", "path or direct URL to the Baltimore arrest CSV")
	flag.Parse()
	p1, p2 := *requests, *arrests
	if *offline {
		p1 = "data/311_sample.csv"
		p2 = "data/crime_sample.csv"
	}
	var a, b Summary
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a = summarize("311", p1, []string{"srtype", "service_type", "type", "description"}, []string{"neighborhood", "council_district", "zipcode", "community", "police_district"})
	}()
	go func() {
		defer wg.Done()
		b = summarize("arrests", p2, []string{"incidentoffence", "chargedescription", "charge", "description", "crime_type", "offense"}, []string{"neighborhood", "district", "police_district", "arrestlocation"})
	}()
	wg.Wait()
	if a.Err != nil || b.Err != nil {
		fmt.Fprintln(os.Stderr, "Could not load both datasets.")
		if a.Err != nil {
			fmt.Fprintln(os.Stderr, "311:", a.Err)
		}
		if b.Err != nil {
			fmt.Fprintln(os.Stderr, "arrests:", b.Err)
		}
		fmt.Fprintln(os.Stderr, "Use --offline for samples, or pass --311 and --arrests with valid CSV file paths.")
		os.Exit(1)
	}
	fmt.Println("BALTIMORE MUNICIPAL INTELLIGENCE\n================================")
	fmt.Printf("311 service requests: %d | Arrest records: %d\n", a.Count, b.Count)
	chart("1. Which 311 service-request types appear most often?", a.ByType)
	chart("2. Which neighborhoods have the most 311 requests?", a.ByArea)
	chart("3. Which offenses or charge descriptions appear most often in arrest records?", b.ByType)
	chart("4. Which neighborhoods or districts appear most often in arrest records?", b.ByArea)
	joined := map[string]int64{}
	for area, n := range a.ByArea {
		if n2, ok := b.ByArea[area]; ok && key(area) != "unknown" {
			joined[area] = n + n2
		}
	}
	chart("5. Which matching area labels have records in both datasets? (combined counts; exploratory)", joined)
	fmt.Println("\nInterpretation: These are descriptive counts, not proof that one dataset causes the other. Arrest records are not a complete count of all crimes. Dataset date ranges, neighborhood labels, and coverage differ; combined counts should not be interpreted as risk or crime rates.")
}
