// Package geo turns free-text street addresses into comparable "block keys"
// and maintains the block -> neighborhood index used to join datasets.
//
// The three datasets do not share an ID. Crime and 311 carry neighborhoods but
// the liquor-license file only has a street address. All three do have a
// street address, so we normalise every address to its 100-block
// ("4410 NORTHERN PARKWAY WEST" -> "4400 W NORTHERN PKWY") and join on that.
package geo

import (
	"strconv"
	"strings"
)

var streetTypes = map[string]string{
	"STREET": "ST", "AVENUE": "AVE", "ROAD": "RD", "DRIVE": "DR", "LANE": "LN",
	"BOULEVARD": "BLVD", "COURT": "CT", "PLACE": "PL", "TERRACE": "TER",
	"HIGHWAY": "HWY", "PARKWAY": "PKWY", "CIRCLE": "CIR", "SQUARE": "SQ",
}

var directions = map[string]string{
	"NORTH": "N", "SOUTH": "S", "EAST": "E", "WEST": "W",
}

func isDirection(t string) bool {
	return t == "N" || t == "S" || t == "E" || t == "W"
}

// BlockKey normalises an address to "<hundred-block> <DIR> <STREET> <TYPE>".
// It returns "" when the address has no usable house number.
func BlockKey(addr string) string {
	if i := strings.IndexByte(addr, ','); i >= 0 {
		addr = addr[:i] // 311 addresses look like "3954 DOLFIELD AVE, Baltimore City, 21215"
	}
	addr = strings.ToUpper(addr)
	addr = strings.Map(func(r rune) rune {
		if r == '.' || r == '#' {
			return -1
		}
		return r
	}, addr)
	fields := strings.Fields(addr)
	if len(fields) < 2 {
		return ""
	}
	num := fields[0]
	if i := strings.IndexByte(num, '-'); i > 0 {
		num = num[:i] // license ranges such as "1724-26"
	}
	n, err := strconv.Atoi(num)
	if err != nil || n < 0 {
		return ""
	}
	toks := make([]string, 0, len(fields)-1)
	for _, f := range fields[1:] {
		if f == "BLK" || f == "BLOCK" {
			continue
		}
		if d, ok := directions[f]; ok {
			f = d
		} else if t, ok := streetTypes[f]; ok {
			f = t
		}
		toks = append(toks, f)
	}
	if len(toks) == 0 {
		return ""
	}
	// Licenses write the direction last ("CHARLES STREET NORTH"); crime and 311
	// write it first ("N CHARLES ST"). Move a trailing direction to the front.
	if len(toks) >= 2 && isDirection(toks[len(toks)-1]) && !isDirection(toks[0]) {
		toks = append([]string{toks[len(toks)-1]}, toks[:len(toks)-1]...)
	}
	return strconv.Itoa(n/100*100) + " " + strings.Join(toks, " ")
}

// CanonHood upper-cases a neighborhood name and strips the " NEIGHBORHOOD"
// suffix some 311 rows carry, so names from different datasets compare equal.
func CanonHood(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, " NEIGHBORHOOD")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || s == "N/A" || s == "NA" || s == "UNKNOWN" {
		return ""
	}
	return s
}

// Votes counts, for each block, how often each neighborhood was reported.
type Votes map[string]map[string]int32

// NewVotes returns an empty vote table.
func NewVotes() Votes {
	return make(Votes)
}

// Add records one observation that block lies in hood.
func (v Votes) Add(block, hood string) {
	if block == "" || hood == "" {
		return
	}
	m, ok := v[block]
	if !ok {
		m = make(map[string]int32, 1)
		v[block] = m
	}
	m[hood]++
}

// Merge adds another vote table into v.
func (v Votes) Merge(o Votes) {
	for b, hoods := range o {
		m, ok := v[b]
		if !ok {
			m = make(map[string]int32, len(hoods))
			v[b] = m
		}
		for h, n := range hoods {
			m[h] += n
		}
	}
}

// Index maps a street block to its most frequently reported neighborhood.
// It is immutable after BuildIndex, so any number of goroutines may call
// Lookup concurrently without synchronisation.
type Index struct {
	m map[string]string
}

// BuildIndex resolves each block to its majority neighborhood. Ties break
// alphabetically so results are identical from run to run.
func BuildIndex(v Votes) *Index {
	ix := &Index{m: make(map[string]string, len(v))}
	for b, hoods := range v {
		best, bestN := "", int32(0)
		for h, n := range hoods {
			if n > bestN || (n == bestN && h < best) {
				best, bestN = h, n
			}
		}
		ix.m[b] = best
	}
	return ix
}

// Lookup returns the neighborhood for a block key, or "" if unknown.
func (ix *Index) Lookup(block string) string {
	if ix == nil || block == "" {
		return ""
	}
	return ix.m[block]
}

// Len returns the number of blocks in the index.
func (ix *Index) Len() int {
	if ix == nil {
		return 0
	}
	return len(ix.m)
}
