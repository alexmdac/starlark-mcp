package main

import (
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
		wantErr bool
	}{
		{"", 0, 0, false},
		{"3", 3, 3, false},
		{"1-4", 1, 4, false},
		{"4-1", 0, 0, true},
		{"abc", 0, 0, true},
		{"1-abc", 0, 0, true},
		{"abc-3", 0, 0, true},
	}
	for _, tt := range tests {
		min, max, err := parseTierSpec(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseTierSpec(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if min != tt.wantMin || max != tt.wantMax {
			t.Errorf("parseTierSpec(%q) = (%d, %d), want (%d, %d)", tt.input, min, max, tt.wantMin, tt.wantMax)
		}
	}
}
