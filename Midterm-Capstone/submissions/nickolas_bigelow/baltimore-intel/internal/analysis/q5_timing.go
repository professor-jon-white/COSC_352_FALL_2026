package analysis

import (
	"fmt"
	"time"

	"baltimore-intel/internal/ingest"
	"baltimore-intel/internal/join"
	"baltimore-intel/internal/model"
	"baltimore-intel/internal/report"
)

// LateNight (Q5, BPD patrol command + Mayor's Office of Neighborhood Safety):
// when, and around which places, should late-night resources be deployed?
type LateNight struct{}

func (LateNight) ID() string { return "q5-late-night" }

var days = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
var blocks = [6]string{"00-04", "04-08", "08-12", "12-16", "16-20", "20-24"}

type timeAcc struct {
	grid          [7][24]float64 // violent incidents by weekday x hour
	nearBlock     [6]float64
	farBlock      [6]float64
	nearLate      float64 // 22:00-02:59 near an outlet
	farLate       float64
	nearAll       float64
	farAll        float64
	exactMidnight float64 // records stamped 00:00:00 (often "time unknown")
	hoodBlock     map[[2]int]float64
}

func isLate(h int) bool { return h >= 22 || h < 3 }

func (LateNight) Run(j *join.Joined) (*report.Section, error) {
	ds := j.DS
	acc := join.ParallelReduce(ds.Crimes, j.Workers,
		func() *timeAcc { return &timeAcc{hoodBlock: map[[2]int]float64{}} },
		func(a *timeAcc, c *model.Crime) {
			if !c.Violent {
				return
			}
			t := time.Unix(c.Time, 0).In(ingest.Eastern)
			h := t.Hour()
			n := float64(c.Incidents)
			if h == 0 && t.Minute() == 0 && t.Second() == 0 {
				a.exactMidnight += n // almost always "time unknown": exclude from timing
				return
			}
			wd := (int(t.Weekday()) + 6) % 7 // Monday first
			a.grid[wd][h] += n
			blk := h / 4
			if c.Hood != 0 {
				a.hoodBlock[[2]int{int(c.Hood), blk}] += n
			}
		},
		func(dst, src *timeAcc) {
			for d := range dst.grid {
				for h := range dst.grid[d] {
					dst.grid[d][h] += src.grid[d][h]
				}
			}
			dst.exactMidnight += src.exactMidnight
			for k, v := range src.hoodBlock {
				dst.hoodBlock[k] += v
			}
		})

	// The near/far split needs each crime's index (to read CrimeNearOutlet),
	// so it is a separate pass sharded across workers, each owning one slot.
	type split struct{ near, far [24]float64 }
	w := max(1, j.Workers)
	parts := make([]split, w)
	shard := (len(ds.Crimes) + w - 1) / w
	join.ParallelFor(w, w, func(k int) {
		p := &parts[k]
		for i := k * shard; i < min((k+1)*shard, len(ds.Crimes)); i++ {
			c := &ds.Crimes[i]
			if !c.Violent || c.Lat == 0 {
				continue
			}
			lt := time.Unix(c.Time, 0).In(ingest.Eastern)
			if lt.Hour() == 0 && lt.Minute() == 0 && lt.Second() == 0 {
				continue // unknown-time records, see above
			}
			h := lt.Hour()
			if j.CrimeNearOutlet[i] {
				p.near[h] += float64(c.Incidents)
			} else {
				p.far[h] += float64(c.Incidents)
			}
		}
	})
	for _, p := range parts {
		for h := 0; h < 24; h++ {
			acc.nearBlock[h/4] += p.near[h]
			acc.farBlock[h/4] += p.far[h]
			acc.nearAll += p.near[h]
			acc.farAll += p.far[h]
			if isLate(h) {
				acc.nearLate += p.near[h]
				acc.farLate += p.far[h]
			}
		}
	}

	// Heatmap and its peak.
	heat := &report.Heatmap{Title: "Violent crime by day of week and hour (Baltimore local time; darker = more)",
		RowLabels: days[:], ColLabels: make([]string, 24), Values: make([][]float64, 7)}
	peakD, peakH, total := 0, 0, 0.0
	for h := 0; h < 24; h++ {
		heat.ColLabels[h] = fmt.Sprintf("%02d", h)
	}
	for d := 0; d < 7; d++ {
		heat.Values[d] = acc.grid[d][:]
		for h := 0; h < 24; h++ {
			total += acc.grid[d][h]
			if acc.grid[d][h] > acc.grid[peakD][peakH] {
				peakD, peakH = d, h
			}
		}
	}
	var weekendLate, allLate float64
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			if isLate(h) {
				allLate += acc.grid[d][h]
				if (d == 4 || d == 5) && h >= 22 || (d == 5 || d == 6) && h < 3 { // Fri/Sat nights
					weekendLate += acc.grid[d][h]
				}
			}
		}
	}

	hourBars := &report.Bars{Title: "Violent crimes by hour of day (all days)", Format: "%.0f"}
	for h := 0; h < 24; h++ {
		s := 0.0
		for d := 0; d < 7; d++ {
			s += acc.grid[d][h]
		}
		hourBars.Labels = append(hourBars.Labels, fmt.Sprintf("%02d:00", h))
		hourBars.Values = append(hourBars.Values, s)
	}

	blockTbl := &report.Table{Title: "When violence happens near liquor outlets vs elsewhere (share of each group's violent crime)",
		Headers: []string{"Time block", "Near an outlet", "Elsewhere", "Difference"},
		Right:   []bool{false, true, true, true}, CSVName: "q5_time_blocks.csv"}
	for b := 0; b < 6; b++ {
		n, f := ratio(acc.nearBlock[b], acc.nearAll), ratio(acc.farBlock[b], acc.farAll)
		blockTbl.Rows = append(blockTbl.Rows, []string{blocks[b], report.Pct(n), report.Pct(f),
			fmt.Sprintf("%+.1f pts", (n-f)*100)})
	}

	// Neighborhood x time-block hotspots for patrol scheduling.
	hot := map[string]float64{}
	for k, v := range acc.hoodBlock {
		hot[fmt.Sprintf("%s|%s", ds.Hoods.Name(model.ID(k[0])), blocks[k[1]])] = v
	}
	hotTbl := &report.Table{Title: "Top neighborhood + time-block combinations for violent crime",
		Headers: []string{"#", "Neighborhood", "Time block", "Violent crimes", "Share of city total"},
		Right:   []bool{true, false, false, true, true}, CSVName: "q5_hotspots.csv"}
	for i, r := range topN(hot, 12) {
		var name, blk string
		for k := 0; k < len(r.key); k++ {
			if r.key[k] == '|' {
				name, blk = r.key[:k], r.key[k+1:]
			}
		}
		hotTbl.Rows = append(hotTbl.Rows, []string{fmt.Sprint(i + 1), name, blk, report.F(r.value, 0),
			report.Pct(ratio(r.value, total))})
	}

	nearLateShare, farLateShare := ratio(acc.nearLate, acc.nearAll), ratio(acc.farLate, acc.farAll)
	finding := fmt.Sprintf("Violent crime peaks on %s around %02d:00. Late night (22:00-03:00) accounts for %s of violent crime "+
		"near liquor outlets versus %s elsewhere, and Friday/Saturday nights alone hold %s of all late-night violence. "+
		"The neighborhood and time-block table below gives patrol command a ranked deployment schedule.",
		days[peakD], peakH, report.Pct(nearLateShare), report.Pct(farLateShare), report.Pct(ratio(weekendLate, allLate)))

	notes := []string{"Times are converted from UTC to America/New_York, so daylight-saving changes are handled."}
	if acc.exactMidnight > 0 && total > 0 {
		notes = append(notes, fmt.Sprintf("%.0f violent records stamped exactly 00:00:00 were excluded from the timing analysis: "+
			"that stamp is how the source data records an unknown time, and it would otherwise create a false midnight peak "+
			"(%.1fx a typical hour).", acc.exactMidnight, acc.exactMidnight/(total/168)))
	}

	return &report.Section{
		ID: "q5", Stakeholder: "BPD Patrol Command & Mayor's Office of Neighborhood Safety and Engagement",
		Question: "When, and around which places, should late-night patrol and outreach be concentrated?",
		Datasets: []string{"Part 1 Crime", "Liquor Licenses"},
		Finding:  finding,
		Blocks:   []report.Block{heat, hourBars, blockTbl, hotTbl},
		Actions: []string{
			fmt.Sprintf("Shift overtime and violence-interruption outreach toward %s and the hours around %02d:00.", days[peakD], peakH),
			"Schedule Friday and Saturday closing-time (22:00-03:00) foot patrols around the highest-exposure outlets from Q2.",
			"Use the hotspot table as the starting point for a time-and-place deployment plan, reviewed monthly.",
		},
		Notes: notes,
	}, nil
}
