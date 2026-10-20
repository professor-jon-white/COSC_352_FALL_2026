// Package fetch downloads Open Baltimore datasets from their ArcGIS REST
// endpoints and writes them as raw CSV files (original field names, dates as
// epoch milliseconds exactly as the API returns them, plus X/Y coordinates).
//
// Download strategy, which works on both ArcGIS Online and on-premise
// ArcGIS Server layers:
//  1. query?returnIdsOnly=true            -> every matching object ID
//  2. split the IDs into chunks of 500    -> POST query?objectIds=...
//  3. a pool of goroutines fetches chunks concurrently, a single writer
//     goroutine owns the CSV file (no locking around the writer)
package fetch

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Dataset describes one Open Baltimore layer.
type Dataset struct {
	Kind     string // crime | 311 | liquor | vacants
	Title    string
	LayerURL string // .../FeatureServer/N
	Where    string
	File     string
	MaxRows  int // 0 = all; otherwise an evenly spaced sample of IDs
}

// Catalog returns the layers used for an analysis year. URLs were taken from
// the Open Baltimore portal (data.baltimorecity.gov) dataset pages.
func Catalog(year, max311 int) []Dataset {
	const agol = "https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services"
	const egis = "https://baltegis.baltimorecity.gov/mapping/rest/services"
	start, end := fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-01-01", year+1)
	return []Dataset{
		{Kind: "crime", Title: "Part 1 Crime Data",
			LayerURL: agol + "/Part1_Crime_Beta/FeatureServer/0",
			Where:    fmt.Sprintf("CrimeDateTime >= DATE '%s' AND CrimeDateTime < DATE '%s'", start, end),
			File:     "part1_crime.csv"},
		{Kind: "311", Title: fmt.Sprintf("311 Customer Service Requests %d", year),
			LayerURL: service311Layer(agol, year), Where: "1=1",
			File: "service_requests_311.csv", MaxRows: max311},
		{Kind: "liquor", Title: "Liquor Licenses",
			LayerURL: egis + "/CityView/Licenses/FeatureServer/0", Where: "1=1",
			File: "liquor_licenses.csv"},
		{Kind: "vacants", Title: "Vacant Building Notices",
			LayerURL: egis + "/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1", Where: "1=1",
			File: "vacant_building_notices.csv"},
	}
}

// 311 data is published as one layer per year (2004-2022) in a yearly
// service, with newer years in their own services.
func service311Layer(base string, year int) string {
	if year >= 2004 && year <= 2022 {
		return fmt.Sprintf("%s/311_Customer_Service_Requests_Yearly/FeatureServer/%d", base, 2022-year)
	}
	return fmt.Sprintf("%s/311_Customer_Service_Requests_%d/FeatureServer/0", base, year)
}

// Client fetches layers.
type Client struct {
	HTTP    *http.Client
	Workers int
	Chunk   int
	Log     func(format string, args ...any)
}

func NewClient(workers int) *Client {
	return &Client{HTTP: &http.Client{Timeout: 90 * time.Second}, Workers: workers, Chunk: 500,
		Log: func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) }}
}

type apiError struct {
	Error *struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Details []string `json:"details"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, layer string, form url.Values, out any) error {
	form.Set("f", "json")
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt*attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, layer+"/query", strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "baltimore-intel/1.0 (course project)")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("HTTP %d: %v", resp.StatusCode, err)
			continue
		}
		var ae apiError // ArcGIS reports errors as HTTP 200 + {"error":{...}}
		if json.Unmarshal(body, &ae) == nil && ae.Error != nil {
			lastErr = fmt.Errorf("ArcGIS error %d: %s %v", ae.Error.Code, ae.Error.Message, ae.Error.Details)
			if ae.Error.Code == 400 {
				return lastErr // bad query: retrying won't help
			}
			continue
		}
		d := json.NewDecoder(strings.NewReader(string(body)))
		d.UseNumber() // keep epoch-ms dates and IDs exact
		return d.Decode(out)
	}
	return lastErr
}

type idResp struct {
	ObjectIDs []int64 `json:"objectIds"`
}

type feature struct {
	Attributes map[string]any `json:"attributes"`
	Geometry   *struct {
		X     *float64       `json:"x"`
		Y     *float64       `json:"y"`
		Rings [][][2]float64 `json:"rings"`
	} `json:"geometry"`
}

type featResp struct {
	Fields []struct {
		Name string `json:"name"`
	} `json:"fields"`
	Features []feature `json:"features"`
}

// Fetch downloads one dataset into dir/ds.File. It returns rows written.
func (c *Client) Fetch(ctx context.Context, dir string, ds Dataset) (int, error) {
	start := time.Now()
	var ids idResp
	if err := c.call(ctx, ds.LayerURL, url.Values{"where": {ds.Where}, "returnIdsOnly": {"true"}}, &ids); err != nil {
		return 0, fmt.Errorf("%s: listing IDs: %w", ds.Title, err)
	}
	all := ids.ObjectIDs
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	if ds.MaxRows > 0 && len(all) > ds.MaxRows { // even stride keeps the sample spread across the year
		step := float64(len(all)) / float64(ds.MaxRows)
		sample := make([]int64, 0, ds.MaxRows)
		for i := 0; i < ds.MaxRows; i++ {
			sample = append(sample, all[int(math.Floor(float64(i)*step))])
		}
		c.Log("  %s: sampling %d of %d records", ds.Title, ds.MaxRows, len(all))
		all = sample
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("%s: query returned no records (where=%q)", ds.Title, ds.Where)
	}
	c.Log("  %s: %d records in %d chunks", ds.Title, len(all), (len(all)+c.Chunk-1)/c.Chunk)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	tmp := filepath.Join(dir, ds.File+".part")
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp) // no-op after the successful rename

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	chunks := make(chan []int64)
	pages := make(chan []feature, c.Workers)
	errc := make(chan error, c.Workers)
	var wg sync.WaitGroup
	var fieldsOnce sync.Once
	var fields []string

	for w := 0; w < c.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ch := range chunks {
				strs := make([]string, len(ch))
				for i, id := range ch {
					strs[i] = strconv.FormatInt(id, 10)
				}
				var fr featResp
				err := c.call(ctx, ds.LayerURL, url.Values{
					"objectIds": {strings.Join(strs, ",")}, "outFields": {"*"},
					"returnGeometry": {"true"}, "outSR": {"4326"},
				}, &fr)
				if err != nil {
					errc <- fmt.Errorf("%s: %w", ds.Title, err)
					cancel()
					return
				}
				fieldsOnce.Do(func() {
					for _, fl := range fr.Fields {
						fields = append(fields, fl.Name)
					}
				})
				select {
				case pages <- fr.Features:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(chunks)
		for i := 0; i < len(all); i += c.Chunk {
			select {
			case chunks <- all[i:min(i+c.Chunk, len(all))]:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(pages) }()

	// Single writer: owns the csv.Writer, so no mutex is needed.
	w := csv.NewWriter(f)
	rows, wroteHeader := 0, false
	lastLog := time.Now()
	for page := range pages {
		for _, ft := range page {
			if !wroteHeader {
				if len(fields) == 0 { // fields list missing: fall back to attribute keys
					for k := range ft.Attributes {
						fields = append(fields, k)
					}
					sort.Strings(fields)
				}
				_ = w.Write(append([]string{"X", "Y"}, fields...))
				wroteHeader = true
			}
			rec := make([]string, 0, len(fields)+2)
			x, y := point(ft)
			rec = append(rec, x, y)
			for _, name := range fields {
				rec = append(rec, cell(ft.Attributes[name]))
			}
			_ = w.Write(rec)
			rows++
		}
		if time.Since(lastLog) > 3*time.Second {
			c.Log("  %s: %d/%d", ds.Title, rows, len(all))
			lastLog = time.Now()
		}
	}
	w.Flush()
	if err := f.Close(); err != nil {
		return rows, err
	}
	select {
	case err := <-errc:
		return rows, err
	default:
	}
	if err := w.Error(); err != nil {
		return rows, err
	}
	if err := os.Rename(tmp, filepath.Join(dir, ds.File)); err != nil {
		return rows, err
	}
	c.Log("  %s: wrote %d rows to %s in %s", ds.Title, rows, ds.File, time.Since(start).Round(time.Millisecond))
	return rows, nil
}

func point(ft feature) (string, string) {
	g := ft.Geometry
	if g == nil {
		return "", ""
	}
	if g.X != nil && g.Y != nil && !math.IsNaN(*g.X) {
		return strconv.FormatFloat(*g.X, 'f', 6, 64), strconv.FormatFloat(*g.Y, 'f', 6, 64)
	}
	if len(g.Rings) > 0 && len(g.Rings[0]) > 0 { // polygon parcel: use ring centroid
		var sx, sy float64
		for _, p := range g.Rings[0] {
			sx += p[0]
			sy += p[1]
		}
		n := float64(len(g.Rings[0]))
		return strconv.FormatFloat(sx/n, 'f', 6, 64), strconv.FormatFloat(sy/n, 'f', 6, 64)
	}
	return "", ""
}

func cell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(t)
	}
}
