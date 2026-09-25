# Ledger-Close: Calendar Risk + Alerts + Debt — Dispatch Report (2026-08-19)

Branch `fix/ledger-close-sep-risk` · 12 commits · 43 files · +2,540/−49
Rebased twice per the sequencing rule: cut from `b5aa0f48` (in-position
silence fix, PR #49), preempted mid-run by the phantom-position P0
(PR #50, `d154bb44`, deployed), resumed rebased on that hotfix. No
interleaved commits at any point. Deadlines served: Labor Day Sep 7 ·
roll week ~Sep 10 · SEP26 expiry Sep 18.

---

## 1. Per-phase: findings → what shipped → commits

### P1 — Clock persistence guard (`51c2a9bf`, `d9978599`)

**Report-first [A]:** drift at start of work: +266ms WSL-ahead raw
(sub-200ms after PowerShell launch overhead); rtc-vs-WSL delta −1s;
NTP offset +644ms, systemd-timesyncd active and slewing. The −116s skew
class was NOT active. **This WSL2 has NO root-free resync**: `hwclock`
does not exist (rc=127, not in /usr/sbin), passwordless sudo is absent
(`sudo -n true` → password required), `timedatectl set-ntp` is
polkit-gated. Best root-free MEASUREMENT: `/sys/class/rtc/rtc0/
since_epoch` (world-readable; rtc0 in WSL2 is the Windows host clock)
vs `date +%s`.

**Shipped:** `deploy/nofx-clock-guard.sh` + `systemd-user/
nofx-clock-guard.{service,timer}` + `install-clock-guard.sh` — a
15-minute measure+alert detector (rtc + NTP offset + optional
powershell interop cross-check), logging to the user journal and an
atomic state JSON (`NOFX_CLOCK_STATE`, default
`data/clock-guard-state.json`). The script SAYS resync is unavailable
per dispatch 1.2 ("use the best available resync and SAY SO"); the
owner-side root unit remains the escalation path. **Installed live
during the phase** (sanctioned exception): visible in
`systemctl --user list-timers`, first observed cycle logged
`clock-guard status=OK rtc_vs_wsl_s=-1 win_vs_wsl_ms=n/a
ntp_offset=+644.145ms resync=unavailable-no-root timesyncd=active
warn_s=30`. Go side: `CLOCK_WARN_MS` (default 30000 = 50% of C2's 60s)
adds a "🚨 CLOCK EARLY-WARNING" tier to the clock-health line via a
pure classifier (the injected-fake-drift test hook); `LogClockGuardBoot`
prints live RTC drift + timer freshness + last check in the boot block
(P1.4).

### P2 — stopUntil producer (`71ee0444`, `d4ad9875`)

**Report-first [A] (2.1):** consumer at `trader/auto_trader_loop.go:248`
blocks the WHOLE cycle (not entries-only — contradicting the owner
contract); field `stopUntil time.Time` (auto_trader.go:340) has no
mutex and NO producer anywhere — grep finds zero assignments; the gate
was dead code. `GetStatus` already surfaced it as `stop_until` and the
FE already typed it.

**Shipped (additive; legacy field/gate left dormant):** the REAL pause
is `pauseUntilMs` (atomic) in `trader/auto_trader_pause.go`:
`PauseEntriesUntil`/`ResumeEntries`/`entryPaused` with persistence in
`system_config` key `trader_pause_until:<id>` (restored in `Run()` →
survives restart), auto-resume on first read past deadline (CAS-raced),
exact refusal `"stop_until: paused until HH:MM CT (owner)"`. Gate
inserted in `executeDecisionWithRecord` as the FIRST owner/policy gate
(after the four system-integrity gates, before consecutive-loss) —
opens only; closes/EOD-flat/brackets/60s monitor untouched.
API: `POST /api/traders/:id/pause` (`{minutes}` | `{until_ct:"HH:MM"}`
wrap-safe | `{until:"session_end"}` via the session registry) +
`/resume`, owner-auth identical to start/stop siblings. FE: PauseButton
(30m/1h/session-end/custom + Resume, en/zh/id inline dict) next to
EmergencyFlat, and the live banner "⏸ PAUSED until HH:MM CT (owner)"
on the trading view; `status.stop_until` now reports the real pause.
Master-INDEPENDENT.

### P3 — Contract-roll block for continuous "MNQ" (`ef66d44e`)

**Report-first [A] (3.1):** the existing T19 gate
(`kernel.ShouldBlockEntryForExpiry` → `databento.DaysUntilExpiry`,
kernel/engine.go:1012) parses ONLY the dated code form ("MNQU6", last
two chars). The live symbol is the bare root "MNQ" → parse error → 999
days → the gate is **structurally dead on the live path**. The
resolved front contract already arrives Go-side in the AddOn's
subscription ACK (`SymbolSubState.Contract`, e.g. "MNQ 09-26" —
produced by `VLContractResolver` C#-side, which itself rolls ~8 days
pre-expiry, making the ACK ground truth for what NT8 trades).

**Shipped:** `trader/contract_roll.go` — parse "ROOT MM-YY", derive
third-Friday expiry in CT (**verified against CME's own ProductCalendar
API for product 146: NQU26 lastTrade = settlement = 2026-09-18**,
archived original cited in-code; termination 8:30 CT per Rulebook
35902.G), gate NEW entries within `ROLL_BLOCK_DAYS_BEFORE_EXPIRY`
calendar days (default 3 → Sep 15–18 blocked, covering the Sep 16–17
dying-liquidity target; boundary note: Sep 14 = daysLeft 4 passes at
the default and blocks at env 4+). Refusal:
`"contract_roll: MNQ SEP26 expires 2026-09-18, entries blocked (3d
window)"`. Resolution unavailable → WARN once per contract-string +
PASS (fail-open). `GetStatus` gains the `roll` block (resolved
contract, expiry, window start, days left) for the dashboard (3.6).
Resolver reuse: the established `at.trader.(*ntTrader.TCPTrader)`
assertion → `BarsSubscriptionStates()` — no second resolver built.

### P4 — HalfDays producer (`a37aff45`)

**Report-first [A] (4.1):** `SessionRegistry.HalfDays` lives in the DB
(`system_config` key `session_registry`), empty since birth. The ONLY
live consumer was the EOD-flat pull-in (`effectiveEODFlatCT`,
min-semantics); **last-entry was half-day-blind**, and the kernel twin
`EffectiveFlatCT` is production-dead with weaker semantics.
**Collision found:** `isCMEHoliday` hardcodes Labor Day, Thanksgiving,
day-after-Thanksgiving and Dec 24/31 as FULL closures — the decision
cycle idles on those calendar dates entirely (see §8).

**Shipped:** owner-editable `half_days.json` (env `NOFX_HALF_DAYS`)
seeded with the official table (§3); producer
`trader/auto_trader_halfdays.go` merges it into the stored registry
once per session-day (idempotent, file-wins per key, DB-only keys
survive, validated pre-write), called ABOVE the session gate (weekend
boots seed Monday — the F0 precedent). Malformed file/entry → CRITICAL
+ normal trading (4.5). **Both cutoffs now resolve against
`early_close_CT − offset`** via the new `halfDayCutoffMin` (last-entry
AND EOD-flat; the old zero-offset `effectiveEODFlatCT` kept for
tests/API). `ValidateSessionRegistry` now validates HalfDays keys and
values (the API door previously persisted garbage silently). Boot
prints count + "next half-day: 2026-09-07 12:00 CT (Labor Day…)".

### P5 — 402 / balance alert (`d9a23407`)

**Report-first [A] (5.1):** 402 was ALREADY non-retryable per call at
the mcp layer (absent from the retryable-substring list → one attempt)
and the kernel returns transport errors instantly — "no retry" needed
an OUTAGE latch, not a retry-loop change. The status-code int dies
inside mcp (flattened to text at 4 sites); the existing trader
classifier string-matches "402"/"Insufficient Balance"; decision
records carried only free-text errors. No DeepSeek balance endpoint
existed in code.

**Shipped:** additive `decision_records.error_class` column
("ai_payment_402" | "ai_call_failed") — one-query forensics; outage
latch `on402Failure` → ONE P0 banner per outage ("AI CREDIT EXHAUSTED —
no decisions until topped up", event-id = outage start, alert-bus
dedup makes cycles 2..139 silent no-ops); `onAISuccess` auto-ACKs the
banner via new `AlertStore.AckByEvent` (clears without human ack; a
NEW outage re-alerts). 5.4: DeepSeek DOES expose
`GET /user/balance` (Bearer) — `mcp.DeepSeekBalance` wraps it
defensively; the daily poll runs only when `AI_BALANCE_WARN` is set
(default OFF), WARN + P1 below threshold. Caveat carried from the
alert bus: `emitAlert` is day-plan-gated (the owner's trader qualifies).

### P6 — WARN+ERROR → DB (`85ec608f`)

**Report-first [A] (6.0):** logger is logrus with a single global and
ZERO hooks — `AddHook` is a clean additive seam; trader `logWarnf/
logErrorf` delegate through it (with a parseable `[trader_id=…]`
prefix). "CRITICAL" is Errorf text (no such logrus level). Journald's
2G cap + the per-frame INFO flood erased Aug 13–17; live measurement
during this dispatch: **journald suppressed >114,000 messages per 30s**
— the DB tee is not optional.

**Shipped:** `logger/db_sink.go` (WARN/ERROR/FATAL/PANIC hook; sink
injected post-store-init from main.go; extracts component file:line +
trader_id) → `store/log_event.go` (`log_events` table; Enqueue is ONE
select-default — full buffer drops + counts, never blocks; single
writer goroutine per the plan-store pattern; once-per-day hard prune at
`LOG_DB_RETENTION_DAYS`, default 30). Recursion-proof: the shipper
never logs through logrus. Guarantee mechanism stated per 6.2.
Rollback note (documented, NOT executed): `DROP TABLE log_events;`
Known coverage gaps (by design, reported): INFO-level lines
(routine clock-health, ai_call successes, gate summary) and the
slog-based tcp_server/agent lines are not shipped.

### P7 — Single snapshot instant (`ce1a3988`)

**Report-first [A] (7.1):** one decision cycle carried FOUR clocks —
T0 loop stamp (`SnapshotMs` + `CurrentTime`, before any market data
exists), T1 market-block fetch (`fetchMarketDataWithStrategy`), T2
level/SVP/plan snapshot (`snapshotNow`/`snapshotBars` at
engine_analysis.go:308), T3 post-AI gate clocks. Level-distance math
read the cache seconds-to-minutes after the market block in the SAME
prompt (U4). The engine.go SnapshotMs comment ("when this context's
market data was assembled") was aspirational.

**Shipped:** `snapshotNow` hoisted above the fetch; market block and
the 1m snapshot window read back-to-back; `ctx.SnapshotMs` re-stamped
at assembly (B4 now evaluates at the true data instant); the prompt's
Clock line gains **"· Snapshot: HH:MM:SS CT"** via the new
`kernel.ClockCTSeconds` (layouts live only in tz.go — TZ-guard
enforced). Zero golden churn (the Clock line is per-cycle engine
state). Source-order contract test pins capture < stamp < fetch <
bars-read < clock-line and exactly ONE `time.Now()` capture.

### P8 — B4 hidden-clock removal (`1021c65b`)

**Report-first [A] (8.1):** the SnapshotMs-absent fallback evaluated
B4 on the caller's post-AI `time.Now()`. Production never hits it
(buildTradingContext stamps unconditionally; P7 re-stamps at assembly;
the only SnapshotMs-less Context is the api/strategy.go prompt-preview
which never reaches the gate) — the fallback only ever fired in tests.

**Shipped:** absence is now WARN + fail-open pass — never a silent
wall-clock evaluation; `nowMs` remains diagnostics-only (post-verdict).
The two tests pinning the old fallback gained SnapshotMs (becoming
snapshot-clock tests); a new test pins absence→pass with an injected
fake clock.

Plus: E5 interaction-edge ordering guard (`59e98d3c`, `c0e5ce43`).

### P10 — Scan interval = real cadence (OWNER RULING; addendum)

**Sequencing note:** the addendum says "after Phase 9", but **no Phase 9
dispatch was ever received** (the phantom-position dispatch referenced a
"post-exit rescan addendum" as future work). Phase 10 lands now; the
number 9 stays reserved. Phase 9, when dispatched, must consume the
reconciled position source (PR #50's guarded path), which is this
branch's parent — the ordering is satisfied by construction.

**Ruling implemented:** the Studio scan interval is the ACTUAL decision
cadence. `traders.cadence_mode` (Studio/DB, per-trader like
scan_interval): **"interval" (new default — every scheduler tick runs a
full cycle on the LATEST bar state, forming bar included)** |
"bar_close" (the legacy day-plan P2 gate, byte-identical, selectable).
Garbage/empty resolve to interval — junk never silently invents the
stricter gate. The only interval-mode skip besides existing gates is
the 10.4 dedup: identical newest-primary-bar signature
(open/high/low/close/volume) AND flat → `cycle_skip=no_new_data`
logged, no paid call burned; any bar mutation or an open position runs
the cycle (the #49 in-position heartbeat is never deduped).

**Prompt honesty (10.2):** the market block now labels the newest bar
per timeframe — `current 5m bar: FORMING (closes 10:35 CT) — prior
bars closed` / `current 5m bar: CLOSED at 10:35 CT (next close 10:40
CT)` — rendered against the P7 snapshot instant
(`SetPromptSnapshotMs`), futures-only, absent when no snapshot (all
prompt goldens byte-identical — verified). "Wait for the close" is now
the AI's documented judgment, not a code gate.

**Boot line (10.3):** `📊 Loading trader hoang: ScanIntervalMinutes=2
(source=Studio/DB), cadence=interval 2m0s, mode=interval (DB empty →
default)` — source + resolved behavior named.

**Studio UI (10.5):** the config modal's interval field gains an
Interval/Bar-close toggle; help text states plainly (en/zh/id): "AI
evaluates every N minutes (mode=interval), including the forming bar."
vs "AI evaluates once per closed primary bar; the interval only sets
check frequency (mode=bar_close)." Saving remove+reloads the trader —
**E16: mode switch applies WITHOUT a bot restart.**

**In-flight (10.6):** unchanged by design — the scheduler is
sequential; ticks during a 60–110s AI call drop (one may buffer),
so back-to-back cycles after long calls are expected and logged
honestly by the cadence data itself.

**COST NOTE (owner accepted):** measured BEFORE (bar_close, this
morning 08:22–09:22): 10 cycles/hour ≈ 10 AI calls/h. Expected AFTER
(interval 2m, 60–110s calls, dedup active): ~18–25 calls/h during
sessions (~2–2.5×). Measured-after lands with the E6 soak. E17
(supersession-discard rate under interval mode) is reported from the
same soak window and feeds the parked discard-burn dispatch.

## 2. Canonical gate evaluation order (2.4) — standing documentation

Five stages; a decision survives ALL to reach the wire. Full [A]
inventory (every line read; telemetry counter names in brackets):

**STAGE 0 (cadence):** day-plan traders run `runCycle` only on a NEW
closed primary-TF bar (`barCloseGate`) — silent idle otherwise.

**STAGE A (whole-cycle, auto_trader_loop.go):**
1. trader stopped (:101) → 2. CME session gate + 3-min backoff (:120;
calendar + half-days producers deliberately ABOVE it) → 3. NT8
account-selected (:134) → [dead-man observe; housekeeping] → 4.
EOD-flat flatten-then-skip (half-day aware) → 5. legacy stopUntil
(:248, dormant — superseded by the P2 pause, left per additive rule) →
6. buildTradingContext error → 7. zero-equity/no-balance-frame →
[saveEquitySnapshot always] → 8. skip-while-open (AI-call-only skip,
now with the PR #50 position_state_desync cross-check) → 9. no
candidates.

**STAGE B (kernel pre-prompt holds):** 10. task18 cme_closed → 11.
task19 contract_roll (dated-code form; dead for bare "MNQ" — see P3) →
12. concurrent cap [task21_concurrent_cap] → 13. daily guardrails
(soft when master OFF) [strategy_studio_daily] → 14. blackout window →
15. consistency rule → 16. token-estimate hard error → 17. A5 prompt
ownership → 18. schema-strict ×3 → schema_parse_failed.

**STAGE C (validateDecision, in the parse loop):** action enum →
leverage>0 (cap auto-adjusts) → size>0 → min size → futures notional
cap (always on) → SL/TP>0 → side ordering → F1 real R:R [rr_gate] →
min_confidence (=60).

**STAGE D (post-parse armors):** B2b price-sanity [price_sanity] → B4
stale-data at SnapshotMs [stale_data] → C2 clock-drift (log-only)
[clock_skew_observed] → B7 re-entry cooldown [reentry_cooldown]; back
in the loop: guardrail_skip recorder → P0-latency supersession discard
[stale_bar_discarded] → safe-mode filter.

**STAGE E (execute gates, auto_trader_orders.go — opens only unless
noted):** 1. feed-gate (blocks closes too) [feed_down] → 2. dead-man
[dead_man] → 3. A4 freeze [frozen] → 4. boot integrity
[boot_integrity] → **5. stop_until owner pause [stop_until] (NEW —
first owner gate, E5-pinned) → 6. contract-roll resolved-front gate
[contract_roll_resolved] (NEW)** → 7. consecutive-loss
[consecutive_loss] → 8. last-entry cutoff (half-day aware, NEW)
[last_entry] → 9. session gate [session_gate] → 10. plan-mode
[plan_mode] → 11. approval [approval_required] → [advisory plan
citation] → open executor (reconcile-before-open, max-positions, dupe,
size caps, contract clamp) → broker chokepoint (unbound refusal,
SIM-only allow-list, B3 dedup/rate, SL/TP-preset assertion) → C# AddOn
re-checks (account tradeability, 60s signal age).

## 3. Seeded half-day table + official sources

| Date | Early close (CT) | Label | Source class |
|---|---|---|---|
| 2026-09-07 | 12:00 | Labor Day — equity halt 12:00, reopen 17:00 (trade date Tue 09-08) | CME-official (2026 settlement PDF + archived 2026 trading-hours page) |
| 2026-11-26 | 12:00 | Thanksgiving Day — halt 12:00 | CME-official (2026 settlement PDF) |
| 2026-11-27 | 12:15 | Day after Thanksgiving — final close 12:15, settlement 12:00 | CME-official (settlement) + convention (close) |
| 2026-12-24 | 12:15 | Christmas Eve — final close 12:15, settlement 12:00 | CME-official (settlement) + convention (close) |
| 2026-12-31 | — | **Deliberately ABSENT**: NORMAL equity session per CME (only rates settle early) | CME-official |

URLs (fetched as archived ORIGINAL bytes via web.archive.org —
cmegroup.com IP-blocks this host; also cited in
trader/auto_trader_halfdays.go):
- Expiry: `https://www.cmegroup.com/CmeWS/mvc/ProductCalendar/Future/146` (NQU26 lastTrade = 18 Sep 2026; archived 2026-07-10)
- `…/holiday-calendar/files/2026/labor-day-holiday-settlement-times-2026.pdf`
- `…/thanksgiving-holiday-settlement-times-2026.pdf`
- `…/christmas-holiday-settlement-times-2026.pdf`
- `…/new-years-eve-holiday-settlement-times-2027.pdf`
- CME Rulebook ch. 359 (35902.G termination 8:30 CT SOQ).

## 4. Test outputs P1–P8

Every phase closed green before the next began (one mid-run
regression: the TZ-guard rejected P2's bare "15:04" layout at the P4
gate — fixed forward `d4ad9875`; and the E5 anchor test was corrected
in `c0e5ce43` after its first anchors matched helper comments). Final
state, rebased branch:

- `go build ./...` clean · `go vet` clean on touched packages
- **FULL suite `go test ./...`: exit 0, zero failures** (includes the
  #45 timegate T1–T7, #48 stale-guard, boot-integrity, [MNQ] goldens,
  TZ-guard, and the two P0 hotfix suites)
- `cd web && npx tsc --noEmit && npm run build`: clean (5.1s)
- New tests: 13 files — clock classifier/env/boot-state, pause
  block/persist/expiry/resume, third-Friday table
  (SEP26/DEC26/MAR27/JUN27), roll window boundaries + env + exact
  message, half-days loader/merge/cutoff-math/validation, 402
  classify/burst-dedup/auto-clear/env, log-sink levels + flood
  (50k enqueues < 2s, drops counted) + retention, snapshot-instant
  source contract + ClockCTSeconds, B4 absence fail-open, E5 ordering.

## 5. E-steps

- **E3 (calendar sims, sim clock):** roll — Sep 15/16/17/18 blocked,
  Sep 14 open at default 3 (blocked at env 5), Sep 21 with DEC26
  passes daysLeft≈88, message core pinned `"MNQ SEP26 expires
  2026-09-18"` (TestRollVerdictWindow/TestRollGateMessage). Half-day —
  `2026-09-07`+offset 30 → last-entry 11:30; offset 0 → flat 12:00;
  normal day unchanged; ASIA unaffected via min-semantics; garbage
  value fail-safe (TestHalfDayPullsNYCutoffsForward).
- **E4 (control sims):** pause set→exact refusal shape→resume→clears;
  restart-persistence; expiry auto-resume (pause_test). 402: 139-burst
  → exactly ONE unacked P0 → first success auto-clears → new outage
  re-alerts (ai402_test). #48 feed-gap alert still fires — feedwatch
  suite green post-rebase (regression held).
- **E5 (interaction edges):** stop_until < contract-roll <
  consecutive-loss pinned on the emitted refusal strings
  (gate_order_test) — a paused refusal names the pause. Half-day vs
  session cutoff: min-semantics proven in halfdays_test (earlier wins,
  log names the resolved hhmm).
- **E1 (boot proof), E2 (live cycle trace), E6 (soak ≥60 min):**
  pending the atomic cutover — appended below when run.
- **E7 (adversarial self-check):** multi-agent defect-class sweep
  (literal-on-path, day-scope, computed-then-discarded, never-wired,
  twin-path, correctness/races) over the full branch diff — verdicts
  appended below.

## 6. Touched-file budget audit (addendum)

Planned-but-untouched: `trader/auto_trader_registry.go` (P4 — the
existing once-per-session-day registry cache needed no change),
`kernel/cme_calendar.go` (P4 — full-closure refinement deliberately
out of additive scope, §8), `web/src/types/trading.ts` (stop_until
already typed), `web/src/i18n/translations.ts` (inline en/zh/id dict
in PauseButton — matches the EmergencyFlatButton sibling and keeps the
3k-line hot file untouched), `mcp/client.go` (typed error unnecessary
— the text shape is load-bearing and untouched).
Touched-beyond-budget: `trader/auto_trader_decision.go` (GetStatus:
truthful stop_until + the P3 roll block — the status surface the
dispatch's 2.2/3.6 dashboards read), `mcp/balance.go` +
`store/alert.go` (P5 additions named in-phase), `deploy/*` (P1 new
units, named in-phase), `trader/gate_order_test.go` (E5 artifact).
Full `git diff --stat` (43 files, +2,540/−49) in the PR body.
Note: `git diff --stat main` spans the whole open PR stack
(#45→#50 + this branch); the stat above is scoped to THIS branch's
base `d154bb44` for honesty.

## 7. Found-not-fixed + owner decisions needed

1. **Labor-Day session split (P4 finding):** `isCMEHoliday` full-closes
   calendar Sep 7, so Sunday-evening ASIA (17:00–24:00, session-day
   2026-09-07) trades but Monday 00:00+ idles the cycle — the session
   gate sits ABOVE EOD-flat, so a Sunday-evening entry rides into
   Monday on NT8 brackets alone until the 12:00 halt. Mitigations
   available NOW: the new pause (pause Sunday evening), or refine
   isCMEHoliday later (changes IsCMEOpen semantics repo-wide — not
   additive, owner call).
2. **Dec 24/31 + Thanksgiving half-day entries are inert** while those
   dates stay full closures in code (12-31 is additionally WRONG in
   code — CME says normal equity session; the bot just sits out a
   tradeable day; conservative, not dangerous).
3. **journald suppression** (>114k msgs/30s): P6 ships the durable DB
   tee, but the flood source is per-frame INFO logging in tcp_server;
   demoting those to Debug or raising RateLimitBurst
   (deploy/install-journald.sh, owner-gated sudo) would restore
   journal usability.
4. **Auto-resume after deploy:** after the 09:25 hotfix restart the
   trader did not resume until the owner pressed Start (09:36:31);
   worth a dedicated verify of the `IsRunning` auto-start path on the
   next planned restart.
5. **Clock-drift observation:** a "local clock BEHIND the feed by
   188s" C2 warning fired at 09:06:51 while rtc-vs-WSL measured −1s —
   the feed-vs-local pair disagrees with the host-vs-WSL pair; the P1
   early-warning + 15-min guard will now surface any recurrence with
   data.
6. **Postgres path:** the `error_class`/`log_events` migrations use
   the house AutoMigrate pattern, which SKIPS existing Postgres tables
   (pre-existing convention); SQLite (live) migrates cleanly.
7. Legacy `stopUntil` field + loop:248 gate remain dormant by the
   additive rule; the new pause supersedes them.

## 8. Sequencing note

The P0 phantom-position dispatch (PR #50) preempted this branch after
P8 closed green; verdict there: no phantom (the skip was correct; the
36s "sync" close is the reconciler's deliberate flat-grace), fix =
skip-gate desync cross-check, deployed. This branch resumed REBASED on
it; `skipWhileOpen` carries both the desync check and the ledger-close
changes. V5 of that hotfix: cycles #20165 (09:22:06, 61.9s AI call)
and #20166 (09:37:14, 36.0s AI call), both "MNQ wait succeeded".
