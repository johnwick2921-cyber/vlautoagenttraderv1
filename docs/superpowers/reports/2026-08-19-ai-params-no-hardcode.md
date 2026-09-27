# AI PARAMS — NO HARDCODE REPORT
max_tokens 32768 · finish_reason=stop 33/34 · median completion 1320 tokens · median latency 67s · 8 hidden defaults found and fixed

## Verdict table (every AI parameter, audited)
| Parameter | Before | After | Verdict |
|---|---|---|---|
| max_tokens | hardcoded 2000 (silent truncation disease) | AI_MAX_TOKENS=32768 | WRONG-VALUE-FIX-IT |
| temperature | const 0.5 | AI_TEMPERATURE, default 0.5 | HARDCODED→CONFIG |
| top_p | never sent | AI_TOP_P, 0=omit | HARDCODED→CONFIG |
| timeout | const 300s | AI_TIMEOUT_SECONDS | HARDCODED→CONFIG |
| max retries | const 3 | AI_MAX_RETRIES | HARDCODED→CONFIG |
| retry backoff | const 2s | AI_RETRY_BACKOFF_SECONDS | HARDCODED→CONFIG |
| taskstate summary cap | const 1200 | AI_TASKSTATE_SUMMARY_MAX_TOKENS | HARDCODED→CONFIG |
| taskstate incremental cap | const 500 | AI_TASKSTATE_INCREMENTAL_MAX_TOKENS | HARDCODED→CONFIG |
| replanner cap | literal 500 | AI_REPLANNER_MAX_TOKENS | HARDCODED→CONFIG |
| model routing | provider echo verified | deepseek-v4-pro | CONFIG-DRIVEN-AND-SANE |
| roles / stream=false | sane | sane | CONFIG-DRIVEN-AND-SANE |

## Budget rationale
Provider ceiling probed: max_tokens valid [1, 393216] (1048576 → HTTP 400). Chosen
32768: ≥8× any observed completion, comfortably inside 180s decision timeout and
5-min bar cadence; 393216 risks timeout + queue lag. Env-overridable.

## Guards installed
- Startup audit: main.go logs every effective knob + WARNs any unset default.
- finish_reason=length → WARN + counter (surfaced at startup).
- Per-call log: completion tokens + finish_reason (journalctl-greppable).
- Guard tests: mcp/config_no_hardcode_test.go (wire reflects cfg, no literals).
- Decision-first contract: <decision> MANDATORY before reasoning in all 4 builders.

## Live verification
Boot 11:39:11 CT rev 5b1a9927: 34 completed decision calls by 14:40 CT —
finish_reason=stop 33/34, 0 length (max completion 7908 tokens, median 1320);
median AI latency 67s, max 149s (within the 180s cap, no 5-min overruns).
Follow-up finding (same day): DeepSeek 402 Insufficient Balance killed 139
cycles 02:00–07:00 CT — owner must top up / auto-recharge. Owner next action:
decide explicit values for the 8 unset env knobs.
