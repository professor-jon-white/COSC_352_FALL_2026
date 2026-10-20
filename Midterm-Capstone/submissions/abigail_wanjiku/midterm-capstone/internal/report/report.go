// Package report renders pipeline results for people to read: the ingest
// summary, and the analysis questions, each implemented as a Report strategy.
package report

import (
	"fmt"
	"io"

	"capstone/internal/model"
)

// Matched writes one line describing the neighborhood-month records.
func Matched(w io.Writer, summaries []model.Summary) error {
	neighborhoods := make(map[string]struct{})
	bad := 0
	for _, s := range summaries {
		neighborhoods[s.Key.Neighborhood] = struct{}{}
		bad += s.BadCloseTimes
	}
	_, err := fmt.Fprintf(w, "%d neighborhood-month records across %d neighborhoods (%d 311 close times excluded as before creation)\n",
		len(summaries), len(neighborhoods), bad)
	return err
}
