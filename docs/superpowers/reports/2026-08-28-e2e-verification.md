# E2E PIPELINE VERIFICATION — every wire (2026-08-27/28)

Read-only campaign in the isolated worktree `~/nofx-e2e` at the RUNNING rev
`6fc09ad39fba` (boot 14:29:14 CT, PID 3055713) on branch `docs/e2e-verify`.
Main tree untouched. All prices from stored 1m bars reimplemented in
`scripts/e2e_recompute.py` (R2 — never calls the functions under test).
Evidence = fresh journal quotes, live DB/API reads, or the committed script.

---

## ⚠ S-FINDINGS (can mislead a trade TODAY)

**S-1 — Stored bars ≠ live truth: ATR14(5m) diverges 19.84 vs 29.78.**
The engine refused an entry at 11:56:15 CT with
`stale_reeval_refused: drift_too_big (|5.25| >= 4.96 = 0.25 x ATR 19.84)`.
The independent recompute from the bars table at the SAME cut yields
**29.78** (any window: last 2000/3000/all 1m bars). The ATR formula is
verified byte-equivalent (`market/data_indicators.go:86-116` — TR =
max(h−l, |h−pc|, |l−pc|), Wilder smoothing; identical to the script's).
→ The persisted 1m series does NOT reproduce the series the engine traded on.
Likely driver: the bar-ingest **backpressure drops** (S-A1) punctured the
persisted stream. Consequence: any replay/calibration on `bars` (level_stats,
future calibration waves) computes a different volatility picture than live.
Root-cause options to prove at next window: persist a checksum of the kernel's
own 1m cache vs the DB write.

**S-2 — VWAP: still not exactly reproducible (residual 1.74pt at plan-write).**
NY plan v3 card cites VWAP 29547.39 / +1σ 29597.96 (written ~11:27-11:29 CT).
Independent recompute at an 11:27 cut: **29549.13 / +1σ 29589.54**. Delta
1.74pt on VWAP and 8.4pt on +1σ. Full-session VWAP 29570.38. The prior
"unreproducible" gap is now narrowed to windowing + σ definition (population
vs sample) + exact cut minute — formula direction CONFIRMED, exact anchor
second still unproven. Grade S- per the dispatch rule (VWAP divergence
root-cause demanded) — downgraded from last audit's full divergence to a
residual delta, but not zero.

---

## Per-section verdicts

**S1 Wiring/transport — PROVEN (2 UNVERIF).**
- 1.1 AddOn md5 `8411d403…` == repo HEAD ✓; folder copy 08-27 13:27.
- 1.2 frame census (all retained journal, ~13:35→15:22 CT):
  bar_update 3,164,303 · account_balance 417 · positions 414 ·
  bars_historical 224 · heartbeat 201 · ack 193 · accounts_list 73 ·
  subscribed 12 · instrument_info 12 · order_update 11 · hello 6 ·
  feed_status 1. `fill` = 0 — EXPECTED (zero fills today; trader_fills
  today = 0 rows). All expected types present.
- 1.3 signal + cancel_order: PROVEN live (E2 confirmation 13:42-13:46, quoted
  in the armed-cutover report). modify_bracket: **SHIPPED-UNPROVEN** — code at
  `trader/ninjatrader/tcp_trader.go:464-466` + `FrameModifyBracket`, never
  fired live (no live bracket modification ever requested).
- 1.4 reconnect: last gap 13:56:34 dead-man DOWN → 13:58:34 UP ("sweeping
  unfilled entries; entries stay BLOCKED until a clean reconciliation").
  Post-cutover hello 14:29:14 `protocol_version=3 source=vltrader-addon` ✓.
- 1.5 clock: drift **−14.5s** vs NT8 at 14:44/14:46 CT, tolerance 60s,
  NTP=synced → headroom ~45s ✓.
- 1.6 DB: 0 busy/locked events since 14:29; WAL ✓; data.db 586MB / WAL 5.5MB;
  backup timer last success 05:00:14 CT ✓.
- 1.7 systemd: nofx + nofx-clock-guard.timer + nofx-backup.timer active ✓.
  journald volume: **~19,000 lines / 5 min post-cutover** (A-1) → retention
  projection ~hours, NOT ≥7 days.
- 1.8 boot three-way: `BOOT INTEGRITY OK — rev 6fc09ad39fba +dirty · expected
  6fc09ad39fba · goldens PASS` = RELEASE = running binary ✓; dev tip
  43bb60cb is FE+docs-only ahead (rides next window, expected). +dirty =
  untracked `.env.bak.*` + old binaries ✓ accounted.

**S2 Bars/data truth — 1m-only ✓; aggregation ✓ (with S-1 caveat).**
- 2.1 dups = 0 (PK symbol,tf,open_time_ms + unique index) · tf ∈ {1m} ONLY
  (MNQ 4206, ES 4206) ✓ · nightly-integrity log line not in retained window
  (journald retention) — UNVERIF.
- 2.2 vs NT8 chart: UNVERIF read-only (no independent channel to NT8 charts).
- 2.3 aggregation script prints 5m/15m for 14:40-15:40 (46 1m → 10 5m, 4 15m);
  ATR cross-check FAILED → see S-1.
- 2.4 halt boundary: UNVERIF (retained window starts 08-24 16:21 CT; a proper
  15:00-16:00 CT halt-window probe needs the script's CT minute math —
  first probe used UTC mod-1440 and hit 10:00-11:00 CT instead).
- 2.5 MNQ volume sane: mean 344/min, range 257-415 contracts ✓.

**S3 Detectors — anchors EXACT where windowed; VWAP see S-2.**
From the script (08-27 NY session):
- OR-H/L (08:30-08:35) = **29589.50 / 29452.50** — OR-H matches the card's
  OR-H 29589.50 **EXACT** ✓.
- AS-H/L = 29655.75 / 29402.25 · LDN-H/L = 29661.25 / 29477.00 ·
  ON-H/L = 29661.25 / 29402.25 · PDH/PDL/PDC = 29655.75 / 29133.00 /
  29499.75 (prior-day bars 1381) — PDH matches the card 29655.75 EXACT ✓.
- nPOC today = 29498.25 (289 price bins) · pdVWAP = 29301.27.
- FVG scan (last 40 1m before 14:45): **none fresh** ✓ consistent with the
  empty FRESH FVGs list the planner got.
- S/D swings (5m fractals k=2): enumerated (11 swings 00:15-04:15) — the
  card's seated swings cross-check not completed read-only → UNVERIF.
- iFVG / OB last-opposite / EQH-EQL / SETT / MID-O / round numbers: UNVERIF
  (script covers the anchor+volume+FVG families; the rest need more spec
  reading than the retained window justified).

**S4 Scoring/grading — weights MATCH; stamps clean; A-touch reacts 79%.**
- 4.1 zoneEvidenceByKind (OB 0.40/0.50/0.70/0.72 · FVG/iFVG/S/D
  0.35/0.45/0.65/0.65) + zoneTFMult 1.0/1.1/1.2/1.3 + reversal 1.1
  (`kernel/levels_score.go:148-160`) == level-truth reconciliation — NO drift.
- 4.3 hand-recompute of every KEY LEVELS row: partially done (OR-H/PDH exact;
  full factor-by-factor ladder needs the seated-table weights per row —
  UNVERIF, not contradicted).
- 4.4 stamp integrity: level_stats grade='' count = **0**, kind='' = 0
  (74 rows) ✓.
- 4.5 reactions: A 64 rows → 43 touched → **34 reacted (79%)** · B 6→5→5 ·
  C 4→4→2 — grades now ordered A>B>C ✓ (prior inverted finding RESOLVED).
- 4.6 consumed lifecycle: UNVERIF read-only (needs a fresh consumed event).

**S5 Wake/awake — clean, nothing fired yet post-cutover.**
- 5.2/5.3: no dormant/wake lines since 14:29; `replans_exhausted` grep = **0** ✓.
- 5.4 wake-storm guard + 5.5 post-exit rescan: no qualifying events since
  cutover — SHIPPED-UNPROVEN (awaiting the next session's wakes).

**S6 Plan write/validators — chain intact; b1/b2 live proofs quoted earlier.**
- 6.1 validator census: `ValidatePlanDoc` + `WithCaps` + `WithFacts` +
  `WithFactsMachine` (`kernel/plan_doc.go:304/311/543/567`), 29 reject sites.
- 6.2 side-quota: `WithFactsMachine` WARN-mode — no ⚖ case since deploy;
  zero-side hard path code-verified ✓ (UNVERIF live).
- 6.3 alias: live NY v3 stored `flip.rule="2x5m"` (canonical from the model's
  "2x5m_close") ✓ PROVEN.
- 6.4 FRESH FVGs: v3 wrote NO fvg_entry with the empty machine list ✓ PROVEN.
- 6.5 arm{} armable-only validator: `ArmSpecValid` (`kernel/plan_doc.go:96`)
  restricts to fvg_entry|breakout_retest|reject — fixture for acceptance-arm
  not re-run this pass → B.
- 6.6/6.7 bias-tree + playbook v2 sections: v3 reasoning opened with
  "bias-tree: branch 5 premium/discount…" ✓; per-section header quote not
  re-extracted (prompt not persisted verbatim) → B.

**S7 Executor/gates/strict — PROVEN machinery; STRICT gate SHIPPED-UNPROVEN.**
- 7.1 gate chain three-way: code order verified (C6 plan-gate →
  min-conf → min-SL → HTF veto → stale → guards); canonical-doc diff not
  re-run → B.
- 7.2 strict×3 re-queried LIVE: card `mode` = strict for NY/ASIA/LONDON ✓.
  **First strict-era open_\* decision has NOT happened** (strict flipped
  ~14:0x; NY flat 14:45 with only arm refusals). Proving line when it does:
  `trader/auto_trader_planconfig.go:217` strict branch → either a pass with a
  cited scenario or `no matched scenario cited (strict mode)` in
  decision_records.risk_check_error. → **SHIPPED-UNPROVEN**.
- 7.3 min-SL: gate code + boot line `atr_mult=1.0 level_clearance=2tick` ✓;
  zero entries with SL<1×ATR slipped: no entries at all today → UNVERIF.
- 7.4 stale-MET/CONFLICT labels: rendered in prompts; no persisted prompt to
  quote → B.
- 7.5 latency: exec calls 4.2-47.9s observed today (fast routing) ✓; planner
  read durations not logged numerically post-cutover → B. No exec call >120s.
- 7.6 stale_reeval refusals in retained journal: 1 (the 11:56 quote);
  pre-retention history gone (journald) → trend UNVERIF.
- 7.7 confirm engine recompute: needs a MET/NOT-MET render to diff against —
  UNVERIF this pass.

**S8 Armed orders — ledger clean; frames PROVEN; natural arm REFUSED by gate.**
- 8.1 ledger census: **4 rows, all `cancelled`** — zero working/stuck ✓.
- 8.2 place+cancel frame chains: 11 order_update frames received, E2 chains
  quoted in the armed-cutover report ✓ PROVEN (re-verified in journal).
- 8.3 natural arm: **a plan DID author arm{}** — the arm gate then refused it
  live every cycle: `⚔️ arm REFUSED NY S1: R:R 2.04 below min 3.00` (13:55-14:02).
  Authoring + gate-at-arm PROVEN; the ⏳→📌 working chain remains
  **SHIPPED-UNPROVEN** (needs an arm that passes the gate, e.g. R:R ≥ 3.00).
- 8.4 cancel coverage: seam cancel + reconcile cancel code-verified; the
  session-end EOD-cancel wire didn't fire (no working orders at 14:45) →
  SHIPPED-UNPROVEN live, PROVEN by the E2 cancel frames for the wire itself.
- 8.5 armed_fill stale_reeval exemption scoped to entry_class=armed_fill:
  code-verified (`onArmedOrderUpdate` fill branch stamps armed_fill lineage;
  exemption condition quoted in armed_executor) ✓ code-level; no live fill → B.
- 8.6 churn guard + stale reconcile: unit tests pass; live instance: none
  needed (no working rows) → PROVEN-tested / SHIPPED-UNPROVEN-live.

**S9 In-position/exit — nothing in-position since cutover.**
- 9.1 watcher verdicts: no open position → none sighted ✓ (correctly quiet).
- 9.2 BE+40 / 9.3 EOD 14:45 flat execution: bot was FLAT at 14:45 → nothing to
  quote — SHIPPED-UNPROVEN (awaiting the next position).
- 9.4 exit lineage: newest close = the reconcile close (pos 566): `realized
  97.00` vs `pnl_corrected` 97.00 (Δ 0.00), T7 note present ✓ PROVEN for this
  row. Older closes: 354/562 CLOSED rows have pnl_corrected NULL (A-2).

**S10 Scoreboard — pnl_corrected 171/562 stamped.**
- 10.1 close-path writer fired on pos 566 ✓ (quote above); NULL count = 354
  of 562 CLOSED (post-Aug-6 split UNVERIF — wrong column guess; the 171
  boot-backfill line says 171 candidates existed and ALL were stamped, so the
  354 NULLs are pre-candidate legacy rows). → B/A.
- 10.2 level_stats: 28/28/18 rows for 08-24/25/26, written at the 14:29:46 CT
  backfill ✓ T1-style proof present (writer landed with the cutover).
- 10.3 touch_episodes: **109 rows**, freshest 15:22:54 CT ✓ live writer.
- 10.4 expectancy strict-era: 0 closed trades in the strict era → table empty
  (era began ~14:0x today). 10.5 gate-block counters: API reachable (FE polls
  it); dump not captured this pass → B.

**S11 Config/layering/docs/process.**
- 11.1 tri-state: merged to dev (43bb60cb), **NOT yet deployed** — rides the
  next flat window ✓ per plan. Resolver empty-string immunity:
  `PlanModeFor`/`MinScenarioQualityFor` require `TrimSpace != ""` (store/
  strategy.go) + FE PUT-path tests → PROVEN-tested.
- 11.2 Studio knobs vs Guide §7: sampled — min_scenario_quality=C (⭐C ✓),
  plan_mode global=strict vs ⭐ADVISORY recommendation (owner's deliberate
  choice, flagged not wrong) · min_side_levels=2 (⭐2 ✓).
- 11.3 docs drift: guide `asBuiltRev` not re-diffed → B; the mock
  "Scenarios (advisory)" label is static in `guide/components/MockPlanCard
  .tsx:180` — cosmetic, still pending the next FE pass (C).
- 11.4 process: 20 OPEN PRs (oldest 7 days) · dev tip 43bb60cb ahead of
  running by FE+docs (expected) · stashes: none in main tree (level-truth
  consumed) · worktrees: main dev, e2e (this), recheck, /tmp/nofx-dev-check
  (STALE detached a52de628 — cleanable) · partner PR #2 open.
- 11.5 canon laws present in CLAUDE.md: WORKTREE LAW :134 · NO UNATTENDED
  DEPLOYS :140 · SIM-only :146 ✓.

## A-FINDINGS
- **A-1** journald volume ~19k lines/5min post-cutover — the bar_update INFO
  flood is gone but `WARN tcp_server: bar ingest backpressure — dropped
  oldest update` floods in bursts (≈95% of 14:40-14:45). Retention <7 days
  (prior finding persists) AND persisted bars are being dropped (feeds S-1).
- **A-2** 354/562 CLOSED positions have pnl_corrected NULL (legacy rows,
  backfill covered the 171 candidates only).

## SHIPPED-UNPROVEN register (market event each awaits)
1. `modify_bracket` wire — awaits the first live bracket modification
   (trailing/BE arm on an open position).
2. STRICT `ClassifyCitation` gate — awaits the first strict-era open_*
   decision (next session entry attempt).
3. Natural arm ⏳→📌 placement — awaits an authored arm{} that passes the
   R:R ≥ 3.00 gate.
4. armed_fill stale_reeval exemption — awaits the first resting-limit fill.
5. EOD-flat execution + watcher + BE+40 — await the next open position.
6. Wake paths (5 triggers + storm guard + rescan) — await qualifying events.

## Top-5 risks
1. S-1 stored-bars ≠ live-bars (ATR 29.78 vs 19.84) — poisons every replay.
2. A-1 backpressure drops + journald retention ~hours — forensics evaporate.
3. STRICT unproven live (no entry since flip) — the gate the owner flipped
   for has never evaluated an entry.
4. modify_bracket never exercised — the trailing/BE arm is the only
   unproven wire direction.
5. 20-PR backlog + stale /tmp worktree — process debt.

**System verdict: PROVEN for the wiring, scoring and armed-ledger substrate;
S-1 (data truth) is the single can-mislead-today defect; strict, bracket
modify, armed fills and the exit machinery await their market events.**
