# DR — Disaster Recovery Runbook

Five failure scenarios, each with **detect → mitigate → verify**. See plan doc Task 33.

## 1. DB corruption

**Detect**
- SQLite error on startup: `database disk image is malformed` in `/tmp/nofx.log`.
- Unexpected `NULL`s in `decisions` / `positions` rows that previously had values.
- `sqlite3 data/data.db "PRAGMA integrity_check"` returns anything other than `ok`.

**Mitigate**
```bash
pkill -TERM -f nofx-bin
cp data/data.db.bak.<latest-timestamp> data/data.db
```

Replay missing decisions from the NT-side fills file as needed:

```bash
cat /mnt/c/Users/<u>/NofxTrader/data/trades_taken.csv
# Reconstruct decisions row-by-row in sqlite3 if rebuilding the audit trail
```

Restart:

```bash
./nofx-bin > /tmp/nofx.log 2>&1 &
```

**Verify**
```bash
sqlite3 data/data.db "PRAGMA integrity_check"   # expects "ok"
sqlite3 data/data.db "SELECT COUNT(*) FROM decisions;"   # expects non-zero
```

## 2. NT8 crash mid-trade

**Detect**
- NT process gone from Windows Task Manager.
- Last fill in `trades_taken.csv` older than expected scan interval.
- Go log shows continued signal writes with no corresponding fill rows.

**Mitigate**
1. Restart NinjaTrader 8.
2. Re-attach VLTrader strategy to the MNQ chart (or restore via Workspace).
3. **Manually verify SL/TP** via NT Orders window. `vltrader.cs` tracks active position internally and resumes from the current position on restart — but if SL/TP weren't acknowledged by the broker before the crash, they are lost. The bot will not re-place them.

If position is open and SL/TP missing, place them manually in NT before letting the bot resume.

**Verify**
- NT Orders window shows expected pending SL and TP for any open position.
- Go log resumes printing decisions without `Plan 3 T22: stale data` warnings.

## 3. Databento outage

**Detect**
- HTTP 401/500 from Databento in the Go log.
- Freshness gate trips with `HealthStale` repeatedly: `grep "Plan 3 T22: stale data" /tmp/nofx.log`.

**Mitigate**
- Engine refuses new entries on stale data per Plan 3 Task 22 — that is the correct behaviour. Existing positions remain managed by NT-side SL/TP (set at entry time).
- Fall back to last-known OHLCV cached in DB if available. This is read-only for display; the engine will still refuse to act on it once age exceeds `MaxAgeRTH` / `MaxAgeETH`.
- If outage extends past the current session (multiple hours), manually flat positions via NT to avoid carrying stale-data risk across a session boundary.

**Verify**
- Engine logs `Plan 3 T22: stale data` continuously while the outage persists — that means the gate is doing its job.
- Positions row count in `data/data.db` does not increase during the outage.
- Once Databento recovers, the stale-data warnings stop within one scan interval.

## 4. WSL2 reboot

**Detect**
- `curl http://localhost:8080/` from Windows host returns connection refused.
- WSL session was bounced (e.g. `wsl --shutdown` or host reboot).

**Mitigate**

Check WSL networking mode:

```bash
wsl --version   # run on Windows host
```

The output must report `mode = mirrored`. If NAT mode reverted, edit `%USERPROFILE%\.wslconfig`:

```ini
[wsl2]
networkingMode=mirrored
```

Then `wsl --shutdown` and relaunch WSL. Also verify `/mnt/c/` is writable from Linux side:

```bash
touch /mnt/c/Users/<u>/NofxTrader/data/.ping && rm /mnt/c/Users/<u>/NofxTrader/data/.ping
```

**Verify**
```bash
curl http://localhost:8080/   # from Windows host: responds with HTTP 200 / dashboard HTML
```

## 5. Lost JWT secret

**Detect**
- All users logged out simultaneously.
- HTTP 401 on every `/api/*` endpoint with valid-looking tokens.

**Mitigate**

```bash
openssl rand -base64 64                # generate new secret
# update JWT_SECRET= in .env
sqlite3 data/data.db "UPDATE users SET last_session_token = NULL;"
pkill -TERM -f nofx-bin
./nofx-bin > /tmp/nofx.log 2>&1 &
```

**Verify**
- Re-login through the web UI succeeds and returns a new token.
- Old token replayed via `curl -H "Authorization: Bearer <old>"` returns HTTP 401.

## Cross-references

- `config/config.go:82-90` — JWT secret loading + insecure-default warning.
- `kernel/cme_calendar.go` — session boundary that informs §3 ("past the current session").
- `market/data_freshness.go` — `MaxAgeRTH` / `MaxAgeETH` thresholds.
- Plan doc Task 33 — source of this runbook.
