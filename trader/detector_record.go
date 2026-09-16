package trader

import (
	"encoding/json"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── 1B — THE PRODUCTION CALL PATH ────────────────────────────────────────────
//
// Called once per planner read, from the same place the void scope is resolved,
// so the detector judges the SAME tape the prompt and the validator do. It
// changes no decision: it runs D1′ over the levels the read already produced
// and writes what it finds.
//
// A10: telemetry may WARN, never stop the loop. Every failure here returns
// after logging; nothing propagates to the caller.

// recordDetectorOutputs runs D1′ over the seated levels and persists any NEW
// episodes, then records the candidate pool for this read. Safe to call every
// read: the episode watermark comes from the STORE, so repeats write nothing.
func (at *AutoTrader) recordDetectorOutputs(
	symbol, planID, session string, planVersion int,
	allLevels []kernel.DetectedLevel, seated []kernel.ScoredLevel,
	price, dATR, proximityK float64, maxLevels int, now time.Time,
	// anchors are the plan's scenarios reduced to the one price a level can be
	// compared against. NIL is a legitimate input — it means no scenario was
	// authored, and the row records that as its link basis rather than as a
	// blank (W1 item 1).
	anchors []store.ScenarioAnchor, researchIDs ...string,
) {
	if at == nil || at.store == nil {
		return
	}
	defer func() {
		// A10 + class 23: a telemetry panic must never reach the trading loop.
		if r := recover(); r != nil {
			at.logWarnf("🔬 detector recording panicked and was contained: %v", r)
		}
	}()

	scope := kernel.ResolveVoidScope(symbol, now)
	if len(scope.Bars) == 0 {
		at.logInfof("🔬 detector: no bars in scope — nothing recorded this read (not zero episodes; UNMEASURED)")
		return
	}
	delta := kernel.MeanAbsIncrement(scope.Bars)
	if delta <= 0 {
		at.logWarnf("🔬 detector: Δ resolved to 0 — the band would be degenerate; skipping rather than recording a meaningless verdict")
		return
	}
	k, horizon, exitOn := kernel.DetectorK(), kernel.DetectorHorizonBars(), kernel.DetectorExitOn()
	ts := at.store.TouchOutcomes()

	var identityDoc *kernel.PlanDoc
	if p, err := at.store.Plan().GetPlan(planID, planVersion); err == nil && p != nil && p.StrategyID == at.id {
		_ = json.Unmarshal([]byte(p.Doc), &identityDoc)
	}
	written, preFormation, noFormation := 0, 0, 0
	for _, lv := range seated {
		if lv.Price <= 0 {
			continue
		}
		// D1a — THE SCAN STARTS WHERE THE LEVEL WAS BORN. Scanning the whole
		// void scope for a level that did not exist across most of it is
		// lookahead by construction: it is how all 14 live RTH-L episodes came
		// to open before the 08:36 bar that first printed their price.
		bars := scope.Bars
		if lv.FormedAtMs > 0 {
			bars = barsSince(bars, lv.FormedAtMs)
			preFormation += len(scope.Bars) - len(bars)
		} else {
			// A24/A9: an unknown formation time is NOT a formation time of 0.
			// We still de-duplicate, but this level's episodes cannot be
			// proven post-formation, and the row says so — see below.
			noFormation++
		}
		if len(bars) < 2 {
			continue
		}
		eps := kernel.DetectTouchOutcomes(bars, lv.Price, k, delta, horizon, exitOn)
		if len(researchIDs) > 0 {
			recordResearchEpisodes(researchIDs[0], symbol, lv.DetectedLevel, eps, k, delta, horizon, exitOn, now)
		}
		if len(eps) == 0 {
			continue
		}
		// D1a — the watermark covers THE WHOLE SCANNED WINDOW, not the current
		// session day. Passing dayMs here is what re-wrote every episode that
		// opened before 17:00 CT today, on every read.
		last := ts.LastOpenedAtMs(at.id, symbol, lv.Price, lv.FormedAtMs)
		for _, e := range kernel.NewEpisodesSince(eps, last) {
			row := &store.TouchOutcomeRow{
				TraderID: at.id, Symbol: symbol, CreatedAt: now,
				LevelPrice: lv.Price, LevelKind: string(lv.Kind),
				CandidateSeated: true, PlanID: planID, PlanVersion: planVersion, Session: session,
				// D1c — the ordinal counts within the EPISODE's own session-day.
				// Handing it the day of the READ is what made 471 of 677 rows
				// ordinal 1. One implementation, reused (A24).
				Ordinal: ts.NextOrdinal(at.id, symbol, lv.Price,
					kernel.CMESessionDayStart(time.UnixMilli(e.OpenedAtMs)).UnixMilli()),
				K: k, Delta: delta, BandPts: k * delta, Horizon: horizon, ExitOn: exitOn,
				EntrySide: e.Entry, ExitSide: e.Exit, Outcome: e.Outcome,
				Ambiguous: e.IsAmbiguous(), BarsToExit: e.BarsToExit,
				MFEPts: e.MFE, MAEPts: e.MAE,
				OpenedAtMs: e.OpenedAtMs, ClosedAtMs: e.ClosedAtMs,
				FormedAtMs: lv.FormedAtMs, Validity: validityFor(lv.FormedAtMs),
			}
			// W1 item 1 — the touch → scenario link, recorded as the HEURISTIC
			// it is. The band is the MAP'S OWN cluster width, not a second
			// tolerance; ambiguity resolves NULL with its reason rather than
			// picking a nearest and calling it a fact.
			link := store.ResolveScenarioLink(lv.Price, anchors, delta, scenarioLinkBand())
			row.LevelID = kernel.EpisodeLevelID(lv.DetectedLevel, identityDoc)
			row.ScenarioNearest = link.Scenario
			row.ScenarioLinkBasis = link.Basis
			row.ScenarioLinkDistPts = link.DistPts
			row.ScenarioLinkDistDelta = link.DistDelta
			if err := ts.SaveOutcome(row); err != nil {
				at.logWarnf("🔬 detector: touch_outcomes write failed (level %.2f): %v", lv.Price, err)
				continue
			}
			written++
			// W2 — THE FADE-PERMISSION STAMP, at the episode's OPEN and never
			// again. The clock handed to the facts builder is the episode's
			// own open, not this cycle's now: an evaluation made with a later
			// clock would label the touch with facts it could not have known.
			// A LABEL ONLY (A31/D3) — nothing downstream reads it to refuse.
			at.stampFadePermissionAtOpen(row.ID, symbol, time.UnixMilli(e.OpenedAtMs), lv.Price, seated, e.Entry)
		}
	}

	// D3 — the candidate pool, including everything that did NOT seat.
	pool := kernel.BuildCandidatePool(allLevels, seated, price, dATR, proximityK, maxLevels)
	rows := make([]store.CandidatePoolRow, 0, len(pool))
	for _, c := range pool {
		rows = append(rows, store.CandidatePoolRow{
			TraderID: at.id, Symbol: symbol, PlanID: planID, PlanVersion: planVersion,
			Session: session, ReadAtMs: now.UnixMilli(),
			LevelPrice: c.Price, LevelKind: c.Kind, Label: c.Label,
			Rank: c.Rank, Seated: c.Seated, CutReason: c.CutReason,
			Score: c.Score, Threshold: c.Threshold, Grade: c.Grade,
			ScoreComponents: c.Components,
		})
	}
	cut := 0
	for _, r := range rows {
		if !r.Seated {
			cut++
		}
	}
	if err := at.store.CandidatePool().SavePool(rows); err != nil {
		at.logWarnf("🔬 detector: candidate_pool write failed (%d rows): %v", len(rows), err)
	}
	// THE FOLLOW-PLAN RECORDER (dispatch 102, round 17) — beside every fade
	// episode, the follow side is computed and RECORDED on the same read, over
	// the ring the detector just judged (single-contract after the roll wave).
	// Never arms, never places, never gates (E9). Its own line names the pass.
	if market.FuturesBarsProvider != nil {
		if ring := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount); len(ring) > 0 {
			scanned, breaks, retests := at.recordFollowPlans(symbol, ring, now)
			at.logInfof("📐 follow-plans (RECORDED ONLY): scanned=%d new breaks=%d new retests=%d · tape=%d bars from %s", scanned, breaks, retests, len(ring), kernel.FormatCT(time.UnixMilli(ring[0].OpenTime)))
		}
	}
	// A9 — every skipped bar and every level with no formation time is named.
	at.logInfof("🔬 detector recorded: %d new episode(s) · pool %d candidate(s) (%d seated, %d cut) · k=%.0f Δ=%.2f band=±%.2f H=%d %s · pre-formation bars skipped=%d · levels with no formation time=%d",
		written, len(rows), len(rows)-cut, cut, k, delta, k*delta, horizon, exitOn, preFormation, noFormation)
}

// validityFor says what this row may be used for. A level with no formation
// time cannot be certified post-formation, so its episodes are recorded and
// EXCLUDED from rates rather than silently blessed (A24, A30).
func validityFor(formedAtMs int64) string {
	if formedAtMs > 0 {
		return store.ValidityValid
	}
	return store.ValidityNoFormation
}

// barsSince returns the bars at or after fromMs. The detector needs a previous
// bar to see a touch, so this deliberately keeps the boundary bar: an episode
// still cannot OPEN before fromMs, because a touch is judged on the current
// bar and the episode's opened_at_ms is that bar's time.
func barsSince(bars []market.Kline, fromMs int64) []market.Kline {
	for i := range bars {
		if bars[i].OpenTime >= fromMs {
			return bars[i:]
		}
	}
	return nil
}
