// Package sample generates deterministic SYNTHETIC data in the exact column
// layouts of the four Open Baltimore datasets, so the application can be
// compiled, run and graded without network access. Neighborhood names and
// approximate centroids are real; every record is invented.
//
// The generator plants known relationships (vacancy drives sanitation
// complaints, some violence clusters around liquor outlets late at night,
// high-distress areas wait longer for 311 service) so the analyses have
// structure to find, and tests can check they find it.
package sample

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type hood struct {
	name       string
	lat, lon   float64
	distress   float64 // 0 (stable) .. 1 (most distressed)
	commercial float64 // 0 .. 1 nightlife / retail intensity
}

// Real Baltimore neighborhood names with approximate centroids.
var hoods = []hood{
	{"Downtown", 39.2904, -76.6122, 0.35, 1.0}, {"Inner Harbor", 39.2856, -76.6100, 0.15, 0.9},
	{"Fells Point", 39.2826, -76.5916, 0.15, 1.0}, {"Canton", 39.2808, -76.5747, 0.08, 0.8},
	{"Federal Hill", 39.2785, -76.6131, 0.10, 0.8}, {"Mount Vernon", 39.2976, -76.6155, 0.30, 0.7},
	{"Little Italy", 39.2862, -76.6000, 0.12, 0.6}, {"Highlandtown", 39.2870, -76.5683, 0.30, 0.7},
	{"Hampden", 39.3326, -76.6339, 0.10, 0.6}, {"Charles Village", 39.3241, -76.6156, 0.25, 0.4},
	{"Roland Park", 39.3528, -76.6370, 0.02, 0.1}, {"Locust Point", 39.2700, -76.5950, 0.05, 0.3},
	{"Riverside", 39.2740, -76.6040, 0.08, 0.2}, {"Butchers Hill", 39.2890, -76.5850, 0.15, 0.3},
	{"Brewers Hill", 39.2800, -76.5650, 0.10, 0.4}, {"Remington", 39.3220, -76.6260, 0.25, 0.3},
	{"Sandtown-Winchester", 39.3013, -76.6430, 0.95, 0.3}, {"Upton", 39.3004, -76.6302, 0.80, 0.3},
	{"Harlem Park", 39.2955, -76.6430, 0.85, 0.2}, {"Broadway East", 39.3060, -76.5880, 0.95, 0.3},
	{"Oliver", 39.3072, -76.5970, 0.85, 0.2}, {"Belair-Edison", 39.3195, -76.5730, 0.60, 0.4},
	{"Coldstream Homestead Montebello", 39.3200, -76.5900, 0.70, 0.2}, {"Waverly", 39.3280, -76.6060, 0.45, 0.4},
	{"Pimlico Good Neighbors", 39.3500, -76.6680, 0.75, 0.3}, {"Cherry Hill", 39.2510, -76.6250, 0.80, 0.2},
	{"Brooklyn", 39.2310, -76.6000, 0.65, 0.4}, {"Curtis Bay", 39.2240, -76.5850, 0.70, 0.3},
	{"Morrell Park", 39.2680, -76.6600, 0.35, 0.2}, {"Irvington", 39.2830, -76.6800, 0.50, 0.3},
	{"Edmondson Village", 39.2930, -76.6900, 0.55, 0.3}, {"Carroll-South Hilton", 39.2830, -76.6700, 0.70, 0.2},
	{"Midtown-Edmondson", 39.2980, -76.6600, 0.80, 0.3}, {"Franklin Square", 39.2930, -76.6400, 0.85, 0.2},
	{"Penn North", 39.3110, -76.6390, 0.80, 0.5}, {"Reservoir Hill", 39.3170, -76.6340, 0.55, 0.2},
	{"Greenmount West", 39.3080, -76.6080, 0.60, 0.2}, {"Barclay", 39.3130, -76.6110, 0.65, 0.2},
	{"Johnston Square", 39.3040, -76.6020, 0.85, 0.2}, {"McElderry Park", 39.2970, -76.5800, 0.80, 0.3},
	{"Ellwood Park/Monument", 39.2960, -76.5720, 0.65, 0.4}, {"Madison-Eastend", 39.3010, -76.5800, 0.80, 0.3},
	{"Frankford", 39.3320, -76.5550, 0.50, 0.3}, {"Hamilton Hills", 39.3600, -76.5550, 0.15, 0.2},
	{"Lauraville", 39.3420, -76.5700, 0.20, 0.3}, {"Govans", 39.3550, -76.6080, 0.40, 0.4},
	{"Druid Heights", 39.3050, -76.6320, 0.85, 0.2}, {"Westport", 39.2600, -76.6300, 0.65, 0.2},
	{"Washington Village/Pigtown", 39.2820, -76.6340, 0.55, 0.5}, {"Mondawmin", 39.3150, -76.6560, 0.60, 0.4},
}

var crimeTypes = []struct {
	code, desc string
	w          float64
	violent    bool
}{
	{"6D", "LARCENY FROM AUTO", 14, false}, {"6E", "LARCENY", 22, false}, {"7A", "AUTO THEFT", 10, false},
	{"5A", "BURGLARY", 9, false}, {"4E", "COMMON ASSAULT", 18, true}, {"4C", "AGG. ASSAULT", 11, true},
	{"3B", "ROBBERY - STREET", 6, true}, {"3K", "ROBBERY - COMMERCIAL", 1.5, true},
	{"3AF", "ROBBERY - CARJACKING", 1, true}, {"9S", "SHOOTING", 2, true}, {"1F", "HOMICIDE", 1, true},
	{"2A", "RAPE", 0.6, true}, {"8AO", "ARSON", 0.5, false},
}

type srKind struct {
	typ, agency string
	sla         float64 // due-date days
	medianDays  float64
	w           float64
	sanitation  bool
}

var srKinds = []srKind{
	{"SW-Dirty Alley", "Solid Waste", 7, 4, 10, true}, {"SW-Rat Rubout", "Solid Waste", 7, 5, 7, true},
	{"HCD-Illegal Dumping", "Solid Waste", 10, 6, 8, true}, {"SW-Cleaning", "Solid Waste", 7, 3, 6, true},
	{"SW-Bulk Scheduled", "Solid Waste", 14, 6, 6, true}, {"SW-Trash Can Missed Pickup", "Solid Waste", 2, 1.5, 7, true},
	{"HCD-Sanitation Property", "Housing & Community Dev", 14, 12, 5, true},
	{"HCD-Vacant Building", "Housing & Community Dev", 30, 25, 4, false},
	{"TRM-Pothole", "Transportation", 5, 4, 9, false}, {"TRM-StLight Out", "Transportation", 7, 6, 6, false},
	{"TRS-Parking Complaint", "Transportation", 2, 1, 8, false}, {"TRT-Traffic Signal Repair", "Transportation", 3, 2, 3, false},
	{"WW-Water Leak (Exterior)", "Water Wastewater", 5, 3, 6, false}, {"WW-Sewer Water In Basement", "Water Wastewater", 3, 2, 2, false},
	{"FCS-Tree Maintenance", "Forestry", 60, 40, 4, false}, {"ECC-Noise Complaint", "Environmental Control", 10, 7, 3, false},
}

var liquorKinds = []struct {
	kind, class string
	w           float64
	late        float64 // share of nearby planted violence that is late night
}{
	{"Tavern", "D", 30, 0.75}, {"Restaurant", "B", 35, 0.5}, {"Package Goods", "A", 25, 0.55},
	{"Hotel", "B", 4, 0.4}, {"Night Club", "BD7", 6, 0.85},
}

var typologies = []string{"Regional Choice", "Middle Market Choice", "Middle Market", "Middle Market Stressed", "Distressed"}

var streets = []string{"N Charles St", "E North Ave", "W Baltimore St", "Eastern Ave", "Harford Rd", "Greenmount Ave",
	"Edmondson Ave", "Pennsylvania Ave", "E Monument St", "W Franklin St", "Belair Rd", "Park Heights Ave",
	"S Hanover St", "Washington Blvd", "Frederick Ave", "N Milton Ave", "E Fayette St", "Druid Hill Ave"}

var names1 = []string{"Corner", "Harbor", "Charm City", "Old Line", "Chesapeake", "Oriole", "Rowhouse", "Patapsco",
	"Domino", "Lexington", "Natty", "Crab", "Inner", "Station", "Mount", "Waverly", "Highland", "Union"}
var names2 = []string{"Tavern", "Liquors", "Bar & Grill", "Pub", "Wine & Spirits", "Lounge", "Saloon", "Package Store", "Cafe", "Inn"}

// Options sets the generated volume.
type Options struct {
	Dir   string
	Year  int
	Seed  int64
	Scale float64 // 1.0 ≈ 45k crimes, 120k 311 requests, 1.1k outlets, 12k vacants
}

type gen struct {
	r   *rand.Rand
	opt Options
}

// Generate writes the four CSVs plus SAMPLE_DATA.txt into opt.Dir.
func Generate(opt Options) (map[string]int, error) {
	if opt.Scale <= 0 {
		opt.Scale = 1
	}
	if err := os.MkdirAll(opt.Dir, 0o755); err != nil {
		return nil, err
	}
	g := &gen{r: rand.New(rand.NewSource(opt.Seed)), opt: opt}
	counts := map[string]int{}
	outlets, n, err := g.liquor()
	if err != nil {
		return nil, err
	}
	counts["liquor_licenses.csv"] = n
	vac, err := g.vacants()
	if err != nil {
		return nil, err
	}
	counts["vacant_building_notices.csv"] = sum(vac)
	if counts["part1_crime.csv"], err = g.crime(outlets); err != nil {
		return nil, err
	}
	if counts["service_requests_311.csv"], err = g.requests(vac); err != nil {
		return nil, err
	}
	note := fmt.Sprintf("SYNTHETIC sample data generated %s (seed %d, scale %.2f, year %d).\n"+
		"Columns match the Open Baltimore exports; records are invented. Delete this file's folder and run `make fetch` for real data.\n",
		time.Now().Format(time.RFC3339), opt.Seed, opt.Scale, opt.Year)
	return counts, os.WriteFile(filepath.Join(opt.Dir, "SAMPLE_DATA.txt"), []byte(note), 0o644)
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func (g *gen) scaled(x float64) int { return int(math.Round(x * g.opt.Scale)) }

// jitter scatters a point around a centroid with ~sd meters spread.
func (g *gen) jitter(lat, lon, sdM float64) (float64, float64) {
	dLat := g.r.NormFloat64() * sdM / 111320
	dLon := g.r.NormFloat64() * sdM / (111320 * math.Cos(lat*math.Pi/180))
	return lat + dLat, lon + dLon
}

func (g *gen) pick(weights []float64) int {
	t := 0.0
	for _, w := range weights {
		t += w
	}
	x := g.r.Float64() * t
	for i, w := range weights {
		if x -= w; x < 0 {
			return i
		}
	}
	return len(weights) - 1
}

func (g *gen) when(lateBias float64) time.Time {
	loc, _ := time.LoadLocation("America/New_York")
	start := time.Date(g.opt.Year, 1, 1, 0, 0, 0, 0, loc)
	day := start.AddDate(0, 0, g.r.Intn(365))
	var t time.Time
	if g.r.Float64() < lateBias {
		if g.r.Float64() < 0.5 { // half of late-night violence lands on Friday or Saturday night
			for day.Weekday() != time.Friday {
				day = day.AddDate(0, 0, 1)
			}
			day = day.AddDate(0, 0, g.r.Intn(2))
		}
		h := []int{21, 22, 23, 24, 25, 26}[g.r.Intn(6)] // 24-26 = after midnight, next calendar day
		t = day.Add(time.Duration(h)*time.Hour + time.Duration(g.r.Intn(60))*time.Minute)
	} else {
		// daytime-weighted distribution peaking late afternoon / evening
		hw := []float64{3, 2.2, 1.6, 1.1, .8, .7, 1, 1.6, 2.6, 3.1, 3.6, 4.1, 4.6, 4.6, 5.1, 5.6, 6, 6.2, 6.1, 5.8, 5.3, 4.8, 4.2, 3.6}
		t = day.Add(time.Duration(g.pick(hw))*time.Hour + time.Duration(g.r.Intn(60))*time.Minute)
	}
	if t.Year() != g.opt.Year {
		t = t.AddDate(0, 0, -7)
	}
	if g.r.Float64() < 0.03 { // reproduce the real data's "00:00 = unknown time" artifact
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}
	return t
}

func stamp(t time.Time) string { return t.UTC().Format("2006/01/02 15:04:05+00") }

func create(dir, name string, header []string) (*os.File, *csv.Writer, error) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return nil, nil, err
	}
	w := csv.NewWriter(f)
	return f, w, w.Write(header)
}

func finish(f *os.File, w *csv.Writer) error {
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

type outlet struct {
	hood     int
	lat, lon float64
	late     float64
}

func (g *gen) liquor() ([]outlet, int, error) {
	f, w, err := create(g.opt.Dir, "liquor_licenses.csv", []string{"X", "Y", "LLKey", "CorpKey", "LicenseClass", "SubClass",
		"LicenseNumber", "LicenseDate", "LicenseEndDate", "LicenseYear", "LicenseFee", "CertificateNumber", "LicenseStatus",
		"LicenseeFirstName", "LicenseeLastName", "TradeName", "CorpName", "EstablishmentDesc", "DayPerWeek", "Description",
		"AddrStreet", "AddrZip", "ESRI_OID"})
	if err != nil {
		return nil, 0, err
	}
	kw := make([]float64, len(liquorKinds))
	for i, k := range liquorKinds {
		kw[i] = k.w
	}
	var outs []outlet
	rows, key := 0, 1
	for hi, h := range hoods {
		n := g.scaled(2 + 30*h.commercial + 8*h.distress)
		for k := 0; k < n; k++ {
			lk := liquorKinds[g.pick(kw)]
			lat, lon := g.jitter(h.lat, h.lon, 260)
			outs = append(outs, outlet{hi, lat, lon, lk.late})
			num := fmt.Sprintf("L%05d", key)
			name := names1[g.r.Intn(len(names1))] + " " + names2[g.r.Intn(len(names2))]
			addr := fmt.Sprintf("%d %s", 100+g.r.Intn(3800), streets[(hi*5+g.r.Intn(2)*7)%len(streets)])
			// one row per license year, like the real table
			for y := g.opt.Year - 1; y <= g.opt.Year; y++ {
				status := "Approved"
				_ = w.Write([]string{fmt.Sprintf("%.6f", lon), fmt.Sprintf("%.6f", lat), fmt.Sprint(key*10 + y%10),
					fmt.Sprint(5000 + key), lk.class, "", num,
					stamp(time.Date(y, 5, 1, 12, 0, 0, 0, time.UTC)), stamp(time.Date(y+1, 4, 30, 12, 0, 0, 0, time.UTC)),
					fmt.Sprint(y), fmt.Sprintf("%.2f", 500+g.r.Float64()*2500), fmt.Sprintf("C%06d", key*3+y%10), status,
					"", "", name, strings.ToUpper(name) + " LLC", lk.kind, "7", lk.kind + " license", addr,
					fmt.Sprintf("212%02d", 1+g.r.Intn(30)), fmt.Sprint(rows + 1)})
				rows++
			}
			key++
		}
	}
	return outs, len(outs), finish(f, w)
}

func (g *gen) vacants() ([]int, error) {
	f, w, err := create(g.opt.Dir, "vacant_building_notices.csv", []string{"X", "Y", "NoticeNum", "DateNotice", "DateCancel",
		"DateAbate", "NT", "OWNER_ABBR", "Council_District", "Neighborhood", "BLOCKLOT", "Address", "OBJECTID", "HousingMarketTypology2023"})
	if err != nil {
		return nil, err
	}
	per := make([]int, len(hoods))
	id := 1
	for hi, h := range hoods {
		n := g.scaled(4 + 900*math.Pow(h.distress, 2.4)*(0.8+0.4*g.r.Float64()))
		typ := typologies[min(len(typologies)-1, int(h.distress*float64(len(typologies))))]
		for k := 0; k < n; k++ {
			lat, lon := g.jitter(h.lat, h.lon, 350)
			cancel := ""
			if g.r.Float64() < 0.06 { // some notices already closed; parser must drop them
				cancel = stamp(time.Date(g.opt.Year, time.Month(1+g.r.Intn(12)), 1+g.r.Intn(28), 12, 0, 0, 0, time.UTC))
			} else {
				per[hi]++
			}
			_ = w.Write([]string{fmt.Sprintf("%.6f", lon), fmt.Sprintf("%.6f", lat), fmt.Sprintf("VBN%07d", id),
				stamp(time.Date(g.opt.Year-g.r.Intn(8), time.Month(1+g.r.Intn(12)), 1+g.r.Intn(28), 12, 0, 0, 0, time.UTC)),
				cancel, "", "VBN", "OWNER", fmt.Sprint(1 + hi%14), strings.ToUpper(h.name),
				fmt.Sprintf("%04d%03d", 1000+hi*37+k%37, k%500), fmt.Sprintf("%d %s", 100+g.r.Intn(3000), streets[g.r.Intn(len(streets))]),
				fmt.Sprint(id), typ})
			id++
		}
	}
	return per, finish(f, w)
}

func (g *gen) crime(outs []outlet) (int, error) {
	f, w, err := create(g.opt.Dir, "part1_crime.csv", []string{"RowID", "CCNumber", "CrimeDateTime", "CrimeCode", "Description",
		"Inside_Outside", "Weapon", "Post", "Gender", "Age", "Race", "Ethnicity", "Location", "Old_District", "New_District",
		"Neighborhood", "Latitude", "Longitude", "GeoLocation", "PremiseType", "Total_Incidents"})
	if err != nil {
		return 0, err
	}
	byHood := map[int][]outlet{}
	for _, o := range outs {
		byHood[o.hood] = append(byHood[o.hood], o)
	}
	cw := make([]float64, len(crimeTypes))
	for i, c := range crimeTypes {
		cw[i] = c.w
	}
	vw := make([]float64, 0)
	var violentIdx []int
	for i, c := range crimeTypes {
		if c.violent {
			violentIdx = append(violentIdx, i)
			vw = append(vw, c.w)
		}
	}
	districts := []string{"CENTRAL", "EASTERN", "NORTHEASTERN", "NORTHERN", "NORTHWESTERN", "SOUTHEASTERN", "SOUTHERN", "SOUTHWESTERN", "WESTERN"}
	id := 0
	for hi, h := range hoods {
		n := g.scaled((60 + 700*(0.25+h.distress*1.3+0.45*h.commercial)) * (0.8 + 0.4*g.r.Float64()))
		for k := 0; k < n; k++ {
			ct := crimeTypes[g.pick(cw)]
			lat, lon := g.jitter(h.lat, h.lon, 520)
			late := 0.10
			// Plant outlet-adjacent violence: some violent incidents are moved
			// next to a liquor outlet in the same neighborhood, mostly at night.
			if near := byHood[hi]; len(near) > 0 && g.r.Float64() < 0.06+0.10*h.commercial {
				o := near[g.r.Intn(len(near))]
				ct = crimeTypes[violentIdx[g.pick(vw)]]
				lat, lon = g.jitter(o.lat, o.lon, 60)
				late = o.late * 0.6
			}
			t := g.when(late)
			id++
			geo := fmt.Sprintf("(%.6f, %.6f)", lat, lon)
			slat, slon := fmt.Sprintf("%.6f", lat), fmt.Sprintf("%.6f", lon)
			if g.r.Float64() < 0.01 { // ~1% missing coordinates, as in the real data
				slat, slon, geo = "", "", ""
			}
			hoodName := h.name
			if g.r.Float64() < 0.005 {
				hoodName = ""
			}
			io := []string{"I", "O"}[g.r.Intn(2)]
			_ = w.Write([]string{fmt.Sprint(id), fmt.Sprintf("%02dA%06d", g.opt.Year%100, id), stamp(t), ct.code, ct.desc,
				io, "", fmt.Sprint(100 + g.r.Intn(900)), "", "", "", "",
				fmt.Sprintf("%d %s", 100+g.r.Intn(3800), strings.ToUpper(streets[g.r.Intn(len(streets))])),
				districts[hi%len(districts)], districts[hi%len(districts)], hoodName, slat, slon, geo, "STREET", "1"})
		}
	}
	return id, finish(f, w)
}

func (g *gen) requests(vac []int) (int, error) {
	f, w, err := create(g.opt.Dir, "service_requests_311.csv", []string{"RowID", "SRRecordID", "ServiceRequestNum", "SRType",
		"MethodReceived", "CreatedDate", "SRStatus", "StatusDate", "DueDate", "CloseDate", "Agency", "LastActivity",
		"LastActivityDate", "Outcome", "Address", "ZipCode", "Neighborhood", "CouncilDistrict", "PoliceDistrict",
		"PolicePost", "Latitude", "Longitude", "GeoLocation"})
	if err != nil {
		return 0, err
	}
	var san, other []int
	var sw, ow []float64
	for i, k := range srKinds {
		if k.sanitation {
			san, sw = append(san, i), append(sw, k.w)
		} else {
			other, ow = append(other, i), append(ow, k.w)
		}
	}
	loc, _ := time.LoadLocation("America/New_York")
	methods := []string{"Phone", "Web", "App", "Email"}
	id := 0
	for hi, h := range hoods {
		// Sanitation demand scales with open vacants (+ noise); other demand with size/activity.
		nSan := max(0, int(float64(g.scaled(120)+int(2.4*float64(vac[hi])))*(0.65+0.7*g.r.Float64()))+g.scaled(400*h.commercial*g.r.Float64()))
		nOther := g.scaled(1400 + 1500*h.commercial + 600*h.distress)
		for k := 0; k < nSan+nOther; k++ {
			var kind srKind
			if k < nSan {
				kind = srKinds[san[g.pick(sw)]]
			} else {
				kind = srKinds[other[g.pick(ow)]]
			}
			created := time.Date(g.opt.Year, 1, 1, 7+g.r.Intn(12), g.r.Intn(60), 0, 0, loc).AddDate(0, 0, g.r.Intn(365))
			due := created.Add(time.Duration(kind.sla * 24 * float64(time.Hour)))
			// lognormal close times; distressed areas wait longer
			med := kind.medianDays * (0.85 + 0.75*h.distress)
			days := med * math.Exp(g.r.NormFloat64()*0.8)
			closed := created.Add(time.Duration(days * 24 * float64(time.Hour)))
			status, closeS := "Closed", stamp(closed)
			if g.r.Float64() < 0.05+0.06*h.distress || closed.Year() > g.opt.Year {
				status, closeS = "Open", ""
			}
			lat, lon := g.jitter(h.lat, h.lon, 520)
			id++
			_ = w.Write([]string{fmt.Sprint(id), fmt.Sprintf("SR%09d", id), fmt.Sprintf("%02d-%08d", g.opt.Year%100, id),
				kind.typ, methods[g.r.Intn(len(methods))], stamp(created), status, stamp(created), stamp(due), closeS,
				kind.agency, "", "", "", fmt.Sprintf("%d %s", 100+g.r.Intn(3800), streets[g.r.Intn(len(streets))]),
				fmt.Sprintf("212%02d", 1+g.r.Intn(30)), h.name, fmt.Sprint(1 + hi%14), "", "",
				fmt.Sprintf("%.6f", lat), fmt.Sprintf("%.6f", lon), fmt.Sprintf("(%.6f, %.6f)", lat, lon)})
		}
	}
	return id, finish(f, w)
}
