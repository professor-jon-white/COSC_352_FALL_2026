package ingest

import (
	"strings"
	"testing"
)

func TestNormalizeNeighborhood(t *testing.T) {
	tests := map[string]string{
		"Belair-Edison":         "BELAIR-EDISON",
		"  BELAIR-EDISON ":      "BELAIR-EDISON",
		"Armistead   Gardens\t": "ARMISTEAD GARDENS",
		"":                      "",
		"   ":                   "",
	}
	for in, want := range tests {
		if got := NormalizeNeighborhood(in); got != want {
			t.Errorf("NormalizeNeighborhood(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScannerSkipsMalformedRows(t *testing.T) {
	in := "A,B,C\n" +
		"1,x,keep\n" +
		"2\n" + // too short to reach column C
		"3,\"bad\"quote,y\n" + // CSV parse error
		"4,  y  ,keep\n"
	sc, err := NewScanner(strings.NewReader(in), "C", "B")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for sc.Next() {
		got = append(got, sc.Row()[0]+"/"+sc.Row()[1])
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"keep/x", "keep/y"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("rows = %v, want %v", got, want)
	}
	if sc.Stats.Read != 4 || sc.Stats.Malformed != 2 {
		t.Errorf("Read=%d Malformed=%d, want 4 and 2", sc.Stats.Read, sc.Stats.Malformed)
	}
	if len(sc.Stats.Examples) != 2 {
		t.Errorf("Examples = %q, want 2 entries", sc.Stats.Examples)
	}
}

func TestNewScannerMissingColumn(t *testing.T) {
	if _, err := NewScanner(strings.NewReader("A,B\n"), "A", "Z"); err == nil {
		t.Error("expected an error for missing column Z")
	}
}
