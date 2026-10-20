package join

import (
	"sync"

	"baltimore-intel/internal/model"
	"baltimore-intel/internal/spatial"
)

// HoodStat is the per-neighborhood join of all four datasets on the
// normalized neighborhood key.
type HoodStat struct {
	Crimes, Violent      int
	Requests, Sanitation int
	Vacants, Outlets     int
}

// OutletExposure is the spatial join of one liquor outlet to nearby crime.
type OutletExposure struct {
	All, Violent int
}

// Joined holds every cross-dataset structure the analyses share. It is built
// once by Build and is read-only afterwards.
type Joined struct {
	DS              *model.Dataset
	Workers         int
	Hood            []HoodStat // indexed by neighborhood ID
	CrimeGrid       *spatial.Grid
	OutletGrid      *spatial.Grid
	RequestGrid     *spatial.Grid
	Outlet          []OutletExposure // indexed like DS.Outlets
	Baseline        []OutletExposure // same measure around comparison points
	CrimeNearOutlet []bool           // indexed like DS.Crimes
	OutletsAssigned int
}

const cellM = 100 // grid cell size in meters

// Build runs the cross-dataset joins. Independent steps run concurrently.
func Build(ds *model.Dataset, workers int) *Joined {
	j := &Joined{DS: ds, Workers: workers}
	r := ds.RadiusM

	// Step 1: build the three spatial indexes in parallel.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		j.CrimeGrid = spatial.Build(len(ds.Crimes), cellM, func(i int) (float64, float64, bool) {
			c := &ds.Crimes[i]
			return float64(c.Lat), float64(c.Lon), c.Lat != 0
		})
	}()
	go func() {
		defer wg.Done()
		j.OutletGrid = spatial.Build(len(ds.Outlets), cellM, func(i int) (float64, float64, bool) {
			o := &ds.Outlets[i]
			return float64(o.Lat), float64(o.Lon), true
		})
	}()
	go func() {
		defer wg.Done()
		j.RequestGrid = spatial.Build(len(ds.Requests), cellM, func(i int) (float64, float64, bool) {
			q := &ds.Requests[i]
			return float64(q.Lat), float64(q.Lon), q.Lat != 0 && q.Hood != 0
		})
	}()
	wg.Wait()

	// Step 2: liquor licenses carry only a street address, so each outlet is
	// assigned the majority neighborhood of the 311 requests around it.
	assigned := make([]bool, len(ds.Outlets))
	ParallelFor(len(ds.Outlets), workers, func(i int) {
		o := &ds.Outlets[i] // each goroutine writes only its own index
		near := j.RequestGrid.KNearest(float64(o.Lat), float64(o.Lon), 400, 9)
		votes := map[model.ID]int{}
		var best model.ID
		for _, idx := range near {
			h := ds.Requests[idx].Hood
			votes[h]++
			if votes[h] > votes[best] {
				best = h
			}
		}
		o.Hood = best
		assigned[i] = best != 0
	})
	for _, a := range assigned {
		if a {
			j.OutletsAssigned++
		}
	}

	// Step 3: the three remaining joins are independent: run them together.
	wg.Add(3)
	go func() { // crime within r of each outlet
		defer wg.Done()
		j.Outlet = make([]OutletExposure, len(ds.Outlets))
		ParallelFor(len(ds.Outlets), workers, func(i int) {
			o := &ds.Outlets[i]
			j.Outlet[i] = j.exposure(float64(o.Lat), float64(o.Lon), r)
		})
	}()
	go func() { // same measure around comparison points (see Baseline docs)
		defer wg.Done()
		pts := j.baselinePoints(len(ds.Outlets) * 2)
		j.Baseline = make([]OutletExposure, len(pts))
		ParallelFor(len(pts), workers, func(i int) {
			q := &ds.Requests[pts[i]]
			j.Baseline[i] = j.exposure(float64(q.Lat), float64(q.Lon), r)
		})
	}()
	go func() { // flag crimes that happened near any outlet
		defer wg.Done()
		j.CrimeNearOutlet = make([]bool, len(ds.Crimes))
		ParallelFor(len(ds.Crimes), workers, func(i int) {
			c := &ds.Crimes[i]
			if c.Lat != 0 {
				_, _, ok := j.OutletGrid.Nearest(float64(c.Lat), float64(c.Lon), r)
				j.CrimeNearOutlet[i] = ok
			}
		})
	}()
	wg.Wait()

	// Step 4: neighborhood join (needs outlet neighborhoods from step 2).
	j.Hood = j.hoodStats()
	return j
}

func (j *Joined) exposure(lat, lon, r float64) OutletExposure {
	var e OutletExposure
	j.CrimeGrid.Within(lat, lon, r, func(i int32, _ float64) {
		c := &j.DS.Crimes[i]
		e.All += int(c.Incidents)
		if c.Violent {
			e.Violent += int(c.Incidents)
		}
	})
	return e
}

// baselinePoints picks an evenly spaced, deterministic sample of 311 request
// locations. 311 requests come from where people live, work and travel, so
// they are a fairer comparison than uniformly random points (which would land
// in parks, the harbor and rail yards and understate "normal" crime levels).
func (j *Joined) baselinePoints(n int) []int {
	var cand []int
	for i := range j.DS.Requests {
		if j.DS.Requests[i].Lat != 0 {
			cand = append(cand, i)
		}
	}
	if n <= 0 || len(cand) == 0 {
		return nil
	}
	if n > len(cand) {
		n = len(cand)
	}
	out := make([]int, n)
	step := float64(len(cand)) / float64(n)
	for i := range out {
		out[i] = cand[int(float64(i)*step)]
	}
	return out
}

// hoodStats is a map-reduce over all four datasets keyed by neighborhood ID.
// Each dataset is reduced on its own goroutine (each of which fans out to
// `workers` shards), then the per-dataset vectors are summed.
func (j *Joined) hoodStats() []HoodStat {
	ds := j.DS
	n := ds.Hoods.Len()
	newAcc := func() *[]HoodStat { s := make([]HoodStat, n); return &s }
	merge := func(dst, src *[]HoodStat) {
		for i := range *src {
			d, s := &(*dst)[i], &(*src)[i]
			d.Crimes += s.Crimes
			d.Violent += s.Violent
			d.Requests += s.Requests
			d.Sanitation += s.Sanitation
			d.Vacants += s.Vacants
			d.Outlets += s.Outlets
		}
	}
	parts := make([]*[]HoodStat, 4)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		parts[0] = ParallelReduce(ds.Crimes, j.Workers, newAcc, func(a *[]HoodStat, c *model.Crime) {
			(*a)[c.Hood].Crimes += int(c.Incidents)
			if c.Violent {
				(*a)[c.Hood].Violent += int(c.Incidents)
			}
		}, merge)
	}()
	go func() {
		defer wg.Done()
		parts[1] = ParallelReduce(ds.Requests, j.Workers, newAcc, func(a *[]HoodStat, q *model.ServiceRequest) {
			(*a)[q.Hood].Requests++
			if q.Sanitation {
				(*a)[q.Hood].Sanitation++
			}
		}, merge)
	}()
	go func() {
		defer wg.Done()
		parts[2] = ParallelReduce(ds.Vacants, j.Workers, newAcc, func(a *[]HoodStat, v *model.VacantNotice) {
			(*a)[v.Hood].Vacants++
		}, merge)
	}()
	go func() {
		defer wg.Done()
		parts[3] = ParallelReduce(ds.Outlets, j.Workers, newAcc, func(a *[]HoodStat, o *model.LiquorOutlet) {
			(*a)[o.Hood].Outlets++
		}, merge)
	}()
	wg.Wait()
	out := parts[0]
	for _, p := range parts[1:] {
		merge(out, p)
	}
	return *out
}

// Hoods returns neighborhood IDs (excluding "unknown") that appear in every
// listed measure with a nonzero count, i.e. neighborhoods matched across datasets.
func (j *Joined) Hoods(require func(HoodStat) bool) []model.ID {
	var ids []model.ID
	for id := 1; id < len(j.Hood); id++ {
		if require == nil || require(j.Hood[id]) {
			ids = append(ids, model.ID(id))
		}
	}
	return ids
}
