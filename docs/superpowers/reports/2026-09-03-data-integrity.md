# DATA-INTEGRITY (checklist class 66)

**Branch:** `fix/data-integrity` off `5ebeb5a2` · **Status:** NOT DEPLOYED, rides the next boot
**Checklist:** entry **63** (highest occupied at merge: 62 — number assigned AT MERGE, A16)
**Boot lines:** `🧮 data-integrity: …` · `🧮 e8: …` · `🔗 attribution: …`

**Basis.** The dispatch cites a 2026-09-02 read-only audit. It is not in the
repo — not on `dev`, not in the eight audit-shaped remote branches, and a grep
for its distinctive numbers (`58,409`, `0.9984`) across refs returns nothing.
The owner confirmed it was a chat closeout that was never pushed. **Every claim
below is my own measurement**, cited to file:line and to live rows.

---

## 1. D1 — one price space (`kernel/shadow_ab.go`)

`shadow_ab.go:69-76` mirrored stop/target/ref into a NEGATIVE price space for
shorts "so MFE is always favorable-positive in the replay". The close-rule fill
(`:120`) returns the REAL close and `row.StopPx/TargetPx` (`:166`) are stored
REAL, so everything downstream subtracted one space from the other:

```
risk := row.FillPx - stop      →  29204.50 − (−29226.00) = 58 430.50
row.RR = (target - FillPx)/risk → (−29132.50 − 29204.50)/58 430.50 = −0.9984
```

### Measured, direction from the PLAN and never inferred

All 188 rows resolve to their plan scenario, so direction is read, not guessed:

| direction | rows | RR < −0.9 | \|MAE\| > 1000 | recomputable |
|---|---|---|---|---|
| long | 67 | 0 | 0 | — |
| short | **121** | **109** | 46 | 109 |

My own first pass classified direction by geometry and got 55 shorts. That was
wrong — a short whose fill drifted past its target does not look like one — and
it is the same mistake the backfill must not make.

### A second consequence nobody had named

`above` was flipped AND `ref` negated, so the short close-rule became
`b.Close > -ref` — a real close against a negative number, **true for every
bar**. The FILL BAR was wrong, not just the arithmetic on it. This is what
limits the backfill (§5).

### E1 — quoted RED, then GREEN

RED, reproducing the live row from a synthetic fixture:

```
RR  = -0.998399767    (stored row 166: -0.998399808319285)
MAE = 58418.0000      (stored row 166: 58409.25)
net_pnl -116672.00 on a target hit
```

GREEN: `RR 3.348837209` (reward 72.00 / risk 21.50), MAE inside `[0, 21.50]`, a
target hit profits. Long-side pin asserts that side is untouched; five
pre-existing shadow tests unchanged.

## 2. D3 — the lifecycle log (`store/plan.go`)

`UpdatePlanLifecycle` wrote the lifecycle marker INTO `trigger_reason`, so a row
could answer "why parked" or "why authored", never both. Live rows that lost
their authoring trigger:

```
2026-08-27:ASIA … v7  active   "rearmed:2x5m close back below 29678.25 …"
2026-08-28:NY   … v2  dormant  "dormant:death:death-condition: 15m_close …"
2026-08-26:ASIA … v10 dormant  "dormant:flip:flip-condition: 15m_close …"
```

`plan_lifecycle_log(plan_id, version, event, reason, at)` takes the transitions;
`trigger_reason` is the authoring trigger and nothing else. Caller signatures
unchanged. The append is telemetry — a failure WARNs and the transition stands
(class 23).

## 3. D5 — pre-era label (`store/attribution.go`)

`CountUnstampedClosed` had no era filter, so the boot line read
`unstamped-closed=516 (pre-era history)` — calling the same rows unstamped AND
pre-era in one breath. Two counts now, both scoped by `DayPlanEraStart` (the
named constant, never a typed epoch). **Live: `pre-era=516 ·
unstamped-closed=0`** — E5 exactly. An unstamped row inside the era is a live
defect and can no longer hide inside a number that is 99% history.

## 4. D2 — ordinal seed (`kernel/touch_telemetry.go`)

`touchRegistry` is process memory and `touchLevelState.opened` starts at 0, so
`TouchEpisode.Number` restarted at 1 on every boot while closed episodes kept
persisting. The live skew:

```
touch_number  1 → 513 rows · 2 → 229 · 3 → 131 · 4 → 95 · 5 → 62 · 6 → 34
```

The registry key carries the session-day now, and a registry MISS seeds from
`TouchEpisodeStore.MaxTouchNumber` — the store the sink writes to, installed
beside that sink. The seeder is injected (kernel cannot import store); nil or a
negative answer means NO seed, so behaviour degrades to the old one rather than
inventing an ordinal.

**Nearly shipped a class-28 bug inside a class-28 fix:** I first derived the
session-day as `CMESessionDayStart(now).Format("2006-01-02")`, a second copy of
the format the sink already produces with `CMESessionDayKey`. A different format
there makes the seed query miss every row it exists to read. It uses
`CMESessionDayKey` now.

The 1B lane has not shipped a store-read ordinal (checked
`origin/fix/live-detector-1b` @ `a61a5e5d`), so this is the one implementation;
their episode logic is untouched.

## 5. D6 — the backfill, and why it recomputes 55 and not 109

Flag-guarded (`E8_BACKFILL`, default OFF, `.env` not `export` — A24), backup
first via the online `sqlite3 .backup` (**no backup, no write**), idempotent.

### The first version was wrong in the way this wave exists to prevent

Re-deriving arithmetic for all 109 short rows produced **23 still with negative
RR and 14 still with impossible MAE — all labelled `recomputed`**. Clean
arithmetic on a fill that came from the wrong bar is a precise answer about the
wrong moment.

### Three states, not two

A short whose stored stop sits BELOW its fill, or whose target sits ABOVE it, is
geometrically impossible: that row's fill came from the wrong bar and no
arithmetic on it can be trusted.

```
scanned=188  recomputed=55  no_inputs=12  no_direction=0  bad_fill_bar=54  longs_untouched=67
second run recomputed=0 (idempotent)

recompute                  n   bad_rr  bad_mae
(long, untouched)         67       0        0
recomputed                55       0        0      ← every one clean
unrecomputable:fill-bar   54      54       14      ← labelled, numbers kept
unrecomputable:no-inputs  12       0        5
```

`usable=55` is the count a ruling may rest on. Owner-ruled 2026-09-03.

**`net_pnl` on OPEN rows is cleared**, not left holding a mixed-space value:
this table does not store the last close, so it is not derivable here.
`recompute=recomputed` + `outcome=open` + `0` reads as "not derivable"; the
column disambiguates it from a real zero. The 88 zeros on open rows were always
correct and are untouched — folding them in would have made this look three
times worse than it is.

## 6. D7 — boot lines, all READ

```
🧮 data-integrity: e8-one-price-space=on · ordinal-seed=store · lifecycle-log=on · pre-era-split=on
🧮 e8: rows=188 usable=55 · unrecomputable fill-bar=54 no-inputs=12 no-direction=0 · backfill=off
🔗 attribution: … · pre-era=516 (history — never a plan to find) · unstamped-closed=0 (day-plan era; >0 is a live defect)
```

Seam exclusion is deliberately NOT claimed: D4 shipped on
`fix/seam-never-graded` and belongs to that wave's line. A boot line that claims
a neighbour's work is how a surface starts lying.

## 7. Tests

| id | pins | first run |
|---|---|---|
| E1 | row-166 short: RR, MAE bound, target-hit profit; long side untouched | **RED quoted** |
| E2 | seed 3 → next 4 · new day → 1 · no seeder → 0 · negative → 0 · day-scoped key | RED |
| E3 | author → park → re-arm keeps the authoring trigger; park readable from the log | RED |
| E5 | 516 pre-era → unstamped-closed=0; an era row with no plan → 1 | RED |
| E6 | backfill counts, three states, longs byte-identical, idempotent, direction never inferred | RED |
| E8 | goldens · full Go 0 failures · vitest 41/310 · tsc clean | GREEN |

Four superseded specs migrated with their reasons, none weakened. One of my own
migrations over-asserted (that the authoring trigger is non-empty after a park —
that fixture appends its plan without one) and was corrected to assert what the
wave actually claims.

## 8. Cutover

Rides the next boot. `E8_BACKFILL=1` as a line in `/home/hoang/nofx/.env` for
one boot, then removed — the unit has no `Environment=`, so an `export` never
reaches the process. Gate, GO, A13, A19 four halves, marker pushed.

**PROOF OWED:** the first live short E8 row with a sane RR; an ordinal that
continues across a restart; a parked plan whose row still shows its authoring
trigger.
