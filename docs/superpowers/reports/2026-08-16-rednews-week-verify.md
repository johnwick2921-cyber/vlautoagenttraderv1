# Red-News Week Verify vs Owner Ground Truth (Aug 17–21, NY/USD slice)

**Date:** 2026-08-15 (Sat) · **Method:** live store EMPTY (see companion chain report — running binary predates W3, and FF's thisweek feed still holds Aug 10–16; `ff_calendar_nextweek.json` = 404), so the owner's hand-filtered week was encoded as a raw ForexFactory-format fixture (ET timestamps, FF impact strings, incl. negative-test rows) and pushed through the **REAL code path** via a go-overlay sandbox test (`FetchWeek → EventsForSession("NY") → sessionPlannerEvents → T1BlackoutWindows → BuildPlannerPrompt`) — zero repo changes, `--- PASS`.

## LINE 1: CODE PATH MATCHES OWNER GROUND TRUTH — 1 T1 (FOMC WED 13:00 CT), 0 mismatches across 19 USD events + 11 negative rows. LIVE STORE = EMPTY (restart + Sunday feed-roll required before the [A] live re-check).

## Per-day diff (system NY slice vs owner list)

| Day | Owner ground truth (CT) | System NY slice (real path) | Diff |
|-----|------------------------|------------------------------|------|
| MON 08-17 | Empire 7:30 · NAHB 9:00 · TIC 3:00pm | `07:30 T2 Empire State` · `09:00 T2 NAHB` · `15:00 T2 TIC` | ✅ 3/3, times exact, all T2 |
| TUE 08-18 | ADP 7:15 · Permits+Starts+Imports 7:30 · CapUtil+IndProd 8:15 · PendingHomes 9:00 · API 3:30pm | `07:15 ADP` · `07:30 ×3 (Permits, Starts, Import Prices)` · `08:15 ×2 (CapUtil, IndProd)` · `09:00 PendingHomes` · `15:30 API` — all T2 | ✅ 8/8 |
| WED 08-19 | Crude 9:30 · **🔴 FOMC MINUTES 1:00pm** | `09:30 T2 Crude Oil Inventories` · **`13:00 T1 FOMC Meeting Minutes`** | ✅ 2/2 — the T1, tz-correct (14:00 ET → 13:00 CT) |
| THU 08-20 | Philly+Claims 7:30 · CB Leading 9:00 · NatGas 9:30 | `07:30 ×2 (Philly, Claims)` · `09:00 CB Leading` · `09:30 NatGas` — all T2 | ✅ 4/4 (fixture's `Low "Existing Home Sales"` correctly dropped) |
| FRI 08-21 | Flash Mfg + Services PMI 8:45 | `08:45 ×2 (Flash Mfg PMI, Flash Svcs PMI)` — T2 | ✅ 2/2 |

**T1 assertion:** exactly **1** T1 across the week = FOMC Meeting Minutes, trade_date 2026-08-19, stored-time 13:00 CT. Zero T1 on the other four days. Test asserts `t1Total == 1` and errors on any T1 outside 08-19 → PASS.

**Negative tests (all held):** Mon CAD CPI cluster (3 High rows) → dropped at parse (CAD ∉ interest currencies); GBP CPI y/y (High!) + EUR ZEW + AUD RBA minutes + NZD retail → never reach NY slice; CNY Loan Prime Rate + JPY Trade Balance → interest-currency but filtered by NY=USD-only; GBP Rightmove Low + JPY Low + USD Low → dropped as yellow. Assertion: any non-USD in an NY slice = test error → none fired.

## Planner CALENDAR sections (verbatim from BuildPlannerPrompt)

WED 08-19:
```
## Calendar (this session's window)
  09:30 USD T2 — Crude Oil Inventories (caution)
  13:00 USD T1 — FOMC Meeting Minutes (HARD no-trade blackout)
```
Auto no_trade (count 1): `🔴 FOMC Meeting Minutes 13:00 ±15m — HARD no-trade (red news)` · blackout window = CT minutes [765, 795] = **12:45–13:15**.

MON 08-17:
```
## Calendar (this session's window)
  07:30 USD T2 — Empire State Manufacturing Index (caution)
  09:00 USD T2 — NAHB Housing Market Index (caution)
  15:00 USD T2 — TIC Long-Term Purchases (caution)
```
Auto no_trade (count 0) — T2 = caution only, no blackout. ✅ as specified.

## Gate + frozen receipts

- Entry at **13:10 CT** Wed → **BLOCKED** (`InT1Blackout` → `"FOMC Meeting Minutes 13:00 ±15m"`); 12:30 CT → clear. Wired at [trader/auto_trader_session.go:38] via `currentT1Windows`; tests `TestW3RedNewsEndToEnd` + `TestT1BlackoutWindows` pass.
- Frozen rule: reads consume the stored slice only (`GetSlice`); `SaveSliceIfAbsent` never overwrites ([store/calendar.go:48-67]); `FetchWeek`'s sole caller is the producer.

## Pending [A] live confirmation (after owner restart + Sunday feed roll)

1) `calendar_slices` rows 08-17..08-21, `source=forexfactory` · 2) Wed slice contains the 13:00 CT T1 · 3) re-diff against this table — the fixture encodes the owner's list, so any live-feed divergence (FF reschedules, added events) will surface as an explicit diff, not silence.
