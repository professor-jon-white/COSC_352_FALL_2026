package main

import (
	"fmt"
	"strings"
	"time"
)

// parseMonth converts dates from both datasets
// into the same YYYY-MM format.
func parseMonth(value string) (string, error) {
	value = strings.TrimSpace(value)

	formats := []string{
		"2006/01/02 15:04:05-07",
		"2006/01/02 15:04:05-07:00",
		"1/2/2006 3:04:05 PM",
		"01/02/2006 03:04:05 PM",
	}

	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return parsed.Format("2006-01"), nil
		}
	}

	return "", fmt.Errorf("unsupported date format: %q", value)
}
