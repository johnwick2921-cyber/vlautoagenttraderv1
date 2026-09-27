# ADVERSARIAL VERIFY — B2b BD_CONFIRM_CLOSES / BD_MIN_CLOSES  (verdict: peer's headline REFUTED)

Code identical at deployed 70af663d, dev tip 492d2067, worktree HEAD c28fd337
(`git diff 70af663d..HEAD -- kernel/breakdown_continue.go` = empty). [A]

## What reproduces (peer CORRECT)
- resolved = **1** [A]. Boot-8 `main.go:431` 09-04 08:30:11 CT (PID 878451 log):
  `entry law: bd_min_closes=1 …`. Printed by `kernel/entry_law.go:94-95
  EntryLawBootLedger()` calling `bdConfirmCloses()` itself — resolver, not a file default.
- env unset both places: no `BD_*` in `.env`; no `BD_*` in `/proc/878451/environ`. [A]
- file:line exact: `kernel/breakdown_continue.go:65-72` (func), `:260-263` (REJECT block). [A]
- effect REJECT-at-write, pullback-only — `:260 if !immediate && !st.Leg1Met`. [A]
- entry path `trader/auto_trader_planner.go:1722` → `kernel.ValidateBreakdownContinueScenarios`
  (`breakdown_continue.go:213`) is real. [A]
- NOT dead: live reject fires **n=8** (`data/nofx_2026-08-30.log` 3, `2026-09-04.log` 5),
  message renders "(1 confirming close(s) needed)". [A]

## What does NOT survive
1. **"research value 2" is not a research value.** `2026-08-30-knob-census.md` legend **:14** —
   `[C] | Code-canon — fixed constant or shipped default, no external citation found`; the row
   **:42** is `[C]`. `2026-09-02-belief-census.md:42` labels B2 `[I]` (invented/untested).
   Third and only other mention `2026-08-28-waterfall-class-wave.md:18` is a BUILD description
   ("re-checks at write time: >=2 closes beyond the level (BD_CONFIRM_CLOSES)") — no n, no sweep,
   no tape. **No report asserts 2 as researched.** A24: no research value to print beside 1.
   Correct verdict = **conformance N/A ([I], zero research grounding)**, not "NO".
2. **Caller count wrong: they said 2, actual 6** (0 test-only):
   breakdown_continue.go:142 · :262 · plan_confirm.go:192 · :315 · :316 · entry_law.go:95
3. **Effect understated.** `plan_confirm.go:192` makes bdConfirmCloses() NAME the runtime confirm
   rule `fmt.Sprintf("%dx5m_close", …)` -> "1x5m_close", inside `EvaluateScenarioConfirm:189`,
   reached in production at `trader/auto_trader_levelstate.go:250`. Second, non-write-time effect.
4. **Real defect is a DEAD KNOB NAME, not a drifted number.** `BD_CONFIRM_CLOSES` has not existed
   in any .go file since E3 `60faefbd` (2026-08-30 16:12:05 CT) renamed it to `BD_MIN_CLOSES`
   (`breakdown_continue.go:66`). `git log --all -S"BD_CONFIRM_CLOSES" -- '*.go'` = only 71f0c39f
   (add) + 60faefbd (remove). Setting BD_CONFIRM_CLOSES=2 today changes nothing.
5. Timeline overstated: true for belief-census (ee64a494, 09-02, 3d after E3); the 08-30 knob
   census is pinned only by its ARCHIVE commit 741bfc2a (09-01 07:58:16), so its authoring time
   vs E3 16:12:05 on 08-30 is not established.

## git log -1 for every report cited
741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX
ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02
4016025dd18b148470541ced6e0c3b18acf9f116 2026-08-28 12:05:54 -0500 docs: pin waterfall-class report URL to the wave commit
