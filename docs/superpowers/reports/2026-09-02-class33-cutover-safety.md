# Class 33 — cutover safety: in-flight leg · real leg 4 · boot-time arm sweep

**Dispatch:** CLASS 33, owner hoang, 2026-09-02. One wave, one cutover.
**Base:** dev `f28a6250` (live rev d5a6e138, class 41). **Worktree:** `../nofx-class33` (locked).
**Evidence tiers:** [A] verified directly · [B] inferred from strong evidence · [C] speculation.

## 1. The three defects

**B1 — no in-flight leg.** The cutover rite (PART 3 steps 1-7) checked exposure and never
running work. 2026-08-31 17:34 CT a `kill -9` landed while a planner chain was on attempt
3/3: the chain died silently — no v2, no fail-closed line, nothing re-claimed it. Four later
cutovers held on this by agent discipline (A6), never by the rite. [A]

**B2 — leg 4 could not fail.** `TCPTrader.GetOpenOrders` was `return []types.OpenOrder{}, nil`
(tcp_trader.go:1149). The open-orders leg passed VACUOUSLY at cutovers 35, 36, 37, 38, 39, 40
and 41, and the full-system audit quoted "→ []" as evidence of flatness. NT8 emits no
working-order frame (audit F12), so the `armed_orders` ledger is the only real source — it is
what actually held the 09-01 swap (arm 29 WORKING). A24: a check that cannot fail is not a
check. [A]

**B3 — pre-boot arms orphaned.** 2026-09-02 00:16 CT the owner said "just go" with S1 @29044
and S3 @29068.05 resting. The old process died; its orders at NinjaTrader did not. They sat
with NO listener for 15 minutes until the stale-window reconcile cancelled them at 00:31:48,
while the new binary re-armed its own S1/S3 at 00:16:32 and position 587 opened at 00:17:44 —
so for minutes TWO S3 orders existed at the broker. A fill on the dead process's order would
have been a position no stop was attached to (class 27 again). It resolved by luck. [A]

## 2. What changed (file:line)

| Item | Location | Behaviour |
|---|---|---|
| Leg 4 real | `trader/ninjatrader/tcp_trader.go` `SetOpenOrdersSource` + `GetOpenOrders` (was :1149 stub) | reports the ledger; **unwired or erroring source returns an ERROR**, never empty |
| Source label | `trader/types/interface.go` `OpenOrder.Source` | `ledger (no NT8 order frame — F12 open)` on every row |
| Ledger renderer | `trader/class33_boot_sweep.go` `ledgerOpenOrders` | non-terminal rows → OpenOrder; `armed` rows report status `ARMED` (authorized, not at the broker) |
| Wiring | `trader/auto_trader.go` (end of `NewAutoTrader`) | wired at construction so the dead-man watchdog's own probe has a source from cycle 1 |
| Pre-boot decidability | `store/boot_sweep.go` `ProcessBootID` / `ListPreBoot`; `store/armed_orders.go` `BootID` column + idempotent ALTER | `<pid>-<unix ms>`; empty stamp counts as pre-boot (legacy orphans) |
| Stamp survival | `store/armed_orders.go` UpsertArm | create + re-authorize stamp THIS boot; a same-identity re-arm does **not** refresh it, so a cycle cannot erase the sweep's evidence |
| Boot sweep | `trader/class33_boot_sweep.go` `sweepPreBootArms` / `sweepPreBootArmsWith`, called at `trader/armed_executor.go:184` | head of `maybeManageArmedOrders`; placement is at :480, so sweep-before-arm holds **by position** |
| Counter | `store/boot_sweep.go` `arms_boot_swept_class33` in system_config | recorded, survives restarts (class-35 lesson) |
| Five-leg gate | `trader/class33_cutover_gate.go` + `api/handler_order.go handleCutoverGate` + `api/server.go` route | `GET /api/cutover-gate`; a leg that cannot be evaluated FAILS; panic-safe |
| Leg 5 | `AnyPlannerReadInFlight` | scans the claim map for **any** date/session of this trader |
| Boot lines | `kernel/levels_volume_boot.go` (static) + `BootSweepBootLine` (per sweep) | `🛡 cutover safety (class 33)` |
| Checklist | `docs/superpowers/AUDIT-CHECKLIST.md` slot **33** filled; PART 3 step 4 → five legs; step 5 gains the override rule | 41 was the highest class; 33 was the reserved gap |
| Guide | `web/src/guide/content/guards.ts` (HARD row + five-leg section), `status.ts` (2 boot lines) | knob census unchanged at 44 |

**Sweep semantics, exactly.** For each pre-boot non-terminal row: no `signal_id` (authorized,
never placed) → **left armed**, logged, nothing cancelled — cancelling it would silently
disable the play with nothing at the broker. With a signal id → `CancelOrder` FIRST; only on a
successful cancel does the ledger flip to `cancelled` with reason
`boot_sweep: pre-boot order, process restarted`. A FAILED cancel leaves the row non-terminal
and does **not** latch the sweep — never hide a possibly-live order behind a clean ledger.

## 3. Tests (E1-E9)

E1 quoted **failing** on the pre-fix tree, then passing:

```
=== RUN   TestClass33PinLeg4Stub
    class33_leg4_test.go:38: pre-fix tree: SetOpenOrdersSource does not exist — leg 4 has no source at all
    class33_leg4_test.go:46: CLASS 33: leg 4 saw 0 working order(s) while the ledger holds 1 —
                             the stub returns empty, so the flat gate PASSES with an order resting at the broker
--- FAIL: TestClass33PinLeg4Stub (0.00s)
```

After the fix: `ok nofx/trader/ninjatrader`.

| Test | Covers |
|---|---|
| `TestClass33PinLeg4Stub` | E1 — the pin (RED then GREEN) |
| `TestClass33Leg4LedgerErrorFailsLoudly` · `TestClass33Leg4UnwiredFailsLoudly` | E2 — error and unwired both FAIL the leg |
| `TestClass33PreBootDecidability` · `TestClass33ReArmKeepsForeignStamp` | E6 — stamp semantics, cross-trader scoping |
| `TestClass33BootSweptCounter` | recorded counter across a real store reopen |
| `TestClass33BootSweepCancelsPreBootArms` | E4 — two pre-boot arms cancelled, frames sent in order, counter=2, latched once |
| `TestClass33BootSweepNoPreBootRows` | E5 — no-op, counter unmoved, own arm survives |
| `TestClass33BootSweepLeavesUnplacedArms` | unplaced arm never cancelled |
| `TestClass33BootSweepCancelFailureKeepsRowLive` | failed cancel keeps the row live and retries |
| `TestClass33BootSweepIgnoresShadowedRows` | E7 — 0C shadow sweep untouched, no double-cancel |
| `TestClass33SweepPrecedesPlacement` | ordering pin (sweep@184 < placement@480, at the head) |
| `TestClass33GateReturnsAllFiveLegs` · `TestClass33Leg5PlannerInFlight` · `TestClass33Leg4FailsWithRestingArm` | E8, E3 |

E9: `go test ./...` **green** (full suite) · `go build ./...` clean · vitest guide **10/10** ·
`tsc --noEmit` clean.

## 4. Cutover

**Five legs, quoted from the RUNNING binary (d5a6e138, PID 1744258) at 06:51 CT 2026-09-02.**
The `/api/cutover-gate` endpoint ships IN this wave, so this first gate is assembled by hand
from the same five sources; every cutover after this one quotes the endpoint.

| Leg | Source | Value | Pass |
|---|---|---|---|
| 1 db_open_positions | `sqlite3 -readonly data/data.db "select count(*) from trader_positions where lower(status)='open'"` | `0` | ✅ |
| 2 api_positions | `GET /api/positions?trader_id=…&account=Sim101` | `[]` | ✅ |
| 3 nt8_positions_snapshot | journal 06:51:13 | `positions snapshot account=Sim101 count=0`, `account=SimAccount1 count=0` | ✅ |
| 4 working_orders (ledger) | `select count(*) from armed_orders where state in ('armed','working')` | `0` (22 cancelled, 9 filled) | ✅ |
| 5 planner_in_flight | `GET /api/plan/today` + journal | `replan_in_flight=False reading=False`; no `Request URL (stream` since 06:40 | ✅ |

**Window (A7):** 06:51 CT — outside 16:45-17:10; LONDON active but its read fail-closed at
01:37 CT (no_trade) and no arms rest; position 587 CLOSED at 01:03 CT. NY read is 08:00 CT, so
this boot lands before it. **In-flight (A6):** no planner chain open.

**Build:** clean clone `--no-local` of dev @ 8a756bba → `vcs.revision=8a756bba4a21ab455beafac75bf6415e71de2fb9`,
`vcs.modified=false`. Staged as `nofx-bin.next`. Rollback slot: `nofx-bin.prev.boot` (d5a6e138).

**Swap (A26 — the classifier denies `kill -9` to the agent; the OWNER runs this):**

```
cd /home/hoang/nofx
echo 8a756bba4a21ab455beafac75bf6415e71de2fb9 > deploy/RELEASE
cp nofx-bin nofx-bin.prev.boot && mv nofx-bin nofx-bin.old.d5a6e138 && mv nofx-bin.next nofx-bin
kill -9 1744258
```

_(boot checklist appended after the passed boot; marker committed only then — A19)_

## 5. What the owner will still see wrong (A15)

- **A swept scenario does not re-arm on the same plan version.** The owner ruled the swept row
  goes to `cancelled`; `UpsertArm` re-authorizes a terminal row only on a version change. So a
  restart during a live setup ends that setup until the plan version moves. That is the
  conservative direction and it is the ruled behaviour — but on 2026-09-02 00:16 CT it would
  have meant no position 587.
- **The sweep needs the armed subsystem to run.** It sits at the head of
  `maybeManageArmedOrders`, which returns early when day_plan is off or the exchange is not
  ninjatrader. With day_plan off, pre-boot orphans are still covered only by the stale-window
  reconcile backstop (unchanged, by stop-line).
- **Leg 4 is the ledger, not the broker.** Until NT8 exposes a working-order frame (audit F12),
  an order placed outside this bot, or one whose ledger row was lost, is invisible to leg 4.
  Every row says so in its `source`.
- **`cancelUnfilledEntriesAfterReconnect` now logs on reconnect** where it was blind before
  (its comment claiming "no cancel wire command exists" is stale — `CancelOrder` exists at
  tcp_trader.go:546). Behaviour deliberately unchanged (stop-line); worth its own wave.
- **An unwired leg-4 source fails the dead-man watchdog's probe**, which is fail-closed: it
  blocks trading rather than trading blind. Wiring happens in `NewAutoTrader`, so this is a
  code-path invariant, not a runtime risk.
- The classifier denies `kill -9` to the agent; every cutover stops for the owner to run it.

## 6. Rollback

```
cp nofx-bin.prev.boot nofx-bin && echo d5a6e138da851f2ee9ceba22424363bba0f219eb > deploy/RELEASE && kill -9 <MainPID>
```

### 4.1 Boot record (owner GO 06:57 CT 2026-09-02)

Gate re-quoted fresh at **06:57:32 CT** immediately before the swap: leg 1 `0`, leg 4 `0`,
leg 5 `0` stream calls since 06:40, tree porcelain-clean, running PID 1744258. RELEASE written
BEFORE the swap; marker committed only after the boot below (A19).

Old process exited 06:57:44 (`status=9/KILL`); service started 06:57:49; boot 06:57:50, **PID
2065518, exactly one nofx-bin process, 0 `[ERRO]` lines, 0 TradingRefused**:

```
🔐 BOOT INTEGRITY OK — rev 8a756bba4a21 · built 2026-09-02T11:50:23Z · expected 8a756bba4a21 · goldens PASS
🛡 cutover safety (class 33): flat gate legs=5 (db_open_positions · api_positions ·
   nt8_positions_snapshot · working_orders · planner_in_flight) via GET /api/cutover-gate;
   leg4 reads the armed_orders LEDGER (was a stub returning empty — passed vacuously at
   cutovers 35→41); boot sweep cancels pre-boot arms before ANY re-arm, counter arms_boot_swept_class33
🛡 cutover safety (class 33): gate legs=5 · leg4=ledger · boot sweep cancelled 0 pre-boot arm(s)
   (0 authorized-but-never-placed left for this process)
```

Surviving ledger lines: 🚀 planner speed wave · 🗓 session reads · 🪢 class 27 · 🧪 validator
hints 15 sites (34+38) · 📜 prompt/validator contract 17 restrictions (38) · ⚖ arm normalizer
(39) · 🗓 preflight (36) · 🧾 P&L surfaces 12/0 (40) · 🛰 planner client (37) · 🔁 planner
stream policy (41).

**Live proof of the gate** — `GET /api/cutover-gate` on the running binary, 06:58 CT:

```json
{"ready": true, "legs": [
  {"n":1,"name":"db_open_positions","pass":true,"detail":"0 open row(s)","source":"sqlite trader_positions"},
  {"n":2,"name":"api_positions","pass":true,"detail":"0 position(s)","source":"trader.GetPositions"},
  {"n":3,"name":"nt8_positions_snapshot","pass":true,"detail":"count=0","source":"NT8 positions frame"},
  {"n":4,"name":"working_orders","pass":true,"detail":"0 non-terminal arm(s)","source":"armed_orders ledger (no NT8 order frame — F12 open)"},
  {"n":5,"name":"planner_in_flight","pass":true,"detail":"no planner read claimed","source":"plannerReadInFlight claim"}]}
```

**PROOF STATUS (A20).** Legs 1-5 and the endpoint are PROVEN live. The boot sweep is
SHIPPED-UNPROVEN in the only sense that matters: nothing was resting at 06:57, so it correctly
reported `cancelled 0 pre-boot arm(s)`. **The live proof is the NEXT cutover that happens with
an arm resting** — it must show `🛡 boot sweep CANCELLED pre-boot arm (class 33): …` naming the
signal id, before any re-arm. Until then the sweep's evidence is fixtures only, and this report
says so.
