package crime

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"capstone/internal/ingest"
)

const header = "RowID,CrimeDateTime,Description,Neighborhood,Inside_Outside,x,y\n"

func TestLoad(t *testing.T) {
	in := header +
		"1,6/1/2025 5:00:00 PM,COMMON ASSAULT,mount vernon ,O,1,2\n" + // kept
		"2,1/1/2026 3:00:00 AM,BURGLARY,Frankford,I,1,2\n" + // 10 PM Dec 31 local: kept
		"3,1/1/2025 4:00:00 AM,ROBBERY,Frankford,O,1,2\n" + // 11 PM Dec 31, 2024 local: out of year
		"4,6/1/2025 5:00:00 PM,ROBBERY,,O,1,2\n" + // no neighborhood
		"5,not a date,ROBBERY,Frankford,I\n" // malformed (and short, like ungeocoded rows)

	crimes, st, err := Load(strings.NewReader(in), 2025)
	if err != nil {
		t.Fatal(err)
	}
	if st.Read != 5 || st.Kept != 2 || st.OutOfYear != 1 || st.NoNeighborhood != 1 || st.Malformed != 1 {
		t.Errorf("stats = %+v", st)
	}
	if st.Neighborhoods() != 2 {
		t.Errorf("Neighborhoods() = %d, want 2", st.Neighborhoods())
	}
	if len(crimes) != 2 {
		t.Fatalf("got %d crimes, want 2", len(crimes))
	}
	if c := crimes[0]; c.Neighborhood != "MOUNT VERNON" || c.CrimeType != "COMMON ASSAULT" || c.OccurredAt.Hour() != 13 || !c.Outdoor {
		t.Errorf("crimes[0] = %+v, want outdoor MOUNT VERNON / COMMON ASSAULT at 1 PM EDT", c)
	}
	if c := crimes[1]; c.Outdoor || c.OccurredAt.Year() != 2025 || c.OccurredAt.Month() != 12 || c.OccurredAt.Day() != 31 {
		t.Errorf("crimes[1].OccurredAt = %v, want Dec 31, 2025 local", c.OccurredAt)
	}
}

func TestLoadConcurrentMatchesLoad(t *testing.T) {
	var b strings.Builder
	b.WriteString(header)
	for i := range 500 {
		fmt.Fprintf(&b, "%d,%d/%d/2025 %d:00:00 PM,TYPE %d,Hood %d,%c,1,2\n", i, i%12+1, i%28+1, i%12+1, i%7, i%9, "IO"[i%2])
	}
	b.WriteString("x,garbage,ROBBERY,Abell\n")
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
		t.Errorf("LoadConcurrent differs from Load: %d vs %d crimes, stats %+v vs %+v", len(got), len(want), gotStats, wantStats)
	}
}
