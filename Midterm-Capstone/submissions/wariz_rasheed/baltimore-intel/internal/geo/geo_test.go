package geo

import "testing"

func TestBlockKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"5300 SINCLAIR LN", "5300 SINCLAIR LN"},
		{"800 N CHARLES ST", "800 N CHARLES ST"},
		{"4410 NORTHERN PARKWAY WEST", "4400 W NORTHERN PKWY"},
		{"1724-26 CHARLES STREET NORTH", "1700 N CHARLES ST"},
		{"3954 DOLFIELD AVE, Baltimore City, 21215", "3900 DOLFIELD AVE"},
		{"700 BLK W LOMBARD ST, Baltimore City, 21201", "700 W LOMBARD ST"},
		{"  1263 Fayette Street East ", "1200 E FAYETTE ST"},
		{"", ""},
		{"CHARLES ST", ""},
	}
	for _, c := range cases {
		if got := BlockKey(c.in); got != c.want {
			t.Errorf("BlockKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBlockKeyMatchesAcrossDatasets(t *testing.T) {
	// The same physical block written the way each dataset writes it.
	crime := BlockKey("1700 N CHARLES ST")
	license := BlockKey("1724-26 CHARLES STREET NORTH")
	request := BlockKey("1750 N CHARLES ST, Baltimore City, 21201")
	if crime != license || crime != request {
		t.Errorf("keys differ: crime=%q license=%q 311=%q", crime, license, request)
	}
}

func TestCanonHood(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Patterson Park Neighborhood", "PATTERSON PARK"},
		{"  mount   vernon ", "MOUNT VERNON"},
		{"DOWNTOWN", "DOWNTOWN"},
		{"N/A", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := CanonHood(c.in); got != c.want {
			t.Errorf("CanonHood(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIndexMajorityAndTieBreak(t *testing.T) {
	v := NewVotes()
	v.Add("100 A ST", "CANTON")
	v.Add("100 A ST", "CANTON")
	v.Add("100 A ST", "FELLS POINT")
	v.Add("200 B ST", "ZEBRA")
	v.Add("200 B ST", "ALPHA") // tie: alphabetical winner
	v.Add("", "IGNORED")
	other := NewVotes()
	other.Add("100 A ST", "FELLS POINT")
	other.Add("100 A ST", "FELLS POINT")
	v.Merge(other) // FELLS POINT now 3 vs CANTON 2

	ix := BuildIndex(v)
	if got := ix.Lookup("100 A ST"); got != "FELLS POINT" {
		t.Errorf("100 A ST = %q, want FELLS POINT", got)
	}
	if got := ix.Lookup("200 B ST"); got != "ALPHA" {
		t.Errorf("200 B ST = %q, want ALPHA", got)
	}
	if got := ix.Lookup("nope"); got != "" {
		t.Errorf("unknown block = %q, want empty", got)
	}
	if ix.Len() != 2 {
		t.Errorf("Len = %d, want 2", ix.Len())
	}
	var nilIndex *Index
	if nilIndex.Lookup("x") != "" || nilIndex.Len() != 0 {
		t.Error("nil index must behave as empty")
	}
}
