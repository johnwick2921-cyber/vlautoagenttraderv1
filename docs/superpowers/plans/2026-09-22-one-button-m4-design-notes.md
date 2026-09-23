# W-ONE-BUTTON M4 — design notes (requirements recorded before M4 starts)

**Why this file exists:** the CTO asked for the M4 requirements that came out of M1 to be written down now, so they are not lost before M4 starts (msg `1790133979048-1429-000001`, "§7 → M4 requirements"). Evidence: `docs/superpowers/reports/2026-09-22-nt8-automation-feasibility.md` (PR #180).

**Status:** requirements only. There is no M4 code yet, and M4 starts from dev's tip when M2 and M3 have merged. Each item names its source; [A]/[B]/[C] are the M1 evidence tiers.

## Runtime and worker requirements

| # | Requirement | Source |
|---|---|---|
| R-a | **Restart identity is `(MainPID, /proc/<pid>/stat field 22 starttime ticks)`, never wall clock.** Read MainPID with `systemctl show -p MainPID --value nofx`. Re-read `/proc/<pid>/exe` and field 22 immediately before signalling. Prove the relaunch by a **change of both** MainPID and starttime ticks. | M1 §7 [A]: wall-clock start times disagreed by 151 s because of a WSL clock step. |
| R-b | **The 90 s boot watchdog reads `data/nofx_<boot-date>.log`, not journald.** It looks for `BOOT INTEGRITY OK — rev <new12>` after the new process's start, plus `/api/health` revision == new sha, plus the served `index.html` hash == manifest. | M1 §7 [A]: journald for uid 1000 keeps about 4 h. The boot proof survives only in the data log. |
| R-c | **Refuse activation unless `<installation>/data/data.db` exists and its schema fingerprint matches the tested profile.** | M1 §7 [A]: `main.go:71-76` runs `MkdirAll` on the data dir, so a wrong working directory boots an **empty** DB that looks healthy. |
| R-d | **`.env` is mutable and stays at the installation.** It is never shipped in a release, never overwritten, never symlinked into a release. | M1 §7 [A]: `api/handler_onboarding.go:241-246,339` **writes** `.env`. |
| R-e | **Never symlink the working directory itself.** Only the release-owned entries switch: `nofx-bin`, `web/dist`, `deploy/RELEASE` and `calendar_static_t1.json`. The unit's `WorkingDirectory=/home/hoang/nofx` stays a real directory holding `data/` and `.env`. | M1 §7 [B]: a symlinked working directory pins the release inode, and `data/` and `.env` would then resolve inside the release. |
| R-f | **`calendar_static_t1.json` is release-owned in v1, with no install-local override.** | CTO ruling (§7 f). Evidence: it is git-tracked, read at runtime on calendar fallback (`trader/auto_trader_calendar.go:58,116-121,174`), and its comment was ambiguous. |
| R-g | **`nofx-web.service` (`npm run dev`) is out of scope for partners.** Partners use the Go-served `web/dist` only. | CTO ruling (§7 g). |
| R-h | **Any UIA search is scoped to the NinjaScript Editor window's subtree, never the desktop root by Name.** | M1 §6 [A]: a live Chart window exposes Chart Trader Buy/Sell/Reverse/Close buttons with InvokePattern. |
| R-i | **A connected NT8 is never assumed to stay connected. Readiness is re-proven at every boundary** (before maintenance, after drain, before any NT8 step, before app cutover). | M1 F2 [A], CTO correction (e): the Simulation token refresh sat in a CAPTCHA penalty for about 35.5 h inside a running NT8. |
| R-j | **The NT8 step is compile-in-place primary.** The flat gate runs **before** the compile. Proof is the DLL/pdb/xml artifact change, the `Terminated→Active` pair, and a hello on a **new connection epoch** carrying the new identity. A full NT8 restart is an **attended fallback only**. **Not built until the owner has seen this ruling.** | CTO ruling Q2 (owner may veto). M1 F3 [A]: F5 hot-reloads in-process. F2: unattended relaunch is blocked by the platform login. F1: a restart disconnects non-SIM connections. |
| R-k | **Every NT8-affecting step is blocked while ANY non-SIM connection is connected.** Unknown coverage blocks. | CTO ruling Q1. The census comes from M2's `maintenance_ack`. |
| R-l | **The helper refuses unless `NinjaTrader.exe` FileVersion is in the manifest's tested range.** | CTO ruling F9. NT8 has its own updater (`ControlCenterMenuItemHelpUpdates`). |
| R-m | **`Strategies\vltrader.cs` (never in git) is hashed and backed up to the private `~/nofx-backups/addon/` before any compile, never modified, and its compile errors are reported, never suppressed.** | CTO ruling F10. First backup taken 2026-09-22: `~/nofx-backups/addon/vltrader-cs-20260922/` (sha256 `5c621520…`). |
| R-n | **Build verification binds to M2's per-connection record** (accept sequence after the old epoch, `nt8_pid` + `nt8_start_ms` == the Windows-side process, a content-derived `source_hash`/`assembly_mvid`), **never to `FarSideBuildID()`**. | CTO ruling Q3; M1 F4 [A]. |
| R-o | **Rollback does not trust the old v3 script.** It must also restore `web/dist` atomically, never collide on `nofx-bin.old.*` names, and never use `pgrep -f nofx-bin` (which also matches `go version -m nofx-bin`). | M1 §7 [A]. |
| R-p | **The first compile of any round rewrites the csproj; later compiles may not.** A shadow compile must regenerate the item list from disk, not reuse a stale csproj. | M1 §3 (critic C2) [A]. |

## Open, carried into M4

- Whether an external DLL replacement is allowed while NT8 runs.
- What a failed compile leaves on disk, and what a restart loads afterwards.
- Whether an open Editor auto-compiles when the source files change underneath it.
- The `ConfirmWindowClose` dialog.
- A reliable signal that the Windows session is locked.
- Whether `nofx.service` or a `--user` unit can reach WSL interop.
- All of these are settled on the isolated VM (M1 report §8), never on the live NT8.
