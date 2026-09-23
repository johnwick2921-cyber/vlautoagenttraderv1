package trader

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W5 (Builder B) — THE SHARED EXECUTOR RUNS A PICTURE SCENARIO ─
//
// A Picture opportunity reaches the executor as an ordinary plan_final
// scenario (Source "picture", id P<n>, condition acceptance, no confirm{},
// arm{market_in_zone}, economics.entry_zone, Machine{…}). Everything a planner
// scenario faces, it faces (D12 — a parity CORRECTION: direction bias,
// min_scenario_quality, the HTF veto, shadow, EntryGate). What is different is
// written here, once, and called from the pass at named points:
//
//	machine gate    FIRST in the authoring loop: the eligibility window (D6),
//	                the run epoch (D21), the frozen evidence — so an expired
//	                scenario never reaches a leg that would WARN on it (F14)
//	geometry        no composeArmStop widening, no one_setup obstacle target
//	                (D10, CTO Q18); R:R at the FAR edge ≥ the evidence's R:R
//	                floor (D11), beside the arm chain's own floor
//	one_setup       EXEMPT (D8 ruling): one_setup governs PLANNER plays only
//	placement       the deadline and the run epoch again, at the send point
//	                (CTO 1790192366762); a resting row past its deadline gets
//	                its cancel requested (the deadline replaces
//	                zone_rest_max_min for source rows)
//	one live entry  a placement never cancels the OTHER source's unplaced arm
//	                (D9 + mirror); the one-live-entry guards refuse the second
//	                order while the first is working/open
//	Stop / OFF      Stop, a reload and Day Plan OFF invalidate entry
//	                permission (D21): unplaced → terminal, resting → cancel
//	                requested
//
// Planner scenarios and rows (Source "") never reach any of this: every call
// below returns at once for them, so with Picture off the pass is the W3 pass
// byte for byte.

// pictureHandOffSweepHook is the D17 interrupted-hand-off sweep, called once
// at the head of every armed pass. Builder A binds it (trader/
// picture_plan_source.go) at integration; the default does nothing.
var pictureHandOffSweepHook = func(at *AutoTrader, now time.Time) {}

// Picture refusal / retirement texts — READ by the card, the ledger reason and
// the tests (one definition each).
const (
	pictureWindowClosedNeverPlaced = "picture eligibility window closed — never placed"
	pictureWindowClosedCancel      = "picture eligibility window closed"
	pictureStoppedReason           = "trader stopped"
	pictureDayPlanOffReason        = "Day Plan master off"
)

// Counter classes (store.IncArmRefusal per session-day for the scenario-level
// refusals; store.IncSystemCounter for row events that may have no plan).
const (
	pictureClassWindowClosed = "picture_window_closed"
	pictureClassPreviousRun  = "picture_previous_run"
	pictureClassEvidence     = "picture_evidence"
	pictureClassRRFloor      = "picture_rr_floor"
	pictureClassSharedPrefix = "picture_scenario:"
)

// pictureScenarioEvidence reads a machine scenario's frozen evidence (the
// adapter's PictureEvidence, never the evaluator's row).
func pictureScenarioEvidence(sc kernel.PlanScenario) (PictureEvidence, error) {
	var ev PictureEvidence
	if sc.Machine == nil || len(sc.Machine.Evidence) == 0 {
		return ev, fmt.Errorf("no evidence recorded")
	}
	if err := json.Unmarshal(sc.Machine.Evidence, &ev); err != nil {
		return ev, fmt.Errorf("evidence unreadable: %v", err)
	}
	return ev, nil
}

func pictureEpochText(e int64, ok bool) string {
	if !ok {
		return "none"
	}
	return strconv.FormatInt(e, 10)
}

// refusePictureScenario is one counted, de-duplicated refusal of a machine
// scenario at authoring (once per change of class, like every arm refusal).
func (at *AutoTrader) refusePictureScenario(plan *kernel.ActivePlan, sc kernel.PlanScenario, class, detail string, scope *armedPassScope) {
	scope.note(sc.ID, "refused: "+class+": "+detail)
	key := plan.PlanID + ":" + strconv.Itoa(plan.Version) + ":" + sc.ID + ":picture"
	if !armRefusalChanged(&at.armRefusalLast, key, class) {
		return
	}
	shown := ""
	if at.store != nil {
		if n, err := store.IncArmRefusal(at.store, at.id, kernel.PlanTradeDateFor(plan), plan.Session, class); err == nil {
			shown = fmt.Sprintf(" · %s this session: %d", class, n)
		}
	}
	at.logWarnf("📷 picture scenario %s %s REFUSED — %s: %s%s", plan.Session, sc.ID, class, detail, shown)
}

// countPictureShared counts a SHARED-leg refusal of a machine scenario under
// its own name beside the leg's own counter (D12: the parity correction's
// effect on Picture is quoted from a number, never inferred). Called from
// inside the leg's once-per-change block.
func (at *AutoTrader) countPictureShared(plan *kernel.ActivePlan, class string) {
	if at.store == nil || plan == nil {
		return
	}
	_, _ = store.IncArmRefusal(at.store, at.id, kernel.PlanTradeDateFor(plan), plan.Session, pictureClassSharedPrefix+class)
}

// pictureScenarioGate is THE MACHINE GATE, first in the per-scenario loop.
// ok=false → the pass skips the scenario (the caller `continue`s). Order:
//
//  1. past the deadline → an unplaced row for the opportunity goes terminal
//     "picture eligibility window closed — never placed" (counted); no row
//  2. before the window opens → wait (no row, nothing counted)
//  3. recorded under another run epoch, or no run alive → refused "recorded
//     by a previous run (reload) — not placed" (counted); no row
//  4. evidence unreadable or without an R:R floor → refused (fail-closed)
func (at *AutoTrader) pictureScenarioGate(plan *kernel.ActivePlan, sc kernel.PlanScenario, ledger *store.ArmedOrderStore, now time.Time, scope *armedPassScope) (PictureEvidence, bool) {
	m := sc.Machine
	if m == nil || sc.Source != kernel.ScenarioSourcePicture {
		at.refusePictureScenario(plan, sc, pictureClassEvidence, fmt.Sprintf("source %q without a Picture machine record — fail-closed", sc.Source), scope)
		return PictureEvidence{}, false
	}
	nowMs := now.UnixMilli()
	if nowMs > m.EligibleUntilMs {
		closed := at.closePictureRowsForRef(ledger, m.Ref, now)
		detail := fmt.Sprintf("deadline %s passed at %s (%d unplaced row(s) retired)",
			time.UnixMilli(m.EligibleUntilMs).In(kernel.CTLocation()).Format("15:04:05"),
			now.In(kernel.CTLocation()).Format("15:04:05"), closed)
		scope.note(sc.ID, "refused: "+pictureClassWindowClosed+": "+detail)
		key := plan.PlanID + ":" + strconv.Itoa(plan.Version) + ":" + sc.ID + ":picture"
		if armRefusalChanged(&at.armRefusalLast, key, pictureClassWindowClosed) {
			n := 0
			if at.store != nil {
				n, _ = store.IncArmRefusal(at.store, at.id, kernel.PlanTradeDateFor(plan), plan.Session, pictureClassWindowClosed)
			}
			at.logWarnf("⏱ picture scenario %s %s: %s — %s (%s this session: %d)", plan.Session, sc.ID, pictureWindowClosedNeverPlaced, detail, pictureClassWindowClosed, n)
		}
		return PictureEvidence{}, false
	}
	if !kernel.MachineEligibleAt(sc, nowMs) {
		scope.note(sc.ID, fmt.Sprintf("waiting: picture window opens at %s", time.UnixMilli(m.EligibleFromMs).In(kernel.CTLocation()).Format("15:04:05")))
		return PictureEvidence{}, false
	}
	if live, ok := at.pictureRunEpoch(); !ok || m.RunEpoch != live {
		at.refusePictureScenario(plan, sc, pictureClassPreviousRun,
			fmt.Sprintf("recorded by a previous run (reload) — not placed (epoch %d, live %s)", m.RunEpoch, pictureEpochText(live, ok)), scope)
		return PictureEvidence{}, false
	}
	ev, err := pictureScenarioEvidence(sc)
	if err != nil {
		at.refusePictureScenario(plan, sc, pictureClassEvidence, err.Error()+" — fail-closed", scope)
		return PictureEvidence{}, false
	}
	if ev.RRFloor <= 0 {
		at.refusePictureScenario(plan, sc, pictureClassEvidence, "evidence carries no R:R floor — fail-closed", scope)
		return PictureEvidence{}, false
	}
	return ev, true
}

// pictureScenarioGeometry is D10/D11 for one leg, after the zone composer: a
// Picture leg must be a market_in_zone leg, and its R:R at the FAR edge — the
// worst fill, with the AUTHORED stop (never widened) — must clear the
// evidence's floor. "" = pass.
func pictureScenarioGeometry(zl zoneLeg, side string, leg kernel.PlanArmLeg, ev PictureEvidence) string {
	if !zl.on {
		return "a Picture scenario must be a market_in_zone leg — fail-closed"
	}
	far, stop, target := zl.v.Far, leg.Stop, leg.Target
	rr := 0.0
	switch side {
	case "long":
		if far > stop && stop > 0 {
			rr = (target - far) / (far - stop)
		}
	case "short":
		if stop > far && far > 0 {
			rr = (far - target) / (stop - far)
		}
	}
	if rr+1e-9 < ev.RRFloor {
		return fmt.Sprintf("R:R %.2f at the far edge %.2f below the Picture floor %.2f (stop %.2f target %.2f, never widened)", rr, far, ev.RRFloor, stop, target)
	}
	return ""
}

// stampPictureSource writes the machine source onto an authored row (a
// planner scenario writes nothing — every W5 column stays empty / NULL).
func stampPictureSource(row *store.ArmedOrderDB, sc kernel.PlanScenario) {
	if row == nil || sc.Machine == nil || sc.Source == "" {
		return
	}
	until, epoch := sc.Machine.EligibleUntilMs, sc.Machine.RunEpoch
	row.Source = sc.Source
	row.SourceRef = sc.Machine.Ref
	row.SourceRule = sc.Machine.Rule
	row.EligibleUntilMs = &until
	row.SourceRunEpoch = &epoch
}

// plannerScenariosOnly is the doc one_setup evaluates (D8 ruling: one_setup
// governs PLANNER plays; a machine scenario is admitted by its own switch).
// A doc with no machine scenario is returned as is.
func plannerScenariosOnly(doc kernel.PlanDoc) kernel.PlanDoc {
	n := 0
	for _, s := range doc.Scenarios {
		if kernel.IsMachineScenario(s) {
			n++
		}
	}
	if n == 0 {
		return doc
	}
	out := doc
	out.Scenarios = make([]kernel.PlanScenario, 0, len(doc.Scenarios)-n)
	for _, s := range doc.Scenarios {
		if !kernel.IsMachineScenario(s) {
			out.Scenarios = append(out.Scenarios, s)
		}
	}
	return out
}

// closePictureRowsForRef retires the opportunity's UNPLACED rows (armed, no
// broker signal) as never placed. Returns how many it retired.
func (at *AutoTrader) closePictureRowsForRef(ledger *store.ArmedOrderStore, ref string, now time.Time) int {
	if ledger == nil || strings.TrimSpace(ref) == "" {
		return 0
	}
	rows, err := ledger.ListNonTerminal(at.id)
	if err != nil {
		return 0
	}
	n := 0
	for _, r := range rows {
		if r.TraderID != at.id || r.SourceRef != ref || !store.IsUnplacedArm(r.State, r.SignalID) {
			continue
		}
		if at.retirePictureRow(ledger, r, pictureWindowClosedNeverPlaced, "window_closed") {
			n++
		}
	}
	return n
}

// retirePictureRow moves ONE unplaced Picture row to terminal cancelled with
// reason, counted under picture_scenario:<event>. Nothing is at the broker
// under an unplaced row, so no wire call is needed or made.
func (at *AutoTrader) retirePictureRow(ledger *store.ArmedOrderStore, r store.ArmedOrderDB, reason, event string) bool {
	if err := ledger.SetState(r.ID, store.StateCancelled, reason); err != nil {
		at.logWarnf("📷 picture row %d (%s leg %d) retire failed: %v", r.ID, r.Scenario, r.LegIndex+1, err)
		return false
	}
	n := 0
	if at.store != nil {
		n, _ = store.IncSystemCounter(at.store, pictureClassSharedPrefix+event)
	}
	at.logWarnf("📷 picture row %d %s %s leg %d RETIRED — %s (never placed, nothing at the broker; %s%s recorded: %d)",
		r.ID, r.Session, r.Scenario, r.LegIndex+1, reason, pictureClassSharedPrefix, event, n)
	return true
}

// pictureRowRefusedAtPlacement is the send-point half of the machine gate
// (placeZoneRow calls it FIRST for a source row): a row past its deadline, a
// row recorded under another run epoch (or with no run alive), a row with no
// deadline, and a row of an unknown source are never placed — each goes
// terminal with a named reason, counted (CTO 1790192366762). true = handled.
func (at *AutoTrader) pictureRowRefusedAtPlacement(p zonePass, r store.ArmedOrderDB) bool {
	if r.Source == "" {
		return false
	}
	reason, event := "", ""
	live, liveOK := at.pictureRunEpoch()
	switch {
	case r.Source != store.ArmSourcePicture:
		reason, event = fmt.Sprintf("unknown row source %q — never placed (fail-closed)", r.Source), "unknown_source"
	case r.EligibleUntilMs == nil:
		reason, event = "picture row carries no eligibility deadline — never placed (fail-closed)", "no_deadline"
	case p.now.UnixMilli() > *r.EligibleUntilMs:
		reason, event = pictureWindowClosedNeverPlaced, "window_closed"
	case r.SourceRunEpoch == nil || !liveOK || *r.SourceRunEpoch != live:
		rec := "none"
		if r.SourceRunEpoch != nil {
			rec = strconv.FormatInt(*r.SourceRunEpoch, 10)
		}
		reason = fmt.Sprintf("recorded by a previous run (epoch %s, live %s) — not placed", rec, pictureEpochText(live, liveOK))
		event = "previous_run"
	default:
		return false
	}
	at.setZoneVerdict(p, r, "refused: picture: "+event)
	p.scope.note(r.Scenario, "refused: picture: "+reason)
	at.retirePictureRow(p.ledger, r, reason, event)
	return true
}

// pictureDeadlineCancel is D6's resting half, run beside zoneRestCap: a
// Picture limit still resting past its eligibility deadline gets its cancel
// REQUESTED — the filled-arm guard first, then the wire cancel, then
// cancel_pending with the reason (a send is not a settlement). For a source
// row this deadline replaces zone_rest_max_min. A row with no deadline is
// treated as past it (fail-closed).
func (at *AutoTrader) pictureDeadlineCancel(nt *ntTrader.TCPTrader, ledger *store.ArmedOrderStore, rows []store.ArmedOrderDB, now time.Time) {
	if ledger == nil {
		return
	}
	for _, r := range rows {
		if r.TraderID != at.id || r.Source == "" || strings.TrimSpace(r.SignalID) == "" || store.IsTerminalArmState(r.State) {
			continue
		}
		if r.State == store.StateArmed {
			continue // stamped but not yet sent: placement owns it
		}
		if r.State == store.StateCancelPending {
			continue // a cancel is already in flight
		}
		if r.EligibleUntilMs != nil && now.UnixMilli() <= *r.EligibleUntilMs {
			continue
		}
		at.requestPictureCancel(nt, ledger, r, pictureWindowClosedCancel, "deadline_cancel", now)
	}
}

// requestPictureCancel requests the cancel of ONE resting Picture entry: the
// book decides first (cancelSafetyFor — a filled entry's OCO children are
// never cancelled), then the wire, then cancel_pending with the reason. No
// broker link → cancel_pending all the same (held, never promoted — the
// cancelOtherArmsInPlan rule).
func (at *AutoTrader) requestPictureCancel(nt *ntTrader.TCPTrader, ledger *store.ArmedOrderStore, r store.ArmedOrderDB, reason, event string, now time.Time) bool {
	if nt != nil {
		if v := at.cancelSafetyFor(r, now); !v.Allow {
			if at.admitLast.changed("picture-cancel|"+r.SignalID, v.Why) {
				at.logWarnf("🛟 picture cancel REFUSED (%s) %s leg %d signal=%s — %s", reason, r.Scenario, r.LegIndex+1, shortID(r.SignalID), v.Why)
			}
			return false
		}
		if cerr := nt.CancelOrder(r.SignalID); cerr != nil {
			at.logWarnf("✕ picture cancel SEND failed (%s) %s leg %d signal=%s: %v", reason, r.Scenario, r.LegIndex+1, shortID(r.SignalID), cerr)
		}
	} else {
		at.logWarnf("✕ picture cancel UNSENDABLE (%s) %s leg %d — no broker link; row held cancel_pending, never promoted", reason, r.Scenario, r.LegIndex+1)
	}
	if err := ledger.RequestCancel(r.ID, reason, now.UnixMilli()); err != nil {
		at.logWarnf("✕ picture cancel: ledger write failed for %s leg %d: %v", r.Scenario, r.LegIndex+1, err)
		return false
	}
	n := 0
	if at.store != nil {
		n, _ = store.IncSystemCounter(at.store, pictureClassSharedPrefix+event)
	}
	at.logWarnf("✕ picture cancel REQUESTED (%s): %s %s leg %d limit %.2f signal=%s — pending broker confirmation (%s%s recorded: %d)",
		reason, r.Session, r.Scenario, r.LegIndex+1, r.EntryPx, shortID(r.SignalID), pictureClassSharedPrefix, event, n)
	return true
}

// invalidatePictureRows is D21's cleanup for Stop, a reload and Day Plan OFF:
// every non-terminal Picture row of this trader — unplaced → terminal with
// the reason; resting (a broker signal) → cancel requested. Planner rows are
// never touched (L4). The caller holds armedPassMu (the pass does; the Stop
// hook takes it).
func (at *AutoTrader) invalidatePictureRows(reason, event string, now time.Time) (retired, requested int) {
	if at == nil || at.store == nil || at.exchange != "ninjatrader" {
		return 0, 0
	}
	ledger := at.store.ArmedOrders()
	if ledger == nil {
		return 0, 0
	}
	rows, err := ledger.ListNonTerminal(at.id)
	if err != nil {
		at.logWarnf("📷 picture %s: ledger unreadable (%v) — Picture rows not invalidated this time", reason, err)
		return 0, 0
	}
	nt := at.armedTrader()
	for _, r := range rows {
		if r.TraderID != at.id || r.Source == "" || r.State == store.StateCancelPending {
			continue
		}
		if store.IsUnplacedArm(r.State, r.SignalID) {
			if at.retirePictureRow(ledger, r, "picture: "+reason+" — never placed", event) {
				retired++
			}
			continue
		}
		if strings.TrimSpace(r.SignalID) != "" && at.requestPictureCancel(nt, ledger, r, "picture: "+reason, event, now) {
			requested++
		}
	}
	return retired, requested
}

// pictureDayPlanOffSweep is the pass head's Day-Plan-OFF branch for Picture
// rows (D21): on an NT8 trader with a store and the master OFF, its Picture
// rows lose entry permission. A trader with no Picture row reads the ledger
// and does nothing.
func (at *AutoTrader) pictureDayPlanOffSweep(now time.Time) {
	if at == nil || at.store == nil || at.exchange != "ninjatrader" || at.dayPlanEnabled() {
		return
	}
	at.invalidatePictureRows(pictureDayPlanOffReason, "day_plan_off", now)
}

// retirePictureRowsOnStop is the Stop hook (D21), called from
// stopArmedEventLoop after the run epoch is cleared and the event loop is
// closed. It takes armedPassMu, so it never interleaves with a scan pass
// still in flight.
func (at *AutoTrader) retirePictureRowsOnStop(now time.Time) {
	if at == nil || at.store == nil || at.exchange != "ninjatrader" {
		return
	}
	at.armedPassMu.Lock()
	defer at.armedPassMu.Unlock()
	if retired, requested := at.invalidatePictureRows(pictureStoppedReason, "stopped", now); retired+requested > 0 {
		at.logWarnf("⏹ picture: trader stopped — %d unplaced Picture row(s) retired, %d resting cancel(s) requested (W5 D21)", retired, requested)
	}
}
