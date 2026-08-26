package trader

import (
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// The EOD-flat twin of the last-entry fix (defect class 7). The old day-scoped
// 14:45 CT check would have force-flattened an Asia position ON SIGHT at 21:00
// CT the moment the last-entry fix let Asia entries flow.
//
// enforceEODFlatAt exits early (false) when there is nothing to flatten, so the
// DECISION boundary — "would it act at this instant?" — is what these pin, via
// the same pure helpers the gate uses.

func TestEODFlatIsSessionScoped(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	var dp *store.DayPlanConfig // nil → default offset 15

	for _, tc := range []struct {
		name     string
		date, tm string
		wantFlat bool // is this instant past the ACTIVE session's flat?
	}{
		// The instant that mattered: 21:00 CT inside ASIA. Old code: flatten
		// (21:00 >= 14:45). New: ASIA flat is 01:45 (02:00−15) — do NOT flatten.
		{"asia 21:00 not flat", "2026-08-18", "21:00", false},
		{"asia 01:50 flat (wrapped)", "2026-08-19", "01:50", true},
		{"london 05:00 not flat", "2026-08-18", "05:00", false},
		{"london 08:20 flat", "2026-08-18", "08:20", true},
		// NY: end 14:45 − 15 = 14:30.
		{"ny 14:00 not flat", "2026-08-18", "14:00", false},
		{"ny 14:35 flat", "2026-08-18", "14:35", true},
	} {
		now := ctTimeAt(t, tc.date, tc.tm)
		sess, ok := reg.ActiveSession(now)
		if !ok {
			t.Fatalf("%s: expected an active session", tc.name)
		}
		flatMin, _, okC := sessionCutoffCT(sess, dp.EODFlatOffsetFor(sess.Name))
		if !okC {
			t.Fatalf("%s: cutoff resolution failed", tc.name)
		}
		if got := pastSessionCutoff(now, sess, flatMin); got != tc.wantFlat {
			t.Errorf("%s: pastFlat=%v want %v (session %s)", tc.name, got, tc.wantFlat, sess.Name)
		}
	}
}

// Between sessions (14:45→17:00 CT gap) there is no active session — the gate
// treats that as past-flat by definition, preserving the old rule's one virtue:
// nothing rides through the close.
func TestEODFlatBetweenSessions(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	if _, ok := reg.ActiveSession(ctTimeAt(t, "2026-08-18", "15:30")); ok {
		t.Fatal("15:30 CT must be between sessions in the default registry")
	}
}
