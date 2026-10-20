// Package model holds shared types and data-cleaning rules.
package model

import (
	"strings"
	"time"
)

const RequestURL = "https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/311_Customer_Service_Requests_2025/FeatureServer/0"
const VacancyURL = "https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1"
const PeriodStart = "2025-10-01"
const PeriodEnd = "2026-01-01"

type Feature struct {
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	RequestID string `json:"ServiceRequestNum"`
	NoticeID  string `json:"NoticeNum"`
	BlockLot  string `json:"BLOCKLOT"`
	Address   string `json:"Address"`
	Area      string `json:"Neighborhood"`
	Type      string `json:"SRType"`
	Status    string `json:"SRStatus"`
	Created   *int64 `json:"CreatedDate"`
	Closed    *int64 `json:"CloseDate"`
	Deleted   *int   `json:"IsDeleted"`
}

func Normalize(s string) string { return strings.Join(strings.Fields(strings.ToUpper(s)), " ") }

func (a Attributes) Key(kind string) string {
	if kind == "requests" {
		return Normalize(a.RequestID)
	}
	if k := Normalize(a.BlockLot); k != "" {
		return "parcel:" + k
	}
	if k := Normalize(a.Address); k != "" {
		return "address:" + k
	}
	return ""
}

func InPeriod(ms int64) bool {
	t := time.UnixMilli(ms).UTC()
	start, _ := time.Parse("2006-01-02", PeriodStart)
	end, _ := time.Parse("2006-01-02", PeriodEnd)
	return !t.Before(start) && t.Before(end)
}

// ClosedDays measures recorded closure, not successful resolution.
func ClosedDays(a Attributes) (float64, bool) {
	if a.Created == nil || a.Closed == nil || *a.Closed < *a.Created ||
		!strings.HasPrefix(Normalize(a.Status), "CLOSED") {
		return 0, false
	}
	return float64(*a.Closed-*a.Created) / 86400000, true
}

type Stats struct {
	Count     int
	Types     map[string]int
	Months    map[string]int
	Durations map[string][]float64
}

func NewStats() *Stats {
	return &Stats{Types: map[string]int{}, Months: map[string]int{}, Durations: map[string][]float64{}}
}

func (s *Stats) Add(a Attributes, kind string) {
	s.Count++
	if kind != "requests" {
		return
	}
	t := Normalize(a.Type)
	if t == "" {
		t = "(MISSING TYPE)"
	}
	s.Types[t]++
	month := time.UnixMilli(*a.Created).UTC().Format("2006-01")
	s.Months[month]++
	if days, ok := ClosedDays(a); ok {
		s.Durations[t] = append(s.Durations[t], days)
	}
}

type Quality struct {
	Lines               int
	InvalidJSON         int
	MissingID           int
	Duplicates          int
	MissingNeighborhood int
	MissingCreated      int
	OutsidePeriod       int
	Deleted             int
	Accepted            int
	InvalidClosure      int
}

type Dataset struct {
	Areas   map[string]*Stats
	Quality Quality
}
