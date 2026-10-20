package model

import (
"strings"
"time"
)

type ServiceRequest struct {
RecordID        string
SRType          string
MethodReceived  string
Status          string
Agency          string
Address         string
Neighborhood    string
PoliceDistrict  string
CouncilDistrict int
CreatedDate     time.Time
CloseDate       time.Time
ResolutionHours float64
Latitude        float64
Longitude       float64
IsClosed        bool
HasCoords       bool
}

type VacantBuilding struct {
ReferenceID     string
Block           string
Lot             string
Address         string
Neighborhood    string
PoliceDistrict  string
CouncilDistrict int
NoticeDate      time.Time
Latitude        float64
Longitude       float64
HasCoords       bool
}

type NeighborhoodProfile struct {
Name               string
VacantCount        int
Total311Count      int
Dumping311Count    int
Open311Count       int
SpatialHazardCount int
AvgResolutionHours float64
AppSubmissions     int
PhoneSubmissions   int
WebSubmissions     int
}

type AgencyMetrics struct {
AgencyName             string
HighVacancyAvgHours    float64
LowVacancyAvgHours     float64
HighVacancyTicketCount int
LowVacancyTicketCount  int
}

type DistrictBurden struct {
DistrictID        int
VacantNotices     int
Open311Backlog    int
Total311Volume    int
CompositeBurden   float64
}

type SpatialMatch struct {
RequestID       string
SRType          string
RequestAddr     string
VacantNoticeID  string
VacantAddr      string
Neighborhood    string
DistanceMeters  float64
}

type Telemetry struct {
WorkersUsed       int
IngestDuration    time.Duration
JoinDuration      time.Duration
AnalysisDuration  time.Duration
Total311Parsed    int
TotalVacantParsed int
AllocMB           float64
TotalAllocMB      float64
SysMB             float64
NumGC             uint32
}

type AnalysisReport struct {
Telemetry          Telemetry
Q1BlightHotspots   []NeighborhoodProfile
Q2AgencySLAs       []AgencyMetrics
Q3SpatialHazards   []SpatialMatch
Q3TotalHazardCount int
Q3ByNeighborhood   map[string]int
Q4DistrictBurdens  []DistrictBurden
Q5ChannelEquity    []ChannelTierSummary
}

type ChannelTierSummary struct {
TierName       string
Neighborhoods  int
TotalRequests  int
MobileAppPct   float64
PhoneCallPct   float64
WebPortalPct   float64
ClosureRatePct float64
}

func NormalizeKey(raw string) string {
s := strings.TrimSpace(strings.ToUpper(raw))
s = strings.ReplaceAll(s, "  ", " ")
return s
}
