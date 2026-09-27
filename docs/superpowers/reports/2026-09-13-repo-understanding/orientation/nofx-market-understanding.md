# Orientation 1 — market data, structural maps and planning

Base: `63968be62e44db2fb07a92883e02127b9064b0be`. Isolated worktree: `/tmp/nofx-understanding-market-20260913`; cwd and HEAD verified before reading. Read-only orientation: no code/config/DB/runtime changes, no builds or tests executed, no CGC reindex. Evidence labels: [A] exact source inspected; [B] consequence inferred from inspected sources; [C] unverified concern. This is NOT a certification that every function or runtime path is understood.

Read ledger: `/tmp/nofx-market-reads.json` contains 44 fully read files, 13 excerpt/inventory entries, exact SHA256/bytes and each file's `git log -1` metadata. 98 production files in market/kernel/provider-ninjatrader remain not fully read; exact list below. An inventory hit is not a full read. Root/subsystem AGENTS files are not present in the tracked isolated checkout's file inventory; inherited owner instructions apply. Tracked `docs/superpowers/CLAUDE-canon.md` was fully read, including automatic lock keeper and checked-worktree rules. Relevant audit rules are classes 2,7,8,9,13,18,19,115–118,120–123; full checklist not claimed.

## End-to-end path understood so far

[A] The live futures market provider is an injected function, not a market-package import of a broker. `market/futures_data.go:14` declares `FuturesBarsProvider`; `trader/ninjatrader/bars_market_bridge.go:20` installs a closure calling `barsFromCache` with one entry-point clock. This keeps the dependency edge trader/ninjatrader → market. The ring key is exact `symbol|timeframe`, case-sensitive in `barKey`; the bridge does not remap symbols. `market.Normalize` preserves recognized futures before crypto uppercasing/USDT transforms (`market/data.go:670`). Canonical root input is consequently essential; recognizing a spelling as futures does not guarantee a matching ring key.

[A] TCP handling is asynchronous. `provider/ninjatrader/tcp_server.go:1592` drains bar messages to `SeedHistorical` or `Upsert`, then derives persistence work. `enqueueBarUpdate` stamps last-bar receive time and uses a nonblocking queue/drop-oldest policy; a historical batch uses a bounded two-second send. Queue default is4096 and auto historical ask2000, while ring default2500. The source comments describing bars as unconditionally never dropped are stronger than the implementation: historical timeout and last-resort persistence drops are counted and logged. No actual drop rate measured here.

[A] `BarCache` drops zero-volume, zero-range NT8 placeholders, converts wire period-end timestamps to open stamps once, then labels historical versus live (`bar_cache.go:77,155,240,321`). Historical merge keeps existing live/mixed values on timestamp collisions. Live upsert replaces the final same-open bar, appends newer bars and ignores older updates. `Get` returns a defensive copy. `RehydrateOlder` refuses a cold key, only adds strictly older source rows and trims oldest at capacity; it cannot make an absent live ring look alive.

[A] Contract identity comes from received subscription ACKs. `observeContract` records the named contract; a changed name purges all cached timeframes for that symbol and notifies listeners (`contract_roll.go:97`). First observed contract or same-name reconnect is not a roll. Persistence resolves current ACK first, store fallback second, otherwise unavailable (`trader/ninjatrader/bar_persist_wire.go:21`). However, the callback resolves contract later than wire receipt; see independent-review requests below.

[A] `barsToKlines` computes scheduled CloseTime from open plus duration minus1; the final bar may still be forming (`bars_market_bridge.go:72`). `market.GetWithTimeframes` reads each requested TF separately, fetches enough warm-up for the largest configured EMA/RSI/ATR/BOLL period (clamped200–2500), skips absent secondary TFs and errors on absent primary. CurrentPrice is the primary slice's final close; there is no closed-bar filter at this function. Futures bypass Binance OI/funding queries (`market/data.go:190–359`). Thus the comment saying this reads closed bars is not a semantic guarantee. Closed-bar consumers must filter themselves.

[A] Indicator implementation is deterministic OHLCV math (`market/data_indicators.go`): SMA-seeded EMA; MACD line only (EMA12−EMA26); Wilder RSI/ATR; population-variance Bollinger bands; Donchian uses whatever history exists up to requested length. Insufficient warm-up returns numeric zero in several indicators, not a nullable unknown. The consuming layer must avoid representing those zero values as established market facts.

[A] `BarResolver.CompletedBars` is a separate path (`market/bar_resolver.go:142`): walk configured native→finer-native→own1m ladder, stop at first nonempty in-window result, attach source metadata and exclude periods not yet closed. It does not demand a minimum coverage or merge a thin native answer with a deeper fallback. `AggregateToTF` bins by epoch-floor time and aggregates available bars; constituent completeness is not checked. `Completed=true` means the bucket's scheduled period elapsed, not that every constituent arrived.

[A] In the planner, `assemblePlannerInputWithCtx` takes a read clock and reads base1m plus configured HTF series. The exact detection series are captured for `LevelZoneInputs`; `AssembleResearchLevels` returns seated, pool, price, daily-range proxy and raw detector universe (`trader/auto_trader_planner.go:2138–2187`). Late in assembly, raw universe builds the uncut frozen zone map, attaches research snapshot ID and persists read facts (`:2521`). This is snapshot-oriented, but individual timeframe cache reads are sequential, not one atomic multi-TF snapshot.

[A] Base assembly (`kernel/levels_assemble.go:212`) runs multiday, round, OR/IB, gap, equal-high/low, supply/demand, FVG, order-block, volume and swing detectors, plus supplied extras. Raw pre-dedup evidence is preserved. Same-kind/same-TF within0.25 dedupe runs before scoring. The full detector implementations are NOT all read yet. Native HTF detection uses canonical aliases D→1d/W→1w; exact current supported set is15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w. Four families run per configured supported TF: equal-high/low, supply/demand, FVG, order-block. Fetch500 bars, minimum5 closed; absent/short TF emits a recorded skip.

[A] The score layer and zone layer are different objects. `scoreLevelsPool` applies distance band, freshness, distinct-family confluence, provisional evidence/HTF/zone weights, grade floors/caps, Tier1 proximity, cluster collapse and seating (`levels_score.go:433–638`). Confluence radius is0.10×daily-range proxy, not the0.25 dedupe radius and not the zone merge radius. `ScoreLevelsMinGradeFull`'s so-called pool is itself capped at2×maxLevels (`:657`), not all raw candidates. Grade/minimum/confluence are code rules; none establishes out-of-sample trading superiority.

[A] `BuildMapCandidates` merges score references around strongest surviving anchors, keeps names/source IDs and orders by distance. Its entry-candidacy heuristic asks whether adjacent opposing reference clears a minimum defaulting to MinSLATRMult×ATR5m (`map_candidates.go:132,431`). This candidate flag is not identical to one-setup authorization. `OneSetupBestCandidate` ignores EntryCandidate and chooses nonprojection positive-price candidates in band by grade threshold, then grade/distance/price (`one_setup.go:173`). That distinction is central to understanding why the card and authorization may use different concepts of “best”.

[A] `BuildLevelZones` instead consumes all raw detector references (`level_zones.go:149`): preserves native bands; point width is max(defining wick,k×sourceTF ATR), fixed width for round numbers, otherwise NULL. Known-width references join exactly one nearest compatible existing cluster; fixed anchor never moves, no transitive union, union width capped in ATR5m. Broad bands stay standalone. Unknown widths cannot merge. Family count capped3; rank uses log1p(prior touches), round proximity, family count minus distance/ATR5m. Rank is not the legacy score. All width/merge choices are provisional; they now supply frozen stop/target geometry downstream and must not be treated as purely cosmetic.

[A] Identity has two distinct mechanisms. `LevelBinIndex` is durable kind/price-bin state for freshness. Scenario candidate identity uses recorded symbol/kind/bounds/origin/TF/formation-close inputs (`scenario_level_identity.go:83`). `LevelByID` recomputes and validates hashes; conflicts resolve unknown, never nearest. `StampAuthoredIdentity` strips model-supplied metadata and restores exact machine metadata on unique exact-price matches; scenario references remain explicit model-authored LevelID. Source formation OPEN and evidenced CLOSE are intentionally distinct. Unknown birth does not become a synthetic hash or zero prior touches.

[A] `ParsePlanDocCapped` is the new-authoring door; it validates schema and required economics. `ParsePlanDoc` still schema-validates with defaults but skips new economics. Historical consumers therefore must not blindly use it on12-level records; class120 documents that exact failure. `ValidatePlanDocWithFactsMachine` rejects empty-known machine map, duplicate prices, zero levels on either side, direction-unreachable gap scenarios and out-of-band targets. A nil machine map or facts.Price≤0 retains legacy looser behavior. This function has a literal1.5×DATR target-chain band (`plan_doc.go:950`), while proximity resolver accepts configured0.1–3.0 elsewhere; this distinction deserves a targeted consumer audit.

[A] `scenario_economics.go` treats a present arm's entry/stop/target as authoritative over proposed geometry. New authoring requires economics and explicit first-obstacle response, computes stated-R consistency in price units, checks arm target appears in target chain or declared exception, and rejects obstacle beyond target. Sub1R obstacle and role-use disagreements are WARN observations, not trading authorization. Its R calculations are absolute geometry and require other validation for directional protection.

[A] Entry-law table: reject/fvg touch-only; sweep-reclaim split permits touch then1mMSS/1x5m; reclaim1x5m/MSS; breakout-retest touch or1x5m; acceptance/hold time_hold or1x5m; continuation1x5m/2x5m. Law-backed repair prompts read the same rule table. `STOP_ENTRY_SEAM` defaultsOFF; this lane did not establish its current resolved runtime setting.

[A] Confirmation timing uses explicit clocks. A touch on a forming1m bar can be MET but has no known ordered event time; a closed minute supplies an upper-bound close instant, not exact tick time (`confirmation_evidence.go:24`). Leg2 requires a known part1 event and strictly later qualifying evidence. Bucket elapsed/closed is a separate predicate (`confirmation_bucket.go:26`). `time_hold` counts qualifying closed rows; continuity/gap treatment needs further verification. `StaleConfirmATR5m` aggregates all supplied bars then computes ATR14 without its own now/closed filter (`plan_confirm.go:85`); callers must provide the intended tape.

[A] One-setup authorizes only when level/play/permission legs all pass (`one_setup.go:89`). OFF bypasses this policy. Permission independently evaluates OR width, completed IB break/hold, beyond authored map, T1 news and first-N-minute facts. Model-worded day_type is deliberately not an input. UNKNOWN calendar does not itself exclude; if another component is evaluable and none excludes, Permitted can be true. NULL entire permission does not permit. This is a provisional filter, not a validated early trend classifier.

[A] Plan lifecycle consumption is windowed after plan birth and requires touched plus accepted-through on rule timeframe (`plan_lifecycle.go:88,192`). Structured death/flip uses buffered closes and hysteresis; stale rule-series skip the evaluation. Full-cache historical crossings must not kill a newly written plan. These are plan-thesis state controls, not broker stop management.

[A] Daily clocks differ deliberately: CME session-day begins17:00CT; registry NY default read08:00, active08:30–14:45, Asia/London defaultdisabled. Registry flat field historically is not the execution consumer; window-end+offset enforcement must be traced in execution lane. Exchange openness comes from session calendar plus weekly hours. Runtime enabled sessions/settings were not read here.

[A] Decision path uses strategy TFs and configured indicator periods (`engine_analysis.go:778`), exempts futures from crypto OI filtering, sets plan/key-level context, checks prompt owner identity, invokes model+bounded schema repair, then applies price-sanity, stale-data and reentry checks. This is not the entire arm path. Kernel daily guardrail resolves master/per-control switches and daily realized PnL; an actual daily-force-flat decision latches entry refusal, but this excerpt explicitly does not broker-flatten existing positions. Owner's later daily-loss correction must supersede old per-trade-cap dispatch text.

## Priority independent reviews — not current incidents

1. **[A] Class118 withdrawal versus still-wired workaround.** Checklist:4296–4362 says scale detector and replay hold were withdrawn after a mistaken-cause rollback. Current `BarCache.Upsert` calls `detectScaleMismatch`; `WireBarPersistence` reads its SeedVerdict and resolves replayHold; mismatch listener is registered. `detectScaleMismatch` sets liveSeen before looking for historical witness, and returns without marking unknown if none exists. This is precisely the absence-precondition mechanism described in class118. Verify actual historical merge provenance and whether an intentional later owner ruling reinstated this code before labeling it a regression.
2. **[A/B] Mixed-source exclusion missing at ring bridge.** Detector labels a straddling bar mixed; `BarCache.Get` copies every bar and `barsToKlines` copies every returned value into a struct lacking Source. Thus the direct market path has no exclusion at those inspected boundaries despite logs saying no reader takes mixed. Store-reader filtering is a different path. Independent review should trace consumers and add a narrow source-tag fixture in a future authorized fix, not assume a current bad bar exists.
3. **[A/B] Weekly calendar split.** `weeklyDailyBars` says it returns DAILY for Monday-governed `CompletedWeekCandles`, but actually calls resolver for1w. Resolver aggregates to Unix epoch-floor7-day buckets (ThursdayUTC), then weekly reader rebuckets already aggregated values by their open's governing Monday. Market tests deliberately epoch-align and assert count/source, not Monday/CME bounds. Verify against a two-week fixture with distinct daily extrema; require source-to-consumer calendar equivalence.
4. **[A/B] Delayed contract stamping.** Ingest/persist queue payloads inspected carry symbol/TF/bars, no contract. ACK processing can change CurrentContract before queued old data reaches persistence; callback resolves current value at processing time. Verify whether connection sequencing/drain prevents crossing; if not, queued old rows could inherit new contract label. This lane did not simulate concurrency or assert an observed corruption.
5. **[A] Historical prose is stale at policy boundaries.** `level_zones.go`/SYSTEM-MAP still say display-only although geometry consumes zones. `fade_permission.go` says never a gate although one-setup gates on it. Rulebook Part3 excludes daily HTF while current set includes daily/weekly. `min_sl.go` still advocates floor as three-path stop rule and its resolver comment says default1.0 although constant1.5 and structural reject now bypasses it. None of these old comments should become current implementation instructions.
6. **[A/B] Coverage versus time elapsed.** Resolver completion, confirmation bucket closure and actual complete source coverage are distinct. `HorizonOf` counts open-calendar expected intervals minus served count; its memo key is endpoints/count, not full timestamp membership. Missing/duplicate/closed-session rows can defeat assumptions if ingestion invariants fail. `LevelZoneInputs` ensures oldest≤formation but does not explicitly prove a gapless postformation window. Review coverage-aware decisions separately from display labels; no synthetic fill allowed.
7. **[A/B] Persistence success means callback returned, not DB commit.** `bar_persist.go` catches callback panic silently, increments persistFlushed and stamps persistLastFlushAt after callback invocations regardless of their internal DB error. Watchdog is scheduled in same worker as synchronous callback; a blocked callback cannot run its own ticker. Verify watchdog liveness/failure semantics independently. Do not call these counters committed rows.
8. **[A/B] Best-level geometry interaction.** Zone shortlist ranking, scored candidate grade-ranking and executable structural geometry are separate. One-setup may prefer a grade-leading candidate whose frozen nearest target fails geometry while a lower-ranked candidate is viable; source choice is not global expectancy maximization. Owner's “find best trade” needs an explicit behavior definition before any policy rewrite.

## Tests inspected, not executed

`market/bar_resolver_test.go`: ladder/source priority, forming-bar removal, unknown TF, no source, native-week exclusion, floor alignment. It verifies existing epoch behavior rather than intended CME-week source-to-consumer parity.

`kernel/one_setup_test.go`: five scenarios/exactly one authorized, unknown permission, unknown identity, legacy proximity, grade then distance, off behavior and iteration order. It does not itself test broker placement or geometry interactions.

`kernel/level_zones_test.go`: frozen raw843-reference snapshot →498 zones/25 merges, native bands, no detector mutation, defining-wick evidence, source names, single shortlist, HTF-neutral zone rank, env resolver, future-bar exclusion, NULL width, nontransitive clustering and capped independent families. Those figures are fixture expectations, not newly measured market populations.

Recommended next review assignments: (1) weekly source/calendar end-to-end; (2) replay/source/contract queue semantics; (3) all detectors and score/seating; (4) planner write/repair/overlay lifecycle; (5) confirmation and missing-minute evidence; (6) actual arm-gate best-selection versus geometry; (7) current-document reconciliation. Root should assign new work rather than infer complete coverage from this orientation.

## Exact production files not fully read in this lane

- `market/api_client.go` — inventory-only/not-read, 160 lines.
- `market/data.go` — partially-read, 823 lines.
- `market/data_klines.go` — inventory-only/not-read, 494 lines.
- `market/databento_adapter.go` — inventory-only/not-read, 30 lines.
- `market/decimal_safe.go` — inventory-only/not-read, 88 lines.
- `market/historical.go` — inventory-only/not-read, 104 lines.
- `kernel/adherence.go` — inventory-only/not-read, 163 lines.
- `kernel/arm_kind.go` — inventory-only/not-read, 111 lines.
- `kernel/armed.go` — inventory-only/not-read, 79 lines.
- `kernel/arms_bias_coherent.go` — inventory-only/not-read, 123 lines.
- `kernel/boot_integrity.go` — inventory-only/not-read, 209 lines.
- `kernel/breakdown_continue.go` — inventory-only/not-read, 304 lines.
- `kernel/calendar_blackout.go` — inventory-only/not-read, 146 lines.
- `kernel/candle_disclosure.go` — inventory-only/not-read, 408 lines.
- `kernel/class45_feeds_forward.go` — inventory-only/not-read, 214 lines.
- `kernel/clock_drift.go` — inventory-only/not-read, 178 lines.
- `kernel/clock_health.go` — inventory-only/not-read, 188 lines.
- `kernel/condition_status.go` — inventory-only/not-read, 135 lines.
- `kernel/confirmation_telemetry.go` — inventory-only/not-read, 60 lines.
- `kernel/detector_d1prime.go` — inventory-only/not-read, 282 lines.
- `kernel/detector_recorder.go` — inventory-only/not-read, 122 lines.
- `kernel/digest.go` — inventory-only/not-read, 116 lines.
- `kernel/displacement_feeds_forward.go` — inventory-only/not-read, 142 lines.
- `kernel/engine.go` — partially-read, 1084 lines.
- `kernel/engine_analysis.go` — partially-read, 1133 lines.
- `kernel/engine_position.go` — inventory-only/not-read, 310 lines.
- `kernel/engine_prompt.go` — inventory-only/not-read, 1112 lines.
- `kernel/engine_prompt_futures.go` — inventory-only/not-read, 454 lines.
- `kernel/engine_prompt_observer.go` — inventory-only/not-read, 164 lines.
- `kernel/excursion_path.go` — inventory-only/not-read, 123 lines.
- `kernel/flip_freshness.go` — inventory-only/not-read, 78 lines.
- `kernel/follow_plan.go` — inventory-only/not-read, 221 lines.
- `kernel/formatter.go` — inventory-only/not-read, 645 lines.
- `kernel/fvg_entry.go` — inventory-only/not-read, 438 lines.
- `kernel/golden_selfcheck.go` — inventory-only/not-read, 158 lines.
- `kernel/grid_engine.go` — inventory-only/not-read, 621 lines.
- `kernel/htf_veto.go` — inventory-only/not-read, 122 lines.
- `kernel/level_stats_calc.go` — inventory-only/not-read, 141 lines.
- `kernel/levels_intraday.go` — inventory-only/not-read, 205 lines.
- `kernel/levels_role.go` — inventory-only/not-read, 449 lines.
- `kernel/levels_score.go` — partially-read, 1198 lines.
- `kernel/levels_swing.go` — inventory-only/not-read, 231 lines.
- `kernel/levels_volume.go` — inventory-only/not-read, 429 lines.
- `kernel/levels_volume_boot.go` — inventory-only/not-read, 61 lines.
- `kernel/levels_zones.go` — inventory-only/not-read, 357 lines.
- `kernel/mae_mfe.go` — inventory-only/not-read, 59 lines.
- `kernel/map_projections.go` — inventory-only/not-read, 211 lines.
- `kernel/matched_random.go` — inventory-only/not-read, 192 lines.
- `kernel/mss.go` — inventory-only/not-read, 145 lines.
- `kernel/naked_poc.go` — inventory-only/not-read, 73 lines.
- `kernel/no_trade_band.go` — inventory-only/not-read, 265 lines.
- `kernel/plan_authored_invalidation.go` — inventory-only/not-read, 83 lines.
- `kernel/plan_chain_date.go` — inventory-only/not-read, 92 lines.
- `kernel/plan_doc.go` — partially-read, 1333 lines.
- `kernel/plan_overlay.go` — inventory-only/not-read, 275 lines.
- `kernel/plan_overlay_carry.go` — inventory-only/not-read, 184 lines.
- `kernel/plan_render.go` — inventory-only/not-read, 385 lines.
- `kernel/planner_indicators.go` — inventory-only/not-read, 47 lines.
- `kernel/planner_prompt.go` — inventory-only/not-read, 835 lines.
- `kernel/planner_speed.go` — inventory-only/not-read, 58 lines.
- `kernel/price_sanity.go` — inventory-only/not-read, 77 lines.
- `kernel/prompt_builder.go` — inventory-only/not-read, 376 lines.
- `kernel/prompt_contract.go` — inventory-only/not-read, 172 lines.
- `kernel/prompt_ownership.go` — inventory-only/not-read, 33 lines.
- `kernel/realign.go` — inventory-only/not-read, 112 lines.
- `kernel/regime.go` — inventory-only/not-read, 259 lines.
- `kernel/regime_baseline.go` — inventory-only/not-read, 112 lines.
- `kernel/regime_dark.go` — inventory-only/not-read, 90 lines.
- `kernel/regime_ledger.go` — inventory-only/not-read, 16 lines.
- `kernel/repair_outcome.go` — inventory-only/not-read, 64 lines.
- `kernel/research_score.go` — inventory-only/not-read, 92 lines.
- `kernel/risk_limits.go` — inventory-only/not-read, 459 lines.
- `kernel/scenario_facts.go` — inventory-only/not-read, 499 lines.
- `kernel/scenario_state.go` — inventory-only/not-read, 261 lines.
- `kernel/schema.go` — inventory-only/not-read, 555 lines.
- `kernel/session_calendar.go` — inventory-only/not-read, 425 lines.
- `kernel/shadow_ab.go` — inventory-only/not-read, 271 lines.
- `kernel/structure.go` — inventory-only/not-read, 449 lines.
- `kernel/svp.go` — inventory-only/not-read, 392 lines.
- `kernel/touch_telemetry.go` — inventory-only/not-read, 605 lines.
- `kernel/transition.go` — inventory-only/not-read, 107 lines.
- `kernel/tz.go` — inventory-only/not-read, 87 lines.
- `kernel/validator_hints.go` — inventory-only/not-read, 216 lines.
- `kernel/void_scope.go` — inventory-only/not-read, 124 lines.
- `kernel/weekly_bias.go` — partially-read, 322 lines.
- `kernel/weekly_knobs.go` — inventory-only/not-read, 128 lines.
- `kernel/weekly_prompt.go` — inventory-only/not-read, 565 lines.
- `provider/ninjatrader/csv_tailer.go` — inventory-only/not-read, 110 lines.
- `provider/ninjatrader/csv_writer.go` — inventory-only/not-read, 74 lines.
- `provider/ninjatrader/echo_verify.go` — inventory-only/not-read, 206 lines.
- `provider/ninjatrader/mock_nt.go` — inventory-only/not-read, 155 lines.
- `provider/ninjatrader/order_snapshot.go` — inventory-only/not-read, 259 lines.
- `provider/ninjatrader/order_state.go` — inventory-only/not-read, 170 lines.
- `provider/ninjatrader/research_wire.go` — inventory-only/not-read, 217 lines.
- `provider/ninjatrader/tcp_client_mock.go` — inventory-only/not-read, 228 lines.
- `provider/ninjatrader/tcp_framing.go` — inventory-only/not-read, 690 lines.
- `provider/ninjatrader/tcp_server.go` — partially-read, 2332 lines.
- `provider/ninjatrader/types.go` — inventory-only/not-read, 59 lines.
