package analysis

import (
	"baltimore-capstone/internal/model"
	"reflect"
	"testing"
)

func TestMedian(t *testing.T) {
	for _, tc := range []struct {
		v    []float64
		want float64
	}{{[]float64{3, 1, 2}, 2}, {[]float64{4, 1}, 2.5}} {
		before := append([]float64(nil), tc.v...)
		got, ok := Median(tc.v)
		if !ok || got != tc.want || !reflect.DeepEqual(before, tc.v) {
			t.Fatal(got)
		}
	}
	if _, ok := Median(nil); ok {
		t.Fatal("empty median")
	}
}
func TestTiedPercentiles(t *testing.T) {
	if p := Percentile([]int{1, 2, 2, 3}, 2); p != 50 {
		t.Fatal(p)
	}
	if p := Percentile([]int{5}, 5); p != 50 {
		t.Fatal(p)
	}
}
func TestJoinCohortsAndConcurrency(t *testing.T) {
	r := model.Dataset{Areas: map[string]*model.Stats{}}
	v := model.Dataset{Areas: map[string]*model.Stats{}}
	for i, k := range []string{"A", "B", "C", "D", "E", "F"} {
		r.Areas[k] = model.NewStats()
		r.Areas[k].Count = (i + 1) * 10
		v.Areas[k] = model.NewStats()
		v.Areas[k].Count = i + 1
	}
	r.Areas["ONLY REQUESTS"] = model.NewStats()
	r.Areas["ONLY REQUESTS"].Count = 20
	v.Areas["ONLY VACANCIES"] = model.NewStats()
	v.Areas["ONLY VACANCIES"].Count = 5
	one, many := Join(r, v, 1), Join(r, v, 8)
	if !reflect.DeepEqual(one, many) {
		t.Fatal("nondeterministic join")
	}
	if len(one) != 8 {
		t.Fatal(len(one))
	}
	low, nl := Cohort(one, "low")
	high, nh := Cohort(one, "high")
	if nl != 2 || nh != 2 || low.Count != 30 || high.Count != 110 {
		t.Fatal(nl, nh, low.Count, high.Count)
	}
	for _, row := range one {
		if !row.Matched && row.Group != "unmatched" {
			t.Fatal(row)
		}
	}
	if one[5].Score <= one[0].Score {
		t.Fatal("ranking")
	}
}
func TestTopTies(t *testing.T) {
	p := Top(map[string]int{"Z": 2, "A": 2, "C": 1}, 2)
	if len(p) != 2 || p[0].Name != "A" || p[1].Name != "Z" {
		t.Fatal(p)
	}
}
