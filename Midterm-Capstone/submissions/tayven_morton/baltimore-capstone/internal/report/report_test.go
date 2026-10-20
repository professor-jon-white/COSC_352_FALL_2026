package report

import (
	"baltimore-capstone/internal/analysis"
	"baltimore-capstone/internal/model"
	"bytes"
	"errors"
	"strings"
	"testing"
)

type badWriter struct{}

func (badWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestReportSectionsAndErrors(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, nil, model.Quality{}, model.Quality{}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"## Q1:", "## Q2:", "## Q3:", "## Q4:", "## Q5:", "Data quality", "not a reconstruction"} {
		if !strings.Contains(b.String(), s) {
			t.Fatal("missing", s)
		}
	}
	if err := Write(badWriter{}, nil, model.Quality{}, model.Quality{}); err == nil {
		t.Fatal("writer error lost")
	}
}
func TestCSV(t *testing.T) {
	var b bytes.Buffer
	s := model.NewStats()
	s.Count = 3
	err := CSV(&b, []analysis.Row{{Name: "A, B", Requests: s, Group: "unmatched"}})
	if err != nil || !strings.Contains(b.String(), `"A, B",3,0,false,unmatched,`) {
		t.Fatal(err, b.String())
	}
}
