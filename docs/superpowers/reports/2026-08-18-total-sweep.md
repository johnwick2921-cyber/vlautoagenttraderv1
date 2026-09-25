# TOTAL SWEEP — error census, autopsies, Asia verdict, leftovers, exposure @ HEAD cd5fd153

1. **8 distinct error types · 0 provably cost a trade · 0 fixed (all sized below) · ASIA actually enabled: YES — plan reaches the executor (receipts).** Binary `8d5cfa1f` is one commit behind HEAD (docs-only commit; no code delta). 0 positions (flat). Logs exist only 2 days — "30d" = DB 30d + logs 2d, stated where it matters.

2. **ERROR TABLE (ranked by cost):**
   | type | first→last | count | meaning | cost | verdict |
   |---|---|---|---|---|---|
   | AI API call failure (one Aug-12 outage) | 08-12 06:06→09:07 UTC | 122 | provider down ~3h | 122 decisions lost that morning | transient, self-healed |
   | guardrail_skip: stale_bar_discarded | 08-17 08:45 CT → ongoing | 21 | AI call spanned a primary-bar close → decision dropped by design (2b4162f6) | 21 decisions dropped, content NOT preserved in DB; 2 P1 alerts | EXPECTED-BY-DESIGN · gap sized (5g) |
   | safe-wait fallback (prose→wait) | 30d | 68 | model emitted no JSON → wait | mostly pre-2ddf3a58 era | FIXED-THIS-ERA, 1 tonight |
   | guardrail_skip: schema_parse_failed | 08-17 18:39 CT | 1 | 3 prose attempts, retries fired, fail-closed HOLD | 1 decision lost (nothing to parse) | machinery worked as designed |
   | AI API failures (scattered) | 08-04→08-14 | 7 | transient | 7 decisions | transient |
   | plans fail_closed / no_trade | 08-15, 08-16 | 2 | planner fail-closed; replans_exhausted | known incidents | fixed since |
   | FATA/❌ at shutdown | 2d | 23+23 | graceful-restart artifact, not a crash | none | cosmetic |
   | claw402 payment-402 re-sign | 2d | ~10 | x402 wallet re-sign retries | none (self-heals) | expected |

3. **2a guardrail_skip:** exactly two causes — stale_bar_discarded (21) and schema_parse_failed (1). Nothing else hides behind it. **2b schema_parse_failed autopsy:** 18:35:46 cycle → 3 attempts (18:36/18:38/18:39), each "no decision JSON in full text", retries fed the error back, prose-JSON recovery had nothing to recover, fail-closed HOLD with alert. 0 entries lost tonight.

4. **PART 3 ASIA:** enabled at every deciding site (sessionRunnable single-source; card says ASIA/LONDON/NY runnable) · plan written 00:07 CT · tonight's prompts 17:14 CT+ ALL carry `# DAY PLAN (ASIA)` · buckets tonight: 22 waits, 2 stale-discards, 1 parse-loss, 0 entries, 0 refusals. Waits trace to setup quality (S1/S2 invalidated, S3 targets above price).

5. **5a demo trade:** position id 520 (trader `1ef40f05`, +$224.50 grade A) still in closed history/equity. SIZED (S): sqlite backup → delete position 520 + its 2 fills → verify 179→178, P&L, GPA. Not run — needs the owner's flat-window go.

6. **5b MAE/MFE = 0 on real closes:** path is WIRED (recordClosedTradeAnalytics, auto_trader_clock.go:273/351 → UpdateExcursion) and untested-by-me live; root cause is not a bug — the last real close was **Aug 13**, before the W5 analytics shipped. Unexercised, not broken; resolves at next real close.

7. **5c min_confidence:** stored **60** live; read identically by gate (engine_analysis.go:407) and prompt (engine_prompt.go:85/114/119/153) — no site drift, drift vs the 65 contract only. Owner decision, untouched.

8. **5d two prices:** one prompt, two bars snapshots — KEY LEVELS/SVP fetched at engine_analysis.go:333, PLAN STATUS fetched separately at :360 (~2pt/~2min apart). SIZED (S–M): fetch once, reuse.

9. **5e two death predicates:** prose "Plan dies if" vs system all-levels-touched+accepted are different tests (plan_lifecycle). SIZED (M): one predicate or card shows both.

10. **5f lows:** S3 stale armed dot (freeze at flat) · NY v3 trigger mislabel (owner reset logged as scheduled read) · orphan `scenario_status:2026-08-17:NY` key · closes-beyond counting pre-birth bars. SIZED (S each).

11. **6a/6b guardrails:** OFF = daily loss $450 / profit $900 / 3-trades-day / blackout / consecutive-loss-halt all inactive; ACTIVE = size caps (2 contracts/order), armor, R:R 3, conf gate, hold-lock, session/last-entry/flat, T1, boot integrity. Exposure quantified: **zero fills since Aug 13 → nothing would have tripped; exposure is latent, only size caps + SIM bind today.** 6c soft-alert (evaluate + alert instead of block) SIZED (M) — recommended before re-arming; re-arm condition: before real capital.

12. **PART 4 self-announcing:** existing surfaces already announce P1 stalebar/burned/reset alerts + gate-block counters + dark-regime warnings; the gaps are silent INFO/WARN classes (safe-wait, prose retries) and no single panel. SIZED: 4a error-event telemetry (M) → 4b dashboard errors panel (M) → 4d daily digest line (S). 4c swallow sweep: bare `_ =`/silent defaults found in parse-fallback and discard paths (named above); none hides a trade.

13. **Exit bar:** build/vet/test not run (no code changed); goldens untouched; config untouched. Deploy NOT performed — zero code changes shipped; the sized items need a dedicated session at a flat window.
