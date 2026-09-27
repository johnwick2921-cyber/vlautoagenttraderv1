# VL — Day Plan settings-not-editable fix + indicators verify

**Date:** 2026-08-15 · **Head:** `18e9e5ac` · FE-scoped + additive backend (config-truth).

## Verdict — root cause found
`normalizeStrategyConfig()` (StrategyStudioPage) rebuilt the config from a fixed
field list and **omitted `day_plan`** → stripped on **load** (editor saw
`config=undefined` → master toggle OFF → whole body disabled/greyed) AND **save**
(edits never persisted; the DB had day_plan only via `cmd/dayplan-arm`).

## Investigate table
| Control | was | verdict |
|---|---|---|
| enable · plan_mode · planner_model · proximity · max_levels · scenario_cap · replan_cap · acceptance · approval · digest · session accordion (⚪/🔸) | wired to state, but **stripped by normalize** on load+save | BUG (root) |
| planner_timeframes | rendered as read-only **AUTO spans** | BUG (2nd) |
| regime / AUTO rows · registry windows | non-interactive | **by-design** (now tooltipped) |
| Executor **Indicators** | wired via `updateAIConfig`; `ai_config` was **preserved** by normalize | **no regression** — "indicators too" = the greyed block above, or a read-only `is_default` strategy |

## Fixes (2 commits)
- `79a58523` (FE): normalize now lists `day_plan` (restores load→edit→save→reload
  round-trip for every control); planner_timeframes → **editable multiselect**
  (D/4h/1h/15m/5m toggle, order-preserving); AUTO rows get an "auto-computed — not
  a setting" tooltip.
- `18e9e5ac` (Go, additive): the planner assembler now **honors** the config
  (max_levels cap · per-session min_grade filter — owner grade-A levels always
  survive · planner_timeframes → StructureSummary); nil/default config = prior
  behavior byte-for-byte.

## Config-truth (complete)
save → row JSON shows it → reload → Studio shows it → **assembler reads it**:
`resolveSessionPlanCfg` + `FilterLevelsByMinGrade` unit-tested (timeframes +
min_grade honored; defaults inert). Contract: **edits apply at the NEXT read
(Monday 08:25 CT), never mid-plan.**

## Verify + deploy
tsc + `npm run build` ✓ · FE vitest 154 pass (1 = pre-existing `RegistrationDisabled`
logo, unrelated) · go build/vet/test 26 pkgs 0-fail.
**Owner deploy:** the editability fix is FE-only → **`cd web && npm run build` +
hard-reload** (no bot restart). The assembler-honoring is Go → it rides **★ RESTART
2** (this weekend's rebuild); nothing new to run — it takes effect at Monday's read.
