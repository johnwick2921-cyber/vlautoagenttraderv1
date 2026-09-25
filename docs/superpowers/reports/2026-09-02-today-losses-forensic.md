# TODAY'S LOSSES — FULL FORENSIC EXPORT · 2026-09-02

READ-ONLY. No writes, no restarts, no cancels, no lock. Raw exports in
`docs/superpowers/reports/exports/2026-09-02-losses/` (Section E URLs below).

## ⚠️ FIRST — anything dangerous right now?

One non-terminal arm exists: `armed_orders id=32 · NY S3 · state=armed (NOT
placed) · LONG limit 29070.00 · stop 29019.67 · updated 10:32:30 CT`. Current
price 29147.25 (11:27 CT bar) → the limit sits BELOW market on the correct
(unmarketable) side, stop 50.3pt below entry, 77pt below market. **Not
dangerous.** It stays armed until the price returns to 29070 or the session
flattens. No open positions.

## A7 — premise check

"Every long trade lost" is **TRUE today**: n=4, 4 long, 0 short, 4 losses,
Σ pnl_corrected = **−381.5**. (Rows 587/588/589/590. There were no shorts
today; no other positions.)

## B — trade by trade (ids, quotes)

### 587 — ASIA v7 S3 reject-LONG · −62.5
- entry 29079.25 (00:17:44 CT) → exit 29048.00 · pnl_corrected −62.5 · close=sync · source=system · MAE 33.0 / MFE 25.75 · adherence B.
- Plan `2026-09-01:ASIA` v7 (level_event): bias long/low, flip "5m close below 29035.25 flips short", death `29029.88 below 5m_close`.
- S3: "Price touches 29068.05 (VWAP−1σ) and rejects upward", confirm touch below 29068.05; arm entry 29068.05 stop 29044 tp 29131.66 → **planned R:R 2.65** — passes 2.0, refused under 3.0. Arm row 31 later `cancelled · no order_update within stale window`.
- Entry was a DECISION market long, not the arm: decision 00:17:45 CT `open_long, stop_loss 29048, take_profit 29113.25, confidence 62` — **decision R:R = 34/31.25 = 1.09**, below both 2.0 and 3.0 floors.
- Stop hit: exit == 29048.00 == decision stop exactly (stop-market at the broker).

### 588 — LONDON v3 S2 reclaim-LONG · −65.0
- entry 29082.50 (07:41:05 CT, PRE-0B) → exit 29050.00 · −65.0 · sync · system · MAE 33.25 / MFE 0.0 · adherence B.
- Plan `2026-09-02:LONDON` v3: bias long/low, flip "2x5m close below 29058.75 PDC", death 29001.75 below 5m_close.
- S2: "Undercut of 29058.75 PDC followed by a 1x5m close back above", confirm 1x5m_close above 29058.75. No arm row (decision entry).
- Decision 07:41 CT: `open_long, stop_loss 29050, take_profit 29165, confidence 63` — **R:R = (29165−29081.25)/(29081.25−29050) = 83.75/31.25 = 2.68**.
- Pre-0B gate evidence 07:30:14: attempt-1 parse failed `sl_too_tight: stop 29065.50 does not clear the cited level 29058.75 by ≥2 tick(s)` — the 2-tick level clearance was already live pre-0B.
- Stop hit: exit == 29050.00 == decision stop exactly. MFE 0.0 → filled into a falling tape.

### 589 — NY v3 S3 breakout_retest-LONG · −155.0 (today's big one)
- entry 29192.50 (09:41:04 CT, POST-0B) → exit 29115.00 · −155.0 · sync · system · MAE 80.5 / MFE 10.25 · adherence A.
- Plan `2026-09-02:NY` v3: bias long/low, flip "2x5m below 29001.75 RTH-L flips short; 5m_close below 28927.25 ONL voids the plan", death 28927.25 below 5m_close.
- S3: "Break above 29171.25 ONH with a 1x5m close, then a pullback retest holds 29171.25" (breakout_retest long). Entry 29192.5 is ~21pt ABOVE the 29171.25 retest level — the model chased above the authored level.
- MIN-SL gate fired 3× pre-entry (09:27:13 / 09:33:16 / 09:40:09): `🛑 MIN-SL REJECT MNQ open_long: sl_too_tight: 34.8 < 1.5×ATR (37.4) — widen or skip` — the 0B floor 1.5×ATR5m was LIVE and the executor kept re-asking until a stop cleared it.
- Stop hit: exit 29115.00 (decision stop, not the arm's) after MAE 80.5 pts — a wide 0B-style stop = a bigger loss when it finally hit.
- 09:41:05 also: `⏱ cycle overran the scan interval (2m35.116s > 2m0s)` — cycle pressure during the entry.

### 590 — NY v5 S4 · LONG −99.0 (cited a SHORT scenario)
- entry 29193.25 (10:37:17 CT) → exit 29143.75 · −99.0 · sync · system · MAE 49.75 / MFE 1.0 · adherence —.
- Plan `2026-09-02:NY` v5: bias long/low. **S4 is a SHORT scenario**: "First touch of 29317.25 RTH-H fails and 5m close prints back below", direction **short**, arm entry 29317.25 stop 29348 tp 29246.25.
- The position is LONG while its cited scenario is SHORT — the decision ran counter to the cited scenario. No arm row for S4 was ever placed (market never reached 29317.25; the stop-composition lines 09:21–09:41 show S4's short stop being WIDENED by 0B every cycle: `🛑 arm stop NY S4 leg 1 short: stop 29372.60 (authored 29355.00 WIDENED) · anchor OB(bull)·1h (HTF) 29324.50 → beyond 29325.00 · atr_floor 29372.60 (1.5×ATR5m 36.90) · bound=atr_floor`).
- Stop hit: exit 29143.75, MFE 1.0 — bought at 29193.25 into the fade, filled, never went green.

## C — aggregates

**C1 today:** n=4 · long 4/0 short · wins 0, losses 4 · Σ pnl_corrected **−381.5**
· per session: ASIA −62.5 (n=1), LONDON −65.0 (n=1), NY −254.0 (n=2) · conditions:
reject, reclaim, breakout_retest, S4-reject(mismatch) — one each. Wilson interval
for win-rate 0/4 is meaningless — n is tiny, stated plainly.

**C2 — R:R era (owner change 09-01 08:13 CT, 3.0→2.0):**

| era | n | wins | win% | avg pnl_corr | avg |entry−exit| |
|---|---|---|---|---|---|---|
| 3.0-era (7d, 08-25→09-01 08:12) | 21 | 10 | 48% | **+6.5** | 32.4 |
| 2.0-era (2d, since change) | 7 | 1 | 14% | **−58.1** | 44.2 |

Direct test of the owner's change: arm 31 (ASIA S3) carried **R:R 2.65 — passes
2.0, REFUSED under 3.0**; the position it authorized lost −62.5. The other three
losses were decision entries, where the executor's floor is different (see
verdict). n=7 vs n=21 — the era split is real but small.

**C3 — 0B (cutover ≈ 07:32 CT restart; floor 1.0→1.5×ATR5m):** pre-0B trade 588
stop 29050 (31.25pt) hit; post-0B 589 stop 29115 (77.5pt, widened past authored
levels by the atr_floor — `bound=atr_floor` lines) hit, 590 stop 29143.75 hit.
No trade today had a stop WIDENED that was then hit where the authored stop would
have survived — all three exits printed exactly at their (wider) decision stops;
the wider stops converted 589's loss from a smaller cut into −155.0 (MAE 80.5 >
pre-0B-style ~37pt stops would have been hit earlier). Post-0B stop-hit rate:
2/2; pre-0B: 1/1. All four stops hit today: **4/4**.

**C4 — bias vs tape:** every plan said long/low; tape: ASIA open 29137.25 → 02:00
29087 (−50pt), LONDON chop-down, NY 08:30 open 29098.25 → 09:41 29193.75 local
top → 11:27 29147.25 fade. The bias was wrong in ASIA/LONDON and only
transiently right in NY; entries 589/590 bought within ~1.5pt of the 09:41–10:37
local top zone. **Bias wrong (or entries chased the exhausted bounce): both.**

## D — the gap question (mechanism, quoted)

**D1** The level map is rebuilt per PLAN READ (session read / level_event wake /
death re-plan) — `runPlannerReadCoreWithFactsGrades` → new plan version with a
new `levels` block (E6 levels.csv = 254 rows across today's 23 versions). Between
reads the executor sees ONLY the last plan's seated levels; a level that forms
after the last read does not exist for the executor until the next read
(wake levels fire on level events — `auto_trader_wake_levels.go:279`).

**D2** `trader/armed_executor.go runArmedPlacement` (quoted): the wrong-side
guard `if price > 0 && limitMarketableWrongSide(price, r.EntryPx, r.Side)` →
`SetState(cancelled, "level accepted through — marketable, never placed")` covers
(a) a not-yet-placed arm whose level price already accepted through — it is
VOIDED, never placed (arm 25 on 09-01 shows this exact class). (b) A RESTING
limit beyond the gap: it simply never fills until price returns; `ARM_PLACE_TICKS`
band (`armedPlaceTicks()`, default 100) only bounds how far from market a fresh
arm may be placed. (c) An OPEN position whose stop sits inside a gap: the stop is
a broker-side OCO bracket leg (`ninjascript/VLTraderTCPClient.cs:229` "places OCO
bracket orders", SL/TP keyed by signal_id :68/:75) — on a gap through it, NT8 SIM
fills the stop-market at the gap's first print and the ledger records the sync
close (today: all four `close_reason=sync`).

**D3** Stop type: NT8 bracket SL leg (stop-market semantics; no stop-limit in the
wire — `modify_bracket`/`move_stop` only). Gap-through fill = next print after
the stop price, recorded via close-sync frames (`close_sync.go`) as `sync`.

**D4** Gap scan today (1m, |open−prev_close| ≥ 6pt): **zero** gap events all day
— no level was jumped by a bar open today.

**D5** NY session-open gap 08:29→08:30: 08:29 close 29098.25 → 08:30 open
29098.25 — **no gap**; the first 08:30 bar then sold to 29065.0. No arm or stop
sat inside an open gap today.

## E — raw export (one URL per file, all 200-checked at commit)

- E1 trades.csv · E2 arms.csv · E3 plans.jsonl · E4 bars_1m.csv · E5 5m/15m/1h/
  daily: **ABSENT from the store — only 1m is persisted** (surprise; E4 covers
  30 days) · E6 levels.csv · E7 decisions.csv · E8 journal_today.txt · E9
  config_at_trade.csv (the class-44 `config_changes` table does NOT exist in the
  DB — values reconstructed from boot/arm/journal lines; see the notes file).

## F — VERDICT

Four longs, four stop-outs, −381.5 corrected (n=4 — a normal-size losing day is
unprovable either way at this n). Evidence-split: (1) **Bias/entries [T]:** all
plans long into a tape that fell; NY entries bought the 09:41–10:37 top zone and
the S4 position traded LONG against a SHORT scenario — that is a
plan-adherence DEFECT-class event (590), not a market event. (2) **R:R change
[T]:** arm 31 (R:R 2.65) is the one arm the 2.0 floor admitted that 3.0 would
have refused; it lost −62.5. The other three were decision entries whose
decision-level R:R (e.g. 587 = 1.09) sat below BOTH floors — the floor never
applied to those, which is a floor-coverage question, not evidence the 2.0 choice
itself is wrong. (3) **0B stops [T]:** the wider post-0B stops did not save any
trade — 4/4 stops hit; 589's 77.5pt stop converted a small loss into the day's
largest (−155.0). No stop was widened-and-saved. (4) **Gaps [T]:** zero gap
events today — gap mechanics explain nothing about today's losses. Net: today
was a wrong-bias, chased-entry losing day at n=4; the concrete DEFECTS on the
tape are the S4 long-against-short entry and decision entries bypassing the
risk-reward floor, not the owner's 3.0→2.0 change per se — though that change
did admit the one arm that lost.
