package report

import (
	"fmt"
	"io"

	"capstone/internal/model"
	"capstone/internal/stats"
)

// StreetlightsVsNightCrime asks whether neighborhoods that report many
// broken or inadequate streetlights see more outdoor crime at night.
//
// Night is 8 PM to 6 AM Baltimore time, excluding timestamps of exactly
// midnight (a common placeholder for an unknown time), and outdoor means the
// crime's Inside_Outside field is "O". See match.IsNight.
//
// Raw counts mostly measure neighborhood size, so the report also checks the
// night-outdoor share: what fraction of a neighborhood's crime happens
// outdoors at night. If dark streets matter, that share, not just the count,
// should rise with streetlight complaints.
type StreetlightsVsNightCrime struct{}

func (StreetlightsVsNightCrime) ID() string { return "streetlights" }
func (StreetlightsVsNightCrime) Question() string {
	return "Do neighborhoods with many streetlight requests see more nighttime outdoor crime?"
}

func (StreetlightsVsNightCrime) Render(w io.Writer, in *Input) error {
	lights := func(s *model.Summary) float64 { return float64(countTypes(s.RequestsByType, isStreetlight)) }
	rs := rows(in, lights, func(s *model.Summary) float64 { return float64(s.NightOutdoorCrimes) })

	type group struct {
		n, lightsLo, lightsHi int
		lightsAvg, nightAvg   float64
		share                 float64 // pooled night-outdoor crimes / all crimes
	}
	var groups [4]group
	var maxNight, maxShare float64
	for g, members := range quartiles(rs, byX) {
		gr := group{n: len(members), lightsLo: int(members[0].x)}
		var l, night, crimes int
		for _, r := range members {
			gr.lightsHi = max(gr.lightsHi, int(r.x))
			l += int(r.x)
			night += r.summary.NightOutdoorCrimes
			crimes += r.summary.CrimeCount
		}
		gr.lightsAvg = float64(l) / float64(gr.n)
		gr.nightAvg = float64(night) / float64(gr.n)
		if crimes > 0 {
			gr.share = float64(night) / float64(crimes)
		}
		groups[g] = gr
		maxNight, maxShare = max(maxNight, gr.nightAvg), max(maxShare, gr.share)
	}

	t := newTable(w, "lights quartile", "nbhds", "requests each", "night outdoor (avg)", "", "share of crime", "")
	for g, gr := range groups {
		t.row(quartileNames[g], gr.n, fmt.Sprintf("%s-%s", num(gr.lightsLo), num(gr.lightsHi)),
			fmt.Sprintf("%.1f", gr.nightAvg), bar(gr.nightAvg, maxNight),
			fmt.Sprintf("%.1f%%", gr.share*100), bar(gr.share, maxShare))
	}
	if err := t.flush(); err != nil {
		return err
	}

	// Share only for neighborhoods with enough crime for a percentage to mean much.
	withCrime := make([]row, 0, len(rs))
	shares := make([]float64, 0, len(rs))
	for _, r := range rs {
		if r.summary.CrimeCount >= 20 {
			withCrime = append(withCrime, r)
			shares = append(shares, float64(r.summary.NightOutdoorCrimes)/float64(r.summary.CrimeCount))
		}
	}
	allCrime := make([]float64, len(rs))
	for i, r := range rs {
		allCrime[i] = float64(r.summary.CrimeCount)
	}
	rhoCount := stats.Spearman(xs(rs), ys(rs))
	rhoAll := stats.Spearman(xs(rs), allCrime)
	rhoShare := stats.Spearman(xs(withCrime), shares)

	fmt.Fprintf(w, "\nSpearman rank correlation with streetlight requests:\n")
	fmt.Fprintf(w, "  night outdoor crimes (count)           %s\n", corr(rhoCount))
	fmt.Fprintf(w, "  all crimes (count, baseline)           %s\n", corr(rhoAll))
	fmt.Fprintf(w, "  night outdoor share of crime (%d nbhds) %s\n", len(withCrime), corr(rhoShare))

	lo, hi := groups[0].share*100, groups[3].share*100
	fmt.Fprint(w, "\nAnswer: ")
	switch {
	case rhoCount < 0.3:
		fmt.Fprintln(w, "No. Streetlight requests have little relationship with night outdoor crime.")
	case rhoShare >= 0.3:
		fmt.Fprintf(w, "Yes. They have more night outdoor crime, and a clearly larger share of their crime\n"+
			"happens outdoors at night (%.1f%% in the lowest streetlight quartile vs %.1f%% in the highest).\n", lo, hi)
	case rhoShare >= 0.1:
		fmt.Fprintf(w, "Somewhat. They have more night outdoor crime, but mostly because they have more crime\n"+
			"overall (the count correlation matches the all-crime baseline). The night outdoor share rises\n"+
			"only slightly, from %.1f%% in the lowest streetlight quartile to %.1f%% in the highest.\n", lo, hi)
	default:
		fmt.Fprintln(w, "Not specifically. They have more night outdoor crime only because they have more crime overall;\n"+
			"the share of crime that happens outdoors at night does not rise with streetlight requests.")
	}
	return nil
}
