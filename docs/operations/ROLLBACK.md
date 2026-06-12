# ROLLBACK — nofx NQ Futures Bot

Operator runbook for reverting code, schema, NT scripts, and risk state after a bad deploy or unexpected loss. See plan doc Task 31.

## 1. Code rollback

The repo carries 3 release tags marking known-good points plus 1 pre-merge safety tag:

| Tag | What it contains |
| --- | --- |
| `v1.0-plan1` | Plan 1 CSV bridge only. No risk limits, no CME calendar, no drift detection. Earliest safe rollback. |
| `v1.0-plan2` | Plan 1 + CME domain (cme_calendar, IsCMEOpen, holiday + session gates). |
| `v1.0-plan3` | Plan 1 + Plan 2 + risk limits + drift detection (current live). |
| `pr3-before-merge-2026-05-25` | Safety tag — snapshot immediately before PR #3 merged. Use only if PR #3 itself is the suspected regression. |

To roll back the working tree to a tag, then rebuild:

```bash
git checkout v1.0-plan2 -- .
go build -o nofx-bin .
pkill -TERM -f nofx-bin
./nofx-bin > /tmp/nofx.log 2>&1 &
```

Verify the rolled-back binary boots cleanly (`grep "System started successfully" /tmp/nofx.log`) and serves `/api/traders` (see STARTUP.md §3).

> Note: `git checkout <tag> -- .` rewrites tracked files in place. It does NOT change the branch HEAD. To formally pin to a tag, follow with `git switch --detach v1.0-plan2`.

## 2. Schema rollback

GORM auto-migrate is **additive only** — it adds columns but never drops them. So a code rollback does not corrupt the DB, but a bad migration can leave orphan columns. Before any code deploy:

```bash
cp data/data.db "data/data.db.bak.$(date -u +%Y%m%dT%H%M%SZ)"
```

To restore from a backup:

```bash
pkill -TERM -f nofx-bin
cp data/data.db.bak.20260525T120000Z data/data.db
./nofx-bin > /tmp/nofx.log 2>&1 &
```

For column **removal**, write a one-shot migration file under `store/migrations/`. Auto-migrate will not do it.

## 3. NT script rollback

If `vltrader.cs` (renamed from `claudetrader.cs` on 2026-05-23) was changed and the new build misbehaves:

```bash
git log -- '**/vltrader.cs' '**/claudetrader.cs'
git checkout <prior-good-commit> -- '**/vltrader.cs'
```

Then in NT8:

1. NinjaScript Editor → open `VLTrader.cs`.
2. Press **F5** to recompile.
3. Re-attach the strategy to the MNQ chart (Strategies tab → Add → VLTrader).

The CSV protocol is unchanged across the rename — 5-field signals, 3-field fills.

## 4. Risk wipe — force-flat before resuming

If the rollback follows an unexpected loss, **do not let the bot place new trades until you have manually verified zero open positions**.

The `POST /api/risk/force-flat` REST endpoint is deferred (see Plan 3 follow-up / Plan 4). For now:

- **Preferred (manual)**: in NT8, right-click the chart → Strategies → disable VLTrader, then close any open position via the Orders window.
- **Programmatic** (advanced): write a small Go entrypoint that imports `kernel` and calls `kernel.MaybeForceFlat(...)` against the active trader. Wire the trader handle from `store/` lookups.

Target endpoint once shipped:

```bash
curl -X POST localhost:8080/api/risk/force-flat \
  -H "Authorization: Bearer $TOKEN"
```

> TODO when Plan 4 ships: replace the manual NT-side flatten with the REST endpoint above.

## 5. Daily PnL window reset

After rollback, the new process starts with `lastDailyResetDate` empty. The window auto-resets on the first decision cycle via `kernel.MaybeResetDaily(time.Now())` (UTC-date rollover). To force a reset manually right now, invoke `kernel.ResetDailyPnL()` from a small one-shot. See `kernel/risk_limits.go:142-150`.

After reset, the log will show: `Plan 3 T21: daily PnL window reset at YYYY-MM-DD UTC`.

## Cross-references

- `kernel/risk_limits.go` — `ResetDailyPnL`, `MaybeResetDaily`, `RiskLimits.Classify`.
- Plan doc Task 21 — risk limits + force-flat design.
- Plan doc Task 31 — source of this runbook.
