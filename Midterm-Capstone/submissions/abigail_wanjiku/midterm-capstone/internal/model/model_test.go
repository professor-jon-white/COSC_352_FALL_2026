package model

import (
	"testing"
	"unsafe"
)

// TestRecordSizes pins the in-memory size of the per-row structs, which are
// held once per kept CSV row (about 711,000 rows), so an added field shows up
// as a deliberate change. Sizes are for 64-bit platforms.
func TestRecordSizes(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("sizes below are for 64-bit platforms")
	}
	if got := unsafe.Sizeof(Crime{}); got != 64 {
		t.Errorf("Crime is %d bytes, want 64 (time 24 + 2 strings 32 + bool padded to 8)", got)
	}
	if got := unsafe.Sizeof(ServiceRequest{}); got != 96 {
		t.Errorf("ServiceRequest is %d bytes, want 96 (2 times 48 + 3 strings 48)", got)
	}
}
