package store

import "testing"

// DEFAULTS-SANE (2026-09-24): the shipped Picture defaults must be reachable by
// the real pipeline (2-minute trader cadence, hour-boundary frame storms). The
// resolver is the single canonicalizer — the live path and the boot line both
// read it, so these pins sit at the production call site (canon 53), not at a
// rebuilt copy of the values.
func TestPictureHtfResolvedDefaultsAreSane(t *testing.T) {
	cases := map[string]*PictureHtfConfig{
		"nil config":  nil,
		"zero struct": {},
		"enabled only": {Enabled: true},
	}
	for name, c := range cases {
		out := PictureHtfResolved(c)
		// Mechanism only: a zero knob resolves to a POSITIVE default. The
		// defaults' SANITY is pinned by FLOOR tests in the trader package,
		// derived from production constants (trader cadence + pass latency;
		// the live sink's own frame-age bound) — comparing the resolver to
		// its own constant here is a tautology that passes at ANY value
		// (CTO #212 fold).
		if out.EntryWindowSec <= 0 {
			t.Fatalf("%s: entry window default = %d, want > 0", name, out.EntryWindowSec)
		}
		if out.FreshnessSec <= 0 {
			t.Fatalf("%s: freshness default = %d, want > 0", name, out.FreshnessSec)
		}
	}
}

// An explicitly saved value is the owner's value — defaults never override it.
func TestPictureHtfResolvedHonorsExplicitValues(t *testing.T) {
	out := PictureHtfResolved(&PictureHtfConfig{Enabled: true, EntryWindowSec: 10, FreshnessSec: 2})
	if out.EntryWindowSec != 10 || out.FreshnessSec != 2 {
		t.Fatalf("explicit knob values must survive the resolver, got %d/%d", out.EntryWindowSec, out.FreshnessSec)
	}
}
