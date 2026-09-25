# POST-SOAK BUILD PLAN — fix everything (2026-08-22)

Master sequencing for the full remaining backlog (PRs #64–#68). Standing rules
hold: SIM-only · flat-window deploys · zero calibration changes before the
Sunday soak + replays · one commit per fix · additive-only.

## GATE 0 — Sunday 2026-08-23 (no code, blocks Wave 3+)

1. **16:55 CT — E4 soak** (`e4-soak.timer`, armed) → verdict filed from
   `~/soak-g7/e4.log` / `e4.table`. AI credits topped up (owner, 08-22).
2. **08-22 E2 replay** — 3-variant G6 table (A shipped 4/60 pnl<0 · B research
   3/30 · C ≤0-reset), Σ delta both days (A=0.00, B=0.00, C=−88.50 pre-computed).
3. **Calibration pass — 7 owner decisions:**
   swing window (k=2 vs 10–20) · MSS FVG keep/drop · veto TF (1h vs 15m) ·
   min-conf (60 vs 65) · trail mult (2.0 vs 1.5) · loss-streak (4/60 vs 3/30) ·
   target-R:R conflict (v5 2.5× vs FULL-SPEC ≥3).
4. **Merge PRs #64–#68** as rulings land.

## WAVE 1 — deploy what's already merged-ready (Sunday night flat window)

| Item | Action |
|---|---|
| PR #67 DeepSeek thinking=enabled + effort=max | merge → build → boot-quote cutover |
| Partner repo `5a1d12ef` | owner pushes |

## WAVE 2 — clarity quick-wins (S/XS, one branch, one PR, ~1 day)

| # | Item | Anchor |
|---|---|---|
| 2.1 | PlanHeader **Approve** button (endpoint exists) | `web/.../plan/*` + `/api/plan/approve` |
| 2.2 | Running **rev in UI/API** | `api/server.go:582` handleHealth + Settings |
| 2.3 | **Machine grade** beside model grade | `ZoneTable.tsx` + `levels_score.go` |
| 2.4 | **"No data" line** in prompt when bars missing | `engine_prompt_futures.go` |
| 2.5 | **NT8 venue badge** on futures strategies | `CoinSourceEditor.tsx` |
| 2.6 | **Grid disabled + explainer** on futures | `StrategyStudioPage.tsx` |
| 2.7 | **Duplicate-to-edit hint** on locked default | `StrategyStudioPage.tsx` |
| 2.8 | **London DST-drift warning** (XS) | registry/session UI |
| 2.9 | favicon + rebrand L3 leftovers (XS) | `web/index.html`, i18n |
| 2.10 | grid_* store: wire a writer or remove | `store/grid.go` |
| 2.11 | Rate-breaker/order-dedup **bypass fixtures** | trader tests |

## WAVE 3 — risk/trail hardening (post-calibration; behavior-changing, so gated)

| # | Item | Depends on |
|---|---|---|
| 3.1 | **Trail lock step** +1.5R → entry+0.25R | trail-mult ruling (wave 2 of GATE 0.3) |
| 3.2 | **max_margin_usage**: enforce or relabel | none (clarity) |
| 3.3 | **Graduated DD ladder** 75/90 | guardrails-master ruling (respects OFF = soft-audit) |
| 3.4 | **FOMC/NFP forced-close T-2min** | none (T1 machinery exists) |
| 3.5 | Apply any ruled **calibration changes** (swing/veto/loss-streak/etc.) | GATE 0.3 |

## WAVE 4 — medium builds (each its own PR)

| # | Item | Notes |
|---|---|---|
| 4.1 | **SessionsAccordion + ⚪/🔸 chips** | backend overrides exist; UI-only |
| 4.2 | **Exit-fill persistence** NT8 SIM path | lineage; C# AddOn lockstep review |
| 4.3 | **Multi-session flats** | dormant until ASIA/LONDON enabled — build now, activates on enable |
| 4.4 | **MAE/MFE viz** (SHELF item) | DisciplinePanel/digest |
| 4.5 | **"API auto max"** — per-model reasoning_effort via API | model_config surface + mcp builder hook |

## WAVE 5 — owner product decisions (schedule when asked)

| Item | Decision |
|---|---|
| ML/Qlib pipeline (35 models, RD-Agent) | build / never (superseded by LLM-plan pivot) |
| HandoverBanner | restore the lifecycle writer or strike from spec |
| Prop-firm guardrail templates | only if a funded account ever comes |

## Invariants (every wave)

- Full regression before every deploy: `go test ./...` + vitest 263 + `npm run build`
- Boot line quoted; RELEASE written after build; flat window; one PID
- Calibration changes = env/Studio knobs with `.env.example` docs + ledger line
- Partner mirror: `format-patch → am` (owner pushes) for any shared Go change

## End state

All findings closed or explicitly deferred → the next MASTER AUDIT triggers on
first incident / metric anomaly / gate fire after the soak (per PR #66 verdict).
