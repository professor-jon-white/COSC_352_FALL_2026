package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type WbRecord struct {
	CrimeType    string
	CrimeYear    string
	Neighborhood string
}

type GVTRecord struct {
	CaseID           string
	OfficerName      string
	Neighborhood     string
	SettlementAmount string
	SettlementYear   string
}

type CVIRecord struct {
	Program      string
	Neighborhood string
	ReportYear   string
	Metric       string
	Value        string
	Unit         string
	Source       string
	SourceURL    string
}

func LoadWbData() ([]WbRecord, error) {
	filePath := filepath.Join("data", "wb_records.csv")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	var records []WbRecord
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) < 5 {
			return nil, fmt.Errorf("Empty row for well being CSV row. There's supposed to be 5 columns")
		}
		records = append(records, WbRecord{
			CrimeType:    line[2],
			CrimeYear:    line[4],
			Neighborhood: line[1],
		})
	}

	return records, nil
}

func LoadGVTData() ([]GVTRecord, error) {
	filePath := filepath.Join("data", "gvt results.csv")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	var records []GVTRecord
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) != 5 {
			return nil, fmt.Errorf("GVT CSV row has %d columns", len(line))
		}
		records = append(records, GVTRecord{
			CaseID:           line[0],
			OfficerName:      line[1],
			Neighborhood:     line[2],
			SettlementAmount: line[3],
			SettlementYear:   line[4],
		})
	}

	return records, nil
}

func LoadCVIData() ([]CVIRecord, error) {
	filePath := filepath.Join("data", "cvi_records.csv")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	var records []CVIRecord
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) < 8 {
			return nil, fmt.Errorf("There are %d columns for CVI CSV row. There should be 8", len(line))
		}
		records = append(records, CVIRecord{
			Neighborhood: line[0],
			Program:      line[1],
			ReportYear:   line[2],
			Metric:       line[3],
			Value:        line[4],
			Unit:         line[5],
			Source:       line[6],
			SourceURL:    line[7],
		})
	}

	return records, nil
}
