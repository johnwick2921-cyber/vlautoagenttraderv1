# Transport resets — who closes the socket, then survive it (class 41)

**Dispatch:** TRANSPORT RESETS — Phase 1 read-only forensics + Phase 2 fix. Owner: hoang, 2026-09-02.
**Worktree:** `../nofx-transport` (branch `fix/transport-resets`), main tree untouched during Phase 1, no lock held.
**Evidence tiers:** [A] directly verified · [B] inferred from strong evidence · [C] speculation.

## 0. Surprise logged (A23)

At 00:10:21 CT, while Phase 1 was running, the P&L-truth cutover swapped its binary before writing
`deploy/RELEASE`: the 00:10:20 boot came up **BOOT INTEGRITY REFUSED** (rev 23f56f49 vs expected
aeb11179) — the bot was up but in TradingRefused. I alerted the three sibling sessions and pushed a
notification; the lock holder (pid 1860416) re-stamped RELEASE and re-killed; boot 2 at 00:11:48 CT
is BOOT INTEGRITY OK. Refused window: 87 s. No action of mine touched the tree or the lock. [A]

## 1. Phase 1 — who closed the socket

### P1 — the idle watchdog (rule out ourselves first)

**a. Implementation** — `mcp/client.go` `CallWithRequestStreamDeadlines` (worktree lines ~1139–1195):
a goroutine owns `time.NewTimer(idle)`; `ParseSSEStreamFull`'s `onLine` callback (fires after every
`bufio.Scanner` LINE, blank and `: keep-alive` lines included) pushes to `resetCh`, which stops and
resets the timer. On fire it calls `cancel(ErrStreamIdleDeadline)` on a `context.WithCancelCause`
child that the HTTP request carries — the transport closes the connection. It measures **time since
the last SSE line**, not since the last byte (a partial line without a newline does not reset it).
Pre-fix it did **NOT log** when it fired; the only trace was the returned error, which
`wrapStreamDeadlineErr` labels from `context.Cause(ctx)` → `class=idle_deadline`. So "0 idle kills" in
the audit was **evidence after all**, because the label is derived from the cause, not from the
reader's text (see c). [A]

**b. Reset between client retries** — yes. The timer, its goroutine and the cancel-cause context are
created inside `CallWithRequestStreamDeadlines`, which the retry loop calls once per CALL; the
goroutine exits on `ctx.Done()` via `defer cancel(nil)`. Call 1's timer cannot carry into call 2. [A]

**c. Error string when the watchdog fires** — reproduced in `mcp/transport_cut_probe_test.go`
(fake SSE server, idle 300 ms, then silence):

| Event | Reader error string returned | class |
|---|---|---|
| watchdog fires | `stream idle deadline exceeded (idle 300ms of silence, context canceled): stream interrupted: stream idle deadline exceeded` | `idle_deadline` |
| peer FIN mid-body (hijack + Close) | `stream interrupted: unexpected EOF` | `transport` |
| peer RST mid-body (SetLinger(0) + Close) | `stream interrupted: read tcp …: read: connection reset by peer` | `transport` |

The watchdog is **not** a suspect for "unexpected EOF": Go ≥1.21 returns `context.Cause` as the read
error, and the wrapper checks the cause first. Tonight's three cuts carry `class=transport` and the
peer-FIN string byte-for-byte; the 01:46 cut carries the peer-RST string. [A]

**d. Gap before each cut** — not measurable from the journal: the SSE parser records only ttfb and
the final reasoning-char count, there are no per-chunk timestamps. Bound: any gap ≥30 s would have
fired the watchdog and produced `class=idle_deadline`; it did not, so every gap was <30 s. Average
rates were normal to the end (55k chars / 250 s ≈ 220 chars/s; 70k / 308 s ≈ 227; 3k / 18 s ≈ 170). [B]
The armed socket watcher (P4) records `lastrcv` (ms since last byte) every 250 ms, so the next cut
will have its gap.

### P2 — our transport (`security/url_validator.go:158-190`)

| Setting | Value | Can it close a long body? |
|---|---|---|
| `net.Dialer.Timeout` | = client timeout (600 s) | connect only — no |
| `net.Dialer.KeepAlive` | **30 s** | keepalive probes only; **confirmed in effect** (`ss -o` shows `timer:(keepalive,…)` on every nofx-bin :443 socket) — no |
| `ResponseHeaderTimeout` | unset (0) | headers only — no |
| `IdleConnTimeout` | unset (0 = never) | idle pooled conns only — could make a REUSED conn stale, but a stale reuse fails at request time, not after 55k chars — not these cuts |
| `DisableKeepAlives` | false | — |
| `MaxIdleConnsPerHost` | default 2 | pool size only |
| TLS config / `ForceAttemptHTTP2` / `TLSNextProto` | none set → **HTTP/1.1 only** (custom DialContext disables auto-h2) | no |
| `http.Client.Timeout` | 600 s, lifted to 0 per planner call (class 37) | excluded: lines say `class=transport`, not `client_timeout` |

Nothing in our transport closes a body that is still flowing. Only a context cancel (labelled) does. [A]

### P3 — concurrency, properly (2×2 over every stream call)

Parser `overlap.py` (scratchpad): a stream call = `Request URL (stream …)` open + the `ai_call` line
whose start (end − duration_ms) lands within 2.5 s of it; an executor call overlaps if
`[start,end]` intersects. Client.Timeout kills (600.0 s, pre-class-37) are excluded from "cut".

| 2026-09-01 (81 stream calls, 675 executor calls) | cut (transport) | survived |
|---|---|---|
| executor call overlapped the stream | **4** | 67 (+3 client_timeout kills excluded) |
| no overlap | **0** | 6 |

2026-08-31: 31 streams, 29 overlapped / 2 not, 0 transport cuts either way. 2026-09-02 (to 00:20):
1 stream, no overlap, survived. Streams last minutes and the executor fires every 2 min, so 92% of
streams overlap by construction; the 6 non-overlapped streams would be expected to show 0.34 cuts at
the overlapped rate. **No power, no effect shown — concurrency is not demonstrated as a factor; M4
(serialize) is NOT justified.** [A on the counts, B on the inference]

### P4 — the definitive test

`tcpdump`/`tshark`/`dumpcap` are **not installed** and `sudo` needs a password, so a pcap is not
available to this dispatch. Substitute armed (read-only, no root): a passive TCP-state watcher
(`~/nofx-backups/transport-capture/sockwatch.sh`, log `sockwatch.log`) polling `ss -tnopi` every
250 ms for every nofx-bin :443 socket, logging state transitions with `lastrcv/lastsnd/bytes_received`:

- `ESTAB → CLOSE-WAIT` = the **peer sent FIN first**; `ESTAB → FIN-WAIT-*` = **we closed first**;
- `ESTAB → GONE` with no intermediate state = RST (either side) — `lastrcv` right before tells the gap.

It cannot distinguish the DeepSeek edge from a middlebox that forges a FIN — only a pcap on the
Windows side (`pktmon`, admin) could. Armed since 00:14:54 CT; the 00:19:46 read (358 s, ASIA v8)
completed with **no cut**. Capture stays armed for LONDON 01:30 and ASIA 16:30. Owner action if a
pcap is wanted: `sudo apt install tcpdump` (one-time) — then the same watcher window applies.

### P5 — WSL2 / Windows (read-only)

- `.wslconfig`: `networkingMode=mirrored`; WSL eth0 10.0.0.141/24 + IPv6 global. No proxy env,
  WinHTTP "Direct access (no proxy server)". [A]
- Windows Firewall: Domain/Private/Public enabled, `DefaultOutboundAction NotConfigured` (= allow);
  Hyper-V firewall (mirrored mode) enabled, `DefaultOutboundAction Allow`, loopback enabled. [A]
- Defender Network Protection `EnableNetworkProtection=0` (off). TCP autotuning Normal, timestamps allowed. [A]
- TCP keepalive on the live sockets: 30 s (dialer), visible as `timer:(keepalive,Ns,0)`; kernel
  defaults 7200/75/9 are overridden per-socket. [A]
- `api.deepseek.com` → `d3bbv8sr76az5s.cloudfront.net` (3.173.21.63, 13.33.4.24, 143.204.130.x):
  the peer is a **CloudFront edge**, not DeepSeek's origin directly. [A]

### P6 — verdict

| Component | Verdict | Evidence |
|---|---|---|
| Idle watchdog (mcp) | **INNOCENT** | fires as `class=idle_deadline`, never "unexpected EOF" [A] |
| Total deadline / Client.Timeout | INNOCENT | distinct classes; lines say transport [A] |
| Go transport (SafeHTTPClient) | INNOCENT | no body-closing setting; only labelled cancels close [A] |
| Executor concurrency | NOT SHOWN | 4/71 vs 0/6, no power [A/B] |
| WSL2/Windows middlebox | NOT EXCLUDED | passive watcher armed; needs pcap to exclude [C] |
| **Peer (CloudFront edge or DeepSeek origin)** | **THEM — most likely** | FIN mid-body reproduces the exact string; http 200, no request id, no status incident; edge closes on origin failure look exactly like this [B] |

**Verdict: THEM** (peer/edge), our Go code excluded [A], middlebox unexcluded [C]. Phase 2 keeps its
shape (retry/backoff/resend), M2 is logging only, M3 no change (keepalive already 30 s), M4 not added.

## 2. Phase 2 — survive it (all pins written RED first)

| Item | File:lines (branch `fix/transport-resets`) | Pin (fails on dev, passes here) |
|---|---|---|
| **M0** identical-prompt resend on provider failure | `trader/auto_trader_planner.go:1405-1416` (resend branch) · `:1429-1441` (provider-failure branch: no reject block, no rejected-prompt row) · `mcp/client.go:253-268` (`ClassifyAIError`, `IsProviderFailure`) | `trader/class41_transport_retry_test.go` `TestClass41TransportFailureResendsIdenticalPrompt` — RED on dev: `attempt1 hash d324b51fcaa2 != attempt2 hash f0cd4b8e52e8`; GREEN here. `TestClass41ValidatorRejectStillAppendsBlock` pins the unchanged reject path |
| **M1** exponential backoff, tries count CALLS | `mcp/config.go:45-46` (fields) · `:192-238` (`StreamRetryTries`, `StreamRetryBackoffSchedule`, `streamBackoffFor`) · `mcp/client.go:1282-1311` (loop) | `mcp/class41_stream_retry_test.go` — default `2s→15s→45s`, tries 3; env override; loop test: 2 peer FINs then success = 3 calls, waits in schedule order |
| **M2** watchdog logs its fire | `mcp/client.go:1166-1178` `⏱ stream idle watchdog FIRED: Ns since last SSE line (idle=…, call age …)` | `TestClass41WatchdogFireIsLogged` |
| **M3** keepalive | no change — 30 s dialer keepalive confirmed in effect (P5) | — |
| **M4** serialize executor | **not added** (P3) | — |
| **M5** boot line + guide + checklist | `trader/auto_trader_planner.go:1328` `🔁 planner stream policy (class 41)` · `web/src/guide/content/settings.ts` knob card "Planner stream retry tries + backoff" (census 43→44) · `status.ts` boot ledger · `AUDIT-CHECKLIST.md` class 41 | `TestClass41PlannerStreamPolicyBootLine` · `GuidePage.test.tsx` 10/10 · `tsc` clean |

Knobs: `AI_PLAN_STREAM_TRIES` (default 3, counts CALLS, 1–6) · `AI_PLAN_STREAM_BACKOFF` (default
`2s,15s,45s`; last value repeats). `AI_MAX_RETRIES` is left in place and documented as counting CALLS
on the non-stream paths only. Worst case added wall per planner attempt: 17 s.

Bookkeeping decision: a provider failure writes **no** `planner_rejected_prompts` row (that table
samples validator rejects; the `🛰 planner call FAILED class=…` and `ai_call` lines already record
it). `lastRaw`/`lastErr` are left untouched, so a later validator reject still repairs against the
last real model output.

Tests: `go test ./...` green (worktree, full) · `go vet ./mcp ./trader` clean · vitest guide 10/10 ·
`tsc --noEmit` clean. Pre-existing stream tests needed one helper change (`streamClient` sets
`StreamTries`/fast `StreamBackoff`) — no assertion changed.

## 3. Cutover (pending owner GO)

Clean-clone build: `git clone --no-local` of the branch → `go build` → `vcs.revision=<REV>`
`vcs.modified=false` (quoted in §5 once staged). Rollback binary: `nofx-bin.prev.boot` (= the running
23f56f49 at swap time); rollback command: `cp nofx-bin.prev.boot nofx-bin && echo 23f56f49… >
deploy/RELEASE && kill -9 <MainPID>`. Cutover protocol: flat gate (DB OPEN=0 · API positions [] ·
NT8 count=0 · armed_orders non-terminal=0 — leg 4 is the ledger, `GetOpenOrders` is a stub) ·
in-flight (`replan_in_flight` + attempt state) · window (not 16:45–17:10, no live arms) · owner GO ·
RELEASE file → swap → kill -9 · boot line incl. 36/37/38/39/40/41 lines · marker commit AFTER the
passed boot (A19).

## 4. A15 — what the owner will still see wrong

- The four cuts remain unexplained at the edge level: no request id (DeepSeek returns none of
  X-Request-Id / Request-Id / X-Amzn-Requestid / Cf-Ray) means no provider ticket handle.
- Without tcpdump the FIN/RST direction is inferred from socket states, not packets.
- The watchdog still measures per SSE line, not per byte (documented on the boot line).
- `GetOpenOrders` on NT8 is a stub — flat-gate leg 4 is the ledger (class 33 candidate, unchanged).

## 5. Cutover record

_(appended after the owner GO)_

**Owner GO 06:1x CT 2026-09-02** ("ship Phase 2 M0+M1+M2 IMMEDIATELY as their own boot … live before NY 08:00; position 587 OPEN — A7, hold if live").

| Step | Quote |
|---|---|
| Flat gate 06:14–06:17 CT | position 587 CLOSED (exit 01:03 CT); DB open positions 0; `/api/positions?…&account=Sim101` → `[]`; armed_orders non-terminal 0 (cancelled 22, filled 9); NT8 count via the bound-account API = 0 |
| In-flight | no `Request URL (stream` since 04:00; no planner attempt lines; next read NY 08:00 |
| Window | 06:2x CT, outside 16:45–17:10, no live arms |
| Binary | clean clone `--no-local` of `fix/transport-resets` @ d5a6e138 → `vcs.revision=d5a6e138da851f2ee9ceba22424363bba0f219eb vcs.modified=false`; dev fast-forwarded to the same rev (PR #88) |
| RELEASE file | written to d5a6e138 BEFORE the swap (A19: file before swap, marker COMMIT after boot) |
| Swap | `cp nofx-bin nofx-bin.prev.boot` (= 23f56f49 rollback) · `mv nofx-bin nofx-bin.old.23f56f49` · `mv nofx-bin.next nofx-bin` |
| Restart | the auto-mode classifier denied `kill -9 2097561` and `systemctl kill --signal=SIGKILL nofx.service` three times; the OWNER ran the restart ("DO IT FOR ME ALL" → owner-side kill); old process exited 06:27:45 CT |
| Boot | 06:27:45 CT PID 1744258: `🔐 BOOT INTEGRITY OK — rev d5a6e138da85 · built 2026-09-02T11:15:37Z · expected d5a6e138da85 · goldens PASS` |
| Ledger lines | 🚀 speed wave · 🗓 session reads · 🪢 class 27 · 🧪 validator hints 15 sites (34+38) · 📜 contract 17 restrictions (38) · ⚖ arm normalizer (39) · 🗓 preflight (36) · 🧾 P&L surfaces 12/0 (40) · 🛰 planner client (37) · **🔁 planner stream policy (class 41): stream_tries=3 … backoff=2s→15s→45s … watchdog_log=on … keepalive=30s … serialize_executor=off resend_identical=on** |
| Errors since boot | 0 `[ERRO]`, 0 TradingRefused |
| Rollback | `cp nofx-bin.prev.boot nofx-bin && echo 23f56f49a53667174ca0d929d1e536f716bb1236 > deploy/RELEASE && <owner kill -9 MainPID>` |

**Proof owed (A20):** the next `class=transport|idle_deadline|total_deadline|client_timeout` or 5xx/429 planner failure must show `⏳ Waiting 2s` → `15s` between calls, then `📐 planner attempt N/3 failed on the provider (class=…) — attempt N+1 re-sends the IDENTICAL prompt` and `🧩 planner attempt N+1/3 resend-identical`; an idle kill must show `⏱ stream idle watchdog FIRED: Ns since last SSE line`. The NY 08:00 CT read is the first live candidate. Monitor `be53w8t7e` is armed on those lines.

**Live surface (A15) after this boot:** the four 09-01 cuts stay unattributed at the edge (no request id); FIN/RST direction is inferred from socket states, not packets (no tcpdump/sudo); the watchdog still measures per SSE line; NT8 `GetOpenOrders` is a stub (leg 4 = ledger). Phase 1 P4 (capture) continues read-only and reports separately.
