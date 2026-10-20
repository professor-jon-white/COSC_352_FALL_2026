package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Question is the Strategy interface for one research question. Run receives
// the joined data as READ-ONLY input (the questions run concurrently) and must
// build everything it needs in local variables: it must not sort or otherwise
// modify j.Rows or anything reachable from it.
type Question interface {
	ID() string
	Stakeholder() string
	Title() string
	Ask() string
	Run(j *Joined) Section
}

// questionMeta carries the descriptive fields shared by every question.
type questionMeta struct {
	id, stakeholder, title, ask string
}

func (m questionMeta) ID() string          { return m.id }
func (m questionMeta) Stakeholder() string { return m.stakeholder }
func (m questionMeta) Title() string       { return m.title }
func (m questionMeta) Ask() string         { return m.ask }

func (m questionMeta) newSection() Section {
	return Section{ID: m.id, Stakeholder: m.stakeholder, Title: m.title, Question: m.ask}
}

// allQuestions returns the five questions in presentation order.
func allQuestions() []Question {
	return []Question{
		hotspotQuestion{questionMeta: questionMeta{
			id:          "Q1",
			stakeholder: "Mayor's Office",
			title:       "Where do crime and 311 demand coincide?",
			ask: "Which neighborhoods rank highest on BOTH crime and 311 service demand, " +
				"and how closely do the two track each other city-wide?",
		}},
		liftQuestion{questionMeta: questionMeta{
			id:          "Q2",
			stakeholder: "Department of Public Works",
			title:       "Which service requests signal high-crime areas?",
			ask: "Which 311 request types are over-represented in the highest-crime " +
				"neighborhoods compared with the rest of the city?",
		}},
		hourlyQuestion{questionMeta: questionMeta{
			id:          "Q3",
			stakeholder: "Police Department and Transportation (staffing)",
			title:       "When do incidents and service requests peak?",
			ask: "How do the hour-of-day profiles of crime and 311 requests compare, " +
				"and where do they diverge?",
		}},
		equityQuestion{questionMeta: questionMeta{
			id:          "Q4",
			stakeholder: "Mayor's Office (service equity)",
			title:       "Do high-crime neighborhoods wait longer for service?",
			ask: "Is the time to close a 311 request longer in the highest-crime " +
				"neighborhoods than in the lowest-crime ones?",
		}},
		disorderQuestion{questionMeta: questionMeta{
			id:          "Q5",
			stakeholder: "City Planning",
			title:       "Does physical disorder go together with violent crime?",
			ask: "Do neighborhoods with more physical-disorder 311 requests (trash, vacancy, " +
				"lighting, graffiti, ...) also have more violent crime?",
		}},
	}
}

// ---------------------------------------------------------------------------
// Q1: hotspots and city-wide correlation
// ---------------------------------------------------------------------------

type hotspotQuestion struct{ questionMeta }

func (q hotspotQuestion) Run(j *Joined) Section {
	s := q.newSection()
	n := len(j.Rows)
	if n < 3 {
		s.Findings = []string{"Fewer than 3 neighborhoods appear in both datasets; correlation is not meaningful."}
		return s
	}
	crime := make([]float64, n)
	reqs := make([]float64, n)
	for i, r := range j.Rows {
		crime[i] = float64(r.Crime.Total)
		reqs[i] = float64(r.Req.Total)
	}
	pr, sr := pearson(crime, reqs), spearman(crime, reqs)
	cr, rr := ranks(crime), ranks(reqs)

	type hot struct {
		row       *JoinedRow
		crimeRank int
		reqRank   int
		score     float64
	}
	hots := make([]hot, n)
	for i, r := range j.Rows {
		cRank := n + 1 - int(math.Round(cr[i])) // 1 = highest
		rRank := n + 1 - int(math.Round(rr[i]))
		hots[i] = hot{row: r, crimeRank: cRank, reqRank: rRank, score: float64(cRank+rRank) / 2}
	}
	sort.Slice(hots, func(a, b int) bool {
		if hots[a].score != hots[b].score {
			return hots[a].score < hots[b].score
		}
		return hots[a].row.Crime.Total > hots[b].row.Crime.Total
	})

	limit := 10
	if n < limit {
		limit = n
	}
	rows := make([][]string, 0, limit)
	labels := make([]string, 0, limit)
	cVals := make([]float64, 0, limit)
	rVals := make([]float64, 0, limit)
	for i := 0; i < limit; i++ {
		h := hots[i]
		name := truncate(h.row.Name, 28)
		rows = append(rows, []string{
			fmt.Sprint(i + 1), name,
			fmt.Sprint(h.row.Crime.Total), fmt.Sprint(h.crimeRank),
			fmt.Sprint(h.row.Req.Total), fmt.Sprint(h.reqRank),
		})
		labels = append(labels, name)
		cVals = append(cVals, float64(h.row.Crime.Total))
		rVals = append(rVals, float64(h.row.Req.Total))
	}

	var sb strings.Builder
	sb.WriteString(" Top neighborhoods by combined rank (average of crime rank and 311 rank; 1 = highest):\n\n")
	writeTable(&sb, []string{"#", "Neighborhood", "Crimes", "Crime rank", "311 reqs", "311 rank"}, rows)
	sb.WriteString("\n Crime incidents in those neighborhoods:\n")
	writeBarChart(&sb, labels, cVals, "", 40)
	sb.WriteString("\n 311 requests in those neighborhoods:\n")
	writeBarChart(&sb, labels, rVals, "", 40)
	s.Body = sb.String()

	both, bothCrime, bothReq, totCrime, totReq := 0, 0, 0, 0, 0
	for _, r := range j.Rows {
		totCrime += r.Crime.Total
		totReq += r.Req.Total
		if r.CrimeQuartile == 4 && r.RequestQuartile == 4 {
			both++
			bothCrime += r.Crime.Total
			bothReq += r.Req.Total
		}
	}
	s.Findings = []string{
		fmt.Sprintf("Across %d neighborhoods present in both datasets, crime and 311 volume correlate at Pearson r = %s and Spearman rho = %s (%s).",
			n, fmtFloat(pr, 2), fmtFloat(sr, 2), describeCorr(sr)),
		fmt.Sprintf("%d neighborhoods are in the top quartile for BOTH crime and 311 demand; together they hold %s of joined crime and %s of joined 311 requests.",
			both, pct(bothCrime, totCrime), pct(bothReq, totReq)),
		fmt.Sprintf("Highest combined rank: %s (crime rank %d, 311 rank %d).",
			hots[0].row.Name, hots[0].crimeRank, hots[0].reqRank),
	}
	return s
}

// ---------------------------------------------------------------------------
// Q2: request-type lift in high-crime neighborhoods
// ---------------------------------------------------------------------------

type liftQuestion struct{ questionMeta }

func (q liftQuestion) Run(j *Joined) Section {
	s := q.newSection()
	high := make(map[string]int)
	rest := make(map[string]int)
	highTotal, restTotal := 0, 0
	for _, r := range j.Rows {
		for t, c := range r.Req.ByType {
			if r.CrimeQuartile == 4 {
				high[t] += c
				highTotal += c
			} else {
				rest[t] += c
				restTotal += c
			}
		}
	}
	if highTotal == 0 || restTotal == 0 {
		s.Findings = []string{"Not enough neighborhoods to compare the highest-crime quartile with the rest of the city."}
		return s
	}
	minSupport := int(math.Max(5, float64(highTotal+restTotal)*0.001))

	type lift struct {
		typ              string
		hi, lo           int
		hiShare, loShare float64
		ratio            float64
	}
	var ls []lift
	for t, h := range high {
		l := rest[t]
		if h+l < minSupport || l == 0 {
			continue
		}
		hs := float64(h) / float64(highTotal)
		lsr := float64(l) / float64(restTotal)
		ls = append(ls, lift{typ: t, hi: h, lo: l, hiShare: hs, loShare: lsr, ratio: hs / lsr})
	}
	if len(ls) == 0 {
		s.Findings = []string{"No request type has enough volume in both groups to be compared."}
		return s
	}
	sort.Slice(ls, func(a, b int) bool {
		if ls[a].ratio != ls[b].ratio {
			return ls[a].ratio > ls[b].ratio
		}
		return ls[a].typ < ls[b].typ
	})

	top := 8
	if len(ls) < top {
		top = len(ls)
	}
	rows := make([][]string, 0, top)
	labels := make([]string, 0, top)
	vals := make([]float64, 0, top)
	for _, l := range ls[:top] {
		name := truncate(l.typ, 38)
		rows = append(rows, []string{
			name,
			fmt.Sprint(l.hi),
			fmt.Sprintf("%.1f%%", 100*l.hiShare),
			fmt.Sprintf("%.1f%%", 100*l.loShare),
			fmt.Sprintf("%.2fx", l.ratio),
		})
		labels = append(labels, name)
		vals = append(vals, l.ratio)
	}
	var sb strings.Builder
	sb.WriteString(" Most over-represented request types in the highest-crime quartile\n")
	sb.WriteString(" (lift = share of high-crime requests / share of all other requests):\n\n")
	writeTable(&sb, []string{"Request type", "High-crime n", "High-crime share", "Other share", "Lift"}, rows)
	sb.WriteString("\n Lift (1.00x = same share as the rest of the city):\n")
	writeBarChart(&sb, labels, vals, "x", 40)
	s.Body = sb.String()

	last := ls[len(ls)-1]
	s.Findings = []string{
		fmt.Sprintf("Compared %d requests in the top-crime quartile with %d in all other neighborhoods; types need at least %d requests in total to be ranked.",
			highTotal, restTotal, minSupport),
		fmt.Sprintf("Most over-represented: %s (%.2fx its share elsewhere).", ls[0].typ, ls[0].ratio),
		fmt.Sprintf("Most under-represented: %s (%.2fx).", last.typ, last.ratio),
	}
	return s
}

// ---------------------------------------------------------------------------
// Q3: hour-of-day profiles
// ---------------------------------------------------------------------------

type hourlyQuestion struct{ questionMeta }

func (q hourlyQuestion) Run(j *Joined) Section {
	s := q.newSection()
	ch, rh := j.Crime.byHour, j.Req.byHour // arrays: these are copies
	cTot, rTot := 0, 0
	for h := 0; h < 24; h++ {
		cTot += ch[h]
		rTot += rh[h]
	}
	if cTot == 0 || rTot == 0 {
		s.Findings = []string{"One of the datasets has no usable timestamps."}
		return s
	}
	cs := make([]float64, 24)
	rs := make([]float64, 24)
	peakC, peakR := 0, 0
	for h := 0; h < 24; h++ {
		cs[h] = 100 * float64(ch[h]) / float64(cTot)
		rs[h] = 100 * float64(rh[h]) / float64(rTot)
		if cs[h] > cs[peakC] {
			peakC = h
		}
		if rs[h] > rs[peakR] {
			peakR = h
		}
	}
	var sb strings.Builder
	sb.WriteString(" Share of each dataset's records by hour of day (█ = crime, ▒ = 311):\n\n")
	writeDualHourChart(&sb, cs, rs, 40)
	s.Body = sb.String()

	nightC, nightR, eveC, eveR := 0.0, 0.0, 0.0, 0.0
	for h := 0; h < 6; h++ {
		nightC += cs[h]
		nightR += rs[h]
	}
	for h := 18; h < 24; h++ {
		eveC += cs[h]
		eveR += rs[h]
	}
	corr := pearson(cs, rs)
	s.Findings = []string{
		fmt.Sprintf("Crime peaks at %02d:00 (%.1f%% of incidents); 311 requests peak at %02d:00 (%.1f%% of requests).",
			peakC, cs[peakC], peakR, rs[peakR]),
		fmt.Sprintf("The two 24-hour profiles correlate at r = %s (%s).", fmtFloat(corr, 2), describeCorr(corr)),
		fmt.Sprintf("Evening (18:00-23:59) holds %.1f%% of crime but %.1f%% of 311 requests; overnight (00:00-05:59) holds %.1f%% versus %.1f%%.",
			eveC, eveR, nightC, nightR),
		"Hours are used as written in the source files (use -tz to convert to a local time zone).",
	}
	return s
}

// ---------------------------------------------------------------------------
// Q4: service equity (time to close by crime quartile)
// ---------------------------------------------------------------------------

type equityQuestion struct{ questionMeta }

func (q equityQuestion) Run(j *Joined) Section {
	s := q.newSection()
	type group struct {
		neighborhoods int
		total, closed int
		days          []float32
	}
	var groups [5]group // index 1..4 = crime quartile
	typeTotals := make(map[string]int)
	lowDays := make(map[string][]float32)
	highDays := make(map[string][]float32)
	for _, r := range j.Rows {
		g := &groups[r.CrimeQuartile]
		g.neighborhoods++
		g.total += r.Req.Total
		g.closed += r.Req.Closed
		for t, c := range r.Req.ByType {
			typeTotals[t] += c
		}
		for t, d := range r.Req.Durations {
			g.days = append(g.days, d...)
			switch r.CrimeQuartile {
			case 1:
				lowDays[t] = append(lowDays[t], d...)
			case 4:
				highDays[t] = append(highDays[t], d...)
			}
		}
	}

	labels := []string{"", "Q1 (lowest crime)", "Q2", "Q3", "Q4 (highest crime)"}
	rows := make([][]string, 0, 4)
	chartLabels := make([]string, 0, 4)
	medVals := make([]float64, 0, 4)
	for qi := 1; qi <= 4; qi++ {
		g := groups[qi]
		med, avg := medianF32(g.days), meanF32(g.days)
		rows = append(rows, []string{
			labels[qi], fmt.Sprint(g.neighborhoods), fmt.Sprint(g.total),
			pct(g.closed, g.total), fmtFloat(med, 1), fmtFloat(avg, 1),
		})
		chartLabels = append(chartLabels, labels[qi])
		medVals = append(medVals, nz(med))
	}

	type typeCount struct {
		typ string
		n   int
	}
	var tcs []typeCount
	for t, n := range typeTotals {
		tcs = append(tcs, typeCount{typ: t, n: n})
	}
	sort.Slice(tcs, func(a, b int) bool {
		if tcs[a].n != tcs[b].n {
			return tcs[a].n > tcs[b].n
		}
		return tcs[a].typ < tcs[b].typ
	})
	if len(tcs) > 5 {
		tcs = tcs[:5]
	}
	typeRows := make([][]string, 0, len(tcs))
	for _, t := range tcs {
		lo, hi := medianF32(lowDays[t.typ]), medianF32(highDays[t.typ])
		diff := "n/a"
		if !math.IsNaN(lo) && !math.IsNaN(hi) {
			diff = fmt.Sprintf("%+.1f", hi-lo)
		}
		typeRows = append(typeRows, []string{
			truncate(t.typ, 36), fmt.Sprint(t.n), fmtFloat(lo, 1), fmtFloat(hi, 1), diff,
		})
	}

	var sb strings.Builder
	sb.WriteString(" Time to close by crime-volume quartile of the neighborhood (days):\n\n")
	writeTable(&sb, []string{"Crime quartile", "Neighborhoods", "Requests", "Closed", "Median days", "Mean days"}, rows)
	sb.WriteString("\n Median days to close:\n")
	writeBarChart(&sb, chartLabels, medVals, " d", 40)
	sb.WriteString("\n Five most common request types, Q1 vs Q4 (median days to close):\n\n")
	writeTable(&sb, []string{"Request type", "Requests", "Q1 median", "Q4 median", "Q4 - Q1"}, typeRows)
	sb.WriteString("\n Only requests with a close date are timed, so still-open requests are not\n")
	sb.WriteString(" reflected in the medians (the closure rate column shows how many were timed).\n")
	s.Body = sb.String()

	g1, g4 := groups[1], groups[4]
	m1, m4 := medianF32(g1.days), medianF32(g4.days)
	s.Findings = []string{
		fmt.Sprintf("Median time to close: %s days in the lowest-crime quartile versus %s days in the highest-crime quartile.",
			fmtFloat(m1, 1), fmtFloat(m4, 1)),
		fmt.Sprintf("Share of requests with a usable close date: %s (Q1) versus %s (Q4).",
			pct(g1.closed, g1.total), pct(g4.closed, g4.total)),
	}
	if !math.IsNaN(m1) && !math.IsNaN(m4) && m1 > 0 {
		ratio := m4 / m1
		verdict := "no meaningful gap"
		if ratio > 1.1 {
			verdict = "slower service in high-crime areas"
		} else if ratio < 0.9 {
			verdict = "faster service in high-crime areas"
		}
		s.Findings = append(s.Findings, fmt.Sprintf("The highest-crime quartile's median is %.2fx the lowest quartile's: %s.", ratio, verdict))
	}
	return s
}

// ---------------------------------------------------------------------------
// Q5: physical disorder vs violent crime
// ---------------------------------------------------------------------------

type disorderQuestion struct{ questionMeta }

func (q disorderQuestion) Run(j *Joined) Section {
	s := q.newSection()
	n := len(j.Rows)
	if n < 4 {
		s.Findings = []string{"Fewer than 4 neighborhoods appear in both datasets; quartile comparison is not meaningful."}
		return s
	}
	dCount := make([]float64, n)
	vCount := make([]float64, n)
	dShare := make([]float64, n)
	vShare := make([]float64, n)
	totalDisorder := 0
	for i, r := range j.Rows {
		dCount[i] = float64(r.Req.Disorder)
		vCount[i] = float64(r.Crime.Violent)
		dShare[i] = float64(r.Req.Disorder) / float64(r.Req.Total)
		vShare[i] = float64(r.Crime.Violent) / float64(r.Crime.Total)
		totalDisorder += r.Req.Disorder
	}
	if totalDisorder == 0 {
		s.Findings = []string{"No 311 request type matched the disorder keywords; adjust them with -disorder."}
		return s
	}

	type group struct {
		n                      int
		disorder, violent, all int
	}
	var groups [5]group // index 1..4 = disorder quartile
	for _, r := range j.Rows {
		g := &groups[r.DisorderQuartile]
		g.n++
		g.disorder += r.Req.Disorder
		g.violent += r.Crime.Violent
		g.all += r.Crime.Total
	}
	avg := func(total, count int) string {
		if count == 0 {
			return "n/a"
		}
		return fmt.Sprintf("%.1f", float64(total)/float64(count))
	}
	labels := []string{"", "Q1 (fewest disorder)", "Q2", "Q3", "Q4 (most disorder)"}
	rows := make([][]string, 0, 4)
	chartLabels := make([]string, 0, 4)
	chartVals := make([]float64, 0, 4)
	for qi := 1; qi <= 4; qi++ {
		g := groups[qi]
		rows = append(rows, []string{
			labels[qi], fmt.Sprint(g.n), avg(g.disorder, g.n), avg(g.violent, g.n),
			avg(g.all, g.n), pct(g.violent, g.all),
		})
		chartLabels = append(chartLabels, labels[qi])
		if g.n > 0 {
			chartVals = append(chartVals, float64(g.violent)/float64(g.n))
		} else {
			chartVals = append(chartVals, 0)
		}
	}

	// Priority list: neighborhoods in the top disorder quartile, worst violent crime first.
	var pri []*JoinedRow
	for _, r := range j.Rows {
		if r.DisorderQuartile == 4 {
			pri = append(pri, r)
		}
	}
	sort.Slice(pri, func(a, b int) bool {
		if pri[a].Crime.Violent != pri[b].Crime.Violent {
			return pri[a].Crime.Violent > pri[b].Crime.Violent
		}
		return pri[a].Name < pri[b].Name
	})
	if len(pri) > 5 {
		pri = pri[:5]
	}
	priRows := make([][]string, 0, len(pri))
	for _, r := range pri {
		priRows = append(priRows, []string{
			truncate(r.Name, 28), fmt.Sprint(r.Req.Disorder), fmt.Sprint(r.Crime.Violent), fmt.Sprint(r.Crime.Total),
		})
	}

	var sb strings.Builder
	sb.WriteString(" Neighborhoods grouped by volume of physical-disorder 311 requests:\n\n")
	writeTable(&sb, []string{"Disorder quartile", "Neighborhoods", "Avg disorder reqs", "Avg violent crimes", "Avg all crimes", "Violent share"}, rows)
	sb.WriteString("\n Average violent crimes per neighborhood:\n")
	writeBarChart(&sb, chartLabels, chartVals, "", 40)
	sb.WriteString("\n Priority neighborhoods for coordinated cleanup + enforcement\n")
	sb.WriteString(" (top disorder quartile, ordered by violent crime):\n\n")
	writeTable(&sb, []string{"Neighborhood", "Disorder reqs", "Violent crimes", "All crimes"}, priRows)
	s.Body = sb.String()

	prCount, prShare := pearson(dCount, vCount), pearson(dShare, vShare)
	s.Findings = []string{
		fmt.Sprintf("Disorder-request volume and violent-crime volume correlate at r = %s (%s) across %d neighborhoods.",
			fmtFloat(prCount, 2), describeCorr(prCount), n),
		fmt.Sprintf("Using shares instead of raw counts (disorder as %% of 311 requests, violent as %% of crime) gives r = %s (%s), which is less affected by neighborhood size.",
			fmtFloat(prShare, 2), describeCorr(prShare)),
	}
	if groups[1].n > 0 && groups[4].n > 0 && groups[1].violent > 0 {
		a1 := float64(groups[1].violent) / float64(groups[1].n)
		a4 := float64(groups[4].violent) / float64(groups[4].n)
		s.Findings = append(s.Findings,
			fmt.Sprintf("The most disorder-heavy quartile averages %.1f violent crimes per neighborhood versus %.1f in the least disorder-heavy quartile (%.2fx).", a4, a1, a4/a1))
	}
	s.Findings = append(s.Findings, "Counts are not normalised by population or area, and correlation does not show causation.")
	return s
}
