package ingest

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"baltimore-intel/internal/model"
)

func TestParseTimeFormats(t *testing.T) {
	want := time.Date(2022, 3, 14, 23, 15, 0, 0, time.UTC).Unix()
	for _, s := range []string{
		"2022/03/14 23:15:00+00", // portal CSV export
		"2022-03-14T23:15:00Z",   // ISO
		"1647299700000",          // ArcGIS REST epoch ms
		"2022-03-14 23:15:00",
	} {
		got, ok := ParseTime(s)
		if !ok || got != want {
			t.Errorf("ParseTime(%q) = %d,%v want %d", s, got, ok, want)
		}
	}
	if _, ok := ParseTime("not a date"); ok {
		t.Error("garbage parsed as a date")
	}
	if YearOf(time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC).Unix()) != 2022 {
		t.Error("YearOf must use Baltimore local time (03:00 UTC Jan 1 is still Dec 31 in Baltimore)")
	}
}

func TestHeaderAliasesAndBOM(t *testing.T) {
	h := NewHeader([]string{"\ufeffX", "Y", "Neighborhood"})
	if h.Col("Longitude", "X") != 0 || h.Col("latitude", "y") != 1 || h.Col("NEIGHBORHOOD") != 2 {
		t.Fatal("alias / case-insensitive / BOM lookup failed")
	}
	if Get([]string{"a"}, 5) != "" || Get([]string{" a "}, 0) != "a" {
		t.Fatal("Get must be bounds-safe and trim")
	}
}

func TestClassifiers(t *testing.T) {
	for _, d := range []string{"AGG. ASSAULT", "ROBBERY - CARJACKING", "HOMICIDE", "SHOOTING", "COMMON ASSAULT"} {
		if !IsViolent(d) {
			t.Errorf("%s should be violent", d)
		}
	}
	for _, d := range []string{"LARCENY", "BURGLARY", "AUTO THEFT"} {
		if IsViolent(d) {
			t.Errorf("%s should not be violent", d)
		}
	}
	if !IsSanitation("SW-Rat Rubout") || !IsSanitation("HCD-Illegal Dumping") || IsSanitation("TRM-Pothole") {
		t.Error("sanitation classifier wrong")
	}
}

const crimeCSV = `RowID,CCNumber,CrimeDateTime,CrimeCode,Description,Inside_Outside,Weapon,Post,Gender,Age,Race,Ethnicity,Location,Old_District,New_District,Neighborhood,Latitude,Longitude,GeoLocation,PremiseType,Total_Incidents
1,22A1,2022/03/14 23:15:00+00,4C,AGG. ASSAULT,O,,111,,,,,1 MAIN ST,CENTRAL,CENTRAL,Downtown,39.29,-76.61,"(39.29, -76.61)",STREET,1
2,22A2,2022/03/15 10:00:00+00,6E,LARCENY,I,,111,,,,,2 MAIN ST,CENTRAL,CENTRAL,DOWNTOWN,,,"(39.291, -76.611)",STREET,2
3,22A3,bad date,6E,LARCENY,I,,111,,,,,2 MAIN ST,CENTRAL,CENTRAL,Downtown,39.29,-76.61,,STREET,1
4,21A4,2021/06/01 12:00:00+00,6E,LARCENY,I,,111,,,,,2 MAIN ST,CENTRAL,CENTRAL,Canton,0,0,,STREET,1
5,22A5,2022/07/04 12:00:00+00,3B,ROBBERY - STREET,O,,111,,,,,"5 ""QUOTED"" ST",CENTRAL,CENTRAL,Canton,0,0,,STREET,1
`

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCrimeSource(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "part1_crime.csv", crimeCSV)
	ds := model.NewDataset()
	ds.Year = 2022
	src, _ := NewSource(KindCrime)
	st, err := src.Load(context.Background(), dir, 3, ds)
	if err != nil {
		t.Fatal(err)
	}
	if st.Rows != 5 || st.Kept != 3 || st.Skipped["bad_date"] != 1 || st.Skipped["other_year"] != 1 {
		t.Fatalf("stats = %+v", st)
	}
	sort.Slice(ds.Crimes, func(a, b int) bool { return ds.Crimes[a].Time < ds.Crimes[b].Time })
	c := ds.Crimes[0]
	if !c.Violent || c.Lat == 0 || ds.Hoods.Name(c.Hood) != "Downtown" {
		t.Errorf("first record parsed wrong: %+v", c)
	}
	if ds.Crimes[1].Lat == 0 || ds.Crimes[1].Incidents != 2 {
		t.Errorf("GeoLocation fallback / incidents failed: %+v", ds.Crimes[1])
	}
	if ds.Crimes[0].Hood != ds.Crimes[1].Hood {
		t.Error("'Downtown' and 'DOWNTOWN' must join to one neighborhood")
	}
	if ds.Crimes[2].Lat != 0 {
		t.Error("(0,0) coordinates must be rejected as not in Baltimore")
	}
}

func TestMissingColumnIsReported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "part1_crime.csv", "A,B\n1,2\n")
	src, _ := NewSource(KindCrime)
	_, err := src.Load(context.Background(), dir, 2, model.NewDataset())
	if err == nil || !strings.Contains(err.Error(), "CrimeDateTime") {
		t.Fatalf("want missing-column error, got %v", err)
	}
}

func TestLiquorKeepsLatestLicenseYear(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "liquor_licenses.csv", `X,Y,LicenseNumber,LicenseYear,TradeName,LicenseStatus,EstablishmentDesc,AddrStreet
-76.61,39.29,L1,2021,Old Name,Approved,Tavern,1 Main
-76.61,39.29,L1,2022,New Name,Approved,Tavern,1 Main
-76.60,39.30,L2,2022,Gone Bar,Cancelled,Tavern,2 Main
,,L3,2022,No Location,Approved,Tavern,3 Main
`)
	ds := model.NewDataset()
	src, _ := NewSource(KindLiquor)
	st, err := src.Load(context.Background(), dir, 2, ds)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Outlets) != 1 || ds.Outlets[0].Name != "New Name" {
		t.Fatalf("outlets = %+v", ds.Outlets)
	}
	if st.Skipped["inactive"] != 1 || st.Skipped["no_location"] != 1 || st.Skipped["duplicate_year"] != 1 {
		t.Fatalf("skips = %v", st.Skipped)
	}
}

// The concurrent parser must produce exactly the same records, in file order,
// whatever the worker count.
func TestParserDeterministicAcrossWorkerCounts(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("CreatedDate,SRType,Neighborhood,Latitude,Longitude,Agency,CloseDate\n")
	for i := 0; i < 9000; i++ {
		b.WriteString(time.Date(2022, 1, 1, 0, 0, i, 0, time.UTC).Format("2006/01/02 15:04:05+00"))
		b.WriteString(",SW-Rat Rubout,Canton,39.28,-76.57,Solid Waste,\n")
	}
	writeFile(t, dir, "service_requests_311.csv", b.String())
	var first []int64
	for _, w := range []int{1, 2, 7, 16} {
		ds := model.NewDataset()
		src, _ := NewSource(Kind311)
		if _, err := src.Load(context.Background(), dir, w, ds); err != nil {
			t.Fatal(err)
		}
		got := make([]int64, len(ds.Requests))
		for i, r := range ds.Requests {
			got[i] = r.Created
		}
		if !sort.SliceIsSorted(got, func(a, b int) bool { return got[a] < got[b] }) {
			t.Fatalf("workers=%d: records not in file order", w)
		}
		if first == nil {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Fatalf("workers=%d: %d records, want %d", w, len(got), len(first))
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("workers=%d: record %d differs", w, i)
			}
		}
	}
}
