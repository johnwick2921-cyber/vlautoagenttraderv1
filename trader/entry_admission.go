package trader

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"nofx/discipline"
	"nofx/kernel"
	"nofx/market"
	"nofx/store"
	"nofx/telemetry"
)

// ── W-EXEC-TRUTH W0 (a) — ONE ADMISSION GATE ────────────────────────────────
//
// Every producer that can open a position asks THIS function before it sends:
//
//	decision  the AI decision (executeDecisionWithRecord) — and the agent-chat
//	          door, which must be refused exactly like a decision (CTO Q17)
//	arm       the armed/planner placement (runArmedPlacementAt, per row)
//	picture   Picture HTF (before its claim and again before its send)
//
// The chain is the AI path's PINNED order (gate-order contract 2.4/E5: a
// paused refusal always NAMES the pause; CTO Q1). Each step is the EXISTING
// function, never a copy, and keeps its owner-ruled fail-open contract (Q2);
// Picture's own evidence is checked fail-closed in its pre-check (item 7).
//
// Scope per path (documented, CTO Q3/Q4/Q5):
//   - session/breaker: the decision path keeps its two separate steps
//     (consecutiveLossHalted, sessionEntryBlocked) with their own strings;
//     arm and picture ask sessionRiskGateAt, which already holds the breaker,
//     the band, T1 and the per-session trade cap — never a second copy.
//   - CME closed: arm and picture only (the decision path's runCycle skips
//     the whole cycle when the exchange is closed).
//   - plan_mode: decision (and agent) only; the arm path's EntryGate leg 0 and
//     Picture's picture intent cover it.
//   - re-entry cooldown: arm and picture through discipline.ReentryPeek
//     (non-destructive; the decision path's kernel read is the one that
//     clears). Transition stand-down stays the AI executor's, by its own
//     definition.
//
// Refusals on the decision and agent paths log and count every time, byte-
// identical to the inline chain they replace. Arm and picture refusals log and
// count ONCE per change of (path, key, refusal) — a 2-minute pass or a live-bar
// frame is not a new event (canon 35).

type admitPath string

const (
	admitDecision admitPath = "decision"
	admitAgent    admitPath = "agent"
	admitArm      admitPath = "arm"
	admitPicture  admitPath = "picture"
)

// admitIntent is one request to open a position.
type admitIntent struct {
	Path   admitPath
	Symbol string
	Action string // open_long | open_short
	Now    time.Time
	// Decision is the AI decision (decision and agent paths): the plan-mode
	// and EntryGate input. Record is where the decision path stamps an
	// EntryGate refusal (nil elsewhere).
	Decision *kernel.Decision
	Record   *store.DecisionAction
	// Key identifies an arm row or a Picture opportunity for the once-per-
	// change log and counter.
	Key string
	// Price is the execution-time price the arm/picture path sees (the
	// cooldown's price input; ≤0 = unknown → the cooldown's timer is the only
	// unlock, the stricter reading).
	Price float64
	// Picture carries the picture intent for EntryGate (picture path only).
	Picture *pictureAdmission
}

func (in admitIntent) side() string {
	if in.Action == "open_short" {
		return "short"
	}
	return "long"
}

// admitDedupe remembers the last refusal per (path, key) for arm and picture.
type admitDedupe struct {
	mu   sync.Mutex
	last map[string]string
}

func (d *admitDedupe) changed(k, v string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.last == nil {
		d.last = map[string]string{}
	}
	if d.last[k] == v {
		return false
	}
	d.last[k] = v
	return true
}

func (d *admitDedupe) clear(k string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.last, k)
}

// admitRefuse records one refusal: class is the gate-block counter, record is
// the refusal text the caller stamps, logf emits the path's log line (and any
// alert). Decision/agent: every time; arm/picture: once per change.
func (at *AutoTrader) admitRefuse(in admitIntent, class, record string, logf func()) (string, bool) {
	if in.Path == admitArm || in.Path == admitPicture {
		// CTO M3 — deduped on (path, key, CLASS), never on the reason: the
		// reasons carry moving values (the cooldown's live price and distance,
		// R:R at the execution price, the loss count), and a reason-keyed
		// dedupe re-logged and re-counted ONE refusal on every tick the price
		// moved. The log line still carries the reason.
		if !at.admitLast.changed(string(in.Path)+"|"+in.Key, admitDedupeClass(class, record)) {
			return record, true
		}
	}
	if logf != nil {
		logf()
	}
	telemetry.IncGateBlock(at.id, class)
	return record, true
}

// admitDedupeClass is the value a repeated arm/picture refusal is deduped on
// (CTO M3): the gate-block class — refined, for entry_gate only, by the leg
// armRefusalClass reads (the same "entry_gate:<leg>" string the arm-refusal
// counter family is keyed on), so a new LEG re-logs while a moving number
// inside one leg does not.
func admitDedupeClass(class, reason string) string {
	if class != "entry_gate" {
		return class
	}
	leg := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(reason), "entry_gate:"))
	return "entry_gate:" + armRefusalClass(leg)
}

// admitEntry runs the one admission chain. It returns ("", false) when the
// entry may proceed, or the refusal text and true.
func (at *AutoTrader) admitEntry(in admitIntent) (string, bool) {
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	now := in.Now
	sym, act := in.Symbol, in.Action
	if in.Path == admitArm || in.Path == admitPicture {
		act = in.Action + " [" + string(in.Path) + " " + in.Key + "]"
	}
	refusal, refused := at.admitChain(in, sym, act, now)
	if !refused && (in.Path == admitArm || in.Path == admitPicture) {
		at.admitLast.clear(string(in.Path) + "|" + in.Key)
	}
	return refusal, refused
}

func (at *AutoTrader) admitChain(in admitIntent, sym, act string, now time.Time) (string, bool) {
	if in.Path == admitPicture {
		// Picture runs on the live-bar goroutine, outside runCycle, so it has
		// no loop in front of it that stops when the trader stops or the Day
		// Plan master is off (D26). The decision and arm paths only run inside
		// runCycle, which already requires both.
		if !at.runningNow() {
			return at.admitRefuse(in, "trader_stopped", "picture: the trader is not running", func() {
				at.logWarnf("⏹ picture: %s %s REFUSED — the trader is not running.", sym, act)
			})
		}
		if !at.dayPlanEnabled() {
			return at.admitRefuse(in, "day_plan_off", "picture: the Day Plan master is off", func() {
				at.logWarnf("⏹ picture: %s %s REFUSED — the Day Plan master is off.", sym, act)
			})
		}
	}
	// Feed-down gate (NinjaTrader, TRACK A): the SIM cannot fill without market
	// data, so an entry issued while the feed is down is rejected ("no market
	// data"). Default-ALLOW until a feed_status frame arrives, so a healthy bot
	// is never false-halted. (The CLOSE half of this gate stays in
	// executeDecisionWithRecord — a close is not an admission.)
	if down, status := at.ninjaFeedDown(); down {
		return at.admitRefuse(in, "feed_down", fmt.Sprintf("feed_down: NT8 price feed not Connected (status=%q)", status), func() {
			at.logWarnf("⛔ feed-gate: %s %s skipped — NT8 price feed not Connected (status=%q); SIM would reject 'no market data'. Will act when the feed returns.", act, sym, status)
		})
	}

	// B5 — dead-man watchdog: after an NT8 TCP disconnect, NEW entries stay
	// blocked until a clean positions/orders reconciliation, so we never open on
	// top of a state we haven't re-verified across a link gap. State is advanced
	// once per cycle in runCycle (driveDeadManWatchdog); here we only enforce.
	if at.deadMan.entriesBlocked() {
		return at.admitRefuse(in, "dead_man", "dead_man_watchdog: awaiting reconciliation after link gap", func() {
			at.logWarnf("⛔ dead-man watchdog: %s %s REFUSED — NT8 link not yet reconciled after a disconnect; entries resume after a clean reconciliation.", sym, act)
		})
	}

	// A4 (G4) — FREEZE gate: a trader frozen by an identity/account echo mismatch
	// (A2) or a reconcile belief≠broker divergence is blocked from NEW entries
	// until the owner clears it (POST /api/risk/clear-freeze).
	if reason, frozen := discipline.IsFrozen(at.id); frozen {
		return at.admitRefuse(in, "frozen", "frozen: "+reason, func() {
			at.logErrorf("🚨 A4 FROZEN: %s %s REFUSED — trader is frozen (%s). Investigate, then clear via /api/risk/clear-freeze to resume.", sym, act, reason)
		})
	}

	// P1 — BOOT INTEGRITY: a binary that is not the intended release, or whose
	// prompt goldens drifted, must not open positions. This outranks every other
	// gate (it means we cannot trust WHAT this process is).
	if reason, refused := kernel.TradingRefused(); refused {
		return at.admitRefuse(in, "boot_integrity", "boot_integrity_refused: "+reason, func() {
			at.logErrorf("🔐 BOOT INTEGRITY REFUSAL: %s %s BLOCKED — %s. Fix the deploy and restart; closes still work.",
				sym, act, reason)
			at.emitAlert("P0", "boot-integrity", "boot-integrity:"+kernel.CMESessionDayKey(now),
				"🔐 Trading refused — boot integrity", reason)
		})
	}

	// P2 (ledger-close 2026-08-19) — stop_until OWNER PAUSE: the FIRST owner/
	// policy gate (system-integrity gates above rank it; every policy gate below
	// defers to it, so a paused refusal always NAMES the pause — gate-order
	// contract 2.4/E5). Blocks NEW entries only; closes, EOD-flat, the 60s
	// monitor guards, and NT8 brackets continue. Master-INDEPENDENT.
	if reason, paused := at.entryPausedAt(now); paused {
		return at.admitRefuse(in, "stop_until", "stop_until: "+reason, func() {
			at.logWarnf("⏸ stop_until: %s %s REFUSED — %s. Position management continues; entries resume on expiry or POST /api/traders/:id/resume.", sym, act, reason)
		})
	}

	// W-ONE-BUTTON M2 site 1 — INSTALLATION MAINTENANCE HOLD: while the updater
	// holds (<data>/updater/hold.json; an unreadable file holds too), NEW
	// entries are refused here, before any wire side effect (the orphan-flatten
	// in reconcileBeforeOpenNT included). Position management continues. The
	// broker-layer permit (site 4) is the second, send-side check.
	if reason, held := MaintenanceHeld(); held {
		return at.admitRefuse(in, "maintenance_hold", "maintenance_hold: "+reason, func() {
			at.logWarnf("🔒 maintenance hold: %s %s REFUSED — %s. Position management continues; entries resume when the update completes.", sym, act, reason)
		})
	}

	// P3 (ledger-close 2026-08-19) — CONTRACT-ROLL gate for the continuous
	// symbol: within ROLL_BLOCK_DAYS_BEFORE_EXPIRY of the ACK-resolved front
	// contract's third-Friday expiry, NEW entries are refused (the dated-code
	// T19 gate never fires on bare "MNQ"). Runs AFTER stop_until (a paused
	// refusal must name the pause — E5) and fail-opens when unresolved.
	if reason, blocked := at.entryBlockedByRoll(now); blocked {
		return at.admitRefuse(in, "contract_roll_resolved", "contract_roll: "+reason, func() {
			at.logWarnf("📅 contract-roll: %s %s REFUSED — %s. Position management continues; the resolver rolls to the next quarterly.", sym, act, reason)
		})
	}

	if in.Path == admitDecision || in.Path == admitAgent {
		// D1 — consecutive-loss halt: after N consecutive losing closed trades this CME
		// session-day, block NEW entries until the next session.
		if reason, halted := at.consecutiveLossHaltedAt(now); halted {
			return at.admitRefuse(in, "consecutive_loss", "consecutive_loss_halt: "+reason, func() {
				at.logWarnf("🛑 consecutive-loss halt: %s entry REFUSED — %s. No new entries until the next CME session.", sym, reason)
				// W6 — P0 halt alert, deduped to once per CME session-day.
				at.emitAlert("P0", "halt", "halt:"+kernel.CMESessionDayKey(now),
					"🛑 Consecutive-loss halt", reason)
			})
		}
	} else {
		// Arm / picture: the ONE session-risk verdict (breaker + no-trade band +
		// T1 + per-session trade cap), refusal only — the arm pass's own caller
		// keeps its band-time cancel of resting arms.
		if risk := at.sessionRiskGateAt(now); risk.Refuse {
			return at.admitRefuse(in, risk.Class, risk.Reason, func() {
				at.logWarnf("🛑 session risk: %s %s REFUSED — %s", sym, act, risk.Reason)
				if risk.Class == "consecutive_loss" {
					at.emitAlert("P0", "halt", "halt:"+kernel.CMESessionDayKey(now),
						"🛑 Consecutive-loss halt", risk.Reason)
				}
			})
		}
	}

	// P2.3 — LAST-ENTRY cutoff: block NEW entries after the day-trader last-entry
	// time (default 13:00 CT = 14:00 ET). Gated on day_plan → dormant by default.
	// ALL paths (CTO Q4): an arm placed between the cutoff and the EOD flat is
	// the class of bug W-EXEC-TRUTH exists for.
	if reason, blocked := at.entryBlockedByLastEntryAt(now); blocked {
		return at.admitRefuse(in, "last_entry", "last_entry_cutoff: "+reason, func() {
			at.logWarnf("🕒 last-entry cutoff: %s %s REFUSED — %s. Entries reopen next session.", sym, act, reason)
		})
	}

	if in.Path == admitDecision || in.Path == admitAgent {
		// P3.1 — SESSION GATE: entries only inside an ENABLED session window and
		// outside the no-trade sub-windows (first-5m, lunch). Gated on day_plan.
		if reason, blocked := at.sessionEntryBlockedAt(now); blocked {
			return at.admitRefuse(in, "session_gate", "session_gate: "+reason, func() {
				at.logWarnf("🗓️ session gate: %s %s REFUSED — %s.", sym, act, reason)
			})
		}
	} else {
		// CME closed (weekend / holiday / daily halt): the arm and picture paths
		// have no runCycle skip in front of them (Picture runs on the live-bar
		// goroutine), so they ask the calendar directly (CTO Q3).
		if closed, reason := kernel.CMEClosedReason(now); closed {
			return at.admitRefuse(in, "cme_closed", "cme_closed: "+reason, func() {
				at.logWarnf("🌙 cme closed: %s %s REFUSED — %s.", sym, act, reason)
			})
		}
	}

	if in.Path == admitDecision || in.Path == admitAgent {
		// W9 — PLAN-MODE gate: advisory (default) never gates; direction blocks
		// entries against the plan bias; strict blocks entries with no matched
		// scenario cited. Gated on day_plan → dormant by default.
		if reason, blocked := at.planModeBlockedAt(in.Decision, now); blocked {
			return at.admitRefuse(in, "plan_mode", "plan_mode: "+reason, func() {
				at.logWarnf("📐 plan-mode: %s %s REFUSED — %s.", sym, act, reason)
			})
		}
	}

	// W9 — APPROVAL gate: when approval_required is ON, entries are HELD until the
	// owner approves this CME session-day (POST /api/plan/approve). Default OFF =
	// fully automatic. ALL paths (CTO Q4).
	if at.approvalRequired() && !at.approvalGranted(now) {
		return at.admitRefuse(in, "approval_required", "approval_required", func() {
			at.logWarnf("✋ approval required: %s %s HELD — awaiting owner approval for this session.", sym, act)
			at.emitAlert("P0", "approval", "approval:"+kernel.CMESessionDayKey(now),
				"✋ Entry held — approval required", sym+" "+in.Action)
		})
	}

	if in.Path == admitArm || in.Path == admitPicture {
		// B7 — RE-ENTRY COOLDOWN, non-destructive (CTO Q5): the same verdict the
		// AI kernel reads, but a peek never clears the record the AI path relies
		// on. No ATR15 on these paths → the timer is the only unlock (stricter).
		if cd := at.reentryCooldownMinutes(); cd > 0 {
			if _, reason, blocked := discipline.ReentryPeek(at.id, in.Symbol, in.side(), cd, 0, in.Price, now.UnixMilli()); blocked {
				return at.admitRefuse(in, "reentry_cooldown", "reentry_cooldown: "+reason, func() {
					at.logWarnf("⛔ re-entry cooldown: %s %s REFUSED — %s.", sym, act, reason)
				})
			}
		}
	}

	// CLASS 48 — the ONE canonical entry gate. The decision path runs it here
	// at the LIVE execution price; the picture path runs its picture intent
	// (item 7); the arm path ran it at authoring THIS pass (G1: only a row
	// admitted this pass is placed).
	switch in.Path {
	case admitDecision:
		live := 0.0
		if md, merr := market.GetWithExchange(in.Symbol, at.exchange); merr == nil && md != nil {
			live = md.CurrentPrice
		}
		reason, refused := at.entryGateForDecisionAt(in.Decision, live, now)
		recordResearchGate("decision", "", 0, in.Decision.CitedScenario, reason, refused)
		if refused {
			if in.Record != nil {
				entryGateDecisionTelemetry(at, in.Record, reason)
				return in.Record.Error, true
			}
			return reason, true
		}
	case admitAgent:
		live := 0.0
		if md, merr := market.GetWithExchange(in.Symbol, at.exchange); merr == nil && md != nil {
			live = md.CurrentPrice
		}
		if reason, refused := at.entryGateForDecisionAt(in.Decision, live, now); refused {
			if !strings.HasPrefix(reason, "entry_gate:") {
				reason = "entry_gate: " + reason
			}
			return at.admitRefuse(in, "entry_gate", reason, func() {
				at.logWarnf("🚦 entry-gate REFUSED agent-path: %s", reason)
			})
		}
	case admitPicture:
		if reason, refused := at.pictureEntryGate(in); refused {
			return at.admitRefuse(in, "entry_gate", reason, func() {
				at.logWarnf("🚦 entry-gate REFUSED picture-path: %s", reason)
			})
		}
	}
	return "", false
}

// reentryCooldownMinutes is the strategy's re-entry cooldown (0 = off).
func (at *AutoTrader) reentryCooldownMinutes() int {
	if at.config.StrategyConfig == nil {
		return 0
	}
	return at.config.StrategyConfig.RiskControl.ReentryCooldownMinutes
}

// runningNow reads isRunning under its lock (safe from any goroutine).
func (at *AutoTrader) runningNow() bool {
	at.isRunningMutex.RLock()
	defer at.isRunningMutex.RUnlock()
	return at.isRunning
}

// AdmitManualEntry runs the one admission chain for an entry the owner asked
// for in agent chat (W-EXEC-TRUTH W0, CTO Q17). It is the decision chain with
// a decision that cites nothing, so STRICT refuses it exactly as it refuses an
// uncited AI decision. Returns the refusal text and true, or ("", false).
func (at *AutoTrader) AdmitManualEntry(symbol, action string) (string, bool) {
	return at.AdmitManualEntryAt(symbol, action, time.Now())
}

// AdmitManualEntryAt is AdmitManualEntry on an injected clock.
func (at *AutoTrader) AdmitManualEntryAt(symbol, action string, now time.Time) (string, bool) {
	if action != "open_long" && action != "open_short" {
		return "", false
	}
	return at.admitEntry(admitIntent{
		Path: admitAgent, Symbol: symbol, Action: action, Now: now,
		Decision: &kernel.Decision{Action: action, Symbol: symbol},
	})
}
