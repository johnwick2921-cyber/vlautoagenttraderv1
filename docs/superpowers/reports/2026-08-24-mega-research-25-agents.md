# MEGA RESEARCH — 25 Agents (2026-08-24)

Owner mandate: deep online research (20-min minimum hard rule), 20 external
megaagents, 5 system-reader agents. Full synthesis below.

## Part A — External research (20 agents)

### A1. AI trading agent reliability (R1)
- Published evidence supports: LLM as planner/signal-extractor UNDER a
  deterministic fail-closed gate — never the execution path. Our architecture
  already matches this.
- Retries only help with new external evidence (64.5% self-correction blind
  spot, arXiv 2507.02778). Cap tokens/time per decision.
- Knight Capital lesson: human + machine kill switch OUTSIDE the LLM path.
- Treat non-determinism as a defect signal: pin temp/seed, replay harness,
  alert on action flips.
- Sandbox prompt injection: all ingested text (news/calendar/chat) is data.

### A2. Multi-timeframe + HTF bias (R2)
- 3-TF ladder standard (4h direction / 1h setup / 15m entry). HTF-wins rule.
- Premium/discount on the dealing range is the standard HTF bias mechanism.
- "2 closes beyond a level" (our 2x5m flip/death rule) is a published
  convention (Netpicks "2-candle rule", close-based invalidation). KEEP.
- 30-min flip hold mirrors published 3-bar regime hysteresis. KEEP.

### A3. Position sizing (R3)
- Prop-firm norms: ~1% risk/trade, daily loss limits (Topstep/Apex).
- Stops: 1–2×ATR is the standard range; wick-hunting argues for stop beyond
  the level + buffer, not exactly at it.
- Fixed-fractional over Kelly for noisy short-horizon strategies.

### A4. Calendar/event risk (R4)
- CPI/FOMC/NFP are the genuine NQ movers; blackouts 10–30 min both sides are
  standard practice.
- Treasury auctions move index futures measurably (T2-caution is right).
- Half-day sessions: reduced liquidity, size down or skip.

### A5. Evidence for S/D, OB, FVG, EQH (R5 — most important)
Evidence tiers from the research:
- **Tier 1 (real data):** prior-swing S/R (Osler 2000/2003 — dealer clustering,
  round numbers), PDH/PDL/OR/IB engagement stats (IB breaks ~97% of days,
  retrace ≥50% ~51–56%), HTF FVG size-graded hold rates (~53% at ≥1×ATR).
- **Tier 2 (thin):** OB ≈ S/D zones (PF 2.16 vs 1.65 in one audit);
  OB+FVG confluence = strongest tier in a 5,400-gap study; EQH/EQL = context
  amplifiers, NOT standalone entries (PF 0.8–1.4 standalone).
- **Tier 3 (no/negative data):** LTF 1m FVGs (−0.34R, 44% survival),
  stale zones (77% of successful mitigations within 12 bars), breaker blocks,
  killzones.
- **Grading implication for our system:** freshness is a mandatory axis
  (already have it); zone SIZE ÷ ATR is the dominant published axis (we don't
  measure it — flagged); keep 1m→C (supported); EQH/EQL above C only via
  confluence (our typeEvidence 0.70 + HTF×1.2 handles this).

### A6. Prompt engineering (R6)
- Structured/JSON-mode output, reasoning-first, machine facts in = current
  best practice. Validation-feedback retries with the error message appended
  are standard; 3 attempts reasonable.
- Grounding (Go-computed tables) is exactly right.

### A7. Day-plan/re-plan architectures (R7)
- Session plan with bias/levels/scenarios/invalidations is the documented
  professional pattern (pre-market plan, intraday execution).
- Plan adherence > discretion (psychology literature). Re-plan on invalidation
  with a cap = standard.

### A8. Data feed latency (R8)
- Seconds-level 1m-bar decision loops are fine for intraday scalping.
- Staleness gates (last-bar age) standard; our 10-min FEED_ALERT_S is
  generous but safe. Fail-closed on stale = correct.

### A9. Win-rate/expectancy (R9)
- At 36.6% WR: breakeven R:R ≈ 1.73. PF 0.98 ≈ breakeven — the bot is at the
  boundary; grading must be calibrated by outcome, not narrative.
- n=191 trades: 95% CI on 36.6% ≈ ±6.8pp. Need ~200+ per grade stratum for
  A/B/C validation.
- Expected max losing streak at 40% over 200 trades ≈ 9–10. Daily loss limits
  matter more than grades right now.

### A10. Session windows (R10)
- Verified session table: ASIA 17:00–02:00 CT, LONDON 02:00–08:30, NY
  08:30–15:00 (RTH close 15:00 CT; globex reopens 17:00 after the 16:00–17:00
  maintenance). Our NY flat 14:45 is INSIDE RTH — confirmed correct.
- ONH/ONL convention: 17:00 CT post-settlement to RTH open — matches ours.

### A11-A20 (ATR/round numbers/stops/models/guardrails/FVG-fill/SQLite/deploy/gaps/regime)
Key nuggets:
- Round numbers: Osler replication supports 0/5-ending clustering; round
  numbers = real order-flow structure. Keep 0.55 evidence.
- Stop beyond level + ATR buffer beats exact-level stops.
- Kill switches: daily loss limit + max orders/min are industry standard.
- FVG fill-rate lore is unsourced; size-vs-ATR is the real axis.
- SQLite WAL + busy_timeout + periodic integrity_check = our current setup is
  correct; add VACUUM INTO backups only (never file copy live).
- Deploy discipline: flat-window deploys, config toggles over code changes
  (matches owner's "no hardcode" rule).
- PDH/PDL engagement stats justify typeEvidence 1.0. nPOC 0.85 justified as
  magnet, never standalone trigger.
- Regime: 2x5m flip = keep; VWAP advisory recommended as future level;
  RV buckets before any new hard gate.

## Part B — System readers (5 agents) — bug lists

### S1. Planner lifecycle — 9 findings
1. **HIGH — MSS wake bypasses the re-plan cap** (auto_trader_transition.go:144)
   — unbounded replans, owner overlays stranded.
2. **HIGH — same-cycle double-append** on stale plan (MSS wake writes before
   death check evaluates pre-wake version) — over-spend + death verdict on a
   superseded plan.
3. MED — dual cap sources: cached vs live replan_cap diverges mid-session.
4. MED — W16 claim key is trader-scoped despite the cross-trader-collapse
   design comment (both traders pay for the same read).
5. MED — re-read races: double-spend on concurrent POSTs, silent claim-skip
   reported as success, no carryOwnerEditsInto.
6. LOW — reset baseline race with concurrent death re-plan.
7. LOW-MED — registry validation doesn't constrain read-window geometry.
8. LOW — G7 staleness judges 15m-close conditions on 5m aggregation.
9. LOW — half-day latency (next-day refresh).

### S2. Executor + risk gate — 9 findings
1. HIGH — no executor-side plan-death gate: executor trades a dead plan's
   scenarios until a read-wake re-plans.
2. HIGH-MED — entry rejections invisible: phantom open position until
   reconcile; dashboard can show a position NT8 never held.
3. MED — queued-signal flush on reconnect (stale decisions up to 60s).
4. MED — drawdown safety-close dead on NT futures (crypto-calibrated
   thresholds; leverage always 1.0).
5. MED-LOW — clock-drift gate is log-only.
6. LOW-MED — RISK_MAX_NOTIONAL_USD dormant on futures.
7. LOW — max-positions default mismatch (2 vs 3).
8. LOW — dailyPnL display never written.
9. LOW — misleading LONG bracket failure log; SHORT twin ignores errors.

### S3. Data bridge — 7 findings
1. HIGH-if-used — bar_cache timeframeMs missing 6h/8h/12h/3d/1w (latent).
2. MED — AddOn cursor reset on Go reconnect: full re-emit + incoming-wins
   overwrites fresher live bars.
3. MED — silent .Update stall: up to 75-min exit-price blindness.
4. LOW-MED — IsFeedConnected true while updates discarded.
5. LOW — livenessReporter hardcodes "MNQ".
6. LOW — no gap marking; out-of-order bars dropped silently.
7. LOW — protocol doc drift (10 of 22 frame types).

### S4. Store + calendar — 9 findings
1. **HIGH — reset baseline not trader-scoped** (store/strategy.go:1116):
   trader A's reset re-arms trader B's budget silently.
2. MED-HIGH — owner re-read orphans overlays (no carryOwnerEditsInto).
3. MED — reset vs concurrent death re-plan burns one budget unit.
4. MED — UpdateLiveSliceIfChanged rewrites PAST dates (replay integrity
   violation).
5. LOW-MED — currentT1Windows fail-open residual (store==nil guard).
6. LOW — level-state identity diverges from schema (origin_date dead).
7. LOW — OwnerLevelStore exact float equality vs ±0.125 carry tolerance.
8. LOW — RecordPlay/DecrementFreshness non-transactional.
9. LOW — read-claim comment drift.

### S5. Web plan UI + agent — 9 findings
1. **P0 — Cross-user IDOR on /api/plan/*** (any valid JWT + trader_id reads
   AND mutates anyone's plan).
2. **P0 — owner-level leak/overwrite across users** (symbol-global table).
3. P1 — global-trader fallback lets a user with no traders read another
   user's orders.
4. P2 — transition field never emitted (G4 chip dead UI).
5. P2 — stale index race in EditSheet replace.
6. P3 — trader_diagnosis tool_mapping references nonexistent tools.
7. P3 — ToolMapping parsed but unused.
8. P3 — public /api/agent/klines proxy without auth.
9. P4 — planner-tool vs skill dual surface.

## Part C — Consolidated action list (owner decides priority)

**P0 security (fix first):**
- C1. IDOR on /api/plan/*: scope every plan lookup/mutation by the JWT's
  user_id → trader ownership.
- C2. owner_levels cross-user leak: scope by user_id.
- C3. global-trader fallback in getTraderFromQuery.
- C4. auth on /api/agent/klines.

**High (trading integrity):**
- C5. MSS wake cap + no same-cycle double-append (S1-1/2).
- C6. Executor-side dead-plan gate (S2-1).
- C7. Reset baseline trader-scoped (S4-1).
- C8. Entry-rejection reconciliation visibility (S2-2).

**Medium:**
- C9. re-read races + carryOwnerEditsInto (S1-5, S4-2).
- C10. UpdateLiveSliceIfChanged past-date freeze (S4-4).
- C11. BarCache timeframeMs table completion (S3-1).
- C12. AddOn cursor persistence (S3-2).
- C13. zone size ÷ ATR grading axis (A5 — the dominant published axis we
  don't measure).
- C14. confluence cap at 3 (diminishing returns, prior audit).

**Deferred to replay-data validation (A9):**
- Any new hard gate (VWAP, RV buckets, CHOP) only after ≥200 trades and
  in/out-of-sample validation. Grade calibration by outcome expectancy, not
  narrative.

## Part D — Verification status
- 20 external research reports + 5 system-reader reports complete with
  citations; 3 pages rate-limited (flagged [B] in R19).
- Live system healthy on rev c1cf4fdb: ASIA v3 active, 9 unique levels,
  BOOT INTEGRITY OK.
