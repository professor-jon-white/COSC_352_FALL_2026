package model

import (
	"testing"
	"time"
)

func ms(s string) *int64 { v, _ := time.Parse(time.RFC3339, s); n := v.UnixMilli(); return &n }
func TestNormalize(t *testing.T) {
	if got := Normalize("  Broadway \t east "); got != "BROADWAY EAST" {
		t.Fatal(got)
	}
}
func TestKeys(t *testing.T) {
	a := Attributes{RequestID: " a ", BlockLot: " 123  456 ", Address: "100 Main"}
	if a.Key("requests") != "A" || a.Key("vacancies") != "parcel:123 456" {
		t.Fatal(a)
	}
	a.BlockLot = ""
	if a.Key("vacancies") != "address:100 MAIN" {
		t.Fatal(a)
	}
	a.Address = ""
	if a.Key("vacancies") != "" {
		t.Fatal("missing identity not detected")
	}
}
func TestPeriodBoundaries(t *testing.T) {
	for _, tc := range []struct {
		date string
		want bool
	}{
		{"2025-09-30T23:59:59Z", false}, {"2025-10-01T00:00:00Z", true},
		{"2025-12-31T23:59:59Z", true}, {"2026-01-01T00:00:00Z", false},
	} {
		if InPeriod(*ms(tc.date)) != tc.want {
			t.Fatal(tc)
		}
	}
}
func TestClosure(t *testing.T) {
	a := Attributes{Created: ms("2025-10-01T00:00:00Z"), Closed: ms("2025-10-03T12:00:00Z"), Status: "Closed (Transferred)"}
	if days, ok := ClosedDays(a); !ok || days != 2.5 {
		t.Fatal(days, ok)
	}
	a.Status = "Open"
	if _, ok := ClosedDays(a); ok {
		t.Fatal("open status accepted")
	}
	a.Status = "Closed"
	a.Closed = ms("2025-09-01T00:00:00Z")
	if _, ok := ClosedDays(a); ok {
		t.Fatal("negative duration accepted")
	}
	a.Closed = nil
	if _, ok := ClosedDays(a); ok {
		t.Fatal("missing closure accepted")
	}
}
