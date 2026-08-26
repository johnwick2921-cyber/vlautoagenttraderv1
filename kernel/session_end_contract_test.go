package kernel

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// P3 — SESSION-END CONTRACT. One source of truth (DefaultSessionRegistry) and a
// test that FAILS if any mirror drifts from it: the FE session table, the spec,
// and the internal invariant that a session ends where its flat fires.
//
// The bug this locks down: NY used to end at 15:00 CT while flatting at 14:45 CT,
// so for 15 minutes the gate called the session open even though EOD-flat had
// already brought positions to flat.

const (
	wantNYStart = "08:30"
	wantNYEnd   = "14:45" // = 15:45 ET
	wantNYRead  = "08:25"
	wantNYFlat  = "14:45" // MUST equal wantNYEnd
)

// 1) the registry itself
func TestSessionEndSingleSourceOfTruth(t *testing.T) {
	ny, ok := DefaultSessionRegistry().SessionByName(SessionNY)
	if !ok {
		t.Fatal("NY missing from the registry")
	}
	for _, c := range []struct{ field, got, want string }{
		{"WindowStartCT", ny.WindowStartCT, wantNYStart},
		{"WindowEndCT", ny.WindowEndCT, wantNYEnd},
		{"ReadCT", ny.ReadCT, wantNYRead},
		{"FlatCT", ny.FlatCT, wantNYFlat},
	} {
		if c.got != c.want {
			t.Fatalf("NY %s = %q, want %q (contract: NY ends 15:45 ET = 14:45 CT)", c.field, c.got, c.want)
		}
	}
	// THE INVARIANT: a session ends exactly where its flat fires — no open-but-flat band.
	for _, s := range DefaultSessionRegistry().Sessions {
		if s.WindowEndCT != s.FlatCT {
			t.Fatalf("session %s: window ends %q but flat is %q — that gap is a band where the gate "+
				"calls the session open while positions are already flat", s.Name, s.WindowEndCT, s.FlatCT)
		}
	}
}

// 2) the gate agrees: entries are refused at/after the flat, allowed just before.
func TestSessionEndGateAgreesWithFlat(t *testing.T) {
	ny, _ := DefaultSessionRegistry().SessionByName(SessionNY)
	end, _ := parseHHMM(ny.FlatCT)
	ct, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata")
	}
	day := time.Date(2026, 8, 17, 0, 0, 0, 0, ct) // a Monday
	// 1 minute BEFORE the flat → in window
	before := day.Add(time.Duration(end-1) * time.Minute)
	if !ny.InWindow(before) {
		t.Fatalf("14:44 CT must still be inside the NY window")
	}
	// AT the flat → out of window (end is exclusive) ⇒ no entry can be opened after flat
	at := day.Add(time.Duration(end) * time.Minute)
	if ny.InWindow(at) {
		t.Fatalf("%s CT is the flat — the session window must already be closed there", ny.FlatCT)
	}
}

// 3) the FE mirror (TypeScript) must carry the same numbers.
func TestSessionEndFEMirrorMatches(t *testing.T) {
	b, err := os.ReadFile("../web/src/components/plan/sessionConfig.ts")
	if err != nil {
		t.Skipf("FE mirror not present: %v", err)
	}
	src := string(b)
	ny := sliceBetween(src, "name: 'NY'", "},")
	if ny == "" {
		t.Fatal("NY block not found in sessionConfig.ts")
	}
	for _, c := range []struct{ key, want string }{
		{"startMin", wantNYStart}, {"endMin", wantNYEnd},
		{"readMin", wantNYRead}, {"flatMin", wantNYFlat},
	} {
		re := regexp.MustCompile(c.key + `: ctToMinutes\('([0-9:]+)'\)`)
		m := re.FindStringSubmatch(ny)
		if m == nil {
			t.Fatalf("sessionConfig.ts NY: %s not found", c.key)
		}
		if m[1] != c.want {
			t.Fatalf("FE mirror drift — sessionConfig.ts NY %s = %q but the registry says %q. "+
				"The Go registry is the single source of truth; update the FE table.", c.key, m[1], c.want)
		}
	}
}

// 4) the spec must state the contract unambiguously (CT, with the ET equivalent).
func TestSessionEndSpecUnambiguous(t *testing.T) {
	b, err := os.ReadFile("../docs/VL-DAYPLAN-FULL-SPEC.md")
	if err != nil {
		t.Skipf("spec not present: %v", err)
	}
	spec := string(b)
	if strings.Contains(spec, "NY 08:30–15:45 (read 08:25)") {
		t.Fatal("spec still writes NY as 08:30–15:45 — that mixes CT (08:30) with ET (15:45)")
	}
	if !strings.Contains(spec, "14:45 CT") {
		t.Fatal("spec must state the NY end/flat in CT (14:45 CT = 15:45 ET)")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sliceBetween(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	j := strings.Index(rest, to)
	if j < 0 {
		return rest
	}
	return rest[:j]
}
