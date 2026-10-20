package main

import (
	"fmt"
	"sort"
)

type BacklogGroup struct {
	Neighborhoods int
	Requests      int
	Unresolved    int
}

func reportBacklog(notices map[string]int) error {
	data, err := getServiceData()
	if err != nil {
		return err
	}

	names := make([]string, 0, len(notices))
	for name := range notices {
		names = append(names, name)
	}
	sort.Strings(names)

	groups := [2]BacklogGroup{}
	matchedRequests := 0

	for _, name := range names {
		metrics := data.Neighborhoods[name]
		if metrics.Known == 0 {
			continue
		}

		group := 0
		if notices[name] >= 100 {
			group = 1
		}

		groups[group].Neighborhoods++
		groups[group].Requests += metrics.Known
		groups[group].Unresolved += metrics.Unresolved
		matchedRequests += metrics.Known
	}

	fmt.Println("\nQUESTION 4: Unresolved requests by neighborhood notice count")
	fmt.Println("Unresolved means New, Open, Hold, or Pending in the export.")
	fmt.Println("Records marked Duplicate excluded:", data.Totals.Duplicates)
	fmt.Println("Unknown-status records excluded:", data.Totals.Unknown)
	fmt.Println(
		"Remaining records without a notice-neighborhood match:",
		data.Totals.Known-matchedRequests,
	)

	labels := [2]string{"Below 100 notices", "100 or more notices"}

	for index, group := range groups {
		fmt.Println("\nGroup:", labels[index])
		fmt.Println("Neighborhoods:", group.Neighborhoods)
		fmt.Println("Included request records:", group.Requests)
		fmt.Println("Unresolved request records:", group.Unresolved)

		if group.Requests == 0 {
			fmt.Println("Unresolved percentage: unavailable")
		} else {
			percentage := 100 * float64(group.Unresolved) /
				float64(group.Requests)
			fmt.Printf("Unresolved percentage: %.2f%%\n", percentage)
		}
	}

	fmt.Println("\nPercentages pool request records within each group.")
	fmt.Println("Closed (Transferred) does not establish that an issue was fixed.")
	return nil
}
