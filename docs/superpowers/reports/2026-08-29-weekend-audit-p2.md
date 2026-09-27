# Weekend Deep Audit — Part 2/2: MONEY LAYER (synthesis, 2026-08-28 → 08-29 CT)

- **Orchestration:** 4 parallel read-only agents — B1 full-window refusal autopsy · B2 expectancy & edge · B3 calibration sweeps (report-only) · B4 process & truth — each with terminal+DB access, in isolated worktrees. Orchestrator independently re-verified top claims (marked **[O]**). Branch `docs/weekend-audit-p2` @ deployed rev `db9245dc`. Market closed; nothing faked.
- **Rules:** R1 fresh evidence · R2 independent math · R3 twin long/short · R4 file:line · R5 S/A/B/C · R6 PROVEN/EVENT-WAIT/BROKEN/UNVERIFIED · R7 `pnl_corrected` · R8 trader binding (hoang → strategy `a5b7662e` via `traders.strategy_id`) · R9 1m-bar replays.
- **Window:** 2026-08-26 00:00 CT → Fri 08-28 close · MNQ SIM. **n=9 closed trades, Σ pnl_corrected = −$176.50** (3W/6L) — **[O]** verified: rows 562–570 exactly (562 −98.5 · 563 −49.5 · 564 −69 · 565 −62.5 · 566 +97 · 567 −42 · 568 −32 · 569 +63 · 570 +17). 1m bars: 3,720 in-window, 2 scheduled CME halts only.

---

## S-LIST (ranked, cross-agent)

| # | Sev | Finding | R6 | Agent |
|---|-----|---------|----|-------|
| S1 | A | **Canon breach pair: the shipped guide lies about its own revision.** `GUIDE_BUILT_REV='8666db0b'` (`web/src/guide/types.ts:6`) while the pre-reopen wave shipped guide content at `db9245dc` without the same-PR bump (GUIDE CONTENT LAW) → drift banner on the running binary; plus `deploy/RELEASE` marker never committed. **FIXED THIS SESSION [O]: commits `2b2eab2b` (RELEASE marker) + `dbcb61ac` (rev bump, Guide vitest 10/10 · tsc 0) on dev.** | PROVEN → FIXED | B4 |
| S2 | A | **The week's edge is purely NY; pre-NY is structurally negative.** NY 3/3 +$177 (armed #569 +63, #570 +17, manual #566 +97) vs ASIA+LONDON 0/6 −$353.5 (LONDON AI-proposed 0/4 −$279.5). Stops sit inside noise: 6/8 MAE ≥ stop distance, 7/8 stops ≤1.65×ATR14(5m); 0/9 targets reached; giveback 359.75pt ≈ $719.5. | PROVEN | B2 |
| S3 | A | **arm-R:R (ARM_MIN_RR 2.0) is the only net-COSTING gate: +$127.3 would-have-won** (n=9: LONDON S2 +$68, ASIA S3 +$46, ASIA S1 +$42.4). **Adjudicated KEEP**: N1 sweep at n=18 — 2.0 gives +$994.1, 1.5 admits a loser (−$24), 2.5 sacrifices 8 winners (−$781), 3.0 −$918.3. The floor is right; the leak is 3 ugly-but-tape-run arms. | PROVEN | B1/B3 |
| S4 | A | **breakdown_continue pullback-arm cannot capture a no-retest waterfall.** Crash-day replay (VWAP 29657.39): pullback-mode rested unfilled on the real crash body (10:48 leg1 never retouched, $0); **immediate-mode would-have +$246 incl. +$178 on that leg.** The two swept knobs (disp 0.8–1.2 × pullback 0.3–0.5) do NOT change capture — **entry_mode does.** Escalate to owner as a play-contract ruling before Sunday. | PROVEN | B3 |
| S5 | B | **Giveback $719.5 with zero trail exits ever** (1 trail arm, 5 ratchets, AI exited first). Trail-mult sweep UNVERIFIED (n=0) → EVENT-WAIT. | PROVEN | B2/B3 |
| S6 | B | **4 armed fills in-window, not 2:** #568 (S1 @29642.00 07:20) and #570 (S2 @29463.25 12:11) are journal-proven armed fills whose rows lost lineage (`reconcile` materialized them as untracked; repair failed). Prior audits undercounted. | PROVEN | B2 |
| S7 | B | **Partner repo 185 commits / +14.7k lines behind** (`f6ae7597` = 2026-08-23 mirror) and still carries the **closed-market recreate-livelock C# bug** in `VLBarsSubscriptionManager.cs` (old `if (!anyDead)` re-arm). `VLTraderTCPClient.cs` md5 differs (8411d403 vs 109988f7). Runbook below. | PROVEN | B4 |
| S8 | C | **Process: dev force-reset ×2 on 08-26** (reflog 00:01 + 01:17) + 6 dirty worktrees + cross-terminal echo. Root-process recommendation below. | PROVEN | B4 |
| S9 | C | **`.env.bak.0825-2157` / `.env.bak.2026-08-27-seam` not covered by the `*.bak` gitignore rule** → an accidental `git add -A` stages live keys. | PROVEN | B4 |
| S10 | C | `ListNonTerminal()` unscoped (`store/armed_orders.go:134`) · `strategies.is_active` trap re-confirmed (is_active=1 row is bound to NO trader) · README/PIPELINE-MAP stale (no futures/NT8, no armed path). | PROVEN | B4 |

---

## THE THREE MONEY ANSWERS (week scale)

**(1) Where did the week's P&L actually come from?** From NY-session discipline and one news force-flat — not from the plan edge. **+$177 of NY profit vs −$353.5 of pre-NY losses.** The single best trade, #569 (+$63, grade A), was the week's *gate-refused arm* (min-SL refusal at 08:43:28; the v1 resting order filled anyway) closed by the **T1 force-flat** (Fed Warsh 09:00) at +1.39R — neither its stop nor target did the work. Every AI-proposed LONDON entry lost (0/4, all stopped at the tick). [O] rows confirm.

**(2) Single biggest $ leak remaining.** Forward-looking: the **breakdown_continue entry-mode gap** — the one real waterfall this week (08-28 −347pt crash) paid **+$178 would-have** on the 10:48 leg under immediate-mode and **$0** under the shipped pullback contract (S4). Realized-leak: **pre-NY negative edge −$353.5** (LONDON 0/4 with stops ≤1.65×ATR and 0/9 targets ever reached — the bot is structurally long-drawdown in LONDON). Gate-level: arm-R:R +$127.3 (adjudicated KEEP). Giveback $719.5 awaits a first trail exit.

**(3) Knob changes evidence justifies NOW vs Sep-9.**
- **NOW: ZERO knobs.** All sweeps KEEP on week-n evidence: ARM_MIN_RR 2.0 (+$994.1 optimum) · FAST_MARKET_ATR 1.5 (0 live triggers; 1.0 would make 52% of reads fast) · BD 1.0/0.4 (flat grid; capture is mode-bound) · proximity 0.3 (monotone bucket gradient; hot 0–90pt bucket 84% react) · swing-k 2 · FVG disp 1.5 · HTF veto **cross** (1h-only would have cost +$326.3 again; cross $0).
- **NOW (non-knob, owner):** ruling on waterfall **immediate-mode** authoring (S4) · Binnie partner sync (runbook §B4) · README/PIPELINE-MAP refresh queue · process lock (S8).
- **SEP-9 (needs n):** min-conf 60–64 band (preliminary n=3 −$217; 65+ n=1 −$62.5; <60 n=5 +$103) · proximity distance-bucket reaction (n=45) · trail mult (0 exits) · pre-NY session-edge review at full week-n.

---

## Per-agent summaries

### B1 — Full-window refusal autopsy
716 raw refusal lines → 38 unique events; replay **net −$511.8 SAVING** across the gate suite. Per-gate: stale_reeval n=5 **−$372.5 SAVING** (hero) · arm min-SL n=13 −$115.3 SAVING (flipped on the 08-28 liquidation tape) · HTF-veto-cross n=9 −$114.0 SAVING (flipped; 08-28 vetoed longs stopped) · arm R:R n=9 **+$127.3 COSTING** · session_gate/dead_man −$37.5 SAVING. Decline leak: 19 fresh-MET declines → **true leak = 1 decline +$41** (vs prior +$1,974.5 — pre-mandate waits carry no arm spec, excluded per honesty rule). 5 honest-wait narratives quantified (top: NY S1 min-SL +$127.7 would-have, half-realized by the v6 re-spec +$63). Multi-gate: 5 specs refused by ≥2 gates (NY S3/S4 double-counted between veto+minSL). Honesty: 13 UNRESOLVABLE replays excluded; gate-block counters memory-only (unrecoverable post-hoc).

### B2 — Expectancy & edge
Condition×session: breakout_retest LONDON 0/2 −$148 · reject LONDON 0/2 −$131.5 · reject ASIA 0/1 −$42 · reject NY 1/1 +$63 · UNRESOLVABLE 3 rows (566/568/570, +$82 — journal resolves 2 as armed fills). Entry-class: armed_fill real count **4** (2W/2L +$6) vs AI-proposed normal 0/4 −$279.5 vs manual +$97. **No bias-tree branch paid net-positive; session — not bias — was the driver.** MAE/MFE: 6/8 MAE≥stop (tight), 0/9 TP, giveback $719.5. R-floor rerun: prior crowns collapsed — reject-R≥3 was +$798, now n=2 −$131.5; S3-class was +$866/8W, now n=1 −$69; reject-NY held 2/2 +$80. Time-of-day: pre-NY 0/6 −$353.5, NY 3/3 +$177. Case studies: **#567** perfect fill (authored px exact, arm's own SL, −1.0R, fought the 1h uptrend the veto had flagged 21×) · **#569** zero slippage, T1 force-flat exit, the week's best trade.

### B3 — Calibration sweeps (report-only)
6 GAR reruns: swing-k **KEEP 2** (best 7/9 days) · FVG disp **KEEP 1.5** · veto TF **KEEP cross** (1h-only +$326.3 cost, 2nd straight week) · min-conf band **SEP-9** (60–64: n=3 −$217) · trail mult **EVENT-WAIT** (0 exits) · proximity **KEEP 0.3** (pools 5.3/6.6/7.2/8.4; 0–90pt bucket 100% touch/84% react). NEW: ARM_MIN_RR **KEEP 2.0** (n=18: +$994.1 optimum; 3.0-era refusal of NY S1 R2.04 would have cost +$108.5) · FAST_MARKET_ATR **KEEP 1.5, EVENT-WAIT** (0 live triggers — feature shipped after the crash; census 39% reads at 1.5×) · breakdown_continue **KEEP 1.0/0.4, SEP-9** + the S4 entry-mode finding.

### B4 — Process & truth
Guide: 10/10 cards truthful vs live config (proximity 0.3 ⭐ · min-conf 60 · ARM floors · veto-cross · waterfall cards · chips · 7/10/10 · PERSIST_STALL_WATCHDOG_S=60 · 12/5/4). Canon laws quoted from BOTH memory files: repo `CLAUDE.md` (:21 NT8 rule, :134 worktree, :140 no-unattended-deploys, :146 SIM-only, :150 guide-content, :156 guarded-DB, :162 rebuild+kill-9, :168 no-sudo, :178 5-forbidden-tools, :182 partner, :187 repo-target) + `memory/feedback_agent_toolbox_rules.md` + MEMORY.md. Repo: dev `dbcb61ac` (post-fix) · deployed `db9245dc` · 11 open PRs · 18 worktrees · 0 stashes. **Root-process recommendation:** mechanical main-tree lock — (1) every dispatch gates on `git status --porcelain` clean + branch check; (2) non-deploy work only via `git worktree add` + `git worktree lock`; (3) a shared `~/nofx-main.lock` marker (owner/PID/expiry) acquired before touching `~/nofx`, with `git reset` on dev forbidden outside the deploy-owning dispatch. **Partner drift exact:** 131 shared files +14,703/−311; C#: `VLContractResolver.cs` identical, `VLTraderTCPClient.cs` 5 hunks missing (armed dicts, cancel/modify handlers, limit-entry), `VLBarsSubscriptionManager.cs` 1 hunk (livelock bug). **Binnie runbook:** ① clean partner tree ② `git -C ~/nofx-p2 format-patch -o /tmp/binnie-sync 37a03af3..db9245dc -- ninjascript provider/ninjatrader trader/ninjatrader trader/armed_executor.go kernel mcp config store` ③ `git am` (never --abort mid-series) ④ `md5sum ninjascript/*.cs` must match ⑤ `go build && go vet && go test ./...` + goldens ⑥ secret-scan the applied diff ⑦ owner pushes (`push` classifier-blocked) ⑧ re-verify `isAccountTradeable` parity. Secrets: `.env` never in history; 12 reports + 13,755 log_events rows clean. SIM-only: `isAccountTradeable` (tcp_trader.go:276-296) + `Sim101` pin intact; `LFE05060792090077` (live) untradeable.

---

## COMBINED P1 + P2 VERDICT

**Ready for the Sep-3 full run: YES — conditional.**
- **Machine layer (P1):** sound after the pre-reopen hotfix — S1 scenarioConds (9 conditions) and S2 GORM stall both fixed, deployed `db9245dc`, boot acked, goldens PASS, zero ERRO. Pending only live-fire E-proofs that Sunday's reopen produces (first 8th-condition authoring, fast-market trigger, F6 dedup, FIX-1 session-end cancel).
- **Money layer (P2):** the gate suite is a net saver (−$511.8 replay-adjusted) and **no knob change is justified by week-n evidence** — every sweep KEEPs. The week's loss was edge location (pre-NY −$353.5), giveback ($719.5), and one contract gap (waterfall immediate-mode), not broken machinery.
- **What's missing before Sunday 17:00 CT:** owner ruling on breakdown_continue **immediate-mode** (S4); owner ack of the two canon fixes (S1, already committed); optionally Binnie partner sync (S7) — not Sep-3 blocking.

**One-line system verdict:** *Machine sound, gates saving, zero knobs to move — the money leaks are the pre-NY session edge, the giveback, and a waterfall contract that can't catch a no-retest crash; fix those with rulings, not knobs.*

---

## Fixes landed this session (orchestrator, on dev)
- `2b2eab2b` — `deploy/RELEASE` marker commit (cutover closure).
- `dbcb61ac` — `GUIDE_BUILT_REV` `8666db0b` → `db9245dc` (drift banner closed; Guide vitest 10/10 · tsc 0).

## EVENT-WAIT registry
First trail exit → trail-mult sweep · first live fast-market trigger → N2 re-judge · Sunday first 8th-condition authoring · F6 dedup live proof · FIX-1 working-arm session-end cancel · rearm (hysteresis) · DB recomputes (VWAP residual, missed-turns@0.3, stamp-gap, touch 50-sample).
