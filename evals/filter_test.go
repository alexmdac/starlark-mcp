package main

import (
	"strings"
	"testing"
)

var testCases = []evalCase{
	{name: "print_numbers", tier: 1},
	{name: "reverse_string", tier: 1},
	{name: "fizzbuzz", tier: 2},
	{name: "matrix_multiply", tier: 4},
	{name: "spiral_matrix", tier: 4},
	{name: "game_of_life", tier: 5},
}

func TestFilterCases_NoFilter(t *testing.T) {
	got, err := filterCases(testCases, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(testCases) {
		t.Errorf("got %d cases, want %d", len(got), len(testCases))
	}
}

func TestFilterCases_GlobOnly(t *testing.T) {
	got, err := filterCases(testCases, "*matrix*", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d cases, want 2", len(got))
	}
	for _, c := range got {
		if c.name != "matrix_multiply" && c.name != "spiral_matrix" {
			t.Errorf("unexpected case: %s", c.name)
		}
	}
}

func TestFilterCases_ExactName(t *testing.T) {
	got, err := filterCases(testCases, "fizzbuzz", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].name != "fizzbuzz" {
		t.Errorf("got %v, want [fizzbuzz]", got)
	}
}

func TestFilterCases_SingleTier(t *testing.T) {
	got, err := filterCases(testCases, "", "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d cases, want 2", len(got))
	}
	for _, c := range got {
		if c.tier != 1 {
			t.Errorf("unexpected tier %d for %s", c.tier, c.name)
		}
	}
}

func TestFilterCases_TierRange(t *testing.T) {
	got, err := filterCases(testCases, "", "1-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d cases, want 3", len(got))
	}
}

func TestFilterCases_TierAndGlob(t *testing.T) {
	got, err := filterCases(testCases, "*matrix*", "4")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d cases, want 2", len(got))
	}

	// Glob matches tier-4 cases only
	got, err = filterCases(testCases, "*matrix*", "1-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d cases, want 0 (no matrix cases in tiers 1-2)", len(got))
	}
}

func TestFilterCases_NoMatch(t *testing.T) {
	got, err := filterCases(testCases, "nonexistent", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d cases, want 0", len(got))
	}
}

func TestParseTierSpec(t *testing.T) {
	tests := []struct {
		input   string
		wantMin int
		wantMax int
		wantErr string // empty means no error expected
	}{
		{"", 0, 0, ""},
		{"3", 3, 3, ""},
		{"1-4", 1, 4, ""},
		{"0", 0, 0, `bad tier "0": tiers must be >= 1`},
		{"-1", 0, 0, `bad tier "-1": expected N or N-M where N,M >= 1`},
		{"0-2", 0, 0, `bad tier range "0-2": tiers must be >= 1`},
		{"1--3", 0, 0, `bad tier "1--3": expected N or N-M where N,M >= 1`},
		{"-1-2", 0, 0, `bad tier "-1-2": expected N or N-M where N,M >= 1`},
		{"-1--2", 0, 0, `bad tier "-1--2": expected N or N-M where N,M >= 1`},
		{"4-1", 0, 0, `bad tier range "4-1": min > max`},
		{"abc", 0, 0, `bad tier "abc": expected N or N-M where N,M >= 1`},
		{"1-abc", 0, 0, `bad tier "1-abc": expected N or N-M where N,M >= 1`},
		{"abc-3", 0, 0, `bad tier "abc-3": expected N or N-M where N,M >= 1`},
	}
	for _, tt := range tests {
		min, max, err := parseTierSpec(tt.input)
		if tt.wantErr == "" {
			if err != nil {
				t.Errorf("parseTierSpec(%q) unexpected error: %v", tt.input, err)
				continue
			}
		} else {
			if err == nil {
				t.Errorf("parseTierSpec(%q) expected error containing %q, got nil", tt.input, tt.wantErr)
				continue
			}
			if got := err.Error(); !strings.Contains(got, tt.wantErr) {
				t.Errorf("parseTierSpec(%q) error = %q, want containing %q", tt.input, got, tt.wantErr)
				continue
			}
		}
		if min != tt.wantMin || max != tt.wantMax {
			t.Errorf("parseTierSpec(%q) = (%d, %d), want (%d, %d)", tt.input, min, max, tt.wantMin, tt.wantMax)
		}
	}
}
