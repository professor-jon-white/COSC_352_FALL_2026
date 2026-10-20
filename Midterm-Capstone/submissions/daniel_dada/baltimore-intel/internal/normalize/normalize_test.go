package normalize

import "testing"

func TestKey(t *testing.T) {
	cases := map[string]string{
		"Penn-North":      "PENN NORTH",
		"  penn   north ": "PENN NORTH",
		"PENN NORTH.":     "PENN NORTH",
		"Butcher's Hill":  "BUTCHERS HILL",
		"":                "",
		"   ":             "",
		"Mt. Vernon":      "MT VERNON",
	}
	for in, want := range cases {
		if got := Key(in); got != want {
			t.Errorf("Key(%q) = %q, want %q", in, got, want)
		}
	}
}
