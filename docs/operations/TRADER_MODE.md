# TRADER_MODE — User Runbook

Written for the **user** who is actively trading the bot, not for engineers. Switch modes, do the daily checklist, hit the kill switch if things go wrong. See plan doc Task 34.

## 1. How to switch from SIM to live

**WARN**: before flipping to a real-money account, confirm `RISK_MAX_DAILY_LOSS_USD` is set to a number you are comfortable losing in one day. The default ($500) may be wildly off for your risk tolerance.

Step by step:

1. In NinjaTrader 8: **Control Center → Connections** → switch from the simulation connection to the live (real-money) connection.
2. Stop the trader bot:
   ```bash
   pkill -TERM -f nofx-bin
   ```
3. Update `.env` if you want a tighter loss limit (e.g. $100 instead of $500):
   ```bash
   # in .env
   RISK_MAX_DAILY_LOSS_USD=100
   ```
4. In NT: re-attach `vltrader.cs` (renamed from `claudetrader.cs` on 2026-05-23) to the chart of your real-account contract (e.g. `MNQH6` for the March 2026 contract — pick whichever quarter is the current front-month).
5. Restart the bot:
   ```bash
   ./nofx-bin > /tmp/nofx.log 2>&1 &
   ```

## 2. Daily checklist (before RTH open at 08:30 CT, weekdays)

Run through all six items. If any fail, do not enable live trading.

- [ ] NT8 connected: Control Center → green dot
- [ ] VLTrader strategy enabled: chart toolbar shows green play icon
- [ ] Bot logs healthy:
      ```bash
      tail /tmp/nofx.log | grep "started successfully"
      ```
- [ ] Daily PnL window reset for today:
      ```bash
      grep "Plan 3 T21: daily PnL window reset" /tmp/nofx.log | tail -1
      ```
      (last reset should be today's UTC date)
- [ ] Risk limits configured:
      ```bash
      grep RISK_ .env
      ```
      (expect 4 lines: daily loss / concurrent / notional / per-order)
- [ ] Front-month contract matches NT chart symbol (e.g. `MNQH6`, `MNQM6`, `MNQU6`, `MNQZ6`)

## 3. Weekly checklist

- **Contract roll** — index futures expire on the 3rd Friday of quarterly months (Mar / Jun / Sep / Dec). Plan 2 Task 19 (`kernel.databento.ShouldBlockEntryForExpiry`) blocks new entries starting 5 days before expiry. Verify there are no false positives by spot-checking:
      ```bash
      grep "ShouldBlockEntryForExpiry\|expiry" /tmp/nofx.log | tail -20
      ```
      Roll your NT chart to the next quarter when the block fires (e.g. `MNQH6` → `MNQM6`).
- **Decision audit trail review** — eyeball the last week of decisions in the DB:
      ```bash
      sqlite3 data/data.db \
        "SELECT created_at, action, symbol, entry, stop_loss, take_profit FROM decisions \
         WHERE created_at > datetime('now', '-7 days') ORDER BY created_at DESC LIMIT 50;"
      ```
      > TODO when Plan 4 ships: replace this with `/api/audit/decisions` and a UI tab.
- **API key rotation**:
      ```bash
      # Databento: log into databento.com → API Keys → regenerate
      # Update DATABENTO_API_KEY in .env
      # JWT (server signing key):
      openssl rand -base64 64
      # Paste into JWT_SECRET in .env; existing sessions will invalidate
      ```

## 4. Emergency: force-flat from dashboard

> TODO when Plan 4 ships: a **red "Emergency Flat" button** on the dashboard plus `POST /api/risk/force-flat` REST endpoint. Currently both are deferred.

Until then, the emergency procedure is **manual**:

1. In NT chart: right-click → close any open position (market order).
2. Disable VLTrader strategy on the chart (toolbar → stop icon).
3. Stop the bot:
   ```bash
   pkill -TERM -f nofx-bin
   ```
4. Once safe, audit the most recent decisions to understand what triggered the panic:
   ```bash
   sqlite3 data/data.db "SELECT * FROM decisions ORDER BY created_at DESC LIMIT 5;"
   ```

## 5. Playwright runbook verification

Before any live trade, run this manual end-to-end check to confirm the
emergency-flat button works.

> The button itself is part of the deferred Plan 3 follow-up (see §4). Run this checklist on the first build that ships it.

```
1. cd web && npm run dev
2. mcp__playwright__browser_navigate to /dashboard
3. mcp__playwright__browser_click the red "Emergency Flat" button
4. mcp__playwright__browser_snapshot — assert confirmation modal appears
5. mcp__playwright__browser_click "Confirm"
6. Verify backend logs: "FORCE FLAT initiated by user"
7. mcp__playwright__browser_snapshot — assert positions table empty within 10s
```

If the button does not exist, does not confirm, or does not flatten — **do NOT trade live**. Fix the button first.

## Cross-references

- `kernel/risk_limits.go` — `RISK_MAX_*` env vars + `ResetDailyPnL`.
- `kernel/cme_calendar.go` — `IsCMEOpen`, contract expiry calendar.
- Plan doc Task 19 — `ShouldBlockEntryForExpiry`.
- Plan doc Task 34 — source of this runbook.
