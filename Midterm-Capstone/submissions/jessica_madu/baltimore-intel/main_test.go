package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, body string) string {
	p := filepath.Join(t.TempDir(), "x.csv")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIngestCrime(t *testing.T) {
	p := write(t, "\uFEFFRowID,CrimeDateTime,Neighborhood,Description\n1,2024/03/05 14:30:00+00,Canton,HOMICIDE\n2,2025/03/06 02:00:00+00,Canton,LARCENY\n3,2025/03/06 02:00:00+00,,LARCENY\n")
	a, err := Ingest(p, NewCrimeDataset(), 4, YearRange{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Total != 2 || a.Skipped != 1 || a.ByNbhd["CANTON"] != 2 || a.ViolentByNbhd["CANTON"] != 1 || a.ByHour[14] != 1 {
		t.Fatalf("unexpected aggregate: %+v", a)
	}
}

func Test311Open(t *testing.T) {
	p := write(t, "SRType,CreatedDate,SRStatus,Neighborhood\nPothole,3/5/2025 2:30:00 PM,Open,Canton\nPothole,3/5/2025 2:30:00 PM,Closed,Canton\n")
	a, err := Ingest(p, New311Dataset(), 2, YearRange{})
	if err != nil {
		t.Fatal(err)
	}
	if a.OpenByNbhd["CANTON"] != 1 || a.ByHour[14] != 2 || a.ByMonth["2025-03"] != 2 {
		t.Fatalf("unexpected aggregate: %+v", a)
	}
}

func TestMissingColumns(t *testing.T) {
	p := write(t, "a,b\n1,2\n")
	if _, err := Ingest(p, NewCrimeDataset(), 1, YearRange{}); err == nil {
		t.Fatal("expected error")
	}
}
