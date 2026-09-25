# VL DAY-PLAN CAMPAIGN — P5 · THE DOOR — completion report (+ ★ RESTART 2 handoff)

**Date:** 2026-08-15 · **Branch:** main · **Head after P5:** `41afa1b6`
**Scope:** P5.1–P5.7 — the owner door (edit / conflict / ask-planner), the
learning loop (adherence + stats honesty gate), + an adversarial-review hardening
pass. Additive + dormant until day_plan is enabled AND a plan exists; SIM-only;
owner data / bindings / guardrails master untouched; bot NOT restarted (that is
★2, owner-run).

---

## Verdict

**P5 COMPLETE — the VL DAY-PLAN CAMPAIGN build is COMPLETE.** All seven items
shipped (8 commits), each pushed. A 4-dimension adversarial review (18 agents)
found 12 confirmed issues; the HIGH (B2 armor bypass) + all substantive MEDIUM/LOW
correctness issues are fixed. Exit bar green: `go build/vet/test/-race` (26 pkgs,
0 fail) · `tsc` + `npm run build` clean · vitest 153 pass (sole failure = the
pre-existing `RegistrationDisabled` logo test) · no goldens touched. Live behavior
+ the blind-mark owner hour are gated on ★ RESTART 2 (this weekend).

---

## Commits (8)

| SHA | Item |
|---|---|
| `899f14f5` | P5.1 — overlay API: RFC-6902 applier + test-op concurrency + B2 armor + sticky owner levels + round-trip |
| `94fa6daf` | P5.4 backend — Ask-Planner (verbatim anti-sycophancy contract, code-enforced; verdict-log KPI) |
| `ad9dd424` | P5.5 — adherence grade A–F per closed trade (fixes the never-persisted cited_scenario_id gap) |
| `fa9344b2` | P5.6 — stats honesty gate (matched-random, WARMING, Bonferroni, weekly freeze) |
| `3126b61e` | P5.2+P5.3+P5.4 FE — the owner door (edit sheet, conflict chip, ask-planner panel) |
| `e99a145f` | P5.2 bulk add + P5.7 blind-mark grammar parser |
| `41afa1b6` | P5 hardening — 12 adversarial-review findings |

---

## P5.1 — OVERLAY API (keystone) **[A]**

- `kernel/plan_overlay.go` — a self-contained **RFC-6902 applier** over RFC-6901
  pointers (no lib exists). `ApplyPatchStrict` = atomic (the write-time **test-op
  concurrency** guard); `ApplyOverlayPatches` = read-time fold that SKIPS a bad
  overlay so plan_final never corrupts. `handlePlanToday` now resolves
  plan_final = base + overlays (ASC), re-armored via `ValidatePlanDoc`.
- `POST /plan/overlay` — applies strictly onto the current plan_final (409 on
  test-op/validity conflict), **B2-armors** every resulting price (422 on 8×dATR
  fat-finger), origin tag (owner | planner-revised). `POST /plan/owner-level`
  (+ delete) — sticky owner levels (P3.6-C store; note + scenario tag ride to the
  planner). Guarded write (armored, CreatedAt set, symbol-scoped).
- ROUND-TRIP proven: store↔kernel test appends an owner overlay → resolves →
  plan_final reflects the edit; a stale test-op is rejected (concurrency).

## P5.4 — ASK-PLANNER (anti-sycophancy) **[A]**

- `kernel/ask_planner.go` — the **verbatim §44 contract** IS the system prompt AND
  enforced in code: `ParsePlannerReply` forces a BARE-DISAGREEMENT to DEFEND and
  strips any patch; a patch rides only PROPOSE-MERGE; an empty/un-appliable patch
  downgrades to DEFEND. The model cannot flatter its way to a plan change.
- `store/plan_qa.go` — durable per-plan threads; each reply logs {point_class,
  verdict, patch, applied}; `VerdictStats` = the queryable sycophancy KPI
  (incl. defend-on-bare). `POST /plan/ask` calls the SAME planner model that
  authored the plan; `POST /plan/ask/apply` applies a PROPOSE-MERGE as a
  planner-revised overlay (bound to its authoring plan; one-reply-one-overlay).
- FE `AskPlannerPanel` — bottom-sheet (mobile) / right-panel (≥1200px, portaled);
  EVIDENCE → YOUR POINT[chip] → VERDICT[chip] + patch preview; Apply is an
  explicit tap; a bare reply shows no Apply button.

## P5.5 — ADHERENCE GRADE A–F **[A]**

- Fixes the recon's central gap (cited_scenario_id was never written). trader_
  positions gains plan_version / cited_scenario_id / plan_matched (stamped at
  OPEN from the citation) + adherence_grade (computed at CLOSE). All day_plan-
  gated → dormant for crypto.
- `kernel/adherence.go` — `GradeAdherence` rubric (cited+matched+clean=A, mismatch
  =C, off-plan=D; no-trade-window / outside-killzone step toward F). SEPARATE from
  P&L by construction. `GET /plan/trades` — graded trades + GPA for the review feed.

## P5.6 — STATS HONESTY GATE (MUST-V1) **[A]**

- `kernel/matched_random.go` — the guarantee: **NO green "beats random" on an
  underpowered sample, ever.** The power check (n ≥ 1,565/type) short-circuits to
  WARMING before significance; only powered AND ≥5pp AND Bonferroni-significant
  (α≈0.006 across 8 types) is BEATS-RANDOM. Test proves a 90% rate on n=50 is
  still WARMING.
- `store/matched_random.go` — durable verdicts + weekly snapshot; `SaveWeeklyIf
  Absent` = first-writer-wins (no re-peek). Weekly job (Sundays only, idempotent)
  freezes one snapshot per ISO week; `recordMatchedRandomForClose` records a
  per-type reaction at each graded close. `GET /plan/stats` serves the FROZEN
  snapshot (never recomputed live) + WARMING progress.

## P5.2 / P5.3 — EDIT SHEET · BULK ADD · CONFLICT CHIP **[A]**

- EditSheet (createPortal bottom-sheet, own Esc/focus): tap a level row → edit →
  RFC-6902 overlay; ＋ Add → sticky owner level; controlled vocab (English tokens,
  translated labels). BulkAddSheet — multi-line → parsed preview → one Save.
- ConflictChip + `detectConflicts`: an owner level opposing an AI level at the
  same price ⚡-flags both and GHOSTS the AI row (owner wins, both kept visible).

## P5.7 — BLIND-MARK PREP **[A/B]**

- **Parser verified [A]:** `bulkParse.ts` — the shared marking grammar
  (`<price>[-<price2>] [type] [note…]`) backs both the bulk-add sheet AND the
  owner's blind-mark hour; 6 unit tests incl. the owner-hour grammar.
- **10-day selection [B]:** the blind-mark days come from the LIVE in-memory
  BarCache (not queryable here — data.db holds no session-day bar index). The
  owner picks 10 days spanning the diversity axes below at the owner hour (post-★2),
  when the cache has warmed forward from install:
  1. trend-up day · 2. trend-down day · 3. tight-range day · 4. wide-range day ·
  5. high-ATR/EXTREME regime · 6. low-ATR/LOW regime · 7. large overnight-gap day ·
  8. gap-and-go vs gap-fill · 9. an FOMC/CPI/NFP event day · 10. a quiet
  pre-holiday half-day. Criterion: cover each `atr_regime` bucket + both gap
  behaviours ≥ once so every level TYPE gets a fair reaction sample.

## Adversarial review + hardening (`41afa1b6`)

18 agents across 4 dimensions (dormancy · RFC-6902 · anti-sycophancy · SIM-safety),
each finding independently verified. 12 confirmed; **DORMANCY: 0 leaks** — every
hot-loop stamp is `dayPlanEnabled`-gated, no crypto/plan-off path writes or changes
execution. Fixed:

| Sev | Fix |
|---|---|
| HIGH | B2 armor bypass (whole-array `/levels` replace / target_chain) → armor the RESOLVED plan_final, not the patch shape; + ValidatePlanDoc rejects non-positive prices (read AND write paths) |
| MED | Ask-Planner Apply: guard `msg.Applied` (one-reply-one-overlay) + bind to the authoring plan (409 if rolled) |
| LOW | stale plan-link reset · `hasPatchOps` op-validation (mislabel → DEFEND) · reject signed `-0` index · overlay mutex (atomic test-op) · day-plan-owner gate on the shared-plan mutation |

**Acknowledged, not fixed** (single-owner SIM deployment; codebase-wide pattern,
not P5-introduced): `GetTrader` is not JWT-user-scoped — it mirrors the existing
`/risk/*` + `/audit/*` handlers. A `requireOwnedTrader` multi-user pass is a
separate follow-up, tracked for the next hardening train.

## Exit bar

| Gate | Result |
|---|---|
| go build / vet | clean |
| go test ./... | **26 pkgs ok, 0 fail** |
| go test -race (kernel/api/store/trader) | clean |
| tsc + npm run build | clean |
| vitest | 153 pass / 1 = pre-existing `RegistrationDisabled` logo (predates P4) |
| goldens | **none touched** (bar goldens untouched; full go suite green) |
| config-truth | no new config fields (P5 is runtime: overlays/QA/adherence/stats) |
| i18n | ~40 new keys, en/zh/id (type-enforced) |

---

## ★ RESTART 2 — HANDOFF (owner)

**THIS WEEKEND IS THE IDEAL WINDOW** — the market is closed, so the rebuild is in
a flat/safe state, and the first NY planner read fires **Monday 08:25 CT**, lighting
the whole door + learning loop on a live plan.

**Commands (owner, from `/home/hoang/nofx`):**
```bash
git pull                                  # HEAD 41afa1b6
go build -o nofx-bin ./... && echo BUILD OK
cd web && npm run build && cd ..          # ship the P5 door
# rebuild the AddOn? NO — P5 touched NO ninjascript/*.cs. Skip the F5 dance.
kill -9 $(pgrep -f nofx-bin)              # systemd Restart=on-failure respawns the new binary
```
(`sudo systemctl restart nofx` is classifier-blocked here; SIGKILL is the deploy
per CLAUDE.md. Day_plan is already armed from ★1 — no re-arm needed.)

**VERIFY after boot (5 checks):**
1. **Clean boot** — `journalctl -u nofx -n 40` shows the new binary up, no panic,
   bars flowing (`📊`/`KEY LEVELS`).
2. **API graceful** — `GET /api/plan/today?trader_id=<id>` returns `found:false`
   (no-plan-yet) until Monday's read — never a 500.
3. **FE renders** — the dashboard PlanCard shows the timeline + tabs + bell; the
   ✎ + 💬 door is enabled; the Studio "Day Plan" block renders.
4. **Monday 08:25 CT** — the first NY plan row is written, the card lights (bias +
   levels + scenarios), and advisory citations begin (`📋 advisory:` in the log).
5. **Door smoke** (optional, on Monday's live plan): tap a level → edit → Save
   bumps the version; 💬 Ask-Planner returns a structured reply; a fat-finger
   price is 422-rejected.

**Deferred (post-★2, owner-run):** the blind-mark owner hour (10 days per the axes
above) · the first weekly matched-random freeze (fires the next Sunday, shows
WARMING). **vlauto:** DEFERRED — one propagation train after ★2 (format-patch from
this nofx HEAD; build + goldens green there + secret-scan; owner runs the push).
