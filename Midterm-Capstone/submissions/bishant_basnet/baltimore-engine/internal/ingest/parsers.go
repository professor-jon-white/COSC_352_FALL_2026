package ingest

import (
	"time"

	"github.com/bishant/baltimore-engine/internal/model"
)

// RowParser is the Strategy interface for turning one raw CSV row into a
// typed record. Each dataset supplies its own strategy; the generic Load
// pipeline does not know or care which dataset it is reading.
type RowParser[T any] interface {
	Name() string
	Fields() map[string][]string // logical field -> accepted column names
	Required() []string
	Parse(row []string, s Schema) (T, bool)
}

// CrimeParser parses "Part 1 Crime Data".
type CrimeParser struct{ In *Interner }

func (CrimeParser) Name() string { return "Part 1 Crime Data" }
func (CrimeParser) Fields() map[string][]string {
	return map[string][]string{
		"datetime":     {"CrimeDateTime", "Crime_Date_Time", "CrimeDate", "DateTime", "Date"},
		"neighborhood": {"Neighborhood", "NeighborhoodName", "Nbhd"},
		"description":  {"Description", "CrimeDescription", "Offense"},
		"weapon":       {"Weapon"},
		"district":     {"District", "New_District", "PoliceDistrict", "Old_District"},
	}
}
func (CrimeParser) Required() []string { return []string{"neighborhood"} }

func (p CrimeParser) Parse(row []string, s Schema) (model.CrimeRecord, bool) {
	hood := NormalizeName(s.Get(row, "neighborhood"))
	if hood == "" {
		return model.CrimeRecord{}, false // cannot be joined
	}
	// Open Baltimore documents that NIBRS crime CSVs download in UTC.
	t, ok := ParseTimeIn(s.Get(row, "datetime"), time.UTC)
	return model.CrimeRecord{
		When:         t,
		HasTime:      ok,
		Neighborhood: p.In.Intern(hood),
		Description:  p.In.Intern(NormalizeName(s.Get(row, "description"))),
		Weapon:       p.In.Intern(NormalizeName(s.Get(row, "weapon"))),
		District:     p.In.Intern(NormalizeName(s.Get(row, "district"))),
	}, true
}

// ServiceRequestParser parses "311 Customer Service Requests".
type ServiceRequestParser struct{ In *Interner }

func (ServiceRequestParser) Name() string { return "311 Customer Service Requests" }
func (ServiceRequestParser) Fields() map[string][]string {
	return map[string][]string{
		"created":      {"CreatedDate", "CreateDate", "Created", "DateCreated", "OpenDate"},
		"closed":       {"CloseDate", "ClosedDate", "DateClosed"},
		"neighborhood": {"Neighborhood", "NeighborhoodName", "Nbhd"},
		"type":         {"SRType", "RequestType", "ServiceRequestType", "Type"},
		"agency":       {"Agency"},
	}
}
func (ServiceRequestParser) Required() []string { return []string{"neighborhood"} }

func (p ServiceRequestParser) Parse(row []string, s Schema) (model.ServiceRequest, bool) {
	hood := NormalizeName(s.Get(row, "neighborhood"))
	if hood == "" {
		return model.ServiceRequest{}, false
	}
	c, cok := ParseTime(s.Get(row, "created"))
	d, dok := ParseTime(s.Get(row, "closed"))
	return model.ServiceRequest{
		Created:      c,
		Closed:       d,
		HasCreated:   cok,
		HasClosed:    dok,
		Neighborhood: p.In.Intern(hood),
		Type:         p.In.Intern(NormalizeName(s.Get(row, "type"))),
		Agency:       p.In.Intern(NormalizeName(s.Get(row, "agency"))),
	}, true
}
