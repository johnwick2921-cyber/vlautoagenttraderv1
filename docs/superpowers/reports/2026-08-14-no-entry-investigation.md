# No-Entry Investigation — Zero Entries Since the Hardening Deploy

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **Running rev:** `3624a2a4` (PID 363618) · **Read-only** (sqlite mode=ro, bounded log/journal reads). This report is the only write.

## VERDICT

**THE dominant blocker is the C2 clock-drift guard — 16 of 16 post-deploy `open_long` proposals were blocked because the WSL2 system clock is chronically 201–479 s (mean 331 s) BEHIND the NT8 feed, versus C2's 60 s threshold.** Receipt [A]: `kernel/clock_drift.go:77`. **Classification: CORRECT BEHAVIOR of the guard, driven by a REAL environmental fault (WSL2 clock drift) — not a code bug, not the AI, not a stuck halt/freeze.** The armor works: F1's real R:R gate PASSES every setup (R:R 3.00–3.14), then C2 correctly refuses to emit a signal that NT8 would reject as stale (the AddOn drops signals >60 s old). **The fix is OPS, not code: resync the system clock.** Until then the bot is functioning as designed — it is declining to fire signals that would bounce off NT8.

---

## STEP 1 — Process truth [A]
- `nofx-bin` **running**, PID **363618**, rev **`3624a2a4`** (`go version -m nofx-bin` → vcs.revision 3624a2a4). systemd `active`, **`NRestarts=0`**, `ExecMainStartTimestamp=2026-08-13 19:59:36 CDT`. No restarts since deploy.
- **Both traders cycling:** 468 + 469 decision rows since Aug-13 15:00; latest decision `2026-08-14 15:59:50 UTC` (10:59 CT). The loop is alive and advancing — not a stuck process. Source: `decision_records` (28 713 rows total).

## STEP 2 — Halt/freeze states [A]
- **Guardrails master is OFF** (258× Aug-14: `engine_analysis.go:164 "⚠️ risk guardrails master OFF — daily loss/profit/trade limits + blackout NOT enforced"`). So **max_daily_trades is not even active** — the Aug-13 14:56 CT trip is moot; nothing to reset. (Size caps remain via D3.)
- **No A4 freeze** (0× `A4 FREEZE` on Aug 14; the only 🚨 lines are clock-drift). `GET /api/risk/freezes` not callable read-only (JWT-gated), but the log is authoritative: zero freeze events.
- **D1 consecutive-loss halt: not latched** (0× `consecutive-loss halt` Aug 14).
- **B6 gate-block table** (in-memory, JWT-gated endpoint) not directly readable, but the per-gate log tally IS the ground truth (STEP 3).

## STEP 3 — Decision autopsy (since Aug-13 15:00) [A]
937 cycles: **935 success, 2 AI-call failures**, execution_status **empty on all 937** (zero `guardrail_skip`). Proposed-action tally: **~888 wait, and open_long proposed in the raw model output 19×.** Of those 19:
- **3 executed** — but all **pre-deploy** (Aug-13 10:27/10:34/14:49 CT, old rev 74aac5b6; `exec_log: "✓ MNQ open_long succeeded"`). These are the "morning trades."
- **16 post-deploy `open_long` proposals (all Aug-14) were flipped to `wait`** — every one with a matching pair of log lines:
  1. `📐 R:R eval MNQ open_long: entry=… SL=… TP=… → R:R=3.00 (min 3.00) → PASS` (`engine_position.go:178` — F1 real R:R)
  2. `🚨 clock-drift ENTRY BLOCK: MNQ open_long → WAIT — local clock is BEHIND the feed by NNNs (>60s)` (`clock_drift.go:77` — C2)
- **Tally by cause of the flip: clock-drift = 16/16.** No other gate fired on Aug 14: `feed-gate 0 · stale-data ENTRY BLOCK 0 · re-entry cooldown 0 · dead-man watchdog 0 · consecutive-loss halt 0 · A4 FREEZE 0`.
- **Zero orders / positions / fills since Aug-13 15:00** (`trader_orders`, `trader_positions`, `trader_fills` all 0) — confirms no entry executed post-deploy.
- Sample model reasoning is coherent and cites the new features (e.g. *"below the session value area: VAL 30130 POC 30158 VAH 30253 … RSI7 ~15-23 on 15m/30m/1h"*) → SVP (B9/F6) and RSI7 (F9) are live in the prompt.

## STEP 4 — AI-call health [A]
- **5 transport blips** on Aug-14 (`AI API call failed: failed to read response`) out of 937 cycles (0.5 %) → 2 recorded as failed decisions, rest retried. Negligible; not the cause.
- **B2 price-sanity: 0 rejections** on Aug-14 (no `price-sanity` neutralizations logged) — the AI's stops/targets are physically plausible.
- **F1/F8/F9 intact:** the `📐 R:R eval` lines prove F1's real-entry R:R math runs (PASS at 3.00+); reasoning references RSI7 (F9) and SVP levels; no golden/prompt anomaly. 29/937 cycles hit the B2a "no structured JSON → safe wait" fallback (3 %) — minor, not dominant.

## STEP 5 — Wire / data [A]
- **NT8 link stable:** `dead-man watchdog 0` events Aug-14 (no disconnect), continuous `tcp_server: received frame type=bar_update` through 11:06 CDT; the AddOn is connected (v3, from the deploy).
- **Bars fresh:** `stale-data ENTRY BLOCK 0` (B4 never fired) → the 1m/5m feed is current; bars are flowing.
- **Session gate open:** decisions are being produced through RTH (10:59 CT latest) — CME is open, no session-closed skip blocking.
- **The one thing wrong on the wire is TIME:** the feed's bar timestamps (NT8/Tradovate real time) run 201–479 s ahead of the local WSL2 clock — the exact skew C2 measures.

---

## Classification & distribution (dispatch step 6/7)

**CORRECT-guard / ENVIRONMENTAL-fault.** This is not a bug in the code and not the AI refusing to trade. The distributions show the pipeline is healthy right up to the clock check:

- **R:R of the 16 blocked setups:** `3.00 ×12, 3.02 ×1, 3.08 ×1, 3.14 ×1` — all **PASS** the 3.0 floor. The AI is precisely building 3:1 setups (exactly what the futures prompt asks). **R:R is not the constraint.**
- **Confidence of the 16:** `67,67,68×10,70×3,72` — all ≥ 65. **min_confidence is not the constraint.**
- **Clock drift of the 16:** n=16, **min 201 s, max 479 s, mean 331 s** — every one far past the 60 s threshold. **This is the sole constraint.**

Why the guard is right to block: `placeEntry` stamps the signal `Timestamp` from the local clock; the C# AddOn **rejects any signal older than 60 s** (`STALE_SIGNAL_AGE_SECONDS`, `VLTraderTCPClient.cs`). With the clock 3–8 min behind, every signal would arrive already "5+ min old" by NT8's clock and be rejected. C2 pre-empts a guaranteed rejection rather than firing blind. It is doing its job.

## Root cause & fix (OPS — do NOT change code)

**Root cause:** WSL2 system-clock drift. WSL2 VMs fall behind real time (notably after the Windows host sleeps/hibernates); here the clock is chronically ~3.4–8 min behind and never self-corrected across the whole Aug-14 session (02:47 → 10:00 CT all showed 200–480 s).

**Minimal fix (owner, no code change):**
1. Resync the WSL2 clock, e.g. `sudo hwclock -s` (adopt the hardware/host clock) or `sudo ntpdate time.windows.com` / restart `systemd-timesyncd`. If it won't hold, `wsl.exe --shutdown` from Windows then restart WSL (forces a host time re-sync at boot).
2. Verify: the next AI `open_long` cycle should log `clock-drift` with a drift < 60 s (or no clock-drift line at all) and proceed to `placeEntry`. No bot restart is required — C2 re-measures every cycle and clears itself the moment the drift drops under 60 s (`kernel/clock_drift.go`; the guard is not latched).
3. Optional durability: add an NTP/`hwclock -s` resync to the machine's boot/wake so the drift can't silently return.

**No code fix is warranted** — C2 is behaving correctly and is self-clearing; the defect is the host clock. If the owner nonetheless wants to reconsider policy, the only knob is C2's 60 s threshold (would just push signals into NT8's own 60 s rejection) — not recommended.

## Evidence ledger
All Tier [A] unless noted. Process: `pgrep`/`systemctl show`. Decisions: `decision_records` (sqlite ro). Gate receipts: `data/nofx_2026-08-13.log` (the active log — opened at the 19:59 Aug-13 start, still receiving Aug-14 writes; there is no `nofx_2026-08-14.log`). Key lines: `kernel/clock_drift.go:77` (16×), `kernel/engine_position.go:178` (R:R PASS 16×), `kernel/engine_analysis.go:164` (guardrails master OFF). Live drift last measured 253 s at 10:00:46 CDT; WSL2 drift persists, so it is presumed current [B].
