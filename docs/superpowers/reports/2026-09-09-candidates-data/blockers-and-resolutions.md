# W3 — blockers found by the scout, and how each is resolved

Eight read-only scouts mapped the surfaces. Two findings change the wave's shape; both verified by me
directly, not taken on the scout's word.

## B1 (CRITICAL) — the scoring golden pins the FULL `ScoredLevel`, byte-for-byte

`kernel/stage_a_parity_test.go:21-46` marshals `struct{ Seated, Pool []ScoredLevel }` over **64
fixture combinations** (4 grades × 4 caps × 4 freshness) and byte-compares to
`kernel/testdata/stage_a_score_legacy.json` — **685,898 bytes, 2,680 `"role"` keys**.

```go
seated, pool := ScoreLevelsMinGradeFull(stageAParityLevels(), 30000, 300, func(DetectedLevel) string { return fresh }, cap, 1.5, grade)
results = append(results, struct{ Seated, Pool []ScoredLevel }{seated, pool})
data, err := json.Marshal(results)
```

**Consequence:** adding ANY exported field to `ScoredLevel` (merged names, map role, projection,
entry-candidacy) changes the marshalled JSON and breaks E7's required-empty golden diff.

**RESOLUTION — the new data never touches `ScoredLevel`.** D2/D1/D6 build a **render-time view**
(`MapCandidate`) from `[]ScoredLevel` at the point of rendering. `ScoredLevel` is untouched, the
parity golden stays byte-identical, and E7 passes by construction rather than by re-baselining.
This is also the strictest possible reading of the owner's ruling ("presentation and ordering only").

## B2 (CRITICAL — a hazard the dispatch did not anticipate) — D4 reordering would corrupt owner edits

`web/src/components/plan/EditSheet.tsx` patches plan levels **by array position**:

```ts
:136   [{ op: 'replace', path: `/levels/${levelIndex}`, value }],
:188   [{ op: 'remove',  path: `/levels/${levelIndex}` }],
```

`levelIndex` is the row's index in the rendered list. **If D4 reorders the list the owner is looking
at, an in-flight edit replaces or DELETES a different level than the one clicked** — silent
data corruption of the plan document, and `remove` is not recoverable from the card.

**RESOLUTION — D4 orders the ENTRY SHORTLIST ONLY.** `plan.doc.levels` keeps its existing order and
its existing indices; the reachability ordering applies to the shortlist view (and the model's table),
which carries no edit affordance. This satisfies D4 ("the entry shortlist orders by reachability")
and D1 ("the map is kept whole") without touching the array EditSheet indexes. **No reorder of any
array the card offers an edit control on.**

## B3 — D1's headline is already law: cite class 93, do not re-file it

`docs/superpowers/AUDIT-CHECKLIST.md:2776` already contains **verbatim** "Exclusion is not
invalidation." Filing a new class restating it would re-file 93 and claim another lane's work (A24).
**RESOLUTION:** the new class covers **D3's candidacy refusal** only, and cites 93 for the principle.

## B4 — the A16 census: the dispatch's "highest is 91" does not reproduce

The checklist has **no markdown tables** (`grep -cE '^\|'` → 0), so the dispatch's census command
returns zero rows. Using the two-format census the file itself mandates at `:2568-2573`, the highest
occupied class is **94** on origin/dev, not 91. Also: `:13-14` still reads "Highest occupied class:
**53**" — 41 classes stale — which is the likely origin of a wrong premise. Classes **75, 76 and 77
each appear twice** (once per format); A16 forbids repairing another lane's numbering.
**RESOLUTION:** number AT MERGE from a fresh two-format census, quote it, touch no existing entry.

## B5 — D7's per-read fields cannot be known at boot

`detected` / `merged` / `entry-candidates` / `no-target refused` / `projections` are all **per-read**;
at boot no planner read has happened. Printing `0` would assert a measurement that was never taken
(canon 49; `kernel/detector_d1prime.go:270-271`).
**RESOLUTION — canon already answers this:** CLAUDE.md, *"Boot lines are READ, never literal, and a
field the process cannot know yet prints `n/a`."* The boot line prints `n/a` for the per-read fields
and real values for the resolved ones (`cap`, `order`, `pwh/pwl seatable`); the counts are emitted on
the **per-read** map line. `cap=<n>` is read as the resolved `max_levels`, not `CandidatePoolCap`.

## B6 — practical: no `node_modules` in the worktree

`/home/hoang/nofx-cand/web` has none, and the main tree is deploy-only (A2b), so vitest cannot run
here yet. **RESOLUTION:** `npm ci` inside the worktree before the A12 vitest run — my own worktree,
never the main tree.

## B7 — `GUIDE_BUILT_REV` is stamped at cutover, not in the wave commit

`web/scripts/stamp-guide-rev.sh:19-23` reads the revision from a **running** binary via `/api/health`
and refuses to guess. Per the boot-5 order this is correct: build binary → boot it → stamp → rebuild
`dist`. **RESOLUTION:** the Guide section ships in the wave commit; the rev stamp + `dist` rebuild
happen in the cutover sequence (A4), not before.

## Pre-existing drift found in scope, NOT repaired by this wave (recorded only)

- `GuidePage.tsx:491` renders the literal `12 sections` while `GUIDE_SECTIONS` holds **14** (this
  wave makes 15) — a literal where a read value belongs.
- `faq.ts:7` says "the fourteen questions" while `faq.ts` holds **19**.
- `glossary.ts:122-125` still defines "Thin side" though `faq.ts:106-109` and `levels.ts:123` both
  record it as REMOVED on 2026-08-31.
- `SYSTEM-MAP.md:51` says levels "seat up to 8 per side" — the cap is a **TOTAL**
  (`levels_score.go:603-605`), and 8 is the package default while the bound strategy resolves 12.
- `SYSTEM-MAP.md:86` carries five stale line refs (proximity `:414`→`:423`, confluence `:415-418`→
  `:427`, cluster tolerance `:678-685`→`:719`).
- `kernel/levels_score.go:1148-1150` and `kernel/planner_prompt.go:507-509` both fabricate an
  uncomputed role as `react_zone` (`if role == "" { role = string(RoleReactZone) }`) — a class-49
  fabricated value. The new map-role column will print `n/a`, and the existing column is left alone.

---

## Owner rulings, 2026-09-09 (recorded after the combined boot head was built)

**R1 — 104's commits and the marker.** `fix/session-risk-limits` stays folded in; the boot marker
names WHICH boot carried it. **Superseded by events:** a combined-boot lane merged 102, 103, 101 and
104 onto one head at 21:25–21:27 (`cbbc0346` · `a59b6c9d` · `fcb42c49` · `8941ec68`), so 104 rides
**that** combined boot, not this wave's. The marker for it is that lane's to write.

**R2 — the episode contract is 101's, not 102's; 102 is bars-horizon.** A mis-assignment this
session propagated: the dispatch says *"102 holds fix/episode-contract"* and I repeated it in a
status report without checking whose lane it was. The map view reads its episode ids when it lands,
not before. Nothing to revisit.

**R3 — boot on the first genuine flat gate**, not on a clock: arms terminal, book empty, my own
fresh read. The kill is printed for the owner to run (A3).

## What happened to this branch, recorded because it is the PUSH-EMPTY-AT-ACCEPT case

At 21:25–21:27 a combined-boot lane merged this wave's `35fa69cc` into a shared boot head, together
with 102, 101 and 104. That lane also **renumbered this wave's CLASS 95 to 98 itself**, with a
correct provenance note: dev had already landed 95 and 96 from `fix/session-risk-limits`, and A16
forbids renumbering a LANDED entry, so the unlanded one moves. That is the right call and it is
recorded here so the two provenance notes do not read as contradicting each other.

**The hazard this created.** This branch was based on the pre-merge dev. Every W3 file was already on
dev via the combined head, so `git diff origin/dev HEAD` showed **5,403 deletions** — 102's
`bar_horizon_warn.go`, `regime_input_window.go` and 101's `weeklyBias.ts` among them. **Merging this
branch as it stood would have deleted three other lanes' work.** It was reset onto the combined head
instead, and only the genuinely-unlanded deltas re-applied: the owner's line-13 fix and class 99.

The lesson is the one PUSH-EMPTY-AT-ACCEPT exists for, seen from the other side: a lane can have its
work merged by someone else while it holds, and a branch that was correct an hour ago becomes a
deletion patch without anything failing. Before any merge, diff against the CURRENT dev and read the
deletion count — not just the conflict list.
