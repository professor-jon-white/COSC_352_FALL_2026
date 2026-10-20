package ingestion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDateFormats(t *testing.T) {
	expected := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC).UnixMilli()
	cases := map[string]string{
		"epoch_milliseconds": "1778754600000",
		"epoch_seconds":      "1778754600",
		"string_epoch":       `"1778754600000"`,
		"iso_utc":            `"2026-05-14T10:30:00Z"`,
		"iso_offset":         `"2026-05-14T06:30:00-04:00"`,
		"iso_no_timezone":    `"2026-05-14T10:30:00"`,
		"space_separator":    `"2026-05-14 10:30:00"`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			result := parseDate(map[string]json.RawMessage{"CreatedDate": json.RawMessage(raw)}, "CreatedDate")
			if result != expected {
				t.Fatalf("got %d, want %d", result, expected)
			}
		})
	}
}

func TestNullDate(t *testing.T) {
	if v := parseDate(map[string]json.RawMessage{"CloseDate": json.RawMessage("null")}, "CloseDate"); v != 0 {
		t.Fatalf("expected 0 for null, got %d", v)
	}
}

func TestRequestISODateFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requests.geojson")
	contents := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"SRRecordID":"X","SRType":"Trash","SRStatus":"Closed","CreatedDate":"2026-02-01T12:00:00Z","CloseDate":"2026-02-03T12:00:00Z"},"geometry":{"type":"Point","coordinates":[-76.6,39.3]}}]}`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	reqs, err := Requests(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].Closed-reqs[0].Created != 2*86400000 {
		t.Fatalf("expected one request with 2 day closure, got %+v", reqs)
	}
}
