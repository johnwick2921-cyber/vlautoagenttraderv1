# VL — DEMO PLAN SEED (2026-08-16)

**LINE 1: DEMO PLAN LIVE — go look (today 08:30–15:00 CT; the card only renders while the NY window is open).**

**Seeded key: `trade_date=2026-08-16` (SUNDAY) + `session=NY`, currently at v2.**
Isolated three ways: the **scheduler can never write it** (`maybeRunSessionReads` needs `IsCMEOpen` AND the 08:25–15:00 window; on a Sunday CME is shut until 17:00, by which time the NY window is closed — the two can't both be true); the **executor can't act on it** (`runCycle`'s first gate is `cmeSessionClosedSkip()`, which returns before any context build, AI call or order path all Sunday; after 17:00 CT the active session is ASIA, which is disabled, so `ActivePlanProvider` returns nil); and it **self-expires** — at Monday 00:00 CT the trade date rolls to 2026-08-17 and the demo becomes invisible. Verified: `2026-08-17:NY` does not exist.

Backup taken first: `~/nofx-backups/demo-seed-20260816-001118/data.db` (423M, `integrity_check=ok`).

## The 10 steps

1. Frontend is already up on **http://127.0.0.1:3000** (backend `127.0.0.1:8080`). Log in as usual.
2. Go to **Traders → pick `hoang`** (the MNQ trader), which lands you on `/dashboard`.
3. **Top-left: the Session Plan Card.** Bias reads *long / high conviction* with the flip line. Version chip shows **v2**; overlay count 1.
4. **Levels table** — 8 rows, mixed provenance and grades: `PDH A`, `RTH-H A`, `RN A`, `PDC B`, `👤 D-zone 📝 [S1] A`, `nPOC·Thu B`, `PDL A`, `ONL C`. Distances/sweep/acceptance are computed live from the real bar cache, so they move.
5. **Tap any level row → the Edit sheet.** Change a price/grade/instruction and Save: it writes an RFC-6902 overlay and the version chip bumps. **＋ Add** and **Bulk add** open the owner-level sheets (bulk-add accepts the marking grammar, e.g. `30120 D-zone note here`).
6. **Scenarios** — S1 sweep_reclaim long, S2 reject short, S3 breakout_retest long, each with target chains and invalidation. *(Schema caps scenarios at 3, so there is no 4th.)*
7. **No-trade block + "Plan dies if"** — includes the **🔴 FOMC 13:00 CT ±15m** hard blackout, which is the line that will matter Wednesday.
8. **Alert bell (top right): badge = 2.** One **P0 unacked** drives the persistent banner; a P1 sits in the feed; a P2 collapses into the digest count. **Click Ack** on the P0 → banner clears and stays cleared after reload.
9. **Session tabs / timeline** — NY is the active tab; ASIA and LONDON render as disabled/night. Switching tabs is currently cosmetic (known gap, below).
10. **Ask-Planner (💬)** opens and shows the EVIDENCE→POINT→VERDICT form. ⚠️ **Sending a question costs one real API call** (~pennies) — the panel opens free; only pressing send spends.

**Mobile:** 390×844 works; the sheets are bottom-sheets and the card scrolls without horizontal overflow.

## Honest limits (these are pre-existing findings, not seeding shortcuts)

- **Scenario dots all show ◉ armed.** `/api/plan/today` never emits `scenario_status`, so the FE falls back to `'armed'` — the ○/●/✕ states can't be demonstrated without a Go change (acceptance-gate **F-5**).
- **👤/📝 are label text, not the real markers.** ZoneTable's true markers key off `fact.origin === 'OWNER'`, which the API also never emits (**F-5**). I embedded the glyphs in the level label so you can see the intent.
- **Alerts in the live DB were all `acked=1`** before I touched them, including two the *real* rehearsal emitted yesterday. The store code is correct in isolation (a fresh emit gives `acked=0`, badge 1) — most likely you hit "Mark all read" at some point. I explicitly un-acked the demo P0/P1 so the banner fires.
- The completed trade (S1 LONG 30040.25→30152.50, +$224.50, MAE −18.75 / MFE +126.50, **adherence A**) is seeded, but there is **no UI that renders it** — `/plan/trades` has no FE consumer yet (**F-6**). It's queryable, not clickable.

## Cleanup — one command, run it before Monday

```bash
cd /home/hoang/nofx && sqlite3 data/data.db "
DELETE FROM plans            WHERE trade_date IN ('2026-08-16','2026-08-15');
DELETE FROM plan_overlays    WHERE plan_id   IN ('2026-08-16:NY','2026-08-15:NY');
DELETE FROM day_plan_alerts  WHERE event_id LIKE 'demo:%' OR event_id LIKE '%:2026-08-15:%';
DELETE FROM owner_levels     WHERE label = 'DEMO D-zone';
DELETE FROM trader_positions WHERE source = 'demo_seed';
" && sqlite3 "file:data/data.db?mode=ro" "
SELECT 'demo plans left:   '||COUNT(*) FROM plans WHERE trade_date IN ('2026-08-16','2026-08-15');
SELECT 'demo overlays left:'||COUNT(*) FROM plan_overlays WHERE plan_id LIKE '2026-08-1%';
SELECT 'demo alerts left:  '||COUNT(*) FROM day_plan_alerts WHERE event_id LIKE 'demo:%';
SELECT 'demo levels left:  '||COUNT(*) FROM owner_levels WHERE label='DEMO D-zone';
SELECT 'demo trades left:  '||COUNT(*) FROM trader_positions WHERE source='demo_seed';
SELECT 'MONDAY 2026-08-17: '||COUNT(*)||' rows (must be 0 until 08:25 CT)' FROM plans WHERE trade_date='2026-08-17';"
```

It also clears the 2026-08-15 acceptance-rehearsal leftovers (2 expired plans + 2 alerts). **Dry-run proven**: executed against a WAL-safe copy of the live DB — all demo rows went to 0 while **516 real trades, 2 traders and 9 strategies were untouched**. (Use `sqlite3 .backup`, not `cp`, to copy this DB — a plain `cp` misses the WAL and silently loses recent tables.)

## Files

Seeder `trader/demo_seed_test.go` (idempotent, `NOFX_DEMO_SEED=1`, no paid call — the JSON goes through the real `ParsePlanDoc`/`ValidatePlanDoc`) and verifier `trader/demo_verify_test.go` (`NOFX_DEMO_VERIFY=1`, read-only, replays the exact `/api/plan/today` data path). Both left **untracked** per the dispatch's "commit report only" — say the word and I'll commit them.

---

## ADDENDUM — isolated PREVIEW stack (added 00:25 CT, because the card showed "Night")

The owner opened the dashboard at ~00:45 CT and correctly got **Night — markets quiet**. That
is not a bug: `handlePlanToday` hardcodes `kernel.DefaultSessionRegistry()`
(`api/handler_plan.go:67`), so the card renders only while an *enabled* session is active — NY
08:30–15:00 CT. No config change can move it; the admin session-registry is not consulted here
(the D1 residual). So a second, throwaway stack now serves the demo immediately:

| | live (untouched) | preview |
|---|---|---|
| API | `127.0.0.1:8080` — real DB, real traders running | `127.0.0.1:8232` — **copy** of the DB, all traders `is_running=0` |
| UI | `127.0.0.1:3000` | **`127.0.0.1:3001`** ← open this |
| binary | `nofx-bin` @ HEAD, no demo code | `nofx-preview` (scratchpad) with an env-gated session override |

Safeguards: the override lives **only** in the preview binary (`strings` confirms the live
binary has zero occurrences); the source patch was reverted immediately after the build and the
tree is clean at HEAD; the preview runs on a WAL-safe DB copy with every trader forced
non-running; its log shows no order/NT8 activity; JWT is still enforced (`/api/plan/today`
unauthenticated → 401). Log in at `:3001` with your normal credentials.

**Stop the preview when done** (removes everything, touches nothing live):
```bash
pkill -f nofx-preview; pkill -f vite.preview.config.ts
rm -f /home/hoang/nofx/web/vite.preview.config.ts
rm -rf /tmp/claude-1000/-home-hoang-nofx/*/scratchpad/preview
```
The live seed on the real DB still needs the SQL cleanup above before Monday.
