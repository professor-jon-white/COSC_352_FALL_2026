package main

import (
	"fmt"
	"path/filepath"
	"time"
)

func demo(dir string) error {
	a, b := Page{}, Page{}
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("DEMO-%03d", i)
		n := []string{"DEMO NORTH", "DEMO SOUTH", "DEMO EAST"}[i%3]
		date := start.Add(time.Duration(i) * 24 * time.Hour).UnixMilli()
		a.Features = append(a.Features, Feature{Row{"RowID": i + 1, "CCNumber": id, "CrimeDateTime": date, "Neighborhood": n, "Description": []string{"ROBBERY", "BURGLARY", "ASSAULT"}[i%3], "Weapon": []string{"HANDGUN", "NONE", "UNKNOWN"}[i%3]}})
		if i%3 != 0 {
			b.Features = append(b.Features, Feature{Row{"RowID": i + 1, "IncidentNumber": id, "ArrestNumber": fmt.Sprintf("ARREST-%03d", i), "ArrestDateTime": date + int64(i%20)*3600000, "Neighborhood": n}})
		}
	}
	if e := save(filepath.Join(dir, "crime.json"), a); e != nil {
		return e
	}
	if e := save(filepath.Join(dir, "arrests.json"), b); e != nil {
		return e
	}
	return save(filepath.Join(dir, "manifest.json"), Manifest{Retrieved: time.Now().UTC().Format(time.RFC3339), Start: "2025-01-01", End: "2025-04-01", Demo: true, Crime: Source{URL: "synthetic", Rows: len(a.Features), Available: len(a.Features)}, Arrest: Source{URL: "synthetic", Rows: len(b.Features), Available: len(b.Features)}})
}
