# 2026-08-31 — Lead-Time Cutover + Weekly "none" Render Fix

One wave, one cutover (10:39:29 CT, rev `2bc58ed9a4ac`).

## Part A verdict (session reads → open−30, owner ruling 2026-08-31)

| Item | Result |
|---|---|
| A1 times | ASIA **16:55→16:30** · LONDON **01:55→01:30** · NY **08:25→08:00** CT. Source = BOTH: code default `kernel/session_registry.go:90/99/108` AND the persisted `system_config` row (`key='session_registry'`) — the live system loads the DB row, so it was updated at cutover (backup `~/nofx-backups/lead-time/data.db.pre-readtimes`, idempotent WHERE-scoped). Windows/flats UNCHANGED. |
| A2 Sunday collision | Weekly read fires Sunday 16:30 (`kernel/weekly_knobs.go` `WeeklyReadSpec`). Chosen sequencing (mechanically simplest, no timers): the ASIA read **defers until this week's weekly doc lands** — `sundayAsiaDeferred` (`trader/auto_trader_planner.go`), pure + fixture-tested; the per-cycle retry fires ASIA right after the weekly write. Weekdays defer nothing. |
| A3 staleness | No plan-age staleness gate exists anywhere (grep-proven: zero `planAge`/`stalePlan`-class gates). A plan authored at open−30 is treated identically at open; trade-time staleness belongs to the executor guards (F3 fast-market drift re-read, stale_reeval) — untouched. |
| A4 boot line | `🗓 session reads (owner ruling 2026-08-31, open−30): ASIA 16:30 · LONDON 01:30 · NY 08:00 CT — windows/flats unchanged; Sunday weekly 16:30 → ASIA follows` (boot 10:39:29). |

Live read-time proof: cutover landed 10:39 CT → the **ASIA read at 16:30 today** is the live fire (next scheduled proof; the scheduler next-fire fixture asserts 16:30/01:30/08:00 already).

## Part B verdict (weekly "none" + strikethrough)

Surface census (post-boot state; weekly doc = v2, bias neutral, `invalidated_at "2026-08-30 17:07 CT"`):

| Surface | Before | After | Miss class |
|---|---|---|---|
| (a)+(b) chip (dashboard + plan card, `WeeklyChip.tsx`, fed by `/api/plan/today` `weekly` key) | "▼ WEEKLY bear" or "• WEEKLY neutral" **with line-through + opacity 0.7** (`textDecoration: line-through` at `WeeklyChip.tsx:66`) | **"• WEEKLY neutral" — NO strikethrough, full opacity**; tooltip "WEEKLY: neutral (invalidated 2026-08-30 17:07 CT)". Live payload quote: `{"bias":"neutral",…,"invalidated_at":"2026-08-30 17:07 CT"…}` | B4 — CSS styling on the invalidated state implied dead data |
| (d) planner "## Weekly Context" (`WeeklyCtx` ← `weeklyDocCached` ← `WeeklyContextLine`) | **"WEEKLY: none"** — `weeklyDocCached` CACHED NEGATIVE results (`loaded=true` set before the fetch): a cycle that ran before the Sunday weekly read landed pinned nil for the whole week | retries the lookup every cycle until the doc lands (only successful loads cached, `trader/auto_trader_weekly.go`) | B2 — cached payload (sticky negative cache) |
| (c) executor weekly line (`WeeklyExecutorLine` via `WeeklyDocFor`) | silent when the same sticky nil hit | "WEEKLY: neutral (invalidated 2026-08-30 17:07 CT)" — computed from the live doc; same cache fix | same sticky-negative cache |
| (e) Guide `weeklyBias.ts` | documented the strikethrough as intended | updated: invalidated = VALID neutral state, never struck through; "none" reserved for genuinely-no-doc | doc-only |

B3/B4 law applied: invalidated → neutral + date, no strikethrough, no opacity drop; the stale bias (bear) never displays. "none" is reserved for no-doc (verified live: the payload IS present, so the chip can never show "none" for this week).

B5 fixtures: chip invalidated-state (neutral, no strike, no stale bias) · `weeklyDocCached` negative-not-sticky · `WeeklyContextLine` forms unchanged and covered · Sunday defer-then-fire + weekday-plain read fixtures.

## Tests

New/updated: `trader/read_times_lead_test.go` (registry times, IsReadTime, Sunday defer, negative-cache unsticky) · `trader/p0b_asia_clock_test.go` (16:30 fire + Sunday defer-then-fire with seeded weekly doc) · `trader/w1_read_guard_test.go` + `kernel/session_registry_test.go` + `trader/acceptance_scheduler_test.go` re-pointed to the new times · `W20_weekly_chip.test.tsx` no-strike contract. Full `go test ./...` green, FE vitest 22/22, tsc clean.

## Cutover

Flat-gate: DB OPEN 0 · orders NT 0 · armed NT 0 · API `[]` · NT8 snapshot count=0. RELEASE + GUIDE_BUILT_REV = `2bc58ed9`. Boot checklist: `🔐 BOOT INTEGRITY OK — rev 2bc58ed9a4ac · goldens PASS` · new session-reads boot line · entry-law/armed seams unchanged. Rollback: `nofx-bin.prev.boot` = `5bf48951` (RELEASE revert + swap + kill -9).
