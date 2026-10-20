package model

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"
)

func TestRecordSizes(t *testing.T) {
	// Guard the memory budget documented in the README.
	if s := unsafe.Sizeof(Crime{}); s != 24 {
		t.Errorf("Crime is %d bytes, want 24", s)
	}
	if s := unsafe.Sizeof(ServiceRequest{}); s != 40 {
		t.Errorf("ServiceRequest is %d bytes, want 40", s)
	}
}

func TestNormalizeNeighborhood(t *testing.T) {
	cases := map[string]string{
		"Belair-Edison":              "BELAIR EDISON",
		"BELAIR EDISON":              "BELAIR EDISON",
		"  belair  -  edison ":       "BELAIR EDISON",
		"Ellwood Park/Monument":      "ELLWOOD PARK MONUMENT",
		"Butcher's Hill":             "BUTCHERS HILL",
		"Washington Village/Pigtown": "WASHINGTON VILLAGE PIGTOWN",
		"UNKNOWN":                    "",
		"":                           "",
	}
	for in, want := range cases {
		if got := NormalizeNeighborhood(in); got != want {
			t.Errorf("NormalizeNeighborhood(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInternerJoinsSpellingsAndPrefersMixedCase(t *testing.T) {
	in := NewInterner(NormalizeNeighborhood)
	a := in.ID("MCELDERRY PARK")
	b := in.ID("McElderry Park")
	if a != b || a == 0 {
		t.Fatalf("spellings not joined: %d vs %d", a, b)
	}
	if got := in.Name(a); got != "McElderry Park" {
		t.Errorf("display = %q, want McElderry Park", got)
	}
	if in.ID("") != 0 || in.ID("unknown") != 0 {
		t.Error("blank/unknown must map to 0")
	}
}

func TestInternerConcurrent(t *testing.T) {
	in := NewInterner(NormalizeLabel)
	var wg sync.WaitGroup
	ids := make([][]ID, 16)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				ids[g] = append(ids[g], in.ID(fmt.Sprintf("type %d", i%100)))
			}
		}(g)
	}
	wg.Wait()
	if in.Len() != 101 { // 100 labels + reserved 0
		t.Fatalf("Len = %d, want 101", in.Len())
	}
	for g := 1; g < 16; g++ {
		for i := range ids[0] {
			if ids[g][i] != ids[0][i] {
				t.Fatalf("goroutine %d got different ID for the same label", g)
			}
		}
	}
}
