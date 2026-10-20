package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"baltimore-intel/internal/ingest"
	"baltimore-intel/internal/model"
)

// fakeArcGIS imitates an ArcGIS FeatureServer layer's query endpoint.
func fakeArcGIS(t *testing.T, total int, calls *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if err := r.ParseForm(); err != nil || !strings.HasSuffix(r.URL.Path, "/query") {
			http.Error(w, "bad", 400)
			return
		}
		if r.Form.Get("where") == "BROKEN" { // ArcGIS reports errors as HTTP 200
			fmt.Fprint(w, `{"error":{"code":400,"message":"Invalid query","details":["bad where"]}}`)
			return
		}
		if r.Form.Get("returnIdsOnly") == "true" {
			ids := make([]int, total)
			for i := range ids {
				ids[i] = total - i // unsorted on purpose
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"objectIdFieldName": "RowID", "objectIds": ids})
			return
		}
		var feats []map[string]any
		for _, s := range strings.Split(r.Form.Get("objectIds"), ",") {
			var id int
			fmt.Sscan(s, &id)
			feats = append(feats, map[string]any{
				"attributes": map[string]any{
					"RowID": id, "CrimeDateTime": 1647299700000 + int64(id)*60000,
					"Description": "AGG. ASSAULT", "Neighborhood": "Canton", "Total_Incidents": 1,
				},
				"geometry": map[string]any{"x": -76.5747, "y": 39.2808},
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": []map[string]string{{"name": "RowID"}, {"name": "CrimeDateTime"}, {"name": "Description"},
				{"name": "Neighborhood"}, {"name": "Total_Incidents"}},
			"features": feats,
		})
	}))
}

func TestFetchWritesParsableCSV(t *testing.T) {
	var calls atomic.Int32
	srv := fakeArcGIS(t, 1234, &calls)
	defer srv.Close()
	dir := t.TempDir()
	c := NewClient(4)
	c.Chunk = 100
	c.Log = func(string, ...any) {}
	n, err := c.Fetch(context.Background(), dir, Dataset{Title: "test", LayerURL: srv.URL + "/FeatureServer/0",
		Where: "1=1", File: "part1_crime.csv"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1234 {
		t.Fatalf("wrote %d rows, want 1234", n)
	}
	if got := calls.Load(); got != 1+13 { // 1 ID listing + ceil(1234/100) chunk requests
		t.Fatalf("%d HTTP calls, want 14", got)
	}
	// The fetched raw file must flow straight into the ingest layer.
	ds := model.NewDataset()
	src, _ := ingest.NewSource(ingest.KindCrime)
	st, err := src.Load(context.Background(), dir, 3, ds)
	if err != nil {
		t.Fatal(err)
	}
	if st.Kept != 1234 || !ds.Crimes[0].Violent || ds.Crimes[0].Lat == 0 {
		t.Fatalf("ingest of fetched file: %+v first=%+v", st, ds.Crimes[0])
	}
}

func TestFetchSamplesAndReportsErrors(t *testing.T) {
	var calls atomic.Int32
	srv := fakeArcGIS(t, 5000, &calls)
	defer srv.Close()
	c := NewClient(3)
	c.Log = func(string, ...any) {}
	n, err := c.Fetch(context.Background(), t.TempDir(), Dataset{Title: "s", LayerURL: srv.URL + "/L",
		Where: "1=1", File: "x.csv", MaxRows: 700})
	if err != nil || n != 700 {
		t.Fatalf("sampled fetch: n=%d err=%v", n, err)
	}
	_, err = c.Fetch(context.Background(), t.TempDir(), Dataset{Title: "b", LayerURL: srv.URL + "/L", Where: "BROKEN", File: "y.csv"})
	if err == nil || !strings.Contains(err.Error(), "Invalid query") {
		t.Fatalf("want ArcGIS error surfaced, got %v", err)
	}
}

func TestCatalog(t *testing.T) {
	cat := Catalog(2022, 1000)
	if len(cat) != 4 {
		t.Fatal("want 4 datasets")
	}
	if !strings.HasSuffix(cat[1].LayerURL, "311_Customer_Service_Requests_Yearly/FeatureServer/0") {
		t.Errorf("2022 311 layer = %s", cat[1].LayerURL)
	}
	if !strings.HasSuffix(service311Layer("b", 2018), "/FeatureServer/4") {
		t.Error("2018 should be layer 4 of the yearly service")
	}
	for _, d := range cat {
		if ingest.FileFor(d.Kind) != d.File {
			t.Errorf("%s: fetch writes %s but ingest reads %s", d.Kind, d.File, ingest.FileFor(d.Kind))
		}
	}
}
