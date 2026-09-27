# Adversarial verification — Queue #1 min-SL floor (ATR leg + level clearance)

VERDICT: **REFUTED** (headline `conforms=NO` is wrong). One sub-finding (prompt drift) SURVIVES
but is already on the record as drift D-2.

## What was re-derived (never a file default)

| item | value | source | label |
|---|---|---|---|
| boot line (running PID 878451) | `min-sl guard: atr_mult=1.5 level_clearance=2tick(s)` | `data/nofx_2026-09-04.log` 07:38:40 + 08:30:11 CT | [A] |
| boot line is RESOLVED, not literal | `kernel.MinSLATRMult()` (env-aware) + `kernel.MinSLTickClearance` (const) | `trader/auto_trader_dayplan.go:57-58` | [A] |
| env override | `MIN_SL_ATR_MULT` **absent** from `/home/hoang/nofx/.env` → default path | grep | [A] |
| live REJECT string | `sl_too_tight: 14.5 < 1.5×ATR (10.7) — widen or skip` | `nofx_2026-09-03.log` 19:10:58 CT, `kernel/engine_position.go:229` | [A] |
| live arm WARN string | `too close (38.50 < 50.73 = 1.5×ATR5m)` | `nofx_2026-09-04.log` 08:09:03 CT | [A] |
| persisted resolved floor | **n=26** `planner_read_facts` rows, `stop_floor_mlt` min=1.5 max=1.5 | `data.db?mode=ro` | [A] |
| deployed-rev parity | `git diff 70af663d -- kernel/min_sl.go` = EMPTY | worktree | [A] |
| const line numbers | `:34 MinSLATRMultDefault = 1.5` · `:40 MinSLTickClearance = 2` | grep -n | [A] |

## Live effect, with n (peer quoted none)

| day | `MIN-SL REJECT` (executor) | arm `too close` |
|---|---|---|
| 2026-09-02 | 24 | 55 |
| 2026-09-03 | 10 | 3 |
| 2026-09-04 (to 09:05 CT) | 0 | 6 |
| **total since 0B** | **34** | **64** |
| 2026-08-30 (pre-0B, 1.0×) | 2 | 14 |
| 2026-09-01 (pre-0B, 1.0×) | 2 | 21 |

## Where the peer's claim breaks

1. **"queue #1 asked for WARN-first at 1.0×" — the `1.0×` is INVENTED.**
   Verbatim, `2026-09-02-belief-census.md:129` (ee64a494):
   `| 1 | **min-SL 1.0×ATR5m + 2t** | [I] | REJECT at write (…) | WARN-first (author + WARN, place if owner-set otherwise); sweep the multiplier (knob census queue) |`
   The `Demote to` column asks for WARN-first **and "sweep the multiplier"** — i.e. the number is
   declared UNMEASURED. `1.0×` sits in the *Belief* column: a description of the then-current code,
   not a research recommendation. There is no research value to place beside 1.5.

2. **The grounding document is self-refuting at the line range it cites.**
   Census `:46` cites `kernel/min_sl.go:23-29` for "min-SL ≥ 1.0×ATR5m".
   `git show ee64a494:kernel/min_sl.go | sed -n '23,29p'` at the census's OWN pinning commit already
   reads *"1.5 is the bottom of the researched range…"*. The 1.0→1.5 change landed at
   **4657560b 2026-09-02T07:33:39-05:00**; the census committed **2026-09-02T08:50:38-05:00** —
   the census was **stale-at-authoring by 77 minutes**.

3. **Label is [R]/[O], not [R] — and [O] is what settles conformance.**
   `kernel/min_sl.go:20` — `0B (owner ruling 2026-09-02): 1.0 → 1.5.` A census demotion queue is a
   *proposal*; an owner ruling is `[O]` in the census's own legend (`:10-19`). The audit's own merged
   row already says so: `2026-09-04-research-conformance.md:671` → `[R]/[O] … REJECT … CONFORMS: yes`.

4. **Production callers: 8 is an UNDERCOUNT; the rule is emphatically not dead.**
   15 non-test invocations of `kernel.MinSLATRMult()` across 8 files, 19 counting
   `MinSLTickClearance` / `MinSLVerdict` / `MinSLAnchorFor`:
   `main.go:335,338` · `kernel/engine_position.go:215,227,236,241` · `trader/entry_gate.go:367,427`
   · `trader/no_chase.go:91,147` · `trader/armed_executor.go:391,392,395,1361,1362`
   · `trader/auto_trader_planner.go:1681,2381,2428` · `trader/auto_trader_dayplan.go:58`.

5. **REJECT file:line is understated.** `:250` is the *level-clearance* leg only. The primary
   ATR-leg REJECT is `kernel/engine_position.go:228-230` (`IncGateBlock` → `MIN-SL REJECT` → `return`).

## What SURVIVES (and is not new)

`kernel/planner_prompt.go:733` (arm feasibility contract: *"the stop distance must be ≥ 1.0× the
current 5m ATR"*) and `:752` (waterfall immediate path: *"min-SL ≥ 1.0×ATR5m"*) are **real**, both
inside `plannerOutputContract` (`:683`), which `BuildPlannerPrompt` writes at `:674`. [A]

BUT the peer's framing — *"still tell the model the floor is 1.0×"* — is a half-truth: the **same
prompt** writes `RenderStopFloorLine` at `:460`, which renders
`## Minimum stop distance this cycle / X pts (1.5×ATR5m Y, resolved)` from the gate's own resolver
(`kernel/class45_feeds_forward.go:195`), fed by `StopFloorMult: kernel.MinSLATRMult()`
(`trader/auto_trader_planner.go:2381`) — proven live by 26/26 persisted rows at 1.5. The prompt is
**self-contradictory**, not uniformly stale. Already recorded as **drift D-2**
(`2026-09-04-research-conformance.md:671`).

## Pinning

    2026-09-02-belief-census.md          ee64a494 2026-09-02T08:50:38-05:00
    2026-09-04-research-conformance.md   c28fd337 2026-09-04T09:00:24-05:00
    docs/superpowers/AUDIT-CHECKLIST.md  158743db 2026-09-04T08:11:57-05:00
    docs/superpowers/research/INDEX.md   4e8e7e1a 2026-09-03T19:37:14-05:00
    kernel/min_sl.go                     4657560b 2026-09-02T07:33:39-05:00 (0B: 1.0→1.5)

## Caveats
- `telemetry.IncGateBlock` is an **in-memory map** (`telemetry/gate_blocks.go:33,43`) with no DB
  table; `/api/risk/gate-blocks` returns `{"error":"Missing Authorization header"}` from this
  session. The n above is from the process log files, not the counter API.
- "Round-7 research" (cited `kernel/min_sl.go:21`) exists ONLY as an in-code citation — no
  document of that name is in `docs/superpowers/`; `docs/superpowers/research/` holds only
  `INDEX.md`. The 1.5–2.5×ATR range and ">60% stop-out below 1.0×" are therefore **unsourced to a
  locatable report**. That weakens `[R]` — but does not restore the peer's `1.0×`, which is
  unsourced by the same test and additionally contradicted by `[O]`.
- `planner_read_facts.created_at` is in **seconds**, not ms — a `ms/1000` conversion yields 1969.
