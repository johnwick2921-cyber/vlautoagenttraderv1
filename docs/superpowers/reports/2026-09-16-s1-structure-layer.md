# S1 — the structure layer (feat/structure-map, 2026-09-16)

**Dispatch:** CTO (nofx-2a) under the owner's delegation; owner's target approved 17:55 CT: TWO tables —
STRUCTURE (D/4h/1h, direction only, never an entry) and ENTRY (the existing 12-seat logic). Every knob
defaults OFF; nothing changes the live plan until DS-R's S4 measurement. No deploy tonight (S6 is
tomorrow's window with the owner present).

**Base (spec-freshness law):** worktree `/home/hoang/nofx-101s` cut from `origin/dev` `f6465143`; claim
`5de94327`. What moved dev past the `054e97e5` marker before this base: `f6465143` only — "docs: 🧮 planner
tape proof landed 17:01:27 CT" (docs-only; the running binary is `7e87a375`).

## What shipped (all behind `day_plan.structure_map`, default OFF)

| | file | pinned by (RED-first) |
|---|---|---|
| refactor | `kernel/structure.go` — `fractalSwings` extracted from `ComputeStructureState` (pure; kernel suite green before/after) | existing structure/MSS tests |
| (a) the table | `kernel/structure_map.go` — `StructureMap{AsOf, Contract, TFs}`, `StructureTF{Trend, LastSwingHigh/Low, ImpulseLo/Hi, PremiumDiscount, Zones, Bars}`, `StructureZone{Kind, Lo, Hi, TF, Fresh, Score}`; trend from the last N=3 labelled swings of the SAME detector; zones = the pool's own zones per TF, best first, cap 6, lines excluded; absent never fabricated | `TestStructureMapReadsTrendImpulseAndPremiumDiscount` (nil → up/108–118/pd≈0.75), `…ReadsDownAndRange`, `…IsAbsentWithoutBars`, `…ZonesAreTheTFsOwnTopZonesLabelIntact`; mutation up↔down proven |
| (b) plan doc | `kernel/plan_doc.go` — `PlanDoc.Structure *StructureMap` `json:"structure,omitempty"`; `PlanFacts.Structure` carries it read → write | doc JSON has the block on / no key off (`TestStructureMapAtTheReadIsAbsentOffAndPresentOn`) |
| (c) knob + prompt | `store/strategy.go` `DayPlanConfig.StructureMap *bool`; `store/resolve_source.go` `ResolveStructureMap` (nil → OFF); `kernel/planner_prompt.go` — `RenderStructureSection` BEFORE the ranked level table; nil → "" | `TestResolveStructureMapDefaultsOff`; `TestStructureSectionIsAbsentWhenTheMapIsNil` (byte-identical) + the untouched planner goldens; `TestStructureSectionRendersBeforeTheLevelTable` |
| (d) observability | `kernel.StructureLogLine` (`🗺 structure @<session>: D=… 4h=… 1h=… zones=<n> pd4h=<0.xx>`, nil → n/a), `kernel.StructureBootLine` (`off|on(D/4h/1h)|n/a`, READ from the bound strategy) beside the 📐 line in `auto_trader_dayplan.go` | `TestStructureLogLines` |
| call site | `trader/structure_map_wire.go` `structureMapForRead` (provider TF "D"→"1d"; pool = the read's uncapped HTF zone universe `htfZonesFull`; `now` from the read, A28) — ONE call in `assemblePlannerInputWithCtx`, `input.Structure` → `facts.Structure` → `doc.Structure` at write | `TestStructureMapAtTheReadIsAbsentOffAndPresentOn` on `trader/testdata/structure_mnq_12-26_{1h,4h,1d}_2026-09-16.json` (store, read-only export: 1h=61 · 4h=22 · 1d=10 NT8-only rows); knob-gate mutation proven |
| registry | `store/knob_registry_table.go` — `structure_map` **advisory** (prompt text only, never a gate; class 122 consumers named) | `TestKnobRegistryIsComplete`, `TestKnobBootLineIsCounted` |
| wiring gate | 7 names registered | `TestEveryClaimedProductionPathHasACallSite` |
| guide | `web/src/guide/content/settings.ts` knob card (48th; census pin re-pointed with the reason), `status.ts` boot line; **no `GUIDE_BUILT_REV` bump** (S6) | vitest src/guide 23/23, tsc |
| checklist | `## CLASS NN (assigned at merge)` — one table for two jobs | — |

`api/strategy.go`: no per-field day_plan handling exists — the knob rides through as JSON; nothing to add.
Validator untouched (S3). `kernel/engine_prompt*` (executor) untouched — the section is the PLANNER prompt's,
which is where the level table lives.

**Contract published first:** bridge msg `1789599693761-281603` (TO: copilot, "FROM: Claude-101 — S1 field
names") before the implementation commits, so S3/S5 build against the same names.

## What the fixture says about today (A21, [A])
On the store's contract-pure 12-26 rows: 1h (61 bars) `range`, last impulse 29052.75–29552.50, pd 0.29 at
29200; 4h (22 bars) `range`, impulse 29208.75–29604.25, pd 0.00 (price below the impulse low); D (10 bars)
`range`, one swing high, no swing low yet. The live ring the planner reads is deeper (1h 1553 · 4h 406 ·
1d 69 NT8 bars, continuous), so a live read will see more; the fixture is what the STORE could give and
the test says so rather than pretending a daily trend from ten bars.

## Suite
`go test ./kernel/ ./store/ ./trader/` ok at `b931827b` (trader 147.7s); guide vitest 23/23; tsc clean.
Full suite at the MERGED head is S6's (with S2/S3/S5).

## Owed / not this PR
S2 (DS-101), S3 (DS-102), S5 (DS-103), S4 (DS-R measurement) · S6 cutover (merge order S2 → S1 → S3 → S5,
full suite at merged head, dist + `GUIDE_BUILT_REV`, flat window with the owner, knobs OFF at boot) · a
DayPlanEditor toggle for the knob if the owner wants it in the UI (today it is an API/JSON field).
