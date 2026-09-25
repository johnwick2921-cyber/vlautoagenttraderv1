# Pre-Reopen Hotfix Wave — F1–F5 (BUILD & PROVE, NO DEPLOY)

- Branch: `fix/pre-reopen` (off `dev` @ `72c8bfd8`)
- Date: 2026-08-28
- Status: **proven & parked — awaiting owner "go cutover"** (target Saturday, before Sunday 17:00 CT market reopen)
- Canon: WORKTREE LAW (isolated `~/nofx-preop`) · GUIDE CONTENT LAW (env knob documented in same wave)

## Why this wave exists

Weekend deep-audit Part 1/2 found **2 BROKEN** items plus 3 follow-ups:

| # | Finding | Class |
|---|---------|-------|
| F1 | Waterfall `scenarioConds` (`breakdown_continue`/`breakup_continue`) parse-rejected by the schema gate | BROKEN |
| F2 | `closes_dropped=8` + ~2h GORM stall (pool starved by cycle spikes) | BROKEN |
| F3 | Dead-row re-log spam; terminal arm rows could never re-arm; missing 1.3 cancel on gate change | follow-up |
| F4 | Fill-time lineage stamp always missed (position row materializes after the fill frame) | follow-up |
| F5 | SWG levels bucketed as inert `KindRound`; `%s`/`%d` fmt verbs fed `time.Duration`/`float64`; WSL2 clock drift | follow-up |

## F1 — waterfall scenarioConds accepted by the schema gate

`kernel/plan_doc.go:225` — added `"breakdown_continue": true, "breakup_continue": true`
to the allowed-conditions map (now 9 entries). The schema reject at `:484` is untouched.

New tests (`kernel/plan_doc_waterfall_gate_test.go`):
- `TestPlanDocSchemaGateAcceptsWaterfallConditions` — both conditions pass `ParsePlanDocCapped` + `ValidatePlanDocWithCaps`.
- `TestPlanDocSchemaGateStillRejectsUnknownCondition` — `"ninth_condition"` still rejects.

## F2 — GORM pool + persist-stall watchdog (root cause of the ~2h stall)

**Root cause:** `store/gorm.go` had `SetMaxOpenConns(1)` — a single connection
served both the cycle reads and the bar-persist writer. During the 100k+ frames/min
cycle spikes (102003 / 74815 frames/min observed) the writer starved: stall WARNs
at 10:05:58, 10:08:57, 10:13:39, 10:16:19, 10:21:31, 11:03:15, with 2s/4s
bounded-block retries and peak_depth 4096 ×7. The 8 dropped closes went through
the **sacred bounded-block (2s×3) → ERROR path as designed** — the design held;
the pool did not.

Fixes:
- `store/gorm.go` — `SetMaxOpenConns(4)`, `SetMaxIdleConns(4)`, `SetConnMaxLifetime(30m)`.
- `provider/ninjatrader/bar_persist.go` — 30s watchdog ticker; every successful
  flush stamps `persistLastFlushAt`; silence beyond `PERSIST_STALL_WATCHDOG_S`
  (default **60s**, min 10, garbage → default) logs a deduped ERROR
  `🔕 PERSIST WATCHDOG: no successful bar flush for %ds (queue=%d)…`. A stall can
  never go silent again.
- Guide (GUIDE CONTENT LAW): `PERSIST_STALL_WATCHDOG_S = 60` documented in the
  env-knobs callout (`web/src/guide/content/settings.ts`).

New test: `provider/ninjatrader/bar_persist_watchdog_test.go` (default/garbage/floor/whitespace).

## F3 — armed ledger lifecycle + log hygiene

- `store/armed_orders.go` `UpsertArm` rewritten:
  - **terminal row → re-authorized** (`armed`, reason/signal/fill lineage cleared, fresh prices) — a dead row could previously never re-arm for the same (plan, scenario);
  - **non-terminal row** → identity preserved, prices refreshed;
  - missing → create as-is.
- `trader/armed_executor.go`:
  - **1.3 gate-change cancel**: a verdict-refused scenario whose arm is `working`
    now gets its NT8 order cancelled the same cycle (`✕ armed cancel (gate changed …)`),
    before the refusal-dedup key;
  - **⚔️ armed log dedup** via `armAuthoredLast` (fires once per spec, again only when prices change) — kills the 69+ lines/day dead-row spam.
- `trader/auto_trader.go` — `armAuthoredLast map[string]string` field.

New tests: `store/armed_orders_test.go` (`TestUpsertArmReauthorizesTerminalRow`,
`TestUpsertArmPreservesNonTerminalIdentity`).

## F4 — fill-time lineage stamp becomes a pending marker

`trader/armed_executor.go` `stampArmedFillLineage`: all 4 live fills hit the race
where the position row doesn't exist yet at fill time. Now:
- ledger row → `filled`, `state_reason += ";stamp_pending"` + ONE INFO log
  (was a WARN per fill);
- `trader/ninjatrader/reconcile.go` — when `StampArmedLineageIfMatched`
  materializes the row, it completes the stamp and clears the marker.

## F5 — labels, fmt verbs, WSL2 clock

- **F5a** `kernel/levels_role.go` `KindForLabel`: `SWG-H*` → `KindSWGH`, `SWG-L*` → `KindSWGL` (were inert `KindRound`). Test: `kernel/levels_role_test.go`.
- **F5b** both wake-skip logs now use correct verbs:
  `"🗓️ level wake … SKIPPED: %.0fm elapsed < wake_min_interval_min (%dm)."` with `.Minutes()` args (was `%s`/`%d` fed a `time.Duration`/`float64`).
  Same fix for the MSS twin in `trader/auto_trader_transition.go`.
- **F5c** `deploy/fix-wsl2-clock.sh` — sized chrony install (`pool time.windows.com iburst`, `makestep 1 -1`) + 10-min root cron `hwclock -s --utc` fallback. **Owner runs it with sudo** (not run in this session).

## Test-flake repair (found during gates)

`TestMoveStopUsesMaterializedIdentity` flaked ~50% on pure `HEAD` — the TCP
server registers the dialed conn in its async accept loop and the immediate
`MoveStopToBreakeven` send raced it ("no NT client connected"). The test now
retries only that transient error for ≤1s. (Prod path unaffected; the live
bot always sends after a handshake is established.)

## Gates

| Gate | Result |
|------|--------|
| `go build ./...` | ✅ |
| `go vet ./...` | ✅ clean |
| `go test ./...` | ✅ 27 packages ok, 0 fail |
| Goldens (`-run Golden`) | ✅ 5/5 |
| Guide `vitest run src/guide/GuidePage.test.tsx` | ✅ 10/10 |
| `tsc --noEmit` | ✅ 0 errors |

## Deployment

**NOT deployed.** Parked on `fix/pre-reopen` for the owner's explicit
"go cutover". Deploy protocol at cutover: temp-clone build at merge sha →
flat-gate (DB OPEN=0, armed non-terminal=0, API positions `[]`, NT8 positions 0)
→ binary swap → `kill -9` relaunch → boot poll + goldens.

---

# CUTOVER RECORD (2026-08-28, owner-acked "GO CUTOVER")

## Merge + build
- `fix/pre-reopen` (9e4a3ae0) → `dev` merge commit **`db9245dcccbab2bdd415d2c9ff4dadaadab7e7f2`** (pushed).
- Temp-clone build at merge sha (`/tmp/nofx-cut`, clean tree): `vcs.revision=db9245dcccbab2bdd415d2c9ff4dadaadab7e7f2 · vcs.modified=false · vcs.time=2026-08-28T23:43:34Z` — matches merge sha exactly.

## Flat-gate ALL-ORIGIN (market closed — trivially flat, all four quoted)
- DB `trader_positions` status=OPEN: **0**
- NT8 truth (live 30s cadence): `tcp_server: positions snapshot account=Sim101 count=0` + `account=SimAccount1 count=0` @ 18:47:31
- API `GET /api/positions` (hoang trader): `[]`
- `armed_orders` non-terminal (armed|working): **0** (+ risk: concurrent_trades=0, kill_switch armed)

## Swap (18:49:50 CT)
- `nofx-bin` 8666db0b → `nofx-bin.prev.prereopen`; new binary rev db9245dc; `deploy/RELEASE=db9245dcccbab2bdd415d2c9ff4dadaadab7e7f2`
- `kill -9 3619700` → systemd relaunch (restart counter 84) → **PID 3747820**

## Boot block (18:49:55, PID 3747820)
- `🔐 BOOT INTEGRITY OK — rev db9245dcccba · built 2026-08-28T23:43:34Z · expected db9245dcccba · goldens PASS`
- `🛡️ regime ledger: htf_veto=ON · htf_veto_tf=1h` + structure-engine/hysteresis/freshness lines
- `🎛 volume wave: detectors=on · seats=8 · proximity=cfg(resolved per-trader; retuned 0.3) · family-confluence(cap=3) · zone-ladder=1.0/0.6/0.3/0.15 …`
- `✅ bars integrity OK: dups=0 tfs=1m total=15646`
- `🕰 clock-health [boot] … timesync{NTP=yes NTPSynchronized=yes} tolerance_ms=60000`
- `🛡 clock-guard [boot] rtc_vs_go=0s · ntp_offset=+110.553ms · resync=unavailable-no-root`

## The three knob quotes the boot block doesn't print (honest gap)
- **pool=4** — proven AT RUNTIME instead: PID 3747820 holds **3 concurrent fds on data.db** (fds 6/9/12) — physically impossible under the old `MaxOpenConns(1)`; the 4-cap comes from the merge-sha source (`store/gorm.go`).
- **watchdog** — no boot line by design (it logs only on stall); `PERSIST_STALL_WATCHDOG_S` unset → default **60s** floor 10, compiled at merge sha.
- **9-condition schema** — `scenarioConds` now **9 entries** (7 canonical + `breakdown_continue` + `breakup_continue`) at `kernel/plan_doc.go` (merge sha). No boot line. The gate's only runtime caller is the planner's AI-response parse → first live exercise is Sunday 17:00 CT plan generation; merge-sha tests green (`TestPlanDocSchemaGateAcceptsWaterfallConditions`). No active plan exists this weekend, so the API overlay path (the only deterministic in-binary surface) is gated by "no active plan to edit". **If the owner wants a deterministic in-binary schema check, a micro-wave adding a boot schema-ledger line (or a `/api/plan/schema-check` dry-run) is the clean fix — say the word.**

## arm_rr / veto / proximity intact
- `ARM_MIN_RR` unset → default **2.0** (market-entry floor 3.0 unchanged) — day-plan boot line prints at first plan gen.
- `.env:34 HTF_VETO_MODE=cross` + boot `htf_veto=ON · htf_veto_tf=1h`.
- R8 strategy (a5b7662e) live config: `proximity_filter_atr=0.3 · min_side_levels=4 · max_levels=12 · scenario_cap=5 · plan_mode=strict`.

## Post-boot
- `GET /api/status`: `is_running=true · revision=db9245dcccba · exchange=ninjatrader · roll resolved (MNQ SEP26, 21d, not blocked)`
- `GET /api/health`: `{status:ok, revision:db9245dcccba}`
- **Zero ERRO/panic/FATAL** since boot (journal scan 18:49:55→).
- Clock baseline before owner's script: `rtc_vs_go=0s · ntp_offset=+110.553ms` (healthy; script is the suspend/resume belt-and-suspenders).
