package analysis

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
)

// ---- Q3: high crime, low 311 (possible under-reporting) ----

type Underreporting struct{}

func (Underreporting) Name() string {
	return "Q3: Neighborhoods with high crime but low 311 use (possible under-reporting)"
}
func (Underreporting) Stakeholder() string { return "Baltimore Police Department" }

func (Underreporting) Run(d *join.Dataset) report.Section {
	minC := minCrimes(d, 400)
	es := entries(d)
	// Ignore tiny neighborhoods (ratios on a handful of incidents are noise)
	// and neighborhoods with zero 311 rows (those are join-key gaps, not
	// evidence of under-reporting).
	var big []entry
	var unmatched []string
	for _, e := range es {
		if e.Stats.Crime < minC {
			continue
		}
		if e.Stats.Calls311 == 0 {
			unmatched = append(unmatched, e.Name)
			continue
		}
		big = append(big, e)
	}
	sec := report.Section{Title: Underreporting{}.Name(), Stakeholder: Underreporting{}.Stakeholder()}
	if len(big) == 0 {
		sec.Lines = append(sec.Lines, "Not enough data (no neighborhood has enough crimes and 311 matches).")
		return sec
	}
	ratio := make([]float64, len(big))
	var sum float64
	for i, e := range big {
		ratio[i] = float64(e.Stats.Crime) / float64(e.Stats.Calls311)
		sum += ratio[i]
	}
	n := float64(len(big))
	mean := sum / n
	var ss float64
	for _, r := range ratio {
		ss += (r - mean) * (r - mean)
	}
	sd := math.Sqrt(ss / n)

	type scored struct {
		e entry
		z float64
	}
	sc := make([]scored, len(big))
	for i, e := range big {
		z := 0.0
		if sd > 0 {
			z = (ratio[i] - mean) / sd
		}
		sc[i] = scored{e, z}
	}
	sort.SliceStable(sc, func(i, j int) bool { return sc[i].z > sc[j].z })
	if len(sc) > 10 {
		sc = sc[:10]
	}
	sec.Lines = append(sec.Lines,
		fmt.Sprintf("Metric: crimes per 311 request (%d neighborhoods with %d+ crimes and 311 matches; mean %.2f, sd %.2f).", len(big), minC, mean, sd),
		"A high z-score means unusually many crimes relative to 311 activity.", "")
	maxZ := 0.0
	for _, s := range sc {
		if s.z > maxZ {
			maxZ = s.z
		}
	}
	if maxZ <= 0 {
		maxZ = 1
	}
	for _, s := range sc {
		bars := int(math.Round(s.z / maxZ * 30))
		if bars < 0 {
			bars = 0
		}
		sec.Lines = append(sec.Lines, report.Bar(s.e.Name, bars, 30, 30)+
			fmt.Sprintf("   z=%.2f  crimes=%d  311=%d", s.z, s.e.Stats.Crime, s.e.Stats.Calls311))
	}
	if len(unmatched) > 0 {
		sec.Lines = append(sec.Lines, "",
			"Excluded (enough crimes but zero 311 rows; likely a neighborhood-name mismatch between datasets):",
			"  "+strings.Join(unmatched, ", "))
	}
	sec.Lines = append(sec.Lines, "", "Caveat: about 31% of 311 rows lack a neighborhood, so 311 counts are undercounted.")
	return sec
}

// ---- Q4: hour of day ----

type HourOfDay struct{}

func (HourOfDay) Name() string        { return "Q4: Crime vs. lighting and transportation requests by hour" }
func (HourOfDay) Stakeholder() string { return "Department of Transportation" }

func (HourOfDay) Run(d *join.Dataset) report.Section {
	maxC, maxL, maxT := 0, 0, 0
	peakC, peakL, peakT := 0, 0, 0
	for h := 0; h < 24; h++ {
		if d.CrimeByHour[h] > maxC {
			maxC, peakC = d.CrimeByHour[h], h
		}
		if d.LightingHour[h] > maxL {
			maxL, peakL = d.LightingHour[h], h
		}
		if d.TransportHour[h] > maxT {
			maxT, peakT = d.TransportHour[h], h
		}
	}
	sec := report.Section{Title: HourOfDay{}.Name(), Stakeholder: HourOfDay{}.Stakeholder()}
	sec.Lines = append(sec.Lines,
		fmt.Sprintf("Peak hours: crime %02d:00, lighting requests %02d:00, transportation requests %02d:00", peakC, peakL, peakT),
		"Each column is scaled to its own maximum.", "",
		fmt.Sprintf("%-6s %-22s %-22s %-22s", "Hour", "Crime", "Lighting", "Transport"))
	for h := 0; h < 24; h++ {
		sec.Lines = append(sec.Lines, fmt.Sprintf("%02d:00  %-22s %-22s %-22s", h,
			bar(d.CrimeByHour[h], maxC, 18), bar(d.LightingHour[h], maxL, 18), bar(d.TransportHour[h], maxT, 18)))
	}
	sec.Lines = append(sec.Lines, "", "Note: many records carry 00:00 or 12:00 default times, which can inflate those hours.")
	return sec
}

// bar draws a block bar scaled so that max fills `width` characters.
func bar(v, max, width int) string {
	n := 0
	if max > 0 {
		n = v * width / max
	}
	return strings.Repeat("█", n)
}

// ---- Q5: monthly trend and divergent neighborhoods ----

type MonthlyTrend struct{}

func (MonthlyTrend) Name() string {
	return "Q5: Monthly trends and neighborhoods that diverge from the city"
}
func (MonthlyTrend) Stakeholder() string { return "Department of Planning" }

func (MonthlyTrend) Run(d *join.Dataset) report.Section {
	minC := minCrimes(d, 146)
	var cityCrime, cityCalls [12]int
	for _, s := range d.ByHood {
		for m := 0; m < 12; m++ {
			cityCrime[m] += s.CrimeByMonth[m]
			cityCalls[m] += s.CallsByMonth[m]
		}
	}
	maxC, maxR := 0, 0
	for m := 0; m < 12; m++ {
		if cityCrime[m] > maxC {
			maxC = cityCrime[m]
		}
		if cityCalls[m] > maxR {
			maxR = cityCalls[m]
		}
	}
	names := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	sec := report.Section{Title: MonthlyTrend{}.Name(), Stakeholder: MonthlyTrend{}.Stakeholder()}
	sec.Lines = append(sec.Lines, fmt.Sprintf("%-5s %-22s %-22s", "Month", "Crime", "311 requests"))
	for m := 0; m < 12; m++ {
		sec.Lines = append(sec.Lines, fmt.Sprintf("%-5s %-22s %-22s  (%d / %d)", names[m],
			bar(cityCrime[m], maxC, 18), bar(cityCalls[m], maxR, 18), cityCrime[m], cityCalls[m]))
	}

	// Divergence: correlation of each neighborhood's monthly crime curve
	// with the citywide curve. Low correlation = different seasonal pattern.
	city := make([]float64, 12)
	for m := range city {
		city[m] = float64(cityCrime[m])
	}
	type div struct {
		e entry
		r float64
	}
	var ds []div
	for _, e := range entries(d) {
		if e.Stats.Crime < minC { // small neighborhoods give noisy monthly curves
			continue
		}
		v := make([]float64, 12)
		for m := range v {
			v[m] = float64(e.Stats.CrimeByMonth[m])
		}
		ds = append(ds, div{e, pearson(v, city)})
	}
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].r < ds[j].r })
	if len(ds) > 5 {
		ds = ds[:5]
	}
	sec.Lines = append(sec.Lines, "", "Neighborhoods whose monthly crime pattern differs most from the citywide pattern",
		fmt.Sprintf("(lowest correlation with the city curve; neighborhoods with %d+ crimes):", minC))
	for _, x := range ds {
		sec.Lines = append(sec.Lines, fmt.Sprintf("  %-30s r=%.2f  (%d crimes)", x.e.Name, x.r, x.e.Stats.Crime))
	}
	return sec
}
