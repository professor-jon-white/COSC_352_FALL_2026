package normalize

import (
	"strings"
	"unicode"
)

// Key turns "Penn-North", " penn  north " and "PENN NORTH." into "PENN NORTH".
func Key(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.NewReplacer("'", "", "’", "").Replace(s)
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
		} else if !lastSpace && b.Len() > 0 {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}
