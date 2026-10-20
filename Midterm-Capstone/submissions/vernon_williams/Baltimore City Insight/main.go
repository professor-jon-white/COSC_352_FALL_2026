package main
import "fmt"
func main() {
	results := make(chan Result, 2)
	go loadConcurrent("data/crime.csv", "viol23", results)
	go loadConcurrent("data/poverty.csv", "hhpov23", results)

	var crimeData map[string]string
	var povertyData map[string]string

	for i :=0; i < 2; i++ {
		result := <-results
		if result.Err != nil {
			fmt.Println("Error:", result.Err)
			return
		}
		if result.Filename == "data/crime.csv" {
			crimeData = result.Data
		} else {
			povertyData = result.Data
		}
	}
	matched := 0
	for community, crime := range crimeData {
		poverty, exists := povertyData[community]
		if exists{
			fmt.Println(community, "Crime:", crime, "Poverty:", poverty)
			matched++
		}
	}
	fmt.Println("Total joined communities:", matched)
	communities := processCommunities(crimeData, povertyData)
	fmt.Println("Communites ready for analysis:", len(communities))
	
	processedCrime := make(map[string]string)
	processedPoverty := make(map[string]string)
	for _, community := range communities {
		processedCrime[community.Name] = fmt.Sprintf("%f", community.Crime)
		processedPoverty[community.Name] = fmt.Sprintf("%f", community.Poverty)
	}

	highestCrime(processedCrime)
	highestPoverty(processedPoverty)
	averageCrime, averagePoverty := calculateAverages(processedCrime, processedPoverty)
	highCrimeAndPoverty(processedCrime, processedPoverty, averageCrime, averagePoverty)
	highCrimeLowPoverty(processedCrime, processedPoverty, averageCrime, averagePoverty)
	
	displayCrimeChart(communities)
	displayPovertyChart(communities)
}