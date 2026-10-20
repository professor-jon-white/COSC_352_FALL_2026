package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type parseJob struct {
	Source  string
	Feature json.RawMessage
}
type record struct{ Source, Neighborhood, RequestType, Status string }

// RunPipeline is a bounded producer/worker/collector pipeline. Source fetches run
// concurrently, while a fixed-size worker pool parses each raw feature.
func RunPipeline(ctx context.Context, sources []Source, workers int, rawDir string) (Analysis, error) {
	if err := validateSources(sources); err != nil {
		return Analysis{}, err
	}
	if workers < 1 {
		return Analysis{}, errors.New("worker count must be positive")
	}
	jobs := make(chan parseJob, workers*8)
	parsed := make(chan record, workers*8)
	errCh := make(chan error, len(sources))
	var fetchWG sync.WaitGroup
	for _, source := range sources {
		fetchWG.Add(1)
		go func(s Source) {
			defer fetchWG.Done()
			if err := fetchSource(ctx, s, rawDir, jobs); err != nil {
				errCh <- fmt.Errorf("%s: %w", s.Name, err)
			}
		}(source)
	}
	go func() { fetchWG.Wait(); close(jobs) }()
	var workerWG sync.WaitGroup
	workerWG.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer workerWG.Done()
			for job := range jobs {
				r, ok := parseFeature(job.Source, job.Feature)
				if ok {
					parsed <- r
				}
			}
		}()
	}
	go func() { workerWG.Wait(); close(parsed) }()
	result := newAccumulator()
	for r := range parsed {
		result.add(r)
	}
	select {
	case err := <-errCh:
		return Analysis{}, err
	default:
	}
	if result.requestCount == 0 || result.vacantCount == 0 {
		return Analysis{}, errors.New("one or both sources returned no usable records")
	}
	return result.finish(), nil
}

func fetchSource(ctx context.Context, source Source, rawDir string, jobs chan<- parseJob) error {
	if source.LocalPath != "" {
		info, err := os.Stat(source.LocalPath)
		if err != nil {
			return err
		}
		paths := []string{source.LocalPath}
		if info.IsDir() {
			paths, err = filepath.Glob(filepath.Join(source.LocalPath, source.Key+"_page_*.geojson"))
			if err != nil {
				return err
			}
			if source.Key == "vacant" {
				snapshot := filepath.Join(source.LocalPath, "vacant_snapshot.geojson")
				if _, statErr := os.Stat(snapshot); statErr == nil {
					paths = append(paths, snapshot)
				}
			}
			if len(paths) == 0 {
				return fmt.Errorf("no saved GeoJSON files found for %s in %s", source.Key, source.LocalPath)
			}
		}
		for _, path := range paths {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var collection rawFeatureCollection
			if err := json.Unmarshal(b, &collection); err != nil {
				return fmt.Errorf("read GeoJSON %s: %w", filepath.Base(path), err)
			}
			for _, feature := range collection.Features {
				owned := append(json.RawMessage(nil), feature...)
				select {
				case jobs <- parseJob{source.Key, owned}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		return nil
	}
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 90 * time.Second}
	if source.PageSize <= 0 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		var collection rawFeatureCollection
		if err := json.Unmarshal(body, &collection); err != nil {
			return fmt.Errorf("decode GeoJSON snapshot: %w", err)
		}
		if collection.Type != "FeatureCollection" {
			return errors.New("unexpected response format in GeoJSON snapshot")
		}
		if err := os.WriteFile(filepath.Join(rawDir, source.Key+"_snapshot.geojson"), body, 0o644); err != nil {
			return fmt.Errorf("save raw snapshot: %w", err)
		}
		for _, feature := range collection.Features {
			owned := append(json.RawMessage(nil), feature...)
			select {
			case jobs <- parseJob{source.Key, owned}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	offset := 0
	for page := 0; ; page++ {
		values := url.Values{}
		where := source.Where
		if where == "" {
			where = "1=1"
		}
		values.Set("where", where)
		values.Set("outFields", "*")
		values.Set("returnGeometry", "true")
		values.Set("outSR", "4326")
		values.Set("f", "geojson")
		values.Set("resultOffset", strconv.Itoa(offset))
		values.Set("resultRecordCount", strconv.Itoa(source.PageSize))
		values.Set("orderByFields", source.SortField+" ASC")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL+"?"+values.Encode(), nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		var collection rawFeatureCollection
		if err := json.Unmarshal(body, &collection); err != nil {
			return fmt.Errorf("decode GeoJSON page %d: %w", page+1, err)
		}
		if collection.Type != "FeatureCollection" {
			return fmt.Errorf("unexpected response format on page %d", page+1)
		}
		var apiErr map[string]any
		if json.Unmarshal(body, &apiErr) == nil {
			if e, exists := apiErr["error"]; exists {
				return fmt.Errorf("ArcGIS API error: %v", e)
			}
		}
		if len(collection.Features) == 0 {
			break
		}
		file := filepath.Join(rawDir, fmt.Sprintf("%s_page_%05d.geojson", source.Key, page+1))
		if err := os.WriteFile(file, body, 0o644); err != nil {
			return fmt.Errorf("save raw page: %w", err)
		}
		for _, feature := range collection.Features {
			owned := append(json.RawMessage(nil), feature...)
			select {
			case jobs <- parseJob{source.Key, owned}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		offset += len(collection.Features)
		if !collection.Properties.Exceeded && len(collection.Features) < source.PageSize {
			break
		}
	}
	return nil
}

func parseFeature(source string, feature json.RawMessage) (record, bool) {
	var wrapper struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(feature, &wrapper) != nil {
		return record{}, false
	}
	get := func(keys ...string) string {
		for _, key := range keys {
			for field, raw := range wrapper.Properties {
				if strings.EqualFold(field, key) {
					var s string
					if json.Unmarshal(raw, &s) == nil {
						return strings.TrimSpace(s)
					}
					if string(raw) != "null" {
						return strings.TrimSpace(string(raw))
					}
				}
			}
		}
		return ""
	}
	r := record{Source: source, Neighborhood: cleanNeighborhood(get("Neighborhood", "NEIGHBORHOOD"))}
	if source == "requests" {
		r.RequestType = get("SRType")
		r.Status = get("SRStatus")
	}
	return r, true
}
