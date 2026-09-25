# DAY-PLAN CAMPAIGN — P2 · THE CLOCK (+ ★ RESTART 1 handoff)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `3bcd1132` (P1 head) → `b0151d98` · 5 feature commits
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**P2 · THE CLOCK is COMPLETE (5/5) and pushed.** Bar-close cadence, the
skip-while-open gate, the last_entry/eod_flat clock, MAE/MFE + confidence logging,
and the arm tool all landed with tests; full suite + `-race`(trader,kernel,store)
green; **no golden touched** (P2 doesn't touch prompts); every behavior is GATED on
day_plan → the running bot (rev 3624a2a4, PID 363618) is byte-identical until the
owner arms day_plan and restarts at ★ RESTART 1 (procedure below).

## STEP 0 gate
PASS — HEAD `3bcd1132` · tree clean · bot PID 363618 (systemd) untouched · cycling.

## Items shipped
- **P2.1** `069500c9` — true bar-close cadence. `barCloseGate` (pure) fires a cycle
  once per NEW primary-TF bar close (never mid-bar; restart resumes on next close);
  the Run loop routes first-run + ticks through `tickOnce`. GATED
  (`barCloseCadenceActive` = futures + day_plan) → scan-timer default unchanged.
- **P2.2** `3bb7a730` — skip-while-open gate (RECON #5: did not exist). Holding →
  skip the AI cycle entirely (the bracket/breakeven/close-sync still manage the
  trade). GATED via `dayPlanEnabled()`. Placed in runCycle after the snapshot.
- **P2.3** `21c57118` — day-trader clock fields. `DayPlanConfig.LastEntryCT`
  (13:00 CT = 14:00 ET) + `EODFlatCT` (14:45 CT = 15:45 ET). last_entry: a new
  entry gate refuses opens after the cutoff. eod_flat: `enforceEODFlat` (in
  runCycle, before skip-while-open) force-closes via the trader close path
  (CloseLong/CloseShort qty 0 — bypasses hold-lock, RECON #10) + cancels the OCO
  bracket. Half-day aware via `effectiveEODFlatCT` (registry HalfDays, calendar-fed).
- **P2.4** `a43d006d` — MAE/MFE + entry-confidence. `ComputeExcursion` (pure) +
  additive columns `mae/mfe/entry_confidence` on trader_positions. Confidence
  threaded from `decision.Confidence` and captured at open; MAE/MFE computed at
  close over the hold's 1m bars. GATED (futures + day_plan).
- **P2.5** `b0151d98` — `cmd/dayplan-arm` (guarded, idempotent, dry-run) to arm
  day_plan + clock fields on the AI strategies at ★1.

## config-truth (new fields)
`last_entry_ct` / `eod_flat_ct` ride the `DayPlanConfig` codec (P0.1); persistence
proven save→row→reload→read (`TestDayPlanClockFieldsPersistThroughRow`). The
`mae/mfe/entry_confidence` columns are additive (AutoMigrate), written by the store
+ trader, round-trip tested. FE editors = P4.

## EXIT BAR
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./trader ./kernel ./store` ✓.
- Goldens: **none changed** (`git diff 3bcd1132..HEAD -- kernel/testdata/` empty).
- tsc/npm: N/A — zero `web/` files (the clock has no UI; the Plan Card is P4).

---

## ★ RESTART 1 — OWNER HANDOFF

This lights up the whole map + clock: the KEY LEVELS block in the live prompt
(P1.7), bar-close cadence, skip-while-open, last_entry/eod_flat, MAE/MFE, and the
session-profile snapshot writer. **Preconditions:** prefer a FLAT moment (no open
positions) in a calm window; if not flat, the position/close reconcile re-anchors
on restart (no loss), but flat is cleanest. The bot is systemd-managed, so use
`systemctl` (owner has sudo) — that gives a clean stopped window for the arm; the
old binary predates the day_plan codec, so it must NOT be running during the arm.

```bash
cd /home/hoang/nofx

# 1. Rebuild the new binary (has all of P0–P2).
go build -o nofx-bin .

# 2. Stop the bot (clean stopped window; prevents auto-relaunch during the arm).
sudo systemctl stop nofx

# 3. Back up the DB (before the arm write + the additive schema migration).
mkdir -p ~/nofx-backups/dayplan-restart1
cp data/data.db ~/nofx-backups/dayplan-restart1/data.db.$(date +%Y%m%d-%H%M%S)

# 4. ARM day_plan on the AI strategies (bot STOPPED). Preview, then confirm.
go run ./cmd/dayplan-arm              # dry-run: lists what would be armed
go run ./cmd/dayplan-arm --confirm    # writes plan_enabled=true + last_entry 13:00 + eod_flat 14:45

# 5. Start the new binary (reads the armed config; KEY LEVELS lights up next cycle).
sudo systemctl start nofx
```

### VERIFY (5 lines — `journalctl -u nofx --since "$(date +%H:%M -d '-5 min')"`)
1. **Cadence:** decision cycles fire on 5m bar closes, not the old scan interval
   (idle ticks between closes; a cycle right after each 5m bar).
2. **Skip-gate:** while holding, a `🧘 skip-while-open` line appears (no AI decision
   mid-trade).
3. **Clock armed:** `dayplan-arm` printed `plan_enabled=true last_entry=13:00
   eod_flat=14:45`; after 13:00 CT opens show `🕒 last-entry cutoff`, at 14:45 CT a
   `🕒 EOD-FLAT` flatten.
4. **KEY LEVELS live:** the latest `decision_records.system_prompt` contains
   `KEY LEVELS (map` (`sqlite3 data/data.db "SELECT substr(system_prompt,1,0) ...`
   or grep the assembled prompt in the log).
5. **Clean boot:** `journalctl -u nofx --since <start> | grep -iE "error|panic|🚨"`
   is empty; `hello handshake OK protocol_version=3` present.

**Rollback:** `sudo systemctl stop nofx` → restore `~/nofx-backups/dayplan-restart1/`
→ rebuild the prior commit → start. (day_plan is additive; disabling it =
`plan_enabled:false` or restoring the pre-arm config — everything returns to the
scan-timer default.)

## What's next — P3 · THE PLANNER
Per-session read jobs at the registry times (16:55 closed-market read = first-class
tested path), the planner-model binding (RECON #12), the spec input package,
schema-strict fail-closed JSON, the prompt reorder (RECON #4), advisory mode +
match-rate. vlauto: DEFERRED to the next propagation train.
