# Shift-Day Loss Forensics — 2026-08-21 (trader hoang, all sessions)

Read-mostly dispatch: research import + read-all + today's regime-shift loss
forensics. **No code changed.** PnL cites `pnl_corrected` (all rows today:
corrections absent, corrected == realized).

## 1 · Import manifest

| # | Source (Windows Downloads) | Repo path | Format |
|---|---|---|---|
| 1 | `VL_Trading_System_Final_Build_Plan_v5 (1).docx` | `docs/research/plan-card/VL_Trading_System_Final_Build_Plan_v5-(1).{docx,md}` | original + md conversion |
| 2 | `VL_Trading_System_Implementation_Plan.docx` | `docs/research/plan-card/VL_Trading_System_Implementation_Plan.{docx,md}` | original + md |
| 3 | `VL-DAYPLAN-FULL-SPEC (2).md` | `docs/research/plan-card/VL-DAYPLAN-FULL-SPEC-(2).md` | markdown |
| 4 | `Strategy-Studio-Complete-Plan.docx` | `docs/research/plan-card/Strategy-Studio-Complete-Plan.{docx,md}` | original + md |
| 5 | `PLAN-CARD-DESIGN-SYSTEM (1).md` | `docs/research/plan-card/PLAN-CARD-DESIGN-SYSTEM-(1).md` | markdown |
| 6 | `VL_Trading_System_Build_Plan_v3.docx` | `docs/research/plan-card/VL_Trading_System_Build_Plan_v3.{docx,md}` | original + md |
| 7 | `plan-config-mockup (1).html` | `docs/research/plan-card/plan-config-mockup-(1).html` | html |

All 7 found at the specified path; none missing. Commit:
`docs: import plan-card design research (7)`.

## 2 · Per-file digest — built / never-built / deviated

| File | What it specifies | BUILT today | SPEC'D, NEVER BUILT | Built DIFFERENTLY |
|---|---|---|---|---|
| **VL-DAYPLAN-FULL-SPEC (2)** (v1 FINAL 08-14 — the live contract) | bias + explicit flip condition on the card; scenario grammar {trigger, condition∈reclaim/hold/sweep_reclaim/reject/acceptance/breakout_retest, direction, target_chain, invalid, quality}; plan-dies-if line; re-plan cap; level detectors incl. S/D zones + FVG/OB; Go grading; proximity 1.5×ATR; activation window; re-arm rules (freshness A→B→C, 20m cooldown, consumed-on-acceptance 2×5m); plan_mode advisory→direction→strict; auto regime block | plan JSON, scenarios+confirm{}, deaths/flips (re-plan on flip/death), level pipeline, grading, proximity, activation window, consumed state, digest | matched-random stats + blind-mark calibration; adherence grade A–F is BUILT; Ask-Planner anti-sycophancy verdicts BUILT | flip is treated as a plan-death → re-plan (spec lists flip on the card but does not prescribe death semantics) — see §4 |
| **PLAN-CARD-DESIGN-SYSTEM (1)** | ZoneRow `price\|range:[proximal,distal]`; one-array contract (card/chart/prompt render the same levels); RFC6902 overlays; conflict chip | zone rows render (levels carry lo/hi); overlays, edit sheet, bulk add, conflict chip | `--vl-*` token system (UI still `nofx-*` tokens — brand census) | instruction verbs exist; "magnet — no entries" / "no touch" verbs not enforced (advisory only) |
| **Strategy-Studio-Complete-Plan** | Studio truth/wiring: risk boxes editable-but-ignored fix, prompt-preview honesty | the risk-config truth fixes shipped (F1, minconf 60, etc.) | — | crypto/futures label work partially reverted by later design |
| **Implementation Plan** (May 2026, Nautilus era — superseded) | smc.fvg/ob/bos_choch/liquidity feature layer; 8-factor bias ±25; **bias hysteresis ("Locked until 10:30")**; daily bias gate LONG_ONLY/SHORT_ONLY/BOTH/STAND_ASIDE; kill switch; consecutive-loss cooldown; news blackout ±5m | kill switch equivalent (stopUntil pause), consecutive-loss halt (D1), calendar blackouts (T1), feed-loss halts | **BOS/CHoCH/sweep detectors** (never built — see 2.3 census) · **hysteresis lock on bias** (never built — flip is instantaneous) · STAND_ASIDE bias label (no equivalent) | engine is Go/TCP, not Nautilus/CSV |
| **Build Plan v3** (May, superseded) | ICT setups + per-setup stop rules; backtest phase gates | — (this architecture was abandoned) | all ICT setup detectors as first-class triggers | day-plan scenario conditions are a superset of these setups |
| **Final Build Plan v5** (Qlib/RD-Agent era, superseded) | 35-model ensemble; **hysteresis enter 0.30 / exit 0.15**; **HTF veto: "1d signal fresher than 24h AND opposes the entry → veto"**; 1d+1w opposing → half size; 5m algo disagreement → skip; stuck-signal circuit breakers | ensemble → the Go level/scenario engine replaced it | **HTF veto** · signal hysteresis · circuit breakers | daily-bias concept now the day-plan bias (per-session) |
| **plan-config-mockup (1).html** | Day Plan config block UI; edit sheet (PRICE/ZONE RANGE, type D-zone/S-zone/Level/Liquidity/My level, instruction verbs, grade, note→AI); planner has NO indicator toggles — fixed 7-field regime line | config block + edit sheet ship in Studio (DayPlanEditor, session accordion, InheritOverrideChip) | fixed 7-field regime line → the Go regime block computes it (closer to built than not) | — |

## 3 · REGIME EXTRACT — what the research prescribes for a shift day

**(a) Flip conditions are FIRST-CLASS, not decorations.** DAYPLAN spec §Card-1:
"**Bias** — direction + conviction + explicit flip condition ("flips short on
2×5m < 30148")". Today's NY plan carried exactly this
("15m close above 29470.25 RTH-H flips session bias to long") — the spec's own
shape.

**(b) Structure breaks (BOS/CHoCH) and sweeps are reversal triggers.**
Implementation Plan §2.2 / v3: `bos_choch` — "BOS (1=bull break,-1=bear break),
CHoCH (character change)"; v5 §11.1: "**Break of Structure (BOS)** … HTF bias
change · **Change of Character (CHoCH)** … Reversal confirmation · **Liquidity
Sweep** … Reversal setup trigger".

**(c) The HTF trend vetoes counter-trend entries.** v5 Appendix K.3:
"**If 1d signal is fresher than 24 hours old AND opposes the entry → veto the
entry.**" K.3 also: "If 1d AND 1w both have |signal| > 0.5 in OPPOSITE
direction of 1m/5m signal → reduce position size by 50%".

**(d) Bias flips are damped, not instant.** v5 K.1 hysteresis: "Enter long if
signal > +0.30 · Exit long if signal < +0.15"; Implementation §4.1: bias
"**Hysteresis: locked until 10:30**".

**(e) Stand-down during transition.** v5 K.5 circuit breakers + Implementation
§6.1: "Consecutive losses N → pause 30 min"; TradingAgents bias label
**STAND_ASIDE** ("stand aside" is a named bias state). DAYPLAN spec: plan dies
→ "auto re-read (max 2/day → then no-trade rest of day)".

**(f) Zones with proximal/distal; acceptance consumes levels.** DAYPLAN spec
§Levels: S/D zones "base ≤6 candles body ≤0.5×ATR + departure ≥1.5–2×ATR;
proximal/distal"; "**permanently consumed once price ACCEPTS through the level
(2×5m) — the level flipped roles**". This is exactly what the live PLAN STATUS
line printed today ("flipped(consumed — tradeable both directions)").

## 4 · Today's trade table + plan lifecycle (all CT, Sim101, MNQ)

Session windows (registry): ASIA 17:00→02:00 · LONDON 02:00→08:30 · NY
08:30→14:45. Effective cutoffs (offset 15): ASIA 01:45 · LONDON 08:15 · NY
14:30.

| pos | session | entry CT | dir | entry | exit | exit CT | pnl_corrected | cited | plan | confirm{} at entry (stored prompt) | structure context at entry | watcher trail | exit path |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 533 | ASIA | 08-20 17:59 | S | 29322.25 | 29349.50 | 19:12 | −54.50 | S2 | ASIA v1 | — (plan alive; S2 accept-below met) | price below flip 29338.50 | intact→intact (conf 78–96, 0 WARN) | sync SL |
| 534 | ASIA | 08-20 19:22 | S | 29332.25 | 29365.25 | 20:49 | −66.00 | S2 | ASIA v1/v2 | — | plan alive | intact→weakening | sync SL |
| 535 | ASIA | 08-20 21:08 | S | 29360.75 | 29376.25 | 21:20 | −31.00 | S1 | ASIA v2 | — | plan alive | intact→weakening | sync SL |
| 536 | ASIA | 08-20 21:36 | S | 29393.00 | 29331.00 | 23:27 | **+124.00** | S1 | ASIA v3 | — | plan alive (v3 flip 29485) | intact | sync TP |
| 537 | ASIA | 08-21 00:25 | S | 29360.25 | 29400.00 | 01:35 | −79.50 | S2 | ASIA v3 | — | plan alive | intact→weakening | sync SL |
| 538 | ASIA | 08-21 01:40 | S | 29388.25 | 29373.00 | 01:45 | **+30.50** | S1 | ASIA v3 | — | plan alive | intact | sync TP |
| 539 | LONDON | 08-21 03:54 | S | 29453.00 | 29495.25 | 05:00 | −84.50 | **off-plan** | LDN v2 | S1 MET · S2 NOT MET · S3 MET; cited off-plan (decision 31255) | **RTH-H 29470.25 dist −5.2, valid** — shorted 5 pts under the flip line, clock-drift WARN 156s same minute | intact→weakening, **1 WARN** | sync SL (through RTH-H) |
| 540 | LONDON | 08-21 05:04 | **L** | 29509.00 | 29460.00 | 06:01 | −98.00 | S1 | LDN v3 | plan v3 (flip product, bias long) | entered near ONH 29539.75 zone top; clock-drift WARN 190s same minute | intact→weakening | sync SL |
| 541 | LONDON | 08-21 06:26 | **L** | 29511.75 | 29470.25 | 08:13 | −83.00 | S1 | LDN v3 | plan v3 | stopped exactly at RTH-H as 2×5m closed below | intact→weakening | sync SL |
| 542 | NY | 08-21 08:49 | S | 29310.00 | 29310.00 | 09:27 | 0.00 | S2 | NY v1 | S2 MET | plan alive (post-dump) | intact | sync BE |
| 543 | NY | 08-21 10:12 | S | 29355.75 | 29400.00 | 10:34 | −88.50 | S2 | NY v1 | **S2 MET** ("last 2x5m close 29350.75"); all plan levels CONSUMED·flipped | dist −120 from RTH-H; bounce off 29220 bottom | intact→weakening | sync SL |
| 544 | NY | 08-21 10:47 | S | 29445.00 | 29475.75 | 10:51 | −61.50 | S1 | NY v1 | **S1 MET** ("last 1x5m close 29451.75"); RTH-H **dist −18.5** | **10:30 15m bar high 29471.75, 10:45 high 29488.50 — the upside break, UNCONFIRMED (15m closes 29451.75/29443.75)** | intact→intact (only 3 rows) | sync SL |

### Plan lifecycle timeline (2026-08-20/21)

| CT time | Event |
|---|---|
| 17:04 | ASIA v1 born — short/trend, flip 2×5m>29338.50 OR-L, death 15m>29345.25 ONH |
| 19:19 | ASIA v2 — flip 15m>29375.75 PDL (plan died via flip) |
| 21:33 | ASIA v3 — flip 15m>29485 PDL |
| 02:03 | LONDON v1 — short/balance, flip 2×5m>29402.25, death 2×5m>29470.25 RTH-H |
| 02:17 | LONDON v2 — flip 15m>29470.25 RTH-H (the line the whole day pivots on) |
| **03:54** | pos 539 enters short 5.2 pts under RTH-H (off-plan); **clock-drift WARN 156s** (no entry block by design) |
| 04:49 | **LONDON v3 — bias LONG** (the flip machinery DID flip; plan died on the 15m>29470.25 flip condition and re-planned long) |
| 05:04 / 06:26 | longs 540/541 enter near the top (ONH zone) |
| 08:20:01 | **LONDON v3 DIED** — journal: "flip-condition: 2x5m close below 29470.25 (2× 5m closes) → bias short. Re-planning (cap 4/session)" |
| 08:22 | LONDON v4 — neutral/balance |
| 08:29 | **NY v1 — short/trend-down**; flip 15m>29470.25 RTH-H; death 15m>29539.75 ONH |
| 08:49→10:51 | NY shorts 542 (BE) · 543 (−88.50) · 544 (−61.50) |
| 10:45 | 15m bar high **29488.50**, close 29443.75 — upside break NOT confirmed by close |
| 11:00 | 15m close 29394.25 — rejected; **the flip never fired in NY** |
| 11:00→14:45 | all decisions `wait` — bot flat, plan still "active", no NY v2 (death ONH 29539.75 never reached; max 15m close all day 29530.75 @ 06:30 London) |

### Answers to the dispatch's lifecycle questions
- **Did the machine flip fire?** Yes — at 04:49 CT (LONDON v2→v3, short→long) via
  its flip-condition. And the reverse at 08:20 (v3→v4). In NY the flip line was
  never touched by a 15m close → correctly never fired.
- **Entries after the market was above 29470.25 but before a 15m flip confirmed:**
  zero entries while price was *above* the line; **two entries (539, 544) inside
  the forming-break window** — 539 5.2 pts under the line with the 15m close
  pending, 544 with the 10:30/10:45 bars breaking the line intrabar (highs
  29471.75/29488.50) but closing below.
- **Plan death at ONH:** never fired (no 15m close > 29539.75 all day; the
  06:30 bar's high = 29539.75 exactly). After the 10:51 stop the bot simply
  stopped proposing (scenarios stale/consumed) and waited out the session.
- **Watcher (ai_watch, 332 watch cycles today):** every losing position was
  watched "intact → … → weakening" — the observer never reached invalidation on
  ANY of the 10 losers, and fired its R3 WARN exactly once (539). The watcher's
  thesis was anchored to the plan's own short bias and missed every stop-out.
- **Feed/clock integrity during the transition:** `kernel/clock_drift.go` WARNs
  at 03:54:09 (156s behind) and 05:04:39 (190s) — the C2 guard is alert-only
  for entries ("signals are feed-stamped so entries proceed"). The stored
  prompts of the 03:54/04:49/05:04 entries show multi-TF tables lagging well
  behind the session clock (e.g. 31255's 15m table latest close 01:30 at 03:54)
  — TF-dependent bar staleness during London. Whether B4's stale-block covers
  this path is a verification item (flagged, not claimed).

## 5 · Loss classification (pnl_corrected)

| Class | Positions | Count | Σ PnL |
|---|---|---|---|
| **(a) FAITHFUL-BUT-WRONG** — followed a live plan correctly; the plan's direction lost (grind-up overnight vs short plans; the flip's long plan bought the top) | 533, 534, 535, 537, 540, 541, 542, 543 | 8 | **−500.50** |
| **(b) TRANSITION-ENTRY** — entered the unconfirmed-shift window (the break-trap: shorting into an upside break before the 15m flip confirmed) | 539, 544 | 2 | **−146.00** |
| **(c) POST-FLIP-WRONG-WAY** — wrong way AFTER a flip/death should have blocked it | — | **0** | **0** |
| **(d) OTHER** — winners / neutral | 536 (+124), 538 (+30.50) | 2 | **+154.50** |

**Day total: 12 closed · −492.00 corrected · 2W/1BE/9L.** Session split:
ASIA −27.50 · LONDON −314.50 · NY −150.00. No class-(c) event — the flip/death
machinery never entered the wrong way after a confirmed flip; its failures were
slowness (45-min flip→replan latency vs the move) and no stand-down during the
transition window.

## 6 · GAP TABLE — research prescription ↔ what happened ↔ missing piece

| # | Research prescription (quote) | Today's reality | Missing component | Size |
|---|---|---|---|---|
| G1 | "If 1d signal is fresher than 24h AND opposes the entry → **veto the entry**" (v5 K.3) | 540/541 long INTO a down-trending daily tape (day_type trend-down per NY plan); 543/544 short into an intraday upside break with 1d context up (08-20 1d close 29451.75, price 29530s in London) | **HTF-veto gate**: 1d/4h structure disagreement blocks or halves entries | **M** |
| G2 | BOS = "HTF bias change", CHoCH = "Reversal confirmation", sweep = "Reversal setup trigger" (v5 §11.1) | No BOS/CHoCH/sweep detector exists (census §7); the only structure signal the executor saw was the level table | **BOS/CHoCH/sweep detectors** feeding the plan status / entry gates | **M** |
| G3 | Bias hysteresis "locked until 10:30" (Impl §4.1); enter>0.30 / exit<0.15 (v5 K.1) | Flips are single-condition instantaneous (15m close) with zero hysteresis; LONDON v2→v3→v4 whipped 3 times in 6 hours | **Flip hysteresis / re-flip cooldown** (e.g., one flip per session or N-bar lock) | **S** |
| G4 | "STAND_ASIDE" bias label (Impl §2.5); "plan dies → … then no-trade rest of day" (DAYPLAN spec) | After NY plan's scenarios were all consumed the bot produced 3.5h of `wait`s — but only because the model chose to wait; nothing stood the plan down, and it kept paying for wait cycles | **Transition stand-down**: when bias-line distance < k×ATR and flip pending, or after N consumed levels, suppress entries + downgrade plan to stand-aside | **M** |
| G5 | Zones "proximal/distal" + "permanently consumed once price ACCEPTS through (2×5m)" (DAYPLAN spec) | Levels already carry lo/hi + consumed state and the prompt renders "flipped(consumed — tradeable both directions)" — **the machinery exists**; but S1/S2 scenarios still fired off consumed levels (543: all 8 levels consumed; 544: RTH-H consumed) | Policy: scenarios on CONSUMED levels require the flipped-direction setup (acceptance/retest) instead of the original-direction reject | **S** |
| G6 | "Consecutive losses N → pause 30 min" (Impl §6.1) | 9 losses across three sessions with no inter-loss stand-down (B7 cooldown exists but is OFF, `reentry_cooldown_minutes=0`) | Arm B7 or a per-session loss-streak pause | **S** |
| G7 | Hysteresis + HTF veto were the research's flip-adjacent protections — but the live flip evaluator's data path lagged | 15m table at the 03:54 entry had its latest close 2h+ old (31255); clock-drift WARNs active; entries proceeded (C2 is alert-only) | Verify B4 stale-block coverage for 15m; harden flip/death evaluator against TF staleness (evaluate only on fresh bars) | **S–M** |
| G8 | Watcher (final-bundle) — thesis-anchored observer with invalidation rails | 332 watch cycles, 0 invalidations on 10 losers, 1 WARN | Watcher prompt lacks structure-relative invalidation hooks (BOS/CHoCH/sweep/HTF veto facts) — feed it G2's primitives | **M** |

## 7 · Recommended build order (grounded ONLY in research + today's evidence)

1. **G7 (verify B4 + flip-evaluator freshness)** — today's flip machinery was
   decisive but ran on a lagging 15m feed during the transition; this is a
   correctness check, smallest first.
2. **G1 + G2 (HTF veto + structure detectors)** — v5's HTF veto is the single
   prescription that would have blocked both transition shorts and the top-
   buying longs. BOS/CHoCH/sweep primitives are the research's reversal
   vocabulary and feed G8 too.
3. **G4 (transition stand-down)** — the DAYPLAN spec's own no-trade posture for
   a dying plan, keyed to flip-line proximity + consumed-level count.
4. **G3 (flip hysteresis)** — cheap, kills the v2→v3→v4 whipsaw.
5. **G5 + G6 (consumed-level scenario policy + loss-streak pause)** — policy
   tightening on machinery that already exists.
6. **G8 (watcher structure hooks)** — depends on G2's primitives.

Sizing: 2×S + 3×M + 1×S–M. No invented features — every item above is a
direct quote from the imported research mapped to a proven failure today.

---
*Read-only forensics — 2026-08-21 · positions/plans/decisions cited from
data/data.db (readonly) + journald + stored prompts · pnl_corrected throughout.*
