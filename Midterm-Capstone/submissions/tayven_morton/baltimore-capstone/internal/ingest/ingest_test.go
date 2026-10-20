package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"baltimore-capstone/internal/model"
)

func fixture(t *testing.T, features []model.Feature) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.jsonl")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	e := json.NewEncoder(f)
	for _, a := range features {
		if err = e.Encode(a); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	return p
}
func TestDedupQualityAndWorkerConsistency(t *testing.T) {
	date, _ := time.Parse("2006-01-02", "2025-10-01")
	created := date.UnixMilli()
	a := model.Attributes{RequestID: "1", Area: " Area  A ", Created: &created, Type: "Trash"}
	b := a
	b.RequestID = "2"
	b.Area = "Area B"
	missing := a
	missing.RequestID = "3"
	missing.Area = ""
	features := []model.Feature{{Attributes: a}, {Attributes: a}, {Attributes: b}, {Attributes: missing}}
	p := fixture(t, features)
	one, err := Load(p, "requests", 1)
	if err != nil {
		t.Fatal(err)
	}
	many, err := Load(p, "requests", 8)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, many) {
		t.Fatal("worker count changes output")
	}
	if one.Quality.Accepted != 2 || one.Quality.Duplicates != 1 || one.Quality.MissingNeighborhood != 1 {
		t.Fatal(one.Quality)
	}
	if one.Areas["AREA A"].Types["TRASH"] != 1 {
		t.Fatal("normalization/count")
	}
}
func TestVacancyDedup(t *testing.T) {
	p := fixture(t, []model.Feature{
		{Attributes: model.Attributes{BlockLot: "123", Area: "A"}},
		{Attributes: model.Attributes{BlockLot: "123", Area: "A", NoticeID: "different notice"}},
		{Attributes: model.Attributes{Address: "1 Main", Area: "B"}},
		{Attributes: model.Attributes{Area: "A"}},
	})
	d, err := Load(p, "vacancies", 3)
	if err != nil {
		t.Fatal(err)
	}
	if d.Quality.Accepted != 2 || d.Quality.Duplicates != 1 || d.Quality.MissingID != 1 {
		t.Fatal(d.Quality)
	}
}
func TestInvalidInputs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.jsonl")
	for _, content := range []string{"", "not json\n", "{}\n"} {
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(p, "requests", 2); err == nil {
			t.Fatal("expected error", content)
		}
	}
	if _, err := Load(p, "requests", 0); err == nil {
		t.Fatal("invalid workers")
	}
	if _, err := Load(p, "unknown", 1); err == nil {
		t.Fatal("invalid kind")
	}
	if _, err := Load(p+".missing", "requests", 1); err == nil {
		t.Fatal("missing file")
	}
}
func TestLargePipelineNoDeadlock(t *testing.T) {
	date, _ := time.Parse("2006-01-02", "2025-11-01")
	n := date.UnixMilli()
	features := make([]model.Feature, 10000)
	for i := range features {
		features[i].Attributes = model.Attributes{RequestID: strconv.Itoa(i + 1), Area: "A", Created: &n}
	}
	p := fixture(t, features)
	d, err := Load(p, "requests", 4)
	if err != nil {
		t.Fatal(err)
	}
	if d.Quality.Accepted != 10000 {
		t.Fatal(d.Quality)
	}
}
