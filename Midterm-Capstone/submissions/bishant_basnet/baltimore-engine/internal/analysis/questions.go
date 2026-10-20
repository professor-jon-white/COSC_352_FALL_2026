package analysis

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/bishant/baltimore-engine/internal/report"
)

// Result is a question's answer, independent of output format.
type Result struct {
	ID          int
	Title       string
	Stakeholder string
	Question    string
	Findings    []string
	Charts      []report.Chart
	Tables      []report.Table
}

// Question is the Strategy interface every research question implements.
type Question interface {
	ID() int
	Answer(c *Context) Result
}

type base struct {
	id                         int
	title, stakeholder, prompt string
}

func (b base) ID() int { return b.id }
func (b base) result() Result {
	return Result{ID: b.id, Title: b.title, Stakeholder: b.stakeholder, Question: b.prompt}
}

// NewQuestion is a Factory for the five research questions.
func NewQuestion(id int) (Question, error) {
	switch id {
	case 1:
		return burdenQ{base{1, "Where is combined public-safety and service burden highest?",
			"Mayor's Office", "Which neighborhoods carry the heaviest combined load of Part 1 crime AND 311 service demand, and how concentrated is it?"}}, nil
	case 2:
		return engagementQ{base{2, "Which high-crime neighborhoods are under-using 311?",
			"Mayor's Office of Neighborhood Engagement", "Does 311 demand track crime, and which high-crime neighborhoods file unusually few 311 requests per crime (possible under-reporting)?"}}, nil
	case 3:
		return hourlyQ{base{3, "When do crime and service demand peak during the day?",
			"Baltimore Police Dept. & Department of Transportation", "How do crime and 311 intake vary by hour of day, and do the peaks line up for staffing and patrol planning?"}}, nil
	case 4:
		return blightQ{base{4, "Is physical blight associated with crime?",
			"Department of Planning / Housing & Community Development", "Do neighborhoods with more blight-related 311 requests (trash, rats, vacants, dumping) also experience more Part 1 crime?"}}, nil
	case 5:
		return equityQ{base{5, "Are 311 requests resolved equitably across crime levels?",
			"Department of Public Works / Office of Equity", "Do high-crime neighborhoods wait longer for their 311 requests to be closed than low-crime neighborhoods?"}}, nil
	}
	return nil, fmt.Errorf("unknown question %d", id)
}

func DefaultQuestions() []Question {
	qs := make([]Question, 0, 5)
	for i := 1; i <= 5; i++ {
		q, _ := NewQuestion(i)
		qs = append(qs, q)
	}
	return qs
}

// RunAll answers every question concurrently (one goroutine each). Results
// are written to distinct slice indexes, so no mutex is required.
func RunAll(c *Context, qs []Question) []Result {
	res := make([]Result, len(qs))
	var wg sync.WaitGroup
	for i, q := range qs {
		wg.Add(1)
		go func(i int, q Question) {
			defer wg.Done()
			res[i] = q.Answer(c)
		}(i, q)
	}
	wg.Wait()
	return res
}

func noJoin(r Result) Result {
	r.Findings = append(r.Findings, "No neighborhoods appear in both datasets, so a cross-dataset answer is not possible. Check that both files contain a Neighborhood column.")
	return r
}

func itoa(n int) string { return report.Commas(int64(n)) }

// ---------------- Q1 ----------------
type burdenQ struct{ base }

func (q burdenQ) Answer(c *Context) Result {
	r := q.result()
	if len(c.Joined) == 0 {
		return noJoin(r)
	}
	var maxC, maxR float64
	for _, h := range c.Joined {
		maxC = math.Max(maxC, float64(h.Crimes))
		maxR = math.Max(maxR, float64(h.Requests))
	}
	type row struct {
		h   *HoodStats
		idx float64
	}
	rows := make([]row, len(c.Joined))
	for i, h := range c.Joined {
		rows[i] = row{h, 50*float64(h.Crimes)/maxC + 50*float64(h.Requests)/maxR}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].idx > rows[b].idx })
	n := min(c.TopN, len(rows))
	ch := report.Chart{Title: fmt.Sprintf("Top %d neighborhoods by composite burden index (0-100)", n)}
	tb := report.Table{Headers: []string{"Rank", "Neighborhood", "Part 1 crimes", "311 requests", "Burden index"}}
	topC, topR := 0, 0
	for i := 0; i < n; i++ {
		h := rows[i].h
		topC += h.Crimes
		topR += h.Requests
		ch.Labels = append(ch.Labels, h.Name)
		ch.Values = append(ch.Values, math.Round(rows[i].idx*10)/10)
		tb.Rows = append(tb.Rows, []string{fmt.Sprint(i + 1), h.Name, itoa(h.Crimes), itoa(h.Requests), fmt.Sprintf("%.1f", rows[i].idx)})
	}
	r.Findings = append(r.Findings,
		fmt.Sprintf("%d neighborhoods were matched across both datasets.", len(c.Joined)),
		fmt.Sprintf("Highest combined burden: %s (index %.1f of 100).", rows[0].h.Name, rows[0].idx),
		fmt.Sprintf("The top %d neighborhoods (%.1f%% of matched neighborhoods) generate %.1f%% of crime and %.1f%% of 311 demand.",
			n, pct(n, len(rows)), pct(topC, c.JoinedCrimes), pct(topR, c.JoinedRequests)),
		"Recommendation: prioritize these neighborhoods for coordinated multi-agency investment rather than single-agency responses.")
	r.Charts = append(r.Charts, ch)
	r.Tables = append(r.Tables, tb)
	return r
}

// ---------------- Q2 ----------------
type engagementQ struct{ base }

func (q engagementQ) Answer(c *Context) Result {
	r := q.result()
	if len(c.Joined) < 2 {
		return noJoin(r)
	}
	xs := make([]float64, len(c.Joined))
	ys := make([]float64, len(c.Joined))
	for i, h := range c.Joined {
		xs[i], ys[i] = float64(h.Crimes), float64(h.Requests)
	}
	corr := Pearson(xs, ys)
	q3 := Percentile(xs, 75)
	var cand []*HoodStats
	for _, h := range c.Joined {
		if float64(h.Crimes) >= q3 {
			cand = append(cand, h)
		}
	}
	ratio := func(h *HoodStats) float64 { return float64(h.Requests) / float64(h.Crimes) }
	sort.Slice(cand, func(a, b int) bool { return ratio(cand[a]) < ratio(cand[b]) })
	n := min(c.TopN, len(cand))
	ch := report.Chart{Title: "311 requests per Part 1 crime, high-crime neighborhoods (lowest = least engaged)"}
	tb := report.Table{Headers: []string{"Neighborhood", "Part 1 crimes", "311 requests", "311 per crime"}}
	for _, h := range cand[:n] {
		ch.Labels = append(ch.Labels, h.Name)
		ch.Values = append(ch.Values, math.Round(ratio(h)*10)/10)
		tb.Rows = append(tb.Rows, []string{h.Name, itoa(h.Crimes), itoa(h.Requests), fmt.Sprintf("%.2f", ratio(h))})
	}
	cityRatio := float64(c.JoinedRequests) / float64(max(c.JoinedCrimes, 1))
	r.Findings = append(r.Findings,
		fmt.Sprintf("Crime and 311 volume across neighborhoods have a %s correlation (Pearson r = %.2f).", DescribeR(corr), corr),
		fmt.Sprintf("Citywide (matched) there are %.2f 311 requests per Part 1 crime.", cityRatio),
		fmt.Sprintf("%d neighborhoods are in the top crime quartile (>= %.0f crimes); those listed below have the lowest 311 engagement relative to their crime level.", len(cand), q3))
	if n > 0 {
		r.Findings = append(r.Findings, fmt.Sprintf("Lowest engagement: %s at %.2f requests per crime (%.0f%% of the citywide rate).",
			cand[0].Name, ratio(cand[0]), 100*ratio(cand[0])/cityRatio))
	}
	r.Findings = append(r.Findings, "Recommendation: target 311 awareness and community-liaison outreach here; low reporting can hide service needs.")
	r.Charts = append(r.Charts, ch)
	r.Tables = append(r.Tables, tb)
	return r
}

// ---------------- Q3 ----------------
type hourlyQ struct{ base }

func peak(a [24]int) int {
	best := 0
	for i := range a {
		if a[i] > a[best] {
			best = i
		}
	}
	return best
}

func (q hourlyQ) Answer(c *Context) Result {
	r := q.result()
	mk := func(title string, a [24]int) report.Chart {
		ch := report.Chart{Title: title}
		for h := 0; h < 24; h++ {
			ch.Labels = append(ch.Labels, fmt.Sprintf("%02d:00", h))
			ch.Values = append(ch.Values, float64(a[h]))
		}
		return ch
	}
	if c.CrimeTimed == 0 && c.ReqTimed == 0 {
		r.Findings = append(r.Findings, "Neither dataset contained parseable timestamps.")
		return r
	}
	night, business := 0, 0
	for h := 0; h < 24; h++ {
		if h >= 20 || h < 5 {
			night += c.CrimeByHour[h]
		}
		if h >= 8 && h < 17 {
			business += c.ReqByHour[h]
		}
	}
	pc, pr := peak(c.CrimeByHour), peak(c.ReqByHour)
	r.Findings = append(r.Findings,
		fmt.Sprintf("Crime peaks at %02d:00 (%s incidents); 311 intake peaks at %02d:00 (%s requests).", pc, itoa(c.CrimeByHour[pc]), pr, itoa(c.ReqByHour[pr])),
		fmt.Sprintf("%.1f%% of timestamped crime occurs between 20:00 and 05:00.", pct(night, c.CrimeTimed)),
		fmt.Sprintf("%.1f%% of 311 requests arrive during business hours (08:00-17:00).", pct(business, c.ReqTimed)),
		fmt.Sprintf("Peaks are %d hours apart, so patrol staffing and 311/DOT crew staffing should be scheduled on different curves.", absDiff(pc, pr)))
	r.Charts = append(r.Charts,
		mk("Part 1 crime by hour of day (Baltimore local time)", c.CrimeByHour),
		mk("311 requests created by hour of day", c.ReqByHour))
	return r
}

func absDiff(a, b int) int {
	d := a - b
	if d < 0 {
		d = -d
	}
	if d > 12 {
		d = 24 - d // circular clock distance
	}
	return d
}

// ---------------- Q4 ----------------
type blightQ struct{ base }

func (q blightQ) Answer(c *Context) Result {
	r := q.result()
	if len(c.Joined) < 4 {
		return noJoin(r)
	}
	qs := quartiles(c.Joined, func(h *HoodStats) float64 { return float64(h.Blight) })
	labels := []string{"Q1 (least blight)", "Q2", "Q3", "Q4 (most blight)"}
	ch := report.Chart{Title: "Average Part 1 crimes per neighborhood, by blight-request quartile"}
	tb := report.Table{Headers: []string{"Blight quartile", "Neighborhoods", "Blight requests (range)", "Avg crimes / neighborhood"}}
	var avg [4]float64
	for i, grp := range qs {
		sum := 0
		for _, h := range grp {
			sum += h.Crimes
		}
		if len(grp) > 0 {
			avg[i] = float64(sum) / float64(len(grp))
		}
		rng := "-"
		if len(grp) > 0 {
			rng = fmt.Sprintf("%s - %s", itoa(grp[0].Blight), itoa(grp[len(grp)-1].Blight))
		}
		ch.Labels = append(ch.Labels, labels[i])
		ch.Values = append(ch.Values, math.Round(avg[i]))
		tb.Rows = append(tb.Rows, []string{labels[i], fmt.Sprint(len(grp)), rng, fmt.Sprintf("%.1f", avg[i])})
	}
	xs := make([]float64, len(c.Joined))
	ys := make([]float64, len(c.Joined))
	for i, h := range c.Joined {
		xs[i], ys[i] = float64(h.Blight), float64(h.Crimes)
	}
	corr := Pearson(xs, ys)
	r.Findings = append(r.Findings,
		fmt.Sprintf("%.1f%% of all 311 requests are blight/sanitation related.", pct(c.TotalBlight, c.TotalRequests)),
		fmt.Sprintf("Blight requests and crime have a %s correlation (r = %.2f).", DescribeR(corr), corr))
	if avg[0] > 0 {
		r.Findings = append(r.Findings, fmt.Sprintf("Neighborhoods in the highest-blight quartile average %.1fx the crime of the lowest-blight quartile.", avg[3]/avg[0]))
	}
	r.Findings = append(r.Findings, "Caveat: correlation is not causation; both may share underlying drivers such as vacancy and disinvestment.",
		"Recommendation: pair code enforcement and vacant-lot remediation with public-safety strategy in Q4 neighborhoods.")
	r.Charts = append(r.Charts, ch)
	r.Tables = append(r.Tables, tb)
	return r
}

// ---------------- Q5 ----------------
type equityQ struct{ base }

func (q equityQ) Answer(c *Context) Result {
	r := q.result()
	if len(c.Joined) < 4 {
		return noJoin(r)
	}
	qs := quartiles(c.Joined, func(h *HoodStats) float64 { return float64(h.Crimes) })
	labels := []string{"Q1 (lowest crime)", "Q2", "Q3", "Q4 (highest crime)"}
	ch := report.Chart{Title: "Median hours to close a 311 request, by neighborhood crime quartile", Unit: " h"}
	tb := report.Table{Headers: []string{"Crime quartile", "Neighborhoods", "Closed requests", "Median hrs", "90th pct hrs"}}
	var med [4]float64
	total := 0
	for i, grp := range qs {
		var pool []float64
		for _, h := range grp {
			for _, v := range h.ResolutionHrs {
				pool = append(pool, float64(v))
			}
		}
		total += len(pool)
		med[i] = Median(pool)
		p90 := Percentile(pool, 90)
		ch.Labels = append(ch.Labels, labels[i])
		v := med[i]
		if math.IsNaN(v) {
			v = 0
		}
		ch.Values = append(ch.Values, math.Round(v*10)/10)
		tb.Rows = append(tb.Rows, []string{labels[i], fmt.Sprint(len(grp)), itoa(len(pool)), fmtH(med[i]), fmtH(p90)})
	}
	if total == 0 {
		r.Findings = append(r.Findings, "No 311 records had both a created and closed date, so resolution time cannot be measured.")
		return r
	}
	r.Findings = append(r.Findings, fmt.Sprintf("%s closed 311 requests had measurable resolution times.", itoa(total)))
	if !math.IsNaN(med[0]) && !math.IsNaN(med[3]) && med[0] > 0 {
		diff := 100 * (med[3] - med[0]) / med[0]
		word := "longer"
		if diff < 0 {
			word = "shorter"
		}
		r.Findings = append(r.Findings, fmt.Sprintf("Highest-crime neighborhoods wait a median %.1f h vs %.1f h in the lowest-crime ones (%.0f%% %s).",
			med[3], med[0], math.Abs(diff), word))
	}
	r.Findings = append(r.Findings, "Recommendation: set equity-based service-level targets so resolution time does not depend on neighborhood.")
	r.Charts = append(r.Charts, ch)
	r.Tables = append(r.Tables, tb)
	return r
}

func fmtH(v float64) string {
	if math.IsNaN(v) {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", v)
}
