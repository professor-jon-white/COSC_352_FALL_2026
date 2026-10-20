package analysis

import (
	"fmt"
	"testing"

	"baltimoreintel/internal/model"
)

// syntheticDataset builds a small, fully deterministic dataset so every
// analysis runs its normal (non-fallback) code path.
func syntheticDataset() *model.Dataset {
	acc := model.NewAcc()
	for i := 0; i < 16; i++ {
		h := acc.Hood(fmt.Sprintf("HOOD %02d", i))
		h.Crimes = 20 + i*5
		h.Violent = 5 + i*3
		h.TimedViolent = h.Violent
		h.NightViolent = i
		h.Requests = 100 + i*10
		h.Open = i
		h.ReqByType["SW-Dirty Alley"] = 30 + i*4
		h.ReqByType["TRS-Parking Complaint"] = 50 - i
		h.Licenses = i % 8
		for k := 0; k < 25; k++ {
			h.CloseHours = append(h.CloseHours, float64(24*(1+i%5)+k))
		}
	}
	for i := 0; i < 40; i++ {
		b := acc.Block(fmt.Sprintf("%d MAIN ST", i*100))
		b.Crimes = i % 7
		b.Violent = i % 4
		b.NightViolent = i % 2
		if i%4 == 0 {
			b.Kinds = []string{"TAVERN"}
		}
		if i%5 == 0 {
			b.Kinds = append(b.Kinds, "RESTAURANT")
		}
	}
	return &model.Dataset{Hoods: acc.Hoods, Blocks: acc.Blocks}
}

func TestAllAnalysesProduceOutput(t *testing.T) {
	d := syntheticDataset()
	list := All()
	if len(list) != 5 {
		t.Fatalf("expected 5 analyses, got %d", len(list))
	}
	results := RunAll(d, list)
	for i, r := range results {
		want := fmt.Sprintf("Q%d", i+1)
		if r.ID != want || list[i].ID() != want {
			t.Errorf("result %d has ID %q / %q, want %q (order must be preserved)", i, r.ID, list[i].ID(), want)
		}
		if r.Title == "" || r.Audience == "" || r.Question == "" || r.Finding == "" {
			t.Errorf("%s has empty text fields: %+v", want, r)
		}
		if len(r.Charts) == 0 || len(r.Tables) == 0 {
			t.Errorf("%s produced no chart or table (finding: %s)", want, r.Finding)
		}
	}
}

func TestPriorityScoresAreSorted(t *testing.T) {
	r := PriorityIndex{W: DefaultWeights()}.Run(syntheticDataset())
	if len(r.Charts) != 1 || len(r.Charts[0].Bars) < 2 {
		t.Fatalf("expected one chart with several bars, got %+v", r.Charts)
	}
	bars := r.Charts[0].Bars
	for i := 1; i < len(bars); i++ {
		if bars[i].Value > bars[i-1].Value {
			t.Errorf("scores not descending at %d: %v then %v", i, bars[i-1].Value, bars[i].Value)
		}
	}
	for _, b := range bars {
		if b.Value < 0 || b.Value > 100 {
			t.Errorf("score %v outside 0-100", b.Value)
		}
	}
}

func TestEmptyDatasetDoesNotPanic(t *testing.T) {
	d := &model.Dataset{Hoods: map[string]*model.HoodAgg{}, Blocks: map[string]*model.BlockAgg{}}
	for _, r := range RunAll(d, All()) {
		if r.Finding == "" {
			t.Errorf("%s: empty dataset should still explain itself", r.ID)
		}
	}
}

func TestHelpers(t *testing.T) {
	if got := commas(1234567); got != "1,234,567" {
		t.Errorf("commas = %q", got)
	}
	if got := commas(999); got != "999" {
		t.Errorf("commas = %q", got)
	}
	if got := pretty("BUTCHER'S HILL"); got != "Butcher's Hill" {
		t.Errorf("pretty = %q", got)
	}
	if got := trunc("abcdefghij", 6); got != "abc..." {
		t.Errorf("trunc = %q", got)
	}
	if got := pct(1, 4); got != "25.0%" {
		t.Errorf("pct = %q", got)
	}
}
