package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// streamNeighborhoods reads a CSV file one record at a time.
func streamNeighborhoods(
	filename string,
	jobs chan<- string,
) (int, error) {

	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return 0, err
	}

	index := findColumn(headers, "Neighborhood")
	if index == -1 {
		return 0, fmt.Errorf(
			"Neighborhood column not found in %s",
			filename,
		)
	}

	count := 0

	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return count, err
		}

		count++

		if index >= len(record) {
			continue
		}

		name := strings.ToUpper(
			strings.TrimSpace(record[index]),
		)

		if name != "" {
			jobs <- name
		}
	}

	return count, nil
}
