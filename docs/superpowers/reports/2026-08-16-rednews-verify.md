# Red-News Pipeline Verify — end-to-end chain (read-only)

**Date:** 2026-08-15 (Sat) · **HEAD:** 8c335333 · **Running binary:** started 10:05:34 CT (predates W3) · **Verdict line:**

## LINE 1: BROKEN AT LINK 1 — FETCH NEVER FIRED: the running process (10:05:34) predates the W3 calendar commit (11:38:41); the final binary (built 12:24) is on disk but NOT running. Everything downstream of the fetch is PROVEN via the real code path.

## Chain table

| # | Link | Verdict | Receipt |
|---|------|---------|---------|
| 1 | Fetch fired | **BROKEN (live)** [A] | Zero calendar/forexfactory lines in `data/nofx_2026-08-15.log` + journal since boot. Root cause: process started **10:05:34** (ps lstart) when HEAD was `aef31090` (10:02); W3 (`89e2809f`, calendar producer) landed **11:38:41**; binary rebuilt **12:24** (HEAD `8c335333` 12:22) but never launched. The ★RESTART 2 fired 93 min too early. Code is correct in tree: `maybeFetchCalendar` — on-boot-if-empty (`GetSlice==nil`) + new-day trigger + 1h outage throttle ([trader/auto_trader_calendar.go:20-45]) |
| 2 | Slices stored | **BROKEN (live)** [A] | `SELECT count(*) FROM calendar_slices` → **0 rows** (mode=ro). Consistent with link 1. All 6 day-plan tables (plans, overlays, digests, alerts, qa) also 0 rows — nothing W3+ ever ran |
| 3 | Tier mapping | **PROVEN** [A/B] | Code: High→T1, Medium→T2, Low/Holiday/unknown→dropped; currency whitelist USD/EUR/GBP/JPY/CNY ([calendar/calendar.go:126-155]); NY session = USD-only (:174-183). Live thisweek feed cross-check (74 events): `USD High "Core CPI m/m"`→T1 rule ✓, `USD Medium "Unemployment Claims"`→T2 ✓, `Low "Bank Lending y/y"`→dropped ✓. Full-week sandbox: see companion report. Tests: 5/5 pass (`TestFetchWeekLive`, `...FallbackOnOutage`, `...NoStaticNoCrash`, `...BadJSONFallsBack`, `TestSessionCurrencyFilter`) |
| 4 | Planner receives | **PROVEN (sandbox — real code path)** [B] | Live store empty → per dispatch, demonstrated with fixture through the SAME path: `FetchWeek` → `EventsForSession` → `sessionPlannerEvents` ([trader/auto_trader_calendar.go:77-90]) → `kernel.BuildPlannerPrompt`. Calendar section rendered with `T1 — ... (HARD no-trade blackout)`; `kernel.T1NoTradeLines` auto-writes `🔴 ... ±15m — HARD no-trade (red news)` into `doc.NoTrade`, deduped, model can't omit it ([trader/auto_trader_planner.go:249-250, 277-296]). Tests: `TestW3RedNewsEndToEnd`, `TestBuildPlannerPrompt`, `TestSamplePlannerPrompt` pass |
| 5 | Executor honors | **PROVEN** [B] | Entry gate: `sessionEntryBlocked` → `sessionGateDecision(reg, now, at.currentT1Windows(now))` ([trader/auto_trader_session.go:38]); blackout check `kernel.InT1Blackout` wrap-aware ±15m ([kernel/calendar_blackout.go:14, 26-60]). Sandbox: 13:10 CT entry **blocked** (`"FOMC Meeting Minutes 13:00 ±15m"`), 12:30 CT clear. Tests: `TestW3RedNewsEndToEnd` (trader), `TestT1BlackoutWindows` + `TestT1BlackoutEmpty` (kernel) pass |
| 6 | Freshness (frozen) | **PROVEN** [C] | `SaveSliceIfAbsent`: existing row → `return false, nil` — "frozen — never overwrite" ([store/calendar.go:48-67]); readers consume ONLY `GetSlice(tradeDate)` ([trader/auto_trader_calendar.go:25,63], planner [trader/auto_trader_planner.go:425]); `calendar.FetchWeek` has exactly ONE caller (`maybeFetchCalendar`) — no read-time refetch anywhere |

## Gate preconditions (all satisfied once restarted)

- `day_plan.plan_enabled: true` on BOTH live strategies [A: DB ro] — `dayPlanEnabled()` requires exchange==ninjatrader + PlanEnabled ([trader/auto_trader_clock.go:22-28]) ✓
- Static-T1 fallback loader returns nil by design (no file yet) → outage = live-feed warning, never fabrication ([trader/auto_trader_calendar.go:47-49])
- FF next-week URL does not exist (`ff_calendar_nextweek.json` → 404): the owner's Aug-17 week becomes fetchable when `ff_calendar_thisweek.json` rolls on Sunday

## Required owner action (not performed — read-only dispatch)

Restart onto the already-built 12:24 binary (standing deploy rule: `kill -9 <PID 1010421>` in the flat window; Saturday = market closed = safe). After Sunday's feed roll, the first cycle of the new trade date fetches + freezes the owner's week; re-run the live verify then (companion report has the ground-truth diff ready to compare).
