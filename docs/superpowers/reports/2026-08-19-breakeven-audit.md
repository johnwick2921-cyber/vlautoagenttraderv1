# P0-ADJ — Breakeven "Not Firing" Audit (hotfix/breakeven-dead)

**Date:** 2026-08-19 · **Branch:** `hotfix/breakeven-dead` (off #51's merge base `49dd83c9`)
**Dispatch:** "BREAKEVEN FUNCTION NOT FIRING (LIVE POSITION)" — Phase 1 live triage, Phase 2 fix per verdict, Phase 3 verify.

---

## 1.1 VERDICT (first line, per spec)

**NOT A BUG — the +50pt trigger had not been reached when the dispatch was filed, and the moment it WAS reached (11:36:24 CT, mid-audit) auto-breakeven fired end-to-end: Go monitor → `move_stop` wire frame → NT8 in-place `Change` → stop now resting at entry 29650.75, target + OCO preserved. [A]**

### Live position numbers

| Fact | Value | Evidence |
|---|---|---|
| Position | #524 MNQ SHORT 1.0 @ **29650.75** | `trader_positions` row 524 [A] |
| Entry time | 11:18:15 CT | `entry_time` epoch → CT [A] |
| MFE since entry | **+51.2 pts** (monitor sample at fire; 2-min equity snapshots peak $100.5 = 50.25 pts) | Go file log + `trader_equity_snapshots` [A] |
| Trigger reached? | **YES — at ~11:36 CT**, 18 min after entry, *after* the dispatch was written | snapshot series: +43.0 (11:27) → +45.0 (11:29) → +48.75 (11:31) → +42.0 (11:33) → +44.25 (11:35) → **+50.25 (11:37)** [A] |
| Breakeven fired? | **YES — 11:36:24 CT**, first 60s monitor beat with a ≥50 sample | both logs below [A] |

The owner's observation ("stop has not moved to BE") was correct **at the time it was made** — the position was oscillating at +42…+48 pts, below trigger. Per the dispatch's own rule: *"If N — say so plainly (not a bug yet)"* — it was not a bug then, and when the trigger crossed, the feature proved itself live.

### The live firing, verbatim

Go (complete file log `data/nofx_2026-08-19.log` — journald suppressed this line, see §Root-cause-of-the-false-alarm):

```
08-19 11:36:24 [INFO] trader/auto_trader.go:179 🎯 auto-breakeven: MNQ SHORT +51.2 pts in profit → stop moved to breakeven (entry 29650.75)
```

NT8 AddOn (`log/log.20260819.00001.txt`, same second — the in-place amendment + implicit ACK):

```
2026-08-19 11:36:24:874|1|16|VLTraderTCPClient: move_stop → 29650.75 for signal_id=07ab6d26-e856-4d24-85b8-e6ffbd29363a (auto-breakeven, in-place Change — target + OCO preserved)
```

Zero `move_stop refused` / `move_stop_error` / `Change failed` lines in today's NT8 log [A]. The AddOn also sends `SendAck("move_stop")`; Go's `FrameAck` handler (tcp_server.go:1446) verifies the seq silently (logs only on mismatch — none).

---

## 1.2 Chain map (Studio → stop amendment), file:line

| Layer | Where | What |
|---|---|---|
| Studio field | `web/src/components/strategy/RiskControlEditor.tsx` | breakeven toggle + points input → `risk_control.breakeven_enabled` / `breakeven_trigger_points` |
| DB | `strategies.config` JSON, path `ai_config.risk_control.breakeven_enabled=true, breakeven_trigger_points=50` | strategy `a5b7662e` "MNQ", saved 2026-08-17 ~20:00 CT [A] |
| Config load | `store/strategy.go:803` `StrategyConfig.UnmarshalJSON` → `c.RiskControl = raw.AIConfig.RiskControl` | the `json:"-"` fields are restored from the nested `ai_config` block [A] |
| Evaluator | `trader/auto_trader_risk.go:46-48` `monitorTick` (60s wall-clock) → `checkPositionDrawdown` → `:95` `maybeMoveStopToBreakeven` | runs every minute per open position, independent of the decision cycle |
| Trigger math | `trader/auto_trader.go:187` `breakevenTrigger` — pure fn, default 50 pts, lowercase-normalized side | fires once via `breakevenDone` map (`:161`), re-armed on flat by `pruneBreakevenDone` (`auto_trader_risk.go:148`) |
| Wire | `trader/ninjatrader/tcp_trader.go:357` `MoveStopToBreakeven` → `provider/ninjatrader/tcp_server.go` `SendMoveStop` → `FrameMoveStop` (`move_stop`, tcp_framing.go:304) | seq-registered (A2/G1) immediate command |
| NT8 | `ninjascript/VLTraderTCPClient.cs:627` `type=="move_stop"` → `:1271` `HandleMoveStop` | looks up the live bracket by signal_id, `account.Change` on the resting SL **in place** — same `-exit` OCO group, target preserved; no-op if already ~equal; `SendAck` |
| Deploy state | deployed AddOn md5 `16e705e7…` **identical** to repo `ninjascript/VLTraderTCPClient.cs` | handler IS live (copied Aug 16, NT8 restarted since) [A] |

## 1.3 Dead-control check

**Not a dead control.** The saved value has real consumers on the live path (chain above) and demonstrably drives behavior. (Contrast MaxMarginUsage precedent — does not apply here.)

## 1.4 Units check

Consistent **price points at every layer** — no points/ticks/dollars mismatch [A]:
- Studio "50" → JSON `breakeven_trigger_points: 50` (dimensionless number).
- Go `breakevenTrigger`: `pts = entry − mark` (SHORT) / `mark − entry` (LONG) — raw price-point difference, compared to trigger directly.
- Wire + C#: `new_stop_loss` is an absolute price (29650.75), no unit conversion anywhere.
- Cross-check with dollars: MNQ $2/pt × 51.2 pts = $102.4 ≈ the $100.5 snapshot P&L (sampling offset). A 4× ticks error would have fired at +12.5 pts — history shows fires at +51…+54, never near +12.5.

## 1.5 Twin check (#49 skip-while-open orphaning)

**Not orphaned — by construction and by live proof.** Breakeven evaluation lives in the 60s wall-clock monitor (`monitorTick`, auto_trader_risk.go:46), NOT the decision cycle, so #49's skip-while-open cannot starve it. Live proof: the fire at 11:36:24 landed **between** skip-while-open cycles #37 (11:35:23) and #38 (11:37:27) — the monitor was managing the position while the decision cycle was deliberately idle [A]. This is #49 doctrine working as intended ("bracket/breakeven manage the trade").

## 1.6 History — has breakeven EVER fired?

**Yes — 6 times since Aug 6** (NT8 logs, deduped across en/native pairs) [A]:

| When (NT8 clock) | Stop moved to | Note |
|---|---|---|
| 2026-08-06 04:04:31 | 29525.5 | early handler (pre in-place-Change wording) |
| 2026-08-06 08:00:19 | 29441.5 | |
| 2026-08-07 03:31:44 | 29580.5 | |
| 2026-08-13 08:36:13 | 29901.25 | in-place Change — target + OCO preserved |
| 2026-08-19 04:27:14 | 29623.5 | position #521 (Go log 04:27:17: "+53.8 pts"; #521 later closed, MFE 71.25) |
| **2026-08-19 11:36:24** | **29650.75** | **position #524 — the live position of this dispatch** |

Each signal_id appears exactly once — the `breakevenDone` idempotence held live every time. (Go-vs-NT8 timestamps differ by ~2–3s on the 04:27 pair — the known WSL clock skew, caught separately by the timegate clock-health line.)

---

## Root cause of the false alarm (why the owner couldn't see it)

Two compounding observability gaps — the same pair that produced the phantom-position false alarm:

1. **journald flood suppression.** The TCP frame flood (~58k INFO frames/min) trips journald's rate limit — 5k–25k messages dropped per 30s window today [A]. The 🎯 auto-breakeven line was among the suppressed; `journalctl` looked silent while `data/nofx_2026-08-19.log` (complete) had it. Owner fix already queued: `sudo bash deploy/install-journald.sh` (§9 of the ledger-close report).
2. **INFO level → not in `log_events`.** The P6 DB log sink captures WARN+ only, so breakeven fires are invisible to the dashboard/DB even though they're the single most owner-relevant in-trade event.

**Recommended follow-up (owner call, 1 commit + flat-window deploy):** promote the auto-breakeven fire/fail lines to WARN (or write an in-app alert row) so every fire lands in `log_events`/alerts regardless of journald. Not done in this dispatch: verdict is no-defect, the change would touch a live path, and deploy is blocked anyway (position open — flat-window rule). Dispatch 2.1's log spec is otherwise already satisfied (fired line exists; ACK is NT8-logged; `breakeven_armed` line would ride along with the same follow-up).

## Fix commits

**None.** Phase 2 is conditioned "per verdict"; the verdict is working-as-designed. Every 2.x item already exists in the shipped feature: 2.1 wiring ✓ (chain map), idempotent + never-backward ✓ (`breakevenDone` + NT8 no-op guard), 2.2 evaluation home = 60s monitor ✓, 2.3 units contract = points with one comparison site ✓, 2.4 NT8-side ATM auto-BE not needed (Go-side control is the architecture; C# executes in place preserving OCO), 2.5 handled live — see below.

## V1–V5

| V | Status | Evidence |
|---|---|---|
| V1 trigger→one modify, direction-correct | **PASS (unit + live)** | `trader/breakeven_test.go` `TestBreakevenTrigger` (LONG/SHORT/threshold/default-50/disabled); live: SHORT stop moved DOWN-path… stop sits AT entry 29650.75 below the prior stop [A] |
| V2 idempotence / never backward | **PASS (live ×6)** | one `move_stop` per signal_id across all 6 historical fires; C# no-ops on ~equal stop; `breakevenDone` re-arms only on flat |
| V3 units (50 pts ≠ 50 ticks) | **PASS** | §1.4; `TestBreakevenTrigger_NT8UppercaseNotInverted` guards the casing/inversion regression |
| V4 live stop amendment + ACK + dashboard | **PASS (organic, better than sim)** | the real position did it at 11:36:24; NT8 log line + ack; dashboard reads positions from NT8 truth |
| V5 regression (#49 heartbeats, #50 guard) | **PASS** | heartbeats #37/#38/#39 continuous through the fire; no desync CRITICALs; no code changed, so `go test ./...` state = #51's green matrix |

## Live-position outcome (2.5)

Trigger was reached while the audit ran; the monitor's next beat fired it. The amendment + NT8 confirmation are pasted in §1.1. Position #524 MNQ SHORT is now **risk-free**: worst case is a stop-out at entry 29650.75 ($0 gross), upside still open with the original target + OCO intact. No counterfactual needed.

## PR

Report-only PR (no code): **see PR number in the chat delivery — parsed from the `gh pr create` output URL per standing rule.**
