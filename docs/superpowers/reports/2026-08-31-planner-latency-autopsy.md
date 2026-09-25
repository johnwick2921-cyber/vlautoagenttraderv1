# 2026-08-31 — Planner Latency Autopsy (read-only)

No code, config, or deploys touched. Evidence = journalctl (systemd `nofx`),
read-only DB, source reads. Report branch `docs/planner-latency-autopsy`.

## TL;DR verdict

**The dominant eater is the model's completion generation at `reasoning=max`:
~26.4k output tokens per planner attempt at ~63 t/s ≈ 420s of pure provider
stream time. Our-side pre (T0→T3) is 0-1s and our-side post (T5→T8) is
sub-second — combined <0.5% of the wall clock.** The executor proves the
ceiling: the same model, same session, *larger* prompts (~10.5-11.5k tokens vs
the planner's ~9.4k) but `reasoning=fast→low` and 100-2.2k-token outputs
finish in 3.5-36s. The planner is not slow per token (65 t/s ≈ executor's
71 t/s at its biggest); it is slow because each attempt emits ~26k tokens and
a rejected read emits them up to 3×.

## Per-attempt table (T0-T8)

Legend: A = LONDON proof read (old binary `5d7be58a`, owner reset 07:18:15).
B = LONDON read (new binary `e86ae805`, owner reset 08:14:44). N = NY
scheduled read (new binary, fired 08:25:51). X = last night's 600s timeouts.
All planner calls: `reasoning=max wire=enabled/max cap=65536` (logged per
call), full-body HTTP (no SSE). Tokens = provider `usage` (prompt/completion).

| Att | T0 trigger | T3 request | T5 complete (wall) | dur | in-tok | out-tok | finish | verdict / write |
|---|---|---|---|---|---|---|---|---|
| A1 | 07:18:15 OWNER RESET | 07:18:16 | 07:27:26 | 550.3s | 9430 | 30955 | stop | reject parse/schema (confirm2.rule) |
| A2 | — | 07:27:26 | 07:32:46 | 319.9s | 9519 (+89) | 18721 | stop | 2× side-quota WARN (old bin) → reject breakdown-void |
| A3 | — | 07:32:46 | 07:40:35 | 469.5s | 9490 (+60) | 27752 | stop | PASS → **LONDON v2 active** (DB 12:40:35.636Z) |
| B1 | 08:14:44 OWNER RESET | 08:14:44 | 08:21:51 | 427.3s | 9423 | 28002 | stop | reject breakdown-void |
| B2 | — | 08:21:51 | 08:27:12 | 321.2s | 9483 (+60) | 21152 | stop | reject split-arm (1 leg) |
| B3 | — | 08:27:12 | 08:33:54 | 401.5s | 9471 (+48) | 27869 | stop | reject confirm=touch → **LONDON v3 no_trade** (DB 13:33:54.052Z) |
| N1 | 08:25:51 "dark regime at the NY read" | 08:25:51 | 08:34:13 | 502.0s | 9377 | 33082 | stop | reject arm-legs |
| N2 | — | 08:34:13 | 08:40:22 | 368.7s | 9439 (+62) | 23395 | stop | reject breakdown-void |
| N3 | — | 08:40:22 | 08:47:19 | 417.0s | — | — | stop | 1st HTTP died mid-stream (transport reset @272s) → client retry won → reject arm-legs → **NY v1 no_trade** |
| X1 | — | 21:10:01 | 21:20:01 | 600.0s | — | — | n/a | client deadline, mid-stream (below) |
| X2 | — | 23:20:11 | 23:30:11 | 600.0s | — | — | n/a | client deadline, mid-stream (attempt 3/3) |

T1 (map assembly) and T2 (prompt render) have **no duration logs**; their
combined footprint is the T0→T3 gap, which is 0-1s in every observed case —
measured, not estimated. T4 (first token) has no log — MISSING. T6 (parse)
has no log; T7 (validator verdict) and T8 (write) are same-second as T5, and
the DB `created_at` stamps confirm the persist is sub-second.

Reject-append block size: +48 to +89 tokens per retry prompt (avg ~60) —
the CHANGE-2 block is negligible.

### Executor baseline (same model, seconds)

| time | prompt tok | completion tok | duration | mode |
|---|---|---|---|---|
| 07:22:41 | 10495 | 330 | 7.4s | fast→low |
| 08:16:20 | 11437 | 1475 | 25.2s | fast→low |
| 08:28:00 | 11529 | 8769 | 124.3s | fast→low (biggest exec call; 71 t/s) |
| 08:36:01 | 10508 | 206 | ~4s | fast→low |

Executor prompts are ~1-2k tokens BIGGER than planner prompts and still
complete in seconds. The delta profile = output size × reasoning effort, not
prompt size.

## Attribution (T0-T8)

- **Our-side pre (T0→T3): 0-1s (~0.2%).** Reset/wake → request sent in the
  same second (A1: 07:18:15→07:18:16; B1 and N1: same-second).
- **Model (T3→T5): 319.9-600.0s (~99.5%).** Average successful completion
  420.0s (n=8); worst non-timeout 550.3s.
- **Our-side post (T5→T8): sub-second (~0.3%).** Verdict + `PLAN written` +
  DB row all within the same journal second (e.g. 08:33:54 for reject, fail-
  closed, write; DB 13:33:54.052Z).
- **Queue (T3→T4) vs think+stream (T4→T5): UNKNOWN — no first-token log.**
  [C] inference: throughput is strikingly flat across attempts (30955/550s,
  18721/320s, 27752/469s, 28002/427s, 21152/321s, 27869/401s, 33082/502s,
  23395/369s ≈ **63-66 t/s on 7/8, and 56 t/s on one**), which is
  consistent with a small queue and near-linear generation time. A queue-
  dominated profile would show erratic throughput. Label: inference, not
  proof — it needs the T4 instrumentation.

## The 600s timeout post-mortem

Two events, 2026-08-30 evening (binary `746955` / `848146`, same client
config):

```
21:20:01 ai_call model=deepseek-v4-pro duration_ms=600000 finish_reason=n/a
        ok=false timeout_source=client deadline_s=600
        err="failed to read response: context deadline exceeded
        (Client.Timeout or context cancellation while reading body)"
        → planner attempt 1/3 failed
23:30:11 same signature (600001ms) → planner attempt 3/3 failed
```

- **Which segment died: mid-stream.** The error is "while reading body" —
  the request was sent, headers/body were in flight, and the read was still
  open at the deadline. NOT a connect/queue failure (that would be "failed
  to send request" or a transport dial error).
- Stall vs slow trickle is indistinguishable without T4/final-token logs —
  that ambiguity is the single most valuable instrumentation fix below.
- Both timeouts consumed a full attempt; X2 was the third attempt, so the
  whole read died after ~25 min with no plan.
- **Third failure class (2026-08-31 morning): mid-stream transport reset.**
  NY attempt 3's first HTTP call died at 08:44:54, 272s in:
  `ai_call … duration_ms=272226 ok=false timeout_source=transport deadline_s=600
  err="failed to read response: read tcp 10.0.0.141:45938->3.173.21.63:443:
  read: connection reset by peer"`. The client's fixed retry flow (retries=2,
  backoff=2s) re-sent it and the retry won — the attempt completed at
  08:47:19 (417.0s wall incl. the retry) and the plan then failed on a real
  validator defect. Contrast with the client-deadline class above, which is
  NOT recoverable in-loop: the 600s ceiling is the whole call, so it burns
  the full attempt with zero output salvaged.
- Context: a bare `Gateway Timeout` line at 20:13:02 the same evening
  (provider-side gateway trouble) suggests upstream pressure that night.
- The only deadline is `http.Client.Timeout = ResolvedAITimeout()` =
  `AI_HTTP_TIMEOUT_SECONDS=600` (mcp/config.go:41,67,76); the planner path
  does not use the 30s-idle-watchdog streaming client.

## Streaming check

The session planner calls `client.CallWithMessages`
(`trader/auto_trader_planner.go:920`), the executor calls
`CallWithMessages` (`kernel/engine_analysis.go:676`), weekly too
(`trader/auto_trader_weekly.go:172`). **Nobody uses
`CallWithRequestStream` (mcp/client.go:896, SSE with a 30s idle watchdog) in
production** — every path waits for the FULL body, so no first-token
visibility and no idle-stall protection exist today.

## Instrumentation gap list (missing timestamps → where they'd live)

1. **T4 first token / per-chunk stamps** — `mcp/client.go` body-read site
   (after `HTTPClient.Do`, before `io.ReadAll`); or adopt the SSE path.
2. **T1 map assembly duration** — after `collectMachineGrades`
   (`trader/auto_trader_planner.go:894`) / the `seated N/M` log
   (`kernel/levels_score.go:575`).
3. **T2 prompt render duration + token size at build** —
   `kernel.BuildPlannerPrompt` call site
   (`trader/auto_trader_planner.go:862`).
4. **T6 parse/schema pass duration** — around `ParsePlanDocCapped` in the
   retry loop.
5. **Reasoning tokens** — `mcp/client.go:402` logs only
   `Usage.CompletionTokens/PromptTokens`; the reasoner's think tokens (if
   the provider returns them) are dropped.
6. **Prompt persistence** — only a hash is stored (`prompt b992aeb6cce6`);
   no table holds the verbatim prompt, which is why the optional offline
   fast-vs-max A/B curl was SKIPPED per dispatch rules. Storing one verbatim
   rejected prompt would unlock that measurement.
7. **Client-internal retry evidence** — the fixed retry flow
   (`mcp/client.go` ~236-300, retries=2 backoff=2s) logged no "retrying"
   lines in any sampled window; per-call retry counts are not in the
   `ai_call` line. (Exception: the N3 transport reset — the retry engaged
   and won, proving the flow works for the transport class.)

## Ranked fix shortlist (recommendations only — nothing built)

1. **Instrument T4 + stream ticks first** — the queue-vs-generation split is
   unmeasurable today and decides everything else (the "latency mode" lever
   can't be costed blind).
2. **Completion-side diet** — planner outputs run 19-33k tokens per attempt.
   Bound `AI_PLAN_MAX_TOKENS` well below 65536 and/or shrink the emitted
   schema. At 63 t/s, capping 16k bounds worst case ≈ 250s/attempt. Watch
   `finish_reason=length` (the historical 32768 truncation disease) — cap
   after, not instead of, schema diet.
3. **Latency-mode experiment** — executor (fast→low, same model, bigger
   prompts, seconds) is the strongest available evidence that reasoning=max
   is where the think-time lives. Enable the offline A/B by storing one
   verbatim rejected prompt (gap #6), then curl fast vs max.
4. **Streaming + idle watchdog for the planner** — `CallWithRequestStream`'s
   30s idle cancel turns a 600s stall into a 30s abort + a retry carrying
   the verbatim reason (CHANGE-2 block already rides attempts ≥2).
5. **Retry economics** — a 3-attempt reject costs ~21 min. Attempt-2 carries
   the verbatim reason yet the model re-authored the same defect on most
   sampled reads (split-arm/breakdown-void repeated); consider fast→low for
   attempts 2-3, and measure attempt-2 success rate before touching counts.
6. **Split the deadline** — separate total-body ceiling (600s) from an
   idle-chunk deadline (~30s) so a live-but-slow stream is never killed and
   a stalled one dies fast.

## Evidence appendix (verbatim lines)

- `📊 AI call complete: completion=33082 prompt=9377 finish_reason=stop` —
  token counts are provider `usage`, mcp/client.go:402.
- `🧠 planner call (reasoning=max wire=enabled/max cap=65536) completed in
  502.0s` — planner loop, after the call returns.
- `ai_call model=deepseek-v4-pro duration_ms=600001 finish_reason=n/a
  ok=false timeout_source=client deadline_s=600 err="failed to read
  response: context deadline exceeded (Client.Timeout or context
  cancellation while reading body)"` — mcp/client.go:225.
- `🗓️ OWNER RESET 2026-08-31 LONDON — chain abandoned at v2; budget re-armed`
  (08:14:44) · `🌑 dark regime at the NY read` (08:25:51) — T0 markers.
- DB: `plans.created_at` 12:40:35.636Z / 13:33:54.052Z for LONDON v2/v3 —
  persist is sub-second after the completion line.
