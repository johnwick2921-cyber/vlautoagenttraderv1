# CLASS 48 — THE DECISION PATH BYPASSES THE ARM-SEAM GATES
**Phase 1 read-only report** · 2026-09-02 · owner hoang · read-only, no lock, no writes
Worktree `~/nofx-gate` · branch `fix/class48-entry-gate` · base `8dad304e` (dev HEAD)
Evidence tiers: [A] directly verified · [B] inferred · [C] speculation.

---

## SECTION A — DANGER CHECK (first, per A1)

1. **One live arm, safe side.** `armed_orders` id=32 · NY v5 S3 · LONG limit
   29070.00 · stop 29019.67 · state=`armed` (no signal_id → nothing at the broker).
   Market 29131.00 (12:19 bar). A long limit resting BELOW market is the correct
   side. No action needed. [A]
2. **`test_seam=ON`** — the armed test seam (a POST-driven placement endpoint,
   `armedTestSeamOn()`, armed_executor.go:1450+) booted ON today
   (`⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=ON arm_rr=2.0…`,
   boot 00:01:07). Gated to SIM by code, but it is a live debug surface — flagged,
   not touched (read-only). [A]
3. **The R:R entry-reference mismatch is still live.** The running binary
   `0465a10b` evaluates the entry floor against the prompt-time snapshot
   (`kernel/engine_position.go:136-145`, `entrySource="current_price_snapshot"`),
   then fills a MARKET order ~10 points away. Today that admitted TWO entries
   below the owner's 2.0 floor at the real fill: 587 (1.09) and 589 (1.61).
   Any further decision-path entry today can still pass the floor on a stale
   price and fill sub-floor. This is the live-0465a10b finding of the parallel
   PARTIAL forensic (`2026-09-02-today-losses-forensic-PARTIAL.md`), re-verified
   here. [A]
4. No new trades since 10:37 (still exactly positions 587-590). [A]

---

## SECTION B — SOURCE AND PLACING PATH FOR 587 / 589 / 590

DB `trader_positions` (read-only, 12:20 CT) [A]:

| id | side | source | entry_order_id | exit_order_id | close_reason | plan / version / cited | plan_matched | adherence |
|---|---|---|---|---|---|---|---|---|
| 587 | LONG | **system** | **NULL** | d1e5c248-…-acda477d | **sync** | ASIA v7 · S3 | 1 | B |
| 589 | LONG | **system** | **NULL** | 56d0fa5b-…-0539ae6d | **sync** | NY v3 · S3 | 1 | A |
| 590 | LONG | **system** | **NULL** | 447ef539-…-d5ce8cef | **sync** | NY v5 · S4 | 1 | A |

`source=system` + `entry_order_id NULL` + `close_reason=sync` = the kernel-cycle
MARKET path: the position is materialized later by the reconcile/positions
stream, not by an armed-limit fill (an armed fill would carry the arm's
signal_id in `entry_order_id`). [A]

The placing code path, quoted [A]:

1. `kernel/engine_analysis.go:866` — `parseFullDecisionResponse` → `validateDecisions(...)`
   (the R:R / min-SL / HTF-veto chain, evaluated at the PROMPT snapshot price).
2. `trader/auto_trader_loop.go:884` — `at.executeDecisionWithRecord(&d, &actionRecord)`
   (feed / dead-man / frozen / boot-integrity / pause / roll / halt / last-entry /
   session / plan-mode / approval gates).
3. `trader/auto_trader_orders.go:327-329` — `case "open_long": return at.executeOpenLongWithRecord(...)`.
4. `trader/auto_trader_orders.go:449` — `executeOpenLongWithRecord`:
   reconcile → `enforceMaxPositions` → same-side duplicate check →
   NT8 pre-entry `SetStopLoss`/`SetTakeProfit` (OCO bracket) →
   **`at.trader.OpenLong(decision.Symbol, quantity, decision.Leverage)`**
   (market order, NT8 SIM). The AI's authored stop rides as-is.
5. Journal proof of each pass-through (live log, quoted [A]):
   - 587: `00:17:43 📐 R:R eval MNQ open_long: entry=29069.5000 (current_price_snapshot) SL=29048.0000 TP=29113.2500 → risk=21.5000 reward=43.7500 R:R=2.03 (min 2.00) → PASS` — then `📋 advisory: open_long cited S3 ✓ matched (plan v7)`; fill 29079.25 → **real R:R 1.09**.
   - 589: `09:41:04 📐 R:R eval … entry=29182.0000 … R:R=2.02 (min 2.00) → PASS` — `📋 advisory: open_long cited S3 ✓ matched (plan v3)`; fill 29192.50 → **real R:R 1.61**.
   - 590: `10:36:09 📐 R:R eval … entry=29196.5000 … R:R=2.00 (min 2.00) → PASS` — `10:37:16 📋 advisory: open_long cited S4 ✓ matched (plan v5)`; fill 29193.25.

A second, fully-gateless path exists and is quoted for completeness: the agent
chat `execute_trade` (`agent/trade.go:175-202`) calls
`underlyingTrader.OpenLong/OpenShort` directly after only `validateTradeAction`
(quantity/notional/leverage caps). None of the five gates below run there.
Today's four trades did NOT use it (`source=system`), but it is a live bypass. [A]

---

## SECTION C — THE FIVE GATES: ARM SEAM vs DECISION PATH (call-site census)

### C1. R:R floor — enforced on BOTH paths, with different references and floors

- **Arm seam:** `armMinRR()` = `ARM_MIN_RR` env, default **2.0**
  (`trader/armed_executor.go:54-68`); applied in `armGateVerdictFor`
  (`armed_executor.go:1199-1205`), the only caller chain being
  `maybeManageArmedOrders` → `armGateVerdictFor` (`:381`). Rationale comment
  (`:55-58`): "armed limits fill AT the level … the global entry floor (3.0) is
  NOT lowered". [A]
- **Decision path:** `validateDecision` (`kernel/engine_position.go:146-196`)
  enforces `min_risk_reward_ratio` (resolved **2.00** today — the owner's 09-01
  08:13 save). Entry reference = `d.Price` if the AI set a limit, else
  `ctx.MarketDataMap[symbol].CurrentPrice` — **the prompt-time snapshot**
  (`:136-145`). Fill is a market order. The mismatch today: 587 snapshot 29069.50
  vs fill 29079.25 (+9.75); 589 snapshot 29182.00 vs fill 29192.50 (+10.50) —
  both sub-floor at the real fill. [A]
- **Chat path:** no R:R gate at all (`agent/trade.go`). [A]

### C2. Shadow map (0C) — ARM SEAM ONLY

- `conditionShadowedFor(condition, session)` (`armed_executor.go:30-42`) resolves
  session override > strategy base > env; called ONLY at `armed_executor.go:286`
  (arm authoring: shadowed arms are written inert as state `shadowed`, never
  placed) and `:603` (E8 shadow-A/B logging). [A]
- Resolved map today (boot 00:01:07, journal): `🔬 conditions: live [acceptance,
  breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim] ·
  shadow [breakout_retest, fvg_entry]`. **`breakout_retest` is SHADOW.** [A]
- **Decision path: zero references.** 589 = NY v3 S3 `long breakout_retest`
  (plans.jsonl) → shadowed, admitted. 590 = NY v5 S4 `long breakout_retest` →
  shadowed, admitted. [A]
- The planner uses `ResolvedLiveConditions` only to build the prompt's live
  vocabulary (`auto_trader_planner.go:1403`) — authoring guidance, not a gate. [A]

### C3. Scenario-direction consistency — detector on the decision path, never a gate

- **Arm seam:** `armGateVerdictFor` refuses a non-long/non-short direction
  (`armed_executor.go:1175-1178`) and — only in `plan_mode=direction` — an arm
  against the PLAN BIAS (`:1179-1185`). The arm side IS the scenario side. [A]
- **Decision path:** `recordPlanCitation` calls `kernel.ClassifyCitation`
  (`auto_trader_planner.go:2453-2509`) which DETECTS a direction mismatch, but
  the function comment says it plainly: **"Advisory only — it never gates the
  trade (plan restricts, never compels)"** (`:2456`). The `planModeBlocked`
  gate (`auto_trader_planconfig.go:188-233`) checks the plan BIAS in
  `direction` mode and citation-match in `strict` mode — the default
  `advisory` mode returns `("", false)` immediately (`:198-200`). Today's mode
  was advisory. [A]
- **Measured correction to the dispatch pin on 590:** at the version the
  decision cited (v5), S4's `direction` is **long** (`condition: breakout_retest`,
  trigger "…the retest touch at 29171.25 from above is the continuation long
  entry"). At v1/v3/v4/v7 S4 was **short** (reject). So "long on a short
  scenario" is true of S4 as the day's other versions describe it, but at the
  cited v5 the executor's own classifier logged `✓ matched (plan v5)` — the
  direction-mismatch leg, as stated, does NOT fire for 590. What measurably
  refuses 590 is the shadow leg (breakout_retest). The direction leg remains a
  real guard: it refuses an action opposite the cited scenario's direction at
  the cited version, which is exactly the 587-style class if it ever occurs. [A]

### C4. min-SL / stop composition (0B) — split enforcement

- **Arm seam:** `composeArmStop` (`trader/arm_stop_anchor.go:71`) rewrites the
  stop: beyond the nearest seated level + clearance, floored at
  `MIN_SL_ATR_MULT × ATR5m`, widest wins (`armed_executor.go:336-349`, the `🛑
  arm stop …` line). Plus the ×ATR5m leg in `armGateVerdictFor`
  (`armed_executor.go:1207-1217`). [A]
- **Decision path:** `validateDecision` min-SL legs 1+2 (`kernel/engine_position.go:205-252`):
  ×ATR5m and cited-level clearance — enforced. But **stop COMPOSITION is
  arm-seam-only**: no anchor/floor/bound rewrite exists on the decision path.
  The AI's authored stop rides verbatim. [A]
- Evidence: 589's authored stop 29115.00 (77.5 pt, wide enough to pass min-SL
  anyway) was never composed against a level; on the arm path the S4 short leg's
  stop WAS composed: `09:41:05 🛑 arm stop NY S4 leg 1 short: stop 29371.09
  (authored 29355.00 WIDENED) · anchor OB(bull)·1h (HTF) 29324.50 … atr_floor
  29371.09 (1.5×ATR5m 35.89) · bound=atr_floor`. Two different stop regimes,
  two paths. [A]

### C5. one-live-arm — ARM SEAM ONLY

- `oneLiveArmGuard` (`armed_executor.go:533-558`) refuses an opposite-side arm
  while a position is open (netting account: a fill would silently NET the open
  position — the 2026-08-31 S3-SellShort live proof). Called only at
  `armed_executor.go:434`. [A]
- **Decision path:** no opposite-side gate. `executeOpenLongWithRecord` checks
  only the SAME side (`auto_trader_orders.go:463-469`: "❌ … already has long
  position, close it first"). Nothing prevents the AI opening the opposite side
  into an open position via the market path. Chat path: nothing. [A]

### Gate × path matrix (summary)

| Gate | Arm seam (`armed_executor.go`) | Decision path (kernel cycle) | Chat `execute_trade` |
|---|---|---|---|
| R:R floor | ✅ `armGateVerdictFor:1199` (ARM_MIN_RR 2.0, leg prices) | ✅ `engine_position.go:146-196` (min_risk_reward_ratio, **snapshot ref — fills sub-floor**) | ❌ none |
| Shadow map (0C) | ✅ `:286` | ❌ none | ❌ none |
| Scenario-direction | ✅ side armable + plan-bias (direction mode) | ⚠️ detected + logged, never gates (`recordPlanCitation`); plan-bias only in `direction` mode | ❌ none |
| min-SL ×ATR5m + level clearance | ✅ `armGateVerdictFor:1207` + compose | ✅ `engine_position.go:205-252` | ❌ none |
| Stop composition (0B) | ✅ `composeArmStop` | ❌ none | ❌ none |
| one-live-arm | ✅ `:434` | ❌ none (same-side dup check only) | ❌ none |

---

## SECTION D — DOES THE CLASS-44 `config_changes` TABLE EXIST?

**No.** `SELECT name FROM sqlite_master WHERE name='config_changes'` → empty at
12:20 CT (re-checked; the losses forensic found the same at 11:30). [A]

What class 44 actually is and persisted [A]:

- `store/config_diff.go` (authored 09-02 07:57, commit `62a2e3dd`
  "feat(repair-parse): … config diff on Studio save, boot line, checklist class
  43"): `ConfigChange{ID, TraderID, Strategy, Knob, OldValue, NewValue, Source,
  At}` with `TableName() = "config_changes"`; `NewConfigChangeStore` runs
  `db.AutoMigrate(&ConfigChange{})` (`config_diff.go:38-42`).
- Wiring: `api/strategy.go:868-869` — on every Studio save,
  `s.store.ConfigChanges().Save(changes)` with
  `DiffStrategyConfig(before, after)` (resolved-value dotted paths); lazy
  construction at `store/store.go:400-404`.
- **The wiring commit IS in the live binary** (`62a2e3dd` is an ancestor of the
  running rev `0465a10b`, merge-base verified). [A]
- Therefore class 44 has persisted **NOTHING yet**: the table materializes on
  the FIRST strategy save after the wiring was deployed (AutoMigrate at first
  `ConfigChanges()` access). The owner's 09-01 08:13 R:R 3.0→2.0 save PREDATES
  the wiring and has **no row** — its values are inferable only from journal
  lines (this is why the losses-forensic E9 `config_at_trade.csv` was a
  reconstruction, documented in its NOTES.md). [A]

---

## SECTION E — MEASURED PIN VERIFICATION (the direct test of the dispatch premise)

| pin | dispatch claim | measured truth | gate leg that refuses it on the fix |
|---|---|---|---|
| 587 | R:R 1.09 | TRUE — PASS 2.03 @ snapshot 29069.50, fill 29079.25 → **1.09** | R:R at the REAL execution price |
| 589 | breakout_retest shadowed | TRUE — NY v3 S3 = long breakout_retest, shadow list = `[breakout_retest, fvg_entry]` | shadow |
| 590 | long on a short scenario | **NOT at the cited version** — NY v5 S4 = long breakout_retest (`✓ matched (plan v5)`); S4 was short at v1/v3/v4/v7 | shadow (breakout_retest); direction leg does not fire at v5 |

All three PASSED on the current code today (journal `→ PASS` + `✓ MNQ open_long
succeeded` lines, quoted above). The fix must refuse all three: 587 via R:R,
589 via shadow, 590 via shadow — and the direction leg guards the "long on a
short scenario" class for any future citation where the cited version's
direction actually opposes the action.

---

## SECTION F — PHASE 2 CONTRACT (implemented after this report)

ONE `EntryGate(intent EntryIntent) (reason string, refused bool)`:

- Legs, in order: direction (action vs cited scenario's direction at the cited
  version) → shadow (cited condition) → R:R (real execution-time price vs
  `min_risk_reward_ratio`) → min-SL (×ATR5m) → one-live-arm (opposite-side
  while a position is open).
- Called by BOTH seams before any order leaves:
  - arm seam: `maybeManageArmedOrders` before the `ArmedOrderDB` row write
    (refusal → `IncArmRefusal(store, at.id, tradeDate, session, "entry_gate:"+leg)`
    + `⚔️ arm REFUSED` log);
  - decision path: `executeDecisionWithRecord` before `executeOpen*`
    (refusal → `actionRecord.Error = "entry_gate: "+reason`,
    `Success=false`, `telemetry.IncGateBlock(at.id, "entry_gate")`,
    `⛔ … refused` execution-log line → recorded in `decision_records`).
- Pins as unit tests replaying the three trades' real intents.

---

## SECTION G — PHASE 2 DELIVERED (owner-approved)

**Code (worktree `~/nofx-gate`, branch `fix/class48-entry-gate`):**

- **`trader/entry_gate.go` (new)** — `EntryGate(intent EntryIntent) (reason string, refused bool)`, pure and nil-safe, legs in order: plan-bias (direction mode) → scenario-direction (action vs the cited scenario's direction) → shadow map → **R:R at the execution price** (the 587/589 fix) → min-SL ×ATR5m → one-live-arm. Fail-open on missing inputs (mirrors `validateDecision`).
- **Arm seam wiring** — `armed_executor.go` in `maybeManageArmedOrders`, after the `oneLiveArmGuard` block and before the `ArmedOrderDB` row write: `at.entryGateForArm(...)`; on refusal an existing resting arm for the spec is cancelled (reason `entry_gate:…`), and the refusal is RECORDED via `store.IncArmRefusal(..., "entry_gate:"+class)` + `🚦 entry-gate REFUSED` log.
- **Decision path wiring** — `auto_trader_orders.go` in `executeDecisionWithRecord`, after the legacy gates and before `executeOpen*`: `at.entryGateForDecision(decision, livePrice)` with `livePrice` = execution-time `market.GetWithExchange().CurrentPrice`. On refusal: `telemetry.IncGateBlock(at.id, "entry_gate")`, `actionRecord.Success=false`, `actionRecord.Error="entry_gate: …"` → recorded in `decision_records` (execution_log + risk_check_error) and rendered as a ⛔ refused row.
- **`trader/entry_gate_test.go` (new, 10 tests, all green)** — pin replays: 587 refused with `R:R 1.09` at the real fill (and admitted at the snapshot price the old gate used — the fix is the reference); 589 and 590 refused as SHADOW (`breakout_retest`); direction-mismatch refused for the owner's stated 590 class (long citing a SHORT scenario); one-live-arm; min-SL; clean-intent admit; arm-seam builder; decision-path builder. **"Pass on current code, fail on the fix":** the pass-on-current evidence is the live journal quoted in Section B (all three logged `→ PASS` + `✓ MNQ open_long succeeded`); the fail-on-fix side is these tests.
- **AUDIT-PLAYBOOK LAW** — class 48 appended to `docs/superpowers/AUDIT-CHECKLIST.md` in this branch.
- **GUIDE CONTENT LAW** — `web/src/guide/content/guards.ts` truth table: added the `Entry gate (class 48) — ONE gate, BOTH paths` row and corrected the stale `ARM floors` row (it claimed "the 3.0 floor stays the FULL market-entry gate" — the entry floor is the resolved `min_risk_reward_ratio`, now judged at the live price). `GUIDE_BUILT_REV` bump is deferred to the deploy dispatch: it must equal the BUILD rev of the shipped binary, which does not exist until a cutover is authorized (A19; no deploy was authorized in this dispatch).

**Not shipped.** No RELEASE, no marker, no cutover — A2b honored (main tree untouched, `git status` clean, worktree locked). `go build ./...` green; `go test ./...` exit 0 (27 packages).

