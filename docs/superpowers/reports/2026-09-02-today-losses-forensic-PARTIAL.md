# Today's losses — PARTIAL forensic (dispatch interrupted mid-run)

**Status: INCOMPLETE.** The owner redirected to the bar-source wave before Sections D (gap
mechanics from code), E (raw export) and F (verdict) were produced. What follows is what was
MEASURED before the interrupt. Read-only; nothing was changed. [A] = verified directly.

## LIVE RISK — read this first

The R:R gate's entry reference does not match the price the market order fills at, and the
mismatch was ~10 points on two of four trades today. Combined with the 2.00 floor and passes at
2.02-2.03, both trades executed BELOW the floor the owner set. **This is still live in
`0465a10b`.** It is not an emergency (no arm is resting; the one `armed` row id 32 has no
signal_id, so nothing is at the broker) but it can admit more sub-floor entries today.

## The four trades — all long, all losses, −381.50 corrected [A]

| id | entry CT | entry | exit | pnl_corrected | plan | MAE | MFE | exit was |
|---|---|---|---|---|---|---|---|---|
| 587 | 00:17:44 | 29079.25 | 29048.00 | **−62.50** | ASIA v7 S3 | 33.00 | 25.75 | CLOSE below |
| 588 | 07:41:05 | 29082.50 | 29050.00 | **−65.00** | LONDON v3 S2 | 34.50 | **−1.25** | WICK only |
| 589 | 09:41:04 | 29192.50 | 29115.00 | **−155.00** | NY v3 S3 | 81.25 | 10.25 | WICK only |
| 590 | 10:37:17 | 29193.25 | 29143.75 | **−99.00** | NY v5 S4 | 52.25 | **1.00** | CLOSE below |

**Three of four never traded meaningfully above entry** (MFE −1.25, 1.00, 10.25). 588 never
traded above its entry at all. These were wrong from the first tick — an ENTRY problem, not a
stop problem. A tighter stop would have lost less money on the same four losers.

All four entered via the executor's MARKET path (`source=system`, `entry_order_id` NULL,
`close_reason=sync`) — **not** the armed-limit path. The 0B stop wave and the arm R:R gate did
not author these trades.

## THE R:R GATE — the direct test of the owner's 3.0 → 2.0 change [A]

Journal `📐 R:R eval` lines (`kernel/engine_position.go:187`) vs the actual fills:

| id | gate entryRef | gate R:R | verdict | actual fill | slip | **R:R at the real fill** | 3.0 would? |
|---|---|---|---|---|---|---|---|
| 587 | 29069.50 | **2.03** | PASS | 29079.25 | **+9.75** | **1.09** | REFUSE |
| 588 | 29081.25 | 2.68 | PASS | 29082.50 | +1.25 | 2.54 | REFUSE |
| 589 | 29182.00 | **2.02** | PASS | 29192.50 | **+10.50** | **1.61** | REFUSE |
| 590 | 29196.50 | 2.29 | PASS | 29193.25 | −3.25 | 2.51 | REFUSE |

**ALL FOUR would have been refused at a 3.0 floor.** Today would have been a zero-trade day and
the −381.50 would not exist. Zero `R:R REJECT` / `→ FAIL` lines fired all day.

**Two filled below the 2.0 floor the owner set** (1.09 and 1.61), because the gate evaluated a
snapshot price and the market order filled ~10 points worse. Those two are −62.50 and −155.00 =
**−217.50**. The 3.0 floor's extra margin had been absorbing exactly this mismatch.

**It is not a tape gap.** 1m bars around all four entries are contiguous (max open-vs-previous-
close 0.75 pts). The mismatch is between the gate's reference and the fill, not a market gap. [A]
The mechanism of the reference itself (`current_price_snapshot`,
`kernel/engine_position.go:136-145`) needs its own investigation — on 589 the reference 29182.00
equals the 09:41 bar's CLOSE, a price that had not yet occurred at 09:41:04. [B]

## Era comparison, pnl_corrected, stated with its tiny n [A]

| Window | side | n | wins | Σ | avg move |
|---|---|---|---|---|---|
| 7 days before 09-01 08:13 | LONG | 10 | 4 (40%) | −96.00 | 25.98 |
| 7 days before | SHORT | 11 | 6 (55%) | +232.00 | 38.18 |
| After 09-01 08:13 | LONG | **5** | **0** | **−435.50** | **43.55** |
| After 09-01 08:13 | SHORT | 2 | 1 | +28.50 | 45.88 |

Per-long loss went from −9.60 to −87.10 and the average move grew 68%. **n=5 after; no
significance claimed.** The long side has bled across the whole record (124 longs, 27.4% wins,
−3167.09 lifetime vs shorts 98, 31.6%, +119.32) — today continues a long-standing pattern, it
did not start it.

## Not done (the interrupt)

Sections D (gap mechanics quoted from code), E (raw CSV/JSONL export), F (formal verdict), the
per-trade level/seating detail, the 0B before/after stop split (C3), and bias-vs-tape (C4).
Bars available for export: MNQ 1m from 2026-08-19, ES 1m from 2026-08-24; **no 5m/15m/1h/daily
rows exist in the store** — higher TFs are derived at runtime, which is itself the subject of the
bar-source wave that interrupted this one.
