# W13 — PLAN RE-ALIGNMENT ON OWNER EDIT (owner decision 2026-08-16 · AUTO trigger)

**LINE 1 — W13 DONE + GREEN.** An overlay save now auto-triggers a whole-plan
re-examination that returns a **proposal only**; nothing changes without an Apply tap.
Go 26 pkgs green · FE 50/50 plan tests · tsc + build clean · **0 goldens touched**.

## Commits
- `db5949fa` W13.1 backend — `POST /api/plan/realign`, guardrails, KPI logging
- `14b26aae` W13.2 FE — inline status · no-change chip · proposal card · Apply/Keep · manual button
- (this) doc + report — VL-DAYPLAN-FULL-SPEC owner-interaction section updated

## How it works
Reuses the Ask-Planner path **verbatim** — same `AskPlannerSystemPrompt`
(anti-sycophancy), same `ParsePlannerReply`, same verdict vocabulary, same `plan_qa`
table — so agreement/sycophancy stats stay ONE comparable series. Only the user
prompt differs (`kernel.BuildRealignUserPrompt`): overlay-resolved **plan_final** +
live Go facts + the owner's change (incl. their NOTE, any language) + the four
questions, under *DEFEND unless genuinely new information*. Apply routes through the
existing `/plan/ask/apply` → **one mutation door**.

**Guardrails:** bulk-add = ONE call (never per row) · 20s debounce · per-plan auto cap
`realign_cap` (default 5) → manual **⟳ Re-align plan** button (bypasses cap+debounce)
· SKIP on day-plan-off / night / disabled session / no plan / non-active · FAIL-CLOSED
(no proposal, plan untouched, P1 alert row, never a partial patch) · cost + latency on
every row.

## SAMPLE — owner adds a level → returned proposal (real assembled prompt; reply is a fixture in the live wire format)
Owner change: `add-level · D-4h @ 30156 · A · "sweep+reclaim = entry" · S1 · note "strong 4h OB — tôi tin zone này"`

```
VERDICT: PROPOSE-MERGE   (point_class: NEW-INFO)

EVIDENCE  S1 already triggers on a sweep+reclaim of 30160-52; your D-4h at 30156 sits
          4pts under that band and shares its origin (4h OB). The plan's long bias
          rests on the same demand shelf, and the flip line (2x5m < 30148) is 8pts
          below your zone.
SUMMARY   Your zone does not need a new play — it EXTENDS S1's trigger band. Widening
          S1's trigger and leaving bias/flip untouched: nothing you gave me
          contradicts them.
PATCH (would become v1+o2)
  − replace /scenarios/0/trigger  → "sweep 30160-30156 (4h OB) then reclaim"
  − replace /levels/2/instruction → "magnet — expect touch; no entries at it (S1 target 1)"
```
Note the shape the contract forces: it **defended** bias + flip (no patch ops there)
and only touched what the new evidence actually justified.

## Cost per call
`estimateRealignCostUSD` (logged per row): a typical re-align prompt (~4k chars plan +
facts + change) with a ~1.5k-char reply ≈ **$0.0006–0.0025** at the pinned reasoner's
rate. With the default cap of 5 that is **≤ ~$0.013 per plan/session**. Latency is
recorded alongside (`latency_ms`) on every QA row.

## Tests
Go: gate matrix (bulk→debounced · 19s still debounced · 21s proceeds · cap→capped ·
manual bypasses both · cap 0 = unlimited) · prompt carries change+note+4 questions ·
bulk framed as ONE · no-change classifier · **bare disagreement never patches** · KPI
row persistence (trigger/challenge/cost/latency) + `VerdictStats` comparability ·
fail-closed writes a P1 alert and **zero overlays** · cost scaling.
FE (8): reviewing line · no-change chip auto-fades · proposal card shows target
version + patch rows · Apply→`applyAsk` · **Keep never mutates** · capped points at the
manual button · failed says "plan unchanged" · **bulk-add = exactly ONE `onSaved` with
`batch_count:3` while still saving 3 rows**.
**Goldens: none regenerated** — W13 adds no executor-prompt content.

## Deploy (owner)
```bash
cd /home/hoang/nofx
git pull
go build -o nofx-bin . && echo BUILD OK
sudo systemctl restart nofx
go version -m ./nofx-bin | grep vcs        # expect vcs.revision=<HEAD>, vcs.modified=false
```
Frontend: `cd web && npm run build`, then **hard-reload** the browser (Ctrl+Shift+R) —
Vite's HMR cache otherwise serves the old modules.

## Notes / follow-ups
- Verdict vocabulary stays DEFEND/CONCEDE/PROPOSE-MERGE (contract kept byte-identical
  for KPI comparability); "NO-CHANGE-NEEDED" is the **presentation** of a non-patching
  verdict, mapped in `kernel.IsNoChange`.
- "Keep as-is" currently just dismisses (no server row). If you want declines counted
  in the KPI, that is a small additive endpoint — say the word.
- Swipe-away-to-Keep on mobile is the sheet's existing dismiss gesture; the proposal
  renders inline in the card (not a portal), so it scrolls with the plan.
