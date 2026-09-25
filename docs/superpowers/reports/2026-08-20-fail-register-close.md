# FIX-ALL WAVE — the unified FAIL register closed (2026-08-20)

**Branch:** `fix/fail-register-close` (base = #58 stack head `029ccb85`; RELEASE==running binary verified at start: `39554ea8`).
**Deployed:** `4e32cd5a` at **01:11:05 CT** in a verified flat window (0 open rows, ASIA quiet). Boot: `🔐 BOOT INTEGRITY OK — rev 4e32cd5a … expected 4e32cd5a · goldens PASS` — the embedded golden self-check validates the NEW prompt contract (cite + D2 lines).
**Scale:** 26 commits · 53 files · +1,357 −220. Full `go test ./...` 27/27 ok · named suites #45–#58 green · FE **263/263 — fully green for the first time** · `npm run build` + `go vet` clean.

## Phase 0 — triage (all 19 anatomy rows re-verified at HEAD)

| id | reproduced? | disposition |
|---|---|---|
| F1/F3 (prose triggers, no confirm field) | Y | **C1** structured confirm{} |
| F2/F8 (dot heuristic, anchor mining) | Y | **A1+A4** |
| F4 (5m_close phantom) | Y — `conditionRule` still mapped it to 2x5m | **A2** |
| F5 (object↔prose unchecked) | Y | **A3** |
| F6 (strict = direction-only) | Y | **B3** |
| F7/F9/F10 (advisory state / target_chain / quality) | Y | **D1–D3** rulings implemented |
| F11 (id format) | Y | **A5** |
| F12 (cite absent from contract) | Y — zero `cited_scenario` in the futures contract | **A6** |
| F13 (dead activePlanIsDead) | Y (3 refs) | **A7** |
| F14 (silent overlay fallback) | Y at read-merge + planner re-validate (write path was already loud) | **A8** |
| T3 (three TF tables) | Y | **B1** |
| T5 (C2 post-call clock) | Y | **A9** |
| T6 (feed asymmetry/blindness) | Y | **B2** |
| T7/T8 + stale comments + dup gates + legacy stopUntil | Y | **A10+A11** |

**v1 master-audit delta (22 findings / 6 unproven):** CLOSED before this wave — level_state trader-scoping (P0-cleanup), B3/B4 fixtures (order_guard/stale tests exist), fill alerts (7 live rows), **T1 calendar fail-open (P0.6 fail-closed + static fallback + P0 alert — already shipped)**. CLOSED THIS WAVE — chat-path Time(UTC) table (now CT), UI confidence fallback 75→60, no-rev-in-UI (status API `revision`), contracts hint 10→2 (was #55's 6.4). STILL-OPEN → found-not-fixed: dead wires (grid_* store, /plan/approve, top-traders FE calls) [M] · MAE/MFE never rendered [M] · duplicated default constants [S cosmetic] · gate-blocks memory-only (mitigated by the B6 journal line) [M] · plan grades model-written display [S] · no-data-not-in-prompt [S] · digest producer unproven [M]. UNPROVEN (6): unchanged — live-proof class items for Audit V2.

## Per-fix ledger (id → commit → test)

| id | commit | fix | test |
|---|---|---|---|
| A2/F4 | `239cd7ab` | 5m_close evaluates as authored (one close); death log names the fired rule | `TestConditionRuleAsAuthored` + `TestFiveMCloseDeathFiresOnOneClose` (+2x5m/15m regression pins) — **V3 satisfied at unit level** |
| A9/T5 | `6119b7d5` | C2 drift = snapshot clock vs snapshot bars; stale "refuses entries" comment fixed | `TestClockDriftIgnoresAICallLatency` (200s mocked call ⇒ drift≈0 at the snapshot clock) |
| A7+A11 | `f6a59c8d` | dead `activePlanIsDead` + the producer-less legacy `stopUntil` field/consumer deleted — ONE pause remains | build + pause/gate-order suites |
| A6/F12 | `dc68f4ee` | `cited_scenario` in the futures output contract + schema-retry enforcement when a plan is active; goldens regenerated (cite lines only) | `TestRequireCitedScenario` (uncited open → retry; off-plan valid; waits exempt) |
| A1+A4 (F2+F8) | `e5b68e25` | verdict BASIS (machine when anchor==structured death/flip price, heuristic otherwise) rendered distinctly; NO-status scenarios render "unevaluable ?" instead of a fake armed; token mining widened past the 4-digit floor (unit-suffix filtered in code — RE2 has no lookahead); unevaluable WARNs to log_events | FE P4_3 contract updated (unknown ≠ armed) + kernel suite |
| A3+A5 (F5+F11) | `27b357ff` | structured death/flip price MUST match its prose twin (validator, planner retries); prose-only death labeled "(prose-only — AI-judged)" in prompt + WARN at write; ids enforced S0..S99 (S0 = the Go no-trade stub); miner magnitude floors removed with a price-band guard in `continuationReachable` | `TestDeathFlipProseCrossCheck` · `TestScenarioIDFormatEnforced` |
| A8/F14 | `6cffb5e8`+`5d5cdd00`+`4c9bd78c` | read-merge SKIPPED overlays WARN + `overlay_errors` on plan-today; planner fallback-to-base WARNs the edits are NOT active (write path already rejects loudly + #58 preserves input) | api suite; write-path rejection pre-existing |
| A10/T7 | `53da0e86` | bridge forming-bar comment truth; kernel CME gate demoted to assert-only backstop (loop primary); roll's two layers documented complementary | suites |
| B1/T3 | `cf51baf7` | ONE TF table (`kernel/timeframes.go`), three consumers delegated, unknown→60s fabrication gone, unmapped primary TF = named BOOT FAIL | `TestOneTimeframeTable` (3m/30m coverage) |
| B2/T6 | `001762ca` | one feed policy (FEED_ALERT_S=600 flat / INTRADE_FEED_ALERT_S=120) consumed by B4 + alerts; any-subscribed-TF fallback kills the {15m,1h}-only blindness; 1m/5m per-period contract stays primary | `TestFeedPolicyThresholds` · `TestStaleEntryFeedAnyTFFallback` |
| B3/F6 | `ad5a5eb8` | strict adherence = direction + activation-band + SL/TP structure; off-band/struct-inconsistent cites grade **B not A**; forward-only (`plan_band` additive column; legacy "" keeps old grades) | `TestStrictAdherenceBandGrades` (the spec case) · `TestCitationStructure` |
| C1/F3+F1 | `a8096585` | **structured confirm{rule, ref_price, side} REQUIRED per scenario** — planner contract + validator (A3 prose agreement); machine advisory lines in the executor prompt via the sanctioned ever-fired acceptance API (`AcceptanceRunEver`); MET/NOT-MET card chips via scenario_meta; `CONFIRM_GRACE_SESSIONS=3` dual-accept (compliance fast-forwards); **no new hard gate** | `TestConfirmRuleIdentity` (1x5m≠2x5m, A2-consistent) · `TestConfirmDetailCarriesLastClose` · `TestConfirmValidator` · `TestRenderConfirmLines` |
| D1–D3 | `130870bd`·`b335d844`+`2d8dae1f`·`7bdda06c` | advisory label + tooltips + contract guidance lines (goldens regen); anatomy-doc halves committed on the #56 branch (`5debb664`) | FE suite |
| E5 strays | `1a9f81c4`·`227ffa6e` | chat table CT + confidence fallback 60; `revision` on the status API (cached at boot assertion) | suites |
| E2 | `ad2d0115`+`b48cd2a6` | RISK_MAX_CONTRACTS_PER_ORDER removed end-to-end (grep-proven zero readers); NOTIONAL kept deprecated (has an API reporter → found-not-fixed) | fixture updated |
| E3 | `3e01b503` (+prior) | write-only trader prompt pair removed from code (fields, setters, manager + API feeds); columns stay, physical drop parked per ruling | suites |
| E4 | `f82a2418` | logo test asserted pre-rebrand alt text (component renders VL) — fixed; playwright e2e excluded from vitest — **FE 263/263** | the suite itself |
| E1 | `4e32cd5a` | Studio save honesty: success only on HTTP 200 (guardedCall), loud failure with edits preserved, "saved <time> CT · unsaved changes" beside the button — the Aug-19 toggle mystery is visible at a glance | tsc + suite + build |

**Deviations (found-not-fixed / noted):** A3+A5 share one commit (same validator block); A1 implements dot honesty as basis-marking rather than literally re-deriving from `PlanConditionFiredSince` (the machine-verdict path IS shared where a structured object exists — the ruling's intent); E2's NOTIONAL env kept (consumer exists); the v1 M-class strays listed above; the confirm-grace counter is store-wide (not per-strategy) — sufficient for a single-owner deploy.

## Cutover + V-evidence

Boot block (verbatim, all wave-relevant state): `position_mode=ai_watch (source: db) · watcher[70/2/2] · trailing=OFF · stale_dodge=on reeval_drift=0.25×ATR14 · post_exit_rescan=on delay=2000ms · guardrails=master=OFF (soft-audit only)` + `BOOT INTEGRITY OK — goldens PASS` (single stopUntil by construction — the legacy field no longer exists; the TF table is compile-time + boot-validated).

- **V1 — PASS, first attempt [RUNTIME]:** the 02:00 CT LONDON read (plan v1, 01:59:34) authored **confirm{} on all three scenarios** with NO grace WARN needed (S1/S2 `1x5m_close` @ 29688.5 above/below, S3 @ 29617 below) + structured death (2x5m @ 29541.5) + flip (15m_close @ 29688.5) — all through the new A3/A5/C1 validators; compliance fast-forwarded the grace window. The stored executor prompt renders the machine advisory verbatim:
```
Machine-computed confirmations (advisory — you remain the judge):
  S1 confirm: 1x5m close above 29688.50 — NOT MET (last 1x5m close 29639.75 (best run 0/1 closes above 29688.50 since plan birth))
```
Card chips render from `scenario_meta.confirm`.
- **V2** (live-cycle gate trace): cycle #1 at 01:11:05 deferred correctly on `no balance frame yet` (the AddOn reconnect gate — first gate in the documented order to fire); subsequent cycles clean, zero asserts from the demoted kernel CME gate. Full 87-step order intact apart from the documented dedups (kernel CME → assert-only; legacy stopUntil deleted).
- **V3** (5m_close single-close death): proven at unit level on aggregated real-shaped bars — `TestFiveMCloseDeathFiresOnOneClose` asserts ONE 5m close fires it, names `5m_close` in the reason, and the same bars do NOT fire `2x5m`.
- **V4** (overlay reject visible): the write path returns 400/409/422 with named reasons; #58's EditSheet keeps the owner's input and toasts on any failure (`AskPlannerPanel`/EditSheet vitest); the read-merge/fallback silences now WARN + surface as `overlay_errors`.
- **V5 — 65-min soak (01:11–02:16 CT), zero regressions [RUNTIME]:** 31 paid calls (~29/h, interval cadence intact) · 3 dodge deferrals→kicks at close+1s (#55 behavior intact) · discards 2/31 = 6.5%, BOTH quiet superseded-waits (free) → **0 lost entries, 0 legacy discards, 0 "Failed" badges** · 0 reeval refusals · 0 watch/post-exit instances (flat all soak — both self-evidence on first occurrence) · zero new refusal types, zero asserts from the demoted kernel CME gate, zero panics. Adherence non-D grading on cited entries pends the next cited entry (B3 is forward-only). **Owner-relevant soft-audit catch during the soak:** `daily loss would trip (realized today=-1589.00, limit=-450.00)` — the unarmed cage would have halted the session-day at −450 while actual realized ran to −1,589; the would-trip telemetry is doing its job — arming the master is one Studio toggle.

## Owner decision queue (near-empty, as demanded)

1. The v1 M-class strays (dead wires · MAE/MFE rendering · gate-block persistence · digest proof) — park for the maintenance window or say go.
2. Physical drop of the 8 deprecated trader columns — parked with backup per ruling.
3. `trailing_enabled` still OFF (yours), B7 still OFF (yours).

## PR

**See chat delivery** — number parsed from the `gh pr create` output URL.
