package main
import (
	"fmt"
	"sync"
)
type Job struct {
	Community string
	Crime string
	Poverty string
}
func worker(id int, jobs <-chan Job, results chan<- CommunityData, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		crime, err1 := convertNumber(job.Crime)
		poverty, err2 := convertNumber(job.Poverty)
	
	if err1 != nil || err2 != nil {
		fmt.Println("Worker", id, "skipped:", job.Community)
		continue
	}
	results <- CommunityData{
		Name: job.Community,
		Crime: crime,
		Poverty: poverty,
	}
  }
}
func processCommunities(crimeData map[string]string, povertyData map[string]string) []CommunityData {
	jobs := make(chan Job, 55)
	results := make(chan CommunityData, 55)

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go worker (i, jobs, results, &wg)
	}
	totalJobs := 0
	for community, crime := range crimeData {
		poverty, exists := povertyData[community]
		if exists {
			jobs <- Job{
				Community: community,
				Crime: crime,
				Poverty: poverty,
			}
			totalJobs++
		}
	}
	close(jobs)
	wg.Wait()
	close(results)
	communities := []CommunityData{}

	for result := range results {
		communities = append(communities, result)
	}
	fmt.Println("\n Worker pool finished processing.")
	fmt.Println("Jobs Submitted:", totalJobs)
	fmt.Println("Valid records processed:", len(communities))
	return communities
}