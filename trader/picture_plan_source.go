package trader

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W5 — PICTURE AS A DAY PLAN SCENARIO SOURCE (builder A).
//
// A Picture opportunity used to own a send: the evaluator claimed it and the
// seam fired a MARKET entry through its own path — a trade that could happen
// behind a card reading "No plan" (D27). The seam is now the HAND-OFF: the
// opportunity is RECORDED as a machine scenario of the Day Plan (design O′,
// CTO 1790191033566) before any order exists, and only the armed executor —
// the one entry path every plan scenario takes — may ever place it.
//
//	pictureHandOffAt: (i) the live run epoch · (ii) the live, runnable session
//	· (iii) the scenario (D3/D4/D5) · (iv) RECORD — an active plan gets one
//	machine overlay, no plan row gets a machine plan v1, a plan that is not
//	active refuses ("Day Plan says no") · (v) SETTLE the Picture row
//	place_pending → planned · (vi) POKE the event loop · (vii) one INFO line.
//
// An error before the record keeps the evaluator's "never sent → refused"
// contract (the row settles refused). Once the scenario is recorded the
// hand-off never returns an error: a failed settle is left for the D17 sweep
// (sweepInterruptedPictureHandOffsAt), which finds the record and settles the
// row "planned" — never "refused" behind a scenario that may still trade.

func init() {
	// Go runs a package's init functions in file-name order: picture_htf_send.go
	// binds the retired market-entry send first, and this binding (a later
	// file) wins. Builder C deletes that file; this binding stays.
	pictureHtfSubmitSeam = pictureHtfHandOffSeam
	// D17 — the executor's pass head runs the interrupted-hand-off sweep once
	// per pass through this hook (declared beside the executor, bound here).
	pictureHandOffSweepHook = func(at *AutoTrader, now time.Time) { at.sweepInterruptedPictureHandOffsAt(now) }
}

// pictureHtfHandOffSeam is the production submission seam: the evaluator's row
// becomes PictureEvidence (the ONLY reader of the row is picture_evidence.go)
// and is handed to the Day Plan. It never reaches the wire.
func pictureHtfHandOffSeam(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, _ float64, now time.Time) error {
	ev, err := pictureEvidenceFrom(e, row, stopPx, targetPx, now)
	if err != nil {
		return err
	}
	return e.at.pictureHandOffAt(ev, now)
}

// pictureHandOffLocks serializes one trader's hand-offs against its D17 sweep
// (the sweep must never judge a row a hand-off is still recording). Keyed by
// trader id so W5 adds no AutoTrader field.
var pictureHandOffLocks sync.Map // trader id → *sync.Mutex

func (at *AutoTrader) pictureHandOffLock() *sync.Mutex {
	v, _ := pictureHandOffLocks.LoadOrStore(at.id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// pictureRecord is where a machine scenario was recorded.
type pictureRecord struct {
	PlanID         string
	PlanVersion    int
	OverlayVersion int // 0 = the scenario lives in a machine plan's own doc
	ScenarioID     string
	MachinePlan    bool // the no-plan door wrote machine plan v1
	Existing       bool // the opportunity was already recorded (idempotent)
}

// stageReason is the Picture row's stage_reason once it settles "planned".
func (r pictureRecord) stageReason() string {
	if r.OverlayVersion > 0 {
		return fmt.Sprintf("Day Plan scenario %s · %s v%d · o%d", r.ScenarioID, r.PlanID, r.PlanVersion, r.OverlayVersion)
	}
	return fmt.Sprintf("Day Plan scenario %s · %s v%d · machine plan", r.ScenarioID, r.PlanID, r.PlanVersion)
}

// where names the record for the INFO line.
func (r pictureRecord) where() string {
	if r.OverlayVersion > 0 {
		return fmt.Sprintf("%s v%d overlay o%d", r.PlanID, r.PlanVersion, r.OverlayVersion)
	}
	if r.MachinePlan {
		return fmt.Sprintf("%s machine plan v%d", r.PlanID, r.PlanVersion)
	}
	return fmt.Sprintf("%s v%d (in the plan doc)", r.PlanID, r.PlanVersion)
}

// pictureHandOffAt records one Picture opportunity as a Day Plan scenario.
func (at *AutoTrader) pictureHandOffAt(ev PictureEvidence, now time.Time) error {
	if at == nil || at.store == nil {
		return fmt.Errorf("picture hand-off refused: no trader store — nothing recorded")
	}
	mu := at.pictureHandOffLock()
	mu.Lock()
	defer mu.Unlock()

	// (i) the live run epoch — a Stop clears it; a reload starts a new one.
	epoch, ok := at.pictureRunEpoch()
	if !ok {
		return fmt.Errorf("picture hand-off refused: trader not running (no live run epoch) — nothing recorded")
	}
	if !at.dayPlanEnabled() {
		return fmt.Errorf("picture hand-off refused: Day Plan is off — nothing recorded")
	}
	// (ii) the live, runnable session — Picture never trades a session the
	// trader does not run (the provider's own two checks).
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok || sess == nil {
		return fmt.Errorf("picture hand-off refused: no session is live at %s — nothing recorded", kernel.ClockCTSeconds(now))
	}
	if runnable, why := at.sessionRunnable(sess); !runnable {
		return fmt.Errorf("picture hand-off refused: session %s is not runnable for this trader (%s) — nothing recorded", sess.Name, why)
	}
	if ev.EvalAtMs > ev.WindowCloseMs {
		return fmt.Errorf("picture hand-off refused: the eligibility window closed at %s, before the hand-off — nothing recorded", kernel.ClockCTSeconds(time.UnixMilli(ev.WindowCloseMs)))
	}
	// (iii) the scenario (its P id is minted against the doc it joins, at
	// record time).
	sc, err := at.pictureScenarioFrom(ev, epoch)
	if err != nil {
		return err
	}
	// (iv) RECORD.
	tradeDate := sessionChainDate(sess, now)
	rec, err := at.recordPictureScenario(sess, tradeDate, sc, now)
	if err != nil {
		return err
	}
	// (v) SETTLE — from here on the scenario exists; never return an error.
	if moved, serr := at.store.PictureHtfHandOff(ev.OppKey, ev.ClaimID, rec.stageReason()); serr != nil || !moved {
		// A repeat hand-off of an opportunity already settled is not a loss.
		if cur, ok, gerr := at.store.PictureHtfGet(ev.OppKey); serr != nil || gerr != nil || !ok || cur.Stage != store.PictureStagePlanned {
			at.logWarnf("🖼 picture hand-off: %s recorded as %s in %s but the opportunity row did not settle planned (moved=%v err=%v) — left for the interrupted-hand-off sweep", store.RedactPictureOppKey(ev.OppKey), rec.ScenarioID, rec.where(), moved, serr)
		}
	}
	// (vi) POKE — never a pass on this (the bar sink's) goroutine.
	at.zoneArmActive.Store(true)
	if l := at.armedEvent.Load(); l != nil {
		l.poke()
	}
	// (vii) one INFO line.
	if rec.Existing {
		at.logInfof("🖼 picture → Day Plan scenario %s (%s) ref %s — already recorded; nothing appended (idempotent on the opportunity)", rec.ScenarioID, rec.where(), store.RedactPictureOppKey(ev.OppKey))
	} else {
		at.logInfof("🖼 picture → Day Plan scenario %s (%s) ref %s window until %s", rec.ScenarioID, rec.where(), store.RedactPictureOppKey(ev.OppKey), kernel.ClockCTSeconds(time.UnixMilli(ev.WindowCloseMs)))
	}
	return nil
}

// pictureScenarioFrom builds the machine scenario (D3/D4/D5). The id is a
// placeholder until the record mints it against the doc it joins.
func (at *AutoTrader) pictureScenarioFrom(ev PictureEvidence, epoch int64) (kernel.PlanScenario, error) {
	dir := strings.ToLower(strings.TrimSpace(ev.Direction))
	if dir != "long" && dir != "short" {
		return kernel.PlanScenario{}, fmt.Errorf("picture hand-off refused: direction %q is not long|short — nothing recorded", ev.Direction)
	}
	// D5 — the zone is exactly the price set the evaluator admitted: from its
	// reference (the newest completed 5m close) to the newest 1m close; one
	// price when the 1m close is unknown (0 is never a price).
	lo, hi := ev.EntryRef, ev.EntryRef
	if ev.LatestClose > 0 {
		lo, hi = math.Min(ev.EntryRef, ev.LatestClose), math.Max(ev.EntryRef, ev.LatestClose)
	}
	evidence, err := json.Marshal(ev)
	if err != nil {
		return kernel.PlanScenario{}, fmt.Errorf("picture hand-off refused: evidence did not serialize: %v — nothing recorded", err)
	}
	h1Close := time.UnixMilli(ev.H1CloseMs)
	beyond := "above"
	if dir == "short" {
		beyond = "below"
	}
	sc := kernel.PlanScenario{
		ID: "P1",
		Trigger: fmt.Sprintf("H1 close %.2f %s the 4H body %.2f (%s v%d, H1 closed %s)",
			ev.H1NewClose, beyond, ev.H1Boundary, ev.Rule, ev.RuleVer, kernel.ClockCT(h1Close)),
		Condition:   "acceptance",
		Direction:   dir,
		TargetChain: []float64{ev.Target},
		Invalid: fmt.Sprintf("back inside the 4H body %.2f–%.2f, or the eligibility window closes (%s)",
			ev.BodyBot, ev.BodyTop, kernel.ClockCTSeconds(time.UnixMilli(ev.WindowCloseMs))),
		Quality:   "B", // the machine never self-grades higher (min_scenario_quality still applies)
		Economics: &kernel.ScenarioEconomics{Version: 1, EntryZone: []float64{lo, hi}},
		Arm: &kernel.PlanArmSpec{Enabled: true, Entry: ev.EntryRef, Stop: ev.Stop, Target: ev.Target,
			Policy: kernel.EntryPolicyMarketInZone, WaitConfirm: false},
		Source: kernel.ScenarioSourcePicture,
		Machine: &kernel.PlanMachineSource{
			Rule: ev.Rule, RuleVer: ev.RuleVer, Ref: ev.OppKey,
			EligibleFromMs: ev.EvalAtMs, EligibleUntilMs: ev.WindowCloseMs,
			RunEpoch: epoch, Evidence: evidence,
		},
	}
	// The zone as the executor will judge it (width ≤ zone_max_pts, the entry
	// inside, the bracket outside, a tick inside after inward rounding) …
	tick := market.FuturesTickSize(at.futuresSymbol())
	if tick <= 0 {
		tick = 0.25
	}
	maxPts, _ := store.ResolveZoneMaxPts(at.dayPlanCfg())
	v := kernel.ArmZoneVerdict(sc, ev.EntryRef, ev.Stop, ev.Target, dir, tick, maxPts)
	if v.Code != "" {
		return kernel.PlanScenario{}, fmt.Errorf("picture hand-off refused: zone %.2f–%.2f (%.2f pts, zone_max_pts %.2f) — market_in_zone:%s — nothing recorded", lo, hi, hi-lo, maxPts, v.Code)
	}
	// … and R:R at the FAR edge (the limit price and the worst fill) against
	// Picture's own floor, max(knob, strategy floor) (D11).
	risk, reward := v.Far-ev.Stop, ev.Target-v.Far
	if dir == "short" {
		risk, reward = ev.Stop-v.Far, v.Far-ev.Target
	}
	if risk <= 0 {
		return kernel.PlanScenario{}, fmt.Errorf("picture hand-off refused: stop %.2f is not beyond the far edge %.2f — nothing recorded", ev.Stop, v.Far)
	}
	if rr := reward / risk; rr+1e-9 < ev.RRFloor {
		return kernel.PlanScenario{}, fmt.Errorf("picture hand-off refused: R:R %.2f at the far edge %.2f is below the Picture floor %.2f — nothing recorded", rr, v.Far, ev.RRFloor)
	}
	return sc, nil
}

// machinePlanDoc is the no-plan door's doc: the machine scenario alone, and
// words that say exactly what the plan is.
func machinePlanDoc(sess string, now time.Time, sc kernel.PlanScenario) kernel.PlanDoc {
	return kernel.PlanDoc{
		Reasoning: fmt.Sprintf("MACHINE-AUTHORED (Picture HTF rule v%d) — no AI plan existed for %s at %s; superseded by the first AI plan",
			sc.Machine.RuleVer, sess, kernel.ClockCT(now)),
		Bias:           kernel.PlanBias{Direction: "neutral"},
		Levels:         []kernel.PlanLevel{},
		Scenarios:      []kernel.PlanScenario{sc},
		NoTrade:        []string{},
		DeathCondition: "machine plan: ends when an AI plan is written",
	}
}

// recordPictureScenario is the D1 door: no row → machine plan v1; an active
// row → one machine overlay; a row that is not active → refuse.
func (at *AutoTrader) recordPictureScenario(sess *kernel.SessionDef, tradeDate string, sc kernel.PlanScenario, now time.Time) (pictureRecord, error) {
	plans := at.store.Plan()
	row, err := plans.GetLatestPlanForTraderSession(tradeDate, sess.Name, at.id)
	if err != nil {
		return pictureRecord{}, fmt.Errorf("picture hand-off refused: plan lookup %s %s failed: %v — nothing recorded", tradeDate, sess.Name, err)
	}
	if row == nil {
		// (b) the no-plan door — recorded BEFORE any authorization.
		sc.ID = kernel.NextMachineScenarioID(kernel.PlanDoc{})
		if err := kernel.ValidateMachineScenario(kernel.PlanDoc{}, sc); err != nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: the machine scenario is invalid: %v — nothing recorded", err)
		}
		doc := machinePlanDoc(sess.Name, now, sc)
		if err := kernel.ValidatePlanDocWithCaps(&doc, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios); err != nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: the machine plan is invalid: %v — nothing recorded", err)
		}
		blob, err := json.Marshal(doc)
		if err != nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: the machine plan did not serialize: %v — nothing recorded", err)
		}
		mp := &store.PlanDB{
			PlanID:        plans.ResolvePlanID(tradeDate, sess.Name, at.id),
			StrategyID:    at.id,
			TradeDate:     tradeDate,
			Session:       sess.Name,
			TriggerReason: kernel.MachinePlanTriggerPicture,
			Lifecycle:     "active",
			ModelID:       kernel.MachinePlanModelID,
			Doc:           string(blob),
		}
		wrote, err := plans.AppendPlanIfAbsent(mp)
		if err != nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: machine plan write failed: %v — nothing recorded", err)
		}
		if wrote {
			return pictureRecord{PlanID: mp.PlanID, PlanVersion: mp.Version, ScenarioID: sc.ID, MachinePlan: true}, nil
		}
		// A row landed between the read and the insert (the planner's write
		// won the single writer): continue on the row that exists now.
		if row, err = plans.GetLatestPlanForTraderSession(tradeDate, sess.Name, at.id); err != nil || row == nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: the machine plan was not written and no plan row is readable for %s %s (err %v) — nothing recorded", tradeDate, sess.Name, err)
		}
	}
	// Active exactly as the provider judges it: lifecycle "active".
	if row.Lifecycle != "active" {
		return pictureRecord{}, fmt.Errorf("picture hand-off refused: Day Plan says no (%s) — plan %s v%d for %s %s is not active; nothing recorded", row.Lifecycle, row.PlanID, row.Version, sess.Name, tradeDate)
	}
	return at.appendPictureOverlay(row, sc)
}

// appendPictureOverlay is the (a) door: one append-only machine overlay on an
// ACTIVE plan version, idempotent on the opportunity. The admission check runs
// inside the plan store's single writer against the overlays already stored,
// so two hand-offs of one opportunity cannot both append, and the P id is
// minted there — never taken twice.
func (at *AutoTrader) appendPictureOverlay(row *store.PlanDB, sc kernel.PlanScenario) (pictureRecord, error) {
	ref := sc.Machine.Ref
	o := &store.PlanOverlayDB{
		OverlayID:   "picture:" + ref,
		PlanID:      row.PlanID,
		PlanVersion: row.Version,
		Origin:      kernel.MachineOverlayOriginPicture,
	}
	var hit *pictureRecord
	ver, appended, err := at.store.Plan().AppendOverlayChecked(o, func(existing []*store.PlanOverlayDB) (bool, error) {
		pf, perr := kernel.ResolvePlanFinal([]byte(row.Doc), kernel.OverlayRefsFrom(existing))
		if perr != nil {
			return false, fmt.Errorf("plan %s v%d does not parse: %v", row.PlanID, row.Version, perr)
		}
		if prev, ok := kernel.MachineScenarioByRef(pf.Doc, ref); ok {
			r := pictureRecord{PlanID: row.PlanID, PlanVersion: row.Version, ScenarioID: prev.ID, Existing: true}
			for _, ma := range pf.MachineApplied {
				if ma.Ref == ref {
					r.OverlayVersion = ma.OverlayVersion
				}
			}
			hit = &r
			return true, nil
		}
		for _, e := range existing {
			if !kernel.IsMachineOverlayOrigin(e.Origin) {
				continue
			}
			if prev, perr := kernel.MachineScenarioFromPatch(e.Patch); perr == nil && prev.Machine != nil && prev.Machine.Ref == ref {
				// Recorded but not folding (a skipped machine overlay): never
				// append a second record of the same opportunity.
				hit = &pictureRecord{PlanID: row.PlanID, PlanVersion: row.Version, OverlayVersion: e.OverlayVersion, ScenarioID: prev.ID, Existing: true}
				return true, nil
			}
		}
		sc.ID = nextPictureScenarioID(pf.Doc, existing)
		if verr := kernel.ValidateMachineScenario(pf.Doc, sc); verr != nil {
			return false, verr
		}
		patch, merr := kernel.MachineOverlayPatch(sc)
		if merr != nil {
			return false, merr
		}
		o.Patch = patch
		return false, nil
	})
	if err != nil {
		return pictureRecord{}, fmt.Errorf("picture hand-off refused: machine overlay on %s v%d not recorded: %v — nothing recorded", row.PlanID, row.Version, err)
	}
	if !appended {
		if hit == nil {
			return pictureRecord{}, fmt.Errorf("picture hand-off refused: machine overlay on %s v%d skipped without a record — nothing recorded", row.PlanID, row.Version)
		}
		return *hit, nil
	}
	return pictureRecord{PlanID: row.PlanID, PlanVersion: row.Version, OverlayVersion: ver, ScenarioID: sc.ID}, nil
}

// nextPictureScenarioID mints P<n> past every P id the version holds: those
// in the resolved doc AND those in any machine overlay (even one the fold
// skipped), so an id is never reused within a version.
func nextPictureScenarioID(doc kernel.PlanDoc, existing []*store.PlanOverlayDB) string {
	all := kernel.PlanDoc{Scenarios: append([]kernel.PlanScenario(nil), doc.Scenarios...)}
	for _, e := range existing {
		if !kernel.IsMachineOverlayOrigin(e.Origin) {
			continue
		}
		if s, err := kernel.MachineScenarioFromPatch(e.Patch); err == nil {
			all.Scenarios = append(all.Scenarios, kernel.PlanScenario{ID: s.ID})
		}
	}
	return kernel.NextMachineScenarioID(all)
}

// sweepInterruptedPictureHandOffsAt is the D17 sweep: a Picture row left
// place_pending with a synthetic claim id and no submission stamp (a crash or
// a failed settle between claim, record and settle) is settled from the
// RECORD — "planned" when its machine scenario exists, "refused" (never sent)
// only once its eligibility window has closed with no record; a row whose
// window is still open is left alone (its hand-off may not have run yet).
// Builder B's pass head calls it through pictureHandOffSweepHook.
func (at *AutoTrader) sweepInterruptedPictureHandOffsAt(now time.Time) {
	if at == nil || at.store == nil {
		return
	}
	rows, err := at.store.PictureHtfHandOffPendingByTrader(at.id)
	if err != nil || len(rows) == 0 {
		return
	}
	mu := at.pictureHandOffLock()
	mu.Lock()
	defer mu.Unlock()
	for _, r := range rows {
		if rec, ok := at.pictureScenarioInLivePlan(r.OppKey, now); ok {
			at.settleSweptHandOff(r, rec)
			continue
		}
		found, ok, ferr := at.store.PictureHandOffRecordedFor(at.id, r.OppKey)
		if ferr != nil {
			continue // unknown ≠ absent: never refuse on a failed read
		}
		if ok {
			at.settleSweptHandOff(r, pictureRecord{PlanID: found.PlanID, PlanVersion: found.PlanVersion, OverlayVersion: found.OverlayVersion, ScenarioID: "(recorded)"})
			continue
		}
		if r.WindowClose > 0 && now.UnixMilli() > r.WindowClose {
			if moved, rerr := at.store.PictureHtfRefuse(r.OppKey, "refused", "hand-off interrupted — never sent"); rerr == nil && moved {
				at.logWarnf("🖼 picture hand-off sweep: %s was claimed but never recorded as a Day Plan scenario and its window closed at %s — refused (never sent)", store.RedactPictureOppKey(r.OppKey), kernel.ClockCTSeconds(time.UnixMilli(r.WindowClose)))
			}
		}
	}
}

func (at *AutoTrader) settleSweptHandOff(r store.PictureHtfOpportunityDB, rec pictureRecord) {
	if moved, err := at.store.PictureHtfHandOff(r.OppKey, r.SignalID, rec.stageReason()+" (settled by the interrupted-hand-off sweep)"); err == nil && moved {
		at.logInfof("🖼 picture hand-off sweep: %s settled planned — %s", store.RedactPictureOppKey(r.OppKey), rec.where())
	}
}

// pictureScenarioInLivePlan resolves the plan the executor reads right now
// (the provider's session → chain date → latest row, folded by the ONE fold)
// and finds the opportunity's machine scenario in it.
func (at *AutoTrader) pictureScenarioInLivePlan(ref string, now time.Time) (pictureRecord, bool) {
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok || sess == nil {
		return pictureRecord{}, false
	}
	row, err := at.store.Plan().GetLatestPlanForTraderSession(sessionChainDate(sess, now), sess.Name, at.id)
	if err != nil || row == nil {
		return pictureRecord{}, false
	}
	ovs, err := at.store.Plan().ListOverlays(row.PlanID, row.Version)
	if err != nil {
		return pictureRecord{}, false
	}
	pf, err := kernel.ResolvePlanFinal([]byte(row.Doc), kernel.OverlayRefsFrom(ovs))
	if err != nil {
		return pictureRecord{}, false
	}
	sc, ok := kernel.MachineScenarioByRef(pf.Doc, ref)
	if !ok {
		return pictureRecord{}, false
	}
	rec := pictureRecord{PlanID: row.PlanID, PlanVersion: row.Version, ScenarioID: sc.ID, MachinePlan: store.IsMachinePlan(row)}
	for _, ma := range pf.MachineApplied {
		if ma.Ref == ref {
			rec.OverlayVersion = ma.OverlayVersion
		}
	}
	return rec, true
}
