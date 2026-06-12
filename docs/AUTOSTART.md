# NOFX auto-start on reboot (WSL2 + Windows)

Goal: after a Windows restart + login, the full stack returns with **zero manual
steps** — WSL boots, `nofx-bin` + the frontend run as systemd services
(auto-restart on crash), and the NT8 AddOn reconnects on its own (it retries
every 5s — `VLTraderTCPClient.cs` `RECONNECT_INTERVAL_MS`).

Everything here is **portable**: the unit files are templates with no
hardcoded user, path, or node location — the installer detects all of that on
your machine at install time.

## 0. Prerequisites (one-time)

- WSL2 distro with **systemd enabled**: `/etc/wsl.conf` must contain
  ```ini
  [boot]
  systemd=true
  ```
  If you had to add it, run `wsl.exe --shutdown` once from Windows, reopen the
  distro, and check `systemctl is-system-running` says `running` (or `degraded`).
- A built backend: `go build -o nofx-bin .` in your clone.
- `.env` configured in the clone root. **It must contain `NT_TRANSPORT=tcp`**
  (see `.env.example`) — systemd services never read `~/.bashrc`, so any env
  var living only there is invisible to the service. The installer checks
  this and appends the line if missing.

## 1. WSL side — one command (needs sudo)

```bash
cd <your-clone> && sudo bash deploy/install-autostart.sh
```

The installer detects your **user** (from sudo), your **clone path** (its own
location), and your **node/npm** (from your own shell — nvm-aware), renders
the templates in `deploy/`, and installs + enables two units:

| Unit | What | Logs |
|---|---|---|
| `nofx.service` | `./nofx-bin` from the clone root (reads `.env`, `data/data.db`) | `journalctl -u nofx -f` |
| `nofx-web.service` | `npm run dev` in `web/` (vite :3000, proxies /api → :8080) | `journalctl -u nofx-web -f` |

Re-running the installer is safe and is also the upgrade path: after a
`git pull` that changes the templates, just run it again.

**Why logs are the JOURNAL (and never unit-level files):** two file designs
both died with `Failed to set up standard output: Permission denied` (exit
`209/STDOUT`) **before the binary even ran** on this WSL2 systemd:
1. `append:/tmp/backend.log` — Ubuntu's `fs.protected_regular` sysctl refuses
   to open another user's file in sticky world-writable `/tmp`, and the /tmp
   logs end up owned by whichever side (root vs user) created them first.
2. `append:/var/log/nofx/...` + a root `ExecStartPre` to prep the dir — the
   `StandardOutput=` directive applies to EVERY `Exec*` line, and systemd
   opens the append-file in the forked child BEFORE exec, so even the
   root-prefixed pre-step died at stdout setup without running at all.
The journal sink has no file-open in the child, so it cannot 209. Do NOT
reintroduce `StandardOutput=` file directives or a log-file `ExecStartPre`.
**Tooling note:** the services no longer write `/tmp/backend.log` — use
`journalctl -u nofx` (it also keeps history across reboots). A MANUAL
fallback launch (`nohup ./nofx-bin >> /tmp/backend.log 2>&1 &`) still writes
the /tmp file as before.

**Never permanently dead:** the units use `StartLimitIntervalSec=0` with
`Restart=on-failure` / `RestartSec=5`. A persistent failure will retry every
5s forever (you'll see it in `journalctl -u nofx`) — the tradeoff is a
visible loop instead of a silently dead bot.

Verify after install:
```bash
systemctl status nofx nofx-web --no-pager | grep Active
ss -tlnp | grep -E ':(8080|3000|36974)'   # 36974 binds once a ninjatrader trader loads
# crash-restart proof:
sudo kill -9 $(pgrep -x nofx-bin); sleep 6; pgrep -x nofx-bin && echo RESTARTED
```

**NT8 closed is fine.** The bot is the TCP *server* — it starts, binds
`:36974`, and waits. Whenever NT8 opens (even hours later), the AddOn
connects within ~5s and bars resume. Order is never a problem in either
direction.

## 2. Windows side — Task Scheduler (boots WSL at login)

First find your distro name (PowerShell or cmd):
```
wsl.exe -l -v
```
Use that exact name (e.g. `Ubuntu-24.04`, `Ubuntu`, `Debian`) below.

1. Start → "Task Scheduler" → **Create Task** (not Basic):
   - **Name:** `Start nofx WSL`
   - **General:** Run only when user is logged on.
   - **Triggers:** New → *At log on* → Specific user (you) → ✅ *Delay task for:* `30 seconds`.
   - **Actions:** New → Program: `C:\Windows\System32\wsl.exe`
     Arguments: `-d <YOUR-DISTRO-NAME> --exec /bin/true`
   - **Conditions:** untick "Start the task only if the computer is on AC power" (laptops).
2. That command boots the WSL VM; systemd then auto-starts `nofx` +
   `nofx-web`, and the running services keep the VM alive.

## 3. Windows side — NT8 (optional but recommended)

The bot runs fine without NT8 open (see above) — but for live bars/execution
you want NT8 back automatically too:

1. `Win+R` → `shell:startup` → Enter → copy a **NinjaTrader 8 shortcut** into
   that folder (NT8 launches at every login; the AddOn loads with it and
   connects to the bot within ~5s).
2. Inside NT8: **Tools → Options → General →** set **"On startup, connect
   to:"** = your SIM/data connection — so the data feed reconnects without
   clicks.

## 4. Optional: zero-login boot (headless after power loss)

Everything above fires **at login**. For unattended recovery (power returns →
machine boots → stack up with nobody at the keyboard) enable Windows
auto-login: `Win+R` → `netplwiz` → untick "Users must enter a user name and
password…" → enter your password once. **Security caveat:** anyone who powers
on the PC is logged in as you. Default recommendation: leave login required.

## 5. Full-reboot test (the real proof)

Restart Windows → log in → wait ~1 minute → check:
- `wsl.exe -d <YOUR-DISTRO-NAME> -- systemctl is-active nofx nofx-web` → both `active`
- `:36974` listening; once NT8 is open, the AddOn log shows CONNECTED
- bars flowing (market open) or clean idle (closed); UI at `http://localhost:3000`

SIM-only: nothing here touches trading code; the live-account block is
unchanged. Rollback: `sudo systemctl disable --now nofx nofx-web` and launch
manually as before.
