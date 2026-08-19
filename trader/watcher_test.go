package trader

import (
	"os"
	"strings"
	"testing"

	"nofx/kernel"
)

func defaultKnobs() watchRailKnobs { return watchRailKnobs{minConf: 70, minHold: 2, warnConsec: 2} }

// R1 — low-confidence or uncited "invalidated" is recorded as weakening.
func TestWatchRailsR1LowConfidenceDemotes(t *testing.T) {
	st := watchState{}
	st, accepted, warn := applyWatchRails(st, "invalidated", 60, "OR-L reclaimed", defaultKnobs())
	if accepted != "weakening" || warn {
		t.Errorf("low-conf invalidated: accepted=%s warn=%v, want weakening/false", accepted, warn)
	}
	if st.ConsecInvalid != 0 {
		t.Errorf("demoted read must not count toward the WARN streak (got %d)", st.ConsecInvalid)
	}
	st, accepted, _ = applyWatchRails(st, "invalidated", 90, "", defaultKnobs())
	if accepted != "weakening" {
		t.Errorf("uncited invalidated: accepted=%s, want weakening", accepted)
	}
}

// R3 — WARN only after 2 consecutive ACCEPTED invalidated reads, once per episode.
func TestWatchRailsWarnAfterTwoConsecutive(t *testing.T) {
	st := watchState{}
	var warn bool
	st, _, warn = applyWatchRails(st, "invalidated", 85, "stated stop zone lost", defaultKnobs())
	if warn {
		t.Fatal("first accepted invalidated must NOT warn")
	}
	st, _, warn = applyWatchRails(st, "invalidated", 85, "stated stop zone lost", defaultKnobs())
	if !warn {
		t.Fatal("second consecutive accepted invalidated must WARN")
	}
	st, _, warn = applyWatchRails(st, "invalidated", 85, "stated stop zone lost", defaultKnobs())
	if warn {
		t.Fatal("episode already warned — no second WARN")
	}
	// Recovery clears the episode…
	st, _, warn = applyWatchRails(st, "intact", 80, "", defaultKnobs())
	if warn || st.ConsecInvalid != 0 || st.Warned {
		t.Fatal("recovery must clear the episode")
	}
	// …and a NEW streak can warn again.
	st, _, _ = applyWatchRails(st, "invalidated", 85, "again", defaultKnobs())
	_, _, warn = applyWatchRails(st, "invalidated", 85, "again", defaultKnobs())
	if !warn {
		t.Fatal("a fresh episode after recovery must be able to WARN again")
	}
}

// R2 — one-step downgrade + hold window; upgrades free.
func TestWatchRailsOneStepDowngradeAndHold(t *testing.T) {
	k := defaultKnobs()
	st := watchState{}
	// intact → accepted invalidated read clamps the STATUS to weakening.
	st, accepted, _ := applyWatchRails(st, "invalidated", 90, "cited", k)
	if accepted != "weakening" || st.Status != "weakening" {
		t.Fatalf("intact→invalidated must clamp to weakening (got %s)", accepted)
	}
	// Within the hold window the status freezes at weakening.
	st, accepted, _ = applyWatchRails(st, "invalidated", 90, "cited", k)
	if accepted != "weakening" {
		t.Fatalf("downgrade within min-hold must freeze (got %s)", accepted)
	}
	st, accepted, _ = applyWatchRails(st, "invalidated", 90, "cited", k)
	if accepted != "weakening" {
		t.Fatalf("still within min-hold (got %s)", accepted)
	}
	// Past the hold window the next step is allowed.
	st, accepted, _ = applyWatchRails(st, "invalidated", 90, "cited", k)
	if accepted != "invalidated" {
		t.Fatalf("past min-hold the one-step downgrade must proceed (got %s)", accepted)
	}
	// Upgrade (recovery) is free and immediate.
	_, accepted, _ = applyWatchRails(st, "intact", 80, "", k)
	if accepted != "intact" {
		t.Fatalf("recovery must be immediate (got %s)", accepted)
	}
}

func TestPositionModeResolution(t *testing.T) {
	at := &AutoTrader{}
	if at.positionMode() != PositionModeWatch {
		t.Error("empty position_mode must resolve to ai_watch (the new default)")
	}
	at.config.PositionMode = PositionModeBracketOnly
	if at.positionMode() != PositionModeBracketOnly {
		t.Error("explicit bracket_only must keep the legacy skip")
	}
	at.config.PositionMode = "ai_watch"
	if at.positionMode() != PositionModeWatch {
		t.Error("explicit ai_watch must resolve to ai_watch")
	}
}

func TestWatchKnobsEnvOverride(t *testing.T) {
	os.Setenv("WATCH_INVALIDATE_MIN_CONF", "80")
	defer os.Unsetenv("WATCH_INVALIDATE_MIN_CONF")
	if watchInvalidateMinConf() != 80 {
		t.Error("WATCH_INVALIDATE_MIN_CONF override not honored")
	}
	os.Unsetenv("WATCH_INVALIDATE_MIN_CONF")
	if watchInvalidateMinConf() != defaultWatchInvalidateMinConf {
		t.Error("default must be 70")
	}
}

// 3.3 — observer parse: schema enforced, action-like fields flagged, never invented.
func TestParseObserverAssessment(t *testing.T) {
	a, err := kernel.ParseObserverAssessment(`noise {"thesis_status":"weakening","note":"drifting","confidence":66} tail`)
	if err != nil || a.ThesisStatus != "weakening" || a.Confidence != 66 {
		t.Fatalf("valid parse failed: %v %+v", err, a)
	}
	if a.ActionIgnored {
		t.Error("clean response must not flag ActionIgnored")
	}
	if _, err := kernel.ParseObserverAssessment(`{"thesis_status":"bullish","note":"x","confidence":50}`); err == nil {
		t.Error("out-of-enum thesis_status must error")
	}
	if _, err := kernel.ParseObserverAssessment(`no json here`); err == nil {
		t.Error("no-object response must error (recorded unparseable, no rail movement)")
	}
	a, err = kernel.ParseObserverAssessment(`{"thesis_status":"invalidated","invalidation_cited":"OR-L lost","note":"n","confidence":88,"action":"close_short"}`)
	if err != nil || !a.ActionIgnored {
		t.Errorf("action-like field must be flagged ActionIgnored (err=%v flagged=%v)", err, a.ActionIgnored)
	}
}

// 3.2 STRUCTURAL RAIL — the watcher file must be incapable of emitting orders:
// no broker reference, no executor call, no wire command may appear in it.
func TestWatcherFileHasNoOrderAuthority(t *testing.T) {
	src, err := os.ReadFile("auto_trader_watcher.go")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"at.trader", "OpenLong", "OpenShort", "CloseLong", "CloseShort",
		"PlaceOrder", "placeEntry", "SendSignal", "SendMoveStop", "MoveStop",
		"executeDecision", "emergencyClosePosition", "SetStopLoss", "SetTakeProfit",
		"CancelAllOrders", "recordAndConfirmOrder",
	}
	for _, f := range forbidden {
		if strings.Contains(string(src), f) {
			t.Errorf("auto_trader_watcher.go references %q — watch cycles must have ZERO order authority (structural rail)", f)
		}
	}
}
