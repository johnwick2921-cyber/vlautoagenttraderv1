# VL DESIGN SYSTEM — EXTEND: The Plan Card Pattern
*How the Plan Card connects to every other component, and how the pipeline is designed. Companion to NOFX-MASTER-TRADING-SPEC §2#6 and the Day-Plan Integration Blueprint.*

---

## Problem

The dashboard shows what the system **did** (positions, decisions, equity) but nothing shows what it **intends**. The Plan Card is the intent layer: the day's plan, live scenario status, and the owner's editing door — and it must ship as a *system pattern* (shared tokens, reused primitives, defined contracts) so it looks and behaves like it was always part of VL, not bolted on.

## Existing Patterns (what we reuse, why they're not enough alone)

| Related component | Similarity | Why it's not enough |
|---|---|---|
| **SVP chart primitive** | Draws kernel-computed data (POC/VAH/VAL) on lightweight-charts | Draws one profile; no zones-as-bands, no provenance labels, no scenario markers → we generalize it into `LevelOverlayPrimitive` |
| **Strategy Studio config blocks** (RiskControlEditor, IndicatorEditor) | Per-strategy settings card, futures-visibility gating (`!isFutures` pattern from F5) | No plan settings exist → new `DayPlanEditor` block, same visual grammar |
| **Trader/decision cards** | Dark card, status chips, mono numerics | Show past events; no live scenario state machine, no edit mode |
| **Guardrail counters endpoint pattern** | GET /api/risk/* JSON for UI | Extended with /api/plan/* family |

## Design Tokens (formalized from the v2 mockup — single source: `web/src/theme/vl-tokens.css`)

```css
/* color */
--vl-ink:#0C0E12;        --vl-card:#14171D;      --vl-card-2:#181C23;
--vl-gold:#C9A24B;       --vl-gold-dim:rgba(201,162,75,.14);
--vl-gold-line:rgba(201,162,75,.22);
--vl-ivory:#E8E3D8;      --vl-muted:#8A8F98;     --vl-faint:#5A6069;
--vl-long:#3FBF8F;       --vl-short:#E06C6C;     --vl-hair:rgba(232,227,216,.08);
/* semantic aliases (never use raw hex in components) */
--vl-grade-a:var(--vl-gold); --vl-grade-b:#9AA3AE; --vl-grade-c:var(--vl-faint);
--vl-status-armed:var(--vl-gold); --vl-status-triggered:var(--vl-long);
--vl-status-invalid:var(--vl-short);
/* type */
--vl-font-display:'Cormorant Garamond',serif;   /* card titles, bias word */
--vl-font-ui:'Inter',system-ui;                  /* labels, body */
--vl-font-data:'JetBrains Mono',monospace;       /* ALL prices/numerics */
/* spacing: 4-base scale (4/8/12/16) · radius: 14 card / 10 inner / 6 chip
   motion: pulse 2.2s ease · sheet 200ms ease-out · respects reduced-motion */
```

**Token rule:** components consume tokens only. Any raw hex/px in a PR is a review flag (the F5 lesson applied to style: the UI must not lie *or* drift).

## Component Inventory (tree + contracts)

```
PlanCard
├── PlanHeader        title · version chips · lifecycle chip · [✎][Re-read][Approve]
├── BiasBlock         direction word · conviction · flip-condition line
├── PlanMiniChart     wraps chart + LevelOverlayPrimitive (zones/levels/markers)
├── ZoneTable
│   └── ZoneRow       price/range · ProvenanceChip · GradeChip · FreshDot · instruction
├── ScenarioList
│   └── ScenarioRow   StatusDot · id · QualityChip · name · grammar line
├── RulesBlock        no-trade windows · plan-death line (short-tinted container)
├── PlanFooter        entries · match% · day-type · gap · re-plans used
├── EditSheet         (modal) controlled-vocab forms for level/scenario add-edit
├── BulkAddSheet      (modal) multi-line natural-text level entry → parsed rows → preview → ONE overlay version; same parser as the blind-mark calibration; owner levels always seated, AI rows drop first at the cap. **Batch flow (owner-confirmed): stage levels one-by-one or in bulk, each row carries price/type/instruction/grade + NOTE (goes to the AI) + SCENARIO TAG — link an existing scenario (S1/S2) or "＋ new play from this level" (drafts a scenario via the grammar, owner-origin); ONE Save commits all rows as one overlay version; card renders each owner level with 👤 + 📝 note icon + its scenario chip, and each scenario row lists the levels it uses.**
├── AskPlannerThread  (modal) **PLACEMENT (decided): never a separate page — mobile = bottom-sheet over the dashboard (entry: 💬 on card header or long-press a level row; Apply drops the sheet so the card's version-bump is seen live); desktop ≥1200px = right-side panel beside the card; thread is per-plan, archives with the plan into the history browser.** plan-scoped Q&A with the reasoner: question + plan + live facts → answer; read-only — may PROPOSE a patch, applied only on explicit owner tap (tagged planner-revised); never touches the cached executor block unless applied. **Reply form (mockup: askplanner-mockup.html): three labeled sections — EVIDENCE (restates provenance/score/live facts with numbers) → YOUR POINT (classified chip: NEW-INFO green / BARE-DISAGREEMENT grey) → VERDICT chip (PROPOSE-MERGE gold-solid / DEFEND red-outline / CONCEDE). PROPOSE-MERGE renders a patch preview card (− remove / + add rows, target overlay version named) + [Apply merge]/[Keep as-is]; Apply → green applied-strip "plan v1+oN · card updated · 👤🤖 planner-revised · verdict logged". Re-analyze rule: new info = surgical overlay patch only, never a silent full re-plan (full re-reads = death line or Re-read tap). Input bar with quick-chips; any language.**
└── ConflictChip      auto-flag on overlapping owner/AI elements with opposing instructions; default OWNER WINS (AI element ghosted, both logged for attribution); [Ask planner] action opens the thread

### Multi-session additions (per the Multi-Session Handoff research)
```
SessionTimelineStrip  24h bands (asia/ldn/ny hues) + now-marker + killzone ticks; DST-warning variant
SessionTabs           role=tablist; states: active(gold underline + pulse dot)/inactive/disabled(night)/reading; auto-advance on handover
SessionPlanCard       the existing PlanCard, session-scoped; states: reading/active/expired/night/disabled/error-failclosed
HandoverBanner        phases: expired → reading → born → read-failed(fail-closed)
SessionsAccordion     per-session rows in the Day Plan block; InheritOverrideChip ⚪inherit/🔸override on every field
Tokens (new)          --vl-session-asia #5B7FB0 · --vl-session-london #C98A4B · --vl-session-ny #3FBF8F ·
                      shade-active rgba(gold,.10) · shade-prior rgba(ivory,.05) · night wash · killzone fill rgba(gold,.22)
Level naming          PDH/PDL = calendar-day only · RTH-H/L · AS-H/L · LDN-H/L · ONH/ONL = Asia+London composite —
                      every level session-prefixed (one array, all renderers, unchanged contract)
Storage note          plan key = (trade_date, session, version); decisions carry session
```
```

### Key component contracts (skill format, abbreviated to the load-bearing three)

**ZoneRow** — props: `{level: {id, price|range:[proximal,distal], provenance, grade:'A'|'B'|'C', fresh:'fresh'|'tested'|'consumed', instruction, origin:'AI'|'OWNER', distancePts}}`. States: default · **near** (|distance| < 0.25×ATR → distance chip goes gold) · **consumed** (row dims 50%, strikethrough none — history stays legible) · **owner** (👤 tag) · editing. A11y: `role="row"`, full text announce "PDH thirty-two eighty-eight, grade A, fresh, target only, fifty-seven points above".

**ScenarioRow** — props: `{scenario: {id, quality:'A+'|'A'|'B', name, grammar:{trigger, condition, direction, targets[], invalid}, status:'armed'|'waiting'|'triggered'|'invalidated'|'expired', origin}}`. States map 1:1 to `--vl-status-*` tokens; `triggered` pulses (reduced-motion: static ring). The **status is read-only from the backend state machine** — the UI never computes trading state (single-authority rule, same as the gates).

**LevelOverlayPrimitive** (chart) — input: the SAME level array ZoneTable renders (one prop, one source). Zones → translucent bands (long/short tint at 10-12% alpha); lines → dashed gold (naked POC), solid gold (PDH/PDL), hairline (RN); scenario triggers → ▲/▼ markers with id label. Never fetches its own data — **props only**, so card and chart can't diverge.

### DayPlanEditor (Strategy Studio block)
Same visual grammar as RiskControlEditor: tile grid, futures-visible, per-strategy. Fields per the Integration Blueprint settings table (13 keys, defaults OFF). Uses the existing save→hot-reload path — no new persistence mechanics.

## The Pipeline (connection design — who talks to whom)

```
                    ┌──────────── Go KERNEL (one compute) ───────────┐
 BarCache ─────────►│ indicators (existing) + level/zone DETECTORS   │
                    │ + confluence SCORER (ranks, grades, filters)   │
                    └──────┬─────────────────────────┬───────────────┘
                           │ scored candidates       │ per-cycle fact
                           ▼                         │ primitives
                    8:25 PLANNER (LLM) ──plan JSON──►│ (sweep, closes-
                           │                         │  beyond, dist)
                           ▼                         ▼
                    SQLITE: plans (append-only) + plan_overlays (RFC6902)
                           │ hot-read per cycle (D2 pattern)
        ┌──────────────────┼──────────────────┬────────────────────┐
        ▼                  ▼                  ▼                    ▼
  EXECUTOR PROMPT    GET /api/plan/today   PlanCard UI      LevelOverlay
  (cached block +    (plan_final = plan    (renders          Primitive
   dynamic facts)     + overlay merged,     plan_final)      (same array)
                      + live scenario
                      states via SSE/poll)
        │
        ▼
  executor_decisions rows carry (plan_id, version, overlay_version,
  scenario_id) ──► DecisionFeed shows a ScenarioChip per entry ──►
  replay harness + AI-vs-OWNER attribution join
```

**Contract rules (the ones that keep it honest):**
1. **One array, three renderers.** ZoneTable, LevelOverlayPrimitive, and the executor's level lines all consume the identical `plan_final.levels` — divergence is structurally impossible.
2. **UI computes nothing tradable.** Distances shown are backend-computed; scenario states come from the Go state machine. The card is a *view*, exactly like the chart is.
3. **Edits are patches, not mutations.** EditSheet emits RFC 6902 ops → POST /api/plan/overlay (with `test`-op concurrency) → new overlay version → every consumer re-renders from the new `plan_final`. The AI's original plan is never rewritten.
4. **Owner inputs pass the same armor** (B2 price sanity) — the EditSheet surfaces the rejection reason inline, same voice as agent logs ("⛔ 8.2×ATR from price").
5. **New API family:** `GET /api/plan/today` · `POST /api/plan/overlay` · `POST /api/plan/replan` · `POST /api/plan/approve` — same auth, same JSON conventions as /api/risk/*.

## Placement & Responsive

Desktop: PlanCard top-left main column (chart keeps center); mobile: PlanCard first, chart collapses into PlanMiniChart. No existing panel moves — the pattern is additive (same rule as every fix this month).

## Accessibility

Card: `role="region"` labeled "Today's plan, version 2, active". Full keyboard path: header buttons → rows (Enter opens EditSheet) → sheet is a focus trap, Esc closes. All status conveyed by text+shape, never color alone (dots carry labels). Pulse honors `prefers-reduced-motion`. Mono numerics use `font-variant-numeric: tabular-nums`.

## Do's and Don'ts

| ✅ Do | ❌ Don't |
|---|---|
| Consume `plan_final` from the API everywhere | Merge overlay client-side per component |
| Use `--vl-*` tokens and the 3-font system | Introduce a 4th font or raw hex |
| Reuse `!isFutures` gating for the Studio block | Invent a second visibility mechanism |
| Keep origin tags visible (👤) | Hide whose idea a level was |
| Dim consumed levels | Delete them mid-session (audit trail) |

## Open Questions (design review before build)

1. EditSheet on mobile: bottom-sheet vs full-screen? (Recommend bottom-sheet, matches tap-price flow.)
2. Version chips: how many before overflow → "v1…v4" collapse? (Recommend collapse at 4.)
3. Should DecisionFeed's ScenarioChip deep-link back to the card's scenario row? (Recommend yes, cheap.)
4. Vietnamese i18n strings for instruction verbs — translate or keep the controlled vocab English-only for replay-log consistency? (Recommend: UI translates, stored values stay English.)
