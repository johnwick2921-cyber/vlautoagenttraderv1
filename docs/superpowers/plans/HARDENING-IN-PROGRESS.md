# HARDENING RUN — IN PROGRESS

**Started:** 2026-08-13 14:03:40 CDT
**Base HEAD:** 6378cdee (F5 landed)
**Principle:** construction is not verification — every hop CHECKS identity and REFUSES on mismatch, loudly. Make this month's bug classes STRUCTURALLY IMPOSSIBLE.
**Build order:** D → C → B → A (D first = instant risk reduction).
**Rules:** additive; own commit per item, pushed as you go; goldens byte-identical except deliberate wire additions; both traders may be LIVE (no restarts outside a flat/safe window; C# staged+md5 for ONE owner F5 at the end); SIM-only.

> This file is the heartbeat proving the run is alive. It is DELETED in the final commit when the run completes. A prior hardening session died silently — this prevents that.

## Part list + status

### PART D — GUARDRAILS ARMED (instant protection)
- [x] D1 — `consecutive_loss_halt`: backend (config field + store query + entry-gate + store test) ✅ + Strategy Studio FE field (GuardrailRow, futures-visible, i18n zh/en/es, tsc+build green) ✅ committed
- [~] D2 — ARMED in DB (guarded write, backup data-20260813-1431-pre-D2-arm.db; both strategies: master ON, daily_loss 450, max_daily_trades 3, max_contracts 2, daily_profit 900 soft, consecutive_loss_halt 2, blackout OFF); persistence PROVEN. Hot pickup NOT achievable (config cached — running old binary still logs guardrails-DISABLED at 14:38:49 post-write [A]) → **RESTART-REQUIRED**; activates at the deploy restart. API hot-reload declined (would need a minted owner token to reload sacred live config; rail = never force).
- [x] D3 — (F2) decouple futures size caps (notional×20 + 10-contract clamp) from master → always-on venue defaults; truthful guardrails-off log; tests ✅ committed
- [x] D4 — (F3) pre-prompt concurrent gate reads per-strategy max_positions (env = fallback); log source; tests ✅ committed

### PART C — OPS ARMOR
- [x] C1 — auto-backup USER systemd timer (no sudo; linger=yes so it fires logged-out). deploy/nofx-db-backup.sh (python3 sqlite3.backup() online API — no sqlite3 CLI; integrity-checked; gzip; daily/ keep-14 + weekly/ keep-8 ISO-week promotion), deploy/systemd-user/nofx-backup.{service,timer} (OnCalendar 05:00+17:30 — host is America/Chicago so native CT; Persistent=true), deploy/install-db-backup.sh, deploy/RESTORE.md. INSTALLED+ENABLED (next Fri 05:00 CDT) + RAN ONCE NOW (402MB→34MB gz @ ~/nofx-backups/auto/daily + weekly promoted; Result=success). READ-BACK TESTED: gunzip→quick_check ok, 19 tables schema-identical to live, core rows readable (decisions 28042/pos 516/strat 9/exch 1). RESTORE.md also copied to ~/nofx-backups/. Backups are OUTSIDE the repo (not committed).
- [x] C2 — clock-drift guard: kernel/clock_drift.go applyClockDriftBlock (called after B4). Measures local-vs-feed skew as nowMs-(freshestBar+interval) [bar labeled at open → ~0 under correct clock+live feed]; |drift|>60s → neutralize open→wait + 🚨 + telemetry "clock_drift". Catches BOTH clock-ahead/feed-lag AND clock-behind/future-dated-bar (which B4 cannot). Fail-open no-data; exits never blocked; strict-> boundary. Injected-skew test (±5min both directions block, 60s boundary passes, close never blocked, no-data fail-open) PASS. Not deployed (running rev unchanged).
- [~] C3 — journald persistent + SystemMaxUse=2G drop-in. FINDING: /var/log/journal already exists → journald ALREADY persisting; gap is the CAP (journal at 3.9G, no SystemMaxUse → defaults). Staged deploy/journald-nofx.conf ([Journal] Storage=persistent + SystemMaxUse=2G) + deploy/install-journald.sh (mkdir drop-in, install to /etc/systemd/journald.conf.d/nofx.conf, restart systemd-journald, --vacuum-size=2G, verify --disk-usage). ROOT-ONLY (/etc + journald restart) → OWNER runs `sudo bash deploy/install-journald.sh` (the one C3 sudo step; autostart precedent). Baseline recorded: --disk-usage=3.9G pre-cap.
- [x] C4 — AGENT TOOLBOX block appended to root CLAUDE.md (additive "Agent toolbox & standing rules (hardening C4)" section: evidence tiers A/B/C, SIM-only, guarded DB writes + config-cache caveat, deploy=kill-9, no-sudo ops surfaces [backup timer / gate-blocks endpoint / owner-gated journald+autostart], FORBIDDEN 5 GDrive tools, vlauto format-patch flow, repo target). No existing content changed.

### PART B — MACHINE ARMOR (deterministic invariants)
- [x] B1 — stop-widen ban: MoveStopToBreakeven refuses a risk-increasing stop (pure stopWouldWiden long/short, cur-stop tracked + updated on move); tests long/short/equal + widen-refused ✅
- [x] B2 — AI output armor COMPLETE: B2a bounded-retry (schema_parse_failed skip) + B2b price-sanity (one authority, >8×ATR15/entry>1%, neutralize→wait, fail-open) + B2c fuzz (162k parse + 416k sanity execs, 0 crashes); goldens byte-identical ✅
- [x] B3 — dupe guard (idempotence key account|side|symbol|qty within ~1 bar → drop ⛔) + rate breaker (>10 order-actions/min → halt 🚨) at placeEntry chokepoint; orderGuard.admit tested (dedup + breaker, both recover) ✅
- [x] B4 — stale-data entry block: applyStaleDataBlock neutralizes open_long/short→wait when freshest 1m/5m bar > 2×interval (pure barIsStale/staleEntryFeed); exits+position-mgmt untouched; no-data fail-open; tests fresh/stale/close/no-data ✅  → **B1-B4 ALL AT HEAD → F1 UNBLOCKED**
- [x] B5 — dead-man watchdog: pure state machine (dmLive→disconnect→dmDisconnected→reconnect→dmAwaitingReconcile→reconcile→dmLive; entriesBlocked until clean reconcile) + WIRED into runCycle (driveDeadManWatchdog observes raw TCP IsConnected each cycle) + entry gate in executeDecisionWithRecord blocks open_long/open_short when entriesBlocked() + reconciled() driven off a clean GetPositions+GetOpenOrders probe (one-cycle-deferred after reconnect). step() driver + both-paths/flap tests PASS. HONEST NOTE: "cancel unfilled entries" is a documented no-op hook on the current market-order NT8 path (CancelAllOrders/GetOpenOrders unsupported/empty → cancel wire command = Part A). Not deployed (running rev unchanged).
- [x] B6 — gate-block counters: telemetry per-trader/CME-session-day registry (IncGateBlock/RolloverGateBlocks/GateBlockSnapshot/GateBlockDailySummary) wired at EVERY gate — trader layer (feed_down, dead_man, consecutive_loss @ at.id), kernel (task18_cme_closed, task19_contract_roll, task21_concurrent_cap, strategy_studio_daily/blackout/consistency, task22_drift, price_sanity=B2, stale_data=B4 @ ctx.TraderID [new Context field]), B3 order guard (b3_order_dedup/b3_rate_breaker @ process-wide "" bucket). ONE endpoint GET /api/risk/gate-blocks (table + summary). Daily journal summary line logged at 17:00 CT rollover (first trader across the edge). Test: trip-two-gates-show-exactly-them + process-wide bucket + rollover-clears-and-summarizes (all PASS). Not deployed (running rev unchanged).
- [x] B7 — re-entry cooldown: new leaf pkg discipline/ (NoteStopLossExit + ReentryBlocked pure fn; blocked = within cooldown AND price hasn't moved ≥1×ATR15 from stop; unlock on EITHER, whichever first; per (trader,symbol,side); clears record on unlock). TRIGGER: close_sync.recordClose arms it on NT8 ExitReason=="sl" @ owner.TraderID + SL fill price. GATE: kernel applyReentryCooldown neutralizes same-dir open→wait (0=OFF; fail-open no-data; atr15<=0 → timer-only), increments telemetry gate "reentry_cooldown". CONFIG: store RiskControlConfig.ReentryCooldownMinutes (0=off; FE default 20; not master-gated). FE: RiskControlEditor futures-only GuardrailRow + strategy.ts field + strategy-translations reentryCooldown (zh/en/es). Tests: time-unlock + price-move-unlock + isolation(dir/symbol/trader/0=off) + zero-ATR-timer-fallback (all PASS); web build green. Not deployed (running rev unchanged).
- [x] B8 — (F4) funding-rate suppressed on futures (both emit sites via isFuturesInstrument); test absent-on-futures/present-on-crypto; goldens byte-identical (none asserted funding) ✅
- [x] B9 — (F6) INFO log when SVP toggle-ON but omitted (live path engine_analysis.go, no preview noise). SCOPE: per-TF/array line-skips deferred (no clean live-path spot; 📊 Strategy timeframes already lists requested TFs) ✅
- [x] B10 — (F7) primary_timeframe always announced in Available-Data (formatKlineTimeframes appends it when absent from selected); 6 goldens byte-identical + new F7 case ✅

### PART A — IDENTITY GATES (wire change, lockstep)
- [ ] A1 — G3 pre-submit invariant (signal.account == boundAccount) + test
- [ ] A2 — G1 identity stamp + echo-verify (Go+C#, wire vNEXT, protocol bump) + goldens
- [ ] A3 — G2 AddOn account allowlist (C#) + error frame
- [ ] A4 — G4 post-fill verify + freeze + clear-API + tests
- [ ] A5 — G5 prompt-ownership tags + golden

### VERIFY + DEPLOY
- [ ] Full suite (build/vet/test/goldens/-race/tsc/npm) verbatim
- [x] Go restart DONE 2026-08-13 14:53:02 CDT (SIGKILL→systemd relaunch; PID 190403 @ rev 74aac5b6). Part D now LIVE: D3/D4/D1 code active + D2 armed config loaded — guardrails master ON ENFORCING (max_daily_trades=3 tripping on both traders @14:56:12 [A], today=5/6). Both traders re-bound to own accounts, id=519 re-attached, wrong-account=0. NOTE: max_daily_trades=3 halts NEW entries for the rest of this session-day (both already at 5-6 trades) — intended protection; resumes at 17:00 CT rollover.
- [ ] C# staged + md5 → owner ONE F5 + NT8 restart (lockstep sequence stated)
- [ ] vlauto full propagation (green, secret-scan, push)
- [ ] Delete this file in the final commit

**QUEUED NEXT (do not build in this run):** F1 min-R:R redesign, Stage 0+1, levels pack, day-plan v1, replay harness.
