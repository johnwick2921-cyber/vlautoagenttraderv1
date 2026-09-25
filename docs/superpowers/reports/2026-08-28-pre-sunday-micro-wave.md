# Pre-Sunday Micro-Wave — F1–F5 (BUILD & PROVE, NO DEPLOY)

- Branch: `fix/pre-sunday` (off `dev` `dbcb61ac`) · Date: 2026-08-28
- Status: **proven & parked — awaiting owner "go cutover"** (deploy well before Sunday 17:00 CT)
- Canon: MAIN-TREE LOCK LAW recorded this wave · GUIDE CONTENT LAW satisfied (guide updated in-PR; rev bump rides the cutover marker)

## F1 — waterfall immediate-mode is PLAN-AUTHORABLE (ruling S4, scoped)

Owner ruling: immediate-mode is approved **for the AI-proposed path only**; arms stay pullback-only (no stop-entry frames exist in the C# bridge).

- `kernel/breakdown_continue.go` — `ValidateBreakdownContinueScenarios`: immediate-mode authoring is legal **as soon as the displacement exists** (measured ≥ `BD_MIN_DISP_ATR`×ATR5m and no reclaim close) instead of requiring the full 2-close leg 1. The 2nd confirming close is the ENTRY trigger itself — requiring it at write time made the play un-authorable mid-waterfall. Pullback keeps the strict full-leg rule; an immediate scenario with ZERO displacement is still rejected; a reclaim close voids either mode. Arms remain pullback-only (unchanged `ArmSpecValid` + validator).
- `kernel/planner_prompt.go` — WATERFALL PLAY rule rewritten: `entry_mode=immediate` = AI-path market entry on the 2nd confirming close through the FULL gate chain (min-SL ≥ 1.0×ATR5m, R:R ≥ min_risk_reward_ratio, min-conf, HTF veto), **CHOOSE immediate for no-retest waterfalls (one-sided delivery, displacement expanding)**, pullback when a retest is likely. Immediate arms explicitly rejected by the machine.
- Guide (canon): both waterfall play cards updated with the immediate ruling + choice guidance.
- **Fixture tests** (`kernel/breakdown_continue_test.go`):
  - `TestBreakdownImmediateAuthorableBeforeSecondClose` — 1 beyond close + 1.3×ATR displacement: immediate validates, pullback rejects, zero-displacement immediate rejects.
  - `TestBreakdownImmediateFixturePassesGateChain` — **the 08-28 10:48 leg**: entry at the 2nd confirming close (29646.00), SL = pullback extreme + 1.0×ATR5m (risk 40.5 ≥ min-SL), TP = entry − 3R (29524.50, R:R = 3.00 ≥ 3.0), confidence 65 ≥ 60 — all three numeric gates PASS, and the real tape fills the target at 11:15 (low 29523.75) without touching the stop. Would-have **+$243** on this instance. (HTF veto: 1h was already TRENDING_DOWN that morning — a continuation short aligns, so cross-veto passes; noted [B], not unit-asserted.)

## F2 — RepairArmedLineage matcher fixed (#568 + #570)

**Root cause of the miss:** the matcher keyed on the ledger row's `entry_px`, which **drifts on re-arm** — the v6 re-spec overwrote the v1 entry that actually filled (#568: row entry 29702.0 vs real fill 29642.00; #570: row entry 29480.0 vs fill 29463.25). The true fill lived only in the reason string (`fill@29642.00`).

- `store/armed_orders.go` — new `SetFillPrice(id, fillPrice)`: the fill handler now persists the fill price on the row.
- `trader/armed_executor.go` — `onArmedOrderUpdate` filled-branch writes `fill_price`.
- `trader/ninjatrader/reconcile.go` — `StampArmedLineageIfMatched` resolves the authoritative fill via `armedFillPriceFor`: `fill_price` first, then the `fill@…` reason, then `entry_px` fallback; match within one tick of the position entry.
- Tests (`trader/ninjatrader/armed_lineage_repair_test.go` + `store/armed_orders_test.go`): the exact #568/#570 shapes stamp correctly; a drifted entry with NO fill record does not false-match; fill_price persists; `ListNonTerminal` trader-scoping.
- **Cutover E-proof:** the boot-time `RepairArmedLineage` pass will now stamp #568 and #570 — expect the boot line `🩹 RepairArmedLineage: stamped 2 position(s)` (both F-grades clear on the W5 regrade).

## F3 — .gitignore

`.env.bak*` added (and the original `*.bak` rule restored alongside) — live-key backups can no longer be staged by an accidental `git add -A`.

## F4 — ListNonTerminal trader-scoped

`store.ArmedOrdersStore.ListNonTerminal(traderID)` now filters `trader_id = ?`; all 9 call sites in `trader/armed_executor.go` pass `at.id`; the cross-trader isolation test now asserts the OTHER trader's row survives via the scoped query.

## F5 — boot scenario schema-ledger

`kernel.ScenarioSchemaLedger()` (sorted condition vocabulary) + a boot line in `main.go` after the volume-wave ledger:
`📜 scenario schema: 9 conditions [acceptance, breakout_retest, breakdown_continue, breakup_continue, fvg_entry, hold, reclaim, reject, sweep_reclaim]`
— a shipped schema change can never go silent again (the 8th-condition parse-reject class).

## CANON ADDITION — MAIN-TREE LOCK LAW (per P2 S8)

Recorded in `CLAUDE.md` (untracked local) + repo memory: porcelain-clean gate before any dispatch touches `~/nofx` · non-deploy work only via `git worktree add` + `git worktree lock` · `~/nofx-main.lock` marker (owner/PID/expiry) acquired first · `git reset` on dev FORBIDDEN outside the deploy-owning dispatch.

## Gates

| Gate | Result |
|------|--------|
| `go build ./...` | ✅ |
| `go vet ./...` | ✅ clean |
| `go test ./...` | ✅ 27 packages ok, 0 fail |
| Goldens (`-run Golden`) | ✅ 5/5 (prompt change green) |
| Guide `vitest run src/guide/GuidePage.test.tsx` | ✅ 10/10 |
| `tsc --noEmit` | ✅ 0 errors |

Diff: 13 modified + 1 new test file (+235/−38) — formatter churn on unrelated kernel files was reverted (repo is not gofmt-stable under the local Go version).

## Cutover checklist (owner "go cutover", before Sunday 17:00 CT)

1. Merge `fix/pre-sunday` → dev; temp-clone build at merge sha; `vcs.revision == merge sha`.
2. Flat-gate all-origin (DB OPEN=0 · NT8 both accounts count=0 · API `[]` · armed non-terminal=0).
3. Marker commit on dev: `deploy/RELEASE=<merge sha>` **+ `GUIDE_BUILT_REV` → `<merge sha>`** (GUIDE CONTENT LAW — the bump ships with the binary that carries this guide change).
4. Swap → `kill -9` → boot poll: `🔐 BOOT INTEGRITY OK` · goldens PASS · **`📜 scenario schema: 9 conditions […]`** · **`🩹 RepairArmedLineage: stamped 2 position(s)`** · zero ERRO.
5. Post-boot: `/api/status` rev == merge sha · #568/#570 now plan-linked (grades clear on W5).
