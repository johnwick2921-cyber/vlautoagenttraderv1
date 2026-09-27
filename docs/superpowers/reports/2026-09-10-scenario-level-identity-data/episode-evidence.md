# Dispatch 105 — C2 and the D4/D5 boundary, source evidence

Read-only investigation of the running source pin `770e2297`, performed in the claimed identity worktree. This artifact records source evidence only; the parent report owns live process verification and the live database census. No implementation, build, test, migration or live database query was performed by this investigator. The parent accepted STOP after the gate dependency was reported. The only write is this report artifact.

## Result: D4 changes a live arm gate if applied to the shared evaluator

**[A] The scenario evaluator is not a recording-only consumer.** Its verdict supplies EntryGate's arm-invalidation refusal. Therefore making candidate identity authoritative over its evaluation anchor can change which arms pass. This remains true even if the diff never touches `entry_gate.go` or `armed_executor.go`.

The exact production dependency at `770e2297` is:

```
entryGateForArm
  -> scenarioInvalidationResolver(plan)
  -> scenarioInvalidationAt(plan, scenarioID, now)
  -> kernel.EvaluatePlanScenarios
  -> kernel.EvaluateScenario
  -> kernel.ScenarioAnchor
  -> EvaluateLevelFacts(bars, anchor, ...)
  -> ScenarioInvalidated
  -> EntryGate returns (message, true)
```

Pinned excerpts:

- `trader/entry_gate.go:383`: `ScenarioInvalidation:      at.scenarioInvalidationResolver(plan),`
- `trader/invalidation_resolver.go:55-56`: `_, evals := kernel.EvaluatePlanScenarios(` with `plan.Doc, windowed, price, dATR, kernel.ActivationWindowK, rule, true, now.UnixMilli())`.
- `kernel/scenario_state.go:174`: `anchor, ok := ScenarioAnchor(s, levels)`.
- `kernel/scenario_state.go:198`: `f := EvaluateLevelFacts(bars, anchor, dir, rule, 3, nowMs)`.
- `kernel/scenario_state.go:205-206`: `case f.Accepted:` then `out.Status = ScenarioInvalidated`.
- `trader/invalidation_resolver.go:67-68`: `if e.Status != kernel.ScenarioInvalidated { return InvalidationVerdict{}, true }` (formatted on separate lines in source).
- `trader/invalidation_resolver.go:84-89`: the opposite branch returns an `InvalidationVerdict` with `Invalidated: true`, `Anchor: e.Anchor`, and `Reason: e.Reason`.
- `trader/entry_gate.go:229-250`: calls `in.ScenarioInvalidation(in.CitedScenario)`; `case v.Invalidated:` ends with `return msg, true`.

The source itself states the shared contract at `trader/invalidation_resolver.go:13-17`:

> The evaluator that writes "🎯 scenario S1 → ≈invalidated @ 29285.00" lives at
> trader/auto_trader_levelstate.go:260 and calls kernel.EvaluatePlanScenarios.
> This resolver calls THE SAME FUNCTION with the same windowing, so the gate
> refuses on the verdict the system already published rather than on a second
> opinion that could drift from it — the void-parity lesson.

**[A] Replacing `ScenarioAnchor` globally reaches a second gate.** `kernel/min_sl.go:88` returns `ScenarioAnchor(s, ap.Doc.Levels)` to `MinSLAnchorFor`. The production caller at `kernel/engine_position.go:236-250` compares the stop with that anchor minus/plus tick clearance and returns `sl_too_tight` on violation. That is another A31 change even without editing the gate file.

**[B] Safe scope proposal requiring the owner's correction:** a read-side identity record can resolve and display the candidate price/zone/formation while preserving the current evaluation anchor, facts, status and gate inputs. Then the ID is authoritative for *which candidate the scenario records*, not for the level at which the existing rule judges invalidation or stop clearance. If D4 instead requires the evaluation itself to use the ID's price, A31 must be amended explicitly. This investigator did not choose between these scopes or build either.

## C2: there are different heuristics with different types and tolerances

**[A] W1's `ScenarioNearest` is a nullable SCENARIO ID on a touch row, not a resolved level price on `ScenarioEval`.** The direction of this link is *detected level touch -> scenario*. The evaluator's heuristic goes *scenario prose/FVG -> plan level price*. Calling both “ScenarioNearest” obscures the distinction.

| Reader | Input -> result | Rule at the running source |
|---|---|---|
| W1 `store.ResolveScenarioLink` | detected level price + scenario anchor prices -> `*string` scenario ID and basis | Exactly one scenario anchor within 3 points may link. Two within the band are unresolved; no nearest-wins tie-break. |
| `kernel.ScenarioAnchor` | scenario FVG/prose + plan levels -> `(float64, bool)` | FVG distal edge first; otherwise first price token in Trigger, then Invalid, that snaps to a plan level within 2 points. |
| W3 `kernel.MatchMapCandidate` | plan level price + current merged map -> candidate | Nearest candidate within map cluster width, 3 points. Unlike W1, multiple candidates do not produce W1's ambiguity basis. |

The W1 derivation is exact:

- `trader/scenario_anchor.go:19-26`: positive `Confirm.RefPrice` wins; otherwise positive, enabled `Arm.Entry`; otherwise unanchorable. This does **not** call `kernel.ScenarioAnchor` and does not parse Trigger/Invalid.
- `trader/scenario_anchor.go:54`: `func scenarioLinkBand() float64 { return float64(kernel.LevelClusterTicks) * 0.25 }`.
- `kernel/levels_score.go:737`: `const LevelClusterTicks = 12`; `clusterToleranceFor` at lines 741-743 returns `LevelClusterTicks * 0.25`. Thus the dispatch's 3-point W1 band premise is correct.
- `store/scenario_link.go:68`: `if band > 0 && d <= band { inBand++ }` (three lines in source).
- `store/scenario_link.go:76-85`: `inBand > 1` -> ambiguous; `bestDist > band` -> outside band; otherwise stores `anchors[best].ID` with `ScenarioLinkPriceProximity`.
- `trader/detector_record.go:111-115` is the production writer: calls `ResolveScenarioLink(lv.Price, anchors, delta, scenarioLinkBand())`, assigns `row.ScenarioNearest`, `row.ScenarioLinkBasis`, and both distance fields.
- `trader/auto_trader_planner.go:2416-2434` reads the latest plan, derives anchors with `scenarioAnchorsFrom(doc)`, and calls `recordDetectorOutputs` with them. Lines 2445-2447 explicitly say the recorded version is the version in force, not the version the read is about to author.

W1's full basis vocabulary, `store/scenario_link.go:29-32`:

```
price_proximity
unresolved:two_scenarios_within_band
unresolved:nearest_outside_band
unresolved:no_scenario_at_seat
```

An empty or NULL basis must be separately reported if present; the report at `docs/superpowers/reports/2026-09-10-episode-contract.md:160-164` says the empty string marks a pre-wave row. Do not silently coalesce it into one of the four measured reasons.

The evaluator's distinct rule, `kernel/scenario_state.go`:

```
:72       const anchorTolerance = 2.0
:90-94    FVG long -> Fvg.Lo; otherwise Fvg.Hi
:96       for _, text := range []string{s.Trigger, s.Invalid} {
:115      bestDiff := anchorTolerance
:120-121  if d := absF(l.Price - v); d <= bestDiff {
              best, bestDiff, found = l.Price, d, true
```

Its design comment at lines 19-22 is still present as quoted in the dispatch. Its “carries NO price” wording is stale as an absolute field claim: the current code immediately handles structured FVG prices, while W1 reads structured confirm/arm prices. The precise supported premise is **no candidate-level identity**, not no numerical price anywhere in a scenario.

## W1 third-boot cutoff and the correct timestamp column

**[A] The exact third-boot time is in the deployment marker's commit message, not the cited report.**

`git show 580e88b39412f6b8cb77021f31651509ccfc1e4d`:

```
marker: boot adae3bb4 — W1 follow-up, the other three items WIRED
Booted 16:51:28 CDT on owner's explicit order, mid-session.
🔐 BOOT INTEGRITY OK — rev adae3bb41b31 +dirty · built 2026-09-10T21:09:02Z
   · expected adae3bb4 · goldens PASS
```

Full built revision: `adae3bb41b3151e12c9c7e0553f522860fccbea4`.
Cutoff: **2026-09-10 16:51:28 CDT = 2026-09-10T21:51:28Z = 1789077088000 milliseconds**. The commit's own time is 16:52:52 CDT; the built revision's commit time is 16:09:02 CDT. Neither commit time is the boot time. This is [A] as a quotation of the marker, not an independent journal verification by this investigator.

**[A] Use `touch_outcomes.created_at` for rows WRITTEN since this boot.** `opened_at_ms` describes an episode on the historical tape being scanned. At `trader/detector_record.go:104`, episode times come from `e.OpenedAtMs` / `e.ClosedAtMs`; `store/touch_outcomes.go:183-185` sets `CreatedAt = time.Now().UTC()` when not supplied. Filtering `opened_at_ms >= boot` would discard rows actually recorded under the third binary merely because their historical episode was earlier.

Source schema (`store/touch_outcomes.go`):

- table `touch_outcomes` (`TableName`, line 141);
- `id`, `trader_id`, `symbol`, `level_price`, `level_kind`, `plan_id`, `plan_version`, `session`;
- `opened_at_ms`, `closed_at_ms`, `formed_at_ms`, `created_at`;
- `scenario_nearest`, `scenario_link_basis`, `scenario_link_dist_pts`, `scenario_link_dist_delta`.

Parent census should give each basis's count and row IDs plus the total, using a date comparison that respects SQLite's stored timezone representation. It should also report the as-of timestamp and boundary inclusivity. This source investigation intentionally supplies no live counts.

**[A] Basis-report mismatch:** at `770e2297`, the cited `2026-09-10-episode-contract.md` contains no section titled “next wave's basis,” no `adae3bb4`, and no third-boot timestamp. Its latest file commit is `5dd0e8c7` at 13:29:04 CDT. It does contain the W1 heuristic explanation at lines 64-90 and the first-boot wiring correction in §J. The dispatch's claimed verbatim section should not be falsely quoted as text read from that report; source fields and the marker supply the independently checked evidence.

## D4/D5 consumer map and additive-recording feasibility

**Evaluator / stored card status [A].** `trader/auto_trader_levelstate.go:221-222` calls the shared evaluator; lines 237-240 persist its status map using `store.ScenarioStatusKey`; lines 247-265 persist basis/unevaluable/confirm metadata; lines 271-279 record scenario-death evidence with the evaluator anchor. An additive identity field on the evaluation record can be carried here, but changing the existing `Anchor` or `Status` is the gate conflict above.

**Episode opener [A].** The sole W1 row writer is `trader/detector_record.go:91-116`. Its `lv` is the seated detected level; the W1 link is a separate scenario-ID heuristic. A nullable `level_id` can be added to this row without replacing `ScenarioNearest` or its basis. D5 says the new column stays NULL until a scenario names the candidate, which means “detected candidate has an ID” and “an authored scenario named it” must not be conflated.

**Episode closer [A].** `store/opportunity_close.go:87` passes `rows[i].ScenarioNearest` into `factsFor`; line 94 passes the same scenario ID to `entryFor`. `trader/episode_close_wiring.go:62-107` keys those callback maps by `sc.ID` from the active plan. Candidate IDs cannot be substituted into these callbacks directly: they are a different key type. The current closer is recording-only (`:16-17`).

**[B] D5 cardinality needs an explicit rule.** A candidate ID identifies a candidate, not necessarily a unique scenario; two scenarios could name the same candidate. Therefore `level_id -> candidate` alone cannot replace `scenario_nearest -> scenario` for opportunity facts. Any follow-up must resolve the row's exact plan/version and distinguish zero, one, and multiple naming scenarios without silently selecting the first. The source inspection proves the key-type difference; no live multi-scenario example was measured here.

**Scenario card [A].** `api/handler_plan.go:423-425` resolves the old heuristic anchor before reading `ScenarioDeathFor`. `web/src/components/plan/ScenarioList.tsx:369-394` consumes the stored scenario status and meta basis, and displays recorded death anchor. It does not consume `ScenarioNearest` (a touch-row field) or run its own level resolver. Its tooltip explicitly names ±2 points at line 379. New candidate identity should arrive as server-resolved metadata if the one-Go-resolver constraint is retained.

**Level table on the card [A].** `api/handler_plan.go:523-527` builds the current W3 map once per request. At lines 575-595 it enriches each plan level through `kernel.MatchMapCandidate(w3Map, l.Price)`. `kernel/map_candidates.go:497-509` chooses the nearest candidate inside `clusterToleranceFor(price)`. This is a separate display heuristic from W1.

**Desk strip [A].** `trader/desk_facts.go:151` routes the SCENARIOS row to `deskScenarioEconomics(now)`. `trader/scenario_economics_desk.go:12-21` reads the active plan and renders `kernel.EconomicsSummary(s)` / `EconomicsFor(s)`; it currently performs no “which level” resolution. Additive candidate metadata is possible here without changing economics or trading rules. The dispatch explicitly assigns the other lane its fade stamp; coordinate before overlapping this row.

**Other shared-reader blast radius [A].** `kernel/plan_render.go:299` uses `ScenarioAnchor` for citation structure; `kernel/plan_confirm.go:120` uses it for FVG stale-confirm annotation. These are further reasons not to replace the helper globally on a “recording/schema only” claim.

## Source freshness receipts

Each line below is the actual result of `git log -1 770e2297 --format='%H %cI %s' -- <file>`. All source excerpts were read with `git show 770e2297:<file>`. A `git diff 770e2297 --` of the investigated evaluator/episode/gate/card/desk files was empty in the claimed worktree before this artifact write. The tracked `docs/superpowers/CLAUDE-canon.md` was read; its keeper instruction supersedes obsolete manual-heartbeat prose.

- `docs/superpowers/CLAUDE-canon.md`
  - `290044296c482afdee04acd740d189b89bfd040d 2026-09-10T11:55:22-05:00 docs(canon): two different rc 3s sat on adjacent lines`
- `docs/superpowers/reports/2026-09-10-episode-contract.md`
  - `5dd0e8c7f9ef32038bf5f2574e63b2f02b84633c 2026-09-10T13:29:04-05:00 docs(W1 17/n): §G4 corrected — it was the wall clock, not interference, and I had the disproof`
- `kernel/scenario_state.go`
  - `2eaf7ab59ff1cf89ce88d0317d71a3f3390eff74 2026-08-26T15:21:56-05:00 FVG entry model — 5th scenario condition (pure-math play) (#79)`
- `store/scenario_link.go`
  - `c35dfecb2e11cbc763174516983919885cf0b4cc 2026-09-10T11:31:42-05:00 feat(W1 2/n): the touch → scenario link as a HEURISTIC, and the boot line that names its resolver`
- `trader/scenario_anchor.go`
  - `9dd15a5378ae17bdaa7dd7c2ab70142b82744db4 2026-09-10T11:31:42-05:00 feat(W1 3/n): the scenario anchor, and the link band read from the map itself`
- `trader/detector_record.go`
  - `c6f75756f3e54a56646ca3cfa9c86f116b5d4541 2026-09-10T11:31:42-05:00 feat(W1 4/n): wire the link — the recorder stamps it, the gate keeps it wired`
- `trader/auto_trader_planner.go`
  - `c6f75756f3e54a56646ca3cfa9c86f116b5d4541 2026-09-10T11:31:42-05:00 feat(W1 4/n): wire the link — the recorder stamps it, the gate keeps it wired`
- `store/touch_outcomes.go`
  - `c35dfecb2e11cbc763174516983919885cf0b4cc 2026-09-10T11:31:42-05:00 feat(W1 2/n): the touch → scenario link as a HEURISTIC, and the boot line that names its resolver`
- `store/opportunity_close.go`
  - `31e6a13db2af8eb99fb6dc1655c3b943f30b3516 2026-09-10T13:19:20-05:00 fix(W1): wire the other three — and the census that would have caught them`
- `trader/episode_close_wiring.go`
  - `31e6a13db2af8eb99fb6dc1655c3b943f30b3516 2026-09-10T13:19:20-05:00 fix(W1): wire the other three — and the census that would have caught them`
- `trader/invalidation_resolver.go`
  - `9c754e369f397d04a462fbcbefb47c2feef37ab0 2026-09-08T08:56:19-05:00 fix(plan-liveness): bind death evidence to version and validate authored closes`
- `trader/auto_trader_levelstate.go`
  - `6f677b55daa1c7da33b8c35f8bcc67883f36b470 2026-09-08T18:01:58-05:00 test(w7): freeze consumed-level clocks and add behavior-preserving clock seam`
- `kernel/min_sl.go`
  - `4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep`
- `kernel/engine_position.go`
  - `0b8b41c3704570eccfc4d918d26cbc880ac4c897 2026-08-27T00:02:52-05:00 fix4+8k [F2]: honest C6 — GateRefusalError never parse-retried, planless prompt warning, risk_check_error logged; [F6] executor bias_ctx PDC from full universe`
- `kernel/plan_render.go`
  - `b71d424c0435806dbb30f8a1845ac817f8834e7f 2026-09-02T23:48:30-05:00 docs+fix(hygiene): the owed small fixes — checklist backfill, bias aliases, no-trade NOTES, guide corrections to RESOLVED values`
- `api/handler_plan.go`
  - `35fa69cc8b54d3493f544fe9ca5e7c075f282b99 2026-09-09T18:56:53-05:00 candidates(D6 card): the merged map reaches the plan card`
- `kernel/plan_confirm.go`
  - `b99963857c453f909570eba7a854a8ec23eed087 2026-09-08T14:30:06-05:00 test(confirmation): pin the ruled carve-out and publish mutation evidence`
- `trader/entry_gate.go`
  - `01ce808839becd61120140e178bffa7cbc225d30 2026-09-05T12:12:00+00:00 fix(risk,planner): wire RiskForceFlat and BiasArmWarning — both shipped uncalled`
- `trader/scenario_economics_desk.go`
  - `d5e2414e0d30a275b6239f5d609f9e22f8e381d7 2026-09-08T16:02:39-05:00 feat(scenario-economics): enforce new authoring contract and preserve legacy unknowns`
- `trader/desk_facts.go`
  - `14e1cbcba0bf3ab51ca42fe15a81e348f9e90257 2026-09-09T14:10:38-05:00 fix(risk): D4(c) the CME roll lifts the daily trip; desk strip needs BOTH toggles`
- `web/src/components/plan/ScenarioList.tsx`
  - `0533c8d052e8f871af26d8725522a019d0ebafe1 2026-09-08T16:19:16-05:00 test(scenario-economics): pin write boundary, legacy parity and counter mutations`
- `kernel/levels_score.go`
  - `32c1cfb5c198cbd2fed90d9de155e1b71a4cfcd1 2026-09-10T13:03:38-05:00 W-TF D6: the boot line, the detector table, and the docs in the same commit`
- `kernel/map_candidates.go`
  - `35fa69cc8b54d3493f544fe9ca5e7c075f282b99 2026-09-09T18:56:53-05:00 candidates(D6 card): the merged map reaches the plan card`
