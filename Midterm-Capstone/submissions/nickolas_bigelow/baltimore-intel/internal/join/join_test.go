package join

import (
	"sync/atomic"
	"testing"

	"baltimore-intel/internal/model"
)

func TestParallelReduceMatchesSequential(t *testing.T) {
	items := make([]int, 100_003)
	for i := range items {
		items[i] = i % 97
	}
	want := map[int]int{}
	for _, v := range items {
		want[v]++
	}
	for _, w := range []int{1, 3, 8, 64} {
		got := ParallelReduce(items, w,
			func() map[int]int { return map[int]int{} },
			func(m map[int]int, v *int) { m[*v]++ },
			func(dst, src map[int]int) {
				for k, v := range src {
					dst[k] += v
				}
			})
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("workers=%d key %d: %d want %d", w, k, got[k], v)
			}
		}
	}
	empty := ParallelReduce([]int{}, 4, func() map[int]int { return map[int]int{} },
		func(map[int]int, *int) {}, func(_, _ map[int]int) {})
	if len(empty) != 0 {
		t.Fatal("empty input must give empty result")
	}
}

func TestParallelForVisitsEachIndexOnce(t *testing.T) {
	var hits [1000]int32
	ParallelFor(len(hits), 7, func(i int) { atomic.AddInt32(&hits[i], 1) })
	for i, h := range hits {
		if h != 1 {
			t.Fatalf("index %d visited %d times", i, h)
		}
	}
}

func TestBuildJoins(t *testing.T) {
	ds := model.NewDataset()
	ds.RadiusM = 100
	canton := ds.Hoods.ID("Canton")
	fells := ds.Hoods.ID("FELLS POINT")
	// outlet in Canton, with one violent crime 50 m away and one 500 m away
	ds.Outlets = []model.LiquorOutlet{{Name: "Bar", Lat: 39.2808, Lon: -76.5747}}
	ds.Crimes = []model.Crime{
		{Lat: 39.28125, Lon: -76.5747, Hood: canton, Violent: true, Incidents: 1},
		{Lat: 39.2853, Lon: -76.5747, Hood: canton, Violent: true, Incidents: 1},
		{Hood: fells, Incidents: 1}, // no coordinates: counts for neighborhood only
	}
	for i := 0; i < 10; i++ {
		ds.Requests = append(ds.Requests, model.ServiceRequest{Lat: 39.2809, Lon: -76.5748, Hood: canton})
	}
	ds.Vacants = []model.VacantNotice{{Hood: canton}, {Hood: fells}}
	j := Build(ds, 4)
	if ds.Outlets[0].Hood != canton {
		t.Fatalf("outlet assigned to %q, want Canton", ds.Hoods.Name(ds.Outlets[0].Hood))
	}
	if j.Outlet[0].Violent != 1 {
		t.Fatalf("crimes within 100 m = %d, want 1", j.Outlet[0].Violent)
	}
	if !j.CrimeNearOutlet[0] || j.CrimeNearOutlet[1] || j.CrimeNearOutlet[2] {
		t.Fatalf("near flags = %v", j.CrimeNearOutlet)
	}
	h := j.Hood[canton]
	if h.Crimes != 2 || h.Violent != 2 || h.Requests != 10 || h.Vacants != 1 || h.Outlets != 1 {
		t.Fatalf("Canton stats = %+v", h)
	}
	if j.Hood[fells].Crimes != 1 || j.Hood[fells].Vacants != 1 {
		t.Fatalf("Fells Point stats = %+v", j.Hood[fells])
	}
}
