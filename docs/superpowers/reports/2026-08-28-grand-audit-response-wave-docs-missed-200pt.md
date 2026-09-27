# Grand-Audit Response Wave — BUILT & PROVEN (NO DEPLOY)

- **Branch:** `fix/grand-audit-response` off dev `e44a66a8` · worktree `~/nofx-gar`
- **Scope:** F1–F6 from the 2026-08-28 grand-audit verdict page. Built, fixture-proven, full regression green. **STOPPED for owner "go cutover" — nothing deployed.**
- **Owner clicks (after cutover):** `proximity_filter_atr → 0.3` · `min_confidence → 65` (Sep-9 re-check scheduled).

---

## F1 [S] — move_stop for materialized positions (the #566 dead cell)

**Root cause (re-confirmed at the code):** `MoveStopToBreakeven` resolved the wire identity ONLY from `lastEntrySignalID` (`tcp_trader.go:485-490`). Reconcile-materialized positions never set it — live proof: position #566 fired auto-breakeven 4× while open (11:45:49 / 11:46:49 / 11:59:50 / 12:00:50 CT) and every send failed `ninjatrader/tcp: no open entry to move the stop`. The trailing path shares the same funnel (`auto_trader_trailing.go:183`), so trailing was equally dead for that entry class. #567/#569's rows also carried empty `entry_order_id`.

**Fix:**
- `store/position.go` — `SetEntryOrderID` (idempotent, never overwrites a non-empty identity).
- `trader/ninjatrader/reconcile.go` — `StampArmedLineageIfMatched` now returns `(stamped bool, signalID string)` and persists the armed ledger's `SignalID` as the row's `entry_order_id`; the materialization site caches it in-memory; `RepairArmedLineage` inherits the backfill (idempotent).
- `trader/ninjatrader/tcp_trader.go` — new `st *store.Store` + `entryOrderID map[string]string` wired at `StartPositionReconcile`; `resolveEntrySignalID` resolves `lastEntrySignalID → materialization cache → persisted row` before the move_stop frame is built. This also unblocks the ModifyBracket live E-proof.

**Fixtures (quoted, PASS):**
- `TestMaterializedArmedFillCarriesEntryOrderID` — twin long/short: armed fill + materialized row → stamp returns the signal id and the row carries `entry_order_id`; second write never overwrites.
- `TestMoveStopUsesMaterializedIdentity` — twin long/short over a REAL loopback (server `Start` + `ListenAddrForTest` + client `ReadFrame`): no in-process signal, empty cache → **move_stop frame SENT with `SignalID = <armed signal>`**, correct `NewStopLoss`, `Account=Sim101` (A2 stamp). The trailing path shares this funnel.
- `TestMoveStopStillFailsWithoutAnyIdentity` — the honest explicit error survives (never a silent no-op).

## F2 [S] — proximity archaeology + restore path

**Verdict: NO revert ever happened — the retune was never a code default.**
- `store/strategy.go:1234` still defaults `ProximityFilterATR: 1.5` at dev head; the S-wave S1 commit `eb3920b7` touched only the clamp (0.5→0.1) and naming — zero diff on the default.
- The stored strategy row says `proximity_filter_atr=1.5`, `updated_at=2026-08-27 14:44:43 CT` — that is the owner's own strict-mode re-save, which round-tripped the already-stored 1.5. Not a persistence bug, no fix needed on storage.
- **The one real defect:** the ENGINE prompt path still had a **0.5 floor** (`kernel/engine_analysis.go:371`) while the bot gate accepted 0.1–3.0 — a 0.3 click would have been honored by one consumer and silently dropped by the other.
- **Fix:** `kernel.ResolveProximityK` (one clamp, 0.1–3.0, default 1.5) now serves BOTH `kernel/engine_analysis.go` and `trader/auto_trader_planconfig.go:proximityFilterATR`. Guide §7's proximity card corrected (default is 1.5; the 0.3 is a saved-value change; units = daily-range proxy).
- **Fixture:** `TestResolveProximityK` — 0.3/1.5/0.1/3.0 pass through; 0.05/3.5/0/−1 fall back to 1.5. **PASS.** The value itself is NOT written — the owner clicks 0.3.

## F3 [A] — HTF veto cross-check (`HTF_VETO_MODE`)

- `kernel/htf_veto.go`: `HTFVetoMode()` reads `HTF_VETO_MODE` — `1h` (default, today's behavior) | `cross` (veto only when **1h AND 4h** both oppose) | `4h`. Unknown values fall back to `1h` with a WARN. Missing snapshots fail open per-TF (cross vetoes strictly less).
- **Replay fixture:** `TestHTFVeto_CrossModeAuditSeven` — the 7 REAL vetoed arms from the C3 table: 1h-only blocks the first 3 (the +$352 would-have-wins), **cross blocks ZERO**, both-oppose blocks under cross, missing-4h fails open, 4h mode vetoes on 4h alone. **PASS.** `TestHTFVeto_UnknownModeFallsBack` **PASS.**

## F4 [A] — breakout_retest arm exclusion

- `kernel/armed.go` `ArmableCondition` now accepts `fvg_entry | reject` only; `plan_doc.go` arm-validator message updated; planner ARMED-ORDERS contract text updated (breakout_retest = normal AI play, never armed). It remains a fully supported non-armed play.
- **Fixture:** `TestArmableCondition` (breakout_retest now in the must-NOT set) + `TestArmSpecValidatedByPlanValidator` **PASS.**

## F5 [A] — FVG demand (supply exists, demand was zero)

- Prompt: FRESH FVGs block header + contract rule **A2c** — a non-empty list with a direction agreeing with the bias demands an fvg_entry OR a one-line reason.
- Compliance: `kernel.FvgDemandWarnings` (WARN-only, never a fail) emitted at write (`trader/auto_trader_planner.go`, `🎯 fvg demand:` line) using the same `FreshFvgCandidates` feed as the prompt; nil-provider guarded.
- **Fixture:** `TestFvgDemandWarnings` — biased-no-author warns; authored passes; against-bias passes; neutral+bias-agnostic warns; empty list passes. **PASS.**

## F6 [XS] — refusal dedup key = verdict class

- `trader/armed_executor.go`: the dedup map now stores the verdict CLASS (`armRefusalClass`: `rr|min_sl|veto|not_armable|other`), so an ATR-drift re-refusal of the same class stays silent while a class change logs.
- **Fixture:** `TestArmRefusalClassIgnoresATRDrift` — the live LONDON S4 pair (18.29×ATR5m → 18.67×ATR5m) classifies identically and the second one does NOT re-log. **PASS.**

---

## GATES

| Gate | Result |
|------|--------|
| `go build ./...` + release binary build | **PASS** |
| `go vet` (trader/kernel/store/provider) | **PASS** |
| Full regression `go test ./...` | **27 packages ok · 0 FAIL** |
| Goldens (`TestFuturesKeyLevelsGolden`, `TestFuturesPlanGolden`, `TestFvgValidateGolden`, `TestVerifyPromptGoldensPasses`, `TestFormatKlineTimeframes_Golden`) | **5/5 PASS** |
| Guide content census (`GuidePage.test.tsx`, 35-knob lint) | **10/10 PASS** |
| New fixtures (F1×3, F2, F3×2, F4, F5, F6) | **all PASS** |

**Changed:** 21 files (+377/−40): kernel 7 + 4 test files, trader 5 + 3 test files, provider accessor, store 1, Guide card 1.

**STOP — awaiting the owner's "go cutover".** Cutover notes for the next dispatch: build from a temp clone at the branch sha (worktree builds lose vcs stamping), flat-gate all origins, boot-line ack, then the owner clicks proximity 0.3 / min-conf 65 in Studio.

---

## POST-CLICK VERIFICATION (2026-08-28, 11:59 CT click)

- **DB (quoted):** `0.3 | 60 | 2026-08-28 11:59:00` (strategy `a5b7662e`) — owner ruling: min_confidence stays **60**; the 60–64 band is judged by Sep-9 data at real n (protection lives in strict + R:R + min-SL + armed, not the conf knob).
- **Seated line (verbatim):** `🗺️ seated 24/191 in-band levels (proximity band ±85pt, 24 of them retained)` and `🗺️ seated 12/170 in-band levels (proximity band ±85pt, 12 of them retained)` — band = 0.3 × DailyRangeProxy(≈283pt) = ±85pt, within the predicted ±91–110pt envelope (the proxy drifted 305→283 as days completed). The C6 "pool ~5–6" figure was the per-plan-doc in-band measure; the seated line counts the full generated universe (170–191 candidates, 12–24 in band, capped downstream by max_levels 12).
- **Resolver on both consumers:** bot gate/watcher `trader/auto_trader_planconfig.go:139` → `kernel.ResolveProximityK` (`kernel/plan_lifecycle.go:25-30`) — live value proven by the ±85pt seated line; engine prompt path `kernel/engine_analysis.go:374` → the SAME `ResolveProximityK` (the old 0.5 floor is gone) — standing evidence: DB 0.3 + clamp test `TestResolveProximityK`; next scheduled planner read (ASIA 16:55 CT) is the fresh engine-path live proof.
- **min_conf = 60 everywhere, zero 65 residue:** gate `kernel/engine_analysis.go:550` + `kernel/engine_position.go:197` + futures prompt `kernel/engine_prompt_futures.go:63` (fallback `store.SafeDefaultMinConfidence = 60`, `store/strategy.go:81`) — nothing clamps or defaults to 65.
- **HTF veto:** `HTF_VETO_MODE=cross` live since 11:32:37 CT boot (`🛡️ htf veto: mode=cross tf=1h`).

## SEP-9 CALIBRATION SCOPE ADDITIONS (owner ruling 2026-08-28)

(a) **Confidence band 60–64 outcomes at full n** — re-judge the min-conf question from real Sep-9 data, not C4's tiny sample (n=3, −$217); the 65 deferral is conditional on this.
(b) **Proximity distance-bucket reaction table** — level touch/react outcomes bucketed by distance-from-price at seat time (0–90 / 90–180 / 180+ pt), sourced from `level_stats` — the 0.3-vs-1.5 question re-judged from data.
