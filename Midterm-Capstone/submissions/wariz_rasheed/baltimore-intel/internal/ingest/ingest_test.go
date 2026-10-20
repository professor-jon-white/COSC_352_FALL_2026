package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"baltimoreintel/internal/model"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadCrime(t *testing.T) {
	text := "\ufeffCrimeDateTime,Description,Neighborhood,Location,Shooting\r\n" +
		"10/11/2026 5:00:00 AM,AGG. ASSAULT,Frankford,5300 SINCLAIR LN,Y\r\n" +
		"10/11/2026 4:00:00 AM,LARCENY,DOWNTOWN,\"800 N CHARLES ST\",\r\n" +
		"10/11/2026 8:00:00 PM,ROBBERY - CARJACKING,,200 S BROADWAY,\r\n" +
		"not a date,BURGLARY,CANTON,100 S HIGHLAND AVE,\r\n"
	rows, st, err := Load[model.Crime](writeTemp(t, text), 2, CrimeParser)
	if err != nil {
		t.Fatal(err)
	}
	if st.Read != 4 || st.Kept != 3 || st.Rejected != 1 {
		t.Fatalf("stats = %+v, want read 4, kept 3, rejected 1", st)
	}
	byOffense := map[string]model.Crime{}
	for _, c := range rows {
		byOffense[c.Offense] = c
	}

	a := byOffense["AGG. ASSAULT"]
	if !a.Violent || !a.Shooting || a.Hood != "FRANKFORD" || a.Block != "5300 SINCLAIR LN" {
		t.Errorf("aggravated assault parsed wrongly: %+v", a)
	}
	if a.LocalHour != 1 { // 05:00 UTC on 11 Oct is 01:00 EDT
		t.Errorf("local hour = %d, want 1", a.LocalHour)
	}
	if !model.IsNight(a.LocalHour) {
		t.Error("01:00 local should count as night")
	}

	l := byOffense["LARCENY"]
	if l.Violent || l.LocalHour != -1 { // 04:00 UTC is exactly local midnight: unknown time
		t.Errorf("larceny parsed wrongly: %+v", l)
	}

	r := byOffense["ROBBERY - CARJACKING"]
	if !r.Violent || r.Hood != "" || r.LocalHour != 16 || r.Block != "200 S BROADWAY" {
		t.Errorf("carjacking parsed wrongly: %+v", r)
	}
}

func TestLoadMissingColumn(t *testing.T) {
	_, _, err := Load[model.Crime](writeTemp(t, "Foo,Bar\r\n1,2\r\n"), 1, CrimeParser)
	if err == nil || !strings.Contains(err.Error(), "CrimeDateTime") {
		t.Fatalf("expected a missing-column error naming CrimeDateTime, got %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, _, err := Load[model.Crime]("/no/such/file.csv", 1, CrimeParser); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadManyBatches(t *testing.T) {
	var b strings.Builder
	b.WriteString("CrimeDateTime,Description,Neighborhood,Location,Shooting\n")
	const n = 10000 // several batches of 2048 spread over 4 workers
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "1/2/2026 3:00:00 PM,VANDALISM,CANTON,%d S HIGHLAND AVE,\n", 100+i%900)
	}
	rows, st, err := Load[model.Crime](writeTemp(t, b.String()), 4, CrimeParser)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != n || st.Read != n || st.Kept != n {
		t.Fatalf("got %d rows, stats %+v, want %d", len(rows), st, n)
	}
}

func TestRequestParser(t *testing.T) {
	text := "CreatedDate,CloseDate,SRType,SRStatus,Address,Neighborhood\n" +
		"2026/01/03 14:29:00+00,2026/01/03 16:29:00+00,SW-Dirty Alley,Closed,\"3954 DOLFIELD AVE, Baltimore City, 21215\",Patterson Park Neighborhood\n" +
		"2026/01/04 10:00:00+00,,TRS-Parking Complaint,Open,,\n"
	rows, st, err := Load[model.Request](writeTemp(t, text), 1, RequestParser)
	if err != nil {
		t.Fatal(err)
	}
	if st.Kept != 2 {
		t.Fatalf("kept %d rows, want 2", st.Kept)
	}
	byType := map[string]model.Request{}
	for _, r := range rows {
		byType[r.Type] = r
	}
	a := byType["SW-Dirty Alley"]
	if !a.Completed || a.Prefix != "SW" || a.Hood != "PATTERSON PARK" || a.Block != "3900 DOLFIELD AVE" {
		t.Errorf("closed request parsed wrongly: %+v", a)
	}
	if h := a.Closed.Sub(a.Created).Hours(); h != 2 {
		t.Errorf("hours to close = %v, want 2", h)
	}
	o := byType["TRS-Parking Complaint"]
	if o.Completed || !o.Closed.IsZero() || o.Block != "" {
		t.Errorf("open request parsed wrongly: %+v", o)
	}
}

func TestLicenseParser(t *testing.T) {
	text := "LLKey,LicenseDate,LicenseEndDate,LicenseStatus,EstablishmentDesc,AddrStreet\n" +
		"100005,2025/05/01 04:00:00+00,2026/04/30 04:00:00+00,Renewed,Tavern License,1724-26 CHARLES STREET NORTH\n" +
		"100006,2025/11/20 16:43:32.300+00,2026/04/30 04:00:00+00,Closed,Restaurant,320 EUTAW STREET NORTH\n" +
		",2025/05/01 04:00:00+00,2026/04/30 04:00:00+00,Renewed,Tavern,1 MAIN ST\n"
	rows, st, err := Load[model.License](writeTemp(t, text), 1, LicenseParser)
	if err != nil {
		t.Fatal(err)
	}
	if st.Kept != 2 || st.Rejected != 1 {
		t.Fatalf("stats = %+v, want kept 2, rejected 1 (missing key)", st)
	}
	byKey := map[string]model.License{}
	for _, l := range rows {
		byKey[l.Key] = l
	}
	a := byKey["100005"]
	if a.Kind != "TAVERN" || !a.Active || a.Block != "1700 N CHARLES ST" {
		t.Errorf("tavern parsed wrongly: %+v", a)
	}
	b := byKey["100006"] // fractional seconds must parse; status Closed means inactive
	if b.Active || b.Kind != "RESTAURANT" || b.Start.IsZero() {
		t.Errorf("closed restaurant parsed wrongly: %+v", b)
	}
}
