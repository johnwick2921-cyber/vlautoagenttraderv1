# DAY-PLAN CAMPAIGN — P0 · FOUNDATIONS (checkpoint report)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `97512286..` (spec+heartbeat) → `041e4450` (P0.5) + config-truth test
**Build contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md) (v1 FINAL, recon-verified @ca1f38c6)

## LINE 1 — VERDICT
**P0 · FOUNDATIONS is COMPLETE and pushed.** All five keystone items landed as
their own commits with tests; the full backend suite (build / vet / `go test ./...`
/ `-race` on the two changed packages) is green; every change is ADDITIVE and
**dormant** — the running bot (rev `3624a2a4`, PID 363618, both traders cycling)
is untouched. New tables, the WAL flip, and the FK columns activate only at the
next owner-driven rebuild + restart (★ RESTART 1, after P2). No live behavior
changed this session.

## STEP 0 gate (pre-build)
PASS — HEAD `c051b975` ≥ 3624a2a4 (ancestor) · tree clean · running rev 3624a2a4
= HEAD ancestor (changes since were FE-only) · both traders live (`hoang` #440,
`15m` #84 @ 18:08 CT). The `/api/traders` `[]` was JWT-gating; the
"Competition data excludes trader (hidden)" log lines confirm both loaded.

## What shipped, per item

### P0.1 — day_plan config on strategy rows — `d1851dac`
- `DayPlanConfig` + `DayPlanSessionOverride` at **ROOT** of `StrategyConfig`
  (sibling of `strategy_type`), wired into BOTH hand-rolled codecs
  (`MarshalJSON` out + `UnmarshalJSON` raw). RECON #1's hazard — a grid switch
  drops `ai_config` — is neutralized: `day_plan` at root survives it.
- Additive + defaults-off: a nil `*DayPlanConfig` emits no `day_plan` key, so
  every existing strategy is **byte-identical**. `DefaultDayPlanConfig()` holds
  the spec defaults but is NOT injected into `GetDefaultStrategyConfig` (opt-in).
- Goldens written FIRST: wire-schema byte golden, root-placement + deep-equal
  round-trip, grid-switch survival, absent-is-byte-identical, per-session
  inherit/override pointer semantics.
- **Config-truth persistence** (added at exit bar): `day_plan` survives
  `MergeStrategyConfig` — an unrelated edit preserves the whole block; a partial
  `day_plan` patch deep-merges (keeps siblings). `mergeJSONMaps` is recursive.

### P0.2 — plans / plan_overlays + decision FK + WAL — `0a974d31`
- `store/plan.go`: `PlanDB` (PK `plan_id,version`), `PlanOverlayDB` (PK
  `plan_id,plan_version,overlay_version`), **append-only** (never UPDATE).
- `doc`/`patch` guarded by `CHECK(json_valid(...))` via raw SQLite DDL on
  fresh-create (SQLite-only; postgres falls back to AutoMigrate). Unique index
  `(trade_date,session,version)` enforces the campaign plan key.
- **Single-writer goroutine** serializes read-max+insert version assignment.
  `SetMaxOpenConns(1)` serializes each *statement* but NOT the read+insert
  *pair*, so concurrent appends would collide on the composite PK — proven by a
  25-way concurrent test that yields versions {1..25} with no gaps/dupes, `-race`
  clean.
- `decision_records` gains 4 additive FK columns (`plan_id`, `plan_version`,
  `overlay_version`, `cited_scenario_id`) at all 4 sites (DB struct, API struct,
  `toRecord`, `LogDecision`) — empty/zero when no plan cited, so existing
  decision logging is byte-identical (no real FK constraint added; join-only).
- `gorm.go`: `journal_mode DELETE → WAL`. Whole-DB, activates at next restart;
  safe on the ext4 DB path, plays well with the C1 online `sqlite3.backup()`,
  survives WAL↔DELETE rollback.

### P0.3 — CT-anchored session registry — `6a0d233b`
- `kernel/session_registry.go`: `SessionRegistry` / `SessionDef` / `KillzoneCT`,
  America/Chicago wall-clock only (NT8 Trading-Hours pattern). Default ships
  ASIA/LONDON/NY with **only NY enabled**.
- Windows `[start,end)` with midnight-wrap reusing the proven `InBlackoutWindow`.
  ASIA 17:00→02:00 read 16:55 · LONDON 02:00→08:30 read 01:55 · NY 08:30→15:00
  read 08:25 flat 14:45. Pure evaluators: `ActiveSession`, `InWindow`,
  `InKillzone`, `IsReadTime`, `EnabledSessions`, `EffectiveFlatCT`.
- Half-day truth hook: `HalfDays` map (session-day → early-close CT), **empty by
  default (dormant)** — `EffectiveFlatCT` honors it when the calendar feed
  populates it later (RECON #6). Nothing changes live.
- Codec `Marshal`/`LoadSessionRegistry` (system_config key `session_registry`);
  empty/malformed → default registry (fail-safe, never runs empty).
- **FOUNDATION only** — NOT wired into the live gate (that is P2 · THE CLOCK).

### P0.4 — scenario-fact evaluator (the keystone) — `b51ab5c2`
- `kernel/scenario_facts.go`: pure, deterministic Go facts about a price level
  given recent bars (facts=Go, judgment=AI). Bar convention matches `svp.go`
  (chronological; closed iff `CloseTime < nowMs`).
- Primitives: `SignedDistancePoints`/`DistanceTicks` (tick≤0 guard),
  `ClosesBeyond` (consecutive-from-newest), `Acceptance` (2x5m→2 / 15m-close→1,
  normalizes unicode `2×5m`), `Swept` (wick pierced, closed back), `Reclaimed`,
  `Rejected` (S/R held), `LevelStillValid` (consumed detection),
  `EvaluateLevelFacts` (one-pass snapshot for the prompt tail).
- 11 historical-style bar-fixture tests incl. open-bar skip and a
  sweep→reclaim→accept aggregate. NO LLM wiring (as the spec demands).

### P0.5 — level-state table (identity-keyed) — `041e4450`
- `store/level_state.go`: `LevelStateStore` + `LevelStateDB`. State persists
  across the sessions that re-derive a level, keyed by IDENTITY = `(symbol,
  level_type, origin_date, bin_index)` — a burned level can't return fresh.
  `origin_date` = the session-day of formation (stable); `bin_index` = the
  absolute price grid (mirrors `svp.go` `svpBinIndex`).
- API: `EnsureLevel` (create-fresh-if-new / preserve-state-if-exists / refresh
  price only), `RecordTest` (atomic +1), `MarkConsumed`, `DecrementFreshness`
  (A→B→C→done, done⇒consumed), `Get`, `ListForSymbol`, `ListValid`. Freshness +
  level-type string consts. Written from the trader layer (kernel is DB-blind).
- **Foundation only** — the 17:00-CT snapshot WRITER that populates it is a P1
  item (RECON #2, the durable session-profile store).

## EXIT BAR
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./store/ ./kernel/` ✓.
- **Goldens:** untouched — `git diff` on `kernel/testdata/` across the P0 range
  is empty (deliberate-only rule honored; no golden regeneration).
- **Frontend:** zero files under `web/` touched in P0 → tsc/npm N/A, inherits the
  last green state. (Studio Day Plan block is P4.)
- **config-truth 4-step** for `day_plan`: step 1 (struct+tag) ✓, step 2
  (codec/persist incl. MergeStrategyConfig survival) ✓; step 3 (live-loop read)
  = P3 executor, step 4 (FE editor + i18n) = P4 — deferred by campaign design,
  not gaps.

## Deploy status
Nothing to deploy this session. All P0 code is dormant until the next owner
rebuild + restart:
- The NEW tables (`plans`, `plan_overlays`, `level_state`) are created by
  AutoMigrate/DDL at store init — i.e. **on the next binary start**, not now.
- The 4 additive `decision_records` columns are added by AutoMigrate on next
  start (nullable/defaulted; non-destructive).
- WAL flips on next `InitGorm` (next start). The C1 backup timer already runs;
  new-table creation + additive columns are non-destructive, but the owner's
  ★ RESTART 1 should still be preceded by the standard backup.

## Owner flags (decisions surfaced, not blocking)
1. **NY flat time** encoded as **14:45 CT** = the spec's "15:45 ET" eod-flat (15
   min before the 15:00 CT / 16:00 ET RTH close). The registry is admin-editable
   and not yet wired to force-flatten; confirm/adjust at ★ RESTART 1.
2. **WAL is whole-DB** (SQLite journal mode isn't per-table). Standard + safe;
   flagged because it is a system-wide storage change bundled under P0.2.

## What's next — P1 · THE MAP
Kernel siblings of SVP, computed once/session: multi-day level detectors
(PDH/PDL/PDC, ONH/ONL, RTH/AS/LDN H-L, prior wk/mo, round numbers, gaps, OR+IB,
naked-POC) → the **durable session-profile store + 17:00-CT snapshot writer**
(RECON #2, MANDATORY, warms forward) → EQH/EQL + S/D zones + FVG/OB → confluence
scorer (graded TOP-8, deterministic golden-day fixtures) → REGIME block →
**KEY LEVELS block into the live executor prompt** (goldens deliberate, B9-style
empty-log) → calendar fetcher. Ends before ★ OWNER RESTART 1 (map + clock live).

_vlauto: DEFER to the next propagation train (per dispatch)._
