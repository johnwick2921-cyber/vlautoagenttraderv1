# Class 49 — instrument honesty: the boot line that could not be wrong

**Dispatch:** CLASS 46 (checklist slot **49** — 45/47/48 landed first, A16). Owner hoang, 2026-09-02.
Audit basis: E2E DeepSeek audit @ 8c1e52ef. **Tiers:** [A] verified directly · [B] inferred · [C] speculation.
**Built in** `~/nofx-honest`, branch `fix/class46-instrument-honesty`. No lock held during the build.

## 0. Every premise verified before building (A23)

| # | Claim | Verdict |
|---|---|---|
| C1 | class-41 boot-line fields are literals | **TRUE [A]** — `trader/auto_trader_planner.go` `plannerStreamPolicyBootLine` printed `watchdog_log=on`, `serialize_executor=off`, `resend_identical=on` verbatim, and `keepalive=%ds` took **30** as a literal argument at the call site. Its fixture asserted the same strings. |
| C2 | `timeout_source` defaults to transport | **TRUE [A]** — `source := "transport"` at client.go:409, overridden for 4 sentinels only |
| C3 | `class=other` reaches the model | **TRUE [A]** — the old `IsProviderFailure` switch omitted parse/empty/too-long |
| C4 | the watchdog resets on comment lines | **TRUE [A]** — `onLine()` is called inside `for scanner.Scan()` before any data check |
| C6 | sockwatch never caught a FIN | **TRUE [A]** — 12,947 log lines, `grep -cE "CLOSE-WAIT|FIN-WAIT"` = **0** |

**This wave audits my own work.** Class 41 was mine; its boot line was the theatre, and its
fixture was the accomplice. Recorded here rather than softened.

## 1. D1/D7 — the boot line, field by field

| Field | Before | After — read from |
|---|---|---|
| tries | resolver ✓ | `StreamRetryTries()` |
| backoff | resolver ✓ | `StreamRetryBackoffSchedule()` |
| keepalive | **literal `30`** at the call site | `security.DialerKeepAlive`, the one place that sets it |
| observed keepalive | not printed (line implied set == seen; wire was 14-20 s) | `ObservedKeepAlive()` → **`observed=n/a`**, an honest gap |
| watchdog | **literal `watchdog_log=on`** | `WatchdogPreTokenSeconds()` / `WatchdogPostTokenSeconds()` |
| resend_identical | **literal `on`** | `ResendIdenticalOnProviderFailure()` |
| serialize | **literal `off`** | `SerializeExecutorDuringPlannerStream()` |
| storm_cap | absent | `StormCapPerRead()` |
| trace | absent | `TransportTraceEnabled()` |

**E1 RED, quoted against the class-41 line:**

```
CLASS 46 E1 RED: "watchdog_log=on" is a STRING LITERAL on the class-41 boot line …
CLASS 46 E1 RED: "serialize_executor=off" is a STRING LITERAL …
CLASS 46 E1 RED: "resend_identical=on" is a STRING LITERAL …
CLASS 46 E1 RED: "keepalive=30s" is a STRING LITERAL …
--- FAIL: TestClass46RedOnClass41Line
```

GREEN after: `TestClass46BootLineEveryFieldReadFromEnforcer` compares each field to its enforcer;
`TestClass46BootLineTracksTheResolvers` moves four env knobs and asserts the line moved with them —
a literal cannot pass that.

## 2. D2 — one classification (re-classification table)

`timeout_source` deleted. `ClassifyFailure(err, httpStatus)` is the only classifier and it sees the
status, so text sniffing can no longer misread a 503 body.

| failure shape | old `timeout_source` | class 49 | provider-side |
|---|---|---|---|
| peer FIN mid-body | transport | `transport` | yes |
| peer RST | transport | `transport` | yes |
| 503 Server Overloaded | **transport** | `http_5xx` | yes |
| 502 bad gateway | **transport** | `http_5xx` | yes |
| 429 rate limited | **transport** | `http_5xx` | yes |
| 400 invalid request | **transport** | `http_4xx` | no |
| 401 unauthorized | **transport** | `http_4xx` | no |
| no JSON object | **transport** | `parse` | **no** |
| unmarshal type error | **transport** | `parse` | **no** |
| empty 200 | **transport** | `empty_200` | yes |
| too long | **transport** | `too_long` | yes |
| planner ceiling | planner_total | `total_deadline` | yes |
| watchdog | transport | `idle` | yes |
| Client.Timeout | client | `total_deadline` | yes |
| no API key | **transport** | `auth_config` | no |
| caller cancelled | context | `context` | yes |

**11 of 17 fixture rows were mislabelled by the old default** — the audit's 23-of-50 proportion.
`ai_call` now carries `class=` and `provider_side=`.

**A correction I had to make mid-wave.** I first made `parse` provider-side, following the
dispatch's list too literally. The pre-existing class-41 fixture failed and was right: resending
an unparseable document with the identical prompt would loop on the same malformed output
forever. Only an ABSENT body (`empty_200`) is provider-side; a document the model wrote that will
not parse is its own defect, which is what the repair path is for.

## 3. D4 — a watchdog that can fire

Two timers, because "idle" means two things. **Pre-token** (600 s, DeepSeek's own ~10-minute queue
close): heartbeat comments legitimately reset it — a queued request is alive. **Post-token** (90 s):
reset ONLY by content/reasoning deltas; comment lines do not touch it. Its close raises
`ErrWatchdogIdle`, textually distinct from a peer's `unexpected EOF`.

E4 proves both directions: a stream emitting only heartbeats AFTER the first token dies at the post
limit (and the log names `post gap=`), while the same heartbeats BEFORE the first token carry the
call to completion.

## 4. D5/D6 — storm cap and the trace that replaced the blind loop

D5: provider calls are bounded **per read** (`AI_PLAN_STORM_CAP`, default 5), holding across the
three planner attempts; hitting it logs `🌩 storm cap reached`. E5 asserts the tries are spaced by
the schedule and that the cap holds across attempts.

D6: the sockwatch bash loop is **removed** (processes killed, `~/nofx-backups/transport-capture`
deleted per A25) after 12,947 lines and zero FIN/CLOSE-WAIT states. `httptrace` now reports
`closed_by=peer_fin|local_close|clean` with reused/idle/dial/ttfb/bytes/elapsed. **`closed_by` is
INFERRED and the line says so** — httptrace sees no TCP flags; the inference is sound only because
those are the two ways this reader stops early.

## 5. R0 — the rider (owner ruling)

`lawExcerptsForDoc` attaches `RepairConfirmVocabLaw` whenever the rejected DOCUMENT carries a
confirm object, regardless of the incoming error. Pinned against chain 4 (2026-09-02 14:23 CT),
where a repair of a void-breakdown defect introduced `confirm.rule "1x5m_close"` on a reject fade —
a confirm-rule violation it had never been shown the enum for. ~60 tokens on a ~1,200-token prompt;
not attached when the document has no confirm object, and never attached twice.

## 6. Tests

E1 (red quoted, then green) · E2 re-classification table · E3 via `TestClass41ProviderFailureClasses`
(corrected) · E4 both watchdog directions · E5 storm cap + spacing · E6 peer_fin vs local_close ·
E7 rider pin · **full `go test ./...` green · vitest guide 10/10 · `tsc --noEmit` clean**.

Nine pre-existing tests were migrated to the new vocabulary — each with the reason in the diff, none
weakened. One (`TestClass41ProviderFailureClasses`) caught a real defect in this wave's own code.

## 7. Cutover

_(pending: this boots after 45 and 48, which are already live; the wave is staged and green)_

## 8. What the owner will still see wrong (A15)

- **Until this boots, the running binary prints the class-41 literals**, including
  `keepalive=30s` while the wire runs 14-20 s. That field is wrong on the live boot line today.
- **`observed=n/a` is permanent for now.** Nothing measures the wire keepalive; the line says so
  rather than guessing. Measuring it needs a socket-level probe that does not exist.
- **`closed_by` is inferred, not observed.** No pcap on this box.
- **The storm cap is per-read and per-client.** Two traders sharing a provider row can still add up.
- **`breakdown_continue` void is the dominant live reject** — six today (12:57, 14:23, 18:00,
  18:13 and two on 09-01). Untouched by this wave; it is a prompt/authoring question (lane 45).
- **This wave is unproven live.** Its proof is the next provider cut (trace line + class + identical
  resend), the next 503 burst (spaced retries + storm counter), and the first watchdog fire. **None
  has occurred yet.**

## 9. Rollback

```
cp nofx-bin.prev.boot nofx-bin && echo <previous rev> > deploy/RELEASE && kill -9 <MainPID>
```
