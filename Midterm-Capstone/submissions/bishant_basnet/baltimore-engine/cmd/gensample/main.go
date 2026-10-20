// Command gensample writes SYNTHETIC CSVs that use the same column layout as
// Open Baltimore's Part 1 Crime Data and 311 Customer Service Requests, so the
// engine can be built and verified without downloading anything.
// The numbers are randomly generated and are NOT real Baltimore statistics.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var hoods = []string{"Downtown", "Fells Point", "Canton", "Federal Hill", "Mount Vernon",
	"Sandtown-Winchester", "Upton", "Harlem Park", "Belair-Edison", "Hamilton Hills",
	"Hampden", "Roland Park", "Charles Village", "Waverly", "Coldstream Homestead Montebello",
	"Broadway East", "Oliver", "Brooklyn", "Cherry Hill", "Highlandtown",
	"Patterson Park Neighborhood", "Greektown", "Druid Heights", "Penn North",
	"Reservoir Hill", "Frankford", "Cedonia", "Northwood", "Glen", "Washington Village/Pigtown",
	"Remington", "Lauraville", "Ednor Gardens-Lakeside", "Carroll-Camden Industrial Area",
	"Midtown-Edmondson", "Edmondson Village", "Mondawmin", "Park Heights", "Pimlico Good Neighbors", "Curtis Bay"}

type picker struct{ cum []float64 }

func newPicker(w []float64) picker {
	c := make([]float64, len(w))
	s := 0.0
	for i, x := range w {
		s += x
		c[i] = s
	}
	for i := range c {
		c[i] /= s
	}
	return picker{c}
}
func (p picker) pick(r *rand.Rand) int {
	i := sort.SearchFloat64s(p.cum, r.Float64())
	if i >= len(p.cum) {
		i = len(p.cum) - 1
	}
	return i
}

func arcgis(t time.Time) string { return t.UTC().Format("2006/01/02 15:04:05") + "+00" }

func main() {
	out := flag.String("out", "data/sample", "output directory")
	nCrime := flag.Int("crimes", 80000, "number of crime rows")
	nReq := flag.Int("requests", 150000, "number of 311 rows")
	seed := flag.Int64("seed", 42, "random seed")
	flag.Parse()
	r := rand.New(rand.NewSource(*seed))
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}

	risk := make([]float64, len(hoods))
	cw, rw := make([]float64, len(hoods)), make([]float64, len(hoods))
	for i := range hoods {
		risk[i] = r.Float64()
		cw[i] = 0.2 + 3*math.Pow(risk[i], 1.5)
		rw[i] = 0.6 + 1.4*risk[i] + 0.6*r.Float64()
		if risk[i] > 0.7 && r.Float64() < 0.35 { // some high-crime areas under-report
			rw[i] *= 0.4
		}
	}
	cp, rp := newPicker(cw), newPicker(rw)
	crimeHourW := []float64{6, 5, 4, 3, 2, 1.5, 1.5, 2, 3, 3.5, 4, 4.5, 5, 5, 5, 5.5, 6, 6.5, 7, 7.5, 8, 8, 7.5, 7}
	reqHourW := []float64{0.3, 0.2, 0.2, 0.2, 0.3, 0.6, 1.5, 4, 8, 10, 10, 9.5, 9, 9, 8.5, 8, 6, 3.5, 2, 1.5, 1.2, 1, 0.7, 0.5}
	chp, rhp := newPicker(crimeHourW), newPicker(reqHourW)
	year := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	when := func(hp picker) time.Time {
		return year.AddDate(0, 0, r.Intn(366)).Add(time.Duration(hp.pick(r))*time.Hour + time.Duration(r.Intn(3600))*time.Second)
	}

	descs := []string{"LARCENY", "COMMON ASSAULT", "BURGLARY", "AUTO THEFT", "AGG. ASSAULT", "ROBBERY - STREET", "LARCENY FROM AUTO", "SHOOTING", "HOMICIDE"}
	dp := newPicker([]float64{30, 20, 12, 12, 10, 6, 8, 1.5, 0.5})
	weapons := []string{"NA", "FIREARM", "KNIFE", "OTHER", "HANDS"}
	wp := newPicker([]float64{60, 15, 7, 8, 10})
	districts := []string{"CENTRAL", "EASTERN", "WESTERN", "NORTHERN", "SOUTHERN", "NORTHEAST", "NORTHWEST", "SOUTHEAST", "SOUTHWEST"}

	f, _ := os.Create(filepath.Join(*out, "crime_sample.csv"))
	w := csv.NewWriter(f)
	w.Write([]string{"RowID", "CrimeDateTime", "CrimeCode", "Description", "Inside_Outside", "Weapon", "District", "Neighborhood", "Latitude", "Longitude"})
	for i := 0; i < *nCrime; i++ {
		h := hoods[cp.pick(r)]
		if r.Float64() < 0.01 {
			h = "" // realistic: some rows lack a neighborhood
		}
		ts := when(chp).UTC().Format("2006/01/02 15:04:05")
		if r.Float64() < 0.002 {
			ts = "" // and some lack a timestamp
		}
		w.Write([]string{fmt.Sprint(i + 1), ts, fmt.Sprintf("%dX", 1+r.Intn(9)), descs[dp.pick(r)],
			[]string{"I", "O"}[r.Intn(2)], weapons[wp.pick(r)], districts[r.Intn(len(districts))], h,
			fmt.Sprintf("%.5f", 39.29+r.Float64()*0.06-0.03), fmt.Sprintf("%.5f", -76.61+r.Float64()*0.06-0.03)})
	}
	w.Flush()
	f.Close()

	blight := []string{"SW-Dirty Alley", "SW-Rat Rubout", "SW-Trash Can/Recycling Container Complaint", "HCD-Vacant Building", "SW-Illegal Dumping", "SW-Bulk Scheduled", "HCD-Graffiti Removal"}
	other := []string{"TRM-Pothole", "BGE-StLight(s) Out", "WW-Water Leak (Exterior)", "TRS-Parking Complaint", "ECC-Noise Complaint", "FOR-Tree Pruning", "TRM-Sidewalk Repair"}
	agency := map[byte]string{'S': "Solid Waste", 'H': "Housing", 'T': "Transportation", 'B': "BGE", 'W': "Water Wastewater", 'E': "ECC", 'F': "Forestry"}

	f, _ = os.Create(filepath.Join(*out, "311_sample.csv"))
	w = csv.NewWriter(f)
	w.Write([]string{"ServiceRequestNum", "SRType", "MethodReceived", "CreatedDate", "SRStatus", "CloseDate", "Agency", "Address", "Neighborhood", "CouncilDistrict"})
	for i := 0; i < *nReq; i++ {
		hi := rp.pick(r)
		var t string
		if r.Float64() < 0.15+0.45*risk[hi] {
			t = blight[r.Intn(len(blight))]
		} else {
			t = other[r.Intn(len(other))]
		}
		created := when(rhp)
		status, closed := "Closed", ""
		if r.Float64() < 0.08 {
			status = "Open"
		} else {
			hrs := r.ExpFloat64() * (20 + 110*risk[hi])
			closed = arcgis(created.Add(time.Duration(hrs * float64(time.Hour))))
		}
		w.Write([]string{fmt.Sprintf("24-%08d", i+1), t, []string{"Phone", "Web", "App"}[r.Intn(3)], arcgis(created), status, closed,
			agency[t[0]], fmt.Sprintf("%d SAMPLE ST", 100+r.Intn(4900)), hoods[hi], fmt.Sprint(1 + r.Intn(14))})
	}
	w.Flush()
	f.Close()
	fmt.Printf("Synthetic sample written to %s (%d crime rows, %d 311 rows)\n", *out, *nCrime, *nReq)
}
