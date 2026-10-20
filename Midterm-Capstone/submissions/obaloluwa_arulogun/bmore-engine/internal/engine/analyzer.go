package engine

import (
"runtime"
"sort"
"strings"
"sync"
"time"

"bmore-engine/internal/model"
)

type Engine struct {
workers              int
highVacancyThreshold int
spatialStrategy      SpatialJoinStrategy
}

type EngineBuilder struct {
workers              int
highVacancyThreshold int
spatialStrategy      SpatialJoinStrategy
}

func NewEngineBuilder() *EngineBuilder {
return &EngineBuilder{
workers:              runtime.NumCPU(),
highVacancyThreshold: 25,
spatialStrategy:      HaversineGridJoin{MaxRadiusMeters: 150.0},
}
}

func (b *EngineBuilder) WithWorkers(w int) *EngineBuilder {
if w > 0 {
b.workers = w
}
return b
}

func (b *EngineBuilder) WithHighVacancyThreshold(t int) *EngineBuilder {
if t > 0 {
b.highVacancyThreshold = t
}
return b
}

func (b *EngineBuilder) WithSpatialStrategy(s SpatialJoinStrategy) *EngineBuilder {
if s != nil {
b.spatialStrategy = s
}
return b
}

func (b *EngineBuilder) Build() *Engine {
return &Engine{
workers:              b.workers,
highVacancyThreshold: b.highVacancyThreshold,
spatialStrategy:      b.spatialStrategy,
}
}

func (e *Engine) Execute(
requests []model.ServiceRequest,
vacants []model.VacantBuilding,
ingestDuration time.Duration,
) model.AnalysisReport {
joinStart := time.Now()

var (
spatialMatches []model.SpatialMatch
vacantByHood   map[string][]model.VacantBuilding
reqByHood      map[string][]model.ServiceRequest
joinWG         sync.WaitGroup
)

joinWG.Add(3)
go func() {
defer joinWG.Done()
spatialMatches = e.spatialStrategy.Match(requests, vacants, e.workers)
}()
go func() {
defer joinWG.Done()
vacantByHood = GroupBy(vacants, func(v model.VacantBuilding) string {
return v.Neighborhood
})
}()
go func() {
defer joinWG.Done()
reqByHood = GroupBy(requests, func(r model.ServiceRequest) string {
return r.Neighborhood
})
}()
joinWG.Wait()
joinDuration := time.Since(joinStart)

analysisStart := time.Now()

spatialByHood := make(map[string]int)
for i := range spatialMatches {
spatialByHood[spatialMatches[i].Neighborhood]++
}

allHoods := make(map[string]struct{})
for k := range vacantByHood {
allHoods[k] = struct{}{}
}
for k := range reqByHood {
allHoods[k] = struct{}{}
}

profiles := make([]model.NeighborhoodProfile, 0, len(allHoods))
highVacancySet := make(map[string]bool, len(allHoods))

for hood := range allHoods {
vList := vacantByHood[hood]
rList := reqByHood[hood]

if len(vList) >= e.highVacancyThreshold {
highVacancySet[hood] = true
}

var dumping, openCnt, closedWithTime, appCnt, phoneCnt, webCnt int
var sumResHours float64

for i := range rList {
r := rList[i]
srUpper := strings.ToUpper(r.SRType)
if strings.Contains(srUpper, "DUMP") || strings.Contains(srUpper, "TRASH") ||
strings.Contains(srUpper, "SANITATION") || strings.Contains(srUpper, "DIRTY") {
dumping++
}
if !r.IsClosed {
openCnt++
}
if r.IsClosed && r.ResolutionHours > 0 {
sumResHours += r.ResolutionHours
closedWithTime++
}
switch {
case strings.Contains(r.MethodReceived, "APP") || strings.Contains(r.MethodReceived, "MOBILE"):
appCnt++
case strings.Contains(r.MethodReceived, "PHONE") || strings.Contains(r.MethodReceived, "CALL"):
phoneCnt++
default:
webCnt++
}
}

avgRes := 0.0
if closedWithTime > 0 {
avgRes = sumResHours / float64(closedWithTime)
}

profiles = append(profiles, model.NeighborhoodProfile{
Name:               hood,
VacantCount:        len(vList),
Total311Count:      len(rList),
Dumping311Count:    dumping,
Open311Count:       openCnt,
SpatialHazardCount: spatialByHood[hood],
AvgResolutionHours: avgRes,
AppSubmissions:     appCnt,
PhoneSubmissions:   phoneCnt,
WebSubmissions:     webCnt,
})
}

var (
q1Hotspots []model.NeighborhoodProfile
q2Agencies []model.AgencyMetrics
q3TopMatch []model.SpatialMatch
q4Burdens  []model.DistrictBurden
q5Channels []model.ChannelTierSummary
ansWG      sync.WaitGroup
)

ansWG.Add(5)

go func() {
defer ansWG.Done()
candidates := Filter(profiles, func(p model.NeighborhoodProfile) bool {
return p.VacantCount >= 5 && p.Dumping311Count > 0
})
sort.Slice(candidates, func(i, j int) bool {
ratioI := float64(candidates[i].Dumping311Count) / float64(candidates[i].VacantCount)
ratioJ := float64(candidates[j].Dumping311Count) / float64(candidates[j].VacantCount)
if ratioI == ratioJ {
return candidates[i].Dumping311Count > candidates[j].Dumping311Count
}
return ratioI > ratioJ
})
if len(candidates) > 8 {
candidates = candidates[:8]
}
q1Hotspots = candidates
}()

go func() {
defer ansWG.Done()
type acc struct {
highSum, lowSum float64
highCnt, lowCnt int
}
byAgency := make(map[string]*acc)
for i := range requests {
r := requests[i]
if !r.IsClosed || r.ResolutionHours <= 0 || r.Agency == "" {
continue
}
entry, ok := byAgency[r.Agency]
if !ok {
entry = &acc{}
byAgency[r.Agency] = entry
}
if highVacancySet[r.Neighborhood] {
entry.highSum += r.ResolutionHours
entry.highCnt++
} else {
entry.lowSum += r.ResolutionHours
entry.lowCnt++
}
}

for ag, d := range byAgency {
if d.highCnt+d.lowCnt < 10 {
continue
}
var highAvg, lowAvg float64
if d.highCnt > 0 {
highAvg = d.highSum / float64(d.highCnt)
}
if d.lowCnt > 0 {
lowAvg = d.lowSum / float64(d.lowCnt)
}
q2Agencies = append(q2Agencies, model.AgencyMetrics{
AgencyName:             ag,
HighVacancyAvgHours:    highAvg,
LowVacancyAvgHours:     lowAvg,
HighVacancyTicketCount: d.highCnt,
LowVacancyTicketCount:  d.lowCnt,
})
}
sort.Slice(q2Agencies, func(i, j int) bool {
return q2Agencies[i].HighVacancyAvgHours > q2Agencies[j].HighVacancyAvgHours
})
if len(q2Agencies) > 6 {
q2Agencies = q2Agencies[:6]
}
}()

go func() {
defer ansWG.Done()
copied := make([]model.SpatialMatch, len(spatialMatches))
copy(copied, spatialMatches)
sort.Slice(copied, func(i, j int) bool {
return copied[i].DistanceMeters < copied[j].DistanceMeters
})
if len(copied) > 8 {
copied = copied[:8]
}
q3TopMatch = copied
}()

go func() {
defer ansWG.Done()
distMap := make(map[int]*model.DistrictBurden)
for d := 1; d <= 14; d++ {
distMap[d] = &model.DistrictBurden{DistrictID: d}
}
for i := range vacants {
d := vacants[i].CouncilDistrict
if d >= 1 && d <= 14 {
distMap[d].VacantNotices++
}
}
for i := range requests {
d := requests[i].CouncilDistrict
if d >= 1 && d <= 14 {
distMap[d].Total311Volume++
if !requests[i].IsClosed {
distMap[d].Open311Backlog++
}
}
}
for _, b := range distMap {
if b.Total311Volume == 0 && b.VacantNotices == 0 {
continue
}
b.CompositeBurden = (float64(b.VacantNotices) * 0.6) + (float64(b.Open311Backlog) * 0.4)
q4Burdens = append(q4Burdens, *b)
}
sort.Slice(q4Burdens, func(i, j int) bool {
return q4Burdens[i].CompositeBurden > q4Burdens[j].CompositeBurden
})
}()

go func() {
defer ansWG.Done()
tiers := []struct {
name string
pred Predicate[model.NeighborhoodProfile]
}{
{
name: "Tier 1: High Vacancy (25+ Notices)",
pred: func(p model.NeighborhoodProfile) bool { return p.VacantCount >= 25 },
},
{
name: "Tier 2: Moderate Vacancy (5-24 Notices)",
pred: func(p model.NeighborhoodProfile) bool { return p.VacantCount >= 5 && p.VacantCount < 25 },
},
{
name: "Tier 3: Low Vacancy (0-4 Notices)",
pred: func(p model.NeighborhoodProfile) bool { return p.VacantCount < 5 },
},
}

for _, t := range tiers {
subset := Filter(profiles, t.pred)
totalReq := Fold(subset, 0, func(acc int, p model.NeighborhoodProfile) int {
return acc + p.Total311Count
})
totalApp := Fold(subset, 0, func(acc int, p model.NeighborhoodProfile) int {
return acc + p.AppSubmissions
})
totalPhone := Fold(subset, 0, func(acc int, p model.NeighborhoodProfile) int {
return acc + p.PhoneSubmissions
})
totalWeb := Fold(subset, 0, func(acc int, p model.NeighborhoodProfile) int {
return acc + p.WebSubmissions
})
totalOpen := Fold(subset, 0, func(acc int, p model.NeighborhoodProfile) int {
return acc + p.Open311Count
})

var appPct, phonePct, webPct, closePct float64
if totalReq > 0 {
appPct = float64(totalApp) * 100.0 / float64(totalReq)
phonePct = float64(totalPhone) * 100.0 / float64(totalReq)
webPct = float64(totalWeb) * 100.0 / float64(totalReq)
closePct = float64(totalReq-totalOpen) * 100.0 / float64(totalReq)
}

q5Channels = append(q5Channels, model.ChannelTierSummary{
TierName:       t.name,
Neighborhoods:  len(subset),
TotalRequests:  totalReq,
MobileAppPct:   appPct,
PhoneCallPct:   phonePct,
WebPortalPct:   webPct,
ClosureRatePct: closePct,
})
}
}()

ansWG.Wait()
analysisDuration := time.Since(analysisStart)

var mem runtime.MemStats
runtime.ReadMemStats(&mem)

return model.AnalysisReport{
Telemetry: model.Telemetry{
WorkersUsed:       e.workers,
IngestDuration:    ingestDuration,
JoinDuration:      joinDuration,
AnalysisDuration:  analysisDuration,
Total311Parsed:    len(requests),
TotalVacantParsed: len(vacants),
AllocMB:           float64(mem.Alloc) / (1024 * 1024),
TotalAllocMB:      float64(mem.TotalAlloc) / (1024 * 1024),
SysMB:             float64(mem.Sys) / (1024 * 1024),
NumGC:             mem.NumGC,
},
Q1BlightHotspots:   q1Hotspots,
Q2AgencySLAs:       q2Agencies,
Q3SpatialHazards:   q3TopMatch,
Q3TotalHazardCount: len(spatialMatches),
Q3ByNeighborhood:   spatialByHood,
Q4DistrictBurdens:  q4Burdens,
Q5ChannelEquity:    q5Channels,
}
}
