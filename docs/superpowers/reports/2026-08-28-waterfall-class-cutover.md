# Waterfall-Class Cutover Record + Post-Click Verify (2026-08-28, 15:22:35 CT)

- **Deployed rev:** `8666db0b` · **merge sha (S-4):** `8f5734f8` (`fix/waterfall-class e902253c` merged into dev) + `8666db0b` = merge + recovered `bf841d37` docs commit · **RELEASE marker:** `daa5a681` (dev tip).
- **Owner:** GO CUTOVER given; clicks confirmed pre-cutover (`0.3 | 60`, 11:59:00 CT).

## 1. Main-tree reconciliation (nothing lost)

- 38-file GAR WIP found uncommitted in the main tree → snapshotted to branch **`wip/main-gar-2026-08-28`** (commit preserved, not in this deploy).
- Mid-flight discovery: local `dev` had been reset to `2850e351` (concurrent interference in the main tree), so the initial merge dropped `bf841d37` (post-click verification report). Recovered via cherry-pick onto the merge → final build sha `8666db0b` contains it. Nothing lost: WIP branch + both docs commits exist.

## 2. Flat-gate (all four origins, 15:2x CT)

- DB: `trader_positions OPEN = 0` · `armed_orders non-terminal (armed/working) = 0`
- API positions: `[]`
- NT8: `positions snapshot account=Sim101 count=0` · `positions snapshot account=SimAccount1 count=0`
- Nothing working → proceeded.

## 3. Boot block (15:22:35 CT, PID 3532738)

```
🔐 BOOT INTEGRITY OK — rev 8666db0b6c89 · built 2026-08-28T20:20:13Z · expected 8666db0b · goldens PASS
⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off arm_rr=2.0 (gate-at-arm only; market-entry floor 3.0 unchanged) (resting limits fill at the authorized price; stale_reeval NOT applied)
🛡️ htf veto: mode=cross tf=1h (1h|cross|4h via HTF_VETO_MODE; cross = 1h AND 4h agree)
🩹 move_stop identity: materialized positions persist armed-ledger signal_id → entry_order_id; move_stop/trailing resolve it (GAR-F1)
📐 planner cap: plan_max_tokens=65536 (AI_PLAN_MAX_TOKENS; default 65536) · truncation → 🚨 WARN, never silent
📜 planner playbook: playbook=v2 … (unchanged, ALL ADVISORY)
```

Post-boot: `/api/status` revision `8666db0b6c89` running ✓ · positions `[]` ✓ · **0 ERRO/panic** since boot ✓.

**Honest note on the two expected lines:** the waterfall wave's "8th-condition" is a BASE-CONTRACT change in the futures prompt (rendered at read time) and "fast-market" is wake-runtime observability (`reasoning=fast` + FAST TAPE on wake reads) — **neither is a boot line**. Their live fire = the ASIA 16:55 read / next wake, captured in the 17:00 window below (pending at report time).

## 4. Post-click verify (clicks landed pre-cutover; binary honors them)

- **DB (quoted):** `0.3 | 60 | 2026-08-28 11:59:00` (strategy `a5b7662e`, fresh `updated_at`).
- **Seated line (verbatim, captured live on ae8b04b5 — same `ResolveProximityK` code the new binary carries):**
  `🗺️ seated 24/191 in-band levels (proximity band ±85pt, 24 of them retained)` and
  `🗺️ seated 12/170 in-band levels (proximity band ±85pt, 12 of them retained)`
  — band = 0.3 × DailyRangeProxy(≈283pt) = ±85pt, inside the predicted ±91–110pt envelope. First seated line from the NEW binary fires at the ASIA read (~17:05) — pending.
- **Resolver 0.3 on BOTH consumers:** bot gate/watcher `trader/auto_trader_planconfig.go:139` → `kernel.ResolveProximityK` (`kernel/plan_lifecycle.go:25-30`), live value proven by the ±85pt band; engine prompt path `kernel/engine_analysis.go:374` → the same `ResolveProximityK` (0.5 floor removed).
- **min_conf = 60, zero 65 residue:** gate `kernel/engine_analysis.go:550` + `kernel/engine_position.go:197` + futures prompt `kernel/engine_prompt_futures.go:63` (fallback `store.SafeDefaultMinConfidence = 60`, `store/strategy.go:81`). Owner ruling stands: 65 deferred; the 60–64 band judged by Sep-9 data at real n.

## 5. 17:00–17:10 CT window — PENDING at report time

- Nightly `level_stats` solo #2 (evaluates 08-27) — the per-trader nightly line.
- Globex reopen: ingest summary `peak_depth<4096` + `closes_dropped=0`.
- First planner read on the new binary: `cap=65536` + seated line ±85pt + the 8th-condition contract + any FAST TAPE wake line.
- FIX-1 session-end cancel-first wire proof if any arm is working at a boundary.
