# LONDON zero trades — post-mortem (session 2026-08-17)

1. **VERDICT: DEFECT, not correct silence.** The running binary REFUSED ALL TRADES at boot (stale binary), and the H8 bug withheld the LONDON plan from the AI and would have vetoed any entry. The model also chose wait on all 94 cycles, so no entry was actually lost to a gate — but the bot could not have traded even if the AI had wanted to.

2. **Running binary ≠ HEAD.** Running: rev `8e7b816a` (built Aug 16 20:17 CT, dirty tree, booted 21:14:30). HEAD: `9199e9f0` — the binary is **18 commits behind** and lacks `2ddf3a58` (prose-JSON recovery), `570c6c32` (H8 session single-source) and `2b4162f6` (latency cap).

3. **Boot refusal (the wall).** At 21:14:30 on Aug 16 the boot assertion refused: *"binary is revision 8e7b816ab7f2 but the intended release is 184fe200b37d"*. `deploy/RELEASE` was edited to `184fe200` at 21:14:27, but the binary was never rebuilt. The boot_integrity gate blocks every entry until a matching binary is deployed.

4. **Plan: written, alive, never died.** LONDON plan written at its read time (01:58 CT, scheduled 01:55): v1, lifecycle `active`, 7 levels (ONH 30327.50 → PDL/ONL 30166.25), 3 scenarios, long bias. Death logic re-checked it every cycle; it never fired (this binary already has the post-plan, 5m-aggregated death logic).

5. **Cycles fired.** 94 decision records in the 02:00–08:30 CT window (trader "hoang" 70, trader "15m" 24). Cadence normal; night mode/scheduler were not involved.

6. **Decisions.** 93 genuine waits, 1 prose-only safe-wait, **0 entries proposed, 0 gate refusals**. The waits were mostly sane — price broke below the plan's line-in-sand (EQH 30210) and the long bias flipped, e.g. "no confirmed bullish reversal candle yet — price still under pressure", "1h bar strongly bearish", "oversold but no reclaim of the level".

7. **The 1.2% class.** 1 prose-only/unparseable decision (1.1% of cycles) at 05:32 CT — and it was the **slowest call of the window (131 s)**. Not elevated tonight, but the correlation holds.

8. **H8 is live in the running binary.** Strategy override says LONDON `enable=true` (DB), the admin registry default says `enabled=false` (UI endpoint confirms, `is_default:true`), and in this binary the registry flag vetoes: (a) `installActivePlanProvider` returns nil → the AI cycled the whole session with **no day plan in its prompt** (0 of 27,193 stored prompts for this trader ever contain the plan block); (b) `sessionGateDecision` blocks any LONDON entry ("LONDON session not enabled"). Fixed by `570c6c32` — not deployed.

9. **Latency.** n=95, avg 60.3 s, p50 57.3 s, p95 99.5 s, max 131 s; 39 (41%) over 60 s, 3 over 120 s, 0 over the 5-min bar. Slightly above the 14-day normal baseline (avg 51.2 s), far below the failure class (avg 110.3 s).

10. **Acceptance sanity.** Rule in force: "2x5m" (strategy + LONDON override). The interval fix exists only in the DEATH path (`plan_lifecycle.go` aggregates 1m→5m). **Still counted on 1-minute bars** in the executor PLAN STATUS tail (`kernel/engine_analysis.go:366` → `RenderPlanStatus` → `EvaluateLevelFacts`) and the UI card (`api/handler_plan.go` `planLevelFacts`/`liveStatus`) — half-fixed, live at HEAD too.

11. **Fix, sized.**
    - REQUIRED (owner action): rebuild at HEAD **and update `deploy/RELEASE` to the new rev in the same step** — otherwise this exact refusal repeats. Bot is flat; pre-NY (08:30 CT) is the safe window.
    - After deploy: H8 gone (`570c6c32`), prose recovery lands (`2ddf3a58`), latency cap lands (`2b4162f6`).
    - Still open at HEAD: acceptance facts on 1m bars in prompt tail + card — small code fix, but it changes prompt bytes and needs golden updates.

12. **Next-session expectation.** With a matching binary, LONDON reads reach the AI prompt and entries gate normally. Expect waits to stay common while price sits below the plan's invalidation line — that part was the model doing its job.
