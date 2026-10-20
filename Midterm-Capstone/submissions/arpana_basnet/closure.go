package main

import (
	"fmt"
	"sort"
)

type ClosureGroup struct {
	Records   int
	TotalDays float64
}

func reportClosureTimes(notices map[string]int) error {
	data, err := getServiceData()
	if err != nil {
		return err
	}

	names := make([]string, 0, len(notices))
	for name := range notices {
		names = append(names, name)
	}
	sort.Strings(names)

	groups := [2]ClosureGroup{}
	matched := ServiceMetrics{}

	for _, name := range names {
		metrics := data.Neighborhoods[name]
		matched = addServiceMetrics(matched, metrics)

		group := 0
		if notices[name] >= 100 {
			group = 1
		}

		groups[group].Records += metrics.ValidClosed
		groups[group].TotalDays += metrics.Days
	}

	fmt.Println("\nQUESTION 5: Average recorded closure time by notice count")
	fmt.Println("Included status: exactly Closed.")
	fmt.Println("Dates interpreted in Baltimore's local time zone.")
	fmt.Println(
		"Other-status records excluded:",
		data.Totals.Records-data.Totals.Closed,
	)
	fmt.Println(
		"Closed records without a notice-neighborhood match:",
		data.Totals.Closed-matched.Closed,
	)
	fmt.Println("Matched closed records missing dates:", matched.MissingDate)
	fmt.Println("Matched closed records with invalid dates:", matched.InvalidDate)
	fmt.Println("Matched closed records with closure before creation:", matched.Negative)

	labels := [2]string{"Below 100 notices", "100 or more notices"}

	for index, group := range groups {
		fmt.Println("\nGroup:", labels[index])
		fmt.Println("Included closed records:", group.Records)

		if group.Records == 0 {
			fmt.Println("Average closure time: unavailable")
		} else {
			average := group.TotalDays / float64(group.Records)
			fmt.Printf("Average closure time: %.2f days\n", average)
		}
	}

	fmt.Println("\nThis measures recorded closure time, not verified issue resolution.")
	fmt.Println("Request types may differ between groups; averages can be affected by outliers.")
	return nil
}
