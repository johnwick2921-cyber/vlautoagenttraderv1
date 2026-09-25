# 2026-08-18 — P0-A cross-trader governance + P0-B ASIA clock + P1 H8 residuals

**LINE 1:** Was any decision governed by the wrong trader's plan? **YES — exactly 3** (the `15m` trader's decision cycles since 08-14 carried `hoang`'s plan block in the prompt; all 13 stored plan rows belong to hoang, 15m wrote none). Also documented live at 08:31 today by the watch session (both traders' prompts got the NY plan via the global provider). Now impossible by construction. Commits: `99fd67e1` (P0-A) · `1e8cc591` (P0-B) · `91748082` (P1), pushed; binary built at `91748082`, `deploy/RELEASE` armed. Restart + hard reload are the owner's steps.

## P0-A — cross-trader governance

- **Store:** `GetLatestPlanForTraderSession(trade_date, session, strategy_id)`; every production reader converted — read dedupe/death, executor provider, re-read, reset, matched-random, card/overlay/ask/realign, versions (`ListVersionsForTrader`), history (`ListRecentForTrader`), scenario-status key. Source-scan test fails if any production file uses the unscoped lookups.
- **Kernel:** the three global singletons (`ActivePlanProvider`, `SessionRegistryProvider`, `PlanProximityKProvider`) are DELETED. Providers register per-trader (`SetTraderPlanProviders` / `ActivePlanFor` / `ResolvedSessionRegistryFor`, keyed by `ctx.TraderID` in the executor prompt). The day-trade proximity k now comes from the deciding engine's own strategy config — no global to capture. A source-scan test fails if the removed globals or a `sync.Once` provider install reappear; the trader layer logs loudly when >1 day-plan trader is registered (the runtime half of the startup assertion; the compile-time half is the globals' absence).
- **Tests:** two-trader isolation at the store seam; provider map (B with no plan receives NOTHING); per-trader install; scenario-status key scoping; three source-scan guards. `go test ./...` + `-race` (kernel/trader/store/api) green.

## P0-B — ASIA clock

- **(1) 16:55 read unreachable:** the read gate tested `IsCMEOpen(now)`; 16:55 sits in the 16:00–17:00 CME break (live receipt: first ASIA read fired 17:12). The gate now tests the SESSION INSTANCE'S OPEN via `kernel.SessionInstanceStart` (wrap-aware), so the 16:55 read fires from stored data; Sunday-17:00 sessions read, Friday/holiday instances don't, and the death-check still runs through the wrapped tail.
- **(2) midnight double-plan:** the chain identity was the midnight-roll date. `kernel.PlanChainTradeDate` maps a moment to the session INSTANCE's CT date (pre-open gap → the instance about to open; wrapped tail → yesterday's instance) and is applied at the read scheduler, provider, re-read, reset, matched-random, session digests, and the card/overlay/ask/versions/thread handlers.
- **Tests:** chain-date table (16:55/17:00/22:00/00:30/01:59/02:00, NY 08:25/12:00/15:00); the 16:55 read fires while the market is closed; 16:30 does not; Sunday 16:55 fires; the 00:30 cycle writes NO second plan for the same instance.

## P1 — H8 residuals (api/handler_plan.go)

Sites `:156, 670, 918, 1271, 1740` converted from the raw registry flag to the trader's exported `SessionRunnable(sess)` (the bot's own resolver), with a nil-manager-safe default. A source-scan test fails if any `api/` production file reads `sess.Enabled` again.

## P2 — report-only (the unsupported prior claims)

`TestB3PlanCannotBypassAnyGate`, `TestB4PlanDocHasNoRiskFields`, `TestC3FeedToStoreToGateRefusal` **are absent from the repo** (grep across all `*.go`; only per-gate tests exist: `w14_boot_gate`, `dead_man_watchdog`, `w9_plan_mode`, hold-lock, `w16_gate_refusal`). The checklist-run-1 rows B3/B4/C3 ("ALL PASS") referenced fixtures that lived only in isolated /tmp copies and were never committed — those specific "PASS" claims are not reproducible from the repo. The gate chain itself was re-verified structurally at HEAD.

## Exit bar

`go build`/`vet`/`test ./...`/`-race` (kernel/trader/store/api) green · `tsc` clean · vitest 243/244 (the same pre-existing `RegistrationDisabled` logo failure + `e2e/gate.spec.ts` collection error — zero FE files touched this session) · goldens untouched (`git diff 6dcee05a HEAD` shows no testdata changes) · config-truth: no config schema changed; the enforcement paths read the same persisted fields (proximity now directly from the deciding engine's config).

## Deploy handoff (mandatory order, executed)

`git pull` (pushed `6cc9ce11..91748082` first) → `go build -o nofx-bin .` ✅ → `git rev-parse HEAD > deploy/RELEASE` ✅ (`91748082…`) → **owner: `sudo systemctl restart nofx`** → `cd web && npm run build` ✅ + hard reload. Restart at a flat/safe window only; verify with the BOOT INTEGRITY line (expected == rev).
