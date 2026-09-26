package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W-EXEC-TRUTH W5 — THE ADAPTER SEAM FOR PICTURE EVIDENCE.
//
// This is the ONLY file in W5 that reads the Picture evaluator's opportunity
// row and admission record. W4 (lane Claude-103, CTO order 1790189321937)
// changes those fields — the depth count, the knowable instant, the window
// anchor, the source / receipt / evaluation clocks, the contract proof, the run
// generation — and when it lands only pictureEvidenceFrom changes. The plan
// source, the executor and the receipts read PictureEvidence, never the row.
//
// PictureEvidence is also the machine scenario's frozen evidence record
// (PlanMachineSource.Evidence): what the plan card shows and what no AI
// commentary can rewrite.

// PictureEvidence is one Picture opportunity as the Day Plan records it.
type PictureEvidence struct {
	OppKey     string `json:"opp_key"`
	ClaimID    string `json:"claim_id"`
	TraderID   string `json:"trader_id"`
	StrategyID string `json:"strategy_id"`
	Contract   string `json:"contract"`
	Symbol     string `json:"symbol"`
	Direction  string `json:"direction"` // long | short
	Rule       string `json:"rule"`
	RuleVer    int    `json:"rule_ver"`

	// The broken 4H body (the level) and when it became knowable.
	LevelRole       string  `json:"level_role"`
	BodyTop         float64 `json:"body_top"`
	BodyBot         float64 `json:"body_bot"`
	WickHi          float64 `json:"wick_hi"`
	WickLo          float64 `json:"wick_lo"`
	LevelBarOpenMs  int64   `json:"level_bar_open_ms"`
	LevelKnowableMs int64   `json:"level_knowable_ms"`

	// The confirming H1 close.
	H1PrevClose float64 `json:"h1_prev_close"`
	H1NewClose  float64 `json:"h1_new_close"`
	H1Boundary  float64 `json:"h1_boundary"`
	H1OpenMs    int64   `json:"h1_open_ms"`
	H1CloseMs   int64   `json:"h1_close_ms"`

	// Entry geometry the evaluator admitted on.
	EntryRef    float64 `json:"entry_ref"`
	LatestClose float64 `json:"latest_close,omitempty"` // 0 = unknown (never a price)
	Stop        float64 `json:"stop"`
	Target      float64 `json:"target"`
	ATR5m       float64 `json:"atr5m"`
	StopSource  string  `json:"stop_source"`
	TargetZone  string  `json:"target_zone"`
	RREstimate  float64 `json:"rr_estimate"`
	RRFloor     float64 `json:"rr_floor"` // pictureMinRR = max(knob, strategy floor)
	KnobMinRR   float64 `json:"knob_min_rr,omitempty"`

	// The eligibility window and the evaluation clock.
	WindowOpenMs  int64 `json:"window_open_ms"`
	WindowCloseMs int64 `json:"window_close_ms"`
	EvalAtMs      int64 `json:"eval_at_ms"`

	// W4's evidence (lane 103, fix/w4-picture-evidence @8d0fa6a9), READ at the
	// seam under the evaluator's lock. A clock the evaluator never saw stays
	// nil (absent ≠ 0): the three clocks of the freshest COMPLETED 5m frame —
	// the AddOn's own stamp (SOURCE, another machine's clock), Go's receipt,
	// and the candle's close — and the 4H depth the levels were drawn from.
	SourceEmittedAtMs *int64 `json:"source_emitted_at_ms,omitempty"`
	ReceivedAtMs      *int64 `json:"received_at_ms,omitempty"`
	FrameCloseMs      *int64 `json:"frame_close_ms,omitempty"`
	DepthFetched      *int   `json:"depth_fetched,omitempty"`
	DepthCompleted    *int   `json:"depth_completed,omitempty"`
	DepthNeeded       *int   `json:"depth_needed,omitempty"`
	// Generation is W4's trader generation, not exposed at the seam at
	// 8d0fa6a9 — absent until it is (the run epoch guards reloads meanwhile).
	Generation *uint64 `json:"generation,omitempty"`
}

// positiveMs is an optional instant: nil for 0 (never seen), never a fake 0.
func positiveMs(ms int64) *int64 {
	if ms <= 0 {
		return nil
	}
	return &ms
}

// pictureEvidenceFrom reads the evaluator's hand-off (called at the seam,
// under the evaluator's lock, with the evaluation's own clock). Missing
// required evidence fails closed — the evaluator's "never sent → refused"
// contract then records the refusal — mirroring pictureEntryGate's checks.
func pictureEvidenceFrom(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx float64, now time.Time) (PictureEvidence, error) {
	if e == nil || e.at == nil || row == nil {
		return PictureEvidence{}, fmt.Errorf("picture evidence: no evaluator or opportunity row")
	}
	adm := e.pendingAdmission
	if adm == nil {
		return PictureEvidence{}, fmt.Errorf("picture evidence: no admission record for %s", store.RedactPictureOppKey(row.OppKey))
	}
	dir := strings.ToLower(strings.TrimSpace(row.Direction))
	switch {
	case strings.TrimSpace(row.OppKey) == "":
		return PictureEvidence{}, fmt.Errorf("picture evidence: opportunity has no key")
	case dir != "long" && dir != "short":
		return PictureEvidence{}, fmt.Errorf("picture evidence: direction %q is not long|short", row.Direction)
	case adm.EntryRef <= 0 || stopPx <= 0 || targetPx <= 0:
		return PictureEvidence{}, fmt.Errorf("picture evidence: geometry unknown (entry %.2f stop %.2f target %.2f)", adm.EntryRef, stopPx, targetPx)
	case adm.ATR5m <= 0:
		return PictureEvidence{}, fmt.Errorf("picture evidence: ATR5m unknown")
	case row.WindowClose <= 0:
		return PictureEvidence{}, fmt.Errorf("picture evidence: no eligibility window")
	}
	floor, ok := e.at.pictureMinRR(adm.KnobMinRR)
	if !ok {
		return PictureEvidence{}, fmt.Errorf("picture evidence: no R:R floor resolves")
	}
	// W4 contract: a level becomes usable at its COMPLETION instant, and 0
	// means "unknown" — refuse, never treat it as already knowable.
	if row.LevelKnowable <= 0 {
		return PictureEvidence{}, fmt.Errorf("picture evidence: the level's knowable instant is unknown")
	}
	// W4 contract: the depth the levels were drawn from, READ with the same
	// rule the evaluator judged (depth4H at the evaluation clock). The
	// evaluator refuses below it before the seam; re-refuse here rather than
	// record a scenario on history this evaluation cannot justify.
	dep, _ := e.depth4H(row.Symbol, now.UnixMilli())
	if !dep.OK() {
		return PictureEvidence{}, fmt.Errorf("picture evidence: %s", dep.Reason())
	}
	fetched, completed, needed := dep.Fetched, dep.Completed, dep.Required
	var received *int64
	if !e.freshest5mAt.IsZero() {
		received = positiveMs(e.freshest5mAt.UnixMilli())
	}
	return PictureEvidence{
		OppKey: row.OppKey, ClaimID: row.SignalID, TraderID: row.TraderID, StrategyID: row.StrategyID,
		Contract: row.Contract, Symbol: row.Symbol, Direction: dir,
		Rule: kernel.MachineRulePictureH1CloseBreak, RuleVer: row.RuleVer,
		LevelRole: row.LevelRole, BodyTop: row.LevelBodyTop, BodyBot: row.LevelBodyBot,
		WickHi: row.LevelWickHi, WickLo: row.LevelWickLo,
		LevelBarOpenMs: row.LevelBarOpen, LevelKnowableMs: row.LevelKnowable,
		H1PrevClose: row.H1PrevClose, H1NewClose: row.H1NewClose, H1Boundary: row.H1Boundary,
		H1OpenMs: row.H1OpenTime, H1CloseMs: row.H1CloseTime,
		EntryRef: adm.EntryRef, LatestClose: adm.LatestClose, Stop: stopPx, Target: targetPx, ATR5m: adm.ATR5m,
		StopSource: row.StopSource, TargetZone: row.TargetZone,
		RREstimate: row.RREstimate, RRFloor: floor, KnobMinRR: adm.KnobMinRR,
		WindowOpenMs: row.WindowOpen, WindowCloseMs: row.WindowClose, EvalAtMs: now.UnixMilli(),
		SourceEmittedAtMs: positiveMs(e.freshest5mEmitted), ReceivedAtMs: received, FrameCloseMs: positiveMs(e.freshest5mClose),
		DepthFetched: &fetched, DepthCompleted: &completed, DepthNeeded: &needed,
	}, nil
}
