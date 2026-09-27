# Exact proposed cuts/replacements — never applied

These are policy/authority/evidence/format changes proposed separately from the semantically preserving numbered rewrite. Quote boundaries are exact substrings of the adjacent current-contract planner replay or actual executor prompt.

## 1. weekday conviction
Source: `planner-132-current-contract-replay.txt:254`

```text
Conviction: down on Monday, up Thursday/Friday.
```

## 2. week rates
Source: `planner-132-current-contract-replay.txt:271`

```text
Condition×session guidance (week evidence): reject-based setups are best in NY RTH (75% win, +665 this week); acceptance needs a clear displacement or skip (0% win this week); sweep_reclaim requires the reclaim CLOSE on the decision TF, never the wick alone (0% win this week).
```

## 3. acceptance prior
Source: `planner-132-current-contract-replay.txt:257`

```text
acceptance entries WITHOUT a prior sweep + displacement are 0% win evidence — skip them (A5).
```

## 4. arm escape hatch
Source: `planner-132-current-contract-replay.txt:271`

```text
If your setup cannot meet BOTH, OMIT arm{} and let the AI path take it.
```

## 5. immediate authority
Source: `planner-132-current-contract-replay.txt:271`

```text
entry_mode=immediate is AI-path ONLY (no arm; the machine rejects immediate arms): the market entry fires on the CONFIRMING close (BD_MIN_CLOSES, default 1) and runs the FULL gate chain (min-SL ≥ 1.0×ATR5m, R:R ≥ min_risk_reward_ratio, min-conf, HTF veto) — CHOOSE immediate for no-retest waterfalls (one-sided delivery, displacement EXPANDING, price running away from the level): SL beyond the pullback extreme, target at the next liquidity pool.
```

## 6. obsolete side quota
Source: `planner-132-current-contract-replay.txt:263`

```text
MUST include ≥3 below AND ≥3 above the current price
```

## 7. false merge and boundary
Source: `planner-132-current-contract-replay.txt:271`

```text
levels MUST be at least 3 points apart — near-duplicates are merged by the system
```

## 8. stale waterfall floor
Source: `planner-132-current-contract-replay.txt:271`

```text
min-SL ≥ 1.0×ATR5m
```

## 9. off-plan heading
Source: `executor-37768-system_prompt.txt:34`

```text
# DAY PLAN (NY) — preferred: follow it · a valid off-plan setup may still be traded (cite "off-plan")
```

## 10. off-plan citation option
Source: `executor-37768-system_prompt.txt:110`

```text
- `cited_scenario`: REQUIRED on every open when a DAY PLAN is shown — the plan scenario id ("S1"…) you are trading, or "off-plan" for a valid non-plan setup. (A6/F12: this used to live only inside the plan block; a contract-literal model omitted it and every adherence grade silently degraded to D.)
```

## 11. open example
Source: `executor-37768-system_prompt.txt:87`

```text
<decision>
```json
[
  {"symbol": "MNQ", "action": "open_long", "leverage": 1, "position_size_usd": 60000, "stop_loss": 21480.00, "take_profit": 21560.00, "confidence": 80, "cited_scenario": "S1"}
]
```
</decision>
```

## 12. promised retrace arm
Source: `executor-37768-system_prompt.txt:112`

```text
- ARMED PATH (autopsy-response): if a scenario confirm is MET and you decline on timing/extension grounds, the retrace is already covered by the plan's wait_confirm arm (it rests at the retrace level and fills without you) — prefer leaving that arm live over waiting for a cleaner touch; do not chase, and do not skip the retrace.
```

## 13. generic stop range
Source: `executor-37768-system_prompt.txt:19`

```text
- Stop distance: roughly 1.5-3x ATR; sanity range ~15-50 points from entry.
```

## 14. breakeven and hold-time beliefs
Source: `executor-37768-system_prompt.txt:28`

```text
-Focus On Quality trtade and 50 point  move stop loss to breakeven 
- Excellent trader: 2-4 trades per day ≈ 0.1-0.2 trades per hour
- >2 trades per hour = overtrading
- Single position holding time ≥ 30-60 minutes
If you find yourself trading every cycle → standards are too low; if closing positions in <30 minutes → too impulsive.
```

## 15. reasoning first instruction
Source: `executor-37768-system_prompt.txt:74`

```text
3. Write your chain of thought, THEN output the structured JSON decision.
```

## 16. reasoning first example
Source: `executor-37768-system_prompt.txt:83`

```text
<reasoning>
Your chain-of-thought analysis of the MNQ bars and indicators.
</reasoning>
```
