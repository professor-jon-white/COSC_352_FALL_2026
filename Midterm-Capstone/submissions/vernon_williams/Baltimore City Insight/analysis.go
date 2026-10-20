package main
import "fmt"
func highestCrime(crimeData map[string]string) {
	highest := -1.0
	highestCommunity := ""

	for community, value := range crimeData {
		crime, err := convertNumber(value)
		if err != nil {
			fmt.Println("Invalid crime rate for:", community)
			continue
		}
		if crime > highest {
			highest = crime
			highestCommunity = community
		}
	}
	fmt.Println("\nResearch Question 1:")
	fmt.Println("Which community has the highest violent crime rate?")
	fmt.Printf("%s: %.2f crimes per 1000 residents\n", highestCommunity, highest)
}
func highestPoverty(povertyData map[string]string) {
	highest := -1.0
	highestCommunity := ""

	for community, value := range povertyData {
		poverty, err := convertNumber(value)
		if err != nil {
			fmt.Println("Invalid poverty rate for:", community)
			continue
		}
		if poverty > highest {
			highest = poverty
			highestCommunity = community
		}
	}
	fmt.Println("\nResearch Question 2:")
	fmt.Println("Which community has the highest poverty percentage?")
	fmt.Printf("%s: %.2f%%\n", highestCommunity, highest)
}
func calculateAverages(crimeData map[string]string, povertyData map[string]string) (float64, float64) {
	totalCrime := 0.0
	totalPoverty := 0.0
	count := 0

	for community, crimeValue := range crimeData {
		povertyValue, exists := povertyData[community]
		if !exists {
			continue
		}
		crime, err1 := convertNumber(crimeValue)
		poverty, err2 := convertNumber(povertyValue)
		if err1 != nil || err2 != nil {
			continue
		}
		totalCrime += crime
		totalPoverty += poverty
		count++
	}
	if count == 0 {
		fmt.Println("No valid data found")
		return 0, 0
	}
	averageCrime := totalCrime / float64(count)
	averagePoverty := totalPoverty / float64(count)
	
	fmt.Println("\nResearch Question 3:")
	fmt.Println("What are the average crime and poverty rates?")
	fmt.Printf("Average crime rate : %.2f per 1000 Residents", averageCrime)
	fmt.Printf("Average poverty: %.2f%%\n", averagePoverty)
	return averageCrime, averagePoverty
}
func  highCrimeAndPoverty(crimeData map[string]string, povertyData map[string]string, averageCrime float64, averagePoverty float64) {
	count := 0
	
	for community, crimeValue := range crimeData {
		povertyValue, exists := povertyData[community]
		if !exists {
			continue
		}
		crime, err1 := convertNumber(crimeValue)
		poverty, err2 := convertNumber(povertyValue)
		if err1 != nil || err2 != nil {
			continue
		}
		if crime > averageCrime && poverty > averagePoverty {
			fmt.Print("%s | Crime: %.2f | Poverty: %.2f%%\n", community, crime, poverty)
			count++
		}
	}
	fmt.Println("\nResearch Question 4:")
	fmt.Println("Which communities have above average crime and poverty?")
	fmt.Println("Total communities above bothe averages:\n", count)
}
func highCrimeLowPoverty(crimeData map[string]string, povertyData map[string]string, averageCrime float64, averagePoverty float64) {
	count := 0

	for community, crimeValue := range crimeData {
		povertyValue, exists := povertyData[community]
		if !exists {
			continue
		}
		crime, err1 := convertNumber(crimeValue)
		poverty, err2 := convertNumber(povertyValue)
		if err1 != nil || err2 != nil {
			continue
		}
		if crime > averageCrime && poverty < averagePoverty {
			fmt.Printf("%s | Crime: %.2f | Poverty: %.2f%%\n", community, crime, poverty)
			count++
		}
	}
	fmt.Println("\nResearch Question 5:")
	fmt.Println("Which communities have a high crime rate but low poverty rate?")
	fmt.Println("Total communities meeting both conditions:", count)
}