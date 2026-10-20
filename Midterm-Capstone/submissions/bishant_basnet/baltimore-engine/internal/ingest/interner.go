package ingest

import (
	"strings"
	"sync"
)

// Interner deduplicates strings across goroutines.
//
// Why this matters: encoding/csv allocates ONE string per line and returns
// each field as a substring of it. Keeping even a short field (say, a
// neighborhood name) therefore keeps the entire line alive on the heap.
// Interning clones each distinct value once, so the raw lines become garbage
// immediately and low-cardinality fields (neighborhood, weapon, SR type)
// cost one allocation total instead of one per row.
type Interner struct{ m sync.Map }

func NewInterner() *Interner { return &Interner{} }

func (in *Interner) Intern(s string) string {
	if s == "" {
		return ""
	}
	if v, ok := in.m.Load(s); ok {
		return v.(string)
	}
	c := strings.Clone(s)
	v, _ := in.m.LoadOrStore(c, c)
	return v.(string)
}

// NormalizeName upper-cases and collapses whitespace so "Fells  point" and
// "FELLS POINT" join correctly across datasets.
func NormalizeName(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}
