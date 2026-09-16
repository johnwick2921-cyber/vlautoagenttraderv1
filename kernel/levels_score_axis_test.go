package kernel

import (
	"os"
	"testing"
)

// C14 (2026-08-25) — the confluence cap is a USER knob (env CONFLUENCE_CAP),
// never a hardcode. Default 3; invalid/absent values are safe.
func TestConfluenceCapKnob(t *testing.T) {
	os.Unsetenv("CONFLUENCE_CAP")
	if got := ConfluenceCap(); got != 3 {
		t.Fatalf("default cap = %d, want 3", got)
	}
	os.Setenv("CONFLUENCE_CAP", "2")
	defer os.Unsetenv("CONFLUENCE_CAP")
	if got := ConfluenceCap(); got != 2 {
		t.Fatalf("env cap = %d, want 2", got)
	}
	os.Setenv("CONFLUENCE_CAP", "not-a-number")
	if got := ConfluenceCap(); got != 3 {
		t.Fatalf("malformed env must fall back to 3, got %d", got)
	}
	os.Setenv("CONFLUENCE_CAP", "-5")
	if got := ConfluenceCap(); got != 3 {
		t.Fatalf("negative env must fall back to 3, got %d", got)
	}
}

// C13 (2026-08-25) — the zone-size axis: tight bases outscore oversized ones,
// banded 0.5..1.25 in daily-ATR units; line levels (no Lo/Hi) are neutral.
func TestZoneSizeMultBands(t *testing.T) {
	cases := []struct {
		name string
		lo   float64
		hi   float64
		atr  float64
		want float64
	}{
		{"tight base", 100.0, 100.2, 2.0, 1.25}, // 0.10×ATR
		{"small base", 100.0, 101.0, 2.0, 1.10}, // 0.50×ATR
		{"normal base", 100.0, 101.5, 2.0, 1.0}, // 0.75×ATR
		{"wide base", 100.0, 103.0, 2.0, 0.85},  // 1.50×ATR
		{"oversized base", 100.0, 105.0, 2.0, 0.70},
		{"huge base", 100.0, 108.0, 2.0, 0.50},
		{"line level neutral", 0, 0, 2.0, 1.0},
		{"no ATR neutral", 100.0, 101.0, 0, 1.0},
	}
	for _, tc := range cases {
		if got := zoneSizeMult(tc.lo, tc.hi, tc.atr); got != tc.want {
			t.Fatalf("%s: zoneSizeMult(%v,%v,%v) = %v, want %v", tc.name, tc.lo, tc.hi, tc.atr, got, tc.want)
		}
	}
}
