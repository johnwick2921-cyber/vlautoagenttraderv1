# VL DAY-PLAN — DESIGN CONFORMANCE AUDIT (built system vs blueprints)

**LINE 1 — NOT YET DESIGN CONFORMANT: 57 breaks-intent deviations** (+82 cosmetic).
245 matrix rows: **112 MATCH · 88 DEVIATION · 45 MISSING**. Tokens are a perfect match;
the *components* are largely built to design — but the **API under-feeds them**, so a
large share of the designed card is unreachable code. READ-ONLY audit; only this file
is committed. HEAD `298d75b0`, tree otherwise clean, bot untouched.

## Method + evidence tiers
- **Blueprints:** the 4 mockups (`plan-card-v2` · `multisession-mockup` · `plan-config-mockup`
  · `askplanner-mockup`, read in full — their `:root` block is the token truth) +
  `docs/PLAN-CARD-DESIGN-SYSTEM.md` + `docs/VL-DAYPLAN-FULL-SPEC.md`.
- **Live render:** the REAL components mounted with fixtures in a temporary Vite harness
  (`web/audit.html` + `web/src/__audit__/`, **uncommitted, deleted after**) behind a
  fixture API stub — the owner's account, the live API and the DB were never touched.
  This renders true component output; it does not prove end-to-end data flow (that is
  audited separately in code, and is where the failures are).
- **Tiers:** **[A]** = I personally verified the code/render this session. **[B]** = a
  sub-auditor's finding with a file:line receipt that I did not re-derive. Every
  breaks-intent item below is marked.

## Score by dimension
| Dim | Area | MATCH | DEVIATION | MISSING | breaks-intent |
|---|---|---|---|---|---|
| D1 | The card (7 sections + states) | high | many | 9 | 9 |
| D2 | Multi-session (strip/tabs/handover/accordion) | mid | many | 4 | 7 |
| D3 | Config + edit sheet | mid | many | 5 | 9 |
| D4 | Ask-Planner | high | some | 4 | 7 |
| D5 | Tokens | **24/24 exact** | 0 | 0 | **0** |
| D6 | Behavior contracts | mid | 6 | 2 | 6 |
| D7 | Full contract sweep | mid | many | 21 | 19 |

## THE ROOT CAUSE — one API omission strands ~⅓ of the built card [A]
`api/handler_plan.go:158-169` builds each level fact field-by-field and emits **only**
`price·label·grade·instruction·distance·sweep·closes_beyond·accept_*·still_valid`.
It never emits **`origin`, `note`, `scenario_id`**, there is no **range** (proximal/distal)
anywhere in the stack (`kernel/plan_doc.go:21-26` `PlanLevel{Price,Label,Grade,Instruction}`),
and **`scenario_status` is emitted nowhere in Go** (repo-wide grep: FE-only).

Because of that single gap, all of this is *built and correct but unreachable*:
| Designed feature | Built at | Why it never renders |
|---|---|---|
| 👤 owner marker | `ZoneTable.tsx:65-73` | `fact.origin` never sent |
| 📝 note icon | `ZoneTable.tsx:74-82` | `fact.note` never sent |
| [S-tag] play chip | `ZoneTable.tsx:91-101` | `fact.scenario_id` never sent |
| ⚡ ConflictChip + AI ghosting | `levelState.ts:44-68` + `chips.tsx:199-216` | keys on `origin` |
| Zones as **bands** (the design's headline rule) | `LevelOverlayPrimitive.ts:94-106` | no `range` to feed it → every zone draws as a line |
| Scenario ○/◉/●/✕ live status | `chips.tsx:100-156` | no `scenario_status` → `?? 'armed'` |

**My harness proves the components work**: fed `origin/note/scenario_id`, the card rendered
👤 📝 and the `S1` chip correctly (`live-D1-card-active.png`). The FE is not the problem.

## Breaks-intent findings (severity-ordered)
### The card asserts state the system never computed — "the UI must not lie"
1. **Every scenario shows gold ○ ARMED forever** [A] — `ScenarioList.tsx:131` `?? 'armed'`;
   no Go emitter. Triggered/invalidated/expired plays all read ARMED. Visible in the
   screenshot: S1, S2 **and** S3 all "ARMED".
2. **Card says ADVISORY even in direction/strict** [A] — `handler_plan.go:77,125` hardcode
   `"mode":"advisory"` ("P3.5 wired advisory only") while W9 made the executor honor the
   real per-session mode (`trader/auto_trader_planconfig.go:38-49`). The card *understates
   the plan's live authority over entries*.
3. **"Keep as-is" renders the green "✔ Applied" strip** [A] — `AskPlannerPanel.tsx:229`
   `onClick={() => setApplied(true)}`. Declining a patch tells the owner it was applied;
   nothing was sent, server `applied` stays false, buttons return on reload.
4. **"2 re-reads left" is hardcoded** [A] — `handler_plan.go:129` `maxI(0,2-(row.Version-1))`
   ignores the resolved `replan_cap` (W9 `replanCapFor`). Set 0 or 4 → card still says 2.
5. **A disabled session renders "No plan yet — the planner arms at the session open"** [B]
   (`SessionPlanCard.tsx:121-138`) — a false promise; the correct `disabled` strings exist
   but are dead code.
6. **UNCALIBRATED chip clears without any blind-mark** [B] — gated on the SVP counter; no
   blind-mark artifact exists anywhere in the repo.

### Owner-facing controls that don't exist or don't reach the engine
7. **No Approve button** [A] — `approval_required=ON` (Studio `DayPlanEditor.tsx:415-418`)
   holds every entry, and the card offers no way to approve. *Correction to the
   sub-auditor: the route **does** exist* (`POST /api/plan/approve`, `api/server.go:478`,
   built in W9) — only the FE surface is missing. Today the owner must curl to trade.
8. **Tab selection is inert** [A] — `usePlanToday(traderId, symbol)` takes **no session**;
   tapping LONDON moves the gold underline and keeps showing the NY card.
9. **No per-session ENABLE toggle**, and `sessions_enabled` has no UI [B] — so ASIA/LONDON
   can never be turned on from the Studio, while the accordion invites overrides for them.
10. **Timeline/tabs read a hardcoded `enabled`** [A] — `sessionConfig.ts` consts; the W8
    admin registry (`GET /api/plan/session-registry`) has **zero FE callers**. Two clocks.
11. **Edit-mode silently drops the note** [A] — `EditSheet.tsx:117-128` patches
    `{price,label,grade,instruction}`; the typed note is discarded after a success toast.
12. **Edit mode cannot change a level's TYPE** [B] — the row is gated `{!isEdit && …}`.
13. **Planner model is a free-text input, not a picker** [B]; the **pinned exact id is never
    echoed** [B] — the owner edits an alias whose real effect (exact id + stats-window reset)
    is invisible.
14. **No Re-read action** [B] — the footer counts re-reads the owner cannot spend.

### Contract violations
15. **Zones cannot be expressed at all** [A] — no range in `PlanLevel`/`PlanLevelFact`/handler;
    the edit sheet accepts "30160 – 30152" and silently saves `30160`.
16. **No `test`-op concurrency on owner saves** [B] — `EditSheet` posts bare index paths
    (`/levels/N`) from a ≤15s-old snapshot; the server guard exists but nothing emits it.
17. **Ask-Planner reasons over the BASE plan, not `plan_final`** [B] — `handler_plan.go:449-462`
    skips overlay folding, so the planner can deny an owner level the card is showing.
18. **Bulk add = N POSTs, not one overlay version** [B] — partial failures leave a half-applied
    set; no atomic version to replay.
19. **No version bump is visible after an edit/apply** [B] — `overlay_count` is typed and
    never consumed; the apply response's `overlay_version` is discarded by the client.
20. **Ask-Planner never closes on Apply** [B] — the design's one "see the bump live" moment.

### Multi-session correctness (latent — ASIA/LONDON are off today)
21. **The 16:55 ASIA read can never fire at 16:55** [A] — `IsCMEOpen` returns false for
    Mon-Thu `hour==16` (`kernel/cme_calendar.go`), and W1's read gate requires it.
22. **`plannerTradeDateCT` has no 17:00 roll** [A] — Sunday/evening ASIA plans file under the
    closing calendar day, breaking the `(trade_date, session)` key the spec defines.
23. **One flat, not one per session boundary** [B] — `enforceEODFlat` never reads
    `SessionDef.FlatCT`; enabling ASIA/LONDON would carry positions through the boundary.
24. **Read-side APIs hardcode the default registry** [B] — `handler_plan.go:67,357,437,527,586`.
25. **Risk defaults (ASIA min_grade=A + max_trades=1) not shipped** [B]; `max_trades` and
    per-session `acceptance_rule` have **zero readers** [B].
26. **HandoverBanner: only `expired` is wired** [A] — all four phases render correctly in the
    harness (screenshot), but `reading`/`born`/`read-failed` are never passed. The owner is
    never told a read FAILED and the bot is sitting out.
27. **No next-handover times** [B] — `readMin`/`flatMin` exist in the FE table and are unused.

### Governance
28. **The spec still forbids what HEAD ships** [A] — `VL-DAYPLAN-FULL-SPEC.md:73` "NO indicator
    toggles" and `:147-148` DO-NOT-BUILD "planner indicator toggles" vs W11's shipped indicator
    mirror. W11 was an explicit **owner override** and adds *zero* new config fields, so the
    letter is honored — but the doc was **never amended** (no W11 line, no 8th RECON
    amendment), violating `:151` "changes update the doc first". A future agent reading the
    governing doc would delete the mirror as out-of-spec. **Recommend: add the amendment.**
    Note the invariant it protected is now partly false: the planner reads the *executor's*
    tables, and the executor's indicator toggles silently double as planner inputs.

## D5 — TOKENS: 24/24 EXACT [A]
Computed from the live render: every hex (`--vl-ink #0c0e12` … `--vl-short #e06c6c`), grade
a/b/c, the three session hues (`#5b7fb0`/`#c98a4b`/`#3fbf8f`), radii **14/10/6**, and
`--vl-motion-pulse: 2.2s` all match the design table exactly. The full designed set is
present (killzone-fill, night-wash, shade-active/prior, status-armed/triggered/invalid,
4-base spacing). Font stacks add safe fallbacks (`Georgia`, `IBM Plex Mono`) — a superset,
not drift. **The design system's token layer shipped perfectly.**

## What renders correctly (not everything is broken)
Header + Cormorant title + lifecycle chip · bias word/conviction/flip line · levels table
with provenance chip, grade badge, fresh dot **and the distance column** · consumed row
dimmed (not deleted) · scenario grammar lines with targets + invalid · the red NO-TRADE /
PLAN-DIES-IF block · footer version chips + day-type · 24h timeline with **killzone ticks**
(which the mockup itself omits) + gold now-marker · tabs with gold underline + pulse dot ·
Ask-Planner as a desktop right-side panel with scope chip, EVIDENCE → Your-point →
NEW-INFO/BARE chip → gold PROPOSE-MERGE verdict → RFC-6902 patch rows → Apply/Keep
(both present [A]) · quick-chips + any-language input bar.

## Screenshot index (uncommitted, `.playwright-mcp/audit/` — gitignored)
| File | Shows |
|---|---|
| `live-D1-card-active.png` | full active card: 👤📝+S1 owner row, dimmed consumed row, all-ARMED scenarios |
| `live-D2-nav.png` | timeline (3 hues + killzone ticks + now-marker), tabs, all 4 handover phases |
| `live-D4-askplanner.png` | Ask-Planner panel: scope chip, evidence, NEW-INFO, PROPOSE-MERGE, patch rows |
Blueprint side = the 4 mockup HTMLs at `/mnt/c/Users/hoang/Downloads/` (read in full).

## Fix order (highest value first)
1. **Emit 4 fields + scenario status** from `handler_plan.go` (`origin`, `note`,
   `scenario_id`, and a real `scenario_status`) → lights up 👤/📝/S-tag/ConflictChip and
   stops the ARMED lie. One handler + the scenario state machine.
2. **Stop the three lies**: real `mode`, real `replans_left`, and `Keep as-is` must not
   set `applied`.
3. **Add the Approve button** (the route already exists) — otherwise `approval_required`
   is a trap.
4. **Add `range`** to `PlanLevel` end-to-end so a zone is a zone.
5. **Session-scope the fetch** (`usePlanToday(session)`) so tabs navigate.
6. **Amend the spec** for W11 (governance rule `:151`).
