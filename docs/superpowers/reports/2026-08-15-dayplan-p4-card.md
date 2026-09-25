# VL DAY-PLAN CAMPAIGN — P4 · THE CARD (frontend) — completion report

**Date:** 2026-08-15 · **Branch:** main · **Head after P4:** `859353b3`
**Scope:** P4.1–P4.5 (the Plan Card + its API + the Studio config block). Additive
+ dormant: nothing renders/changes until `day_plan` is enabled AND a plan exists;
the "enabled but no plan yet" pre-★2 state degrades gracefully. SIM-only, owner
data untouched, guardrails master switch untouched, no bot restart.

---

## Verdict

**P4 COMPLETE.** All five items shipped, each its own commit, pushed per-part. Exit
bar green: `tsc` clean · `npm run build` ✓ · `vitest` 140 passed (the 1 failure is
a pre-existing unrelated logo test) · Go suite 26 pkgs ok / 0 fail · no goldens
touched · i18n en/zh/id complete. Live behavior verify is owner-gated (needs the
bot serving `/api/plan/*` + a futures trader) — deferred to the next ★ window.

---

## Commits (7)

| SHA | Item |
|---|---|
| `7674261b` | P4.1 — `/api/plan/*` API (today/history/alerts + ack) + AlertStore |
| `40ec67a3` | P4.1 fix — scope alert-ack by trader_id (IDOR guard, commit-review) |
| `f1ccb76e` | P4 FE foundation — tokens, plan API client, i18n, hooks, chips, sessions |
| `842edb57` | P4.2 — SessionTimelineStrip + SessionTabs + HandoverBanner |
| `fd27f63a` | P4.3 — SessionPlanCard (bias/chart/levels/scenarios/rules/footer + states) |
| `d4564cf5` | P4.4 — in-app alert center (bell + feed + P0 toast/banner) |
| `859353b3` | P4.5 — Studio Day Plan block (config + sessions accordion) |

---

## P4.1 — `/api/plan/*` (Go) **[A]**

Mirrors `/api/risk/*` (inline `trader_id`, `routeWithSchema`, `Safe*` helpers),
JWT-protected group. Four routes:

- `GET /plan/today` — active plan (overlay-resolved) + live per-level facts from
  the P0.4 evaluator (`EvaluateLevelFacts`: sweep / closes-beyond / acceptance /
  still-valid). Returns `found:false` for night / disabled session / **no-plan-yet
  (pre-★2)** so the card shows its graceful state. Overlays empty pre-P5 →
  `plan_final = base doc` (RFC-6902 apply is P5).
- `GET /plan/history` — recent plan versions (`PlanStore.ListRecent`).
- `GET /plan/alerts` — feed + unacked bell count.
- `POST /plan/alert-ack` — ack, **trader-scoped**.

**New:** `store/alert.go` — `AlertStore` (`day_plan_alerts`), the P4.4 bus. `Emit`
is idempotent (deduped by `(trader_id, event_id)`).

**Security (commit-review, addressed):** the ack was an IDOR — any id ackable.
Fixed with `AckForTrader(traderID, id)` (`WHERE id=? AND trader_id=?`) → 404 if not
owned. The `ListRecent` cross-tenant flag is **N/A**: `plans` has no owner column
by design (global session artifacts keyed by trade_date/session), the route is
JWT-protected (single-owner SIM bot), and `trader_id` is validated as an owned
trader. No cross-tenant plan data exists to leak.

**Go tests (11):** alert store (dedupe / order / unacked / ack / IDOR scoping) +
handler `trader_id`/`alert_id` contracts.

## P4 FE foundation **[A]**

- `theme/vl-tokens.css` — the design-system tokens (the LAW): colors, 3-font
  system (Cormorant/Inter/JetBrains Mono, added to `index.css`), session hues,
  motion. Components consume tokens only (no raw hex). App stays dark-only.
- `lib/api/plan.ts` — typed client; GETs silent + fail-soft (null/empty → no-plan
  state, never toasts).
- `i18n/plan-translations.ts` — self-contained feature i18n (en/zh/id enforced by
  type; mirrors `strategy-translations` pattern), `tp()` with `{param}` interp.
  Chosen over editing the 4000-line `translations.ts` (identical anchors made
  insertion fragile).
- `usePlan.ts` (SWR hooks) · `chips.tsx` (Grade/Provenance/Fresh/Status/Version/
  Lifecycle — status is always text+shape, never color alone) · `sessionConfig.ts`
  (24h session/killzone table mirrored from `kernel/session_registry.go`; visual
  only — backend owns live session state).

## P4.2 — timeline / tabs / handover **[A]** (10 tests)

Strip: asia/london/ny bands + killzone ticks + now-marker + DST-warning variant
(London out-of-phase weeks). Tabs: `role=tablist`, active(gold underline + pulse
dot)/inactive/reading/disabled(night), arrow-key nav, disabled refuses select.
HandoverBanner: expired → reading → born → read-failed(fail-closed), `aria-live`.

## P4.3 — SessionPlanCard **[A]** (14 tests)

Pure VIEW of `plan_final` (computes nothing tradable). Bias · **PlanMiniChart +
LevelOverlayPrimitive** (v5 `ISeriesPrimitive` cloned from
`SessionVolumeProfile.ts`; consumes the SAME level array as the table — one array,
three renderers; lines styled by provenance, grade→opacity; zone bands + scenario
markers activate on owner-overlay data, P5; canvas-less env → placeholder,
jsdom-safe) · ZoneTable (price/provenance/grade/fresh/instruction/signed-distance;
near→gold; consumed dims 50%; 👤/📝/S-tag on overlay data) · ScenarioList
(status **read-only from backend**, absent→'armed'; grammar + uses-chain) ·
RulesBlock (short-tinted no-trade + death) · PlanFooter. States: loading / night /
no-plan-yet / fail-closed / expired / active + WARMING(n) + UNCALIBRATED badges +
advisory banner. **PlanCard** panel mounts top-left in TraderDashboardPage, gated
on `isFutures` (crypto untouched).

*Scenario live status:* no backend state machine exists yet (kernel has per-LEVEL
facts only). Per the single-authority rule the UI does NOT compute it — an OPTIONAL
`scenario_status` map lights the dots when the executor-phase state machine ships;
zero FE change needed. **[A]**

## P4.4 — in-app alert center **[A]** (5 tests)

NO external push. Bell + unacked badge + dropdown feed. P0 (halt/fill/close+P&L/
read-fail/plan-died) → sonner toast per NEW P0 (backlog seeded silently) + a
persistent `role=alert` banner until ack. P1 → feed rows. P2 → digest count.
Ack is trader-scoped + revalidates.

## P4.5 — Studio Day Plan block **[A]** (4 tests)

Futures-only block (same grammar as RiskControlEditor), saved via existing
PUT→hot-reload. `plan_enabled=false` master switch; enabling materializes the Go
`DefaultDayPlanConfig` defaults. Model(pinned) · mode segmented · planner-reads
AUTO chips (zero indicator toggles) · one-line REGIME(AUTO) · filters (proximity
slider/max-levels/scenarios/re-plans/acceptance/approval-OFF/digest-ON) · sessions
ACCORDION with ⚪inherit/🔸override chips per field (min_grade/max_trades/plan_mode/
replan_cap/acceptance) + ACTIVE windows (killzone→"ACTIVE" wording) + London DST.
Root config via `updateConfig('day_plan', …)`; dropped for non-futures.

---

## Exit bar

| Gate | Result |
|---|---|
| `tsc --noEmit` | **clean** |
| `npm run build` | **✓** (pre-existing chunk-size warning only) |
| `vitest` | **140 passed** / 1 failed = pre-existing `RegistrationDisabled` logo test (proven failing at `40ec67a3`, pre-P4) |
| Go `build`+`vet`+`test ./...` | **26 pkgs ok, 0 fail** (api + store touched) |
| Goldens | none touched (bar-goldens untouched; full go suite green) |
| i18n | en/zh/id complete via `plan-translations` + `tp()` |
| Mobile 430px | designed responsive (flex-wrap, relative widths, truncation, grid collapses to 1 col) — **[B]** not browser-verified in this headless env |

## Deferred / follow-ups

- **Live behavior verify** (owner-gated): bot serving `/api/plan/*` + a futures
  trader with `day_plan.plan_enabled=true` — deferred to the next ★ window.
- **`day_plan_enabled` on SystemStatus** — the mount is gated on `isFutures`; a
  precise status flag would hide the panel for a futures trader with the feature
  off (harmless-dormant "no plan yet" meanwhile). **[B]**
- **P5 door:** EditSheet / BulkAdd / AskPlanner / RFC-6902 overlay apply + owner
  origin/note/scenario-tag + scenario state machine (dots) — the card's forward-
  compat fields are already in place.
- **vlauto propagation:** DEFERRED (per dispatch) to the next propagation train.
