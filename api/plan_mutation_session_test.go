package api

import (
	"testing"
	"time"

	"nofx/kernel"
)

// F18 (WAVE 117 PR-D, ports #117 a6b88b7d) — the plan mutation session resolves
// through the wrap-aware chain date BEFORE the session is dereferenced: an
// overnight ASIA apply at 00:30 CT must address the PREVIOUS calendar day's
// chain, and a session gap must REFUSE (nil session, ok=false) — never hand a
// nil session to the caller (today's inline form dereferenced sess.Name before
// checking ok).
func TestPlanMutationSessionGapAndOvernightChain(t *testing.T) {
	s, st := askTestServer(t)
	reg := kernel.DefaultSessionRegistry()
	for i := range reg.Sessions {
		if reg.Sessions[i].Name == kernel.SessionAsia {
			reg.Sessions[i].Enabled = true
			reg.Sessions[i].WindowStartCT = "17:00"
			reg.Sessions[i].WindowEndCT = "02:00"
			reg.Sessions[i].ReadCT = "16:30"
			reg.Sessions[i].FlatCT = "02:00"
		}
	}
	raw, err := reg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, raw); err != nil {
		t.Fatal(err)
	}
	// 00:30 CT on 2026-09-14: ASIA (17:00–02:00) is active and its chain is the
	// previous calendar date's.
	session, date, ok := s.planMutationSessionAt("t", time.Date(2026, 9, 14, 0, 30, 0, 0, kernel.CTLocation()))
	if !ok || session == nil || session.Name != kernel.SessionAsia || date != "2026-09-13" {
		t.Fatalf("overnight chain: session=%+v date=%q ok=%v", session, date, ok)
	}
	// 15:00 CT: NY ended 14:45 and ASIA starts 17:00 — a gap must refuse.
	session, date, ok = s.planMutationSessionAt("t", time.Date(2026, 9, 14, 15, 0, 0, 0, kernel.CTLocation()))
	if ok || session != nil || date != "" {
		t.Fatalf("session gap must refuse: session=%+v date=%q ok=%v", session, date, ok)
	}
}
