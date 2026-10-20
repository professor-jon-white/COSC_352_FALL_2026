package report

import (
	"fmt"
	"io"

	"capstone/internal/model"
	"capstone/internal/stats"
)

// SanitationVsCrime asks whether the neighborhoods with the most sanitation
// and dumping complaints are also the ones with the most crime.
//
// It ranks neighborhoods by sanitation requests and by crime, reports the
// Spearman rank correlation between the two, and compares it with the
// correlation between all resident 311 requests and crime. That baseline
// matters: bigger, busier neighborhoods have more of everything, so
// sanitation is only a distinctive signal if it tracks crime more closely
// than 311 volume in general does.
type SanitationVsCrime struct{}

func (SanitationVsCrime) ID() string { return "sanitation" }
func (SanitationVsCrime) Question() string {
	return "Do neighborhoods with the most sanitation/dumping 311 requests also have the highest crime?"
}

func (SanitationVsCrime) Render(w io.Writer, in *Input) error {
	rs := rows(in,
		func(s *model.Summary) float64 { return float64(countTypes(s.RequestsByType, isSanitation)) },
		func(s *model.Summary) float64 { return float64(s.CrimeCount) },
	)
	crimeRank := rankOf(rs, byY)
	top := sortedBy(rs, byX)[:min(10, len(rs))]
	maxX, maxY := top[0].x, sortedBy(rs, byY)[0].y

	t := newTable(w, "#", "neighborhood", "sanitation 311", "", "crimes", "crime rank", "")
	inTop10, inTopQuartile := 0, 0
	for i, r := range top {
		cr := crimeRank[r.name]
		if cr <= 10 {
			inTop10++
		}
		if cr <= len(rs)/4 {
			inTopQuartile++
		}
		t.row(i+1, r.name, num(int(r.x)), bar(r.x, maxX), num(int(r.y)), fmt.Sprintf("%d / %d", cr, len(rs)), bar(r.y, maxY))
	}
	if err := t.flush(); err != nil {
		return err
	}

	rho := stats.Spearman(xs(rs), ys(rs))
	all := rows(in,
		func(s *model.Summary) float64 { return float64(residentRequests(s)) },
		func(s *model.Summary) float64 { return float64(s.CrimeCount) },
	)
	baseline := stats.Spearman(xs(all), ys(all))

	fmt.Fprintf(w, "\nSpearman rank correlation across %d neighborhoods:\n", len(rs))
	fmt.Fprintf(w, "  sanitation 311 vs crime     %s\n", corr(rho))
	fmt.Fprintf(w, "  all resident 311 vs crime   %s   (baseline)\n", corr(baseline))
	fmt.Fprintf(w, "\nOf the top 10 sanitation neighborhoods, %d are also top 10 for crime and %d are in the top quarter.\n", inTop10, inTopQuartile)

	verdict := "Partly"
	switch {
	case rho >= 0.5 && inTopQuartile >= 7:
		verdict = "Yes"
	case rho < 0.3:
		verdict = "Not really"
	}
	fmt.Fprintf(w, "Answer: %s (rank correlation %+.2f).\n", verdict, rho)
	switch {
	case rho > baseline+0.05:
		fmt.Fprintln(w, "Sanitation complaints track crime more closely than 311 volume overall, so they are a distinctive signal.")
	case rho < baseline-0.05:
		fmt.Fprintln(w, "But it is not specific to sanitation: overall 311 volume tracks crime even more closely,")
		fmt.Fprintln(w, "so this mostly reflects busier neighborhoods having more of both.")
	default:
		fmt.Fprintln(w, "Sanitation complaints track crime about as closely as 311 volume overall.")
	}
	return nil
}
