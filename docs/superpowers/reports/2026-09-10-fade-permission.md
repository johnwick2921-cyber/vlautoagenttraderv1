# W2 — FADE PERMISSION: is the book allowed to fade right now?

**Branch** `fix/fade-permission` · **base at accept** dev `580e88b3` · rebased onto
`29eda2b3` (103's W-TF) and `4e9289f7` · **running rev verified** `adae3bb41b31`
from BOTH `/api/health` and `/proc/2560377/exe` (sha256 identical to `nofx-bin`).
**Scope (A31)** RECORD and SHOW. No gate, arm, order, scenario, level, exit or
cadence changed. `trader/fade_no_refusal_test.go` fails if the arm path can read
the label.

---

## THE HEADLINE — E3's NULL, STATED IN ADVANCE

> **On the one day we have, the label would have PERMITTED the damaging trade.**

`2026-09-03` NY authorized exactly three arms — ids **35, 36, 37**, the full n.
The one that **filled** (35, short 29285.00, 09:02 CT, stop 29351.63; price ran to
29543.75) is covered by **none** of the five exclusions. Not because they were
mis-tuned: because the day did not announce itself in the first hour.

| arm | CT | (a) OR wide | (b) IB held | (c) beyond map | (d) T1 | (e) first-N |
|---|---|---|---|---|---|---|
| **35 FILLED** | 09:02 | no — 0.77× | N/A, IB not yet | no (29267.5 < 29498.5) | UNKNOWN | no |
| 36 | 09:20 | no | N/A, IB not yet | no (29350.75 < 29351.05) | UNKNOWN | no |
| 37 | 11:58 | no | **fires** | no (29490 < 29619.5) | UNKNOWN | no |

The label has **no known coverage**. E3 measures, over 20 sessions of stamped
episodes, whether any exclusion acquires some. Shipped anyway, on the owner's
ruling: a label that admits it caught nothing is more useful than no label,
because it records the exclusions' verdicts per episode, which is the only way
E3 can ever find one that works.

---

## C · THE EVIDENCE, each reproduced at `adae3bb4`

### C1 — no permission step exists · REPRODUCED

16 production references to `day_type`, **0 gates**:

| use | n |
|---|---|
| DISPLAY (API render/diff, digest, doc) | 8 |
| AUTHORING (prompt text) | 2 |
| SCHEMA (field, `no-trade` default) | 2 |
| FIXTURE (golden, sandbox seed) | 3 |
| LOG (token-budget string) | 1 |
| **GATE / REFUSAL / AUTHORIZATION** | **0** |

Arm path: `armed_executor.go` 0 · `entry_gate.go` 0 · `auto_trader_orders.go` 0 ·
`session_risk.go` 0. `armed_executor.go`'s single `regime` hit is a comment at
:2006. **`day_type` is model-worded free text** — ten distinct values, including
`trend-down extension / oversold reversal watch`; a naive `= 'trend'` misses 77
of 269. It is never an input; `TestFadeFactsCarriesNoDayType` reflects over the
struct.

### C2 — the book faded the trend · REPRODUCED, and it refutes the dispatch's exclusions

- OR 5m = **62.25 = 0.77× the 13-session median 81.25**. Below median.
- IB 08:30–09:30: high 29375.25, low 29199.25, size **176.00** (median 177.00).
- 09:30 close 29287.75 — **inside** the IB. First closed 5m bucket beyond: **10:00
  at 29436.75**, thirty minutes after the dispatch's deadline for (b).
- The planner re-seated ahead of price all morning: max seated v4 29375.25 →
  v5 29539.38 → v6/v7 29619.50. Price never cleared the authored map. **(c) is
  structurally blind to a map that moves with the run** — A15, a planner finding.

Owner rulings taken: **(b) continuous** (fires 09-03 from 10:05); **(a) ships
`[I]`** with "would not have fired on 2026-09-03 (OR 0.77× median)" in its help
text; **all five ship**, each carrying its own non-coverage.

### C3 — 155 of 258 trend-labelled were fades · REPRODUCED with a split

299 plans, **852 scenarios**, 0 parse failures. Trend-labelled: **269**.

| definition of "fade" | count |
|---|---|
| by **condition** (`reject` + `sweep_reclaim`) | 102 + 55 = **157** / 269 (58%) |
| by **direction opposing the plan's bias** | **64** / 269 (24%) |

The audit's 155/258 reproduces as the condition definition. The two disagree 2.5×
and nothing in the codebase says which is "fade". The stamp sidesteps it (every
scenario is labelled); the counter counts **episodes**, not fades.

W1's episode rows: `opportunity_outcome` NULL on all 5,050; `scenario_nearest` NULL
on all 5,050. **E3's comparison starts from this boot.**

### C4 — knowable at the open

| input | available from |
|---|---|
| OR 5m | 08:35 CT (the 08:30 bar closes) |
| IB | 09:30 CT, then continuous |
| beyond the map | any tick, from the seated map at that evaluation |
| T1 calendar | boot, **iff a real slice exists**; else UNKNOWN |
| first-N | session open + `kernel.FirstNoTradeMinutes()` = 5 |

The builder enforces this: bars opening at/after `now` are not read; a 5m bar is
closed only once `now ≥ open+5m`. Mutation-verified (E10 #4).

### C5 — the first-hour range on this tape

| | median | p80 | min | max |
|---|---|---|---|---|
| OR 5m | **81.25** | **104.05** | 33.75 | 137.00 |
| IB 60m | **177.00** | **203.95** | 44.50 | 262.25 |

**n = 13** complete session-days (`2026-08-25 … 2026-09-10`, each with 12 IB
bars). The dispatch's "prior-20-session" median is **not computable** — the 5m
tape starts 2026-08-24. `PriorSessionORMedian` returns the n it finds and the boot
line prints it. Default k = p80/median = **1.28** `[I]`. 2026-09-07 is Labor Day
(OR 33.75) and is in the sample.

### C6 — the episode row

Every column `notnull=0`, no default; W1's own additions (27–38) follow it.
Additive NULL-able is the seam. `fade_permitted` is `*bool`, not `bool`.

---

## D · WHAT SHIPPED

- **D1** `kernel.FadePermissionAt(now, facts)` — pure; five exclusions,
  independent, all named; coverage notes in the source.
- **D2** four columns on `touch_outcomes`; fixed at open by predicate
  (`WHERE fade_permitted IS NULL`); stamp call site in `detector_record.go` at
  episode open with the episode's own clock; `recover()` pinned.
- **D3** `trader/fade_no_refusal_test.go` — the seven arm-path files cannot
  reference the label.
- **D4** `FadePermissionChip` on the card via `/api/plan/today.fade_permission`;
  desk strip SCENARIOS line carries the chip per scenario and
  `fade permitted N / excluded M / not-evaluated K today`.
- **D5** `🚦 fade permission:` boot line — counts READ, k's resolver named,
  backfill three-state, coverage note in the text.
- **D6** `BackfillFadePermission` — recomputed / unrecomputable:`<why>` /
  untouched, clock at each episode's open.
- **A12** Guide section, SYSTEM-MAP section, class **115** — same commit.
  **RULEBOOK §A: not possible** — see A17 below.

## E · TESTS — RED quoted first

| pin | RED (quoted) | GREEN |
|---|---|---|
| E1 predicate `TestSep3…` | `09:02 must be EVALUATED (OR was complete), got not-evaluated` | ✓ |
| E1 builder `TestSep3ReplayThroughTheBuilder` | passed first run → mutation #4 | ✓ |
| E2 `TestUncompletedOpeningRange…` | `08:36: the OR bar HAS closed — must be evaluated` (strengthened after a vacuous pass) | ✓ |
| E3 `TestArmPathCannotReadTheFadeLabel` | guard → mutation #3 | ✓ |
| E4 `TestEveryFiringExclusionIsNamed` | `both or_wide and ib_held must be named; got []` | ✓ |
| E5 `TestAbsentCalendarSlice…` | `t1_news must be reported UNKNOWN; got unknown=[]` | ✓ |
| E6 `TestEpisodePermissionIsFixedAtOpen` | `undefined: FadeStamp` → assertion RED → mutation #2 | ✓ |
| E7 `TestThresholdCarriesItsResolvedProvenance` | `or_wide must fire at 200 vs median 81.25` | ✓ |
| day_type `TestFadeFactsCarriesNoDayType` | guard, reflects the real struct | ✓ |
| chip (vitest ×4) | absent → `not evaluated`, never permitted | ✓ |

**E10 mutations — all with the changed line quoted, sed confirmed applied, build GREEN:**

| # | file:line | mutation | died on |
|---|---|---|---|
| 1 | `kernel/fade_permission.go:149` | `ratio > k` → `ratio < k` | 3 fixtures incl. the 09-03 pin — which catches it *because* it encodes non-coverage |
| 2 | `store/fade_stamp.go:66` | drop `AND fade_permitted IS NULL` | `permission changed after open … exclusions rewritten after open: "ib_held"` |
| 3 | `trader/armed_executor.go:198` | `_ = kernel.FadePermissionAt` | `D3 NO-REFUSAL: armed_executor.go:198 references the fade label` — the FIRST mutant did not compile and read as dead; redone compiling |
| 4 | `trader/fade_facts.go:79` | closed-bar rule → `if true` | `10:04: the 10:00 bar closes at 10:05 — reading it early is reading the future` |

**E8 (A29)** against the claim list, registered by **wrapper** name: stamp / backfill /
boot line / counter call sites each removed → RED each. Build green each time.

**E11 at the merged head** (`4e9289f7`): Go **31 ok / 0 FAIL**, goldens ok, tsc ok.
Four of the repo's own guards caught this wave first (TZ layout; unregistered
knob ×2; nil bars provider blanking the SCENARIOS line) — all fixed, all in one
commit. **vitest: 11 files red from one pre-existing cause** (vite 6.4.3 fs.allow
on `../branding/product.txt?raw`, W1 §G1), zero assertion failures, my 9 pins
green in isolation. The split-arm fixture is **green** at this head.

## A15 · WHAT THE LABEL CANNOT YET SEE

1. **A trend that starts after the IB with a narrow open** — 09-03 exactly.
2. **A planner that re-seats ahead of price** blinds (c) by construction — a
   finding for the planner, not this wave.
3. **"Fade" has two definitions**; the counter counts episodes to stay honest.
4. **The OR median is over 13 sessions**, not 20.

## A17 · PREMISES REFUTED

- "(a) opening range too wide" — **0.77× median on 09-03.**
- "(b) IB broken and held **by 09:30**" — **break at 10:00.**
- "(c) price beyond the map" — **the map moved with the run.**
- "the corrected rulebook … (PR #99, **on dev**)" — **PR #99 is OPEN; the file is
  on no path in `origin/dev`.** A12's "same commit" is impossible for a file not
  on the base, and I do not edit another lane's open branch. For whoever lands
  #99, §A gains: *"Every scenario carries a fade-permission label (permitted /
  excluded with the exclusion named / not evaluated), fixed at the episode's
  open. It does not gate."*

## E3 · PRE-REGISTRATION

- **Unit:** the W1 episode row, stamped at open.
- **Split:** `fade_permitted = 1` vs `= 0`, NULL excluded and counted.
- **Outcome:** `opportunity_outcome` rung and `pnl_corrected` where filled, era-filtered.
- **Null:** the exclusions' verdict is uncorrelated with outcome — as they were
  on 09-03.
- **Minimum effect:** an exclusion earns a rule only if excluded episodes show a
  worse filled outcome at n ≥ 20 excluded across ≥ 20 sessions. **Held-out** days
  decide, per round 11 §1.

## ROLLBACK

`mv nofx-bin nofx-bin.failed.<rev> && mv nofx-bin.old.<prev> nofx-bin && echo <prev> > deploy/RELEASE && kill -9 $(pgrep -x nofx-bin)`. The columns are additive and NULL; nothing reads them to act.

---

## F · CUTOVER + PROOF — booted 18:47:07 CDT, marker `4dc0fae1`

`🔐 BOOT INTEGRITY OK — rev 4fc670aa4508 · expected 4fc670aa · goldens PASS`.
Clean clone in a dir named `nofx`, **`vcs.modified=false`**. Five references +
`/proc/exe` all `4fc670aa4508`. Leg 5 went in flight between gate and swap;
waited out (1004s), re-read five-for-five, then the kill.

**The line, verbatim:**

```
🚦 fade permission: LABEL ONLY (no refusal) · exclusions=[or_wide k=1.28[I:default:C5 p80/median]
ib_held[I:continuous] beyond_map[I] t1_news[O:UNKNOWN-never-excludes] first_n=5[O:no-trade-band]]
· today permitted=7 excluded=0 not-evaluated=0 · backfill recomputed=1296 unrecomputable=3844
untouched=0 · coverage: on 2026-09-03 the label would have PERMITTED the filled arm (id 35, 09:02)
— E3's null
```

| proof | evidence |
|---|---|
| backfill, table (via WAL) | permitted 912 + excluded 413 = 1325 · unrecomputable 3844 · untouched 0 |
| first episodes under the new binary | ids 5148–5166, ASIA, stamped at their own open |
| first `excluded` with measured values | ids 5144, 5160: `first_n {"measured":4,"threshold":5}` |
| E4 live | episode 4840: `ib_held,t1_news` — both named; `t1_news` **fired** |
| **D3 in production** | arm 150 (S2 SHORT @29123.5) went out at a level whose episodes 4840/4841 are `excluded`; all 13 arms today went out |
| chip + strip | S1/S2 `fade: excluded — ib_held price 29091.00 held beyond IB 29099.50`; counter `permitted 11 / excluded 0 / not-evaluated 0` |

**A correction against myself.** My first table read said recomputed=1014 and I
chased a phantom "294 rows counted but not written". I had copied `data.db`
without its 10 MB `-wal`; the backfill walks `opened_at ASC`, so today's rows sat
in the WAL. Read through the WAL, every number reconciles. Class 110, different
variable.

**Owed:** the backfill's stamp omits `fade_measured` (live path fills it); the
RULEBOOK §A sentence waits on PR #99.

**Sha-pinned:** report at `de26d1e4`+marker `4dc0fae1`; size = `git ls-tree`.
