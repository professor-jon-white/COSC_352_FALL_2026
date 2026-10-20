package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJoinAndWorkerConsistency(t *testing.T) {
	c := []Row{{"CCNumber": " abc ", "Neighborhood": "North", "Description": "ROBBERY", "Weapon": "HANDGUN", "CrimeDateTime": float64(1000)}, {"CCNumber": "ABC", "Neighborhood": "North", "Description": "ROBBERY", "Weapon": "HANDGUN", "CrimeDateTime": float64(1000)}, {"CCNumber": "", "CrimeDateTime": nil}, {"CCNumber": "negative", "CrimeDateTime": float64(4000)}}
	a := []Row{{"IncidentNumber": "ABC", "ArrestNumber": "A1", "Neighborhood": "South", "ArrestDateTime": float64(3601000)}, {"IncidentNumber": "ABC", "ArrestNumber": "A1", "Neighborhood": "South", "ArrestDateTime": float64(7201000)}, {"IncidentNumber": "", "ArrestNumber": "A2"}, {"IncidentNumber": "NEGATIVE", "ArrestNumber": "A3", "ArrestDateTime": float64(3000)}}
	one := analyze(c, a, 1)
	many := analyze(c, a, 4)
	if !reflect.DeepEqual(one, many) {
		t.Fatal("worker results differ")
	}
	if one.MatchedIncidents != 2 || one.UniqueIncidents != 2 || one.UniqueArrests != 3 || len(one.Lags) != 1 || one.Lags[0] != 1 || one.NegativeLag != 1 {
		t.Fatalf("unexpected aggregate: %+v", one)
	}
	if one.Categories["ROBBERY"].Linked != 2 || one.Neighborhoods["SOUTH"].Arrests != 2 || one.MissingCrimeID != 1 || one.MissingArrestID != 1 {
		t.Fatal("join accounting")
	}
}
func TestNoMissingIDMatches(t *testing.T) {
	r := analyze([]Row{{"CCNumber": "UNKNOWN"}}, []Row{{"IncidentNumber": "UNKNOWN"}}, 2)
	if r.MatchedIncidents != 0 || r.Categories["UNKNOWN"].Linked != 0 {
		t.Fatal("missing IDs linked")
	}
}
func TestDownloadPagination(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("where") != "X=1" {
			t.Error("filter lost")
		}
		if r.URL.Query().Get("resultOffset") == "0" {
			fmt.Fprint(w, `{"features":[{"attributes":{"RowID":1}}],"exceededTransferLimit":true}`)
		} else {
			fmt.Fprint(w, `{"features":[{"attributes":{"RowID":2}}]}`)
		}
	}))
	defer s.Close()
	p, e := download(context.Background(), s.URL, "RowID", 0, "X=1")
	if e != nil || calls != 2 || len(p.Features) != 2 {
		t.Fatalf("pagination: %v", e)
	}
}
func TestRepeatedPagesRejected(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[{"attributes":{"RowID":1}}],"exceededTransferLimit":true}`)
	}))
	defer s.Close()
	if _, e := download(context.Background(), s.URL, "RowID", 0, "1=1"); e == nil {
		t.Fatal("repeated page accepted")
	}
}
func TestSchemaFailure(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte(`{"features":[{"attributes":{"Wrong":1}}]}`), 0600)
	if _, e := readRows(p, "CCNumber"); e == nil {
		t.Fatal("missing schema accepted")
	}
}

func TestSelectionAndNeighborhoodProfiles(t *testing.T) {
	r := newResult()
	r.Neighborhoods["UNKNOWN"] = &Cell{Name: "UNKNOWN", Arrests: 100}
	r.Neighborhoods["A"] = &Cell{Name: "A", Arrests: 5, Crimes: 2}
	r.Neighborhoods["B"] = &Cell{Name: "B", Arrests: 0, Crimes: 3}
	chosen := topKnown(r, "arrests")
	if chosen["UNKNOWN"] || chosen["B"] || !chosen["A"] {
		t.Fatal("selection must exclude missing and zero arrest groups")
	}
	var b strings.Builder
	profile(&b, r, map[string]map[string]int{"A": {"ROBBERY": 2}, "B": {"ROBBERY": 1, "BURGLARY": 2}}, chosen)
	if !strings.Contains(b.String(), "2 selected, 3 elsewhere") || !strings.Contains(b.String(), "66.7%") {
		t.Fatal(b.String())
	}
}
func TestEasternMonth(t *testing.T) {
	if month(float64(1738369800000)) != "2025-01" {
		t.Fatal("Eastern month boundary")
	}
}
