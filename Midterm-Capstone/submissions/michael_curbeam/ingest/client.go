package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// arcgisQuery describes a paged query against an ArcGIS FeatureServer layer,
// the REST API behind Open Baltimore datasets.
type arcgisQuery struct {
	LayerURL   string // e.g. .../FeatureServer/0
	Where      string // SQL where clause
	OrderBy    string // optional, makes paging stable
	PageSize   int    // records per request (server max is 1000-2000)
	MaxRecords int    // overall cap; 0 means fetch everything
}

type arcgisResponse[T any] struct {
	Features []struct {
		Attributes T `json:"attributes"`
	} `json:"features"`
	ExceededTransferLimit bool `json:"exceededTransferLimit"`
	Error                 *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// fetchLayer pages through an ArcGIS layer and returns the decoded attributes.
func fetchLayer[T any](ctx context.Context, q arcgisQuery) ([]T, error) {
	var all []T
	for offset := 0; ; offset += q.PageSize {
		size := q.PageSize
		if q.MaxRecords > 0 && len(all)+size > q.MaxRecords {
			size = q.MaxRecords - len(all)
		}

		params := url.Values{
			"where":             {q.Where},
			"outFields":         {"*"},
			"returnGeometry":    {"false"},
			"resultOffset":      {strconv.Itoa(offset)},
			"resultRecordCount": {strconv.Itoa(size)},
			"f":                 {"json"},
		}
		if q.OrderBy != "" {
			params.Set("orderByFields", q.OrderBy)
		}

		var page arcgisResponse[T]
		if err := getJSON(ctx, q.LayerURL+"/query?"+params.Encode(), &page); err != nil {
			return nil, err
		}
		if page.Error != nil {
			return nil, fmt.Errorf("arcgis error %d: %s", page.Error.Code, page.Error.Message)
		}
		for _, f := range page.Features {
			all = append(all, f.Attributes)
		}

		done := !page.ExceededTransferLimit || len(page.Features) == 0
		if q.MaxRecords > 0 && len(all) >= q.MaxRecords {
			done = true
		}
		if done {
			return all, nil
		}
	}
}

// getJSON performs a GET request and decodes the JSON body into out.
func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
