# PLANNER-QUALITY FIX — PLANS NOBODY CAN TRADE (2026-08-19)
**levels below price: was 1-of-8 on the breakdown day (≈0.25/8 avg over the 4 stored plan days) · now ≥3-of-8 on EACH side enforced · scenarios that would have triggered on the last 5 days: was 0 on the breakdown day (coarse 4-day replay: 1 unflagged touch, none clean) · now ≥1 on the gap day, guaranteed any gap day · deploy verdict: DEPLOYED c42b7280, boot 15:19:18 CT, goldens PASS**

## Receipts (root cause, file:line)
1. **Downside blindness** — seating in `levels_score.go`: today-priority kinds (PDH/PDL/PDC/RTH/OR/ONH/ONL) sort FIRST and filled all 8 seats; on a gap-down day they all sit above price (2026-08-18: 29680.75–30092, price 29687). The planner prompt says "levels chosen ONLY from the ranked table" (`planner_prompt.go`) → the plan could only copy a one-sided map. Verified: `plans` row 2026-08-18 NY = 1 below / 7 above.
2. **Scenario shape** — the output contract had no mix rule, so the model defaulted to 2 longs + 1 rally-rejection short daily (`planner_prompt.go:154`), incl. days that opened below PDL.
3. **Death/flip display-only** — Go's death check is all-levels-consumed (`plan_lifecycle.go:PlanIsDeadSince`); the planner's own `death_condition` prose was never evaluated. Today's death text fired ~09:00, zero replans (`plans` NY stayed v1).
4. **Grade inflation** — `levels_score.go` grades EQ clusters by self-confluence (each member A) + no collapse → 4×A within 3 pts.
5. **S/D zones** — dropped at `levels_score.go:167-168` (zero-confluence) + max-8 cap.
6. **Calendar fail-open** — `auto_trader_calendar.go:112-122` returned nil windows on a missing slice.

## Fixes (one commit per item, all deployed)
1. `seatBothSides` + `collapseLevelClusters` + HTF zones seat as C (`73976c96`…`e8a4b710` chain): ≥3 levels each side; EQ family → 1 entry; strong zones seat.
2. `ValidatePlanDocWithFacts` (`c42b7280`): ≥3 each side, continuation short when price < PDL (long when > PDH), trigger must be reachable in the gap direction (today's "rally into 29853" plan is REJECTED), no duplicate seats, targets inside the band; write-time enforced with retry → fail-closed.
3. `PlanDeathOrFlipSince`: structured `death`/`flip` objects (price/side/rule) machine-evaluated every cycle on the rule timeframe, touch-gated; a fired flip = death → existing replan budget. Unparseable conditions are rejected at validation.
4. Planner contract: scenario MIX follows regime/day_type; levels MUST include ≥3 below and ≥3 above; structured conditions required.
5. Calendar: no slice → static T1 fallback blackout + P0 alert once per session-day (never silently unprotected).

## Before/after on real days (stored plans vs realized tape)
| day | price | below/above (was) | move | triggered was | now |
|---|---|---|---|---|---|
| 08-15 NY | ~30150 | 0/8 | up | 0 | ≥3+3, mix rule |
| 08-16 ASIA | ~30200 | 0/8 | up | 0 | ≥3+3 |
| 08-17 ASIA | ~30190 | 0/8 | up | 1 touch (S2) | ≥3+3 |
| 08-18 NY | 29687 | **1/7** | down | **0** (S3 needed a 166-pt rally) | ≥3+3 + forced breakdown short → tape crossed ONL → ≥1 triggered |

Trigger distance (item 2b): today's S3 = 166 pts (~0.55×dATR) — reachable only by rally; now such shapes are rejected unless the trigger level is AT/beyond price in the gap direction.

## Exit bar
go build/vet/test/-race green (kernel/trader/mcp/api); vitest 247 pass, 1 pre-existing env failure (jsdom canvas) + Playwright e2e misconfig — untouched by this change; goldens unchanged (0 diffs). Deployed at the 14:45–17:00 flat window. Boot: `🔐 BOOT INTEGRITY OK — rev c42b72808e8e · goldens PASS`.
