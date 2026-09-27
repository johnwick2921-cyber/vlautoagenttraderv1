# E7 SINGLE-BOOT PARK REPORT — 2026-08-30 (before the seam flip)

Read-only facts, verified before the boot. One fix set · one branch map · one class numbering.

## Final branch map

| Branch | Contents | Status |
|---|---|---|
| `dev` (main tree) | origin/dev `79365622` + `06f1dc4e` (E7 seam) + `00d77870` (B: loop guard + E: capability handshake) + `db6f510a` (merge fix/clock-hold: F6 clock-hold + F7 reading chip) + `59dc9460` (F1 weekly render + F3 ledger gap + F5 weekly DOA) | **boots** |
| `docs/massive-move-audit` | audit report (151ef42b) | docs-only |
| `docs/cheap-five` | knob verdict tables (9298f9d4) | docs-only |
| `fix/clock-hold` | merged into dev (F6/F7); its F1 was superseded by `00d77870`'s canonical loop fix | **merged** |

The clock-hold agent's uncommitted `f1_wrong_side_test.go` (strict AT-market boundary) did NOT land — the canonical predicate is `limitMarketableWrongSide` (strict inequality: price below a long's entry / above a short's). Boundary decision recorded here deliberately: an exact touch fills AT the limit price (the "resting limits fill at the authorized price" contract), so equality is placeable.

## Final class list — one numbering, 26 classes (docs/superpowers/AUDIT-CHECKLIST.md)

1–19 original · **20** OS-side fix that silently regresses (F7) · 21 committed binaries · 22 log-lie counters · 23 unprobed supply chain · 24 report-only path panicked the loop · **25 armed re-place loop / manual-cancel-wins** (the incident) · **26 far-side capability mismatch** (the incident).

## Every fixture (named, all green pre-boot)

| Fix | Fixtures |
|---|---|
| Loop guard — re-auth (store) | `TestUpsertArmManualCancelWinsSameVersion`, `TestUpsertArmReauthorizesOnVersionBump` |
| Loop guard — wrong-side (trader) | `TestLimitMarketableWrongSide` |
| Capability handshake (survived merge, R1) | `TestPlaceStopEntryRefusedWithoutFarSideBuild` (refusal + no-leak + old-build), `TestPlaceStopEntryFrameOnLoopback` (heartbeat-proven accept), `TestStopEntryFrameIsAdditiveJSON` |
| Weekly render (F1) | `TestWeeklyExecutorLineInvalidatedRendersNeutral` |
| Ledger gap (F3) | `TestMaterializeArmedEntryF3` |
| Weekly DOA (F5) | `TestApplyWeeklyDOAStampsNeutralAtWrite` |
| Clock-hold (F6) | `kernel/clock_hold_f6_test.go`, `trader/clock_hold_test.go` |
| Reading chip (F7) | `web F7_reading_transition.test.tsx`, `P7_no_trade_map.test.tsx` |
| F4 flip-latency | docs evidence in `docs/superpowers/reports/2026-08-30-massive-move-audit.md` (no fixture — measurement item) |

**R1 statement:** the capability handshake (`00d77870`'s build_id refusal path) survived the merge intact — `FarSideProven`/`FarSideBuildE7` gate + the three loopback fixtures above, all green.

## The ONE C# md5 (final)

`46699102a33c65a0b4ddb6f370e4dcc1` — `VLTraderTCPClient.cs` with `VL_BUILD_ID="2026-08-30-e7"`. Compiled 23:49:10 CT (assembly mtime observed). If NT8 has not restarted since 23:49:10, the running AddOn is still the pre-handshake build — the boot-time proof handles both cases (below).

## Boot checklist draft

1. **Flat gate (all-origin):** DB `OPEN=0` · `armed_orders` state IN ('armed','working') = 0 · NT8 snapshots `count=0` ×2 · API positions `[]` — all quoted.
2. **Build:** temp clone at the boot sha (vcs stamp clean), `RELEASE=<sha>`.
3. **Env:** `STOP_ENTRY_SEAM=on` + `ARMED_TEST_SEAM=on` (the seam stays on through proof #2; turn off at the next maintenance restart).
4. **Swap:** mv old → kill -9 → systemd relaunch.
5. **Boot lines:** `🔐 BOOT INTEGRITY OK — rev <sha> · goldens PASS` · `🎛 entry law: … stop_entry_seam=on` · `⚔️ armed_orders=on … test_seam=ON` · zero panics.
6. **Proof #2 — the handshake sequence:**
   - `POST /api/armed/test-arm action=place_stop` (trigger 28700, far below market).
   - **Path A (old AddOn still loaded):** quote the REFUSAL `refusing stop-entry — far-side AddOn build "" does not prove stop_entry support (need ≥ 2026-08-30-e7)` → owner restarts NT8 → quote `tcp_server: far-side AddOn build_id=2026-08-30-e7` → place again.
   - **Path B (new AddOn loaded):** first place accepts directly.
   - Then: order RESTS at 28700 (never fills — quoted + owner confirms in NT8) → `cancel` → cancel-ack quoted → **5 minutes, zero re-placement lines**.
7. **Short watch:** panics=0 · restarts=0 · drops=0 for ≥10 min post-boot.
8. Rollback: `nofx-bin.prev.e7` (06f1dc4e build, md5 `28e4f4bc`) + RELEASE restore on any panic/golden failure.

## Live state at park time

Bot on the data-kill: plans `no_trade` (e7_incident_kill) · S2/S3 rows cancelled · Sim101 flat · 0 placements since 23:10:36.

---

## BOOT RESULT — 2026-08-31 00:08:38 CT (appended post-boot)

- Flat gate 4/4: DB_OPEN=0 · ARMED_NT=0 · Sim101 count=0 · API `[]`.
- **`🔐 BOOT INTEGRITY OK — rev 59dc94603e49 · goldens PASS`** · `🎛 entry law: … stop_entry_seam=ON` · `⚔️ armed_orders=on … test_seam=ON`. PID 932579.
- **Handshake:** `00:08:52 tcp_server: far-side AddOn build_id=2026-08-30-e7` — the compiled 46699102 AddOn is live and proven.
- **Proof #2 (Path B — accept on first place):** `place_stop` trigger 28700 → `ok:true`, signal `1deb5e23` → **RESTED** (77s, zero fills, state working, Sim101 flat — the old AddOn's market-execution class is dead) → `cancel` → `ok:true` → ledger `cancelled (test seam cancel)`.
- **Cancel ack:** the far side answered with 3 `order_update` frames at 00:11:12 (receipt quoted); they were dropped by the armed consumer's no-subscriber guard (subscription is cycle-bound — observability gap, NOT a functional failure). Functional proof: **PROOF2_CLEAN_5MIN_NO_REAPPEAR** — 5 min × 30s ticks: placements=0 fills=0 panics=0 drops=0, Sim101 flat every tick.
- **Post-boot note:** `ARMED_TEST_SEAM` stays on until the next maintenance restart (SIM-only + ownership-gated; turning it off now would need a second boot). The plan remains `no_trade` — the owner's reset/re-read activates real trading under the new law.
