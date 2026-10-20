package main

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// Question is the STRATEGY for the analysis stage: each research question is
// an independent, read-only function over the two finished aggregates, so they
// can be added, removed or run concurrently without touching ingestion.
type Question interface {
	Title() string
	Answer(w io.Writer, crime, svc *Agg)
}

func AllQuestions(top int) []Question {
	return []Question{
		hotspots{top}, serviceGap{top}, correlation{}, requestLift{top}, timing{},
	}
}

// ---- Q1 (Mayor's Office) ---------------------------------------------------
type hotspots struct{ n int }

func (hotspots) Title() string {
	return "Mayor's Office: Which neighborhoods carry the most crime, and what does their 311 picture look like?"
}
func (q hotspots) Answer(w io.Writer, c, s *Agg) {
	rows := topN(c.ByNbhd, q.n)
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(w, "%-26s %7s %6s %7s %6s  %s\n", "Neighborhood", "Crime", "Viol%", "311", "Open%", "Crime volume")
	for _, r := range rows {
		fmt.Fprintf(w, "%-26s %7d %5.1f%% %7d %5.1f%%  %s\n", trunc(r.K, 26), r.V,
			pct(c.ViolentByNbhd[r.K], r.V), s.ByNbhd[r.K], pct(s.OpenByNbhd[r.K], s.ByNbhd[r.K]),
			bar(float64(r.V), float64(rows[0].V), 30))
	}
	fmt.Fprintf(w, "\nTakeaway: %s has the most incidents (%d). 'Open%%' is the share of that area's 311 requests still unresolved.\n", rows[0].K, rows[0].V)
}

// ---- Q2 (Dept. of Public Works) -------------------------------------------
type serviceGap struct{ n int }

func (serviceGap) Title() string {
	return "Public Works: Where is crime high but 311 engagement low (possible service gap / under-reporting)?"
}
func (q serviceGap) Answer(w io.Writer, c, s *Agg) {
	if len(c.ByNbhd) == 0 {
		return
	}
	both := 0
	bothCrime := 0
	for k, v := range c.ByNbhd {
		if _, ok := s.ByNbhd[k]; ok {
			both++
			bothCrime += v
		}
	}
	if both == 0 {
		return
	}
	avg := bothCrime / both
	type row struct {
		k          string
		crime, svc int
		ratio      float64
	}
	var rows []row
	for k, v := range c.ByNbhd {
		d, ok := s.ByNbhd[k]
		if !ok || v < avg { // true join only, and above-average crime
			continue
		}
		rows = append(rows, row{k, v, d, float64(v) / float64(d)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ratio != rows[j].ratio {
			return rows[i].ratio > rows[j].ratio
		}
		return rows[i].k < rows[j].k
	})
	if len(rows) > q.n {
		rows = rows[:q.n]
	}
	fmt.Fprintf(w, "(Only neighborhoods present in BOTH datasets with at least the average of %d crimes are considered.)\n\n", avg)
	fmt.Fprintf(w, "%-26s %7s %7s %11s  %s\n", "Neighborhood", "Crime", "311", "Crime/311", "Ratio")
	for _, r := range rows {
		fmt.Fprintf(w, "%-26s %7d %7d %11.2f  %s\n", trunc(r.k, 26), r.crime, r.svc, r.ratio, bar(r.ratio, rows[0].ratio, 30))
	}
	if len(rows) > 0 {
		fmt.Fprintf(w, "\nTakeaway: %s has %.1f crimes per 311 request; consider outreach to boost 311 awareness there.\n", rows[0].k, rows[0].ratio)
	}
}

// ---- Q3 (City Planning) ----------------------------------------------------
type correlation struct{}

func (correlation) Title() string {
	return "City Planning: Do neighborhoods with more crime also generate more 311 requests?"
}
func (correlation) Answer(w io.Writer, c, s *Agg) {
	type pt struct {
		k      string
		cr, sv float64
	}
	var pts []pt
	for k, v := range c.ByNbhd {
		if sv, ok := s.ByNbhd[k]; ok {
			pts = append(pts, pt{k, float64(v), float64(sv)})
		}
	}
	if len(pts) < 4 {
		fmt.Fprintln(w, "Not enough joined neighborhoods.")
		return
	}
	var mx, my float64
	for _, p := range pts {
		mx += p.cr
		my += p.sv
	}
	mx /= float64(len(pts))
	my /= float64(len(pts))
	var sxy, sxx, syy float64
	for _, p := range pts {
		sxy += (p.cr - mx) * (p.sv - my)
		sxx += (p.cr - mx) * (p.cr - mx)
		syy += (p.sv - my) * (p.sv - my)
	}
	r := 0.0
	if sxx > 0 && syy > 0 {
		r = sxy / math.Sqrt(sxx*syy)
	}
	strength := "weak"
	if math.Abs(r) >= 0.6 {
		strength = "strong"
	} else if math.Abs(r) >= 0.3 {
		strength = "moderate"
	}
	dir := "positive"
	if r < 0 {
		dir = "negative"
	}
	fmt.Fprintf(w, "Pearson r across %d neighborhoods = %.3f (%s %s relationship)\n\n", len(pts), r, strength, dir)

	sort.Slice(pts, func(i, j int) bool { return pts[i].cr < pts[j].cr })
	fmt.Fprintf(w, "%-22s %10s %10s\n", "Crime quartile", "Avg crime", "Avg 311")
	var maxSv float64
	type qrow struct {
		name   string
		cr, sv float64
	}
	var qs []qrow
	names := []string{"Q1 (lowest crime)", "Q2", "Q3", "Q4 (highest crime)"}
	for i := 0; i < 4; i++ {
		lo, hi := i*len(pts)/4, (i+1)*len(pts)/4
		var a, b float64
		for _, p := range pts[lo:hi] {
			a += p.cr
			b += p.sv
		}
		n := float64(hi - lo)
		qs = append(qs, qrow{names[i], a / n, b / n})
		maxSv = math.Max(maxSv, b/n)
	}
	for _, q := range qs {
		fmt.Fprintf(w, "%-22s %10.1f %10.1f  %s\n", q.name, q.cr, q.sv, bar(q.sv, maxSv, 30))
	}
}

// ---- Q4 (Dept. of Transportation / DPW) -----------------------------------
type requestLift struct{ n int }

func (requestLift) Title() string {
	return "Transportation/DPW: Which 311 request types are over-represented in the highest-crime neighborhoods?"
}
func (q requestLift) Answer(w io.Writer, c, s *Agg) {
	hot := map[string]int{}
	hotTotal := 0
	for _, r := range topN(c.ByNbhd, q.n) {
		for cat, v := range s.NbhdCat[r.K] {
			hot[cat] += v
			hotTotal += v
		}
	}
	if hotTotal == 0 || s.Total == 0 {
		fmt.Fprintln(w, "No 311 requests matched the top crime neighborhoods.")
		return
	}
	type row struct {
		cat       string
		n         int
		hs, cs, l float64
	}
	var rows []row
	for cat, v := range hot {
		// ignore tiny/noisy types and finance/administrative ones (FCPF-, FCCS-: parking fines, tax, refunds)
		if v < 100 || s.ByCat[cat] < 1000 || strings.HasPrefix(cat, "FC") {
			continue
		}
		hs := float64(v) / float64(hotTotal)
		cs := float64(s.ByCat[cat]) / float64(s.Total)
		rows = append(rows, row{cat, v, hs, cs, hs / cs})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].l > rows[j].l })
	if len(rows) > q.n {
		rows = rows[:q.n]
	}
	fmt.Fprintf(w, "Compared with the whole city, in the top %d crime neighborhoods ('Lift' = hot share / city share):\n\n", q.n)
	fmt.Fprintf(w, "%-30s %6s %8s %8s %6s  %s\n", "311 request type", "Count", "Hot %", "City %", "Lift", "")
	for _, r := range rows {
		fmt.Fprintf(w, "%-30s %6d %7.1f%% %7.1f%% %5.2fx  %s\n", trunc(r.cat, 30), r.n, 100*r.hs, 100*r.cs, r.l, bar(r.l, rows[0].l, 20))
	}
	if len(rows) > 0 {
		fmt.Fprintf(w, "\nTakeaway: '%s' requests are %.2fx more common in high-crime areas than citywide.\n", rows[0].cat, rows[0].l)
	}
}

// ---- Q5 (Mayor's Office / Operations) --------------------------------------
type timing struct{}

func (timing) Title() string {
	return "Operations: When do crime and 311 demand peak (by month and time of day)?"
}
func (timing) Answer(w io.Writer, c, s *Agg) {
	// The two datasets may cover different calendar years, so compare
	// seasonality (month of year) rather than exact months.
	var cm, sm [12]int
	fold := func(src map[string]int, dst *[12]int) {
		for k, v := range src {
			if len(k) == 7 {
				var mo int
				if _, err := fmt.Sscanf(k[5:], "%d", &mo); err == nil && mo >= 1 && mo <= 12 {
					dst[mo-1] += v
				}
			}
		}
	}
	fold(c.ByMonth, &cm)
	fold(s.ByMonth, &sm)
	var maxC, maxS int
	for i := 0; i < 12; i++ {
		maxC, maxS = max(maxC, cm[i]), max(maxS, sm[i])
	}
	names := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	fmt.Fprintf(w, "Seasonality by month of year (each series scaled to its own max)\n")
	fmt.Fprintf(w, "%-6s %7s %-22s %7s %s\n", "Month", "Crime", "", "311", "")
	for i := 0; i < 12; i++ {
		fmt.Fprintf(w, "%-6s %7d %-22s %7d %s\n", names[i], cm[i], bar(float64(cm[i]), float64(maxC), 22), sm[i], bar(float64(sm[i]), float64(maxS), 22))
	}
	blocks := []struct {
		name   string
		lo, hi int
	}{{"Overnight 00-05", 0, 5}, {"Morning   06-11", 6, 11}, {"Afternoon 12-17", 12, 17}, {"Evening   18-23", 18, 23}}
	sum := func(a *Agg, lo, hi int) int {
		t := 0
		for h := lo; h <= hi; h++ {
			t += a.ByHour[h]
		}
		return t
	}
	var tc, ts int
	for h := 0; h < 24; h++ {
		tc += c.ByHour[h]
		ts += s.ByHour[h]
	}
	fmt.Fprintf(w, "\nTime of day (share of rows that carry a timestamp)\n")
	fmt.Fprintf(w, "%-18s %7s %-22s %7s\n", "Block", "Crime%", "", "311%")
	for _, b := range blocks {
		cp, sp := pct(sum(c, b.lo, b.hi), tc), pct(sum(s, b.lo, b.hi), ts)
		fmt.Fprintf(w, "%-18s %6.1f%% %-22s %6.1f%% %s\n", b.name, cp, bar(cp, 50, 22), sp, bar(sp, 50, 22))
	}
	if tc == 0 && ts == 0 {
		fmt.Fprintln(w, "(No time-of-day information in these files.)")
	}
}
