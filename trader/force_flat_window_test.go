package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W1b E13 — THE FORCE-FLAT WINDOWS REFUSE ON EVERY TRIGGER ────────────────
//
// runCycle enforces two "flat windows" only as CANCELS: the T1 force-flat
// lead [W.Start−2m, W.Start) before a red-news blackout, and the in-session
// EOD flat when a session's eod_flat_offset_min is earlier than its
// last-entry cutoff. Nothing on the arm path REFUSED inside them: the band is
// [W.Start, W.End], and last-entry is later than the flat. So:
//   - a scan whose enforce cancelled nothing (flat, no rows) went straight on
//     to author and place inside the lead (first-time authorization);
//   - the live-bar event pass (W3 D14) never calls the enforce functions at
//     all, and re-minted (D5) a placed legacy arm the scan's T1 cancel had just
//     settled, under the SAME plan version, within a second.
// The fix is a refusal in sessionRiskGateAt (both triggers + the arm/picture
// send point) and on the decision/agent path — computed BEFORE the loss-run
// store query, which fails open.
//
// Clock: the zone rig's day, 2026-09-11 (Thu). A USD T1 print at 10:15 CT →
// blackout 10:00–10:30, force-flat lead from 09:58.

func t1LeadSlice(t *testing.T, st *store.Store) {
	t.Helper()
	// A HEALTHY clock: the fixture day is not the wall-clock day, and F6 would
	// read that gap as drift and widen every T1 window by its 2m cap — which
	// happens to equal the lead and would hide it. Production's normal case is
	// no drift, no widening.
	prev := clockHoldDriftFn
	clockHoldDriftFn = func(string) (int64, bool) { return 0, true }
	t.Cleanup(func() { clockHoldDriftFn = prev })
	if _, err := st.Calendar().SaveSliceIfAbsent(&store.CalendarSliceDB{
		TradeDate: "2026-09-11", Source: "forexfactory",
		EventsJSON: `[{"time":"2026-09-11T15:15:00Z","currency":"USD","title":"Synthetic CPI","impact":"T1"}]`,
		CreatedAt:  time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("save slice: %v", err)
	}
}

var (
	e13Lead   = time.Date(2026, 9, 11, 14, 58, 30, 0, time.UTC) // 09:58:30 CT — inside the lead, before the blackout
	e13Before = time.Date(2026, 9, 11, 14, 57, 30, 0, time.UTC) // 09:57:30 CT — before the lead
)

// (a) SCAN trigger, first-time authorization. runCycle's order: the T1
// enforce (flat, no rows → acts on nothing, returns false), then the pass.
func TestArmPassRefusesInsideTheT1ForceFlatLead(t *testing.T) {
	t.Run("control: before the lead the same pass places", func(t *testing.T) {
		r := newZoneRig(t, "e13-scan-control", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
		t1LeadSlice(t, r.st)
		r.now = e13Before
		r.setTape(zoneTape(100.0, r.now, 0))
		r.at.maybeManageArmedOrdersAt(nil, r.now)
		if sigs, _ := r.drain(); len(sigs) != 1 || sigs[0].LimitPrice != 100.5 {
			t.Fatalf("fixture: outside the lead the pass must place one limit at 100.50 (reach): %+v", sigs)
		}
	})
	r := newZoneRig(t, "e13-scan-lead", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	t1LeadSlice(t, r.st)
	r.now = e13Lead
	r.setTape(zoneTape(100.0, r.now, 0))
	if r.at.enforceT1ForceFlatAt(r.now) {
		t.Fatal("fixture: flat with no rows, the T1 enforce acts on nothing — the scan goes on to the pass")
	}
	r.at.maybeManageArmedOrdersAt(nil, r.now)
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("no entry may reach the wire inside the T1 force-flat lead (09:58:30 CT, blackout 10:00): %+v", sigs)
	}
	for _, row := range r.rows() {
		if row.SignalID != "" {
			t.Fatalf("a row was placed inside the lead: %+v", row)
		}
	}
	if n := r.armRefusals("no_trade_band"); n < 1 {
		t.Fatalf("the lead refusal must be counted under no_trade_band, got %d", n)
	}
}

// (b) EVENT trigger, the D5 re-mint. A placed LEGACY arm (S2) is cancelled by
// the scan's T1 enforce inside the lead and the cancel SETTLES; the next
// live-bar pass re-authorizes it — a terminal row that reached the broker
// mints placement_seq+1 under the SAME version (store UpsertArm; only
// market_in_zone rows are pinned by D15) — and places it. S1 (a zone arm,
// never placed) is what makes the plan wake the event loop.
func TestEventPassNeverReMintsAT1CancelledLegacyArmInsideTheLead(t *testing.T) {
	r := eventRig(t, "e13-event-remint", zoneDoc(
		zoneScenario("S1", kernel.EntryPolicyMarketInZone, []float64{103.5, 104.5}, false),
		zoneScenario("S2", "", nil, false)))
	t1LeadSlice(t, r.st)

	// 09:57:30 — the scan places S2 (legacy, confirm met above 100); S1's
	// zone (103.50–104.50) is above the price: short of it, never placed.
	r.now = e13Before
	r.setTape(zoneTape(101.95, r.now, 0))
	r.at.maybeManageArmedOrdersAt(nil, r.now)
	sigs, _ := r.drain()
	if len(sigs) != 1 {
		t.Fatalf("fixture: the scan must place exactly the legacy S2 arm before the lead: %+v", sigs)
	}
	placed := r.row("S2")
	if placed.SignalID != sigs[0].SignalID {
		t.Fatalf("fixture: the placed signal must be S2's: row %+v sig %s", placed, sigs[0].SignalID)
	}

	// 09:58:30 — the scan's T1 enforce cancels it, and the broker confirms.
	r.now = e13Lead
	acks := make(chan ntwire.OrderUpdatePayload, 4)
	r.at.armedSyncSeam = &armedSyncSeam{
		Cancel: func(sid string) error {
			acks <- ntwire.OrderUpdatePayload{SignalID: sid, State: "cancelled", Account: "Sim101"}
			return nil
		},
		Stream:  func() <-chan ntwire.OrderUpdatePayload { return acks },
		Timeout: time.Second,
	}
	if !r.at.enforceT1ForceFlatAt(r.now) {
		t.Fatal("fixture: the T1 enforce must act (cancel the working arm)")
	}
	if got := r.row("S2"); got.ID != placed.ID || got.State != "cancelled" {
		t.Fatalf("fixture: S2's cancel must settle to cancelled: %+v", got)
	}
	// The placement was a minute ago on the fixture clock; B3's 55 s dupe
	// window runs on the wall clock. A fresh broker adapter models the minute
	// that elapsed (production: 09:57:30 → 09:59:00 is past the window).
	r.at.trader = ntTrader.NewTCPTrader(r.srv, "MNQ", "Sim101")

	// 09:58:30 — a final 1m frame wakes the event pass.
	r.setTape(zoneTape(101.95, r.now, 0))
	r.finalFrame(101.95)
	if !r.waitPasses(1, 3*time.Second) {
		t.Fatal("fixture: the final frame must run one event pass")
	}
	if got := r.collectSignals(1500 * time.Millisecond); len(got) != 0 {
		t.Fatalf("the event pass re-placed an arm inside the T1 force-flat lead the scan had just emptied: %+v", got)
	}
	if got := r.row("S2"); got.ID != placed.ID {
		t.Fatalf("the event pass minted a new S2 placement (seq %d → %d) inside the lead", placed.PlacementSeq, got.PlacementSeq)
	}
}

// (c) EVENT trigger, before the scan's cancel. The scan runs every ~2 min and
// the lead is 2 min: an arm authorized before the lead is still resting,
// unplaced, when the first final frame inside the lead arrives — the event
// pass (which never calls the enforce) must not place it.
func TestEventPassNeverPlacesInsideTheT1ForceFlatLead(t *testing.T) {
	r := eventRig(t, "e13-event-lead", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	t1LeadSlice(t, r.st)
	r.now = e13Before
	r.armZoneRow() // authored and armed before the lead, short of the zone
	r.now = e13Lead
	r.setTape(zoneTape(100.0, r.now, 0))
	r.finalFrame(100.0)
	if !r.waitPasses(1, 3*time.Second) {
		t.Fatal("fixture: the final frame must run one event pass")
	}
	if got := r.collectSignals(1500 * time.Millisecond); len(got) != 0 {
		t.Fatalf("the event pass placed inside the T1 force-flat lead: %+v", got)
	}
}

// The in-session EOD flat, earlier than last-entry (a per-session override
// nothing validates): past the flat and before last-entry, no arm is placed.
func TestArmPassRefusesPastAnEarlierInSessionEODFlat(t *testing.T) {
	r := newZoneRig(t, "e13-eod", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	flat, last := 900, 15 // TEST ends 23:59 → flat 08:59 CT, last entry 23:44 CT
	r.at.config.StrategyConfig.DayPlan.Sessions = []store.DayPlanSessionOverride{{Session: "TEST", EODFlatOffsetMin: &flat, LastEntryOffsetMin: &last}}
	r.setTape(zoneTape(100.0, r.now, 0)) // 10:00 CT — past the 08:59 flat, before the 23:44 last entry
	r.at.maybeManageArmedOrdersAt(nil, r.now)
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("no entry may be placed past the session's EOD flat (08:59 CT): %+v", sigs)
	}
	if n := r.armRefusals("no_trade_band"); n < 1 {
		t.Fatalf("the EOD-flat refusal must be counted under no_trade_band, got %d", n)
	}
}

// The decision / agent path: the same windows refuse under the decision
// path's session_gate class.
func TestDecisionPathRefusesInsideTheT1ForceFlatLead(t *testing.T) {
	r := newZoneRig(t, "e13-decision", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	t1LeadSlice(t, r.st)
	reason, refused := r.at.admitEntry(admitIntent{Path: admitDecision, Symbol: "MNQ", Action: "open_long", Now: e13Lead,
		Decision: &kernel.Decision{Action: "open_long", Symbol: "MNQ"}})
	if !refused || !strings.HasPrefix(reason, "session_gate:") {
		t.Fatalf("an AI entry inside the T1 force-flat lead must be refused by session_gate, got refused=%v %q", refused, reason)
	}
	if reason, refused := r.at.AdmitManualEntryBracketAt("MNQ", "open_long", 90, 130, e13Lead); !refused || !strings.HasPrefix(reason, "session_gate:") {
		t.Fatalf("the agent door inside the lead must be refused by session_gate before its bracket is judged, got refused=%v %q", refused, reason)
	}
}

// The window is read BEFORE the loss-run query, which fails OPEN: an
// unreadable positions table skips the breaker for the cycle, never the
// force-flat window. Driven at sessionRiskGateAt, the one function both the
// pass head and the arm/picture send point call.
func TestForceFlatWindowSurvivesALossRunReadFailure(t *testing.T) {
	r := newZoneRig(t, "e13-failopen", zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	t1LeadSlice(t, r.st)
	if err := r.st.GormDB().Exec("DROP TABLE trader_positions").Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := r.st.Position().CountConsecutiveLossesSince(r.at.id, 0); err == nil {
		t.Fatal("fixture: the loss-run read must fail")
	}
	v := r.at.sessionRiskGateAt(e13Lead)
	if !v.Refuse || v.Class != "no_trade_band" || !strings.Contains(v.Reason, "T1 force-flat window") {
		t.Fatalf("a failed loss-run read must not skip the force-flat window: %+v", v)
	}
	if v := r.at.sessionRiskGateAt(e13Before); v.Refuse {
		t.Fatalf("control: before the lead with the read failing, the gate fails open (breaker skipped): %+v", v)
	}
}
