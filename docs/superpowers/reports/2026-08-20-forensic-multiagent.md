# VL FORENSIC MULTI-AGENT VERIFICATION (2026-08-20)
**12 of 13 causes verified fixed live (1 open: owner's 3 prompt lines) · trade possible now: no (outside NY session; in-session yes) · 13 findings (4 can lose money or a trade) · 5 UNPROVEN (DB-level) · binary==HEAD: no (docs-only delta c2efefc9 vs bb966a049d9e, same code)**

## The five questions
1. Causes fixed live: #1 boot-integrity ✓ (boot line bb966a049d9e goldens PASS) · #2 H8 registry ✓ (sessionRunnable) · #3 padded bars ✓ (0 placeholders since boot, ingest refusal) · #4 death scope ✓ · #5 acceptance interval ✓ (single resolver + AST guard) · #6 false burns ✓ code; DB-level UNPROVEN · #7 advisory wording ✓ (plan block header permits off-plan) · #8 UTC/CT ✓ (tz AST guard) · #9 truncation ✓ (32768; 6/6 finish_reason=stop post-boot, median 3251 tokens) · #10 C2 ✓ (no d.Action=wait remains; feed-stamped; 6 killed entries re-verified) · #11 402 ✓ (owner top-up; 0 "Insufficient Balance" since 07:07) · #12 planner ✓ (two-sided levels live-proven at 18:28 arms) · #13 owner's 3 lines OPEN.
2. Trade possible: NOW no — session gate blocks outside NY (orders:241-246). Inside NY: yes — D+E traced parse→gates→feed-stamped SIM order→C# bracket; every late rewrite records a refusal.
3. Plan today: machinery yes (≥3 levels each side, reachable continuation, machine death/flip); two input gaps: auction story + owner note never populated (planner.go:943-950), planner's calendar section can be empty while the gate blackouts (fail-open-empty vs fail-closed).
4. Still suppressing? No silent entry rewrite remains; soft bias = owner's 3 lines + prompt discouraging stack (H verdict: wouldn't trade a 200-pt day) + F-1 unwindowed scenario status (can mislabel armed/invalid).
5. Left, by cost: (1) claw402 data-route 402 re-sign churn all day [money] · (2) hold-lock toggle OFF unknown — AI may close by opinion [safety] · (3) price armor + stale discard fail OPEN on data outage [safety] · (4) F-1 unwindowed scenario status [trade] · (5) planner dead input fields + calendar divergence · (6) dead toggles (max_contracts_enabled, notional_cap_enabled) shown in UI, 0 readers · (7) no lost-decision FE counter; gate-blocks/errors in-memory · (8) no overlay history; ErrorPanel has no FE consumer · (9) validateDecision rejections mislabeled schema_parse_failed · (10) session digests lack MAE/MFE (daily has it) · (11) intraday S/D zones never seat · (12) FE display fallbacks 75/10 vs safe 65/2 · (13) bypass-proof gate fixtures absent (sized S).

## 5-day replay proof table
| day | below/above (was→now) | scenarios triggered (was→now) | proposals | killed after proposal |
|---|---|---|---|---|
| 08-15 NY | 0/8 → ≥3/3 enforced | 0 → n/a (balance day) | — | — |
| 08-16 ASIA | 0/8 → ≥3/3 | 0 → 0 (flat tape) | — | — |
| 08-17 ASIA | 0/8 → ≥3/3 | 1 touch → 1 | — | — |
| 08-18 NY | 1/7 → ≥3/3 (live-armed) | 0 → ≥1 (forced breakdown short) | 7 (5L/2S) | 6 by C2 drift (fixed), 1 unproven |
| 08-19 (today) | live two-sided ✓ | — | 0 (outside session) | 0 |

## UNPROVEN (named)
DB-level only: exact 139 402 count · 0-of-7 burn justification · the 7th 08-18 proposal · a live plan doc quote · live WSL-vs-feed skew. Each provable by sqlite3 -readonly queries + `date`/powershell — tool-restricted agents could not run them.

## Wire (post-boot)
6 calls: finish_reason=stop 6/6 · completion median 3251 (max 5136) · latency median 52s (max 83s) · 0 over 180s/5-min · model echo deepseek-v4-pro · 10 reasonings re-categorized: 9/10 "no S1-S3 trigger", 5/10 oversold, 4/10 poor R:R, 3/10 owner's sideways line, 2/10 lunch.

## Section verdicts
A input PASS (zones die at levels_score.go:167-175) · B planner PASS machinery, 2 dead fields · C wire PASS · D decision PASS, 1 mislabel · E execution PASS, money-to-cent ✓, SIM lock complete · F 1 real bug (unwindowed scenario status) · G engine obeys owner; 2 dead toggles · H 5 claims: (a) 6-of-7 proven, (b) 35/35 stop not 33/34, (c)(d) DB-only, (e) contradicted — was 1/7 not 8/8.

## What agents did not touch
guardrails master OFF · min_confidence 60 · owner's 3 lines · DeepSeek keys — per §4.
