# HARDENING RUN — IN PROGRESS

**Started:** 2026-08-13 14:03:40 CDT
**Base HEAD:** 6378cdee (F5 landed)
**Principle:** construction is not verification — every hop CHECKS identity and REFUSES on mismatch, loudly. Make this month's bug classes STRUCTURALLY IMPOSSIBLE.
**Build order:** D → C → B → A (D first = instant risk reduction).
**Rules:** additive; own commit per item, pushed as you go; goldens byte-identical except deliberate wire additions; both traders may be LIVE (no restarts outside a flat/safe window; C# staged+md5 for ONE owner F5 at the end); SIM-only.

> This file is the heartbeat proving the run is alive. It is DELETED in the final commit when the run completes. A prior hardening session died silently — this prevents that.

## Part list + status

### PART D — GUARDRAILS ARMED (instant protection)
- [ ] D1 — `consecutive_loss_halt` field + entry-gate + Strategy Studio field + tests
- [ ] D2 — ARM both live strategies via API (master ON, max_daily_trades 3, daily_loss 450, max_contracts 2, daily_profit_halt 900 soft, consecutive_loss_halt 2, blackout OFF); persistence + hot-pickup proof
- [x] D3 — (F2) decouple futures size caps (notional×20 + 10-contract clamp) from master → always-on venue defaults; truthful guardrails-off log; tests ✅ committed
- [ ] D4 — (F3) pre-prompt concurrent gate reads per-strategy max_positions (env = fallback); log source; tests

### PART C — OPS ARMOR
- [ ] C1 — auto-backup systemd timer (2×/day, sqlite .backup API, prune keep-14 + weekly×8) + RESTORE.md
- [ ] C2 — clock-drift guard (>60s → block new entries + 🚨) + test
- [ ] C3 — journald persistent + SystemMaxUse=2G drop-in
- [ ] C4 — AGENT TOOLBOX block appended to root CLAUDE.md

### PART B — MACHINE ARMOR (deterministic invariants)
- [ ] B1 — stop-widen ban (tighten-only) + tests
- [ ] B2 — AI output armor (schema-strict + price sanity bounds) + fuzz tests
- [ ] B3 — duplicate-order guard + rate limiter + test
- [ ] B4 — stale-data entry block + test
- [ ] B5 — dead-man's watchdog (TCP disconnect → cancel unfilled + block) + test
- [ ] B6 — gate-block counters + API endpoint + daily journal summary
- [ ] B7 — re-entry cooldown after stop-loss + tests
- [ ] B8 — (F4) remove funding-rate from futures prompt (both emit sites); goldens updated
- [ ] B9 — (F6) INFO log when a toggle-ON prompt section renders empty
- [ ] B10 — (F7) primary_timeframe always in Available-Data announcement; golden updated

### PART A — IDENTITY GATES (wire change, lockstep)
- [ ] A1 — G3 pre-submit invariant (signal.account == boundAccount) + test
- [ ] A2 — G1 identity stamp + echo-verify (Go+C#, wire vNEXT, protocol bump) + goldens
- [ ] A3 — G2 AddOn account allowlist (C#) + error frame
- [ ] A4 — G4 post-fill verify + freeze + clear-API + tests
- [ ] A5 — G5 prompt-ownership tags + golden

### VERIFY + DEPLOY
- [ ] Full suite (build/vet/test/goldens/-race/tsc/npm) verbatim
- [ ] ONE Go restart (flat/safe window)
- [ ] C# staged + md5 → owner ONE F5 + NT8 restart (lockstep sequence stated)
- [ ] vlauto full propagation (green, secret-scan, push)
- [ ] Delete this file in the final commit

**QUEUED NEXT (do not build in this run):** F1 min-R:R redesign, Stage 0+1, levels pack, day-plan v1, replay harness.
