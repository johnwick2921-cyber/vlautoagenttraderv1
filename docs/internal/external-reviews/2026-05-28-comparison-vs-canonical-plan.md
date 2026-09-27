# Comparison — External Architecture Report vs Canonical Plan (2026-05-28)

> **Companion to:** [`2026-05-28-end-to-end-architecture.md`](2026-05-28-end-to-end-architecture.md)
> **Canonical plan:** [`docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md`](../../superpowers/plans/2026-05-22-nq-databento-ninjatrader.md)
> **Purpose:** Verify external claims against the actual codebase + plan, flag drift, capture useful new suggestions.

## Headline summary

| Verdict | Count |
|---|---|
| ✅ **Matches reality** | 18 claims |
| ⚠️ **Partially correct / off in detail** | 5 claims |
| ❌ **Wrong** | 3 claims |
| 🆕 **Useful new suggestions worth adopting** | 6 items |
| ❓ **Missing from the report** | 7 items |

The external report is **substantively accurate** on architecture, the NT8 wire protocol, ADR-007, the N11 root cause, and the EventSource/JWT constraint. Its biggest miss is the **branch confusion** at the top: it probed the public `dev` branch, which is the GitHub-declared default but is NOT where the operator's futures work lives. Once `main` is examined instead, almost everything the report flagged as "INFERRED" is in fact present and matches.

## Branch state — the headline reconciliation

The report's TL;DR opens with "the futures/NT8 adaptation described in the brief is not visible on the public `dev` branch." This is **factually correct but misleading**:

| Branch | Tip SHA | What's there |
|---|---|---|
| `origin/dev` (GitHub default) | `ab5873e2de261fe9327bb760686b5de0e2c4f3fb` | Upstream crypto NOFX — matches the report's observation |
| `origin/main` | `6c3333a6a698d64e8a4108fbb0f34ac3663592bd` | **All the futures work** — `ninjascript/`, `provider/ninjatrader/tcp_server.go`, `kernel/engine_prompt_futures.go`, ADR-007, 25 v1.0-* tags, the canonical plan with the 2026-05-28 NT8 pivot |

Default-branch settings on GitHub are persistent and have not been updated since the upstream fork. The operator's day-to-day trunk is `main`. **Anyone re-running the external probe should target `origin/main`, not the GitHub-declared default.** This single fact converts almost every "INFERRED" item in the report's §12 to "confirmed."

## ✅ Matches reality (18 claims)

These are correct as stated:

1. **Three-tier architecture** (NT8 ⇄ Go ⇄ React) — matches the canonical plan's architecture diagram.
2. **TCP wire format**: 4-byte big-endian length prefix + JSON envelope `{type, payload}`, 1 MB max frame — verified against `provider/ninjatrader/tcp_framing.go`.
3. **4 control-plane message types**: `signal`, `fill`, `heartbeat`, `ack` — matches Plan 1.5 spec lines 4378-4410.
4. **4 data-plane message types added by Plan 4.4**: `bars_subscribe`, `bars_historical`, `bar_update`, `bars_unsubscribe` — matches Plan 4.4 Deep Spec lines 6960+.
5. **Single-client TCP gate**, refuse-or-supersede duplicates — matches `tcp_server.go:207-213`.
6. **Auto-resubscribe on reconnect** — matches the Plan 4.4 design.
7. **v1.5.6 fix**: per-connection write mutex + per-write `SetWriteDeadline` — matches the actual fix exactly.
8. **ADR-007 byte-identical contract** binds `tcp_server.go`, `tcp_framing.go`, `VLTraderTCPClient.cs` — matches `docs/adr/ADR-007-plan1-critical-file-integrity.md`.
9. **"Warn-and-continue on unknown frame type"** safety valve — matches `tcp_server.go` readLoop default branch.
10. **N11 root cause**: Balanced Strategy using dead `ai500` coin source returns HTTP 402 (x402 paywall) and starves the trader. Fix: `coin_source=static` + `["NQ.c.0"]`. Matches the 2026-05-28 plan-doc post-mortem.
11. **Bar cache keyed by `symbol|timeframe`** — matches `market/data.go` structure.
12. **Indicator set** (EMA/MACD/RSI/ATR/Bollinger) — matches `market/data_indicators.go` exports.
13. **GORM models** (`exchange.go`, `strategy.go`, `position.go`, `decision.go`) — matches `store/`.
14. **Lightweight Charts v5 `addSeries(CandlestickSeries)` migration** — matches TradingView's official v4→v5 guide.
15. **JWT-in-query for SSE** because EventSource has no header API (WHATWG html#2177) — matches the constraint cited in Plan 4.4 Deep Spec frontend gotchas #1.
16. **Databento Historical vs Live tier split** as the rejection reason — matches the canonical plan's 2026-05-28 Architecture Pivot block.
17. **`ninjascript/VLBarsSubscriptionManager.cs` is a separate file** — verified: `ls ninjascript/` shows it alongside `VLTraderTCPClient.cs`.
18. **NT_TRANSPORT env var router** between CSV trader and TCP trader — matches `trader/ninjatrader/transport.go`.

## ⚠️ Partially correct / off in detail (5 claims)

### W1 — v1.5.7 fix uses `context.WithCancel` + watcher goroutine, NOT `context.AfterFunc`

The report (§4 "v1.5.7 read-deadline-desync fix") states the fix uses:
> `context.AfterFunc(ctx, func(){ conn.SetReadDeadline(time.Now()) })`

The actual fix (`provider/ninjatrader/tcp_server.go`, commit `803a8727`) uses:
> `connCtx, connCancel := context.WithCancel(ctx); defer connCancel(); go func() { <-connCtx.Done(); c.Close() }()`

Same architectural pattern (cancellable blocking read driven by ctx), different API. `context.AfterFunc` was added in Go 1.21 and would also work; the operator chose the more traditional `WithCancel`+goroutine. The report's substantive claim (read deadline removed, ctx-cancel watcher unblocks the read) is correct; the specific API surface is wrong.

### W2 — v1.5.6 write fix root cause subtly mischaracterized

The report (§4) states the v1.5.6 fix is required because:
> "concurrent goroutines (heartbeat, signal, bars_subscribe) were corrupting frames on the wire under load"

That's the SECONDARY motivation. The PRIMARY root cause (per the 2026-05-28 post-mortem in the plan doc) was **the stale persistent write deadline** on the heartbeat ack path: Go's `net.Conn` deadlines persist until reset, the ack write inherited the 5s deadline from the most recent `heartbeatLoop` send, and once that expired every write failed with `i/o timeout` — producing a deterministic 60s reconnect cycle. The mutex was added as defense-in-depth, not because frame corruption was observed. Also: `bars_subscribe` cannot have contributed because Plan 4.4 Stage 2 (which adds the bars frame handling) is not yet built.

### W3 — Decision cycle action set: 6 valid actions, not 9

The report (§9) cites "upstream issue #982" as enumerating 9 valid actions:
> `open_long, open_short, close_long, close_short, update_stop_loss, update_take_profit, partial_close, hold, wait`

The actual `kernel/engine_position.go` validActions map contains **6** entries: `open_long`, `open_short`, `close_long`, `close_short`, `hold`, `wait`. There is no `update_stop_loss` / `update_take_profit` / `partial_close` action key in the engine_position.go validation path. Those may be runtime-enforced elsewhere (e.g. as decision-parsing fields rather than discrete actions), but the report's "9 validated actions" framing is inaccurate against the current code.

### W4 — CHANGELOG.md observation is correct in fact, wrong in implication

The report (§1) correctly notes:
> "CHANGELOG.md last-updated 2025-11-01… none of which mention NT8, Databento, v1.5.x TCP fixes, Plan 4.4, ADR-007, or N11"

True. **But** this is because the operator's change log is the **canonical plan doc + git tags**, not `CHANGELOG.md`. The latter is a vestige from the upstream NOFX repo. The 25 `v1.0-plan*` and `v1.0-task*` tags on origin (visible via `git tag -l`) ARE the version log for the futures work; the plan doc's Ship Log section reproduces them. CHANGELOG.md being stale is a known cosmetic issue (it should either be updated or marked superseded by the plan doc), not evidence the futures work is absent.

### W5 — FuturesChart.tsx and `/api/klines/stream` SSE endpoint described as if shipped

The report (§5 Stage 4, §11) describes the chart relay + `FuturesChart.tsx` in present tense. **Neither is built yet.** `find web/src -name "FuturesChart*"` returns empty. Plan 4.x Sequencing REVISED 2026-05-28 explicitly lists Stage 4 (chart display + SSE relay + FuturesChart) as the LAST stage of the in-flight Plan 4.4 build, after Stage 1 (C# AddOn, PR #29 awaiting merge), Stage 2 (Go bar handling), and Stage 3 (kernel decisions). The Stage 4 design in the report is sound; it just hasn't been implemented.

## ❌ Wrong (3 claims)

### X1 — "The futures/NT8 adaptation is not visible on the public `dev` branch"

Technically true (verified) but misleading framing. The work lives on `origin/main`. Targeting `dev` is the same as reading an empty box because the label says "default."

### X2 — "No `CLAUDE.md` at root"

Wrong. The repo has `CLAUDE.md` files at multiple levels (root + `provider/CLAUDE.md`, `provider/ninjatrader/CLAUDE.md`, `kernel/CLAUDE.md`, `market/CLAUDE.md`, `web/CLAUDE.md`, `trader/CLAUDE.md`). They're in `.gitignore` on `dev` but exist on `main`. This is part of the same branch-confusion failure mode as X1.

### X3 — "Zero C# in the language breakdown"

Wrong on `main`. The `ninjascript/VLTraderTCPClient.cs` and `VLBarsSubscriptionManager.cs` files are tracked. GitHub's language breakdown is computed against the default branch (`dev`), where they don't exist — so GitHub reports 0% C#. On `main` the C# is real, just not reflected in the GitHub UI.

## 🆕 Useful new suggestions worth adopting (6 items)

These are net-new vs the canonical plan and worth capturing:

### N1 — `protocol_version` field on `heartbeat` (or envelope)

Recommendation: add a version field so the older side logs a warning instead of falling through to the warn-and-continue path for unknown frames. **Adoption value: high.** Would have prevented at least one of the Plan 1.5.x hotfix cycles. Should land alongside the next breaking-protocol change. Not in the canonical plan yet.

### N2 — CI hash-check enforcing ADR-007 byte-identical contract

Recommendation: hash `tcp_server.go`, `tcp_framing.go`, and `VLTraderTCPClient.cs`; fail any PR that touches one without matching changes in the other two. **Adoption value: medium-high.** Currently ADR-007 is enforced by review discipline (`docs/adr/ADR-007-plan1-critical-file-integrity.md`). A CI check would catch drift mechanically. Not in the canonical plan.

### N3 — `bars_resync` frame type

Recommendation: on Go reconnect, ask NT8 for the last N bars per subscription rather than relying on Go-side memory. **Adoption value: medium-high.** Currently Plan 4.4 Stage 2 design talks about "cache last bars_historical for late joiners" but only on the Go side; a wire-level resync handshake would protect against the Go-process-restart-while-NT-keeps-streaming case. Should be added to the Plan 4.4 Stage 2 wire-protocol design before Stage 2 lands.

### N4 — Cap `bars_historical` batch at ~5,000 bars + paginate

Recommendation: even with the 1 MB frame ceiling, a single backfill batch exhausting the read pipeline holds up signal frames. Pagination restores fairness. **Adoption value: medium.** Not in the canonical plan. Worth adding to Plan 4.4 Stage 2 design.

### N5 — Prometheus counter for dropped coalescing-channel ticks

Recommendation: if the in-progress-tick drop rate goes above ~1% of stream rate, the indicator path is too slow. **Adoption value: medium.** Plan 4.x backend visibility (Plan 4.14) is the natural home; right now Plan 4.14 is "design pending" and has no UI surface. This concrete metric is a good starting point for that scope.

### N6 — `fetch`-based SSE polyfill to put JWT in `Authorization` header

Recommendation: replace native `EventSource` with `fetch-event-source` polyfill to eliminate URL-token leak. **Adoption value: medium.** The canonical plan's Plan 4.4 frontend gotchas #1 documents the EventSource constraint but accepts JWT-in-query as the mitigation. The polyfill is a cleaner alternative worth evaluating against the maintenance cost.

## ❓ Missing from the report (7 items)

Items in the canonical plan that the external report does not address:

### M1 — The 8 open Plan 4.4 questions

The plan doc tracks 8 open questions gating Stage 2 + 4 detailed design (trading hours ETH/RTH, MergePolicy, bars_back, multi-symbol overlay, JWT TTL, backpressure, per-tenant dedup, no Go-side persistence). The report describes the Plan 4.4 staging but doesn't surface these decision-required items.

### M2 — Native multi-TF vs Go-side aggregation **is decided** (Option 1)

The report (§5 Stage 3) presents both strategies as "supported." The plan doc locks the decision: **Option 1 — native, one BarsRequest per timeframe**. Reason: HTF bias must match NT8's own bars (DST/session-boundary edges differ from pure 1m roll-up). Worth correcting in any forward use of the external report.

### M3 — Plan 4.4 Stage 1 PR #29 status (in-flight, C# AddOn compiled, awaiting Plan 1.5.7 merge gate)

The report describes Stage 1 in present tense without acknowledging it's not yet merged. Current state: PR #29 exists, compiled clean in operator's Windows VS, gated on Plan 1.5.7 verification (now merged at `6defdc84`; tag `v1.0-plan1-5-7` not yet applied).

### M4 — Trader-mode SelectedTimeframes vocabulary

Plan doc: 14 TF values normalized by `store/strategy.go::normalizeTimeframe` (1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w). Balanced Strategy default: `["5m", "15m", "1h"]`, primary `5m`, longer `4h`. The report describes multi-TF support generically without naming the vocabulary or defaults.

### M5 — Cosmetic / hygiene backlog (Plan 4.9.x)

Favicon `/favicon.ico` 404, footer GitHub/Twitter/Telegram hrefs all `#` placeholders, Chinese label rendering in EN locale on some stat-cards, PageNotFound component orphan, grainy-gradient 404 image. Tracked in the plan doc's Deferred Items section. Worth listing in any audit-quality report.

### M6 — N8 password-change security item

No "old password" verification before allowing change → trivial bypass if session token leaks. ~10 LOC + backend endpoint update. Tracked in plan doc.

### M7 — Vite HMR stale-module cache gotcha (web/CLAUDE.md PR #24)

A documented Vite issue where the dev server caches old module state; F5 hard-reload required after Plan 4.x cosmetic changes to see them. The plan doc's Ship Log includes this; the external report doesn't address frontend dev-environment behaviors at all.

## What this means in practice

1. **The external report is a useful sanity check.** ~80% of its substantive claims match the canonical plan, which corroborates the design from an outside angle (NinjaTrader docs, TradingView v5 guide, WHATWG html#2177, Databento product pages, Go context docs, SlowMist disclosure).
2. **The branch confusion is the single load-bearing error.** Fix it (probe `main` instead of `dev`) and the "INFERRED" section §12 collapses to ~3 items (FuturesChart not built, the action set count, the v1.5.7 API specific).
3. **Six new suggestions are worth lifting into the canonical plan.** Especially N1 (`protocol_version`), N2 (CI hash-check), N3 (`bars_resync`), and N4 (paginate `bars_historical`) — all four are wire-protocol design points that should land before Plan 4.4 Stage 2 to avoid another hotfix cycle.
4. **Don't action W1/X1/X2/X3 in the report's text** — they're branch-confusion artifacts. The factual content beneath them is correct.

## Adoption recommendation

For the next plan-doc update cycle:
- Pull in N1, N2, N3, N4 as additions to the Plan 4.4 Stage 2 wire-protocol design + Plan 5 testing matrix.
- Update `CHANGELOG.md` to either reflect the current ship state or replace it with a one-line pointer to the canonical plan's Ship Log (closes W4).
- Consider setting GitHub's default branch to `main` (closes X1/X2/X3/W4 in a single repo-config change, but be aware some CI / fork-tracking infrastructure may key off the default).

No code changes recommended on the basis of this report alone — the architectural fixes it suggests (N1–N6) are net-new and worth a separate dispatch.
