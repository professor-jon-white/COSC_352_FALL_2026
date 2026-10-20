package main

import (
	"fmt"
	"strings"
	"time"
)

// RecordParser is the Strategy interface.
type RecordParser interface {
	ParseRow(row []string) (interface{}, error)
	Name() string
}

type ServiceRequestParser struct{}

func NewServiceRequestParser() *ServiceRequestParser { return &ServiceRequestParser{} }

func (p *ServiceRequestParser) Name() string { return "311 Service Requests" }

func (p *ServiceRequestParser) ParseRow(row []string) (interface{}, error) {
	if len(row) < 5 {
		return nil, fmt.Errorf("expected 5 columns, got %d", len(row))
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(row[4]))
	if err != nil {
		return nil, fmt.Errorf("bad date %q: %v", row[4], err)
	}
	return ServiceRequest{
		SRNumber:     strings.TrimSpace(row[0]),
		SRType:       strings.TrimSpace(row[1]),
		Neighborhood: strings.TrimSpace(row[2]),
		District:     strings.TrimSpace(row[3]),
		CreatedDate:  date,
	}, nil
}

type CrimeParser struct{}

func NewCrimeParser() *CrimeParser { return &CrimeParser{} }

func (p *CrimeParser) Name() string { return "Crime Incidents" }

func (p *CrimeParser) ParseRow(row []string) (interface{}, error) {
	if len(row) < 5 {
		return nil, fmt.Errorf("expected 5 columns, got %d", len(row))
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(row[4]))
	if err != nil {
		return nil, fmt.Errorf("bad date %q: %v", row[4], err)
	}
	return CrimeIncident{
		CrimeCode:    strings.TrimSpace(row[0]),
		Description:  strings.TrimSpace(row[1]),
		Neighborhood: strings.TrimSpace(row[2]),
		District:     strings.TrimSpace(row[3]),
		CrimeDate:    date,
	}, nil
}
