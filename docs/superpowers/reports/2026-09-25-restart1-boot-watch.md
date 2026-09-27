# RESTART 1 — BOOT WATCH SHEET

**Author:** DS-105 · **Branch:** `docs/restart1-boot-watch` · **Base:** origin/dev `39829e65`, refreshed against **final restart-1 dev sha `9c106d0b`** (all citations re-grepped at that sha)
**Scope:** what attended restart 1 (~07:00 CT, 2026-09-25) ships — ALL merged on dev `9c106d0b`: #215 #217 #214 #219 #220 #197 #221 #212 #216 #222 #213 (and the earlier #197 #207 #208 #209 #210 #211).
**Law:** every proof line below is READ from the code at the cited ref (file:line), never guessed. All citations re-grepped at `9c106d0b`. SIM-only throughout.

---

## Not in this boot (open at `9c106d0b`, intentionally NOT shipped by restart 1)

- **#218 W117-A (fix/w117-a-exec-evidence)** — NT8 execution-evidence ownership: F1 + F4 + F5 in this PR; **F2/F3 moved to a rebuild PR** (the "F2/F3 rebuild" the CTO named). Watch for it after the boot, not at it.
- **#224 W117-B2 (fix/w117-b-cancel-truth follow-up)** — the boot sweep requests the cancel and settles only on a persisted broker snapshot (F8's last caller).
- **#225 W117 PR-C (placement/commit semantics)** — F13, F13b, F14, F15.
- **#206 M4 3b-B (feat/one-button-m4-worker)** — the updater worker (`nofx-updater`) + API glue behind `NOFX_UPDATER`, default OFF.

None of these four is live in restart 1; do not look for their evidence in this boot.

---

## 1. #197 — W1a-plan: owed plan/planner folds (merged)

- **Live change:** planner folds P1–P15/N2/N5/T2/T3 — repair order, zone-accepted identity, clock seams, bias-coherent arm handling. Runtime behaviour changes on the next planner read.
- **Proof line (dev `9c106d0b`):** `🧭 bias-coherent arms: %s (WARN — write proceeds; owner ruling 2026-09-04 is warn-first)` — `trader/auto_trader_planner.go:2345`; `🧭 role mismatch: %s` — `:2405`.
- **Knobs:** none new (folds only).
- **First evidence:** the first post-boot planner read either prints those 🧭 WARNs (if owed conditions apply) or proceeds silently; read output rows appear in `planner_rejected_prompts` / plan tables as before.

## 2. #207 — CENSUS-AUTH (merged)

- **Live change:** TEST-ONLY. Three security censuses (users-writer, mint sites, updateauth constant folding) see what compiles. Nothing changes at runtime.
- **Proof line:** none at boot; the proof is CI: `go test ./auth/ ./store/ ./internal/updateauth/` — `TestUsersWriterCensusCatchesEveryWriterShape`, `TestFutureIatCensusCountsTokenLiteralAndFunctionValueMinters`, `TestUnscopedMintCensusCountsTokenLiteralAndFunctionValueMinters`, `TestUpdateAuthCensusFoldsSiblingFileConstants`, `TestUpdateAuthCensusFoldsExportedConstantsOfAnotherPackage`, `TestUpdateAuthCensusRefusesAModuleWhoseGoListCannotRun`.
- **Knobs:** none.
- **First evidence:** GitHub checks green on the restart branch; nothing to watch live.

## 3. #208 — CENSUS-GUARDS (merged)

- **Live change:** TEST-ONLY. Checklist-numbering, knob method-reader (receiver-typed) and census-walk (symlink-refusing) guards. Nothing changes at runtime.
- **Proof line:** none at boot; CI: `go test ./kernel/ -run Checklist ./store/ ./internal/censuswalk/` (`TestChecklistNumberingCountsEveryClassSpelling`, `TestKnobMethodReaderSitesAreReceiverTyped`, `TestNonTestGoFilesRefusesSymlinkedPackageDir`).
- **Knobs:** none.
- **First evidence:** GitHub checks green.

## 4. #209 — PR B: release-workflow folds (merged)

- **Live change:** DEPLOY-TIME (no bot runtime). Release workflow runs no longer carry user-influenced expressions in `run:` bodies — `TAG`/`RELEASE_ID` travel via step `env:` and are read as quoted data.
- **Proof line:** CI: `TestReleaseWorkflowRunBodiesCarryNoUserInfluencedExpressions` — every `${{ }}` in a `run:` body must be in the allow-list (`steps.src.outputs.sha` only).
- **Knobs:** none.
- **First evidence:** the next release workflow run executes the rebuilt steps (manifest upload, tar ×2, `gh release create`, asset path) with env-carried values; no GitHub "unrecognized named-value" errors.

## 5. #210 — Boot-2 safety folds (merged)

- **Live change:** `deploy/cutover.sh` reconciles OLD_SHA, requires every `trader_cutover:*` + `ledger_exposure` + `planner_in_flight` + `traders_nt8` installation-gate leg (+ `addon_census_prehold` when present), token travels via header file not argv, no kill on a staging failure; web Updates badge/page: 403 not-enrolled backs the 60s poll off to 15 min and the page stops polling with a refusal note.
- **Proof line:** cutover script greps — the leg names and `@$TOKEN_HDR` in `deploy/cutover.sh`; web: `updatesStatus` returns `{status, statusCode}` (badge test pins the 15-min backoff).
- **Knobs:** none.
- **First evidence:** next update attempt — badge settles to the not-enrolled note and is not re-polled at 60s; a cutover run prints every gate leg.

## 6. #211 — W117-E: authenticated chat model isolation (merged)

- **Live change:** every authenticated chat request runs on a request-local `*Agent` (`agent/request_runtime.go`) so two users' model credentials can no longer clobber each other; an authenticated user with no enabled model gets NO client (no silent fallback to another account's key).
- **Proof line:** `agent/request_runtime.go:21` (`requestRuntime`); `HandleMessage`/`HandleMessageStream` first line `a = a.requestRuntime(storeUserID)` — `agent/agent.go:433` / `:480`.
- **Knobs:** none (behaviour fix).
- **First evidence:** two concurrent authenticated chats with different models each answer with their own model; a user with no enabled model gets the refusal, not another account's model.

## 7. #214 — W117-H: frontend truth + isolation (merged)

- **Live change:** WEB ONLY — unknown-order snapshots (F31) and stream/modal session fencing (F32) in the chat UI.
- **Proof line:** vitest: `served-by-model-alice after tool`-style stream-fencing pins in the web suite.
- **Knobs:** none.
- **First evidence:** chat UI renders unknown order states; a stream/modal can no longer leak another session's answer.

## 8. #215 — W117-D: API authz (merged)

- **Live change:** the ownership middleware now sweeps EVERY selector location on EVERY protected route (not only plan/risk prefixes); plan overlay updates carry revision fields and refuse stale revisions (HTTP 409); `planMutationSessionAt` resolves before dereference with the wrap-aware chain date; chat handlers ignore the caller-supplied `user_id`.
- **Proof line:** HTTP behaviour, not a log line: stale overlay revision → 409; cross-owner id → 403; CI `api/ownership_all_selectors_test.go`, `api/plan_overlay_revision_test.go`.
- **Knobs:** none.
- **First evidence:** any cross-owner API request now 403s; a stale overlay edit returns 409 instead of silently winning.

## 9. #217 — W117-G2: swing provenance + successor stamp (merged)

- **Live change:** swing pivots carry the defining candle's open (`pivotOpenMs`) so the wick lookup answers with the real pivot bar, and aggregation seeds each bucket with its first bar's volume; an armed-row successor is re-stamped with the new boot id and `ArmedUnderVersion`.
- **Proof line:** pins, not logs: `TestSwingZoneWickComesFromSelectedPivot` (was FAIL "selected pivot wick=5, got 0xc…"), `TestSwingAggregationConservesFirstBarVolume` ("aggregation lost source volume"), `TestArmedZoneReArmPinnedWithinVersion` ("successor lost authorization provenance: boot=\"\" armed_under=0") — `kernel/levels_swing.go`, `store/armed_orders.go`.
- **Knobs:** none.
- **First evidence:** next re-arm mint of a successor row shows `boot_id` = the restart's boot id and `armed_under` = the new version (DB `armed_orders`); swing wicks render at the defining bar.

## 10. #219 — W117-F: position lifecycle fences (merged)

- **Live change:** delayed-flatten Stop fence and trader-scoped account evidence (F6–F7): a reconcile before an open refuses or flattens an unexplained NT8 position instead of compounding onto an orphan.
- **Proof lines (dev `9c106d0b`):** `⛔ reconcile-before-open: NT8 holds a %s %s that the ledger explains (%s) — refusing the %s open` `trader/auto_trader_orders.go:303`; `🚨 reconcile-before-open: … flattening first` `:306`; `✅ reconcile-before-open: %s flatten fill-confirmed via position_close frame` `:333`.
- **Knobs:** none.
- **First evidence:** if NT8 ever holds an orphan at open time, one of those three lines fires instead of a compounding entry.

## 11. #220 — W117-I: dependency security bumps (merged)

- **Live change:** `golang.org/x/crypto` 0.55.0, `gnark-crypto` 0.19.2, 4 npm transitives.
- **Proof line:** `go version -m nofx-bin | grep -E 'x/crypto|gnark-crypto'` shows the new versions.
- **Knobs:** none.
- **First evidence:** the binary's module list carries the bumped versions.

## 12. #212 — DEFAULTS-SANE (merged)

- **Live change:** Picture HTF entry window default **90 → 360 s**, freshness default **15 → 30 s**; the evaluator's silent decision points now WARN once per (stage,reason); `liveFrameMaxAgeMs` exported as `LiveFrameMaxAgeMs` (sink admission bound, unchanged value 30 s).
- **Proof line:** the 📷 boot line — `picture-htf: mode=%s rule=v1 … window=%ds fresh=%ds · foreign=… · stale=…` (`trader/picture_htf_live.go:206`); it READS the resolver, so a live bot prints `window=360s fresh=30s`. The floor pins: `TestPictureHtfDefaultWindowExceedsWorstDetectionToPlacement`, `TestPictureHtfDefaultFreshnessAdmitsEverySinkAdmittedFrame`.
- **Knobs:** `day_plan.picture_htf.entry_window_sec` (default **360**, 0 = default), `day_plan.picture_htf.freshness_sec` (default **30**). Effective value: the boot line itself, and the Studio effective-values table (`EffectiveMount` rows).
- **First evidence:** post-boot 📷 line shows `window=360s fresh=30s`; on the next hour-open storm, `picture-htf: … stale=…` stays low and `picture_htf_opportunities` can finally produce rows (it was 0 ever before).

## 13. #213 — Planner Lane B: executor arm-reach (merged)

- **Live change:** B1 a far `market_in_zone` arm waits armed-unplaced beyond `zone_place_within_pts` and places when price comes inside; rest-cap expiry goes to cancel_pending with the signal kept and re-arms only on broker-book confirm. B3 one 🧭 per-read line + `planner:read*` counters. B4 `planner_rejected_prompts.response_text` stores the raw refused answer; zone-cap provenance shows the shipped cap 10.
- **Proof lines (dev `9c106d0b`):** `🧭 planner read: session=%s attempts=%d reject_classes=%s read→publish=%s lifecycle=%s` (`trader/auto_trader_planner.go:1828`); counters `planner:read` (`:1840`) / `planner:read_reject_<class>` (`:1846`).
- **Knobs:** `day_plan.zone_place_within_pts` — nil → **25** (ON, shipped), 0 → OFF (legacy byte-identical); effective via resolver `ResolveZonePlaceWithinPts`. `zone_rest_max_min` unchanged (30).
- **First evidence:** the 🧭 line on the next planner read; a far arm resting beyond 25 pts stays armed-unplaced (DB `armed_orders.state`), and a rest-cap expiry first shows cancel_pending with its signal id, then armed-unplaced with a NEW signal only after the book confirms.

## 14. #221 — Planner A6: born-dead retry on the fresh tape (merged)

- **Live change:** when the validator refuses born-dead / flip-met, the next attempt (REPAIR and RE-AUTHOR) appends a FRESH TAPE block — completed 1m/5m closes between the read clock and the refusal, the breached condition verbatim, never the forming bar.
- **Proof line:** the block's header — `## FRESH TAPE SINCE YOUR READ (previous attempt refused born-dead / flip-met)` (`kernel/planner_fresh_tape.go:33`); the refusal WARN now records read→publish latency (B3's 🧭 line reads it).
- **Knobs:** `day_plan.planner_fresh_tape` — `PlannerFreshTape *bool json:"planner_fresh_tape,omitempty"` (`store/strategy.go:1047`); nil = **ON** (shipped default); explicit false = today's blind retry, byte-identical (pinned by `TestPlannerBornDeadRetryKnobOffIsByteIdenticalToday`). Effective via effective-settings truth row.
- **First evidence:** a born-dead rejection's attempt-2 row in `planner_rejected_prompts` carries the FRESH TAPE block; the 🧭 line's `read→publish` field is non-empty for that read.

## 15. #222 — planner-contract (merged)

- **Live change:** A1 authored-entry-zone geometry: a null-width map line no longer admits a zone without provenance (`no_provenance`), and `authoredEntryZoneBand` caps the authored band at `store.ZoneMaxPtsDefault` (10.0).
- **Proof lines (dev `9c106d0b`):** refusal class `entry_zone_edges_or_provenance_unusable` — `trader/structural_geometry.go:154`; the cap constant `ZoneMaxPtsDefault = 10.0` — `store/resolve_source.go:220`.
- **Knobs:** `day_plan.planner_contract` — `PlannerContract *bool json:"planner_contract,omitempty"` (`store/strategy.go:996`), nil = **ON** shipped default; `PlannerContractOn()` (`store/strategy.go:2847`).
- **First evidence:** a plan with a null-width zone map line is refused with the geometry class instead of trading on unproven geometry; the guide knob card shows the contract ON.

---

## Known limits shipping (state, not defect)

- **#212 Picture freshness vs the fallback tick:** the 2-min wall-clock fallback evaluates with STALE freshness stamps, so with freshness 30 s the fallback tick can never enter — it is a recovery of CACHE state only. A dropped boundary frame is recovered by the SUCCESSOR 5m frame (~301 s into the window), which is exactly what the 360 s window is sized for. The fallback's refusal is counted, never silent.
- **#221:** the born-dead CHECK itself is unchanged; only the retry carries the tape. Historical `planner_rejected_prompts` rows predate `response_text` (from #213) and have no raw answer — the replay says so.
- **#213 B1:** `zone_place_within_pts` is resolved at arm time and re-resolved at confirm time; a knob change mid-flight takes effect on the next resolution, not retroactively.
- **Test-only waves** (#207, #208) add no runtime surface — their "boot evidence" is CI, by design.
- **Frontend waves** (#214, #210's badge half) are verified in vitest; live proof is the UI behaviour, which has no server-side log.
- **No `-race` runs were performed by these lanes** — race verification is on the CTO's slot.
