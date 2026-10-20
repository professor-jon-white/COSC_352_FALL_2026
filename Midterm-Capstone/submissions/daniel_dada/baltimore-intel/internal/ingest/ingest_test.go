package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

const crimeCSV = `RowID,CrimeDateTime,Description,Neighborhood
1,8/29/2024 12:00:00 PM,BURGLARY,Penn-North
2,8/30/2024 11:45:00 PM,ROBBERY,PENN NORTH
3,1/1/2023 1:00:00 AM,LARCENY,Hampden
4,not-a-date,LARCENY,Hampden
5,9/1/2024 1:00:00 AM,AGG. ASSAULT,
6,9/2/2024 2:00:00 AM,COMMON ASSAULT,Hampden
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "crime.csv")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func checkStats(t *testing.T, name string, st Stats) {
	t.Helper()
	want := Stats{Read: 6, Kept: 3, BadDate: 1, WrongYear: 1, BlankKey: 1}
	if st != want {
		t.Errorf("%s stats = %+v, want %+v", name, st, want)
	}
}

func TestReadCrimeSequential(t *testing.T) {
	recs, st, err := ReadCrime(writeTemp(t, crimeCSV), 2024)
	if err != nil {
		t.Fatal(err)
	}
	checkStats(t, "sequential", st)
	if len(recs) != 3 {
		t.Errorf("got %d records, want 3", len(recs))
	}
}

func TestReadCrimeParallelMatchesSequential(t *testing.T) {
	recs, st, err := ReadCrimeParallel(writeTemp(t, crimeCSV), 2024)
	if err != nil {
		t.Fatal(err)
	}
	checkStats(t, "parallel", st)
	if len(recs) != 3 {
		t.Errorf("got %d records, want 3", len(recs))
	}
}

func TestMissingColumn(t *testing.T) {
	p := writeTemp(t, "RowID,Foo\n1,2\n")
	if _, _, err := ReadCrime(p, 2024); err == nil {
		t.Error("expected an error for a file missing required columns")
	}
}
