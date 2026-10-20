package ingest

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// scanAll is a stand-in loader: it keeps every row, rejects rows whose second
// column is "bad", and records the first column as the neighborhood.
func scanAll(sc *Scanner) []string {
	var out []string
	for sc.Next() {
		row := sc.Row()
		if row[1] == "bad" {
			sc.Malformed(errors.New("bad value"))
			continue
		}
		sc.Stats.Keep(row[0])
		out = append(out, strings.Join(row, "|"))
	}
	return out
}

func sequential(t *testing.T, in string) ([]string, Stats) {
	t.Helper()
	sc, err := NewScanner(strings.NewReader(in), "A", "B")
	if err != nil {
		t.Fatal(err)
	}
	rows := scanAll(sc)
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rows, sc.Stats
}

// Every chunk size from 1 byte up, with several worker counts, must reproduce
// the sequential rows, order, and stats exactly, including the line numbers
// in malformed-row examples.
func TestConcurrentMatchesSequential(t *testing.T) {
	inputs := map[string]string{
		"simple":           "A,B\nx,1\ny,2\nz,3\n",
		"no final newline": "A,B\nx,1\ny,2",
		"crlf":             "A,B\r\nx,1\r\ny,2\r\n",
		"quoted newline":   "A,B\n\"multi\nline\",1\ny,\"has \"\"quotes\"\", and comma\"\nz,3\n",
		"malformed": "A,B\n" +
			"x,1\n" +
			"bare\"quote,2\n" + // ErrBareQuote: csv skips the rest of the line
			"\"extra\"quote,3\n" + // ErrQuote after a closing quote
			"short\n" + // too few fields
			"y,bad\n" + // rejected by the loader
			"\"spans\nlines\"x,4\n" + // ErrQuote in a multi-line field
			"z,5\n",
		"header only": "A,B\n",
		"quoted crlf": "A,B\r\n\"x\",\"y\"\r\n\"multi\r\nline\",\"z\"\r\nw,1\r\n",
		"odd quoting": "A,B\n" +
			"x, \"leading space\"\n" + // quote after a space is a bare quote
			"\"x\"\r,1\n" + // lone CR after a closing quote
			"x\r\"y\",1\n" + // quote after a lone CR mid-field
			"\"\"\"\",\"\"\n" + // field that is just an escaped quote, then an empty quoted field
			"z,\"unterminated\n" +
			"more,2\n",
	}
	for name, in := range inputs {
		wantRows, wantStats := sequential(t, in)
		for size := 1; size <= len(in)+1; size++ {
			for _, workers := range []int{1, 3} {
				rows, stats, err := Concurrent(strings.NewReader(in), []string{"A", "B"},
					ConcurrentOptions{Workers: workers, ChunkSize: size}, scanAll)
				if err != nil {
					t.Fatalf("%s size=%d workers=%d: %v", name, size, workers, err)
				}
				if !reflect.DeepEqual(rows, wantRows) && !(len(rows) == 0 && len(wantRows) == 0) {
					t.Errorf("%s size=%d workers=%d: rows\n got %q\nwant %q", name, size, workers, rows, wantRows)
				}
				if !reflect.DeepEqual(stats, wantStats) {
					t.Errorf("%s size=%d workers=%d: stats\n got %+v\nwant %+v", name, size, workers, stats, wantStats)
				}
			}
		}
	}
}

func TestConcurrentMissingColumn(t *testing.T) {
	_, _, err := Concurrent(strings.NewReader("A,B\nx,1\n"), []string{"A", "Z"}, ConcurrentOptions{}, scanAll)
	if err == nil {
		t.Error("expected an error for missing column Z")
	}
}
