package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func readRows(path string, required ...string) ([]Row, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	var p Page
	if e = json.NewDecoder(f).Decode(&p); e != nil {
		return nil, e
	}
	if len(p.Error) > 0 {
		return nil, fmt.Errorf("ArcGIS error: %s", p.Error)
	}
	rows := make([]Row, 0, len(p.Features))
	for _, v := range p.Features {
		for _, k := range required {
			if _, ok := v.Attributes[k]; !ok {
				return nil, fmt.Errorf("%s: missing %s field; check source schema", path, k)
			}
		}
		rows = append(rows, v.Attributes)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s has no features", path)
	}
	return rows, nil
}
func save(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}
func download(ctx context.Context, base, oid string, limit int, where string) (Page, error) {
	out := Page{}
	seen := map[string]bool{}
	client := &http.Client{Timeout: 60 * time.Second}
	offset := 0
	for {
		n := 2000
		if limit > 0 && limit-offset < n {
			n = limit - offset
		}
		if n <= 0 {
			break
		}
		q := url.Values{"f": {"json"}, "where": {where}, "outFields": {"*"}, "returnGeometry": {"false"}, "orderByFields": {oid + " ASC"}, "resultOffset": {fmt.Sprint(offset)}, "resultRecordCount": {fmt.Sprint(n)}}
		req, e := http.NewRequestWithContext(ctx, "GET", base+"/query?"+q.Encode(), nil)
		if e != nil {
			return out, e
		}
		resp, e := client.Do(req)
		if e != nil {
			return out, e
		}
		var p Page
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return out, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		e = json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&p)
		resp.Body.Close()
		if e != nil {
			return out, e
		}
		if len(p.Error) > 0 {
			return out, fmt.Errorf("ArcGIS: %s", p.Error)
		}
		for _, f := range p.Features {
			id := fmt.Sprint(f.Attributes[oid])
			if id == "<nil>" || seen[id] {
				return out, fmt.Errorf("missing or repeated object ID %s; source may have changed", id)
			}
			seen[id] = true
		}
		out.Features = append(out.Features, p.Features...)
		offset += len(p.Features)
		if !p.More {
			break
		}
		if len(p.Features) == 0 {
			return out, fmt.Errorf("pagination stalled")
		}
	}
	if len(out.Features) == 0 {
		return out, fmt.Errorf("empty dataset")
	}
	return out, nil
}

func source(ctx context.Context, base, field, start, end string, limit int) (Page, Source, error) {
	where := fmt.Sprintf("%s >= TIMESTAMP '%s 00:00:00' AND %s < TIMESTAMP '%s 00:00:00'", field, start, field, end)
	spec := Source{URL: base, DateField: field, IDField: "RowID", Where: where}
	q := url.Values{"f": {"json"}, "where": {where}, "returnCountOnly": {"true"}}
	req, e := http.NewRequestWithContext(ctx, "GET", base+"/query?"+q.Encode(), nil)
	if e != nil {
		return Page{}, spec, e
	}
	client := http.Client{Timeout: 60 * time.Second}
	resp, e := client.Do(req)
	if e != nil {
		return Page{}, spec, e
	}
	defer resp.Body.Close()
	var count struct {
		Count *int            `json:"count"`
		Error json.RawMessage `json:"error"`
	}
	if resp.StatusCode != 200 {
		return Page{}, spec, fmt.Errorf("count HTTP %d", resp.StatusCode)
	}
	if e = json.NewDecoder(resp.Body).Decode(&count); e != nil {
		return Page{}, spec, e
	}
	if count.Count == nil {
		return Page{}, spec, fmt.Errorf("count failed: %s", count.Error)
	}
	spec.Available = *count.Count
	p, e := download(ctx, base, "RowID", limit, where)
	spec.Rows = len(p.Features)
	spec.Truncated = spec.Rows < spec.Available
	return p, spec, e
}
func fetch(dir, start, end string, limit int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var a, b Page
	var sa, sb Source
	var ea, eb error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a, sa, ea = source(ctx, crimeURL, "CrimeDateTime", start, end, limit)
		if ea != nil {
			cancel()
		}
	}()
	go func() {
		defer wg.Done()
		b, sb, eb = source(ctx, arrestURL, "ArrestDateTime", start, end, limit)
		if eb != nil {
			cancel()
		}
	}()
	wg.Wait()
	if ea != nil {
		return ea
	}
	if eb != nil {
		return eb
	}
	if e := save(filepath.Join(dir, "crime.json"), a); e != nil {
		return e
	}
	if e := save(filepath.Join(dir, "arrests.json"), b); e != nil {
		return e
	}
	return save(filepath.Join(dir, "manifest.json"), Manifest{Retrieved: time.Now().UTC().Format(time.RFC3339), Start: start, End: end, Limit: limit, Crime: sa, Arrest: sb})
}
