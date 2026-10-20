package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bishant/baltimore-engine/internal/model"
)

func TestParseTimeFormats(t *testing.T) {
	cases := map[string]int{ // input -> expected local hour
		"2024/03/05 21:15:00+00": 16, // UTC -> EST
		"2024-03-05T09:00:00":    9,  // zoneless = Baltimore time
		"3/5/2024 1:30:00 PM":    13,
		"1709672400000":          16, // epoch ms = 2024-03-05 21:00 UTC
	}
	for in, hour := range cases {
		got, ok := ParseTime(in)
		if !ok || got.Hour() != hour {
			t.Errorf("ParseTime(%q) = %v ok=%v, want hour %d", in, got, ok, hour)
		}
	}
	if got, _ := ParseTimeIn("3/5/2024 9:00:00 PM", time.UTC); got.Hour() != 16 {
		t.Errorf("UTC crime time not converted: %v", got)
	}
	if _, ok := ParseTime("not a date"); ok {
		t.Error("expected failure for garbage input")
	}
}

func TestSchemaAliases(t *testing.T) {
	s, err := ResolveSchema([]string{"\ufeffCrime_Date_Time", "NEIGHBORHOOD"},
		CrimeParser{}.Fields(), CrimeParser{}.Required())
	if err != nil || !s.Has("datetime") || !s.Has("neighborhood") {
		t.Fatalf("aliases not resolved: %v", err)
	}
	if _, err := ResolveSchema([]string{"foo"}, CrimeParser{}.Fields(), []string{"neighborhood"}); err == nil {
		t.Fatal("expected missing-column error")
	}
}

func TestLoadConcurrent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.csv")
	data := "CrimeDateTime,Neighborhood\n2024/01/01 12:00:00+00,canton\n2024/01/01 13:00:00+00,\n2024/01/01 14:00:00+00,  Fells   point \n"
	os.WriteFile(p, []byte(data), 0o644)
	recs, st, err := Load[model.CrimeRecord](p, CrimeParser{In: NewInterner()}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if st.RowsRead != 3 || st.RowsKept != 2 || st.RowsDropped != 1 {
		t.Fatalf("bad stats %+v", st)
	}
	names := map[string]bool{}
	for _, r := range recs {
		names[r.Neighborhood] = true
	}
	if !names["CANTON"] || !names["FELLS POINT"] {
		t.Fatalf("normalization failed: %v", names)
	}
}
