# FIX-PLANNER item 5 — offline re-judge of the killed chains (zero API)

- **Lane:** DS-101 · branch `fix/planner-killers` · base `origin/dev` `04ae1c2f`
- **Specs built from (L3):** CTO dispatch 2026-09-26 04:59Z (id 1790398758580-4857-000001, scope-amended 05:06Z); CTO fold 05:50Z (id 1790401854275-64319-000001 — DS-104 cross-check PR #242 labels + census boundary); census `docs/superpowers/reports/2026-09-26-plan-death-census.md` (6bbb3a6c).
- **Method:** read-only on the DB COPY (`/home/hoang/nofx-ds-101-plandeath/ab.db`, 09-25 22:39 CT). No API calls, no writes, no live system.
- **Census boundary (fold 3):** 80 rows = 36 reads; pre-boot ids 336-338 are OUT (UTC-vs-CT boundary). Killed reads below are the corrected set.

## Corrected killer labels (fold 2) — final attempt reason, verbatim from the DB

| chain ids | day/session | final killer (verbatim class) |
|---|---|---|
| 339-341 | 09-23 ASIA | flip met |
| 349-351 | 09-23 ASIA | obstacle chain (first_obstacle not nearest) |
| 352-354 | 09-23 ASIA | identity unresolved |
| 376-378 | 09-24 ASIA | flip met |
| 385-387 | 09-24 ASIA | obstacle chain (omits) |
| 391-393 | 09-24 ASIA | schema_json (unmarshal string→float64 breakdown.level) |
| 357-359 | 09-24 NY | schema_json (unmarshal string→float64 breakdown.level) |
| 360-362 | 09-24 NY | tape_verify (breakdown, no confirming close) |
| 363-365 | 09-24 NY | born-dead |
| 394-396 | 09-25 LONDON | schema_json (unmarshal number 0.5→PlanArmLeg…) |
| 397-399 | 09-25 LONDON | born-dead |
| 400-402 | 09-25 LONDON | gap trigger |
| 409-411 | 09-25 NY | born-dead |
| 412-414 | 09-25 NY | confirm-shape (`confirm.side "" invalid`) |

Kills by class: born_dead 3 · schema_json 3 (tied top) · flip 2 · obstacle 2 ·
identity 1 · tape_verify 1 · gap 1 · confirm-shape 1.

## Born-dead kills (3) — would the salvage publish them?

Salvage rule applied to the PERSISTED `born_check` records the refusals wrote
(`system_config plan_liveness_event:born_dead_refusal:*`), never reconstructed:

| chain ids | final attempt event (born_check) | verdicts | salvage |
|---|---|---|---|
| 363-365 | 1790261940785248219 @09:59:00.785 CT (row 365 @14:59:00Z) | S1 alive · S2 alive · S3 invalidated · death not_met · flip not_met | **WOULD PUBLISH** (S1+S2 kept) |
| 397-399 | 1790332615980638386 @05:36:55.980 CT (row 399 @10:36:55Z) | S1 alive · S2 invalidated · death not_met · flip not_met | **WOULD PUBLISH** (S1 kept) |
| 409-411 | 1790345154017862744 @09:05:54.018 CT (row 411 @14:05:54Z) | S1 alive · S2 invalidated · death not_met · flip not_met | **WOULD PUBLISH** (S1 kept) |

**3 of 14 killed chains would now publish**, plus each saves its last attempt's
wall via the one-fresh-tape-repair cap.

## Prompt-side fixes — what the zero-call re-judge can and cannot claim

| class | kills | offline claim |
|---|---|---|
| schema_json | 357-359, 391-393, 394-396 | PROVABLE: the repair prompt now carries the exact decode error (field path + type) + the minimal schema for that field instead of the generic excerpt (fold-1; pinned by TestFpUnmarshalRepairExcerptQuotesDecodeErrorAndSchema against BOTH the production parse and the persisted error text). Whether the model then fixes the field needs a model call — not measurable zero-call. |
| obstacle_chain | 349-351, 385-387 | PROVABLE that the repair prompt lists the exact omitted seated levels with id/price/roles (item 2 pin). Model re-author not measurable. |
| confirm-shape | 412-414 | PROVABLE that the repair prompt states the string enum (fold-2 pin). Model re-author not measurable. |
| flip / identity / tape_verify / gap | 339-341, 376-378, 352-354, 360-362, 400-402 | out of this wave's scope — unchanged. |

## Downstream-gate caveat

A salvaged candidate then runs the SAME downstream validators (write truth,
feasibility). Those were never reached by these saved reads, and the saved
`facts` (400 B) do not carry the frozen identity map / zones, so a fuller
offline replay is not possible from the row alone (only row 411 has a saved
`response_text`, 5,308 B). The publish claim above is scoped to the born gate
exactly as persisted.

**Net for the owner:** 3/14 publish under item 1; 6 more killed reads
(schema_json ×3, obstacle ×2, confirm ×1) now receive targeted repair excerpts
instead of the generic line — their effect is measurable only on live reads or
with an approved API spend re-running the saved outputs through the new prompts.
