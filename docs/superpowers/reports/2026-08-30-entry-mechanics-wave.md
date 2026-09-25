# Entry-Mechanics Full Wave — E1–E9 + acceptance-rule ADDENDUM (2026-08-30)

**Branch:** `feat/entry-mechanics` (off `dev` @ `a9aa9a04`; worktree `~/nofx-entry`, LOCKED — single active dispatch)
**Mode:** full fix (owner override — not staged). NO code deployed. Parked for the owner's "go cutover" per the D-rule.
**Scope:** E1 15m removal · E2 per-condition entry law · E3 breakdown floor relax · E4 sweep-reclaim split entry · E5 1m-MSS · E6 time_hold · E7 stop-entry orders · E8 shadow A/B logger · E9 guide+UI+knobs · ADDENDUM acceptance-rule migration (knob census 39a0481e).

---

## E1 — 15m confirm variant: DEAD

The enum chokepoint (`kernel/plan_doc.go` `confirmRules` / `conditionRules` / `NormalizeConfirmRule` / `NormalizeConditionRule`) no longer contains any 15m variant. New authorship of any 15m spelling is REJECTED BY NAME:

- `confirm.rule "15m_close" → scenario[i].confirm.rule "15m_close" — confirm_rule_15m_removed (…)`
- `death/flip.rule "15m_close" → … — condition_rule_15m_removed (…)`

Vocabulary after: **`touch | 1x5m_close | 2x5m_close | 1m_mss | time_hold`** (death/flip: `2x5m | 5m_close`).

Repo-wide grep proof — 21 remaining `15m_close|1x15m` hits, ALL deliberate:
- legacy EVALUATION tolerance so stored pre-entry-mechanics docs keep evaluating (`plan_confirm.go:25`, `plan_lifecycle.go:209`, `scenario_facts.go` acceptance need/TF cases — each commented `// legacy: stored docs only`);
- rejection-path fixtures (`entry_law_test.go`, `plan_doc_rule_alias_test.go`, `plan_condition_rule_test.go` — they assert the named rejection / pass-through).

Legacy stored docs still parse: fixture `TestLegacy15mConfirmStillEvaluates` (evaluates MET on the legacy acceptance path) + `TestEntryLawLegacyDocsStillParse` (unmarshal + base-path contract). New authorship fixture: `TestConfirmValidator` asserts the NAMED `confirm_rule_15m_removed` message.

## E2 — per-condition ENTRY LAW (one table, one chokepoint)

`kernel/entry_law.go` — `entryLaw` map, enum-keyed by condition; `ValidateEntryLaw` runs inside `ValidatePlanDocWithCaps` (the schema chokepoint). Named rejections:

| condition | allowed confirm rules | rejection name |
|---|---|---|
| reject / fvg_entry | touch ONLY | `fade_requires_touch` (any close-confirm); stop ≥2 ticks beyond the level (arm bracket checked) |
| sweep_reclaim | leg-1 touch · leg-2 `1m_mss` (1x5m alt) | `sweep_leg1_requires_touch` / `sweep_leg2_requires_mss_or_1x5m` |
| reclaim | 1x5m_close · 1m_mss | `2x5m_reserved` |
| breakout_retest | touch · 1x5m_close | `2x5m_reserved` |
| acceptance / hold | time_hold · 1x5m_close | `2x5m_reserved` |
| breakdown/breakup_continue | 1x5m_close · 2x5m_close (**2x5m legal ONLY here**) | — |

Fixtures: **18 twins** (one accept + one reject per condition, `TestEntryLawPerCondition`) + split-contract + fade-stop-law + legacy-parse. **The dress-rehearsal S4 case STILL rejects** — `TestRehearsalS4CaseStillRejects` replays the S4 shape (breakdown_continue 29437 pullback, the flip leg that reclaimed) and quotes the rejection: `S4 breakdown_continue: a close came back across 29437.00 — the breakdown is void; author a reject/retest play instead` (reclaimed=true — the new law cannot resurrect it; the validator now checks Reclaimed FIRST).

## E3 — breakdown floor relax (2→1)

`bdConfirmCloses` now resolves **`BD_MIN_CLOSES` (default 1)** (the double close is gone); displacement (`BD_MIN_DISP_ATR`) + reclaim-check UNCHANGED; reject-message text updated (`…NO confirming close beyond %.2f yet (%d confirming close(s) needed — BD_MIN_CLOSES…)`). Prompt playbook + `retestLegDetail` reworded to the confirming-close trigger. Fixtures: **1-close+disp PASS** (`TestBreakdownImmediateAuthorableBeforeSecondClose` — pullback now validates on ONE beyond close with displacement), **weak-disp REJECT** (unchanged), **reclaimed REJECT** (now fires the reclaimed message first).

## E4 — sweep-reclaim SPLIT entry (money path)

`PlanArmSpec.Legs []PlanArmLeg` (entry/stop/target/size/wait_confirm/rule/kind); `ArmSpecValid` enforces the split contract (sweep_reclaim only — `arm_legs_sweep_reclaim_only`; exactly 2 legs; leg-1 touch rests AT the sweep ref; leg-2 chains on confirm2 with `1m_mss|1x5m_close`). Ledger: `armed_orders` gains `leg_index`/`leg_count`/`kind` (DDL + idempotent ALTER migration for live DBs); two child rows share plan lineage, independent OCO brackets. **EITHER leg's stop-out cancels the sibling's unfilled order** — `splitSiblingCancelDecision` (pure: stop-out→cancel, target-out/open→nothing) + `cancelSplitSiblingOnStopOut` on the existing cancel machinery. Session-end/news/dormant cancel paths cover BOTH legs by construction (cancel-all). Fixtures: `TestArmSpecSplitContractValidation`, `TestSplitSiblingStopOutCancelsUnfilledLeg`, `TestSplitArmWritesTwoLedgerRows` + `TestSplitArmSessionEndCancelsBothLegs` (session-gated, run green during live sessions — the repo's established skip convention).

## E5 — 1m-MSS confirm primitive

`kernel/mss.go` — last CONFIRMED 1m fractal swing (k=2, strict, closed bars) broken by a 1m CLOSE beyond it with displacement ≥ **`MSS_MIN_DISP_ATR` (0.5) × ATR5m** on the breaking bar. Wicks NEVER count. Direction-aware (`above`→swing high, `below`→swing low). Renders `1m-MSS: MET/NOT-MET (swing <px> @<t>)`. Fixtures: twin bull/bear MET + wick-never + weak-disp + no-swing + **R2 independent swing recompute** (`TestMSSR2IndependentSwingRecompute` recomputes the fractal with its own loop) + in-pipeline routing.

## E6 — time_hold acceptance rule

`EvaluateConfirm` case `time_hold`: price holds beyond ref for **`ACCEPT_HOLD_MIN` (10)** minutes of 1m closes, no close back across (run resets). Fixtures: MET at 12/10 · not-MET at 9 · close-back resets (6/10) · closed-bars-only boundary · in-entry-law.

## E7 — stop-entry orders (Go + C#) — THE LONG POLE, park-gated

- Wire: `SignalPayload.StopPrice` + `OrderType "stop_entry"` (additive JSON — old AddOn ignores unknown fields; the GO side therefore never sends a stop_entry frame unless **`STOP_ENTRY_SEAM=on`**).
- Go: `TCPTrader.PlaceStopEntry` (stop-market + bracket-on-fill contract, identity stamp, B3 guard). Executor: `runArmedPlacement` case `Kind=="stop_entry"` — seam gate + **`RETEST_WAIT_BARS` (6)** no-retest fallback window (`stopEntryFallbackDue`, pure) + **`STOP_ENTRY_OFFSET_TICKS` (2)** trigger offset beyond the level. Used by the breakout_retest fallback and as the breakdown immediate alternative.
- C#: `HandleSignal` parses `stop_entry` + `stop_price` → `OrderType.StopMarket` with bracket-on-fill identical to limits; cancelable via `workingEntries`.
- **Far-side frame proof (fixture-replay):** `TestPlaceStopEntryFrameOnLoopback` (twin long/short) receives the frame on the loopback TCP server — `order_type=stop_entry`, `stop_price`, bracket, identity stamp. `TestStopEntryFrameIsAdditiveJSON` proves the old-limit frame shape carries no new fields.
- **Binnie lockstep note:** the C# change needs the NT8 copy → F5 → full restart dance before the seam is flipped ON. Per the D-rule this is cutover #2 and NEVER ships at night.

## E8 — shadow A/B counterfactual logger (Sep-9's courtroom)

`kernel/shadow_ab.go` (pure `ShadowABForScenario`) + `store/ab_confirm.go` (`ab_confirm_log` table, unique per plan:version:scenario:rule) + the executor's `logShadowAB` (once per plan version). 4 counterfactual fills (touch · 1x5m_close · 2x5m_close · 1m_mss): fill px, MFE, MAE, target/stop outcome (intrabar ambiguity AGAINST the trade, R9), time-to-fill. **ZERO real-path effect**: `TestShadowABZeroRealEffect` (pure, deterministic, mutates nothing) + `TestLogShadowABWritesOnlyItsOwnTable` (writes ONLY `ab_confirm_log`; the armed ledger stays untouched; upserts idempotent).

## E9 — guide + UI + knobs

Guide `plays.ts`: every condition card reworded to the new law + NEW sections — THE ENTRY LAW (per-condition table with the named rejections), The new confirm rules (1m-MSS / time_hold / stop_entry), Entry-mechanics knobs (BD_MIN_CLOSES, MSS_MIN_DISP_ATR, ACCEPT_HOLD_MIN, STOP_ENTRY_OFFSET_TICKS, RETEST_WAIT_BARS, STOP_ENTRY_SEAM). Plan card: `ConfirmChip` renders the new rule names (`1×5m` / `2×5m` / `1m-MSS` / `time-hold`); `ArmedChip` + `armedMapFor` show **split-leg badges** (`L1⏳ L2📌`). `GUIDE_BUILT_REV` → `86fc81d9` (re-bumped to the merge sha by the cutover marker per canon). FE acceptance-rule selector: `2×5m`/`15m` options → **`1×5m`** (`5m_close`).

## ADDENDUM — acceptance-rule migration (knob census 39a0481e)

The census found the DB steering the OLD rule — quoted from the live DB (read-only), bound strategy `a5b7662e`:

- BEFORE: `day_plan.acceptance_rule = "2x5m"` · per-session `NY|ASIA|LONDON = "2x5m"`
- AFTER (boot repair): `"5m_close"` everywhere (strategy-level + sessions).

Shipped: `store.RepairAcceptanceRuleMigration` (raw-JSON patch, idempotent, handles both `day_plan` and `ai_config.day_plan` shapes) wired at boot BEFORE `LoadTradersFromStore` (boot line `🩹 acceptance-rule migration: strategy-level=N session=M (2x5m → 5m_close)`); `DefaultAcceptanceRule` → `"5m_close"`; `AcceptanceRuleFor` SELF-HEALS any stored old-vocabulary string at read (defense in depth — a straggler row can never steer the prompt back to the reserved double close). Fixtures: `TestRepairAcceptanceRuleMigration` (1+3 migrated, idempotent) + `TestAcceptanceRuleForSelfHeals`. W15 resolver fixtures updated to the new law; the W7 burned-level fixture made deterministic (closes AT the level under the one-close default).

## GATES (all green)

- `go build ./...` ✓ · `go test ./...` **27/27 packages ok, 0 FAIL** (goldens included)
- `vitest run` **283/283** (guide 10/10) · `tsc --noEmit` 0 errors
- Every fixture named + quoted above; repo-wide 15m grep proof (21 deliberate survivors, all annotated); E8 zero-real-effect diff; wire backward-compat proof (additive JSON).

## D-RULE — deploy sequencing (the only staging left)

Everything is parked as ONE branch (`feat/entry-mechanics`). Cutover decision:

1. **E1–E6 + E8–E9 gates are GREEN; E7 (C#) is built but its far-side frames are unproven on the live AddOn** (loopback-proven only; the owner must NT8-copy → F5 → full restart for the C# half). Per the D-rule: **cutover the Go-only wave FIRST** (single boot — everything except the stop-entry seam; the seam stays OFF so no stop_entry frame can reach an old AddOn). **E7 follows as its own cutover** once the far-side frames prove (Binnie lockstep). Two boots max.
2. **NO cutover between 16:45–17:10 CT** (weekly read + ASIA window sacred). First safe window after tonight's first plan writes, owner-attended.

**STOP — awaiting the owner's "go cutover".** Branch tip: `9a7f605b` (guide rev bump) on top of `86fc81d9`.

---

# CUTOVER RECORD (owner "GO CUTOVER", 2026-08-30 ~17:10 CT) — Go-only, STOP_ENTRY_SEAM=off

**Build commit:** `9ca53e87` (E4 split-index fix + boot ledgers folded in before the cutover — the live-session window exposed a real E4 bug: the legacy 2-column unique index rejected the second leg row; replaced with `(plan_id, scenario, leg_index)`). Binary: `vcs.revision=9ca53e873a1b · vcs.modified=false`, md5 `dd6fd8f5`.

**Debut proofs on the CURRENT binary (captured BEFORE the swap, 23243670):**
- `📅 WEEKLY READ starting for week 2026-08-31 (boot_backfill=false)` 17:01:18 → `📅 WEEKLY READ written 2026-08-31 v1 bias=bear conviction=low draw=PWL@28947.75 invalid=29535.00 thin=true facts_hash=b8591ad6c492…` 17:07:15 → `skip-fresh … idempotent` + `📅 WEEKLY INVALIDATED bear @ 29535.00 (1h, v2) — bias→neutral, no auto-flip` 17:07:18 (the invalidation machinery fired live on the same Sunday).
- `🗓️ PLAN written 2026-08-30 ASIA v1 (model deepseek-v4-pro, lifecycle active, prompt 30cebc96fe8b, ai_config a28d83f159084145)` 17:08:56 — the ## Candles + ## Weekly Context prompt headers were verified at dress rehearsal (e1_prompt_rendered.txt); the stored doc + write are the live pipeline proof.
- **Watchdog:** 0 alarm lines all day (silent = healthy). **Flood:** every reopen minute `intrabar_dropped=0 current_dropped=0 historical_dropped=0 peak_depth=0/4096`.
- Note: `📅 calendar FAIL-CLOSED: no slice for 2026-08-30 — using the static T1 fallback (8 window(s))` — fail-closed to static ✓.

**Flat-gate (all-origin, split-aware):** DB OPEN positions = 0 · DB non-terminal armed = 0 (zero arms ⇒ zero legs, no confirm mid-track) · API positions `[]` (authed) · NT8 snapshots Sim101 count=0 + SimAccount1 count=0.

**Swap:** `mv nofx-bin nofx-bin.prev` (rollback `36c0c681` kept in `~/nofx-backups/cutover-entry/`) → `cp nofx-bin.next` → `kill -9 482741` 17:10:42 → systemd relaunch PID 726053 17:10:47.

**Boot checklist (all quoted):**
- `🔐 BOOT INTEGRITY OK — rev 9ca53e873a1b · built 2026-08-30T22:09:49Z · expected 9ca53e873a1b · goldens PASS`
- `🩹 acceptance-rule migration: strategy-level=9 session=3 (2x5m → 5m_close)` — counts pre-computed from the live DB (9 base + 3 NY/ASIA/LONDON) ✓
- `📜 scenario schema: 9 conditions […]` · `🔐 confirm rules: 5 [1m_mss, 1x5m_close, 2x5m_close, time_hold, touch]` · `🎛 entry law: bd_min_closes=1 bd_min_disp_atr=1.00 mss_min_disp_atr=0.50 accept_hold_min=10 stop_entry_offset_ticks=2 retest_wait_bars=6 stop_entry_seam=off`
- **Weekly doc SURVIVES:** `📅 WEEKLY READ skip-fresh — week 2026-08-31 doc already stored (v2), idempotent.` at 17:10:47 on the NEW binary — no re-read ✓
- **ASIA plan resumes lifecycle:** active row `2026-08-30:ASIA` v1 served; first cycle ran (balance frames recovered 17:11:17, equity 52113.50; `level_stats: 2026-08-29 giving up after 4 attempts` — the known Saturday-no-bars case, real eval Monday 17:05).
- Live-DB migrations applied: `armed_orders` gained `leg_index/leg_count/kind` + the 3-column unique index; 0 strategies still storing `2x5m`.
- Transient `no_balance_frame` WARN at boot cleared when the NT8 account_balance frame arrived (same class as prior cutovers).

**Post-cutover watch:** 15-minute watchdog/flood watch clean (0 critical errors, 0 dropped closes).

---

# PANIC + ROLLBACK (17:12:47 CT) — and the fix

Two minutes after the clean boot, the 15-min watch tripped: **`panic: runtime error: index out of range [5] with length 4`** in `kernel.ShadowABForScenario` (shadow_ab.go:122) via `logShadowAB ← maybeManageArmedOrders ← runCycle`. Root cause: the E8 close-rule fill mapped the 5m bucket back to the 1m bar with `bucket_index × 5` — wrong when the plan window starts mid-bucket or spans <5 bars (the ASIA v1 window was 4 bars crossing a 5m boundary). A report-only path took the trading loop down.

**Rollback (tested, 17:14:49):** restored `nofx-bin.prev` (rev 23243670, md5 36c0c681) + `deploy/RELEASE=23243670af35` → boot 17:14:53 PID 728177 `🔐 BOOT INTEGRITY OK — rev 23243670af35 · goldens PASS`. The bot was flat the whole time; ASIA v1 + weekly v2 rows intact. The DB migrations from the new binary (armed_orders leg columns + 3-col index, acceptance_rule 5m_close, ab_confirm_log) are compatible with the old binary — 15+ min clean on the rollback.

**Fix (dev `cd1d1de3`, marker v3 `4509ca9f`):** bucket→bar mapping by OpenTime (`barIdxForBucket`) + `recover()` at the `logShadowAB` seam (a report-only path may degrade to a warning, never a panic) + regression fixture `TestShadowABWindowCrossingFiveMBoundary` reproducing the exact 4-bar boundary-crossing shape (would have panicked the OLD code) + AUDIT-CHECKLIST class 23 appended. Full gates: go 27/27 ok. Binary rebuilt: `vcs.revision=cd1d1de3 · modified=false`, md5 `676c3b18` — **attempt #2 is one swap away, awaiting the owner's re-ack.**

**E7 remains PARKED** for its own daytime cutover: NT8 copy → F5 → full restart → far-side stop_entry frames prove → then `STOP_ENTRY_SEAM=on`. Until then the seam is OFF and the Go side cannot emit a stop_entry frame.

**Next-session live E-proof (pre-registered):** the NEXT authored plan (LONDON/NY) runs under the new law — expect touch-rules on fade scenarios (reject/fvg_entry), zero 2x5m outside waterfall-class; any violation is named in the validator reject log.
