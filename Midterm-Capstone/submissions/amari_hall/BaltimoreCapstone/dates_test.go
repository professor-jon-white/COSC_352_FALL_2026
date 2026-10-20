package main

import "testing"

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026/01/03 14:29:00+00", "2026-01"},
		{"10/11/2026 5:00:00 AM", "2026-10"},
		{"10/11/2026 6:00:00 AM", "2026-10"},
	}

	for _, test := range tests {
		actual, err := parseMonth(test.input)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if actual != test.expected {
			t.Errorf(
				"Input %s: expected %s, got %s",
				test.input,
				test.expected,
				actual,
			)
		}
	}
}

func TestInvalidDate(t *testing.T) {
	_, err := parseMonth("invalid-date")

	if err == nil {
		t.Error("Expected an error for an invalid date")
	}
}
