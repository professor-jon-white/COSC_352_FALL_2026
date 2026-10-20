package seed

import (
"encoding/csv"
"fmt"
"math/rand"
"os"
"path/filepath"
"time"
)

type hoodSpec struct {
name            string
policeDistrict  string
councilDistrict int
baseLat         float64
baseLon         float64
vacancyWeight   int
}

var baltimoreHoods = []hoodSpec{
{"SANDTOWN-WINCHESTER", "WESTERN", 7, 39.3035, -76.6432, 85},
{"PENN NORTH", "WESTERN", 7, 39.3102, -76.6415, 65},
{"HARLEM PARK", "WESTERN", 9, 39.2954, -76.6428, 75},
{"BROADWAY EAST", "EASTERN", 13, 39.3021, -76.5840, 80},
{"OLIVER", "EASTERN", 12, 39.3075, -76.6061, 60},
{"JOHNSTON SQUARE", "EASTERN", 12, 39.3032, -76.6095, 55},
{"BROOKLYN", "SOUTHERN", 10, 39.2365, -76.6025, 35},
{"CHERRY HILL", "SOUTHERN", 10, 39.2486, -76.6258, 30},
{"WAVERLY", "NORTHERN", 14, 39.3308, -76.6085, 14},
{"RESERVOIR HILL", "CENTRAL", 7, 39.3145, -76.6312, 22},
{"CHARLES VILLAGE", "NORTHERN", 12, 39.3245, -76.6168, 9},
{"HAMPDEN", "NORTHERN", 14, 39.3332, -76.6321, 4},
{"MOUNT VERNON", "CENTRAL", 11, 39.2975, -76.6155, 6},
{"FEDERAL HILL", "SOUTHERN", 11, 39.2778, -76.6115, 2},
{"FELLS POINT", "SOUTHEASTERN", 1, 39.2835, -76.5932, 3},
{"CANTON", "SOUTHEASTERN", 1, 39.2805, -76.5742, 2},
}

func EnsureDatasets(path311, pathVacant string, forceSeed bool) error {
need311 := forceSeed || !fileExists(path311)
needVacant := forceSeed || !fileExists(pathVacant)

if !need311 && !needVacant {
return nil
}

rng := rand.New(rand.NewSource(20260415))

if needVacant {
if err := generateVacantCSV(pathVacant, rng, 1800); err != nil {
return err
}
}
if need311 {
if err := generate311CSV(path311, rng, 12000); err != nil {
return err
}
}
return nil
}

func fileExists(path string) bool {
info, err := os.Stat(path)
return err == nil && !info.IsDir() && info.Size() > 0
}

func generateVacantCSV(path string, rng *rand.Rand, count int) error {
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
return err
}
f, err := os.Create(path)
if err != nil {
return err
}
defer f.Close()

w := csv.NewWriter(f)
defer w.Flush()

headers := []string{
"ReferenceID", "Block", "Lot", "BuildingAddress",
"NoticeDate", "Neighborhood", "PoliceDistrict",
"CouncilDistrict", "Latitude", "Longitude",
}
if err := w.Write(headers); err != nil {
return err
}

streets := []string{"N FULTON AVE", "W NORTH AVE", "E PRESTON ST", "N CAREY ST", "E BIDDLE ST", "S MONROE ST", "GREENMOUNT AVE"}
totalWeight := 0
for _, h := range baltimoreHoods {
totalWeight += h.vacancyWeight
}

baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

for i := 1; i <= count; i++ {
pick := rng.Intn(totalWeight)
var chosen hoodSpec
acc := 0
for _, h := range baltimoreHoods {
acc += h.vacancyWeight
if pick < acc {
chosen = h
break
}
}

lat := chosen.baseLat + (rng.Float64()-0.5)*0.008
lon := chosen.baseLon + (rng.Float64()-0.5)*0.008
noticeDate := baseDate.Add(time.Duration(rng.Intn(750*24)) * time.Hour)
addr := fmt.Sprintf("%d %s", 100+rng.Intn(2400), streets[rng.Intn(len(streets))])

row := []string{
fmt.Sprintf("VBN-%06d", i),
fmt.Sprintf("%04d", 1000+rng.Intn(4000)),
fmt.Sprintf("%03d", 1+rng.Intn(90)),
addr,
noticeDate.Format("2006-01-02"),
chosen.name,
chosen.policeDistrict,
fmt.Sprintf("%d", chosen.councilDistrict),
fmt.Sprintf("%.6f", lat),
fmt.Sprintf("%.6f", lon),
}
if err := w.Write(row); err != nil {
return err
}
}
return nil
}

func generate311CSV(path string, rng *rand.Rand, count int) error {
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
return err
}
f, err := os.Create(path)
if err != nil {
return err
}
defer f.Close()

w := csv.NewWriter(f)
defer w.Flush()

headers := []string{
"SRRecordID", "ServiceRequestNum", "SRType", "MethodReceived",
"CreatedDate", "SRStatus", "CloseDate", "Agency",
"Address", "Neighborhood", "PoliceDistrict",
"CouncilDistrict", "Latitude", "Longitude",
}
if err := w.Write(headers); err != nil {
return err
}

srCatalog := []struct {
srType string
agency string
}{
{"SW-Dirty Alley / Illegal Dumping", "DPW-Solid Waste"},
{"SW-HGW Bulk Trash Collection", "DPW-Solid Waste"},
{"TRM-Pothole Repair", "DOT-Transit & Marine"},
{"BGE-Street Light Outage", "BGE-Utilities"},
{"HCD-Sanitation Property", "HCD-Code Enforcement"},
{"WW-Water Leak Exterior", "DPW-Water & Wastewater"},
{"TRM-Sidewalk / Footway Repair", "DOT-Transit & Marine"},
{"HCD-Illegal Sign / Structure", "HCD-Code Enforcement"},
}

streets := []string{"N FULTON AVE", "W NORTH AVE", "E PRESTON ST", "N CHARLES ST", "LIGHT ST", "EASTERN AVE", "HARFORD RD"}
baseDate := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)

for i := 1; i <= count; i++ {
hood := baltimoreHoods[rng.Intn(len(baltimoreHoods))]
highVac := hood.vacancyWeight >= 30

var itemIdx int
if highVac && rng.Float64() < 0.45 {
if rng.Float64() < 0.7 {
itemIdx = 0
} else {
itemIdx = 4
}
} else {
itemIdx = rng.Intn(len(srCatalog))
}
sr := srCatalog[itemIdx]

method := "Mobile App"
rMethod := rng.Float64()
if highVac {
switch {
case rMethod < 0.54:
method = "Phone Call"
case rMethod < 0.82:
method = "Mobile App"
default:
method = "Web Portal"
}
} else {
switch {
case rMethod < 0.62:
method = "Mobile App"
case rMethod < 0.80:
method = "Phone Call"
default:
method = "Web Portal"
}
}

created := baseDate.Add(time.Duration(rng.Intn(400*24)) * time.Hour)
closeProb := 0.88
if highVac {
closeProb = 0.74
}

status := "Open"
closeDateStr := ""
if rng.Float64() < closeProb {
status = "Closed"
baseHours := 18.0 + rng.Float64()*48.0
if highVac {
baseHours *= 1.65
}
closed := created.Add(time.Duration(baseHours * float64(time.Hour)))
closeDateStr = closed.Format("2006-01-02 15:04:05")
}

lat := hood.baseLat + (rng.Float64()-0.5)*0.008
lon := hood.baseLon + (rng.Float64()-0.5)*0.008
addr := fmt.Sprintf("%d %s", 100+rng.Intn(2400), streets[rng.Intn(len(streets))])

row := []string{
fmt.Sprintf("SR-%07d", i),
fmt.Sprintf("25-%08d", i),
sr.srType,
method,
created.Format("2006-01-02 15:04:05"),
status,
closeDateStr,
sr.agency,
addr,
hood.name,
hood.policeDistrict,
fmt.Sprintf("%d", hood.councilDistrict),
fmt.Sprintf("%.6f", lat),
fmt.Sprintf("%.6f", lon),
}
if err := w.Write(row); err != nil {
return err
}
}
return nil
}
