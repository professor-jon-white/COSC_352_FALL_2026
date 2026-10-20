package engine

import (
"math"
"strings"
"sync"

"bmore-engine/internal/model"
)

const earthRadiusMeters = 6371000.0

type SpatialJoinStrategy interface {
Match(requests []model.ServiceRequest, vacants []model.VacantBuilding, workers int) []model.SpatialMatch
}

type HaversineGridJoin struct {
MaxRadiusMeters float64
}

type gridKey struct {
latBin int
lonBin int
}

const gridResolution = 500.0

func toGridKey(lat, lon float64) gridKey {
return gridKey{
latBin: int(math.Floor(lat * gridResolution)),
lonBin: int(math.Floor(lon * gridResolution)),
}
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
dLat := (lat2 - lat1) * (math.Pi / 180.0)
dLon := (lon2 - lon1) * (math.Pi / 180.0)
rLat1 := lat1 * (math.Pi / 180.0)
rLat2 := lat2 * (math.Pi / 180.0)

a := math.Sin(dLat/2)*math.Sin(dLat/2) +
math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
return earthRadiusMeters * c
}

func isInfrastructureHazard(srType string) bool {
u := strings.ToUpper(srType)
return strings.Contains(u, "POTHOLE") ||
strings.Contains(u, "STREET LIGHT") ||
strings.Contains(u, "STREETLIGHT") ||
strings.Contains(u, "FOOTWAY") ||
strings.Contains(u, "SIDEWALK") ||
strings.Contains(u, "SIGN") ||
strings.Contains(u, "WATER LEAK")
}

func (h HaversineGridJoin) Match(
requests []model.ServiceRequest,
vacants []model.VacantBuilding,
workers int,
) []model.SpatialMatch {
spatialIndex := make(map[gridKey][]model.VacantBuilding, len(vacants)/2)
for i := range vacants {
v := vacants[i]
if !v.HasCoords {
continue
}
k := toGridKey(v.Latitude, v.Longitude)
spatialIndex[k] = append(spatialIndex[k], v)
}

hazards := Filter(requests, func(r model.ServiceRequest) bool {
return r.HasCoords && isInfrastructureHazard(r.SRType)
})

if len(hazards) == 0 {
return nil
}

chunkSize := (len(hazards) + workers - 1) / workers
outCh := make(chan []model.SpatialMatch, workers)
var wg sync.WaitGroup

for w := 0; w < workers; w++ {
start := w * chunkSize
if start >= len(hazards) {
break
}
end := start + chunkSize
if end > len(hazards) {
end = len(hazards)
}

wg.Add(1)
go func(slice []model.ServiceRequest) {
defer wg.Done()
localMatches := make([]model.SpatialMatch, 0, 128)

for i := range slice {
req := slice[i]
center := toGridKey(req.Latitude, req.Longitude)
bestDist := h.MaxRadiusMeters + 1.0
var bestVacant model.VacantBuilding
found := false

for dLat := -1; dLat <= 1; dLat++ {
for dLon := -1; dLon <= 1; dLon++ {
cell := gridKey{latBin: center.latBin + dLat, lonBin: center.lonBin + dLon}
candidates, ok := spatialIndex[cell]
if !ok {
continue
}
for j := range candidates {
cand := candidates[j]
dist := haversineMeters(req.Latitude, req.Longitude, cand.Latitude, cand.Longitude)
if dist <= h.MaxRadiusMeters && dist < bestDist {
bestDist = dist
bestVacant = cand
found = true
}
}
}
}

if found {
localMatches = append(localMatches, model.SpatialMatch{
RequestID:      req.RecordID,
SRType:         req.SRType,
RequestAddr:    req.Address,
VacantNoticeID: bestVacant.ReferenceID,
VacantAddr:     bestVacant.Address,
Neighborhood:   req.Neighborhood,
DistanceMeters: bestDist,
})
}
}
outCh <- localMatches
}(hazards[start:end])
}

wg.Wait()
close(outCh)

var allMatches []model.SpatialMatch
for batch := range outCh {
allMatches = append(allMatches, batch...)
}
return allMatches
}
