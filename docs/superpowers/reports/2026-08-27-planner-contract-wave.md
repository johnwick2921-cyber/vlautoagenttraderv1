# PLANNER CONTRACT WAVE — S/D+FVG playbook + UI fixes (2026-08-27)

Branch `feat/planner-playbook` · base: merged dev `b02461cf` (== running `6b5d17ca`, boot quoted 16:47:20) · cutover rev `e49c82e503` · deployed 17:45 CT inside the 16:00–17:00 halt window (trader runs ASIA+NY; deploy verified flat via `GET /api/positions` → `[]`).

**Advisory law holds:** every new prompt element is WARN-only. Zero new hard gates. `go test ./...` green, FE tsc 0 errors, vitest 261/261, build green.

## PACK A — planner prompt contract (A1–A5) — `41ee7c95`

- **A1 BIAS-TREE.** `RenderBiasTree` renders a machine-computed branch table (PDH/PDL/PDC, value-area position, premium/discount %, nearest liquidity/draw) with 6 branches; the output contract requires `reasoning` to open by naming the branch taken (`"bias-tree: …"`). Replaces free-form bias guidance. Facts only — the AI judges direction.
- **A2 THE CHAIN.** `PlanScenario.chain_after` (optional, links an `fvg_entry` to its `sweep_reclaim` precursor) + `ChainWarnings` validator: WARN at write when an `fvg_entry` has no sweep-reclaim precursor AND its origin is not a fresh A/B zone. Playbook text: entry_mode=ce default, edge only for A-grade HTF-confluent origins, stop beyond the sweep extreme, T1 = first opposing pool, runner = the draw. Cites the raw-FVG null (40k sample).
- **A3 NO-TRADE GATES** (≤8 lines, line-count pinned in test): balance-day → edges-only/skip · opening gap >1.2×ATR or outside-range open → never fade · no A/B zone in reach + no pool swept by 10:30 ET → skip day · lunch 11:30–13:30 ET no entries · Tier-1 news stand aside.
- **A4 KILLZONE WEIGHTING**: NY AM 8:30–11:00 ET primary, 10:00–11:00 premium-FVG window, conviction down Monday / up Thursday–Friday.
- **A5 STOP-DOING**: acceptance requires a prior sweep+displacement or skip (0% win evidence cited).
- Wiring: `BiasCtxFacts` fills from `ComputeBiasContext`; chain warnings log at plan-write next to the existing role-mismatch warns.
- Boot line (cutover): `📜 planner playbook (2026-08-26): playbook=v2 bias_tree=on chain_after=on no_trade_gates=on killzone_weights=on stop_doing=on — ALL ADVISORY, zero new gates`.
- Tests (`kernel/planner_playbook_test.go`): playbook-section render, no-trade-gates ≤8 lines, bias-tree branch flags (branch 1/3, premium, discount), chain-warning matrix (bare C-origin warns; `chain_after` or A-origin silent).

## PACK B — MPM look-ahead rule — `227dc35a`

- `docs/superpowers/reports/2026-08-26-bar-persistence.md` §10 REFUSAL AUTOPSY block now carries **B1**: every hypothetical stop/target in Sep-3 replay + refusal-autopsy tooling MUST resolve on 1m bars, never the 2-min confirm bar (the 73%→50% edge degradation was exactly this coarsening).
- Fixture `scripts/mpm_resolution_fixture.py` proves the inversion: TP+SL inside one 2-min bucket flips a WIN (1m: TP @m8) into a LOSS (2-min SL-first lookup). Runs green: `1m TP first · 2-min SL first · fixture OK`.

## PACK C — README §9 discrepancy fixes (one commit each)

| §9 row | Fix | Commit |
|---|---|---|
| 1 dead TRANSITION chip | removed chip + FE type (backend `transitionState` stays uncalled — documented) | `2c8bff00` |
| 2 vestigial `last_entry_ct`/`eod_flat_ct` | controls hidden, stored values untouched; derived warning removed; tests updated to assert absence | `e02d4079` |
| 3 proximity slider 0.5 floor | 0.5→0.1 to match resolver (0.1–3.0) | `158ba186` |
| 4 static SYSTEM_STATUS strip | live poll of `/api/health` every 30s; status colored, running REV shown | `fdf59115` |
| 5 gate-blocks cross-trader fetch | `?trader_id=` filter added server-side + FE passes it | `0afcc728` |
| 6 scenario_meta missing `confirm` | type gains confirm map; loose cast fixed | `62c9f55b` |
| 7 quality-A mislabel + neutral=short-red | A/A+ → `--vl-grade-a`, B→grade-b, C→grade-c; neutral direction → muted | `0b2bae95` |
| 8 role badges / fresh·xN / ◆ chips | **QUEUE** — feature work, not a bug | — |
| 9 dead `trigger_reason`/`overlay_count` types + unused i18n keys | removed (PlanToday only; PlanVersionItem's trigger_reason stays — it IS rendered) | `289432f4` |
| 10 no 402 banner | dashboard polls `/api/risk/errors`; `ai_payment_402` row latches a red CREDIT EXHAUSTED banner, auto-clears on first success (matches backend P0 auto-ack) | `8f7f6d14` |
| 11 consumed dim vs strike | **QUEUE** — dimming is the documented design; strike = conflict-ghost | — |
| 12 Approve no-modal | **QUEUE** — by design (W9 comment) | — |

## E1 — first plan on the playbook prompt

**ATTEMPTED, NOT YET LANDED — fail-closed on a pre-existing facts rule (not a playbook regression).**

- 17:46:20 CT — owner reset fired (`dayplan_reset:8d5c8af5…:2026-08-26:ASIA = 2`; chain abandoned at v1, budget re-armed 4). Fresh read ran on the new playbook prompt.
- The read consumed its 3 attempts over ~15 min (two attempts passed parse + auto-collapse, 17:51:26 / 17:56:25, then tripped a validator) and **FAIL-CLOSED at 18:01:15 CT**: `only 2 levels above price 29614.00 — the plan must carry ≥3 on EACH side`.
- ASIA now sits out under a NO-TRADE v2 (`trigger: planner_fail_closed`, `ENTIRE SESSION — planner fail-closed`) — the documented safe outcome: never stale, never uncalibrated.
- **Root cause:** the ≥3-levels-per-side facts validator (P0.1/P0.2, 2026-08-19 — pre-existing, unchanged by this wave). Price 29614.00 sits near the TOP of the seated level stack, so the machine map itself only offered 2 levels above; every AI draft inherited that and was rejected 3×.
- **Not a playbook regression:** A1–A5 are advisory-only; no validator was added or tightened in this wave; the validator that fired predates the branch (`git log` shows it landed 2026-08-19).
- **Next evidence window:** the NY scheduled read (08:25 CT 2026-08-27) — the overnight map will have regenerated levels above. The BIAS-TREE branch quote + chained scenario pair will be verified from that plan (or an earlier level-wake re-read if one fires).

## E2 — render tests

See PACK A tests: BIAS-TREE branch line, THE CHAIN playbook, no-trade gates block, killzone line, STOP-DOING line, chain_after in the JSON contract, and the chain-warning matrix. All green pre-cutover.

## E3 — balance-day / skip-day declaration

**N/A tonight, reported honestly.** The ASIA plan that exists is the fail-closed NO-TRADE (`ENTIRE SESSION — planner fail-closed`) — a machine safety sit-out, NOT an AI balance-day/skip-day declaration. No AI-written no_trade gate (balance-day / outside-range / Tier-1 stand-aside) was produced to quote. Checked at the NY read tomorrow alongside E1.

## Regression

- `go test ./...` EXIT 0 (full) · `go build ./...` clean.
- FE: `tsc --noEmit` 0 errors · vitest 32 files / 261 tests PASS · `npm run build` green.
- Goldens untouched (planner prompt is not golden-covered; futures executor goldens unaffected).
- Watcher fix `a473b93f` (S1) rode this cutover — previously deployed binaries still had the hardcoded 1.5 proximity literal.

## Cutover

- Flat check: `GET /api/positions` → `[]`.
- `e49c82e503` built with `main.buildRevision`, `deploy/RELEASE` marker `c640feb8`, SIGKILL restart (systemd `Restart=on-failure`), one PID, boot ledger 17:45:27 quotes the playbook line, health reports `revision e49c82e50343`.
- FE: served by the Vite server on :3000 (node, live) — C1–C9 are live immediately; `npm run build` green for any static deploy later.

## Owner queue (not in this wave)

- §9 rows #8, #11, #12 (above).
- Sep-3 deep verification: refusal autopsy + replay now hard-pinned to 1m resolution (B1).
