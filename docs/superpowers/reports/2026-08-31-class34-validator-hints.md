# CLASS 34 — Validator hints must name only legal conditions

Date: 2026-08-31 CT · Wave: class34-validator-hints · Branch: `class34-validator-hints` (merged → dev `fef656a4`)

**STATUS: SHIPPED — cutover 2026-09-01 00:43:25 CT, owner GO.**
Live rev now: `fef656a4ee7c45860ad0237f48cef90c6b148d17` (PID 1625428).
Rollback kept: `nofx-bin.prev.boot` = previous live `ebc37e01d7dd…` (class 32, PID 1589096).

Cutover evidence:
- Flat gate pre (00:43 CT): DB open pos 0 · open orders 0 · armed nonterminal 0 · API positions `[]` · API open-orders MNQ `[]` · NT8 snapshots `count=0` ×2 @00:43:00.
- Swap: `mv nofx-bin nofx-bin.prev.boot` → staged binary in (stamp verified on the deployed file: `vcs.revision=fef656a4…`, `modified=false`) → `kill -9 1589096` @00:43:25 → systemd relaunch PID 1625428.
- Boot 00:43:30 quoted: `🔐 BOOT INTEGRITY OK — rev fef656a4ee7c · expected fef656a4ee7c · goldens PASS` · `🗓 session reads … ASIA 16:30 · LONDON 01:30 · NY 08:00` · `🔬 conditions: live [7] · shadow [breakout_retest, fvg_entry]` · **`🧪 validator hints: 6 sites — every condition token legal + live (class 34 guard)`** · `📜 scenario schema: 9 conditions`.
- Post-boot: 0 `[ERRO]`/panic, equity 52216.00 preserved, API positions `[]`, open-orders MNQ `[]`.
- LIVE PROOF pending: the next natural validator reject during a read will carry the new `\`reject\`` hint + `Valid conditions: [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim]` — quote it when it fires (and today ~16:35 CT the class-32 halt-fire timestamp).

Staged:
- Build sha `fef656a4ee7c45860ad0237f48cef90c6b148d17` — clean-clone build, `vcs.revision` matches, `vcs.modified=false`. Binary at `~/nofx-staged/nofx-34-bin`.
- Marker `d03db52a` pushed: `deploy/RELEASE` = `fef656a4…` + `GUIDE_BUILT_REV` = `fef656a4…`.
- Suites: `go build ./...` OK · `go test ./...` green · goldens PASS · tsc 0 · vitest 36 files / 292 tests.
- Flat gate pre-park (23:52 CT): DB open pos 0 · open orders 0 · armed nonterminal 0 · API positions `[]` · API open-orders MNQ `[]` · NT8 snapshots `count=0` ×2 @23:52:29.
- Live bot untouched: rev `ebc37e01d7dd…` PID 1589096 (class 32 live since 23:40:22), window check passed.

Cutover runbook (on owner GO):
1. `mv ~/nofx/nofx-bin ~/nofx/nofx-bin.prev.boot` → `cp ~/nofx-staged/nofx-34-bin ~/nofx/nofx-bin`
2. `kill -9 <PID>` (SIGTERM exits 0 — no relaunch)
3. Boot checklist within 90s: rev `fef656a4` · `🔐 BOOT INTEGRITY OK` · goldens PASS · `🧪 validator hints: 6 sites — every condition token legal + live (class 34 guard)` · `🗓 session reads` + `🔬 conditions` lines
4. Post-boot flat-gate re-quote (four legs).
5. LIVE PROOF: the next natural reject during a read will carry the new hint + `Valid conditions: [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim]` — quote it when it fires (and tomorrow ~16:35 CT the class-32 halt-fire timestamp).

Rollback: `mv nofx-bin.prev.boot nofx-bin && <revert deploy/RELEASE> && kill -9 <PID>`
(`nofx-bin.prev.boot` will hold the pre-wave live `ebc37e01d7dd…`, class 32.)

## The bug (tonight's evidence)

Both ASIA chains failed identically:
- Reject: `S3 breakdown_continue: a close came back across 29517.00 — the breakdown is void; author a reject/retest play instead`
- Model authored condition `reject_retest`
- Parse/schema rejected: `scenario[2].condition "reject_retest" invalid`
- Repeated identically in the reset chain → both chains FAIL-CLOSED. The model complied with the hint and was punished for it.
- Compounding: `breakout_retest` is SHADOWED (0C), so the hint steered toward either a nonexistent or a demoted condition.

## Before/after of every changed message

| Site | Before | After |
|---|---|---|
| `kernel/breakdown_continue.go` reclaimed reject | `… the breakdown is void; author a reject/retest play instead` | `… the breakdown is void; author a \`reject\` play instead (do NOT combine condition names; \`reject_retest\` is not a valid condition)` |
| `kernel/breakdown_continue.go` displacement reject | `… not a displacement move, author a normal reject/retest play instead` | `… not a displacement move, author a normal \`reject\` play instead (do NOT combine condition names; \`reject_retest\` is not a valid condition)` |
| `kernel/planner_repair.go` BREAKDOWN-CONTINUE LAW | `… author a reject/retest play instead of breakdown_continue.` | `… author a \`reject\` play instead of breakdown_continue (do NOT combine condition names; \`reject_retest\` is not a valid condition).` |
| `kernel/planner_repair.go` ARM-SPLIT LAW / ENTRY-LAW CONFIRM LAW | inline literals | registry constants `RepairArmSplitLaw` / `RepairEntryConfirmLaw` (text unchanged — they already name only `sweep_reclaim` / `breakdown_continue`) |
| `kernel/plan_doc.go` arm-legs reject | inline `(the split entry is the sweep_reclaim contract; …)` | registry constant `ArmLegsSplitContract` (text unchanged — names only `sweep_reclaim`) |
| `trader/auto_trader_planner.go` reject block | `Fix ONLY this defect, keep the rest structurally identical.` | now also appends `\nValid conditions: [<resolved live list>] (use exactly ONE token from this list; do NOT combine condition names).` |

## The guard

- `kernel.ValidatorHints()` — registry of all 6 hint sites, each declaring the
  condition tokens its text names.
- `kernel.ValidateValidatorHints()` — every token must be in `KnownConditions()`
  AND resolve LIVE under defaults. **The table test IS the guard**
  (`kernel/validator_hints_test.go:TestValidatorHintsNameOnlyLegalLiveConditions`),
  re-run at boot (`kernel/levels_volume_boot.go` logs
  `🧪 validator hints: 6 sites — every condition token legal + live (class 34 guard)`
  or an ERROR if broken).
- The resolved live list the guard checks against (defaults, fixture-verified):
  `[acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim]`
  (the 9-enum minus the 0C shadowed `fvg_entry`, `breakout_retest`).
- The reject block's `Valid conditions:` suffix is computed from the SAME
  resolver with the trader's base + session override maps + env
  (`kernel.ResolvedLiveConditions`) — class-8 style: resolved, never literal.

## Tests (all green)

- `TestValidatorHintsNameOnlyLegalLiveConditions` — the table guard.
- `TestBreakdownHintsNameRejectNotComposite` — tonight's reproduction at the
  hint level: names `` `reject` ``, forbids the composite, never says
  "author a reject/retest".
- `TestResolvedLiveConditionsExcludesShadowed` — live vocabulary sorted,
  shadowed excluded.
- `TestLiveConditionsLine` — the suffix rendering.
- `TestClass34RejectBlockNamesOnlyLiveConditions` — end-to-end reproduction:
  breakdown-void reject → verbatim reason kept + suffix lists only legal live
  tokens.
- `TestClass34RepairLawsNameOnlyLegalLiveConditions` — repair excerpts carry no
  composite instruction; registry validates.
- Goldens PASS · `go build ./...` OK · `go test ./...` green · tsc 0 · vitest
  36 files / 292 tests.

## Stop-lines honored

No enum changes · no new conditions · no validator LOGIC changes — message text
plus a guard only. The planner's output-contract schema still lists all 9
conditions (0C: authoring shadowed conditions is allowed; the ARM seam refuses
them), and the prompt contract was untouched.

## Checklist

Class 34 appended (class 33 is unoccupied — quoted the current highest before
appending: 32).
