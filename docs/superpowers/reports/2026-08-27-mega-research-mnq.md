# MEGA-RESEARCH WAVE — 20 AGENTS · MNQ LEVELS, MATH, DATA · "WHAT ELSE NEEDS FIXING"

- **Repo:** nofx · **Branch:** `docs/mega-research-mnq` · **Mode:** READ-ONLY (zero code/config changes; DB opened `mode=ro`)
- **Base:** deployed rev — `🔐 BOOT INTEGRITY OK — rev 3de37a7e84b7 +dirty · built 2026-08-26T20:21:56Z · expected 3de37a7e · goldens PASS` (boot 15:22:29 CDT, PID 2345900; log `data/nofx_2026-08-26.log`). Built-at equals PR #79 merge commit time.
- **Canonical map:** pipeline-confirmation (report `08d65780`, file `docs/superpowers/reports/2026-08-26-pipeline-confirmation.md`).
- **Prior research respected, not re-derived:** `docs/superpowers/reports/2026-08-24-mega-research-25-agents.md`, `2026-08-24-level-grading-full-audit.md`, `2026-08-25-1h-timeframe-research-wave.md`, `docs/research/plan-card/*`.
- **Wall clock:** START 2026-08-26 15:33:40 CDT → END 2026-08-26 16:06:00 CDT (≥30 min, quoted at commit).
- **Out of scope (already scheduled):** Sep-3 LEVEL-SYSTEM DEEP VERIFICATION · REFUSAL AUTOPSY · FVG displacement 1.25 re-tune · B4 level_stats verdict. Nothing here duplicates them.
- **Grading:** S = wrong math/logic that misleads trades NOW · A = gap vs research consensus · B = calibration doubt · C = polish. Evidence tier on every claim: **[A]** read/ran it, **[B]** inferred, **[C]** speculation. `[OWNER]` = owner-ruling item.

---

## 0. Disagreements surfaced (not averaged)

1. **A13 vs dispatch premise:** the dispatch said the TIER1 12-tick gate ≈ "48 pt band". That is wrong math: 12 MNQ ticks × 0.25 = **3.00 pts** (code comment `kernel/levels_score.go:246` "12 MNQ ticks = 3.00 points"). The 48-pt figure came from multiplying 12 by the *wrong* unit. A13's reading stands: band = 3.00 pts.
2. **A9 vs spec text:** spec says "anchors no-decay"; code comment (`levels_score.go:357-360`) says anchors keep the gentler ladder "per spec" — but the freshness WRITER (`trader/auto_trader_levelstate.go:77-134`) has no kind exemption: PDH/PDL/ONH/SETT decay on every play like zones. Comment ≠ code.
3. **A6 vs fvg_entry docstring:** `fvg_entry.go:22-25` cites "the MSS/displacement research pins ~1.5×" — that research pins 1.5× for **MSS structure detection** (`structure.go:29`), not for FVG entry displacement. Citation drift, not a math error.
4. **A16 vs A13 on "dATR":** both independently found the same thing from different directions — `dATR` in the level path is `DailyATRProxy` = **mean completed session-day range** (~352 pts this week), while regime `ATR14` is a true Wilder daily ATR. Convergent, adopted as S1.
5. **A2 vs pipeline map:** pipeline-confirmation documents VWAP/VWAP+1σ rows; code emits ±1σ only (no ±2σ). The dispatch asked for ±2σ completeness — registered as a gap (A2.2), not a pipeline-map error.

---

## 1. REGISTER (grade · finding · evidence · fix · size · knob)

### S — wrong now

| # | Finding | Evidence | Fix | Size | Knob |
|---|---|---|---|---|---|
| S1 | **dATR duality poisons the proximity lock.** Level-path `dATR = DailyATRProxy` = mean completed session-day H−L range (measured: 403.2 / 301.2 pts → ≈352). `proximityK 1.5×dATR` ≈ **±530 pts** admits essentially the whole day's map into the in-band pool, so seats/HTF/volume rebalances run in cut mode daily. `confBand 0.10×dATR` ≈ ±35 pts counts as "confluence". Regime `ATR14` (Wilder daily) is a different number feeding prompts — same variable name, different meaning. | `levels_assemble.go:209-246` (DailyATRProxy), `levels_score.go:397-398,406-411`; regime `regime.go:71` | Retune `proximity_filter_atr` down (1.5 → 0.2-0.4) OR switch the lock unit to a clamped 5m-ATR multiple; unify naming | M | `DayPlan.ProximityFilterATR` (clamped 0.5–3.0 today — the clamp itself forbids a sane value if dATR≈350: 0.5×350=175 pts is still huge) |
| S2 | **ADDENDUM S stale threshold is nearly inert (pre-ship bug).** `staleConfirmAnnotation` fires when `|now−ref| > 1.0×dATR` with dATR = DailyATRProxy ≈ **350 pts**. Week data: 2,908 MET confirms; median `|price−ref| = 58.75 pts`, p75 116.5, **62% > 40 pts, 30% > 100 pts, max 401.75** — the shipped 1.0×dATR line would mark only **38 of 2,908 (1.3%)** as stale, while 2.0×ATR5m (~80 pts) would mark 1,072 (37%). The annotation the audit asked for is effectively dead on arrival. | `kernel/plan_confirm.go:100-127` (`staleConfirmAnnotation`, `StaleConfirmATR`), DB recompute (decision_records 08-20..08-26) | Redefine the unit to 5m Wilder ATR: `STALE_CONFIRM_ATR = 2.0 × ATR5m` (≈ 40-70 pts) matches the empirical stale mass; document dATR≠ATR5m | S | `STALE_CONFIRM_ATR` (env exists; semantics change before deploy — addendum is on dev `2de2a5f6`, NOT deployed) |
| S3 | **Position→plan attribution join is broken; ~37% of the week's trades unresolvable.** `plans.strategy_id` stores the **trader id** (sample: `('2026-08-15:NY', '8d5c8af5_8ef641a7-…')`); positions carry only `plan_version`, and **version numbers are per (trade_date, session)** (08-25: ASIA max v16, LONDON max v2) — a bare version join is ambiguous. Full (date, session, trader, version) reconstruction resolves 24/38 closed positions; **14/38 unresolvable** (mix of "off-plan" cites, version numbers that don't exist for the reconstructed session — e.g. pos 563, entered 08-26 06:12:45 CT (LONDON of session-day 08-25) carries `plan_version=9` while 08-25 LONDON only ever reached v2; the prior ASIA plan's version counter leaks through at session handoff — and genuinely missing rows). Any Sep-3 condition-type table built from this join silently drops a third of the week. | `plans` schema (strategy_id=tid; unique (date,session,strategy,version)), `trader_positions` (plan_version, cited_scenario_id), DB recompute | Persist the resolved `plan_id` (or date+session) on the position at entry; add a migration view `position→plan` | M | none (schema/backfill) `[OWNER]` |
| S4 | **nPOC double-seat duplicate emission + stale-touch window.** `AssembleScoredLevels` appends both `VolumeLevels(...)` (label `nPOC·08-25`) and store-fed `extraLevels` (label `nPOC·2026-08-25`) — different labels defeat the same-price/same-kind/label skip, so the same POC can hold **two of 8 seats**. Touch scan only covers the ~2000-bar slice while 30 stored sessions are fed → a POC touched 3 days ago can never retire. | `levels_assemble.go:79-80`, `levels_score.go:430-431`, `auto_trader_dayplan.go:148-164`, `naked_poc.go:44-59` | Dedupe the two paths on (price, kind); widen the touch-scan window to the nPOC lookback | S-M | none (code) |

### A — gap vs research consensus

| # | Finding | Evidence | Fix | Size |
|---|---|---|---|---|
| A1 | Level taxonomy holes: weekly/monthly H/L **dead** (gated by impossible bar counts `priorWeekMinBars=4320` vs ~33h 1m ring — "these can never pass" in-code); ETH/RTH profile split missing; composite/multi-day VA missing; swing-anchored VWAP missing; half-gap 50% rule missing; multi-day ONH missing; SETT = last 1m close ~105 min after the true 15:14:30-15:15:00 CT settle VWAP. | `levels_multiday.go:194-226`, `levels_volume.go:117-150,308-340`, `levels_intraday.go:57-107` | Feed PWH/PWL/PMH/PML from the durable session-profile store (like nPOC); compute SETT as final-minute VWAP; add half-gap midpoint | M-L |
| A2 | VWAP: no RTH (08:30 CT) VWAP — only the 17:00 CT ETH-anchored one; no ±2σ bands; **eVWAP anchored 16:00 CT is a degenerate duplicate of session VWAP** (no bars exist 16:00-17:00, so the windows are identical — it measures nothing and wastes a seat). σ formula itself is true volume-weighted std (`sqrt(Σv·d²/Σv)`) — correct. | `levels_volume.go:40-56,61-79,93-103` | Re-anchor eVWAP at prior-day 15:00 CT (or drop the kind); add ±2σ; add `KindRTHVWAP` | S-M |
| A3 | Volume profile: 120 uniform bins over the day's H−L (2.5-pt bins on a 300-pt day) with each bar's full volume dumped into its **close's** bin → POC smeared ±1.25 pts, VAH/VAL quantized ±2.5 pts, close-clustering skew. The store path uses the finer SVP 1.25-pt-row grid → **two different POCs for one session** (`pdPOC` vs `nPOC`). 70% VA constant is duplicated inline (0.70 in `levels_volume.go:198` vs `svp.go:37`). | `levels_volume.go:157-226`, `svp.go:28-37` | Reuse the SVP absolute grid; distribute bar volume across [Low, High]; hoist 0.70 to one constant | M |
| A4 | nPOC: no ±1-tick retire tolerance (a 0.01-pt wick retires — the spec's ±1-tick doesn't exist in code); lookback is 10 **calendar** days (≈7 sessions, weekends skipped silently); weekly nPOC is a label-only relabel of >5-day-old daily POCs ("nPOC·wk"), **not** a weekly composite, and has no HTF-seat eligibility (not Tier-1/zone) so it fights for the single volume seat. Doc/code mismatch: struct comment says "first bar open" but birth = last bar's CloseTime. | `levels_volume.go:238-291`, `naked_poc.go:44-59`, `auto_trader_dayplan.go:152-159`, `levels_score.go:724-748` | Add tick tolerance or fix the comment; build a real weekly composite or rename `nPOC·old`; give weekly nPOC a seat rule | S-M |
| A5 | S/D zones: base up to **6** candles (consensus 1-3); departure 1.5×ATR (unit mismatch vs "≥50-100% of base range"); **no width cap at detection** (only score discount zoneSizeMult 0.85/0.70/0.50); **no overlap merge** — and zones are exempt from cluster collapse, so duplicates survive to seats. | `levels_zones.go:103-155`, `levels_score.go:195-214,679-686` | Cap width ~1×ATR at detection; require base-range overlap; merge overlapping same-kind zones | M |
| A6 | FVG: **no session-boundary guard** — index-adjacent triples can straddle the 16:00-17:00 halt/weekend/DST and invent phantom gaps (both the detector and `validateOneFvgEntry`); 2-pt floor (8 ticks) admits microstructure noise as "FVG"; displacement measured on the **newest candle's body only** — a mid-candle impulse false-rejects; 1.5×ATR-body is stricter than published (≥1×ATR range) → fail-closed, missed plays not bad entries. CE midpoint + >20-pt CE gating + iFVG inversion: **verified correct** (`fvg_entry.go:50,112-114,140-171`; `levels_zones.go:211-238`). | `levels_zones.go:190-244`, `fvg_entry.go:128-203` | Reject windows where `c.OpenTime − a.OpenTime > 3× interval` or session differs; scale floor with ATR; measure max body across impulse+middle candles | S-M |
| A7 | OB: last-opposite-candle rule **correct**; 8-bar lookback **correct** (env `OB_LOOKBACK_BARS`); but zones use the opposing candle's **full range incl. wicks** (ICT is body-based); displacement legs emit the same OB twice (no dedupe); **no mitigation state** (touch → retire) and **no invalidation** (close beyond base-candle extreme) — the single biggest cross-cutting gap (grep: zero `mitigat|invalidat|consum` hits in `levels_zones.go`). | `levels_zones.go:249-314` | Add mitigation/invalidation states mirroring nPOC retire-on-touch; dedupe per leg; body-bounds option | M |
| A8 | Sweeps: shape correct (wick-through + close-back), but **zero min penetration depth** — a 1-tick wick counts as a liquidity sweep; EQH/EQL 3-tick tolerance ✓. **BOS never reprices the swing map**: `lastHigh/lastLow` frozen from the confirmed fractal list, so after a BOS-up, CHoCH/sweep tests run against pre-rally extremes until a new fractal confirms (can take many bars). Swept levels are never burned/demoted (roles are static kind→role). | `structure.go:298-380`, `scenario_facts.go:248-297`, `levels_role.go:33-45` | Require ≥2 ticks penetration; on BOS promote the broken extreme; optionally demote swept EQH/EQL via freshness provider | M |
| A9 | Freshness: zone ladder 0.6/0.3/0.15 **drifts steeper than the only published sample** (0.78/0.55 second/third-touch — an OB study, and OB is a zone kind); anchor ladder 0.8/0.6 matches it. The steep ladder was applied to the kinds the citation *didn't* cover. "Anchors no-decay" is comment≠code (writer decays all kinds, `auto_trader_levelstate.go:77-134`). PDH touched 3× today → grade B (max 0.96). | `levels_score.go:342-375,418-425`, `store/level_state.go:30-37,192-209` | Keep anchors on the gentle ladder; re-examine 0.3/0.15 vs published 0.55/0.78 (or document the choice with a source) | S |
| A10 | Confluence: flat +0.20/family with cap 3 (max ×1.6) **has no quality weighting** — a grade-C round number confirms as strongly as an A PDH; same-family dedupe is hard-coded (VWAP±1σ contribute zero to each other — correct); cap 3 matches the only research number (C14, "diminishing returns after ~3"). **Ten** families exist, not five (dispatch premise). | `levels_score.go:182-192,306-338,429-464` | Weight by confirming level's typeEvidence, re-tune cap if inflation appears | S |
| A11 | Grading: 15m zones have a **hard A-ceiling** (force B regardless of score); OWNER kind caps at B (max 0.96) contradicting the "Owner levels grade A" comment at `FilterLevelsByMinGrade:66-67`; A≥1.0 / B≥0.70 are uncalibrated design constants (the full-audit "byte-for-byte" claim predates Pack B). Worked flips: one touch A→B (PDH); one confluence member B→A (4h FVG); one freshness band A→B→C (1h OB). | `levels_score.go:66-67,466-496,627-637`, stale claim in `2026-08-24-level-grading-full-audit.md` | Wait for B4 level_stats before touching thresholds (already scheduled — out of scope) | S |
| A12 | Role grammar: PDC=pivot is the weakest assignment (NQ-empirical PDC is a gap-fill magnet/target; validator bans it from breakout anchoring — the 2 live WARNs today are exactly this combo); SETT=pivot arguable (settle behaves as reversion magnet); MID-O=pivot sits in the "overnight" family (internal inconsistency); iFVG=react_zone force-fits a "trade-back-through" concept. **Missing roles: `breaker`, `trap`, `mitigation`.** AI does receive roles (ROLE column + playbook + bias_ctx, both prompts). | `levels_role.go:25-45,108-127,350-415`, `data/nofx_2026-08-26.log:32239,34734` | Move PDC → magnet or add `trap`/`breaker`; re-role MID-O; ship a `mitigation` state for OB/FVG | S-M |
| A13 | Seats: with the S1 dATR reality, the in-band pool is huge daily; **VWAP/VWAP±1σ share kind+family so they never merge (σ > 3 pts) and can burn 2 of 8 seats** as one visual band; Tier-1 12-tick (3.00 pts) gate is a weak filter (anchors are dense); VAH/VAL/SETT/nPOC are not Tier-1, so a zone beside a volume anchor can never grade B — internal tension with the volume wave's own thesis. Per-hour in-band counts are **not log-observable** (seated tables never logged; `level_stats` logged "evaluated 0 seated level(s)" all day). | `levels_score.go:245-258,282-309,491-496,679-686,755-825` | Same-family seat dedupe; [OWNER] decide volume-anchor Tier-1 membership; add a seated-table debug log | S-M |
| A14 | Touch telemetry: **`TOUCH_VOL_LOOKBACK=20` is dead code** (`if preN > lookback { _ = pre }` no-op — effective baseline up to ~45 bars); 4-pt band is fixed-tick, not ATR-scaled (misses 5-6-pt reactions and slices slow grinds into 2-3 "touches"); "5m close" buckets are anchored at the ring's first bar, not exchange 5m boundaries. Wick/body penetration + close-side accept/reject math: sound, test-pinned. | `touch_telemetry.go:26-69,198,234-348` | Fix or delete the dead branch; ATR-scale the band (`TOUCH_BAND_TICKS` env already exists) | S |
| A15 | Sessions: **live flat = 14:30 CT, not the 14:45 contract** (`enforceEODFlatAt` = session end − `EODFlatOffsetFor` default 15; tests pin 14:30 as intended); lunch is a **hard gate 12:00-13:30 CT** which overlaps the NY pm killzone starting 13:00 (registry's own "high-probability" window hard-blocked 13:00-13:30); DST handling via tzdb America/Chicago ✓; halt/dead-hour windows correctly untraded ✓. | `session_registry.go:76-120`, `auto_trader_clock.go:326-444`, `store/strategy.go:997-1014`, `auto_trader_session.go:123-125` | [OWNER] reconcile 14:30 vs 14:45 (one number wins, comments+FE updated); align lunch vs pm killzone | S |
| A16 | ATR census: **every ATR is Wilder-14** (conformance fix holds; structure ATR test-pinned). Semantic issues: (1) S1's dATR duality; (2) `atr15From` is TF-variable despite its name (15m→5m→3m→1h fallback) — the 8×ATR price-sanity bound means different things per config; (3) min-SL gate fail-opens on missing 5m structure; (4) dormant grid bounds use a 4h ATR. | `market/data_indicators.go:86-118`, `kernel/structure.go:132-170` (✓), `engine_analysis.go:691-700`, `engine_position.go:216-218` | Rename/align `atr15From`; log which TF the sanity ATR used | S |
| A17 | Data integrity: 20 random 1m bars — **0 OHLC/vol violations, 0 duplicate timestamps**; only **2 phantom bars** stamped exactly 16:00:00 CT (one per halt day, vol 286/412 — forming bar at halt; harmless); completed session-day ranges 403.2 / 301.2 pts; ES rows exist in parallel (value-disjoint, not contamination — both symbols subscribed). Live chart parity was proven 2026-08-26 in the bar-persistence report (3-row spot-audit, OHLCV identical); NT8 chart access unavailable this session so the 20-bar audit is invariant-based. | `bars` table (SQL), `2026-08-26-bar-persistence.md` §8 | Drop the 16:00 phantom on write (guard open_time within session) | XS |
| A18 | Stale-MET quantified: of 2,908 MET confirms this week, **93% >5 pts from the decision-time price, 62% >40 pts, median 58.75 pts** — the "MET" state is frequently mechanically-true but context-dead (e.g. 08-20 S2 written ~400 pts from price). Per-day medians 150.6 (08-20) → 40.4 (08-25) → 67.0 (08-26). The confirm rule (1x/2x 5m close) itself matches the A2 acceptance research; the staleness handling is the gap → S2. | DB recompute over decision_records 08-20..08-26 | S2's re-unit + render stale on ~2×ATR5m | S |
| A19 | Expectancy recompute (38 closed MNQ, 08-20..08-26, Σ **−2160.0**): condition — acceptance n=5 **Σ−1587** (worst, matches research 0%/−157), reject n=14 Σ−103, sweep_reclaim n=2 +29, breakout_retest n=1 −66, unresolvable n=16 −433; session — **ASIA n=14 Σ−1823**, LONDON n=14 −328.5, NY n=10 −8.5; quality — **A n=9 Σ−1688** vs B n=7 **+265** vs C n=5 −242.5; adherence A=22/38 while losing — adherence-to-a-losing-plan. Attribution bugs: S3 + `plan_matched`=0 on 4 rows + plans.strategy_id=tid. | DB recompute (trader_positions ⋈ plans via date+session+trader+version) | S3; [OWNER] A-grade scenarios are the week's biggest losers — quality calibration cannot wait for narrative-only data | M |
| A20 | Synthesis — see §3. | | | |

### B — calibration doubt

- A2.1 ETH-anchored-only session VWAP (deliberate + documented — but no RTH twin). `levels_volume.go:29-40`
- A3.2 70% VA duplicated inline. A4.1 10-calendar-day nPOC lookback ≈ 7 sessions. A5 base≤6 + departure 1.5×ATR units. A6.4 1.5×ATR-body displacement too strict (fail-closed). A8.1 3-tick EQ tolerance (in band ✓). A13 Tier-1 3.00-pt gate weak; VWAP-band double-seat. A14 12-bar episode close at the short end of the published 10-30-bar resolution window; 5-bar approach window short. A15 leaves 15-30 min of RTH on the table by contract. A18 confirm acceptance rule matches A2 (no change).

### C — polish / verified-good

- A1 IB ±1.5×/±2× extensions correct but share Kind with base lines (can't be role/scored distinctly). A5 pattern classification uses one pre-base candle. A6 CE midpoint + width gating + iFVG inversion ✓. A7 last-opposite-candle fidelity ✓. A15 DST/tzdb ✓, halt/dead-hour untraded ✓. A16 all-Wilder ✓ (conformance holds). A17 OHLC/dupe sanity ✓. A2.2 σ formula is the real volume-weighted std ✓.

---

## 2. Per-agent one-paragraph summaries

**A1 (taxonomy).** Enumerated all 10 detector groups (`levels.go:17-55` types; multiday/intraday/zones/volume/HTF re-runs). The pro-menu gaps are real: weekly/monthly H/L detectors exist but are gated by bar counts the 33h ring can never satisfy (`levels_multiday.go:194-226`); no ETH/RTH profile split, no composite VA, no swing-anchored VWAP, no half-gap 50% rule, no multi-day ONH; SETT is a 105-min-late last-close approximation. IB extensions ✓ but share kinds.

**A2 (VWAP).** Session anchor is 17:00 CT ETH (deliberate); σ bands are a true volume-weighted std (±1σ only); eVWAP's 16:00 CT anchor makes it byte-identical in window to session VWAP — a wasted seat. No RTH 08:30 VWAP anywhere.

**A3 (profile).** 120 uniform bins + close-only volume assignment → POC ±1.25 pt smear, VAH/VAL ±2.5 pt quantization, close-clustering skew; the store path uses a different (finer) grid, so one session has two "POCs". 70% VA is duplicated inline. Session-vs-composite semantics are mixed (bias_ctx uses a ~1.4-day rolling composite).

**A4 (nPOC).** 10-calendar-day lookback ≈ 7 sessions; retire is a pure bracket test with no ±1-tick tolerance (spec drift); "weekly nPOC" is a relabel of >5-day-old daily POCs, not a weekly composite, and lacks HTF-seat eligibility. Touch-scan window (2000 bars) is shorter than the 30-session feed → stale nPOCs leak.

**A5 (S/D).** Base 1-6 candles, departure 1.5×ATR, boundaries at base wick extrema ✓; no width cap at detection and no overlap merge, with zones exempt from cluster collapse — duplicates and fat zones reach the seats and only get score-discounted.

**A6 (FVG).** The 3-candle relation and CE math are correct, but no session-boundary guard exists in either FVG path — halt/weekend-straddling triples become phantom gaps; the 2-pt floor admits 8-tick noise; displacement uses only the newest candle's body and the 1.5×ATR-body threshold is stricter than the published range-based definitions. iFVG inversion pass verified.

**A7 (OB).** The last-opposite-candle rule and 8-bar lookback are faithful; but bounds include wicks (ICT body convention), displacement legs double-emit, and mitigation/invalidation states are entirely absent — dead OBs persist with only score-side freshness decay.

**A8 (EQH/EQL + sweep).** 3-tick equality tolerance ✓; sweep shape (wick-through + close-back) ✓ but zero min penetration depth; BOS events never reprice the swing map, so post-BOS sweeps/CHoCH measure against stale extremes; swept levels are never demoted.

**A9 (freshness).** Two ladders: anchors 1.0/0.8/0.6/0.5 (matches the one published sample 0.78/0.55), zones 1.0/0.6/0.3/0.15 (steeper than any source, and the source was an OB study — a zone kind). "Anchors no-decay" is a comment, not code. PDH touched 3× → B.

**A10 (confluence).** Ten families (not five), hard same-family dedupe, cap 3 = ×1.6 matching the only research number (C14). No quality weighting — any in-band level adds flat +0.20.

**A11 (grading).** A≥1.0 / B≥0.70 are uncalibrated constants; 15m zones have a hard A-ceiling; OWNER kind maxes at B (0.96) contradicting its "grade A by design" comment; the full-audit's byte-for-byte claim predates Pack B. Worked sensitivity examples show single-evidence flips A→B→C.

**A12 (roles).** Five roles for 38 kinds is force-fitting: PDC=pivot is the weakest and most-traded (banned from breakout anchoring — the two live WARNs today), SETT and MID-O=pivot dubious, iFVG=react questionable; `breaker`/`trap`/`mitigation` missing. Roles do reach both prompts.

**A13 (seats).** With dATR ≈ 352 pts, the 1.5×dATR lock admits the whole map; VWAP±1σ can burn 2 seats; the Tier-1 gate is 3.00 pts (dispatch's 48-pt premise was wrong tick math); seated tables and per-hour in-band counts are unobservable from logs/DB today.

**A14 (telemetry).** Band 4 pts fixed-tick (misses 5-6-pt reactions; slices slow grinds); 12-bar close at the short end; the 20-bar volume baseline is dead code; "5m close" isn't exchange-aligned. Penetration/close-side math is sound.

**A15 (sessions).** Live flat = 14:30 CT vs the 14:45 contract (tests pin 14:30); lunch hard-gate overlaps the pm killzone 13:00-13:30; halt/dead-hour windows correctly untraded; DST via tzdb ✓.

**A16 (ATR census).** All ATRs are Wilder-14 (conformance holds). The real issues are semantic: the dATR duality (S1) and the TF-variable `atr15From` sanity bound.

**A17 (data).** 20 random bars pass all invariants; 2 phantom 16:00-stamped bars per halt day (harmless); ES rows are parallel-subscription, value-disjoint; completed-day ranges 301-403 pts measured.

**A18 (confirm + stale).** The 1x/2x-5m acceptance rule matches the settled A2 research. Staleness is the headline: 62% of MET confirms are >40 pts off context at decision time; the shipped (not yet deployed) stale annotation uses a ~350-pt unit and would mark only 38 of 2,908 (1.3%).

**A19 (expectancy).** Independently recomputed: acceptance −1587 (n=5, dominated by one ASIA −1458 disaster, pos 526) and A-quality −1688 (n=9) are the week's black holes; ASIA −1823; adherence A=22/38 while Σ=−2160. Attribution infrastructure itself is the poison: plans.strategy_id holds trader ids, version numbers are per (date, session) and leak across session handoffs, and ~37% of the week can't be joined.

**A20 (synthesis).** See §3.

---

## 3. A20 — SYNTHESIS + steelman

### 3.1 Steelman: arguing AGAINST the top 5 proposed fixes

1. **"Switch proximity lock to 5m ATR."** Against: the lock feeds seats, confluence bands, and the Tier-1 gate in lockstep; a fast-moving ATR5m makes the map flicker minute-to-minute (levels in/out mid-session), and 200+ goldens + prompt byte-stability regress. Counter: retune `proximity_filter_atr` on the EXISTING daily unit to 0.2-0.4 first (no code change, config-only) and measure seat stability before touching code.
2. **"Adopt published 0.78/0.55 zone freshness."** Against: weaker decay keeps stale zones graded high longer — the B2 Tier-1 gate and zoneSizeMult were tuned against the steep ladder, and the only source is a single OB study; weakening decay re-opens the stale-zone entry class the steep ladder was added to kill. Counter: settle it with B4 level_stats forward data (already scheduled) instead of a research-table swap.
3. **"Quality-weighted confluence."** Against: typeEvidence already encodes quality; weighting again double-counts and compounds A-grade clusters toward the ×1.6 cap, inflating grades exactly where A11 shows A-scenarios are already overrepresented and losing. Counter: if adopted, cap must drop to 2.
4. **"Width-cap S/D zones at detection."** Against: a hard drop deletes wide multi-hour accumulation bases that research treats as valid HTF zones; the existing zoneSizeMult already discounts them progressively. Counter: cap at generation only for 1m/15m TFs, keep 1h/4h.
5. **"BOS reprices the swing map."** Against: the structure engine feeds prompt text, structure_json (stored for future replays), and goldens; repricing changes bytes and event sequences mid-flight, with downstream MSS/CHoCH consumers unplanned for the new reference. Counter: ship repricing behind a flag after the Sep-3 replay substrate exists so the change is measurable.

### 3.2 Final top 10 by EV (knob/file mapped)

1. **S3 attribution repair** — persist date/session/plan_id on positions + fix `plans.strategy_id` semantics. Every Sep-3 ruling and REFUSAL AUTOPSY join depends on it. `store/position.go`, `store/plan_store.go`. `[OWNER]` backfill decision.
2. **S2 stale-confirm re-unit** — `STALE_CONFIRM_ATR` against 5m Wilder ATR (≈2.0×), before the addendum deploys. `kernel/plan_confirm.go`.
3. **S1 proximity re-tune** — `proximity_filter_atr` 1.5 → 0.2-0.4 (config-only first). `DayPlan.ProximityFilterATR`.
4. **S4 nPOC dedupe + full-window touch scan** — `levels_assemble.go:79-80`, `auto_trader_dayplan.go`.
5. **A2 eVWAP re-anchor (15:00 CT) or drop** — frees a seat, removes the degenerate row. `levels_volume.go:91-103`.
6. **A6 FVG session-boundary guard** — 3-line open-time delta check in both paths. `levels_zones.go`, `fvg_entry.go`.
7. **A3 profile binning fix** — reuse SVP grid + distribute H−L volume. `levels_volume.go:157-226`.
8. **A7 OB mitigation/invalidation** — retire on touch + kill on base-candle breach. `levels_zones.go:269-314`.
9. **A14 touch band ATR-scaling + dead-code fix** — `TOUCH_BAND_TICKS` exists; fix `volRatio` dead branch. `touch_telemetry.go`.
10. **A15 flat-time contract** — one number (14:30 or 14:45) across registry, FE, tests, comments. `[OWNER]`.

### 3.3 Owner-ruling items

- `[OWNER]` A15: 14:30 live flat vs 14:45 contract — which is the intended number?
- `[OWNER]` A19: A-quality scenarios are the week's biggest losers (−1688, n=9) while B is +265 — quality-grade calibration is a data question for Sep-3, but the A-grade overrepresentation is flagging now.
- `[OWNER]` A13: should volume anchors (VAH/VAL/SETT/nPOC) count as Tier-1 for the 12-tick pattern gate?
- `[OWNER]` A19: ASIA enabled → ASIA n=14 Σ−1823. Session-level verdict pending Sep-3 tables.
- `[OWNER]` S3 backfill of position→plan linkage before any ruling is computed.

### 3.4 Verified-good (so future waves don't re-check)

Wilder-14 ATR everywhere (conformance holds) · CE midpoint + 20-pt gating + iFVG inversion · EQH/EQL 3-tick tolerance · OB last-opposite-candle + 8-bar lookback · σ-band formula · anchor freshness ladder vs the one published sample · confluence cap 3 · bars-table OHLC/dupe sanity · DST/halt session math · touch penetration math.
