package report

import (
	"fmt"
	"io"
	"text/tabwriter"

	"capstone/internal/ingest"
)

// Dataset names one input file and the stats from loading it.
type Dataset struct {
	Name  string
	Stats ingest.Stats
}

// Ingest writes a side-by-side table of ingest stats, the number of
// neighborhoods the datasets share, and any malformed-row examples.
func Ingest(w io.Writer, year int, datasets ...Dataset) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Ingest (%d)\t", year)
	for _, d := range datasets {
		fmt.Fprintf(tw, "%9s\t", d.Name)
	}
	fmt.Fprintln(tw)

	rows := []struct {
		label string
		value func(*ingest.Stats) int
	}{
		{"rows read", func(s *ingest.Stats) int { return s.Read }},
		{"rows kept", func(s *ingest.Stats) int { return s.Kept }},
		{"rows skipped", (*ingest.Stats).Skipped},
		{"  outside year", func(s *ingest.Stats) int { return s.OutOfYear }},
		{"  no neighborhood", func(s *ingest.Stats) int { return s.NoNeighborhood }},
		{"  malformed", func(s *ingest.Stats) int { return s.Malformed }},
		{"distinct neighborhoods", (*ingest.Stats).Neighborhoods},
	}
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t", r.label)
		for i := range datasets {
			fmt.Fprintf(tw, "%9d\t", r.value(&datasets[i].Stats))
		}
		fmt.Fprintln(tw)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if len(datasets) == 2 {
		shared := 0
		other := datasets[1].Stats.NeighborhoodSet()
		for n := range datasets[0].Stats.NeighborhoodSet() {
			if _, ok := other[n]; ok {
				shared++
			}
		}
		fmt.Fprintf(w, "\nneighborhoods in both: %d\n", shared)
	}

	for _, d := range datasets {
		for _, ex := range d.Stats.Examples {
			fmt.Fprintf(w, "malformed %s %s\n", d.Name, ex)
		}
	}
	return nil
}
