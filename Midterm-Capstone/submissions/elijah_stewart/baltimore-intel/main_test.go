package main

import "testing"

func TestChooseUsesFirstNonEmptyField(t *testing.T) {
	r := map[string]string{"srtype": "", "service_type": "Trash"}
	if got := choose(r, "srtype", "service_type"); got != "Trash" {
		t.Fatalf("got %q", got)
	}
}
func TestTopSortsByCountThenName(t *testing.T) {
	got := top(map[string]int64{"Pothole": 2, "Trash": 3, "Water": 2}, 3)
	if len(got) != 3 || got[0].Name != "Trash" || got[1].Name != "Pothole" || got[2].Name != "Water" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
