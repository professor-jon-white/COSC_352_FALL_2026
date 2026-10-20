// Command fetch downloads selected raw ArcGIS feature attributes as JSON Lines.
// It preserves source values and does not perform aggregation.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"baltimore-capstone/internal/model"
)

type source struct {
	Name     string
	URL      string
	Where    string
	Fields   string
	ID       string
	PageSize int
}
type metadata struct {
	Name     string `json:"name"`
	URL      string `json:"source_url"`
	Where    string `json:"where"`
	Fields   string `json:"fields"`
	Started  string `json:"started_utc"`
	Finished string `json:"finished_utc"`
	Expected int    `json:"source_count_before"`
	After    int    `json:"source_count_after"`
	Records  int    `json:"records"`
	Pages    int    `json:"pages"`
	SHA256   string `json:"sha256"`
	File     string `json:"file"`
}
type response struct {
	Count    int               `json:"count"`
	Features []json.RawMessage `json:"features"`
	More     bool              `json:"exceededTransferLimit"`
	Error    *struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Details []string `json:"details"`
	} `json:"error"`
}

func get(client *http.Client, base string, params url.Values) (response, error) {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}
		var result response
		resp, err := client.Get(base + "/query?" + params.Encode())
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			last = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		err = json.NewDecoder(io.LimitReader(resp.Body, 64*1024*1024)).Decode(&result)
		resp.Body.Close()
		if err != nil {
			last = err
			continue
		}
		if result.Error != nil {
			last = fmt.Errorf("ArcGIS %d: %s %v", result.Error.Code, result.Error.Message, result.Error.Details)
			continue
		}
		return result, nil
	}
	return response{}, fmt.Errorf("%s: %w", base, last)
}

func download(dir string, s source) (m metadata, err error) {
	m = metadata{Name: s.Name, URL: s.URL, Where: s.Where, Fields: s.Fields, Started: time.Now().UTC().Format(time.RFC3339), File: s.Name + ".jsonl"}
	client := &http.Client{Timeout: 90 * time.Second}
	countParams := url.Values{"f": {"json"}, "where": {s.Where}, "returnCountOnly": {"true"}}
	c, err := get(client, s.URL, countParams)
	if err != nil {
		return m, err
	}
	m.Expected = c.Count
	if c.Count == 0 {
		return m, fmt.Errorf("%s query returned zero rows", s.Name)
	}
	temp := filepath.Join(dir, m.File+".partial")
	f, err := os.Create(temp)
	if err != nil {
		return m, err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(temp)
		}
	}()
	h := sha256.New()
	w := bufio.NewWriter(io.MultiWriter(f, h))
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	seen := map[int64]bool{}
	for offset := 0; ; {
		p := url.Values{"f": {"json"}, "where": {s.Where}, "outFields": {s.Fields},
			"returnGeometry": {"false"}, "orderByFields": {s.ID + " ASC"},
			"resultOffset": {strconv.Itoa(offset)}, "resultRecordCount": {strconv.Itoa(s.PageSize)}}
		r, e := get(client, s.URL, p)
		if e != nil {
			return m, e
		}
		if len(r.Features) == 0 {
			if r.More {
				return m, fmt.Errorf("%s empty page marked incomplete", s.Name)
			}
			break
		}
		for _, raw := range r.Features {
			var feature struct {
				Attributes map[string]json.RawMessage `json:"attributes"`
			}
			if e = json.Unmarshal(raw, &feature); e != nil {
				return m, e
			}
			var id int64
			if e = json.Unmarshal(feature.Attributes[s.ID], &id); e != nil {
				return m, fmt.Errorf("missing source object ID: %w", e)
			}
			if seen[id] {
				return m, fmt.Errorf("duplicate source object ID %d; live data may have shifted, rerun", id)
			}
			seen[id] = true
			if e = enc.Encode(raw); e != nil {
				return m, e
			}
			m.Records++
		}
		m.Pages++
		offset += len(r.Features)
		fmt.Fprintf(os.Stderr, "%s: %d / %d rows\n", s.Name, m.Records, m.Expected)
		if !r.More {
			break
		}
	}
	c, err = get(client, s.URL, countParams)
	if err != nil {
		return m, err
	}
	m.After = c.Count
	if m.Records != m.Expected || m.Records != m.After {
		return m, fmt.Errorf("%s count changed or download truncated: before=%d downloaded=%d after=%d; rerun", s.Name, m.Expected, m.Records, m.After)
	}
	if err = w.Flush(); err != nil {
		return m, err
	}
	if err = f.Close(); err != nil {
		return m, err
	}
	m.SHA256 = hex.EncodeToString(h.Sum(nil))
	m.Finished = time.Now().UTC().Format(time.RFC3339)
	err = os.Rename(temp, filepath.Join(dir, m.File))
	return m, err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Download failed:", err)
		os.Exit(1)
	}
}
func run() error {
	dir := flag.String("out", "data", "destination directory; use a new directory for refreshes")
	flag.Parse()
	if err := os.MkdirAll(*dir, 0755); err != nil {
		return err
	}
	sources := []source{
		{"requests", model.RequestURL, "CreatedDate >= TIMESTAMP '2025-10-01 00:00:00' AND CreatedDate < TIMESTAMP '2026-01-01 00:00:00'",
			"ESRI_OID,ServiceRequestNum,CreatedDate,CloseDate,SRType,SRStatus,Agency,Neighborhood,IsDeleted", "ESRI_OID", 2000},
		{"vacancies", model.VacancyURL, "1=1",
			"OBJECTID,NoticeNum,DateNotice,BLOCKLOT,Address,Neighborhood", "OBJECTID", 1000},
	}
	meta := make([]metadata, len(sources))
	errs := make([]error, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Add(1)
		go func(i int, s source) { defer wg.Done(); meta[i], errs[i] = download(*dir, s) }(i, s)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	payload, err := json.MarshalIndent(struct {
		Description string     `json:"description"`
		Datasets    []metadata `json:"datasets"`
	}{"Selected raw ArcGIS feature attributes; one unchanged feature per JSONL line. Counts checked before and after; not an atomic server snapshot. See README for limitations.", meta}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*dir, "manifest.json"), append(payload, '\n'), 0644); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Downloads complete. Provenance and SHA-256 hashes recorded in manifest.json.")
	return nil
}
