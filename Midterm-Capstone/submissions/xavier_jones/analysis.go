package main

import (
	"sort"
	"sync"
)

// Pipeline stage: bounded worker maps are private; the incident index is immutable during workers.
func analyze(crimes, arrests []Row, workers int) Result {
	result := newResult()
	result.CrimeRows = len(crimes)
	result.ArrestRows = len(arrests)
	firstArrest := map[string]int64{}
	linked := map[string]bool{}
	arrestIDs := map[string]bool{}
	for _, a := range arrests {
		id := incidentID(a, "IncidentNumber")
		if id == "" {
			result.MissingArrestID++
		} else {
			linked[id] = true
			date := epoch(a["ArrestDateTime"])
			if date > 0 && (firstArrest[id] == 0 || date < firstArrest[id]) {
				firstArrest[id] = date
			}
		}
		if id := incidentID(a, "ArrestNumber"); id != "" {
			arrestIDs[id] = true
		}
		n := label(a, "Neighborhood")
		if n == "UNKNOWN" {
			result.MissingArrestNeighborhood++
		}
		cell(result.Neighborhoods, n).Arrests++
		m := month(a["ArrestDateTime"])
		if m == "UNKNOWN" {
			result.MissingArrestDate++
		}
		cell(result.Months, m).Arrests++
	}
	result.UniqueArrests = len(arrestIDs)
	jobs := make(chan Row, 2*workers)
	partials := make(chan Result, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := newResult()
			for c := range jobs {
				id := incidentID(c, "CCNumber")
				if id == "" {
					p.MissingCrimeID++
				}
				n := label(c, "Neighborhood")
				if n == "UNKNOWN" {
					p.MissingCrimeNeighborhood++
				}
				m := month(c["CrimeDateTime"])
				if m == "UNKNOWN" {
					p.MissingCrimeDate++
				}
				for _, g := range []*Cell{cell(p.Neighborhoods, n), cell(p.Months, m), cell(p.Categories, label(c, "Description")), cell(p.Weapons, label(c, "Weapon"))} {
					g.Crimes++
					if id != "" && linked[id] {
						g.Linked++
					}
				}
			}
			partials <- p
		}()
	}
	go func() {
		for _, c := range crimes {
			jobs <- c
		}
		close(jobs)
		wg.Wait()
		close(partials)
	}()
	for p := range partials {
		result.MissingCrimeID += p.MissingCrimeID
		result.MissingCrimeNeighborhood += p.MissingCrimeNeighborhood
		result.MissingCrimeDate += p.MissingCrimeDate
		for i, src := range []map[string]*Cell{p.Neighborhoods, p.Months, p.Categories, p.Weapons} {
			dst := []map[string]*Cell{result.Neighborhoods, result.Months, result.Categories, result.Weapons}[i]
			for k, c := range src {
				g := cell(dst, k)
				g.Crimes += c.Crimes
				g.Linked += c.Linked
			}
		}
	}
	for _, c := range crimes {
		n := label(c, "Neighborhood")
		if result.CategoryByNeighborhood[n] == nil {
			result.CategoryByNeighborhood[n] = map[string]int{}
			result.WeaponByNeighborhood[n] = map[string]int{}
		}
		result.CategoryByNeighborhood[n][label(c, "Description")]++
		result.WeaponByNeighborhood[n][label(c, "Weapon")]++
	}
	incidents := map[string]Incident{}
	for _, c := range crimes {
		id := incidentID(c, "CCNumber")
		if id == "" {
			continue
		}
		date := epoch(c["CrimeDateTime"])
		old, ok := incidents[id]
		if !ok || (date > 0 && (old.When == 0 || date < old.When)) {
			incidents[id] = Incident{date, label(c, "Neighborhood")}
		}
	}
	result.UniqueIncidents = len(incidents)
	for id, c := range incidents {
		if !linked[id] {
			continue
		}
		result.MatchedIncidents++
		a := firstArrest[id]
		if c.When == 0 || a == 0 {
			continue
		}
		if a < c.When {
			result.NegativeLag++
			continue
		}
		result.Lags = append(result.Lags, float64(a-c.When)/3600000)
	}
	sort.Float64s(result.Lags)
	return result
}
