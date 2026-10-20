package main
import (
	"fmt"
	"sort"
	"strings"
)

func displayCrimeChart(communities []CommunityData) {
	sorted := make([]CommunityData, len(communities))
	copy(sorted, communities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Crime > sorted[j].Crime
	})
	fmt.Println("\n Top 5 Communities by Violent Crime Rate")
	fmt.Println(strings.Repeat("-",65))

	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	if limit == 0 {
		fmt.Println("No community data available")
		return
	}
	highest := sorted[0].Crime
	for i := 0; i < limit; i++ {
		community := sorted[i]
		barLength := 0
		if highest > 0 {
			barLength = int((community.Crime/highest) * 25)
		}
		bar := strings.Repeat("~", barLength)
		fmt.Printf("%-38s | %-25s %.2f\n", community.Name, bar, community.Crime)
	}
}
func displayPovertyChart(communities []CommunityData) {
	sorted := make([]CommunityData, len(communities))
	copy(sorted, communities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Poverty > sorted[j].Poverty
	})
	fmt.Println("\nTop 5 Communities by Poverty Percentage")
	fmt.Println(strings.Repeat("-", 65))
	
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	if limit == 0 {
		fmt.Println("No communtity data available")
		return
	}
	highest := sorted[0].Poverty
	for i := 0; i < limit; i++ {
		community := sorted[i]
		barLength := 0
		if highest > 0 {
			barLength = int((community.Poverty/highest)*25)
		}
		bar := strings.Repeat("~", barLength)
		fmt.Printf("%-38s | %-25s %.2f%%\n", community.Name, bar, community.Poverty)
	}
}