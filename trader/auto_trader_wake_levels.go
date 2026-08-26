package trader

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W6 wake wave (2026-08-25) — event-diff planner wake-ups ─────────────────
//
// The planner reads once per session + re-plans on death + wakes on a structure
// MSS (G4.6). W6 adds the FIFTH class of wake: level events — fresh S/D zones,
// FVGs, OBs and iFVGs the plan never saw, plus invalidation of levels it DID
// seat. This is deterministic change detection (research: CUSUM/change-point
// family), not statistical detection: the detector output is diffed against the
// plan row's birth time (row.CreatedAt), a wake fires for the newest event, and
// wake_min_interval_min is the false-alarm/detection-delay knob. W6-D: wakes are
// unlimited and spend NO re-plan budget — only deaths consume replan_cap, so a
// wake can never dark a session. Death-first ordering in maybeRunSessionReadsAt
// is preserved (deaths are handled before any wake runs).

// levelWakeCandidate is one wakeable event, already filtered by the knobs.
type levelWakeCandidate struct {
	key     string // planID:version:kind:label:tier:birthBarTime — one wake per event
	desc    string // human line for the log + trigger context
	kind    string // zone | fvg | ob | ifvg | invalidation
	label   string
	tier    string // 15m | 1h | 4h
	birthMs int64
	prio    int // lower fires first
}

// wakePrio orders candidate classes (invalidation strongest, HTF before 15m).
const (
	wakePrioInvalidation = 0
	wakePrioHTFZone      = 1
	wakePrio15mZone      = 2
	wakePrio15mFVG       = 3
	wakePrioIFVG         = 4
	wakePrioHTFOB        = 5
)

// wakeTFs are the detection timeframes of the W6 wake loop.
var wakeTFs = []string{"15m", "1h", "4h"}

// wakeATR computes ATR for a TF slice with the detector fallback (0.002 × last
// close) so a cold cache never zeroes the tolerance.
func wakeATR(bars []market.Kline) float64 {
	atr := market.ExportCalculateATR(bars, 14)
	if atr <= 0 && len(bars) > 0 {
		atr = 0.002 * bars[len(bars)-1].Close
	}
	return atr
}

// lastClosedBar returns the most recent bar with CloseTime < nowMs (closed-only,
// mirroring kernel.closedBars) or false when none.
func lastClosedBar(bars []market.Kline, nowMs int64) (market.Kline, bool) {
	for i := len(bars) - 1; i >= 0; i-- {
		if bars[i].CloseTime < nowMs {
			return bars[i], true
		}
	}
	return market.Kline{}, false
}

// collectLevelWakeCandidates is the PURE W6 collector: detector output vs the
// plan row's birth time. `row` may be nil (no candidates). `cfg` may be nil
// (all ON defaults). It never mutates state — maybeWakePlannerOnLevelEvents
// applies the throttle, dedupe and budget.
func collectLevelWakeCandidates(cfg *store.DayPlanConfig, fetch func(tf string, count int) []market.Kline, symbol string, row *store.PlanDB, now time.Time) []levelWakeCandidate {
	if row == nil || fetch == nil {
		return nil
	}
	nowMs := now.UnixMilli()
	birthMs := row.CreatedAt.UnixMilli()
	tick := market.FuturesTickSize(symbol)
	if tick <= 0 {
		tick = 0.25
	}
	barsByTF := map[string][]market.Kline{}
	atrByTF := map[string]float64{}
	for _, tf := range wakeTFs {
		bars := fetch(tf, 500)
		barsByTF[tf] = bars
		atrByTF[tf] = wakeATR(bars)
	}

	var out []levelWakeCandidate
	add := func(c levelWakeCandidate) { out = append(out, c) }

	// 15m tier (wake_on_15m_zone): reversal S/D zones + FVG formations.
	if cfg.WakeOn15mZoneEnabled() {
		b15 := barsByTF["15m"]
		for _, z := range kernel.SupplyDemandZones(b15, atrByTF["15m"], now) {
			if z.ZonePattern != "reversal" || z.FormedAtMs <= birthMs {
				continue
			}
			add(levelWakeCandidate{
				key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "zone", z.Label, "15m", z.FormedAtMs),
				desc: fmt.Sprintf("15m reversal S/D zone %s [%.2f–%.2f]", z.Label, z.Lo, z.Hi),
				kind: "zone", label: z.Label, tier: "15m", birthMs: z.FormedAtMs, prio: wakePrio15mZone,
			})
		}
		for _, f := range kernel.FairValueGaps(b15, wakeATR(b15), now) {
			if f.Kind != kernel.KindFVG || f.FormedAtMs <= birthMs {
				continue
			}
			add(levelWakeCandidate{
				key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "fvg", f.Label, "15m", f.FormedAtMs),
				desc: fmt.Sprintf("15m FVG [%.2f–%.2f]", f.Lo, f.Hi),
				kind: "fvg", label: f.Label, tier: "15m", birthMs: f.FormedAtMs, prio: wakePrio15mFVG,
			})
		}
	}

	// HTF tier (wake_on_htf_zone): 1h/4h S/D zones (any pattern).
	if cfg.WakeOnHTFZoneEnabled() {
		for _, tf := range []string{"1h", "4h"} {
			for _, z := range kernel.SupplyDemandZones(barsByTF[tf], atrByTF[tf], now) {
				if z.FormedAtMs <= birthMs {
					continue
				}
				add(levelWakeCandidate{
					key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "zone", z.Label, tf, z.FormedAtMs),
					desc: fmt.Sprintf("%s S/D zone %s [%.2f–%.2f]", tf, z.Label, z.Lo, z.Hi),
					kind: "zone", label: z.Label, tier: tf, birthMs: z.FormedAtMs, prio: wakePrioHTFZone,
				})
			}
		}
	}

	// HTF order blocks (wake_on_htf_ob, OFF by default).
	if cfg.WakeOnHTFOBEnabled() {
		for _, tf := range []string{"1h", "4h"} {
			for _, ob := range kernel.OrderBlocks(barsByTF[tf], atrByTF[tf], now) {
				if ob.FormedAtMs <= birthMs {
					continue
				}
				add(levelWakeCandidate{
					key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "ob", ob.Label, tf, ob.FormedAtMs),
					desc: fmt.Sprintf("%s order block %s [%.2f–%.2f]", tf, ob.Label, ob.Lo, ob.Hi),
					kind: "ob", label: ob.Label, tier: tf, birthMs: ob.FormedAtMs, prio: wakePrioHTFOB,
				})
			}
		}
	}

	// iFVG (wake_on_ifvg): filled→inverted gaps on any wake tier.
	if cfg.WakeOnIFVGEnabled() {
		for _, tf := range wakeTFs {
			for _, f := range kernel.FairValueGaps(barsByTF[tf], wakeATR(barsByTF[tf]), now) {
				if f.Kind != kernel.KindIFVG || f.FormedAtMs <= birthMs {
					continue
				}
				add(levelWakeCandidate{
					key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "ifvg", f.Label, tf, f.FormedAtMs),
					desc: fmt.Sprintf("%s %s [%.2f–%.2f] (%s)", tf, f.Label, f.Lo, f.Hi, f.Info),
					kind: "ifvg", label: f.Label, tier: tf, birthMs: f.FormedAtMs, prio: wakePrioIFVG,
				})
			}
		}
	}

	// Seated-level invalidation (wake_on_seated_invalidation): a zone-kind level
	// the plan SEATED is closed beyond by more than a noise band (max(2 ticks,
	// 0.15×ATR15)) — support (Demand/OB(bull)/iFVG(bull)) below, resistance
	// (Supply/OB(bear)/iFVG(bear)) above. Fresh only (violating close within
	// 2× the wake interval) so a stale invalidation never fires on restart.
	if cfg.WakeOnSeatedInvalidationEnabled() {
		var doc kernel.PlanDoc
		if json.Unmarshal([]byte(row.Doc), &doc) == nil {
			close, ok := lastClosedBar(barsByTF["15m"], nowMs)
			if ok {
				noise := 2 * tick
				if alt := 0.15 * atrByTF["15m"]; alt > noise {
					noise = alt
				}
				freshMs := int64(cfg.WakeMinIntervalMinutes()) * 2 * int64(time.Minute/time.Millisecond)
				for _, l := range doc.Levels {
					dir := seatedLevelSide(l.Label)
					if dir == "" || l.Price <= 0 {
						continue
					}
					violated := (dir == "below" && close.Close < l.Price-noise) ||
						(dir == "above" && close.Close > l.Price+noise)
					if !violated || close.CloseTime < nowMs-freshMs {
						continue
					}
					add(levelWakeCandidate{
						key:  fmt.Sprintf("%s:%d:%s:%s:%s:%d", row.PlanID, row.Version, "invalidation", l.Label, "15m", close.CloseTime),
						desc: fmt.Sprintf("seated %s invalidated: close %.2f %s %.2f (noise %.2f)", l.Label, close.Close, dir, l.Price, noise),
						kind: "invalidation", label: l.Label, tier: "15m", birthMs: close.CloseTime, prio: wakePrioInvalidation,
					})
				}
			}
		}
	}

	// Priority first, newest within a class first.
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].prio != out[b].prio {
			return out[a].prio < out[b].prio
		}
		return out[a].birthMs > out[b].birthMs
	})
	return out
}

// seatedLevelSide classifies a plan level by its provenance label into the
// invalidation direction ("" = not a zone-kind level, never wakes).
func seatedLevelSide(label string) string {
	l := strings.TrimSpace(label)
	switch {
	case strings.HasPrefix(l, "Demand"), strings.HasPrefix(l, "OB(bull"), strings.HasPrefix(l, "iFVG(bull"):
		return "below" // support: invalidated by a close below
	case strings.HasPrefix(l, "Supply"), strings.HasPrefix(l, "OB(bear"), strings.HasPrefix(l, "iFVG(bear"):
		return "above" // resistance: invalidated by a close above
	}
	return ""
}

// maybeWakePlannerOnLevelEvents is the W6 wake entry: fires AT MOST one planner
// wake per cycle on the newest level event, throttled by wake_min_interval_min
// (shared with the MSS wake via lastPlannerWakeAt) and deduped per
// (plan,version,kind,label,tier,birth). W6-D: wakes are UNLIMITED and spend NO
// budget — the per-session re-plan cap belongs to deaths alone. Death-first
// ordering is preserved by the caller.
func (at *AutoTrader) maybeWakePlannerOnLevelEvents(session, tradeDate string, row *store.PlanDB) {
	if market.FuturesBarsProvider == nil || at.store == nil || row == nil {
		return
	}
	var cfg *store.DayPlanConfig
	if at.config.StrategyConfig != nil {
		cfg = at.config.StrategyConfig.DayPlan
	}
	now := time.Now()
	symbol := at.futuresSymbol()
	fetch := func(tf string, count int) []market.Kline {
		return market.FuturesBarsProvider(symbol, tf, count)
	}
	cands := collectLevelWakeCandidates(cfg, fetch, symbol, row, now)
	if len(cands) == 0 {
		return
	}
	ev := cands[0]
	// One wake per (plan,version,event) — the same event must not re-fire on
	// the next cycle after the wake.
	if at.lastLevelWakeKey == ev.key {
		return
	}
	// Min-interval throttle shared with the MSS wake: ANY planner wake resets
	// the clock, so a death/MSS/level cascade can't triple-spend one window.
	if !at.lastPlannerWakeAt.IsZero() && now.Sub(at.lastPlannerWakeAt) < time.Duration(cfg.WakeMinIntervalMinutes())*time.Minute {
		at.logWarnf("🗓️ level wake %s on %s %s — SKIPPED: %s elapsed < wake_min_interval_min (%d).",
			ev.desc, session, tradeDate, now.Sub(at.lastPlannerWakeAt).Round(time.Second), cfg.WakeMinIntervalMinutes())
		return
	}
	// W6-D (2026-08-25) — wakes are UNLIMITED and count against NOTHING: a wake
	// re-plan must never consume the death budget, and can never be the cause of
	// a replans_exhausted NO-TRADE (live bug 19:35 CT: 4 wake re-plans ate the
	// whole budget and a later death darked the session). Only the dedupe key +
	// the min-interval throttle limit wake frequency; deaths keep their own cap.
	at.lastLevelWakeKey = ev.key
	at.lastPlannerWakeAt = now
	at.logWarnf("🗓️ level wake %s on %s %s — waking the planner (W6, 5th wake-up).", ev.desc, session, tradeDate)
	at.warnIfReplanOrphansOverlays(row)
	// W6-C (2026-08-25) — the wake re-read is NON-fatal (a failed read keeps the
	// active plan; failClosed=false) and ASYNC so a slow/timing-out planner can
	// never stall the decision loop (live bug: a 2×300s retry chain blocked 14
	// minutes of cycles and then no-traded a healthy session).
	go func() {
		_ = at.runPlannerReadWithTriggerClaimedCtx(session, tradeDate, "level_event", "level event: "+ev.desc, priorPlanLevelLines(row), false)
		if fresh, fErr := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, session, at.id); fErr == nil && fresh != nil && fresh.Version != row.Version {
			at.carryOwnerEditsInto(fresh.PlanID, row.Version, fresh.Version)
		}
	}()
}
