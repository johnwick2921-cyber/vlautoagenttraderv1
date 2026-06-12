# MONITORING — nofx NQ Futures Bot

What to watch, where it lives, and what should fire an alert. See plan doc Task 32.

## 1. Key dashboards (Grafana / Prometheus)

> The Prometheus metrics referenced below are part of Plan 4 and **not yet shipped**. Treat this section as the target wiring once `nofx_*` metrics are exposed.

- **Decision rate** — `rate(nofx_decisions_total[5m])` — expected ~1/scan_interval (default 60s). Drop to zero = engine stalled.
- **Fill latency** — `histogram_quantile(0.95, nofx_fill_latency_seconds)` — alert if **p95 > 30s**. CSV bridge with NT polling every 2s should never exceed ~5s under normal load.
- **Databento errors** — `rate(nofx_databento_errors_total[10m])` — alert if **> 0.1/sec** sustained. Burst spikes (auth blip) are tolerable.
- **Daily PnL** — read from DB; alert if `SUM(realized_pnl) < -$500` for today. Matches `RiskMaxDailyLossUSD` default ($500). See §2 below.
- **CME session health** — alert if a trade attempt is logged during the daily 16:00–17:00 CT maintenance break. Source-of-truth: `kernel.IsCMEOpen(time.Now())` returns false during that window.

> TODO when Plan 4 ships: register the four `nofx_*` Prometheus collectors and expose `/metrics`.

## 2. Alert thresholds matched to Plan 3 risk limits

| Metric | Threshold | Source / env |
| --- | --- | --- |
| Daily PnL | < -$500 | `RISK_MAX_DAILY_LOSS_USD` (default 500) |
| Concurrent positions | >= 2 | `RISK_MAX_CONCURRENT_TRADES` (default 2) |
| Total notional | > $50,000 | `RISK_MAX_NOTIONAL_USD` (default 50000) |
| Single order size | > 5 contracts | `RISK_MAX_CONTRACTS_PER_ORDER` (default 5) |
| Bar staleness (RTH) | > 90s | `market.MaxAgeRTH` (DefaultFreshnessConfig) |
| Bar staleness (ETH) | > 5min | `market.MaxAgeETH` |
| Drift | > 5% in < 60s | `market.DriftThresholdPct` + `market.DriftWindow` |

All four `RISK_*` env vars are loaded by `kernel.LoadRiskLimitsFromConfig()` (see `kernel/risk_limits.go:100`). The `RiskMaxDailyLossUSD` threshold trips a **ForceFlat** classification (close everything) rather than a soft BlockEntry — review the log carefully.

## 3. Manual checks (no alerting infra yet)

Until Plan 4 ships, lean on `tail` + `curl` + `sqlite3`:

```bash
# General error / warning stream
tail -f /tmp/nofx.log | grep -E "ERROR|WARN"

# Plan 3 risk + drift gate trips specifically
tail -f /tmp/nofx.log | grep "Plan 3 T2[12]"

# TODO when Plan 4 ships: live risk status snapshot
curl localhost:8080/api/risk/status

# Manual daily PnL check (UTC-day; switch to CT once session-aware reset lands)
sqlite3 data/data.db \
  "SELECT SUM(realized_pnl) FROM positions WHERE DATE(closed_at)=DATE('now');"
```

The manual SQL above should track the alert threshold — if it returns a number less than -500, the next decision cycle will trip the daily-loss force-flat gate.

## 4. Log patterns to watch for

These are the most operationally important Plan 3 log lines. Each is grep-able from `/tmp/nofx.log`.

- `🔴 Plan 3 T21 FORCE-FLAT invoked` — kill switch fired (daily-loss limit hit). Audit before resuming.
- `⚠️ Plan 3 T21 risk gate tripped` — entry blocked by a non-fatal limit (concurrent / notional / per-order). Existing positions held.
- `⚠️ Plan 3 T22: stale data` — latest bar older than `MaxAgeRTH` / `MaxAgeETH`. New entries blocked until feed recovers.
- `⚠️ Plan 3 T22: suspicious drift` — > 5% close-to-close move in < 60s. Possible limit move, bad print, or feed glitch.
- `Plan 3 T21: daily PnL window reset` — confirmation that the day rolled over and the limit window restarted.

## Cross-references

- `kernel/risk_limits.go` — risk env loading + decision classification.
- `kernel/cme_calendar.go` — `IsCMEOpen` (drives the 16:00 CT maintenance-break alert).
- `market/data_freshness.go` — `DefaultFreshnessConfig` (90s / 5min / 5% / 60s).
- `config/config.go:60-63` — `RiskMax*` field declarations.
- Plan doc Task 32 — source of this runbook.
