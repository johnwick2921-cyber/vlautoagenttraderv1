package trader

import (
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// W-PICTURE-HTF (2026-09-20) — the production seam's refusal gates. The wire
// send itself is pinned in trader/ninjatrader (market_entry_wire_test.go);
// these pins prove the gates BEFORE the wire: nil state, non-NT8 trader, stale
// feed, closed window, and an unreconciled pending row that blocks re-entry.

func TestPictureHtfSendRefusesNilState(t *testing.T) {
	if err := pictureHtfSend(nil, nil, 0, 0, 0, time.Now()); err == nil || !strings.Contains(err.Error(), "nil state") {
		t.Fatalf("nil state must refuse, got %v", err)
	}
}

func TestPictureHtfSendRefusesNonNTTrader(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	_ = st
	// resetTrader builds a bare AutoTrader with NO concrete trader — the seam
	// must refuse rather than guess.
	ev := NewPictureHtfEvaluator(at, store.PictureHtfResolved(&store.PictureHtfConfig{Enabled: true}))
	ev.freshest5mAt = time.Now()
	row := &store.PictureHtfOpportunityDB{OppKey: "k", SignalID: "claim", Symbol: "MNQ", Direction: "long",
		WindowClose: time.Now().Add(time.Minute).UnixMilli()}
	err := pictureHtfSend(ev, row, 98, 110, 1, time.Now())
	if err == nil || !strings.Contains(err.Error(), "not the concrete NT8 TCP trader") {
		t.Fatalf("a non-TCP trader must refuse at the seam, got %v", err)
	}
}

func TestPictureHtfSendRefusesStaleFeed(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	ev := NewPictureHtfEvaluator(at, store.PictureHtfResolved(&store.PictureHtfConfig{Enabled: true, FreshnessSec: 2}))
	ev.freshest5mAt = time.Now().Add(-10 * time.Second)
	row := &store.PictureHtfOpportunityDB{OppKey: "k", SignalID: "claim", Symbol: "MNQ", Direction: "long",
		WindowClose: time.Now().Add(time.Minute).UnixMilli()}
	err := pictureHtfSend(ev, row, 98, 110, 1, time.Now())
	if err == nil || !strings.Contains(err.Error(), "bar data is") {
		t.Fatalf("a stale feed must refuse at the seam, got %v", err)
	}
}

func TestPictureHtfSendRefusesClosedWindow(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	ev := NewPictureHtfEvaluator(at, store.PictureHtfResolved(&store.PictureHtfConfig{Enabled: true}))
	ev.freshest5mAt = time.Now()
	row := &store.PictureHtfOpportunityDB{OppKey: "k", SignalID: "claim", Symbol: "MNQ", Direction: "long",
		WindowClose: time.Now().Add(-time.Second).UnixMilli()}
	err := pictureHtfSend(ev, row, 98, 110, 1, time.Now())
	if err == nil || !strings.Contains(err.Error(), "entry window closed") {
		t.Fatalf("a closed window must refuse at the seam, got %v", err)
	}
}

func TestPictureHtfContractSizeNeverExceedsClamp(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{})
	if got := pictureHtfContractSize(at); got != 1 {
		t.Fatalf("the two-picture mode sizes 1 contract by design, got %.0f", got)
	}
}

func TestPictureHtfBootLineNamesEverything(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	line := at.pictureHtfBootLine()
	for _, want := range []string{"picture-htf:", "mode=on", "rule=v1", "SIM-only", "final+emitted_at", "addon=not proven", ntwire.MinAddonBuildPictureHtf} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line must name %q, got %q", want, line)
		}
	}
	off, _ := resetTrader(t, store.StrategyConfig{})
	if l := off.pictureHtfBootLine(); !strings.Contains(l, "mode=off") {
		t.Fatalf("disabled mode must read off: %q", l)
	}
}
