# Source provenance and production calls

Accepted base: `e81602bb5c4bacb237ae2921e0188f8aa1d752bf`. Source freshness below uses `git log -1 --format="%H %cI %s" e81602bb -- <file>`; candidate additions have no prior source-file commit. This manifest does not pretend an uncommitted candidate had its own SHA. Final content is pinned by the containing commit.

| File | Last source-file commit at accepted base |
|---|---|
| `api/handler_plan.go` | fc2293f301577e477cfec5fe0dfdb920fcf88aa2 2026-09-11T01:20:14-05:00 one-setup: surfaces — API one_setup payload (the seam's RECORD, never a re-evaluation; absent → switch only, scenarios empty), desk SCENARIOS line, plan-card OneSetupChip (absent = not evaluated, never allowed) + vitest, Guide (the one play [O], the follow recorded-only [T] with round 17's null and the cell criterion, map untouched, OFF), two settings cards, knob census 45→47 |
| `api/handler_plan_geometry.go` | New candidate file |
| `docs/superpowers/AUDIT-CHECKLIST.md` | 88f32aa90eacf7a6ff9dbd7478b276726c779e4d 2026-09-12T08:35:00-05:00 research(backtest 1): the fade at a zone does not pay — negative net expectancy on 4.4y MNQ under every fill assumption, 0/81 surface cells positive |
| `docs/superpowers/Codex-canon.md` | New candidate file |
| `docs/superpowers/SYSTEM-MAP.md` | 0aea0c2e6a11cd194973115bb4cf7b30b6af5f9c 2026-09-11T21:39:47-05:00 feat(history import, wave 101): years of named-contract MNQ history through the wire |
| `docs/superpowers/VL-TRADING-RULEBOOK-v1.md` | 8cfc739499c74f9567ff35a44c95b5e4fc4c2daa 2026-09-11T12:51:47-05:00 feat(level-zones): preserve full map with bounded nearest-cluster presentation |
| `docs/superpowers/research/2026-09-12-backtest-zone-fade/README.md` | 88f32aa90eacf7a6ff9dbd7478b276726c779e4d 2026-09-12T08:35:00-05:00 research(backtest 1): the fade at a zone does not pay — negative net expectancy on 4.4y MNQ under every fill assumption, 0/81 surface cells positive |
| `docs/superpowers/research/2026-09-12-stop-target-geometry/README.md` | 586d00ef1769ad7c740b051d00130da4de8ac22f 2026-09-12T11:30:05-05:00 docs: incorporate fresh structural-geometry backtest and qualify fill results |
| `kernel/level_zones.go` | 5e81db95c1ac9788fd26caa23fa4bfaee119dfff 2026-09-11T13:09:08-05:00 fix(level-zones): require known merge bounds and record pre-existing suite blocker |
| `kernel/one_setup.go` | f047696e09adb5f52b5d7f4cffd7f99dfb8cb8be 2026-09-11T00:55:52-05:00 one-setup: E0 map pin (green on the untouched tree) + the predicate, pure, takes now — E1/E3/E4 RED then GREEN |
| `kernel/plan_doc.go` | 8cfc739499c74f9567ff35a44c95b5e4fc4c2daa 2026-09-11T12:51:47-05:00 feat(level-zones): preserve full map with bounded nearest-cluster presentation |
| `main.go` | 2fb51ecc4f1a12d3b888538e314d3d12dfb0c438 2026-09-11T01:16:42-05:00 one-setup: D8 boot line (READ from the bound strategy and the counters; [O]/[T] labels) + D9 three-state backfills (verdicts at each episode's OPEN since W2's boot against the recorded pool read, permission from W2's stamp; follow-plans since W1's boot through the contract filter), wired in main.go |
| `scripts/mutate.sh` | 07b53e6500c18b2531ec30a78675393cf09caef4 2026-09-10T17:31:05-05:00 cleanup batch 1 — B5 the mutation harness that cannot fake a verdict, B6 the worktree-add check |
| `store/knob_registry_table.go` | 63eb0bb8cc40330e0b0f9811e362e67a265c3489 2026-09-11T01:54:13-05:00 B1: the knob registry resolves ACCESSOR-METHOD readers — seven candidate-unverified knobs were live |
| `store/strategy.go` | 69164e1e19c89849c7c9969742000d23f46e27b8 2026-09-11T00:59:35-05:00 one-setup: store side — episode columns (verdict + follow-plan, all NULL until known), stamp-once writer, price-proximity link, counters, knobs registered + resolver |
| `store/structural_geometry.go` | New candidate file |
| `store/structural_geometry_test.go` | New candidate file |
| `trader/arm_stop_anchor.go` | 4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep |
| `trader/armed_executor.go` | 6fe2c11c1b0dc0618ea029bf8e4eff7337829b02 2026-09-11T01:55:36-05:00 one-setup: the gap the first boot found, closed (owner ruling (b), scoped) — at placement time an authorization whose scenario is currently DECLINED is retired by ledger state with the owner's reason and its three verdicts, never placed, nothing at the broker; an allowed one still places (pinned on the loopback wire, RED first, 2 mutants killed); boot-line key fix folded; CLASS 121; A29 registered; MAPCHECK recomputed; SYSTEM-MAP + RULEBOOK |
| `trader/auto_trader_planner.go` | 8cfc739499c74f9567ff35a44c95b5e4fc4c2daa 2026-09-11T12:51:47-05:00 feat(level-zones): preserve full map with bounded nearest-cluster presentation |
| `trader/entry_gate.go` | 01ce808839becd61120140e178bffa7cbc225d30 2026-09-05T12:12:00+00:00 fix(risk,planner): wire RiskForceFlat and BiasArmWarning — both shipped uncalled |
| `trader/one_setup_boot.go` | 6fe2c11c1b0dc0618ea029bf8e4eff7337829b02 2026-09-11T01:55:36-05:00 one-setup: the gap the first boot found, closed (owner ruling (b), scoped) — at placement time an authorization whose scenario is currently DECLINED is retired by ledger state with the owner's reason and its three verdicts, never placed, nothing at the broker; an allowed one still places (pinned on the loopback wire, RED first, 2 mutants killed); boot-line key fix folded; CLASS 121; A29 registered; MAPCHECK recomputed; SYSTEM-MAP + RULEBOOK |
| `trader/one_setup_boot_test.go` | d8d4faf532e52ea49ba3b0612d0f17dc22020ec8 2026-09-11T01:49:08-05:00 one-setup: the boot line reads the counters under the PLAN's trade date (the seam's key) — the 01:45 boot derived a session-day date and printed 0 for a key holding 2; pinned; takes effect at the next boot |
| `trader/one_setup_golden_fixture_test.go` | b3bd88c74333ee53042c16615121519b5878de82 2026-09-11T01:12:59-05:00 one-setup: the seam — ONE call site (oneSetupVerdictsAt, once per cycle), ONE consult after the class-48 gate, D3 obstacle target composed before the R:R leg, D4 rank order + second_setup_waiting; the follow-plan recorder wired at the detector hook; E2 golden from the base commit |
| `trader/one_setup_retire_test.go` | 30d8c7221a496507edee45a450cadb133aa2ff04 2026-09-11T13:23:22-05:00 test: pin arm fixtures to one clock and record running-source parity |
| `trader/one_setup_seam_test.go` | d4ab87db8ad3445ccd76477b4321d7aec9597139 2026-09-11T01:29:22-05:00 one-setup: five pre-existing wide-book fixtures (fvg_entry, sweep_reclaim split, R:R at the seam, loopback placement, config flip) declare one_setup OFF — their subject is not one-setup and E2 proves OFF is today's book byte for byte |
| `trader/shadow_demotion_test.go` | 30d8c7221a496507edee45a450cadb133aa2ff04 2026-09-11T13:23:22-05:00 test: pin arm fixtures to one clock and record running-source parity |
| `trader/structural_fixture_test.go` | New candidate file |
| `trader/structural_geometry.go` | New candidate file |
| `trader/structural_geometry_boot.go` | New candidate file |
| `trader/structural_geometry_boot_test.go` | New candidate file |
| `trader/structural_geometry_wire_test.go` | New candidate file |
| `trader/structural_stop_seam_test.go` | New candidate file |
| `trader/wiring_gate_test.go` | 6fe2c11c1b0dc0618ea029bf8e4eff7337829b02 2026-09-11T01:55:36-05:00 one-setup: the gap the first boot found, closed (owner ruling (b), scoped) — at placement time an authorization whose scenario is currently DECLINED is retired by ledger state with the owner's reason and its three verdicts, never placed, nothing at the broker; an allowed one still places (pinned on the loopback wire, RED first, 2 mutants killed); boot-line key fix folded; CLASS 121; A29 registered; MAPCHECK recomputed; SYSTEM-MAP + RULEBOOK |
| `web/src/components/plan/SessionPlanCard.tsx` | 8cfc739499c74f9567ff35a44c95b5e4fc4c2daa 2026-09-11T12:51:47-05:00 feat(level-zones): preserve full map with bounded nearest-cluster presentation |
| `web/src/components/plan/StructuralGeometry.test.tsx` | New candidate file |
| `web/src/components/plan/StructuralGeometry.tsx` | New candidate file |
| `web/src/components/strategy/DayPlanEditor.tsx` | e86ae805784b7b0ee10299a3c977738a813d0cd4 2026-08-31T08:11:57-05:00 remove min_side_levels entirely (owner ruling 2026-08-31) |
| `web/src/components/strategy/RiskControlEditor.tsx` | 9470a79d6895fa6be87c4c550220a55d07ed3aa7 2026-08-20T01:04:14-05:00 fix(E5): v1 strays — the chat-path OHLCV table renders CT like every trading prompt (the last Time(UTC) site), and the Studio min-confidence display fallback mirrors the real shared default 60 (was a stale 75) |
| `web/src/guide/content/guards.ts` | 656478f0a9b99dfb9d9974aa589c920ed3b4b53a 2026-09-10T18:10:42-05:00 docs(W2 7/n): Guide, SYSTEM-MAP, class 115 — same commit (A12); RULEBOOK §A cannot be |
| `web/src/guide/content/planCard.ts` | 6509c2e83d954c688988e85aba5f6ee445cd74c6 2026-09-10T18:53:54-05:00 test(identity): pin recording authority and preserve detector output parity |
| `web/src/guide/content/plays.ts` | fc2293f301577e477cfec5fe0dfdb920fcf88aa2 2026-09-11T01:20:14-05:00 one-setup: surfaces — API one_setup payload (the seam's RECORD, never a re-evaluation; absent → switch only, scenarios empty), desk SCENARIOS line, plan-card OneSetupChip (absent = not evaluated, never allowed) + vitest, Guide (the one play [O], the follow recorded-only [T] with round 17's null and the cell criterion, map untouched, OFF), two settings cards, knob census 45→47 |
| `web/src/guide/content/scenarioEconomics.test.ts` | 0533c8d052e8f871af26d8725522a019d0ebafe1 2026-09-08T16:19:16-05:00 test(scenario-economics): pin write boundary, legacy parity and counter mutations |
| `web/src/guide/content/settings.ts` | 9d0dc81969018c8807c9ec510804a9e02bc43764 2026-09-11T02:19:54-05:00 cleanup batch 2: closeout report, class 122 (assigned at merge; census 121 highest), guide note for the reclassified knobs |
| `web/src/guide/types.ts` | 379713e30abed379983a932f045b92b6c43a766b 2026-09-12T00:21:42-05:00 marker(wave 101): boot 400ea26c verified — BOOT INTEGRITY OK rev 400ea26c12c8 expected 400ea26c12c8 goldens PASS (PID 3671783, 00:20:36 CT); five references: /api/health revision, journald boot line, deploy/RELEASE, binary vcs.revision, md5 93e8bd17; RELEASE + GUIDE_BUILT_REV → 400ea26c; seam flag HISTORICAL_IMPORT_SEAM=on in .env |
| `web/src/lib/api/plan.ts` | 8cfc739499c74f9567ff35a44c95b5e4fc4c2daa 2026-09-11T12:51:47-05:00 feat(level-zones): preserve full map with bounded nearest-cluster presentation |
| `web/src/types/strategy.ts` | e86ae805784b7b0ee10299a3c977738a813d0cd4 2026-08-31T08:11:57-05:00 remove min_side_levels entirely (owner ruling 2026-08-31) |

## A29 — calls excluding tests, research/docs, web, data and vendor

AST calls from production Go files; wrappers are named beside the call. The maintained suite separately guards the claim list. This is wiring evidence, not a static proof that every branch is reachable.

| Claimed function | Production call sites |
|---|---|
| `ComposeLevelFadeGeometry` | `trader/arm_stop_anchor.go:81:8 in composeArmStop` |
| `FirstGeometryTarget` | `trader/structural_geometry.go:169:13 in composeGeometry` |
| `ResolveEntryGeometryZone` | `trader/structural_geometry.go:107:14 in ComposeLevelFadeGeometry` |
| `ResolveStructuralStop` | `trader/armed_executor.go:488:15 in maybeManageArmedOrdersAt`; `trader/structural_geometry_boot.go:16:7 in StructuralGeometryBootLine` |
| `SaveStructuralGeometry` | `trader/structural_geometry.go:231:12 in saveArmGeometry` |
| `StructuralGeometryBootLine` | `main.go:548:23 in main` |
| `StructuralGeometryCounts` | `trader/structural_geometry_boot.go:30:17 in StructuralGeometryBootLine` |
| `StructuralGeometryFor` | `api/handler_plan_geometry.go:6:15 in planStructuralGeometry` |
| `composeGeometry` | `trader/structural_geometry.go:108:9 in ComposeLevelFadeGeometry`; `trader/structural_geometry.go:120:9 in ComposeFrozenLevelFadeGeometry` |
| `planStructuralGeometry` | `api/handler_plan.go:505:26 in handlePlanToday` |
| `retireGeometryRefusal` | `trader/armed_executor.go:506:17 in maybeManageArmedOrdersAt` |
| `saveArmGeometry` | `trader/armed_executor.go:505:15 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:521:9 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:592:10 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:664:10 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:711:10 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:751:10 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:765:10 in maybeManageArmedOrdersAt`; `trader/armed_executor.go:776:9 in maybeManageArmedOrdersAt` |

No zero-call production claim. `ComposeFrozenLevelFadeGeometry` is a research-only wrapper and is deliberately excluded.

Protected production directories `kernel`, `market`, `provider`, and `ninjascript`: no diff from accepted base. One-setup predicate/wiring functions are unchanged; their existing seam fixtures now supply structural inputs. No live DB writes or account mutations.
