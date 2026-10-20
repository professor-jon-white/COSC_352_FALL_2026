package main

import (
	"fmt"
	"sort"
)

func reportDumping(
	matched []string,
	notices map[string]int,
) error {
	data, err := getServiceData()
	if err != nil {
		return err
	}

	ranked := append([]string(nil), matched...)

	sort.Slice(ranked, func(i, j int) bool {
		if notices[ranked[i]] == notices[ranked[j]] {
			return ranked[i] < ranked[j]
		}
		return notices[ranked[i]] > notices[ranked[j]]
	})

	limit := 10
	if len(ranked) < limit {
		limit = len(ranked)
	}
	selected := ranked[:limit]

	sort.Slice(selected, func(i, j int) bool {
		left := data.Neighborhoods[selected[i]].Dumping
		right := data.Neighborhoods[selected[j]].Dumping

		if left == right {
			return selected[i] < selected[j]
		}
		return left > right
	})

	fmt.Println("\nQUESTION 3: Illegal dumping in the top-ten notice neighborhoods")
	fmt.Println("All illegal-dumping records:", data.Totals.Dumping)
	fmt.Println(
		"Illegal-dumping records missing neighborhood:",
		data.Neighborhoods[""].Dumping,
	)
	fmt.Printf(
		"%-45s %10s %12s\n",
		"Neighborhood", "Notices", "Dumping",
	)

	for _, name := range selected {
		fmt.Printf(
			"%-45s %10d %12d\n",
			name, notices[name], data.Neighborhoods[name].Dumping,
		)
	}

	fmt.Println("These are neighborhood-level matches, not address-level matches.")
	return nil
}
