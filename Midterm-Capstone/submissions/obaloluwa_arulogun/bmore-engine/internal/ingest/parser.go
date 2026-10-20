package ingest

import (
"bufio"
"encoding/csv"
"fmt"
"io"
"os"
"strconv"
"strings"
"sync"
"time"

"bmore-engine/internal/model"
)

var rowPool = sync.Pool{
New: func() any {
s := make([]string, 0, 24)
return &s
},
}

var timeLayouts = []string{
time.RFC3339,
"2006-01-02 15:04:05",
"2006/01/02 15:04:05+00",
"01/02/2006 03:04:05 PM",
"01/02/2006 15:04:05",
"2006-01-02",
"01/02/2006",
}

func parseFlexibleTime(raw string) (time.Time, bool) {
raw = strings.TrimSpace(raw)
if raw == "" {
return time.Time{}, false
}
for _, layout := range timeLayouts {
if t, err := time.Parse(layout, raw); err == nil {
return t, true
}
}
return time.Time{}, false
}

func buildHeaderIndex(headers []string) map[string]int {
idx := make(map[string]int, len(headers))
for i, h := range headers {
clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
idx[clean] = i
}
return idx
}

func getCol(row []string, idx map[string]int, candidates ...string) string {
for _, c := range candidates {
if pos, ok := idx[c]; ok && pos < len(row) {
return strings.TrimSpace(row[pos])
}
}
return ""
}

func Load311Concurrent(path string, workers int) ([]model.ServiceRequest, error) {
f, err := os.Open(path)
if err != nil {
return nil, fmt.Errorf("open 311 dataset: %w", err)
}
defer f.Close()

reader := csv.NewReader(bufio.NewReaderSize(f, 64*1024))
reader.ReuseRecord = true
reader.FieldsPerRecord = -1
reader.LazyQuotes = true

headers, err := reader.Read()
if err != nil {
return nil, fmt.Errorf("read 311 header: %w", err)
}
colIdx := buildHeaderIndex(headers)

rawCh := make(chan *[]string, workers*256)
outCh := make(chan []model.ServiceRequest, workers)

var wg sync.WaitGroup
for w := 0; w < workers; w++ {
wg.Add(1)
go func() {
defer wg.Done()
localBatch := make([]model.ServiceRequest, 0, 1024)

for rowPtr := range rawCh {
row := *rowPtr
if req, ok := parse311Row(row, colIdx); ok {
localBatch = append(localBatch, req)
}
*rowPtr = row[:0]
rowPool.Put(rowPtr)
}
outCh <- localBatch
}()
}

for {
rec, err := reader.Read()
if err == io.EOF {
break
}
if err != nil {
continue
}
bufPtr := rowPool.Get().(*[]string)
*bufPtr = append((*bufPtr)[:0], rec...)
rawCh <- bufPtr
}
close(rawCh)

wg.Wait()
close(outCh)

var results []model.ServiceRequest
for batch := range outCh {
results = append(results, batch...)
}
return results, nil
}

func parse311Row(row []string, idx map[string]int) (model.ServiceRequest, bool) {
neighborhood := model.NormalizeKey(getCol(row, idx, "neighborhood", "neighborhood_name"))
if neighborhood == "" {
return model.ServiceRequest{}, false
}

id := getCol(row, idx, "srrecordid", "servicerequestnum", "record_id")
srType := getCol(row, idx, "srtype", "service_type", "request_type")
method := model.NormalizeKey(getCol(row, idx, "methodreceived", "method_received", "source"))
status := model.NormalizeKey(getCol(row, idx, "srstatus", "status"))
agency := model.NormalizeKey(getCol(row, idx, "agency", "agency_name"))
address := getCol(row, idx, "address", "street_address")
policeDist := model.NormalizeKey(getCol(row, idx, "policedistrict", "police_district"))

councilStr := getCol(row, idx, "councildistrict", "council_district")
councilDist, _ := strconv.Atoi(councilStr)

created, hasCreated := parseFlexibleTime(getCol(row, idx, "createddate", "created_date"))
closed, hasClosed := parseFlexibleTime(getCol(row, idx, "closedate", "close_date"))

var resHours float64
isClosed := strings.Contains(status, "CLOSE")
if hasCreated && hasClosed && closed.After(created) {
resHours = closed.Sub(created).Hours()
isClosed = true
}

lat, errLat := strconv.ParseFloat(getCol(row, idx, "latitude", "lat"), 64)
lon, errLon := strconv.ParseFloat(getCol(row, idx, "longitude", "lon", "long"), 64)
hasCoords := errLat == nil && errLon == nil && lat != 0 && lon != 0

return model.ServiceRequest{
RecordID:        id,
SRType:          srType,
MethodReceived:  method,
Status:          status,
Agency:          agency,
Address:         address,
Neighborhood:    neighborhood,
PoliceDistrict:  policeDist,
CouncilDistrict: councilDist,
CreatedDate:     created,
CloseDate:       closed,
ResolutionHours: resHours,
Latitude:        lat,
Longitude:       lon,
IsClosed:        isClosed,
HasCoords:       hasCoords,
}, true
}

func LoadVacantConcurrent(path string, workers int) ([]model.VacantBuilding, error) {
f, err := os.Open(path)
if err != nil {
return nil, fmt.Errorf("open vacant dataset: %w", err)
}
defer f.Close()

reader := csv.NewReader(bufio.NewReaderSize(f, 64*1024))
reader.ReuseRecord = true
reader.FieldsPerRecord = -1
reader.LazyQuotes = true

headers, err := reader.Read()
if err != nil {
return nil, fmt.Errorf("read vacant header: %w", err)
}
colIdx := buildHeaderIndex(headers)

rawCh := make(chan *[]string, workers*256)
outCh := make(chan []model.VacantBuilding, workers)

var wg sync.WaitGroup
for w := 0; w < workers; w++ {
wg.Add(1)
go func() {
defer wg.Done()
localBatch := make([]model.VacantBuilding, 0, 512)

for rowPtr := range rawCh {
row := *rowPtr
if vb, ok := parseVacantRow(row, colIdx); ok {
localBatch = append(localBatch, vb)
}
*rowPtr = row[:0]
rowPool.Put(rowPtr)
}
outCh <- localBatch
}()
}

for {
rec, err := reader.Read()
if err == io.EOF {
break
}
if err != nil {
continue
}
bufPtr := rowPool.Get().(*[]string)
*bufPtr = append((*bufPtr)[:0], rec...)
rawCh <- bufPtr
}
close(rawCh)

wg.Wait()
close(outCh)

var results []model.VacantBuilding
for batch := range outCh {
results = append(results, batch...)
}
return results, nil
}

func parseVacantRow(row []string, idx map[string]int) (model.VacantBuilding, bool) {
neighborhood := model.NormalizeKey(getCol(row, idx, "neighborhood", "neighborhood_name"))
if neighborhood == "" {
return model.VacantBuilding{}, false
}

refID := getCol(row, idx, "referenceid", "notice_num", "objectid")
block := getCol(row, idx, "block")
lot := getCol(row, idx, "lot")
address := getCol(row, idx, "buildingaddress", "address", "property_address")
policeDist := model.NormalizeKey(getCol(row, idx, "policedistrict", "police_district"))

councilStr := getCol(row, idx, "councildistrict", "council_district")
councilDist, _ := strconv.Atoi(councilStr)

noticeDate, _ := parseFlexibleTime(getCol(row, idx, "noticedate", "date_notice"))

lat, errLat := strconv.ParseFloat(getCol(row, idx, "latitude", "lat"), 64)
lon, errLon := strconv.ParseFloat(getCol(row, idx, "longitude", "lon", "long"), 64)
hasCoords := errLat == nil && errLon == nil && lat != 0 && lon != 0

return model.VacantBuilding{
ReferenceID:     refID,
Block:           block,
Lot:             lot,
Address:         address,
Neighborhood:    neighborhood,
PoliceDistrict:  policeDist,
CouncilDistrict: councilDist,
NoticeDate:      noticeDate,
Latitude:        lat,
Longitude:       lon,
HasCoords:       hasCoords,
}, true
}
