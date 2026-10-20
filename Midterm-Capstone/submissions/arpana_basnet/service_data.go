package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ServiceMetrics stores summaries, not individual CSV records.
type ServiceMetrics struct {
	Records     int
	Dumping     int
	Duplicates  int
	Unknown     int
	Known       int
	Unresolved  int
	Closed      int
	MissingDate int
	InvalidDate int
	Negative    int
	ValidClosed int
	Days        float64
}

type ServiceData struct {
	Neighborhoods map[string]ServiceMetrics
	Totals        ServiceMetrics
}

// The three reports share one loaded summary per program execution.
var serviceOnce sync.Once
var serviceCache *ServiceData
var serviceError error

func getServiceData() (*ServiceData, error) {
	serviceOnce.Do(func() {
		serviceCache, serviceError =
			readServiceData("data/requests_2025.csv")
	})
	return serviceCache, serviceError
}

func addServiceMetrics(a, b ServiceMetrics) ServiceMetrics {
	a.Records += b.Records
	a.Dumping += b.Dumping
	a.Duplicates += b.Duplicates
	a.Unknown += b.Unknown
	a.Known += b.Known
	a.Unresolved += b.Unresolved
	a.Closed += b.Closed
	a.MissingDate += b.MissingDate
	a.InvalidDate += b.InvalidDate
	a.Negative += b.Negative
	a.ValidClosed += b.ValidClosed
	a.Days += b.Days
	return a
}

// Fields: neighborhood, request type, status, created date, close date.
type ServiceJob [5]string

func serviceWorker(
	jobs <-chan ServiceJob,
	results chan<- ServiceData,
	location *time.Location,
) {
	data := ServiceData{
		Neighborhoods: make(map[string]ServiceMetrics),
	}

	const dateLayout = "1/2/2006 3:04:05 PM"

	for job := range jobs {
		name := strings.ToUpper(strings.TrimSpace(job[0]))
		status := strings.ToUpper(strings.TrimSpace(job[2]))

		metrics := ServiceMetrics{Records: 1}

		if strings.EqualFold(
			strings.TrimSpace(job[1]), "HCD-Illegal Dumping",
		) {
			metrics.Dumping = 1
		}

		if strings.Contains(status, "DUPLICATE") {
			metrics.Duplicates = 1
		} else {
			switch status {
			case "NEW", "OPEN", "HOLD", "PENDING":
				metrics.Known = 1
				metrics.Unresolved = 1
			case "CLOSED", "CLOSED (TRANSFERRED)":
				metrics.Known = 1
			default:
				metrics.Unknown = 1
			}
		}

		if status == "CLOSED" {
			metrics.Closed = 1
			createdText := strings.TrimSpace(job[3])
			closedText := strings.TrimSpace(job[4])

			if createdText == "" || closedText == "" {
				metrics.MissingDate = 1
			} else {
				created, err1 := time.ParseInLocation(
					dateLayout, createdText, location,
				)
				closed, err2 := time.ParseInLocation(
					dateLayout, closedText, location,
				)

				if err1 != nil || err2 != nil {
					metrics.InvalidDate = 1
				} else if closed.Before(created) {
					metrics.Negative = 1
				} else {
					metrics.ValidClosed = 1
					metrics.Days = closed.Sub(created).Hours() / 24
				}
			}
		}

		current, exists := data.Neighborhoods[name]
		if !exists {
			name = strings.Clone(name)
		}

		data.Neighborhoods[name] = addServiceMetrics(current, metrics)
		data.Totals = addServiceMetrics(data.Totals, metrics)
	}

	results <- data
}

func readServiceData(path string) (*ServiceData, error) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	columns := make(map[string]int)
	for index, name := range headers {
		columns[strings.TrimSpace(name)] = index
	}

	required := []string{
		"Neighborhood", "SRType", "SRStatus", "CreatedDate", "CloseDate",
	}

	for _, name := range required {
		if _, exists := columns[name]; !exists {
			return nil, fmt.Errorf("column missing: %s", name)
		}
	}

	const workerCount = 4

	jobs := make(chan ServiceJob, 256)
	results := make(chan ServiceData, workerCount)

	for i := 0; i < workerCount; i++ {
		go serviceWorker(jobs, results, location)
	}

	var readErr error
	recordNumber := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			readErr = fmt.Errorf("%s: %w", path, err)
			break
		}

		recordNumber++

		if len(record) != len(headers) &&
			!(len(headers) == 28 && len(record) == 26) {
			readErr = fmt.Errorf(
				"%s: unexpected field count in record %d",
				path, recordNumber,
			)
			break
		}

		var job ServiceJob
		for index, name := range required {
			column := columns[name]
			if column >= len(record) {
				readErr = fmt.Errorf(
					"%s: record %d missing %s",
					path, recordNumber, name,
				)
				break
			}
			job[index] = record[column]
		}

		if readErr != nil {
			break
		}

		jobs <- job
	}

	close(jobs)

	combined := &ServiceData{
		Neighborhoods: make(map[string]ServiceMetrics),
	}

	// Wait for every worker, even if reading failed.
	for i := 0; i < workerCount; i++ {
		partial := <-results
		combined.Totals = addServiceMetrics(
			combined.Totals, partial.Totals,
		)

		for name, metrics := range partial.Neighborhoods {
			combined.Neighborhoods[name] = addServiceMetrics(
				combined.Neighborhoods[name], metrics,
			)
		}
	}

	if readErr != nil {
		return nil, readErr
	}

	return combined, nil
}
