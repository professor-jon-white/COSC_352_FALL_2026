package requests311

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"capstone/internal/ingest"
)

const header = "SRRecordID,SRType,CreatedDate,SRStatus,CloseDate,Neighborhood,x,y\n"

func TestLoad(t *testing.T) {
	in := header +
		"a,HCD-Illegal Dumping,4/28/2025 4:31:03 PM,Closed,5/1/2025 10:18:12 AM,Broadway East,1,2\n" + // kept, closed
		"b,TRT-Traffic Calming,4/30/2025 6:18:55 PM,Open,,  carrollton ridge ,1,2\n" + // kept, open
		"c,TRT-Traffic Calming,12/31/2024 11:00:00 PM,Open,,Carrollton Ridge,1,2\n" + // out of year
		"d,ECC-Info,4/30/2025 9:51:25 PM,New,,\n" + // no neighborhood (short row)
		"e,ECC-Info,4/30/2025 9:51:25 PM,Closed,yesterday,Abell,1,2\n" // malformed close date

	reqs, st, err := Load(strings.NewReader(in), 2025)
	if err != nil {
		t.Fatal(err)
	}
	if st.Read != 5 || st.Kept != 2 || st.OutOfYear != 1 || st.NoNeighborhood != 1 || st.Malformed != 1 {
		t.Errorf("stats = %+v", st)
	}
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2", len(reqs))
	}
	if r := reqs[0]; r.Neighborhood != "BROADWAY EAST" || r.RequestType != "HCD-Illegal Dumping" || r.Status != "Closed" || r.ClosedAt.IsZero() {
		t.Errorf("reqs[0] = %+v", r)
	}
	if r := reqs[1]; r.Neighborhood != "CARROLLTON RIDGE" || !r.ClosedAt.IsZero() {
		t.Errorf("reqs[1] = %+v, want CARROLLTON RIDGE with zero ClosedAt", r)
	}
}

func TestLoadConcurrentMatchesLoad(t *testing.T) {
	var b strings.Builder
	b.WriteString(header)
	for i := range 500 {
		closed := ""
		if i%3 > 0 {
			closed = fmt.Sprintf("%d/%d/2025 9:00:00 AM", i%12+1, i%28+1)
		}
		fmt.Fprintf(&b, "r%d,Type %d,%d/%d/2025 %d:00:00 AM,Closed,%s,\"Hood, %d\",1,2\n", i, i%5, i%12+1, i%28+1, i%12+1, closed, i%9)
	}
	b.WriteString("x,Type,garbage,New,,Abell\n")
	in := b.String()

	want, wantStats, err := Load(strings.NewReader(in), 2025)
	if err != nil {
		t.Fatal(err)
	}
	got, gotStats, err := LoadConcurrent(strings.NewReader(in), 2025, ingest.ConcurrentOptions{Workers: 4, ChunkSize: 512})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotStats, wantStats) {
		t.Errorf("LoadConcurrent differs from Load: %d vs %d requests, stats %+v vs %+v", len(got), len(want), gotStats, wantStats)
	}
}
