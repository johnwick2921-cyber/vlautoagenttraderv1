# CLEAN-MACHINE READINESS CHECKLIST — 2026-09-08

Companion to `docs/superpowers/runbooks/2026-09-08-clean-machine.md`. One line
per item: the tick box, the fact, the verification command, the expected output.
Tick only when you SEE the expected output. Env values are never printed (A25);
keys are named.

## Windows / WSL
- [ ] WSL2 systemd enabled — `systemctl is-system-running` → `running` or `degraded` (docs/AUTOSTART.md §0)
- [ ] Mirrored networking on — Windows browser opens `http://localhost:3000` (web/vite.config.ts:8-13)
- [ ] Host TZ America/Chicago — `timedatectl` → `Time zone: America/Chicago` (nofx-backup.timer:6-7)
- [ ] NinjaTrader 8 installed, logged in, B's OWN SIM account created — NT8 Accounts window shows it

## Toolchains
- [ ] Go ≥ 1.25.3 — `go version` → `go1.25.x` with x.3 ≥ 25.3 (go.mod:3)
- [ ] C toolchain present (CGO sqlite) — `gcc --version` prints (store/gorm.go:11, mattn/go-sqlite3)
- [ ] node + npm present — `node --version && npm --version` print (web/package.json scripts)
- [ ] python3 + gzip + openssl + git + curl — `python3 --version && gzip -V && openssl version && git --version && curl --version | head -1`

## Repo & build
- [ ] Clone at B's own path — `git rev-parse HEAD` prints B's chosen sha (Step 2)
- [ ] `deploy/RELEASE` coherent with the sha B will build — `cat deploy/RELEASE` matches or prefixes the planned build (kernel/boot_integrity.go:84-95)
- [ ] Backend builds from the clone — `go build -o nofx-bin . && go version -m ./nofx-bin | grep vcs.revision` → `vcs.revision=<sha>` (install-autostart.sh:45-47)
- [ ] Frontend bundle exists — `ls web/dist/index.html` → the file (api/ui_serving.go:34; UNTESTED `npm run build` in the audit wave)

## .env (names only)
- [ ] `.env` exists at clone root — `awk -F= '/^[A-Za-z_][A-Za-z0-9_]*=/{print $1}' .env` lists the keys chosen in Step 4
- [ ] Boot keys present: `RSA_PRIVATE_KEY DATA_ENCRYPTION_KEY JWT_SECRET` — `grep -c '^RSA_PRIVATE_KEY=' .env && grep -c '^DATA_ENCRYPTION_KEY=' .env && grep -c '^JWT_SECRET=' .env` → `1 1 1` (crypto/crypto.go:77-94; config/config.go:117)
- [ ] Trade keys present: `TRADING_MODE=futures NT_TRANSPORT=tcp NT_ALLOWED_ACCOUNTS` — `grep -E '^TRADING_MODE=|^NT_TRANSPORT=|^NT_ALLOWED_ACCOUNTS=' .env` → three lines, values set (config/config.go:172; transport.go:25; tcp_trader.go:312)
- [ ] `NT_ALLOWED_ACCOUNTS` names B's SIM account, NOT A's — compare against NT8 Accounts window (D3)

## Database
- [ ] `data/data.db` is B's own — `python3 -c "import sqlite3;print(sqlite3.connect('file:data/data.db?mode=ro',uri=True).execute(\"select count(*) from users\").fetchone())"` → `(0,)` before first registration (handler_user.go:52-55)
- [ ] Tables auto-created after first boot — same connection, `select count(*) from sqlite_master where type='table'` → ~30 (store/store.go:143-247)
- [ ] No rows copied from A — positions/armed/equity counts all 0 on day one (C4)

## Units
- [ ] `nofx.service` active — `systemctl status nofx --no-pager | grep Active` → `active (running)` (deploy/nofx.service)
- [ ] `nofx-web.service` active (optional) — `systemctl status nofx-web --no-pager | grep Active` → `active (running)`
- [ ] Backup timer installed and paths fixed — `systemctl --user list-timers | grep nofx-backup` shows next 05:00/17:30 CT; `systemctl --user start nofx-backup.service` then `journalctl --user -u nofx-backup.service -n 5` → `wrote …db.gz` (nofx-backup.service:7 hardcoded path edited)
- [ ] Clock-guard timer installed and paths fixed — `systemctl --user list-timers | grep clock-guard` every 15 min; start it once → journal line `clock-guard status=OK|CRITICAL` (nofx-clock-guard.service:9)

## NT8 AddOn
- [ ] Repo `.cs` copied to Documents AddOns folder — `ls "/mnt/c/Users/<B>/Documents/NinjaTrader 8/bin/Custom/AddOns/"` → VLTraderTCPClient.cs + VLBarsSubscriptionManager.cs + VLContractResolver.cs (HARD RULE)
- [ ] F5 compile clean inside NT8 — NT8 compile output shows no errors
- [ ] FULL NT8 restart after compile — NT8 exited and relaunched (AddOns do not hot-reload)

## First boot (journal)
- [ ] Encryption service up — `journalctl -u nofx -n 200 | grep 'Encryption service'` → `initialized successfully`
- [ ] BOOT INTEGRITY OK — `journalctl -u nofx | grep 'BOOT INTEGRITY'` → `OK — rev <sha> · … · goldens PASS` (never REFUSED)
- [ ] Fresh-DB line — `journalctl -u nofx | grep 'No trader configurations'` → the line (main.go:273-277)
- [ ] UI served — `journalctl -u nofx | grep -E '🖥'` → served from web/dist (NOT served-by=none) (main.go:299-307)
- [ ] Calendar loaded — `journalctl -u nofx | grep '🗓'` → session calendar line, unsourced dates counted (main.go D6)

## Provisioning (UI)
- [ ] First user registered — login succeeds on :3000 (handler_user.go:52-55)
- [ ] DeepSeek model row saved with B's key — Settings shows the model, enabled (store/ai_model.go:28)
- [ ] NinjaTrader exchange row: `nt_data_dir` non-empty, instrument `MNQ`, qty 1 — Settings shows it (auto_trader.go:678-680; manager/trader_manager.go:681)
- [ ] Strategy created, `max_contracts_per_order` ≤ 2 — Strategy page shows it (auto_trader_orders.go:25,49-60)
- [ ] Trader created and BOUND to B's SIM account — trader card shows the account; `journalctl -u nofx | grep 'loaded to memory'` → `✓ Trader '…' loaded` (manager/trader_manager.go:713)

## End-to-end
- [ ] Far-side build id proven by receipt — `journalctl -u nofx | grep 'nt8 addon'` → `build_id=<VL_BUILD_ID> expected=… match=yes` (trader/auto_trader.go:44; VLTraderTCPClient.cs:55)
- [ ] Bars flowing (market hours) — dashboard chart moves; journal shows bar updates without backpressure floods
- [ ] Ledger truly empty & flat — positions page shows none, snapshots show B's account with `count=0` (D4 item 2)
- [ ] Planner writes a plan — `journalctl -u nofx | grep 'PLAN written'` after a read (proves AI key + bars + calendar + strategy)
- [ ] Crash-restart — `sudo kill -9 $(pgrep -x nofx-bin); sleep 6; pgrep -x nofx-bin && echo RESTARTED` → `RESTARTED` + boot block re-prints with `goldens PASS`

## Divergence spot-checks (the silent killers)
- [ ] B's NT8 account ≠ A's — NT8 Accounts window on B shows a different SIM name (C7 — nothing in code prevents the share)
- [ ] B never received A's `data/data.db` — checklist item "No rows copied from A" holds
- [ ] No `SANDBOX_MODE=1` in B's `.env` — `grep -c '^SANDBOX_MODE=' .env` → `0` unless deliberate (main.go:530-534)
- [ ] B's claim/branch work uses NEW branch names — `deploy/nofx-claim.sh new` refuses collisions anyway (nofx-claim.sh)
