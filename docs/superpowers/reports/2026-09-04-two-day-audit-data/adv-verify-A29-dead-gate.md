# Adversarial verification — A29 "DEAD GATE" (armGateVerdict)

Source tree: /home/hoang/nofx-2day04 @ 24685b70; `git diff --stat dfbfa660 HEAD -- trader/ kernel/` = EMPTY,
so trader/ is byte-identical to dev tip dfbfa660 (= deployed rev 530009ff / boot 7). [A]

## Reproduced exactly

    cd /home/hoang/nofx-2day04 && grep -rn "armGateVerdict" --include=*.go .

- `armGateVerdict` defined trader/armed_executor.go:1268; comment at :1266 says "(legacy shape)".
- 8 call sites, ALL in trader/armed_executor_test.go: 77, 82, 87, 94, 100, 180, 185, 189. ZERO production callers. [A]
- No bare method-value reference either (grep was run WITHOUT the paren). [A]
- Production calls `armGateVerdictFor` at trader/armed_executor.go:415. [A]

## Corrections

### C1 — line number
`composeArmStop` is called at **trader/armed_executor.go:375**, not :369. Line 369 is the first line of
the 6-line comment header above it. [A]

    sed -n '365,378p' trader/armed_executor.go | cat -n

### C2 — only 6 of the 8 assertions are stop-sensitive
Gate order inside `armGateVerdictFor` (:1279-1340): ArmSpecValid -> direction -> plan_mode -> quality ->
R:R -> min-SL -> HTF veto. Test lines :94 (quality C below floor B) and :100 (short arm vs long bias under
plan_mode=direction) return at gates that run BEFORE the R:R gate and do not read `leg.Stop` at all.
Stop-sensitive: 77, 82, 87, 180, 185, 189 = 6 of 8. [A]

Separately: all 8 calls pass `atr5m = 0`, and the min-SL leg is guarded by `if atr5m > 0` (:1319), so the
OTHER stop consumer never runs in any of the 8. [A]

### C3 — "the AUTHORED stop production never gates on" is false
`composeArmStop` is documented and fixture-pinned as "never tighter than authored"; its `bound="authored"`
branch leaves the stop unchanged, and :375 only assigns `leg.Stop = comp.Stop` when it moved.
`composeArmStop` first shipped 4657560b 2026-09-02 07:33:39 CT, so only ledger arms 32..37 are post-composition:

    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
      "SELECT a.id,a.plan_id,a.version,a.scenario,a.side,a.leg_index,a.entry_px,a.stop_px,a.target_px,p.doc
       FROM armed_orders a JOIN plans p ON p.plan_id=a.plan_id AND p.version=a.version WHERE a.id>=23 ORDER BY a.id;"
    (composed stop_px compared field-by-field against doc scenarios[].arm.stop / .legs[i].stop)

    id 32 09-02:NY v5  S3 long  authored 29033    composed 29019.6691311683  WIDENED
    id 33 09-02:NY v12 S3 short authored 29199.5  composed 29199.5           SAME
    id 34 09-02:ASIA v13 S1 short authored 29226  composed 29226.0           SAME
    id 35 09-03:NY v2  S1 short authored 29340    composed 29351.6284728996  WIDENED
    id 36 09-03:NY v3  S2 short authored 29418.75 composed 29418.9343631416  WIDENED
    id 37 09-03:NY v7  S3 short authored 29592.5  composed 29592.5           SAME

n=6 post-composition arms: composed == authored in 3, widened in 3. [A]
(ids 23..31 are 09-01, i.e. pre-composeArmStop; all 9 SAME by construction, excluded.)
Survivorship caveat: this population is arms that PASSED the gate; arms widened into a refusal never row up.
Even so, "never" is refuted -- production gated on the exact authored stop for 3 of the 6.

### C4 — the live counterfactual "would PASS" is REFUTED
Plan row (authoritative, not the log):

    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
      "SELECT json_extract(doc,'\$.scenarios') FROM plans
       WHERE plan_id='2026-09-02:NY:8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265' AND version=4;"
    -> S1 long B arm {enabled:true, entry:29135.65, stop:29099, target:29209.25, wait_confirm:true}

v4 created_at = 2026-09-02 14:45:52 UTC = 09:45:52 CT; v5 = 10:22:33 CT -> the 09:46:30 and 09:55:01 CT
lines ARE under NY v4. [A]  Their "NY v4 S1" is correct.

Arithmetic (theirs, reproduced): reward 73.60; authored risk 36.65 -> R:R 2.0082; composed risk 77.40 -> R:R 0.9509.
Log lines (grep -n on /home/hoang/nofx/data/nofx_2026-09-02.log, timestamps already CT):

    :25023 09-02 09:46:30 [INFO] 🛑 arm stop NY S1 leg 1 long: stop 29058.25 (authored 29099.00 WIDENED)
           · anchor PDC 29058.75 → beyond 29058.25 · atr_floor 29081.91 (1.5×ATR5m 35.82) · bound=anchor
    :25126 09-02 09:55:01 [WARN] ⚔️ arm REFUSED NY S1 leg 1: R:R 0.95 below arm min 2.00 · rr refusals this session: 1

**But** `armStopCompositionLine` (trader/arm_stop_anchor.go:165) formats `"%.2f (%.1f×ATR5m %.2f)"`, so
35.82 is ATR5m, and the min-SL floor distance is 1.5 × 35.82 = **53.73**. The authored risk is 36.65.
36.65 < 53.73, so with the AUTHORED stop the very next gate in the SAME function (min-SL, :1319-1327)
refuses it. The bot said so itself 9 minutes earlier:

    :25004 09-02 09:45:52 [WARN] ⚔️ arm feasibility: S1 arm stop 29099.00 too close (36.65 < 52.66 = 1.5×ATR5m)
           — min-SL gate will refuse it (WARN — write proceeds; the gate-at-arm chain enforces)

So the authored stop does NOT pass the gate. Only the refusal STRING changes (R:R -> "too close").
Zero behavioural difference on the cited case. [A]

### C5 — the production-shape path IS pinned by a fixture
trader/zerob_exit_sanity_test.go does `composeArmStop(...)` -> feed the composed stop -> `armGateVerdictFor`:

    :145-157 TestZeroBRRGateRefusesWithTheWiderStop  ("authored R:R 3.0, composed R:R 1.0, must be refused")
    :60,:70  TestZeroBPinStopFloor                    (min-SL 1.5×ATR on the production function, atr5m=20)

    cd /home/hoang/nofx-2day04 && go test ./trader/ -run \
      'TestArmedGateRefusesBadRR|TestArmedGateRRShortTwin|TestZeroBRRGateRefusesWithTheWiderStop|TestZeroBPinStopFloor' -v
    -> all 4 PASS (0.861s)

### C6 — "DEAD GATE" is a misnomer
No gate is dead: the live chain fired and blocked at 09:55:01 CT. What is dead is a legacy single-arm
CONVENIENCE WRAPPER that delegates straight to the production function on line 1272. Test-hygiene wart,
not a gate that fails to run.

## Knob values used — all read live, none from a default
- arm R:R floor 2.00 — read off the 09:55:01 CT log line itself, not from `resolvedMinRR`. All rows in
  `strategies` have `risk_control.min_risk_reward_ratio` NULL, so quoting a DB value would have been wrong. [A]
- min-SL mult 1.5 — printed in both the 09:45:52 and 09:46:30 lines. [A]
