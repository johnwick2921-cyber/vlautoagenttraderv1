package trader

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/telemetry"
	ntTrader "nofx/trader/ninjatrader"
)

// PICTURE-HTF EVALUATOR (2026-09-20) — the deterministic half of the owner's
// two-picture mode. It runs from native bar events, NOT from AI cycles: the
// H1 completion, the following 5m interval and the freshness verdicts are
// evaluated directly from received bars, and the AI provides commentary +
// momentum context only.
//
// Ownership: every opportunity is claimed through the store (unique key +
// atomic confirmed→place_pending transition), so repeated frames, restarts and
// the OTHER executor cannot double-submit. Only RECEIVED broker frames may
// move a row to working/filled/rejected.

// PictureHtfEvaluator is the per-trader evaluation state. Zero value is
// unusable; NewPictureHtfEvaluator builds it from the resolved strategy knob.
type PictureHtfEvaluator struct {
	mu      sync.Mutex
	at      *AutoTrader
	cfg     store.PictureHtfConfig
	enabled bool

	// Evaluation state.
	lastH1CloseEval int64 // the newest completed H1 close already scanned
	levels          []kernel.PictureHtfLevel
	levelsEval4H    int64 // the 4H close time the levels snapshot was built from
	freshest5mAt    time.Time

	// H1 close series for the advisory momentum stall (last three closes).
	h1Closes []float64

	// capWarned dedupes the once-per-state "mode unavailable" log.
	capWarned bool

	// foreignFrames counts live frames whose contract is definitely NOT the
	// one this trader is on. READ by the accessor; never inferred.
	foreignFrames int64

	// holdRefusedKey dedupes the maintenance-hold refusal (count + WARN) to
	// once per opportunity rather than once per frame.
	holdRefusedKey string

	// levelsSymbol is the symbol the levels snapshot belongs to (W-EXEC-TRUTH
	// W0 defect 4: the snapshot used to be shared across symbols).
	levelsSymbol string

	// pendingAdmission is the evidence the evaluator admitted this
	// opportunity on, handed to the send's re-admission within the SAME call
	// (both run under e.mu).
	pendingAdmission *pictureAdmission
}

// ownsSymbol reports whether a frame's symbol is this trader's instrument.
// The live sink fans every symbol's frames out to every trader; an MNQ trader
// evaluating ES bars claimed ES opportunities and sent them on MNQ with ES
// geometry (W-EXEC-TRUTH W0 defect 4).
func (e *PictureHtfEvaluator) ownsSymbol(symbol string) bool {
	_, own := e.at.latchScope()
	return instrumentRoot(symbol) != "" && instrumentRoot(symbol) == instrumentRoot(own)
}

// NewPictureHtfEvaluator builds the evaluator from the resolved strategy knob.
func NewPictureHtfEvaluator(at *AutoTrader, cfg store.PictureHtfConfig) *PictureHtfEvaluator {
	return &PictureHtfEvaluator{at: at, cfg: cfg, enabled: cfg.Enabled}
}

// Enabled reports the resolved mode switch.
func (e *PictureHtfEvaluator) Enabled() bool { return e != nil && e.enabled }

// pictureHtfSubmitSeam is the submission seam: the production wiring calls the
// concrete NT8 market-entry method (with the before-send persistence callback);
// tests replace it to prove the admission sequence. The seam receives the
// already-claimed opportunity row and the computed geometry.
//
// now is the EVALUATION's clock (W-EXEC-TRUTH W0, class 60): the send-time
// re-checks read the same instant the evaluator judged, never the wall.
var pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, now time.Time) error {
	return fmt.Errorf("picture_htf submit seam unbound (the NT8 market-entry method is wired in the next wave commit)")
}

// pictureHtfCapabilityProven gates the mode on the AddOn's evidence surface
// (final + emitted_at bar markers, rejection reasons) — proven by RECEIPT of
// the far-side build id, never assumed. Tests override it.
var pictureHtfCapabilityProven = func(at *AutoTrader) bool {
	if at == nil {
		return false
	}
	tcp, ok := at.trader.(*ntTrader.TCPTrader)
	if !ok {
		return false
	}
	return tcp.FarSideProves(ntwire.MinAddonBuildPictureHtf)
}

// bars reads completed bars of a timeframe from the live provider (nil-guarded).
func (e *PictureHtfEvaluator) bars(symbol, tf string, n int, nowMs int64) []market.Kline {
	if market.FuturesBarsProvider == nil {
		return nil
	}
	raw := market.FuturesBarsProvider(symbol, tf, n)
	var out []market.Kline
	for _, b := range raw {
		// Only bars the AddOn PROVED closed count. The mode is gated on the
		// capability that guarantees final markers, so an unmarked bar is a
		// forming one — wall-clock inference is not the law here.
		if b.CloseTime < nowMs && b.Final {
			out = append(out, b)
		}
	}
	return out
}

// pictureHtfDepthMargin is how many 4H candles BEYOND the requirement the
// evaluator fetches. It must be at least 2 and is deliberately larger: the
// provider returns the TAIL of the history it holds, so that tail always
// contains the candle still forming, and the break snapshot additionally cuts
// off at the breaking candle's open, removing one more. Asking for exactly the
// requirement and then filtering to completed candles can therefore never
// satisfy the requirement — not rarely, but by construction.
const pictureHtfDepthMargin = 4

// PictureDepthEvidence is what one evaluation KNOWS about its own 4H history.
// Every field is READ from the bars in hand; none is assumed. It is part of
// the evidence contract the Day Plan scenario source consumes.
type PictureDepthEvidence struct {
	Fetched   int // candles asked of the provider
	Completed int // of those, closed before the cutoff and PROVED final
	Required  int // PivotWindow + 4
}

// OK reports whether the two pictures may be drawn at all.
func (d PictureDepthEvidence) OK() bool { return d.Required > 0 && d.Completed >= d.Required }

// Reason is the refusal text, carrying the true counts rather than a verdict.
func (d PictureDepthEvidence) Reason() string {
	return fmt.Sprintf("insufficient depth %d/%d completed 4H candles", d.Completed, d.Required)
}

// depth4H fetches the 4H history with margin and reports what actually came
// back, so callers refuse on evidence instead of on a guess.
func (e *PictureHtfEvaluator) depth4H(symbol string, cutoffMs int64) (PictureDepthEvidence, []market.Kline) {
	required := e.cfg.PivotWindow + 4
	ask := required + pictureHtfDepthMargin
	bars := e.bars(symbol, "4h", ask, cutoffMs)
	return PictureDepthEvidence{Fetched: ask, Completed: len(bars), Required: required}, bars
}

// rebuildLevels recomputes the 4H body-pivot snapshot from completed bars.
func (e *PictureHtfEvaluator) rebuildLevels(symbol string, nowMs int64) PictureDepthEvidence {
	if e.levelsSymbol != symbol {
		// the snapshot is per symbol — never another instrument's levels
		e.levels, e.levelsEval4H, e.levelsSymbol = nil, 0, symbol
	}
	dep, bars := e.depth4H(symbol, nowMs)
	if !dep.OK() {
		// Too little history to draw the higher picture. Drop any snapshot
		// built from a deeper past so a later short read cannot keep trading
		// on levels this evaluation cannot justify.
		e.levels, e.levelsEval4H = nil, 0
		return dep
	}
	newest := bars[len(bars)-1].CloseTime
	if newest == e.levelsEval4H {
		return dep
	}
	e.levels = kernel.BodyPivots4H(bars, e.cfg.PivotWindow)
	e.levelsEval4H = newest
	return dep
}

// frameContract is the contract the frame's bars agree on. It returns "" when
// the frame names none (an AddOn older than the 2026-09-11 ruling) or when its
// bars disagree — both are UNKNOWN, and unknown is never reported as a match.
func frameContract(bars []market.Kline) string {
	out := ""
	for _, b := range bars {
		if b.Contract == "" {
			continue
		}
		if out == "" {
			out = b.Contract
			continue
		}
		if out != b.Contract {
			return ""
		}
	}
	return out
}

// ForeignContractFrames reports how many live frames named a contract other
// than the one this trader is on. A rising count means the tape and the
// trader disagree about the instrument — during a roll, or after the
// reconnect path that re-subscribes without a fresh ACK.
func (e *PictureHtfEvaluator) ForeignContractFrames() int64 {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.foreignFrames
}

// OnBars is the event entry point: the trader's bar consumers call it for
// native LIVE bar updates. receivedAt is the Go-side receipt time; historical
// or backfill frames must NOT call it. Non-blocking for the caller.
func (e *PictureHtfEvaluator) OnBars(symbol, tf string, bars []market.Kline, receivedAt time.Time) {
	if e == nil || !e.enabled || e.at == nil || len(bars) == 0 {
		return
	}
	if tf != "4h" && tf != "1h" && tf != "5m" {
		return
	}
	if !e.ownsSymbol(symbol) {
		return // another instrument's frame — never this trader's opportunity
	}
	// W4/D21 identity. The AddOn names the front month on EVERY bar frame, so
	// when the frame names one and this trader is provably on another, the
	// frame is a different instrument's tape and must never drive an
	// evaluation — across a roll the SYMBOL alone cannot tell them apart.
	// Read outside the evaluator's lock: currentContract reaches into the TCP
	// server, and a Picture evaluation must never hold a lock across that.
	mine, _ := pictureHtfContractOf(e.at, symbol)
	frameC := frameContract(bars)
	e.mu.Lock()
	defer e.mu.Unlock()
	if mine != "" && frameC != "" && frameC != mine {
		e.foreignFrames++
		logger.Warnf("picture-htf: frame names contract %s but this trader is on %s — frame ignored (%d so far)",
			frameC, mine, e.foreignFrames)
		return
	}
	if tf == "5m" {
		e.freshest5mAt = receivedAt
	}
	e.evaluateLocked(symbol, receivedAt)
}

// Evaluate is the tick-based fallback (session opens, reconnect): same work as
// OnBars but driven by the wall clock. Returns the verdict for the AI context
// and dashboard.
func (e *PictureHtfEvaluator) Evaluate(symbol string, now time.Time) EvaluateResult {
	if e == nil || !e.enabled {
		return EvaluateResult{Stage: "watching"}
	}
	if !e.ownsSymbol(symbol) {
		return EvaluateResult{Stage: "watching", Reason: "not this trader's instrument"}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.evaluateLocked(symbol, now)
}

// EvaluateResult is the per-evaluation verdict for the dashboard/AI context.
type EvaluateResult struct {
	Stage    string // watching | confirmed | refused | expired | submitted
	Reason   string
	OppKey   string
	Momentum *kernel.MomentumStall
}

func (e *PictureHtfEvaluator) evaluateLocked(symbol string, now time.Time) EvaluateResult {
	nowMs := now.UnixMilli()
	if !pictureHtfCapabilityProven(e.at) {
		if !e.capWarned {
			e.capWarned = true
			logger.Warnf("picture-htf: mode unavailable — AddOn evidence missing (need build ≥ %s; F5-compile + full NT8 restart with the new AddOn)", ntwire.MinAddonBuildPictureHtf)
		}
		return EvaluateResult{Stage: "watching", Reason: "mode unavailable — AddOn evidence missing"}
	}
	e.capWarned = false
	if dep := e.rebuildLevels(symbol, nowMs); !dep.OK() {
		// D21: no level and no trade until the history is provably deep
		// enough. The reason carries the counts that were READ.
		return EvaluateResult{Stage: "watching", Reason: dep.Reason()}
	}

	// --- H1 completion scan + advisory momentum ---
	h1 := e.bars(symbol, "1h", 4, nowMs)
	for _, b := range h1 {
		if b.CloseTime <= e.lastH1CloseEval {
			continue
		}
		e.lastH1CloseEval = b.CloseTime
		e.h1Closes = append(e.h1Closes, b.Close)
		if len(e.h1Closes) > 3 {
			e.h1Closes = e.h1Closes[len(e.h1Closes)-3:]
		}
	}
	var stall *kernel.MomentumStall
	if len(e.h1Closes) == 3 {
		if s := kernel.H1MomentumStall(e.h1Closes); s.Fired {
			stall = &s
		}
	}

	// --- H1 breakout over the last two completed H1 candles ---
	if len(h1) < 2 || e.lastH1CloseEval == 0 {
		return EvaluateResult{Stage: "watching", Momentum: stall}
	}
	prev, cur := h1[len(h1)-2], h1[len(h1)-1]
	// SIMULTANEOUS H1/4H COMPLETION (spec): the breakout reference is the
	// level set as it stood BEFORE the confirming H1 opened — retirements
	// from a 4h candle closing AT the same boundary as the H1 (which contains
	// the H1's own move) must NOT retro-kill the breakout. So the break check
	// uses a snapshot rebuilt from 4H bars completed before cur.OpenTime.
	// Target selection, below, uses the as-of-now snapshot — retirements ARE
	// applied before target selection, per the same clause. Both snapshots
	// are time-derived from the cache, so frame arrival order cannot change
	// either verdict.
	// The break snapshot reads the same depth rule: its cutoff removes at
	// least one more candle, which is exactly what the margin is for.
	breakDep, breakBars := e.depth4H(symbol, cur.OpenTime)
	if !breakDep.OK() {
		return EvaluateResult{Stage: "watching", Reason: breakDep.Reason(), Momentum: stall}
	}
	breakLevels := kernel.ActiveLevels(
		kernel.BodyPivots4H(breakBars, e.cfg.PivotWindow),
		cur.OpenTime)
	breakVerdict := kernel.H1CloseBreak(breakLevels, prev, cur, e.cfg.TickSize)
	if !breakVerdict.Fired {
		return EvaluateResult{Stage: "watching", Momentum: stall}
	}

	// --- Eligibility: the following 5m interval + freshness ---
	fiveM := e.bars(symbol, "5m", e.cfg.SwingLookback+4, nowMs)
	if len(fiveM) == 0 {
		return EvaluateResult{Stage: "watching", Momentum: stall}
	}
	newest5m := fiveM[len(fiveM)-1]
	intervalStart := newest5m.OpenTime + 5*60_000 // the boundary of the NEXT 5m interval
	elapsed := nowMs - intervalStart
	windowMs := int64(e.cfg.EntryWindowSec) * 1000
	if elapsed < 0 {
		// The next interval has not begun — wait for its boundary frame.
		return EvaluateResult{Stage: "watching", Momentum: stall}
	}
	level := breakLevels[breakVerdict.LevelIdx]
	// The system's strategy identity IS the trader id (plans.strategy_id =
	// trader id) — the opportunity key uses the same binding.
	strategyID := e.at.id
	contract, _ := e.at.currentContract(symbol)
	oppKey := store.PictureHtfOppKey(strategyID, e.at.currentAccountName(), contract, breakVerdict.Direction, level.Role, level.SourceOpen, cur.CloseTime)
	if elapsed > windowMs {
		return e.refuse(oppKey, "expired", fmt.Sprintf("entry window passed (%dms > %dms)", elapsed, windowMs), stall)
	}
	// ARRIVAL-ORDER INDEPENDENCE: the first 5m frame of the interval has not
	// been received yet. The entry cannot be sent without a fresh receipt, so
	// the mode WAITS — it must NOT write a refusal row that a qualifying
	// in-window frame arriving milliseconds later would have to live with.
	// (A 4h/1h frame landing just before the 5m boundary frame must not kill
	// the setup.)
	if e.freshest5mAt.IsZero() {
		return EvaluateResult{Stage: "watching", Reason: "awaiting the first 5m frame of the interval", Momentum: stall}
	}
	if now.Sub(e.freshest5mAt).Milliseconds() > int64(e.cfg.FreshnessSec)*1000 {
		return e.refuse(oppKey, "expired", "data age exceeds the freshness limit — a late frame cannot enter", stall)
	}

	// --- Geometry: structural stop + opposing-zone target ---
	stopPx, ok := kernel.StructuralSwing5M(fiveM, breakVerdict.Direction, e.cfg.SwingLookback, cur.CloseTime)
	if !ok {
		return e.refuse(oppKey, "refused", "no confirmed 5m swing stop before the H1 close — no trade", stall)
	}
	stopPx -= e.cfg.TickSize
	if breakVerdict.Direction == "short" {
		stopPx = stopPx + 2*e.cfg.TickSize
	}
	entryRef := newest5m.Close
	targetPx, ok := kernel.NearestOpposingZone(kernel.ActiveLevels(e.levels, nowMs), entryRef, breakVerdict.Direction, nowMs)
	if !ok {
		return e.refuse(oppKey, "refused", "no eligible opposing 4H zone — no trade", stall)
	}
	risk := absF(entryRef - stopPx)
	reward := absF(targetPx - entryRef)
	rr := 0.0
	if risk > 0 {
		rr = reward / risk
	}
	// W-EXEC-TRUTH W0 (Q7, D12): the floor is max(Picture's own knob, the
	// strategy floor) — a knob below the strategy floor never loosens it — and
	// with no strategy config there is no floor, so the opportunity refuses.
	minRR, floorOK := e.at.pictureMinRR(e.cfg.MinRR)
	if !floorOK {
		return e.refuse(oppKey, "refused", "no R:R floor resolvable (no strategy config) — fail-closed", stall)
	}
	if rr < minRR {
		return e.refuse(oppKey, "refused", fmt.Sprintf("nearest opposing zone offers %.2fR; the configured minimum is %.2fR — the nearer zone is never skipped", rr, minRR), stall)
	}

	// --- Admission: claim the row, then atomically own the submission ---
	row := &store.PictureHtfOpportunityDB{
		OppKey: oppKey, TraderID: e.at.id, StrategyID: strategyID,
		Account: e.at.currentAccountName(), Contract: contract, Symbol: symbol,
		Direction: breakVerdict.Direction, RuleVer: 1, Stage: "confirmed",
		LevelRole: level.Role, LevelBodyTop: level.BodyTop, LevelBodyBot: level.BodyBottom,
		LevelWickHi: level.WickHigh, LevelWickLo: level.WickLow,
		LevelBarOpen: level.SourceOpen, LevelKnowable: level.KnowableAt,
		H1PrevClose: breakVerdict.PrevClose, H1NewClose: breakVerdict.NewClose, H1Boundary: breakVerdict.Boundary,
		H1OpenTime: cur.OpenTime, H1CloseTime: cur.CloseTime, H1Completion: nowMs,
		WindowOpen: intervalStart, WindowClose: intervalStart + windowMs, FreshVerdict: "fresh",
		EntryRef: entryRef, StopPx: stopPx, StopSource: "5m swing", TargetPx: targetPx, TargetZone: level.Role,
		RREstimate: rr, RRConfigured: minRR,
	}
	if stall != nil {
		row.MomentumStall = stall.Fired
		row.MomentumDir = stall.Direction
	}
	// W-ONE-BUTTON M2 site 3 — THE MAINTENANCE HOLD refuses BEFORE the
	// claim. A claim followed by a refused send would leave the row
	// place_pending ("ambiguous"), blocking re-entry AND the installation gate
	// until reconciled. The refusal is durable (fail-closed): an opportunity
	// seen during maintenance never trades, even after the hold clears.
	if reason, held := MaintenanceHeld(); held {
		return e.refuseHeld(oppKey, reason, stall)
	}
	// W-EXEC-TRUTH W0 (a) — THE ONE ADMISSION GATE, before the claim. A
	// refusal here writes NO row: a transient gate (a pause, a feed flap, the
	// dead-man) must not kill the hour's opportunity for good — the next frame
	// asks again. Counted and logged once per change (admitRefuse).
	adm := &pictureAdmission{
		EntryRef: entryRef, LatestClose: e.latestClose(symbol, nowMs), Stop: stopPx, Target: targetPx,
		ATR5m: armSeamATR5mFromBars(fiveM), KnobMinRR: e.cfg.MinRR,
	}
	if refusal, refused := e.at.admitEntry(admitIntent{
		Path: admitPicture, Symbol: symbol, Action: "open_" + breakVerdict.Direction, Now: now,
		Key: oppKey, Price: entryRef, Picture: adm,
	}); refused {
		return EvaluateResult{Stage: "watching", Reason: "admission refused: " + refusal, OppKey: oppKey, Momentum: stall}
	}
	e.pendingAdmission = adm
	_, fresh, err := e.at.store.PictureHtfClaim(row)
	if err != nil {
		return EvaluateResult{Stage: "watching", Reason: "store claim failed: " + err.Error(), OppKey: oppKey, Momentum: stall}
	}
	if !fresh {
		// Duplicate frame/restart — the opportunity already exists (a refused
		// row counts as an existing opportunity; the claim is durable).
		return EvaluateResult{Stage: "watching", Reason: "opportunity already claimed", OppKey: oppKey, Momentum: stall}
	}
	signalID := fmt.Sprintf("picture-htf-%d", nowMs)
	won, err := e.at.store.PictureHtfClaimSubmission(oppKey, signalID)
	if err != nil || !won {
		return EvaluateResult{Stage: "watching", Reason: "submission ownership lost", OppKey: oppKey, Momentum: stall}
	}
	// W-EXEC-TRUTH W0 defect 2: the claim id rides into the send, whose ledger
	// stamp is refused for any other owner. It was never assigned, so every
	// stamp was refused and Picture's wire path was dead.
	row.SignalID = signalID
	// The atomic owner sends. The seam is the production market-entry method
	// (wired with the next wave commit); until then it returns unbound and the
	// row stays place_pending for the reconciliation sweep — never a blind
	// resend.
	if err := pictureHtfSubmitSeam(e, row, stopPx, targetPx, 0, now); err != nil {
		// A maintenance-hold refusal PROVES nothing reached the wire: the
		// permit is taken before the ledger stamp and the send, and a queued
		// entry dropped under the hold is never written. So the row settles
		// refused instead of sitting ambiguous place_pending.
		if ntTrader.IsMaintenanceHold(err) {
			return e.refuseHeld(oppKey, err.Error(), stall)
		}
		// W-EXEC-TRUTH W0: the submission stamp is written in beforeSend, the
		// last step before the wire. No stamp = the send never started, so the
		// row settles refused — provably unsent — instead of an ambiguous
		// place_pending that blocks every later Picture entry and the
		// installation gate.
		// An explicit "a write had started" (the hold's ambiguous drop) is
		// evidence the other way and always stays pending.
		if cur, ok, gerr := e.at.store.PictureHtfGet(oppKey); gerr == nil && ok && cur.SubmittedAt == 0 && !errors.Is(err, ntwire.ErrEntryDropAmbiguous) {
			return e.refuse(oppKey, "refused", "never sent — "+err.Error(), stall)
		}
		// The send failed AFTER the stamp — the row stays place_pending and
		// blocks re-entry until reconciled against NT8 orders (addendum #4).
		return EvaluateResult{Stage: "submitted", Reason: "send ambiguous: " + err.Error(), OppKey: oppKey, Momentum: stall}
	}
	return EvaluateResult{Stage: "submitted", OppKey: oppKey, Momentum: stall}
}

// refuseHeld is the maintenance-hold refusal: a durable "refused" row, counted
// as the maintenance_hold gate block and WARNed once per opportunity.
func (e *PictureHtfEvaluator) refuseHeld(oppKey, reason string, stall *kernel.MomentumStall) EvaluateResult {
	if e.holdRefusedKey != oppKey {
		e.holdRefusedKey = oppKey
		telemetry.IncGateBlock(e.at.id, "maintenance_hold")
		logger.Warnf("🔒 picture-htf: opportunity %s REFUSED — maintenance hold: %s. It will not be traded after the update.", oppKey, reason)
	}
	return e.refuse(oppKey, "refused", "maintenance hold — "+reason, stall)
}

func (e *PictureHtfEvaluator) refuse(oppKey, stage, reason string, stall *kernel.MomentumStall) EvaluateResult {
	if e.at != nil && e.at.store != nil && oppKey != "" {
		_, _, _ = e.at.store.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: oppKey, TraderID: e.at.id, Stage: stage, StageReason: reason})
		// W-EXEC-TRUTH W0 (Q8): only a row no send has started may be refused.
		_, _ = e.at.store.PictureHtfRefuse(oppKey, stage, reason)
	}
	return EvaluateResult{Stage: stage, Reason: reason, OppKey: oppKey, Momentum: stall}
}

// latestClose is the newest 1m close for symbol (forming allowed — it is the
// price a market entry would meet), 0 when unknown (W-EXEC-TRUTH W0 Q19).
func (e *PictureHtfEvaluator) latestClose(symbol string, nowMs int64) float64 {
	if market.FuturesBarsProvider == nil {
		return 0
	}
	raw := market.FuturesBarsProvider(symbol, "1m", 2)
	if len(raw) == 0 {
		return 0
	}
	if b := raw[len(raw)-1]; b.OpenTime <= nowMs {
		return b.Close
	}
	return 0
}
