# Sunday-Shield Cutover Record

**Cutover:** 2026-08-29 16:35 CT · **Merge M:** `bf2b6e9f` · **Marker R:** `4763a664` · **Deployed rev:** `bf2b6e9f9968` (PID 87437, boot 16:35:30 CT).

## Sequence (per the wave record `2026-08-29-sunday-shield-wave.md`)

1. Flat-gate all-origin BEFORE swap, all four quoted: API positions `[]` · API open-orders `[]` ×2 (symbol=MNQ, trader R8) · DB OPEN positions 0 · DB non-terminal armed 0.
2. Lock marker `~/nofx-main.lock` acquired (owner/pid/expiry) → `git merge --no-ff fix/sunday-shield` → **M = bf2b6e9f**.
3. Marker commit **R = 4763a664**: `GUIDE_BUILT_REV='bf2b6e9f9968'` + `deploy/RELEASE=bf2b6e9f9968`.
4. Temp-clone build at M: `vcs.revision=bf2b6e9f9968…` `vcs.modified=false` ✓ (go version -m).
5. mv-swap (`nofx-bin.prev.sunday-shield` backup) → `kill -9 4050566` → systemd relaunch (Restart=on-failure).

## Boot acceptance (all quoted from journal)

- `🔐 BOOT INTEGRITY OK — rev bf2b6e9f9968 · expected bf2b6e9f9968 · goldens PASS` ✓
- `🧠 AI params … client_max_tokens=32768 planner_max_tokens=65536 …` ✓
- `📜 scenario schema: 9 conditions […]` ✓
- `🎛 volume wave … proximity=cfg(resolved per-trader; retuned 0.3) …` ✓
- pool intact: `level_stats … total rows 101` · `fed 176 touch episode(s) across 67 level(s)` · `bars integrity OK dups=0 tfs=1m total=15646` ✓
- `🔗 … trader MNQ BOUND to account Sim101` · reconcile started ✓
- veto cross: `HTF_VETO_MODE=cross` (.env) + `🛡️ regime ledger: htf_veto=ON … tf=1h` at boot; the `mode=%s` first-cycle line prints Sunday 17:00 (market closed at cutover).

## Post-boot 15-min window (16:35:30 → 16:55+)

- **`🔕 PERSIST WATCHDOG` = 0** — the 373-line storm class is dead. Pre-fix cadence was 1/min from boot+2min (~48 expected fires in the window); observed 0. W1 frame-awareness proven live.
- ERRO = 1 total: the boot-time `🚨 CLOCK CRITICAL [boot] drift 88532981ms` (C2 saw NT8's replayed Friday 16:00 last bar; ~24.6h weekend gap; **log-only, no gate**). 0 after boot. Not a W1–W3 regression — same weekend-blindness family as T15 S1/S3 (backlog: exempt closed-market last_bar age in the C2 boot check).
- panic = 0.
- Triple alignment re-quoted post-boot: positions `[]` · open-orders `[]` · DB 0/0 ✓.
- `/api/health` = `bf2b6e9f9968` ✓ · guide marker on disk + vite live → drift banner dead ✓.

## Post-cutover state

- dev tip = `4763a664` pushed to origin.
- `~/nofx-main.lock` released after the clean window.
- Next event: **16:55 CT Sunday 2026-08-30** (ASIA pre-read window) → live fire 17:00 CT.

W5 (14 tracked binaries / secret purge) remains gated on the owner's "key rotated" confirmation.
