# MASSIVE-MOVE AUDIT — the −204pt ASIA sell-off, minute by minute

- **Date:** 2026-08-30 (Sunday) · ASIA session · MNQ SIM (Sim101)
- **Disposition:** read-only · live journal + DB `mode=ro` · NO fixes, NO restarts
- **Deployed binary:** `cd1d1de39a03` (entry-mechanics wave, cutover #2, boot 18:13:46 CT, PID 746955)
- **Verdict line:** the machine ran **clean under fire** (0 panics, 0 drops, 0 watchdog alarms, E8 shadow logger writing) but **captured $0 of a 204pt move its own plan predicted** — two long-side events lost **−$59.00** against a short tape.

---

## M1 · TAPE TIMELINE

| Leg | Window (CT) | Move | Notes |
|---|---|---|---|
| Open spike | 17:00–17:01 | 29535 → **high 29543.50** | weekly-doc invalidation breach lives here |
| L1 | 17:03–17:04 | 29502 → 29463 (−39pt) | fastest open minutes: 1m ranges **47.5 / 42.5 / 38.25 / 35.75pt** (17:00–17:03) |
| L2 | 17:34–17:36 | 29473 → 29434 (−39pt) | S4 ghost arm (v1) fills 29357… **#571 long 29437** |
| L3 | 18:10–18:14 | 29434 → 29407 (−27pt) | #571 stop 29414.75 hit 18:13:13 (old binary, 33s before the swap) |
| **L4 (the drop)** | **18:22–18:23** | **29411 → 29371.50** (−40pt in 2 min) | 18:22 range 29.5pt, body −22.0; ONL swept; v2 born 18:35 |
| L5 | 18:30 | 29372 → 29347.75 (−24.5pt) | S1 confirm bar: 5m close 29360.25 below ONL 29371.50 |
| L6 | 18:40–18:41 | 29358 → 29321 (−37pt) | S2 long entry+stop round-trip (below) |
| L7 | 18:45–18:47 | 29355 → 29303 (−52pt) | |
| **L8 (the flush)** | **18:55–18:56** | 29324.75 → **low 29291.25** (−33.5pt) | 18:55 body −28.0, range 29.25 |
| Rebound | 18:56–19:08 | 29291 → ~29310 | closes back across levels → breakdown-void rejections (M3) |

**Move totals:** session high 29543.50 (17:00) → low 29291.25 (18:56) = **−252.25pt** (0.75×dATR ≈ 338pt implied by the v2 doc's "0.52×dATR flushed" at 18:35). From the 17:20 local high 29495 → low = **−203.75pt** in ~96 min (≈2.1pt/min avg).

**Speed:** peak 1m range **29.5pt** (18:22) · peak 1m down-body **−28.0** (18:55) · peak 2-min slide **−40pt** (18:22–18:23).

**FAST_MARKET_ATR (1.5×ATR5m):** closed-bar ATR5m(Wilder14) = **23.12pt** at 18:45 → threshold **34.68pt**. No single 1m bar crossed 1.5×ATR5m (peak 29.5). The drift seam fired once, late but correctly:
> 19:07:45 `🧠 planner mode: fast-market (drift 94.5 pts = 3.5×ATR5m) — reasoning downgraded to fast→low for this read (F3)`

The drift check runs once per READ (not per retry), so the 18:45/18:51 "planner call" lines were retries of the 18:35:46 wake read whose drift check ran at read start (drift 13pt → no fire). No discrepancy — the seam worked as designed; it just can't fire mid-read.

---

## M2 · MONEY TRUTH FIRST

### S2 long (v2, sweep_reclaim arm) — FILLED, stopped out, −$14.50, ledger gap
| Time (CT) | Event |
|---|---|
| 18:37:46 | `⚔️ arm S2 leg 1 wait_confirm MET (touch) — arming` → **REFUSED: stop 29350.00 too close (21.50 < 21.82 = 1.0×ATR5m)** |
| 18:40:01 | ATR5m dipped back → **armed → 📌 WORKING limit 29371.50 signal=3ad69de4** — same second the scenario display said `≈invalidated (price accepted through the level … it flipped roles)` |
| 18:40:01 | `tcp_server: received frame type=fill` — marketable buy limit fills at **29357.25** (NT8 truth) |
| 18:40:05 | `🧩 reconcile: NT8 holds UNTRACKED position MNQ LONG @ avg 29357.25 — materializing after 60s if it persists` |
| 18:40:49 | stop hit: `priced close DROPPED — no matching open row … side=LONG exit=29350.00. Parked for reconcile fallback` |
| 18:41:46 | `✕ armed cancel (gate changed min_sl): ASIA S2` — the gate that allowed placement 105s earlier now fails it |

**Result: entry 29357.25 → stop 29350.00 = −7.25pt = −$14.50.** Equity 52069.00 (18:35:46) → 52054.50 (18:41:46) — the only event in that window. **The round-trip is absent from the Go ledger**: no position row, no fill rows, parked close never consumed (position never materialized — it lived ~48s, under the 60s reconcile grace, and was flat before any snapshot saw it). **Ledger-vs-NT8 divergence: −$14.50.**

Mechanical flaw (Phase 2 gap, not a law failure): the arm placed a long whose **stop 29350 was already below the market (29344–29357)** at placement — an instant-stop bracket. The entry-side "tick-managed placement is Phase 2" seam places plain limits with no wrong-side sanity check, and the min_sl gate re-evaluated with live ATR flops within minutes (refuse 18:37 → allow 18:40 → cancel 18:41) because the arm's 21.5pt stop sits exactly on the 1.0×ATR5m oscillation.

### S3 leg-1 short limit 29414.21 (v2, reject fade arm) — never placed, $0
- 18:35:46 `⚔️ armed ASIA S3 leg 1 short limit 29414.21 SL 29440.00 TP 29357.25` — **armed row only; no WORKING line, ever.** Entry was above the market (29367) at arm time and price only fell — the executor does not place a marketable-wrong-side limit, so it sat armed.
- 18:47:45 `⚔️ arm REFUSED ASIA S3 leg 1: stop 29440.00 too close (25.79 < 25.84 = 1.0×ATR5m)`.
- Leg-2 / sibling logic: the v2 arm is **single-leg** (`leg_index=0, leg_count=0` — the split-arm debut did not actually author tonight). The 1m-MSS rule was shadow-evaluated (`ab_confirm_log`: S2/S3 `1m_mss` @ 18:50:01) but no leg-2 existed to consume it.

**The 200pt tailwind: 0 captured.** The plan was short-biased with a valid S1 breakdown_continue short and a resting S3 short fade — the machine held zero shorts for the entire move.

### S4 long 29437 (v1, old binary) — the ghost arm, −$44.50
- Old-binary cancel 17:36:53 (`gate changed: min_sl`) raced the already-working limit → entry filled **#571 LONG 29437 @ ~17:38:33**, never in Go (reconcile-materialized later, `source=reconcile`), stop 29414.75 hit **18:13:13** → **−$44.50** (22.25pt). Fully pre-swap behavior; it's the v1-era arm-evaluation artifact.

### AI decisions during the move — 13/13 wait, zero proposals
| Decision | Time | Reasoning (excerpt) |
|---|---|---|
| 34786 | 18:37:23 | "S1 short not confirmed: 18:30 bar closed below ONL (29360) but break-candle body …" |
| 34788 | 18:40:30 | "S1 breakdown already confirmed and played out: price closed below ONL 29371.50 …" |
| 34789 | 18:42:35 | "S1 breakdown is confirmed (machine: MET), but price has already run to 29341.50" |
| 34793 | 18:50:54 | "Price at 29310.50 has reached the S1 target 29310.00 after heavy water…" |
| 34796 | 18:56:10 | "Price waterfalled to 29302.50, ~70 pts below S1's entry zone" |
| 34802 | 19:08:18 | "Price 29294.50 just swept ONL 29273.50 but no 5m close below it" |

The AI narrated the entire S1 short chain (29371.50 → 29357.25 → 29335.50 → 29310, **all three targets hit by 18:50**) from the sidelines. First post-plan cycle judged the break-candle body too small ("not confirmed"); two cycles later it was "already played out" → no-chase discipline. **Zero C6 refusals** — the AI never proposed an open.

### Net session P&L + exposure NOW
- **Ledger:** #571 −$44.50 (only tonight's row) · **NT8 truth:** −$44.50 (S4) − $14.50 (S2) = **−$59.00** session.
- Equity 52054.50 USDT · positions **0 open** (flat) · NT8 snapshots count=0 ×both accounts. No exposure.
- Ledger divergence: the S2 −$14.50 is on NT8's equity but in no table (parked close, unmaterialized round-trip).

---

## M3 · PLAN LIFECYCLE

### v2 (active, born 18:35:34 via owner reset) — flip NEVER breached; law held
> `Bias: short (low) · flips: 5m close above EQL·4h 29396.75 flips bias to long`
> `death_condition: Two consecutive 5m closes above 29420.00 (EQH·1h)`

The move went DOWN: price never closed above 29396.75 (or 29420) after 18:35 — **no flip, no death, v2 stayed active and short the whole way.** Correct.

### The flip that DID fire was v1's (old binary, 17:40:53) — flip-hygiene latency evidence
> 17:40:53 `😴 plan 2026-08-30 ASIA v1 DORMANT — flip-condition: 2x5m close below 29494.38 (buffer 0.5×ATR14, 7× 5m closes) → bias short`

- Level 29494.38; first 5m close below it: **17:20 (close 29489.50)**.
- The flip required **7× 5m closes** + a **30-min plan-age guard** (`flip_eval_skipped … plan age 964s < 30min` at 17:25) → fired **17:40:53 at ~29449.50**.
- **Flip latency ≈ 40pt / ~21 min** after the first close breach — the flip (which was RIGHT) locked in the short bias after the first two legs of the crash.
- The flip's correctness: v1 was born long-biased 17:08:56; the dormant short-bias flip at 17:40:53 preceded another −158pt of downside. Evidence for the flip-hygiene wave: 7 closes + age guard is slow; the re-plan it should have triggered never ran before the swap, leaving a dormant chain until the owner's reset at 18:26:50.

### Replans: three reads, six validator rejections, v3 never written — THE LAW'S FIRST LIVE TESTS
**Wake read (started 18:35:46, W6 5th wake):**
1. attempt 1 (18:45:26, 580.0s): `📐 planner attempt 1/3 parse/schema rejected: scenario[0].confirm.rule "1x5m_close" — sweep_leg1_requires_touch (the sweep leg of a two-leg sweep_reclaim enters on the touch at the sweep ref)` — **E2 split-sweep law, first live fire.**
2. attempt 2 (18:51:30, 364.2s): `S1 breakdown_continue: a close came back across 29347.75 — the breakdown is void; author a reject/retest play instead` — **breakdown-void law.**
3. attempt 3 (18:58:57, 447.8s): `S3 breakdown_continue: a close came back across 29347.75 — the breakdown is void` → `wake re-read failed (benign — active plan kept)`.

**Fast-market read (19:07:45, drift 94.5pt = 3.5×ATR5m, reasoning downgraded to fast→low):**
1. attempt 1 (19:12:42, 296.3s): `S1 breakdown_continue: a close came back across 29292.25 — the breakdown is void`.
2. attempt 2 (19:15:17, 155.1s): `S1 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (27.2 pts) — not a displacement move, author a normal reject/retest play instead` — **E3 BD_MIN_DISP_ATR displacement gate, first live fire.**
3. attempt 3 (19:18:45, 208.6s): `flip{price 29251.28} does not match any number in bias.flip_condition prose "5m close below 29273.50 ONL flips short"` — **flip-prose consistency validator.**

**Verdict:** the model tried six times to author new-law plans (including the split-sweep schema and repeated breakdown_continues); the new entry law rejected all six on five distinct rules, each time with the correct named rejection and a benign fail (v2 kept). Zero crashes, zero fail-closed stubs. The law is alive and strict — notably it kept voiding breakdown_continues whose closes had come back across the level (the post-low chop), i.e., it was protecting against late shorts.

---

## M4 · WEEKLY "none" QUESTION — render gap CONFIRMED (both surfaces)

**The doc exists:** plans row `2026-08-31:WEEKLY:…` — **v1 bear/low** (draw PWL 28947.75, invalid 1h close beyond 29535.00) born **17:07:15** → **v2 neutral, invalidated_at "2026-08-30 17:07 CT", born 17:07:18 — THREE SECONDS after birth.**

**Executor prompt:** zero weekly substring in **all 13 decisions 34790–34802** (the entire move). Root cause: `kernel/weekly_prompt.go` `WeeklyExecutorLine()` returns `""` when `InvalidatedAt != ""` — an invalidated doc renders **nothing**, not `WEEKLY: neutral (invalidated 17:07 CT)` (the form `WeeklyContextLine()` already knows how to render and uses for the planner).

**UI chip:** `GET /api/plan/today` found=true response contains **no `weekly` key at all** — `api/handler_plan.go` puts `"weekly": weeklyPayload(...)` only in the `base` map used by the not-found/no-plan branches; the full found=true response builds its own map without it. `WeeklyChip.tsx` with `weekly === undefined` renders the grey **"WEEKLY none"** chip ("no Sunday weekly read stored for this week" tooltip). **The UI tells the owner there is no weekly doc while a neutral-invalidated doc exists.**

**The irony (DOA-validator + two-stage-flip evidence):** the bear doc's own invalidation ("1h close beyond 29535.00") was **breached at write** — price spiked to 29541.75 at 17:01, six minutes before the doc was born at 17:07:15; the invalidation watch killed it 3 seconds later. The invalidated BEAR was **right**: from the invalidation breach the market fell **−250pt** (29541.75 → 29291.25). The bear died on a breached-at-write invalidation, not on the tape — and both render surfaces then hid even the neutral state. Strong case for: (a) a DOA check — if the invalidation basis is already crossed at write, re-read instead of writing; (b) the two-stage flip design; (c) the two render gaps above (one-line fixes).

---

## M5 · MACHINE HEALTH UNDER FIRE

| Metric | Value |
|---|---|
| Panics since swap (18:13:46) | **0** |
| Restarts | **0** (PID 746955, uptime 44m+ through the move) |
| E8 shadow logger (`ab_confirm_log`) | **7 rows, no panic** — the fixed, recover-protected path: `touch`/`1x5m_close` @18:37:46, `2x5m_close` @18:40:01, `1m_mss` @18:50:01 across S2/S3. The exact scenario (replay over a live crashing tape) that panicked attempt #1 at 17:12:47. |
| Ingest queue | `peak_depth=699/4096` @18:41:48 · **intrabar_dropped=0, current_dropped=0, historical_dropped=0, closes_dropped=0** through the whole move |
| Watchdog | 0 alarms |
| Executor AI latency | 23.9–96.0s (reasoning=fast→low), all ok=true |
| Planner latency | 523.4s (v2 write) · 580.0 / 364.2 / 447.8 (wake read attempts) · 296.3 / 155.1 / 208.6 (fast-market attempts) — all under the 600s timeout |
| Fast-market seam | fired once (19:07:45, 3.5×ATR5m) — correct behavior, downgrade applied |

**Machine health verdict: clean.** The only damage tonight was two long-side money events, neither a crash.

---

## M6 · VERDICT TABLE

| What the machine did | $ | Class |
|---|---|---|
| S2 sweep_reclaim long: placed with stop below market → instant stop-out (entry 29357.25 → stop 29350.00) | **−$14.50** | Phase-2 placement gap + min_sl gate oscillation (21.5pt stop pinned on 1.0×ATR5m) |
| S4 ghost long (old binary, v1 arm): fill raced the cancel, stop-out | **−$44.50** | v1-era arm behavior (pre-swap) |
| S3 short fade 29414.21: never placed (wrong-side marketable guard), then min_sl refused | **$0** | Phase-2 gap; the 200pt tailwind uncaptured |
| S1 breakdown_continue short (immediate mode): machine-confirmed on tape; AI judged "already played out" and declined | **$0** | no-chase discipline; entry window ≈2 cycles (~4 min) |
| AI decisions during the move | 13/13 wait | correct no-chase, but zero capture |
| New entry law (6 rejections / 5 rules) | n/a | **all correct; protected against late shorts** |
| Flip → short bias (v1, 17:40:53) | ~40pt late | flip-hygiene evidence |
| Fast-market planner mode | fired 19:07 (11 min after the low) | seam works; drift basis = last plan write |

**Net move P&L: −$59.00** (ledger −$44.50 + unledgered NT8 −$14.50). Crash-day comparison: the 08-28 waterfall was also **$0 captured** — tonight the machine additionally paid $59 to two longs against the tape.

**The would-have (E7 stop-entry seam, currently off):** with `STOP_ENTRY_SEAM=on`, the S1 breakdown_continue (immediate mode) becomes a resting **stop-market entry at the breakdown level + offset** instead of waiting for the AI to chase. S1's machine confirm fired with the 18:30 5m close below 29371.50 (the seam places on the machine confirm, not on the AI's body-displacement judgment that declined at 18:37). Would-have: short entry ≈29371.50 (stop placed ~18:31–18:35) → chain 29357.25 / 29335.50 / 29310 all hit by 18:50 → **+$28.50 / +$72 / +$123** depending on scale-out, with the plan's own stop above. E7 would have converted the machine-confirmed S1 into the one short this tape paid for. (Caveats: the AI's displacement objection at 18:37 was against the PROSE trigger's "~21pt displacement" clause, not the machine rule — the seam bypasses prose judgment; and the v2 plan was born 18:35:34, so a seam entry would realistically land 18:36+ at ~29360, still +$51–$100 across the chain. Even the conservative case ≈ **+$51**.)

**Bug-class candidates for the fix PR (audit was read-only; AUDIT-CHECKLIST append rides the fix):**
1. **Weekly render gap ×2** — `WeeklyExecutorLine` returns "" for invalidated docs (prompt silence) + `handlePlanToday` found=true response omits the `weekly` key (UI shows "WEEKLY none" while the doc exists).
2. **Armed-entry wrong-side placement** — Phase-2 tick-managed placement absent: S2 placed a limit whose stop was already beyond the market → instant stop-out; the min_sl gate's live-ATR oscillation (refuse→allow→cancel in 105s) amplifies it.
3. **Sub-60s round-trip ledger gap** — entry+exit inside one snapshot interval leaves the parked close permanently unresolved (NT8 equity diverges −$14.50 from the ledger).
4. **Flip latency** — 7× closes + 30-min age guard = 40pt/21min late on a correct flip (evidence item, hygiene wave).
5. **Weekly DOA** — invalidation basis breached-at-write (3-second lifetime for v1); a write-time breach check would re-read instead of stamping a stillborn doc.
