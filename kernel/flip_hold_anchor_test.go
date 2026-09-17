package kernel

import (
	"strings"
	"testing"
)

// W-FLIP-HOLD-ANCHOR (2026-09-17) — the hold anchor is the plan's STATE,
// never a same-bias re-read version.

func TestResolveFlipHoldAnchorIgnoresSameBiasReReads(t *testing.T) {
	birth := ctMs(2026, 9, 16, 16, 35)
	vs := []PlanVersionFact{
		{Version: 1, TriggerReason: "ASIA_scheduled_read", BiasDirection: "short", CreatedAtMs: birth},
		{Version: 2, TriggerReason: "level_event", BiasDirection: "short", CreatedAtMs: ctMs(2026, 9, 16, 17, 19)},
		{Version: 3, TriggerReason: "structure_mss", BiasDirection: "short", CreatedAtMs: ctMs(2026, 9, 16, 19, 2)},
	}
	got := ResolveFlipHoldAnchor(vs, nil, 3, ctMs(2026, 9, 16, 19, 2))
	if got.SinceMs != birth || got.Source != FlipHoldAnchorBirth {
		t.Fatalf("same-bias re-reads must not move the anchor: got %+v", got)
	}
}

func TestResolveFlipHoldAnchorLatestWins(t *testing.T) {
	birth := ctMs(2026, 9, 16, 16, 35)
	biasChange := ctMs(2026, 9, 16, 20, 28)
	replan := ctMs(2026, 9, 16, 21, 0)
	flipAt := ctMs(2026, 9, 16, 22, 0)
	rearmAt := ctMs(2026, 9, 16, 22, 30)
	vs := []PlanVersionFact{
		{Version: 1, TriggerReason: "ASIA_scheduled_read", BiasDirection: "short", CreatedAtMs: birth},
		{Version: 2, TriggerReason: "level_event", BiasDirection: "neutral", CreatedAtMs: biasChange},
		{Version: 3, TriggerReason: "death_replan", BiasDirection: "neutral", CreatedAtMs: replan},
		{Version: 4, TriggerReason: "level_event", BiasDirection: "neutral", CreatedAtMs: ctMs(2026, 9, 16, 23, 0)},
	}
	// versions only: bias change then re-plan, the re-plan is later.
	got := ResolveFlipHoldAnchor(vs[:2], nil, 2, 0)
	if got.SinceMs != biasChange || got.Source != FlipHoldAnchorBiasChange {
		t.Fatalf("bias change must anchor: %+v", got)
	}
	got = ResolveFlipHoldAnchor(vs, nil, 4, 0)
	if got.SinceMs != replan || got.Source != FlipHoldAnchorReplan {
		t.Fatalf("re-plan version must anchor, v4 re-read must not: %+v", got)
	}
	// transitions on the current version after its birth: flip then re-arm.
	trs := []PlanTransitionFact{
		{Version: 3, Event: "dormant", Reason: "dormant:flip:flip-condition: x", AtMs: flipAt},
		{Version: 3, Event: "active", Reason: "rearmed:2x5m close back", AtMs: rearmAt},
	}
	got = ResolveFlipHoldAnchor(vs[:3], trs, 3, 0)
	if got.SinceMs != rearmAt || got.Source != FlipHoldAnchorRearm {
		t.Fatalf("re-arm must be the latest anchor: %+v", got)
	}
	got = ResolveFlipHoldAnchor(vs[:3], trs[:1], 3, 0)
	if got.SinceMs != flipAt || got.Source != FlipHoldAnchorFlip {
		t.Fatalf("flip must anchor: %+v", got)
	}
	// a death→dormant marker is not a flip and does not anchor.
	got = ResolveFlipHoldAnchor(vs[:3], []PlanTransitionFact{{Version: 3, Event: "dormant", Reason: "dormant:death:x", AtMs: flipAt}}, 3, 0)
	if got.SinceMs != replan {
		t.Fatalf("death dormancy must not anchor: %+v", got)
	}
}

func TestResolveFlipHoldAnchorFallback(t *testing.T) {
	got := ResolveFlipHoldAnchor(nil, nil, 3, 42)
	if got.SinceMs != 42 || got.Source != FlipHoldAnchorVersion {
		t.Fatalf("empty chain must fall back to the version, tagged: %+v", got)
	}
	if !strings.Contains(FlipHoldAnchorLabel(), FlipHoldAnchorRearm) || !strings.Contains(FlipHoldAnchorLabel(), FlipHoldAnchorBirth) {
		t.Fatalf("boot label must be read from the kinds table: %q", FlipHoldAnchorLabel())
	}
}

// TestG3FlipHoldAnchorSeparatesClocks — the condition window stays the
// version's birth while the hold reads its own anchor.
func TestG3FlipHoldAnchorSeparatesClocks(t *testing.T) {
	flip := g7FlipCond()
	fixture := g7Fixture()
	now := ctMs(2026, 8, 21, 8, 10)
	doc := PlanDoc{Bias: PlanBias{FlipCondition: "flip"}, FlipStructured: &flip}
	versionBirth := ctMs(2026, 8, 21, 7, 45) // 25 min old: held on the old clock
	// old semantics (anchor = version) — held.
	_, fired, skipped := PlanDeathOrFlipSinceFreshHold(doc, fixture, "2x5m", versionBirth, now, FlipHoldAnchor{SinceMs: versionBirth, Source: FlipHoldAnchorVersion})
	if fired || !strings.Contains(strings.Join(skipped, ";"), "flip=hold") {
		t.Fatalf("version-anchored hold must still hold: fired=%v skipped=%v", fired, skipped)
	}
	// new semantics: same condition window, chain born 07:00 — evaluated.
	_, fired, skipped = PlanDeathOrFlipSinceFreshHold(doc, fixture, "2x5m", versionBirth, now, FlipHoldAnchor{SinceMs: ctMs(2026, 8, 21, 7, 0), Source: FlipHoldAnchorBirth})
	if !fired {
		t.Fatalf("hold anchored to an old chain birth must let the flip fire, skipped=%v", skipped)
	}
}
