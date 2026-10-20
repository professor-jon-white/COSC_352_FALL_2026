package processor

import "strings"

// IsClosed reports whether a 311 SRStatus is a closed state
// ("Closed", "Closed (Duplicate)", "Closed (Transferred)", ...).
func IsClosed(status string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(status)), "closed")
}

// IsSanitation reports whether an SRType is a sanitation / illegal dumping /
// dirty-property request (e.g. "SW-Dirty Alley", "HCD-Illegal Dumping").
func IsSanitation(srType string) bool {
	t := strings.ToLower(srType)
	for _, s := range []string{"dirty", "dumping", "mixed refuse", "sanitation", "cleaning", "sw-siu", "hgw", "trash"} {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}

// IsEnforcement reports whether an SRType is a code-enforcement request:
// anything routed to Housing & Community Development (HCD-*), plus
// sanitation/dumping complaints.
func IsEnforcement(srType string) bool {
	return strings.HasPrefix(strings.ToUpper(srType), "HCD-") || IsSanitation(srType)
}

// Category buckets an SRType by its agency prefix.
func Category(srType string) string {
	switch {
	case IsSanitation(srType):
		return "Sanitation/Dumping"
	case strings.HasPrefix(srType, "HCD-"):
		return "Housing Code"
	case strings.HasPrefix(srType, "TR"):
		return "Transportation/Streets"
	case strings.HasPrefix(srType, "WW"):
		return "Water/Sewer"
	default:
		return "Other"
	}
}
