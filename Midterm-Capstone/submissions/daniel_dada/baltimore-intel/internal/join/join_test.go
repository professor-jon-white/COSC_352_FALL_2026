package join

import (
	"reflect"
	"testing"
	"time"

	"baltimore-intel/internal/model"
)

func TestClassify(t *testing.T) {
	cases := map[string]Category{
		"ECC-Information Request":    CatExcluded,
		"SW-Rat Rubout Proactive":    CatExcluded,
		"BGE-StLight(s) Out":         CatLighting,
		"TRM-Street Light Out":       CatLighting,
		"SW-Dirty Alley":             CatSanitation,
		"HCD-Illegal Dumping":        CatSanitation,
		"TRT-Traffic Signal Repairs": CatTransport,
		"TRS-Parking Complaint":      CatTransport,
		"WW-Water Leak (Exterior)":   CatOther,
	}
	for in, want := range cases {
		if got := Classify(in); got != want {
			t.Errorf("Classify(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsViolent(t *testing.T) {
	for _, d := range []string{"HOMICIDE", "ROBBERY - CARJACKING", "AGG. ASSAULT", "SHOOTING", "RAPE"} {
		if !IsViolent(d) {
			t.Errorf("%q should be violent", d)
		}
	}
	for _, d := range []string{"COMMON ASSAULT", "LARCENY", "BURGLARY", "AUTO THEFT"} {
		if IsViolent(d) {
			t.Errorf("%q should not be violent", d)
		}
	}
}

func at(m time.Month, h int) time.Time {
	return time.Date(2024, m, 5, h, 0, 0, 0, time.UTC)
}

func TestBuildKnownAnswers(t *testing.T) {
	crimes := []model.CrimeRecord{
		{When: at(1, 17), Description: "ROBBERY", Neighborhood: "PENN NORTH"},
		{When: at(2, 17), Description: "AGG. ASSAULT", Neighborhood: "PENN NORTH"},
		{When: at(3, 9), Description: "BURGLARY", Neighborhood: "PENN NORTH"},
	}
	reqs := []model.ServiceRequest{
		{Created: at(1, 8), Type: "ECC-Information Request", Neighborhood: "PENN NORTH"},
		{Created: at(1, 8), Type: "SW-Dirty Alley", Neighborhood: "PENN NORTH"},
		{Created: at(1, 21), Type: "BGE-StLight(s) Out", Neighborhood: "PENN NORTH"},
		{Created: at(2, 8), Type: "TRT-Traffic Signal Repairs", Neighborhood: "PENN NORTH"},
	}
	for name, d := range map[string]*Dataset{
		"sequential": Build(crimes, reqs),
		"parallel":   BuildParallel(crimes, reqs),
	} {
		s := d.ByHood["PENN NORTH"]
		if s == nil {
			t.Fatalf("%s: PENN NORTH missing", name)
		}
		want := Stats{Crime: 3, Violent: 2, Calls311: 3, Sanitation: 1, Lighting: 1, Transport: 1}
		want.CrimeByMonth[0], want.CrimeByMonth[1], want.CrimeByMonth[2] = 1, 1, 1
		want.CallsByMonth[0], want.CallsByMonth[1] = 2, 1
		if *s != want {
			t.Errorf("%s: stats = %+v, want %+v", name, *s, want)
		}
		if d.Excluded != 1 {
			t.Errorf("%s: Excluded = %d, want 1", name, d.Excluded)
		}
		if d.CrimeByHour[17] != 2 || d.LightingHour[21] != 1 || d.TransportHour[8] != 1 {
			t.Errorf("%s: hour counts wrong", name)
		}
	}
}

// With enough data to be split across many goroutines, the parallel join
// must produce exactly the same result as the sequential one.
func TestParallelEqualsSequentialLargeInput(t *testing.T) {
	hoods := []string{"A", "B", "C", "D", "E", "F", "G"}
	descs := []string{"ROBBERY", "LARCENY", "AGG. ASSAULT", "BURGLARY"}
	types := []string{"SW-Dirty Alley", "ECC-Information Request", "BGE-StLight(s) Out", "TRT-Traffic Signal Repairs", "WW-Water Leak (Exterior)"}

	var crimes []model.CrimeRecord
	var reqs []model.ServiceRequest
	for i := 0; i < 5000; i++ {
		when := time.Date(2024, time.Month(i%12+1), 1+i%28, i%24, 0, 0, 0, time.UTC)
		crimes = append(crimes, model.CrimeRecord{When: when, Description: descs[i%len(descs)], Neighborhood: hoods[i%len(hoods)]})
		reqs = append(reqs, model.ServiceRequest{Created: when, Type: types[i%len(types)], Neighborhood: hoods[(i*3)%len(hoods)]})
	}
	if !reflect.DeepEqual(Build(crimes, reqs), BuildParallel(crimes, reqs)) {
		t.Fatal("parallel join differs from sequential join")
	}
}

func TestBuildParallelEmpty(t *testing.T) {
	d := BuildParallel(nil, nil)
	if len(d.ByHood) != 0 {
		t.Error("expected empty dataset")
	}
}
