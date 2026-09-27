# NO-TRADE BAND (checklist class 51) — session-scoped, config-driven, judged at read time

**Wave:** no-trade-band · worktree `~/nofx-band` off `origin/dev` 0bba743b
**Commits:** 52903708 · c972075d · 091f83ee · d0b6e594 · fa15f2d9 (+ this report)
**Boot line:** `🗓 no-trade band: …` (rendered below, every field resolved)
**Executor diff:** none. No gate, arm, size or order path changed behaviour.

---

## 1. What was wrong

**The pin.** The ASIA card at 23:00 CT rendered three no-trade rules. None of
them could refuse an entry.

| shown as a live rule | what it actually was |
|---|---|
| first 5m after the open | closed at 17:05 — six hours earlier |
| lunch 12:00–13:30 CT | an NY window; ASIA runs 17:00–02:00 and never touches it |
| 🔴 ISM PMI 09:00 CT ±15m | fired fourteen hours before the card was opened |

The card was rendering `doc.no_trade`: prose the model wrote when the plan was
authored, stored, and displayed verbatim for the rest of the session. Nothing
in the path asked what time it was.

**The same defect's second face.** Each fixed window was defined more than once
and the copies could not disagree loudly:

| copy | where | consumed by |
|---|---|---|
| `cur < start+5` | `trader/auto_trader_session.go` `inSessionFirst5m` | the entry gate |
| `InBlackoutWindow(t, "12:00", "13:30")` | same file | the entry gate |
| its own first-5m test + `"12:00"`/`"13:30"` | `kernel/adherence.go:121` | the adherence grader |
| `"first 5m (CT)"`, `"12:00-13:30 CT lunch"` | `kernel/planner_prompt.go` schema example + rule sentence | the model |
| whatever the model wrote | the plan doc | the card |

Five statements of two windows. The gate refused one, the grader scored
another, the prompt taught a third and the card claimed a fourth. No test
could fail if any pair drifted, because nothing compared them.

**And a claim that was never measured.** `WidenCTWindows` appended
`+%dm (clock drift)` whenever `driftWidenMinutes` returned nonzero, and that
function rounds ANY nonzero offset up to at least one minute. This machine's
NTP offset is 108 ms — healthy. Every T1 line in every plan therefore carried
"+1m (clock drift)", baked into the stored text at authoring time, telling the
reader for the rest of the day that the machine's clock was drifting.

---

## 2. What changed

### 2.1 One definition each — gate, grader and card read the same window

`kernel/no_trade_band.go`:

```go
func FirstNoTradeMinutes() int                        { return 5 }
func LunchWindowCT() (startCT, endCT string)          { return "12:00", "13:30" }
func InFirstNoTradeMinutes(startCT string, t time.Time) bool
func InLunchNoTrade(t time.Time) bool
```

The gate (`sessionGateDecision`) and the grader (`adherence.go:121`) now call
these. Every payload built from them is stamped `source: "code-constant"`, so
the surface never implies a knob that does not exist. **These are not
owner-configurable and the card says so.**

A parity test asserts the gate, the grader and the card return the same answer
for the same clock across seven boundary minutes.

### 2.2 The machine writes the windows; the model does not

The plan doc gains an additive `no_trade_windows` field. It is written at plan
time from enforcing sources only:

| kind | source | comes from |
|---|---|---|
| `first_n` | `session-registry` | `BuildMachineNoTradeWindows(sess)` — the gate's own definition, keyed on the session's real open |
| `lunch` | `code-constant` | same |
| `t1` | `calendar` | `at.t1WindowsFor(tradeDate, sess)` |

`t1WindowsFor` was extracted from `currentT1Windows` for one reason: the old
function keyed on *the session active at `now`*, and a 16:55 read authors ASIA
while NY is still live. The plan card now carries **literally the windows the
entry gate will refuse inside** — clock widening and the P0.6 fail-closed
static fallback included.

`T1NoTradeWindowsFromCT` takes already-resolved `CTWindow`s rather than raw
calendar events on purpose. The widening decision is made once, by the
enforcer. A second computation in the kernel could disagree with the gate, and
a card that disagrees with the gate is the defect this wave exists to fix.

**The model's prose is untouched.** It is stored exactly as authored and now
renders under "Model notes".

### 2.3 Read-time evaluation

`EvaluateNoTradeWindows(wins, nowMin, sessionStartMin, eodMin)` stamps each
window `live` / `elapsed` / `other_session`, asking two independent questions:

1. **Does the window touch this session's tradeable span at all?** Geometry,
   measured from the session start so a session that wraps midnight stays
   monotonic. NY lunch on an ASIA card fails here, whatever the clock says.
2. **If it does, has it finished?** Measured from now on a signed ±12h axis —
   offsets-from-session-start alone cannot tell an 08:00 pre-open read from a
   16:00 post-close one, and the first draft of this function got exactly that
   wrong until the NY pre-open pin caught it.

`GET /plan/today` gains `no_trade_band`: every window with its status and its
CT bounds resolved server-side, so the card does no clock arithmetic. A doc
written before this wave has no machine windows, returns nil, and the card
keeps rendering prose as rules rather than claiming a plan with no limits.

### 2.4 The drift claim, fixed at the measurement

The widening is **byte-identical for every input**. Even a 108 ms offset can
carry an event across a minute boundary, so the one-minute guard is real
protection and the enforced blackout at `auto_trader_calendar.go:185` is
unchanged. Only the claim moved: `driftIsSkew` gates the label at 60 s, the
point where the offset alone can shift an event by a whole minute. Below that
the extra minute is silent boundary rounding.

| offset | widening | label |
|---|---|---|
| 108 ms (this machine) | ±1 min | none |
| 90 s | ±2 min | `+2m (clock drift)` |

### 2.5 The prompt

The `no_trade` schema example and its rule sentence are generated from
`FirstNoTradeMinutes()` and `LunchWindowCT()`. The sentence now also tells the
author the windows are enforced whether or not it lists them, so what it writes
is read as notes.

---

## 3. The boot line

```
🗓 no-trade band: first_n=5m lunch=12:00–13:30 (source=code-constant, shared by gate+grader+card) · T1 taken from the enforcing gate at plan time (widening and fail-closed fallback included) · every window judged against the reader's clock, not the write clock · model prose renders as notes
```

Every field resolved: `first_n` from `FirstNoTradeMinutes()`, the bounds from
`LunchWindowCT()`, the source from the constant that stamps the payload.

---

## 4. Tests

| id | what it pins | RED before |
|---|---|---|
| parity | gate, grader and card agree across 7 boundary minutes | — (shipped 52903708) |
| F1 | ASIA card at 23:00 → 0 live; same doc at 17:02 → first-5m live; pre-band doc → nil | the card had no band to evaluate |
| F2 | NY pre-open read at 08:00 → first-5m + lunch + the 10:00 T1 live; the 07:30 T1 not live | evaluator returned 0 live |
| F3 | a window straddling now reads live | evaluator returned elapsed |
| F4 | literal scan: exactly one copy of each bound at the definition site, none in the prompt or the grader | grader held its own `"12:00"`/`"13:30"` |
| F5 | 108 ms → no drift claim, widening kept; 90 s → claim + 2 min | label read `+1m (clock drift)` |
| F6 | T1 band entries map the enforcer's windows, bounds untouched | — |
| class-38 row | the rendered contract states the resolved windows and their independence | — |
| write-site | the machine stamps the doc; the model's prose is unmodified | **quoted RED**: "the machine wrote NO no_trade_windows" |
| FE ×5 | none-live at 23:00, count of 3, expansion with reasons, live rule at 17:02, pre-band fallback | the block joined prose unconditionally |

Suites: `kernel` `trader` `api` `market` green; vitest 39 files / 303 tests green.

---

## 5. E6 — lunch vs ny_pm overlap (REPORT ONLY, no change)

Measured from the live registry:

```
NY     window 08:30–14:45
        killzone ny_am        08:30–11:00
        killzone ny_pm        13:00–14:45   ⚠ OVERLAPS LUNCH
        band     first_n      08:30–08:35 (code-constant)
        band     lunch        12:00–13:30 (code-constant)
```

`ny_pm` opens at 13:00; lunch refuses entries until 13:30. Thirty minutes of
every NY afternoon are advertised as a prime hunting window and cannot trade.

**The gate is safe.** In `sessionGateDecision` the lunch refusal sits above any
killzone consideration, so no entry fires before 13:30 regardless. The defect
is presentational: the registry and the guide promise a window that is a third
dead. **[A]** — read from the running registry and the gate's own ordering.

Two honest resolutions, owner's call: move `ny_pm` to 13:30, or state the
overlap on the surface. Neither is in this wave.

---

## 6. Known and untouched

- **`kernel/plan_render.go:169` renders `doc.NoTrade` into the plan text.**
  Whatever reads that rendering sees the model's prose, including windows that
  have elapsed — the same lie this wave fixed on the card, on a different
  surface. Out of this wave's footprint (zero executor diff); worth its own
  look. **[A]**
- **Lunch is built for ASIA and LONDON too.** It never intersects their
  windows, so the card collapses it as `other_session`. This mirrors the gate
  exactly: `InLunchNoTrade` is session-independent, so lunch is only ever
  effective during NY. **[A]**
- **`GUIDE_BUILT_REV`** is bumped in the deploy marker commit, per the
  established pattern (the Go binary is built at the content rev; the web
  bundle is built after the marker sets the rev).

---

## 7. Cutover

Not deployed. Preconditions, in order: the wave ahead of this one boots by
name, then `GET /api/cutover-gate` reads `ready: true` on all five legs with no
arm resting, then the owner's explicit GO. A19 all three halves: `deploy/RELEASE`
written in `~/nofx` before the kill, the marker committed from that same tree
after the boot line is observed, and the running binary preserved as
`nofx-bin.old.<its own rev>` first.

---

## 8. BLOCKED — the main tree is mid-cutover under a dead lock (A23: measured, not touched)

Measured at 22:12 CT, read-only. I did not touch `~/nofx`.

| fact | value |
|---|---|
| `~/nofx-main.lock` | `owner=weekly-refs-deploy pid=976198 expiry=1788410353` |
| PID 976198 | **DEAD** (`kill -0` fails); the lock does not expire for ~87 more minutes |
| main tree HEAD | `1cee77a8` (class-50b), 7 commits ahead of `origin/dev`, **unpushed** |
| `deploy/RELEASE` | `1cee77a8` — **uncommitted** (` M deploy/RELEASE`) |
| running process | PID 1093919, `vcs.revision=56904ec1`, up 37 min |
| `nofx-bin` | 56904ec1 (unchanged) |
| `nofx-bin.next` | clean build of `1cee77a8`, `vcs.modified=false`, built 21:39 |
| `nofx-bin.old.56904ec1…` | preserved 21:31 |

**The hazard.** `RELEASE` claims `1cee77a8` while the process runs `56904ec1`.
`kernel.AssertBootIntegrity` reads `deploy/RELEASE` relative to the unit's
`WorkingDirectory=/home/hoang/nofx`, prefix-matches it against the embedded
revision, and on mismatch latches `tradingRefused`. With `Restart=on-failure`,
**any crash or host restart right now boots the bot into TradingRefused.** This
is the exact failure mode recorded on 2026-09-02 07:32. **[A]** — read from the
lock file, `ps`, `go version -m`, the unit and `boot_integrity.go`.

The dead dispatch got as far as: build `.next`, preserve the old binary, write
`RELEASE` and `GUIDE_BUILT_REV`. It never swapped and never killed.

**Two safe resolutions, both the owner's call.** Finish that cutover (`mv` the
staged `.next` in, kill, observe its boot line, commit `RELEASE` from the main
tree), or revert `deploy/RELEASE` and `web/src/guide/types.ts` to `56904ec1` so
the file matches the binary that is actually running. I have done neither.

**Consequences for this wave.** My branch was off `origin/dev` (`0bba743b`) and
did not contain the seven class-50 commits. The checklist collision is resolved:
they took 51, this wave is 52.

### 8.1 Resolved — the peer cutover completed at 22:37 CT

```
22:37:30  nofx.service: Main process exited, code=killed, status=9/KILL
22:37:35  Started nofx.service
22:37:38  🔐 BOOT INTEGRITY OK — rev 1cee77a87f1d · built 2026-09-03T02:38:05Z
                              · expected 1cee77a87f1d · goldens PASS
22:37:38  🛡 cutover safety (class 33): gate legs=5 · leg4=ledger
          · boot sweep cancelled 0 pre-boot arm(s)
```

`nofx-bin.next` was swapped in and killed; the binary, `deploy/RELEASE` and
`HEAD:deploy/RELEASE` now all read `1cee77a8`, so A19's third half is done and
the TradingRefused trap is gone. The stale `weekly-refs-deploy` lock file was
still present at that moment naming a dead PID, so a live dispatch is working
the main tree without holding a live lock — worth a note, not a change I made.

### 8.2 CORRECTION — the merge-tree conflict check was wrong

Section 8 above originally claimed `git merge-tree` reported zero conflicts.
That was a bad probe: I used the three-argument form, whose output does not
carry `<<<<<<<` markers, so grepping for them counted zero by construction.
The real rebase onto `1cee77a8` hit **one** conflict — `kernel/plan_doc.go`,
where class-50b's `BiasLabel` and this wave's `NoTradeWindows` are added at the
same point in `PlanDoc`. Both fields kept; resolved at 56efd352.

Resolving it also caught a comment that had gone stale inside this wave: the
field's doc said T1 windows are deliberately NOT stored and are re-resolved at
read time, which was true of the first draft and false by the time the machine
writer took them from `t1WindowsFor`. Corrected in the resolution.

Rebased branch: suites `kernel` `trader` `api` `market` `store` green, vitest
39 files / 302 tests green.

### 8.3 Rebased again — a third wave landed at 22:41:53 CT

`60f214d9` (void-parity) cut over four minutes after class-50b and booted clean
(`🔐 BOOT INTEGRITY OK — rev 60f214d988bc · expected 60f214d988bc · goldens
PASS`). This wave is rebased onto it. Two conflicts, both kept-both:

- `main.go` — its `📜 VoidScopeBootLine` and this wave's `🗓 no-trade band`
  land at the same point in the boot block. Both lines now print.
- `docs/superpowers/AUDIT-CHECKLIST.md` — see 8.4.

Suites after the second rebase: `kernel` `trader` `api` `store` `market`
green, vitest 39 files / 302 tests green.

### 8.4 The checklist has a DUPLICATE 51 — not this wave's, and not fixed here

Three waves numbered their entry from what they could see at the time:

| line | entry | wave |
|---|---|---|
| 835 | a direction shipped on evidence that never existed | class 50 (`6e1a0781`) |
| 860 | one question, two answers — a shared predicate fed different inputs | void-parity (`60f214d9`) |
| 895 | a rule rendered from the clock it was WRITTEN by | this wave |

The void-parity entry opens "Highest occupied at merge: 50", so it was numbered
against a tree that did not yet carry class-50's entry. **Both are 51.** This
wave took 52, which adds no new collision, and left the duplicate alone rather
than renumbering another wave's entry inside a rebase. Someone owns deciding
which of the two 51s becomes 53. **[A]** — read from the merged file.

The separate renumber commit from before the rebase is gone: it became a no-op
once the conflict resolution numbered this entry 52 directly, and was skipped.

---

## 9. RIDER (owner ruling 2026-09-03) — notes collapsed, and the contract miss

### 9.1 The card

The first cut demoted the model's prose to a "Model notes" label but still
printed it inline. The ASIA card at 00:00 read:

```
No-trade · none live now ▸ 2 spent / other session
Model notes · first 5m (CT) · 12:00-13:30 CT lunch
```

The same two dead windows the wave exists to stop showing, one line lower.
Notes are a toggle now and render nothing until opened. The band line carries
only machine windows live for this session, or "none live". Pinned on the real
v14 doc — its actual prose, its actual machine windows.

Also fixed on the way: the machine's label already carries its CT bounds
("first 5m after the ASIA open (17:00–17:05 CT)") and the renderer appended
them a second time.

### 9.2 The contract miss — REPORTED, not fixed

**Do plans written after the 23:24 cutover still carry model no_trade prose?
Yes.** ASIA v14, written 00:08:45 CT, 44 minutes after the boot:

```json
"no_trade": ["first 5m (CT)", "12:00-13:30 CT lunch"]
```

Those two strings are **verbatim the schema example this wave generates**:

```go
func NoTradeSchemaExample() string {
    ls, le := LunchWindowCT()
    return fmt.Sprintf(`  "no_trade": ["first %dm (CT)", "%s-%s CT lunch", "<calendar blackouts>"],`,
        FirstNoTradeMinutes(), ls, le)
}
```

→ `"no_trade": ["first 5m (CT)", "12:00-13:30 CT lunch", "<calendar blackouts>"],`

The model copied the example and dropped the placeholder. This is class 45 in
miniature, and it is **this wave's own defect**: the prose sentence added below
it says the windows are enforced regardless and "do not invent a window of your
own", while the EXAMPLE directly above shows those two windows as the expected
content. The example wins — an example is a demonstration, a sentence is a
request. **[A]**, n=1 (one plan written since the boot).

Per the ruling the field now renders nowhere by default, so the duplication is
inert on the surface. The prompt is unchanged: altering what the model is asked
to emit is a content ruling, and this wave measures the miss rather than
guessing at the fix.

**Recommended, needs a ruling:** make the schema example carry a placeholder
rather than the machine's own windows —
`"no_trade": ["<your own sit-out conditions, or omit>"]` — and re-measure over a
handful of reads. The risk of leaving it: the field is dead weight in every
prompt and every stored doc, and a future reader may take it for a rule again.

### 9.3 Two more copies of the lunch window, and the scan that missed them

The F4 literal scan looked only for the QUOTED bounds, so two copies written as
prose passed it:

| where | text |
|---|---|
| clock line | `…the lunch no-trade (12:00–13:30 CT) are CT wall-clock…` |
| no-trade gate | `lunch 11:30–13:30 ET … (the system hard-gates 12:00–13:30 CT)` |

Both render from `LunchWindowCT` now — **four copies retired by this wave** in
total, counting the gate and the grader.

The scan was wrong in both directions. Widening it to bare bounds over-fires:
`13:30 ET` is a different time from `13:30 CT`, and the ET lull in that same
sentence is deliberately not the machine's window. It scans for the window as a
pair now, in either dash, alongside the quoted single bounds. Quoted RED on both
new copies before the fix.

**Reported, not fixed:** the `11:30–13:30 ET` lull does not line up with the
`12:00–13:30 CT` hard gate (which is 13:00–14:30 ET). The prompt states both and
labels which one is enforced. Reconciling them is a content ruling. **[A]**

---

## 10. RIDER PART 2 (owner rulings 2026-09-03) — one clock, and an example that stopped teaching

### 10.1 The schema example

```
before   "no_trade": ["first 5m (CT)", "12:00-13:30 CT lunch", "<calendar blackouts>"],
after    "no_trade": ["<your own sit-out conditions, or omit>"],
```

The sentence below it is unchanged, as ruled. What changed is that the prompt no
longer *demonstrates* the two windows it then asks the model not to write.

**Residual seam, reported not fixed.** The surviving sentence still opens
"no_trade may contain ONLY the fixed session windows (first 5m, 12:00-13:30 CT
lunch) plus T1 HARD-blackout lines from the calendar", which names those windows
as permitted content while the example now offers a placeholder. The clause that
matters — "ENFORCED by the machine whether or not you list them … what you write
here is read as your notes" — is intact. Dropping the "may contain ONLY the
fixed session windows" clause would close the seam, and it is a content change,
so it waits for a ruling. **[A]**

### 10.2 One clock

The prompt's own clock line says *"EVERY time in this prompt is CT
(America/Chicago) … Never apply these numbers to a UTC clock"* — and three lines
then printed ET wall clocks. A model taking `10:30 ET` at its word as CT is an
hour out.

| line | before | after |
|---|---|---|
| skip deadline | `no pool swept by 10:30 ET` | `by 09:30 CT` |
| primary window | `NY AM 08:30–11:00 ET` | `07:30–10:00 CT` |
| premium FVG window | `10:00–11:00 ET` | `09:00–10:00 CT` |
| lunch | `11:30–13:30 ET … (hard-gates 12:00–13:30 CT)` | `lunch 12:00–13:30 CT: no new entries (hard-gated)` |

All derived through `ETtoCT`, never typed: America/New_York and America/Chicago
change offset on the same instants, so the gap is exactly one hour and needs no
date. The lunch line is now ONE window — it used to carry an advisory ET lull
beside the machine's enforced CT gate, two different windows in two clocks in a
single sentence. **Fifth copy retired**, after the gate, the grader, the clock
line and the no-trade-gate line.

### 10.3 What the model will read

```
clock 08:30 CT (13:30 UTC) — EVERY time in this prompt is CT (America/Chicago) …
## No-trade gates (advisory — declare in no_trade or skip the day)
  - no A/B zone in reach AND no pool swept by 09:30 CT → declare the skip in the plan
  - lunch 12:00–13:30 CT: no new entries (hard-gated — entries inside it are refused)
  NY AM 07:30–10:00 CT is the primary window; 09:00–10:00 CT is the premium FVG window
  "no_trade": ["<your own sit-out conditions, or omit>"],
```

### 10.4 Contract rows

| test | asserts | first run |
|---|---|---|
| `TestNoTradeExampleDoesNotDemonstrateMachineWindows` | the example names none of the machine's windows; the sentence still carries the resolved values | **RED ×4** |
| `TestPromptStatesNoUntypedEasternTimes` | the WHOLE rendered prompt holds no `HH:MM ET` | **RED**, named `10:30 ET` |
| `TestNoTradeContractRendersResolvedWindows` | migrated: the requirement moved from the example to the sentence, with its reason | GREEN |

Suites: Go clean · vitest 40 files / 306 tests · `tsc --noEmit` clean.

### 10.5 PROOF OWED

The re-measure cannot run before the boot. After it: quote the **next three
plans'** `no_trade` contents. Expected — the model's own reasons, or an empty
list. A third plan still echoing `first 5m (CT)` would mean the surviving
sentence in 10.1 is doing the teaching, and that clause is the next thing to go.

---

## 11. RIDER PART 3 (owner ruling 2026-09-03) — the seam closed, and two more copies of the same order

### 11.1 The instruction

```
before  no_trade may contain ONLY the fixed session windows (first 5m,
        12:00-13:30 CT lunch) plus T1 HARD-blackout lines from the calendar …

after   no_trade: the machine enforces the session windows and T1 blackouts
        regardless; do not list them — no_trade is for your OWN sit-out
        conditions, or omit it. A T2 caution event is NEVER added to no_trade
        and never stops entries.
```

No resolved values appear in it any more, because there is nothing left to
resolve. The window is still **stated** in the no-trade gate block, so the
author knows it exists and is simply not asked to repeat it. Contract row:
no machine window token anywhere in the instruction, and the phrases carrying
its meaning still present — **RED ×7** before the change.

### 11.2 Two more copies of the same order — judgement calls, flagged

Closing the sentence surfaced two other places that gave the model the
instruction the sentence now forbids. Neither was named in the ruling; I changed
both, and both are reversible.

| where | before | after | why |
|---|---|---|---|
| gate block header | `## No-trade gates (advisory — **declare in no_trade** or skip the day)` | `## No-trade gates (the machine enforces the windows below; the rest are yours to weigh — skip the day if they stack up)` | the list under it contains the machine's hard-gated lunch window. A header is read before a rule, and leaving it would very likely have defeated the re-measure the ruling asks for. |
| T1 calendar tag | `HARD no-trade blackout — **MUST be added to no_trade**` | `HARD no-trade blackout — the machine writes and enforces it; stand aside around it` | the dictated sentence says the machine enforces T1 blackouts regardless and the author must not list them. The old tag contradicted it outright, so the dictated wording forced this one. |

The blackout is still marked HARD and still reaches the model in both cases;
only the order to restate it is gone. **If either reads as scope creep, revert
the header — the T1 tag cannot stay as it was without contradicting the
sentence you dictated.**

### 11.3 Superseded specs migrated

| spec | why it no longer holds |
|---|---|
| `TestNoTradeContractRendersResolvedWindows` | required the OUTPUT CONTRACT to state the windows because "the model cannot list a window it was not shown". The ruling inverted the premise. Rewritten to assert the gate block states the resolved window. |
| `planner_playbook_test.go` `"10:30 ET"` | one-clock ruling — it is `09:30 CT` now |
| `planner_prompt_test.go` T1 tag wording | follows 11.2 |

None weakened; each carries its reason in the diff.

### 11.4 Five prompt copies retired in total

gate · grader · clock line · no-trade-gate line · the lunch line's ET half.

### 11.5 PROOF OWED — the re-measure

Cannot run before the boot. After it, quote the **next three plans'** `no_trade`
contents. Expected: the model's own conditions, or an empty list. A plan still
echoing `first 5m (CT)` after all five copies and both orders are gone would
mean the behaviour is not coming from the prompt at all, and the next place to
look is the repair path's own prompt.

### 11.6 The BEFORE measurement (baseline for the re-measure)

Two plans have now been written on the deployed pre-rider prompt, and both
carry the schema example verbatim:

| version | written | `no_trade` | machine windows |
|---|---|---|---|
| ASIA v14 | 00:08:45 CT | `["first 5m (CT)", "12:00-13:30 CT lunch"]` | 2 |
| ASIA v15 | 00:34:14 CT | `["first 5m (CT)", "12:00-13:30 CT lunch"]` | 2 |
| LONDON v1 | 01:34:53 CT | `["first 5m (CT)", "12:00-13:30 CT lunch"]` | 2 |
| NY v1 | 08:05:19 CT | `["first 5m (CT)", "12:00-13:30 CT lunch"]` | 2 |

**4 of 4, byte-identical, across three sessions and four independent reads.** That is the baseline the
three post-boot plans are measured against. It also rules out chance and
rules out session: the model is not choosing these two windows, it is copying
the example. Four reads on three sessions, including the full-reasoning NY
read, produced the same two strings and nothing else.

### 11.7 Two reads, two identical rejects — noted, not this wave

Both reads since the band boot were rejected on attempt 1 for the same defect:

```
00:07  S3 breakdown_continue: measured displacement 0.00 pts < 1.0×ATR5m (15.2 pts)
00:33  S2 breakdown_continue: measured displacement 0.00 pts < 1.0×ATR5m (21.5 pts)
```

Both repaired on attempt 2 and landed (`repair_outcome_ok` 10 → 11 across them).
2 of 2 reads authoring a waterfall play with zero measured displacement is a
prompt question — something is inviting `breakdown_continue` where the tape has
no displacement. Answered in §12: the floor was enforced and never stated.
**[A]**, n=2.

**Third read, no reject.** LONDON v1 at 01:34:53 landed on attempt 1 and
authored `S1 reject` + `S2 sweep_reclaim` — no waterfall at all. So the
displacement rejects are not every read; they are every read that reaches for
a waterfall. NY v1 (08:05:19) is the same again — attempt 1, `S1 reject` + `S2
breakout_retest` + `S3 sweep_reclaim`, no waterfall. n=4: two waterfall
attempts, two rejects, two reads that did not reach for one. That is the
population §12's proof will be measured over, and it is small.

---

## 12. RIDER PART 4 (owner ruling 2026-09-03) — the displacement floor feeds forward

Different lane, same boot.

### 12.1 The defect

`ValidateBreakdownContinueScenarios` refuses a `breakdown_continue` whose
measured displacement is below `BD_MIN_DISP_ATR × ATR5m`. The prompt named the
rule in the entry law and never the NUMBER, and never said which levels had any
displacement at all. So the author picked levels the tape had chopped across.

Two consecutive reads on 2026-09-03 burned attempt 1 on it:

```
00:07  S3 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (15.2 pts)
00:33  S2 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (21.5 pts)
```

2 of 2 reads since the band boot, both recovered only by a repair. This is the
class-45 shape exactly — a rule enforced at write and withheld from the prompt.

### 12.2 What the model reads now

```
## Waterfall displacement floor this cycle
15.2 pts (1.0×ATR5m 15.20, resolved). A breakdown_continue / breakup_continue
authored at a level whose MEASURED displacement is below this is REFUSED at write.

## Measured displacement per level (floor 15.2 pts)
  28900.00 ONL — 40.25 pts down · at or above the floor — authorable
  29050.00 VWAP — 3.00 pts down · BELOW the floor — not authorable as a waterfall
  29100.00 PDL — none — no break
```

The `breakdown{}` schema field is qualified the way `fvg{}` and `legs[]` are:
*author ONLY at a level whose MEASURED displacement in the block above is ≥ the
stated floor; a level reading "none — no break" or below the floor is REFUSED at
write.*

Boot line: `waterfall-displacement-floor=1.0×ATR5m=<pts>pts (stated per level)`.

### 12.3 No second implementation

Every number comes from `BreakdownContinueState` — the validator's own function
— reached through a level-oriented probe that synthesises the scenario a planner
would author at that level. It is the same trick `BreakdownLevelReclaimed` uses
for the void list, for the same reason. It runs over the SAME resolved
`VoidScope`, so the void list and the displacement list cannot disagree about
what the tape shows.

`TestDisplacementFeedsForwardMeasuresARealRun` asserts the fed-forward number
equals `BreakdownContinueState(...).BreakLegPts` exactly; a second
implementation appearing would fail it.

### 12.4 Two deliberate honesty choices

- **"none — no break"**, not `0.00 pts`. A level the tape never broke is not a
  level with a small number, and printing 0.00 for it invites precisely the
  reading this fixes.
- **No ATR → no floor line at all.** A `0.0 pt` floor is a lie the model would
  author against.

### 12.5 Tests

| test | asserts | first run |
|---|---|---|
| `TestDisplacementFeedsForwardChoppedLevelReadsNone` | THE PIN — a level chopped across 30 bars reads `Broken=false`, 0.00, and renders "none — no break"; the floor is stated with it | RED (undefined) |
| `TestDisplacementFloorLineIsResolved` | the floor line names its basis, and is EMPTY with no ATR | RED |
| `TestDisplacementFeedsForwardMeasuresARealRun` | a delivered run reports the validator's own number, on the delivering side | RED |
| `TestBreakdownSchemaRequiresMeasuredDisplacement` | the schema field is qualified; the facts block carries floor + per-level lines + both verdicts | RED |

Suites: Go clean · vitest 40 files / 306 tests · `tsc --noEmit` clean.

### 12.6 PROOF OWED

After the boot: whether a read still authors `breakdown_continue` at a level the
block marks "none — no break". Baseline is 2 of 2 reads doing exactly that.

---

## 13. RIDER PART 5 (owner ruling 2026-09-03) — wake cutoffs promoted to ENFORCE

### 13.1 The evidence that decided it

Today's WARN-era counters, read from `system_config`:

```
wake_cutoff_class47:…:2026-09-03:LONDON    1
wake_cooldown_class47:…:2026-09-03:NY      2
wake_stream_defer_class47:…:LONDON         3
```

**Three** wakes this morning would have been refused — one by the cutoff, two by
the cooldown — each of which spent a max-reasoning planner read:

| time | rule | detail | cost |
|---|---|---|---|
| 08:15 | cutoff | 24 min to flat (25m) — wrote LONDON v2 for an 08:30 flat | ~500s |
| 09:15 | cooldown | 21 min since the last wake-authored version | ~517s |
| 09:44 | cooldown | 23 min since the last wake-authored version | ~339s |

The three stream-defers show the cross-session rule already working.

### 13.2 The steady-state effect, measured — a HALVING, not a switch-off

The two clocks are offset by the planner call itself. Measured on NY today:

| wake fires | previous wake-authored version | version → wake |
|---|---|---|
| 09:06:54 | v2 08:45:05 | **21.8 min** → inside 30 → skip |
| 09:38:54 | v3 09:15:31 | **23.4 min** → inside 30 → skip |

Wake ATTEMPTS are throttled to ~30 min apart (08:06:54 · 08:36:54 · 09:06:54 ·
09:38:54), and a version lands **8–9 minutes after** the wake fires (08:36:54 →
v2 08:45:05; 09:06:54 → v3 09:15:31). So the version-to-next-attempt gap is
30 − 8.5 ≈ 21.5 min, always inside a 30-minute cooldown.

That does NOT switch wakes off. Skipping a wake writes no version, so the clock
keeps running and the FOLLOWING attempt is ~51 min past the last version and
proceeds:

```
version T          →  attempt T+21.5  SKIPPED  →  attempt T+51.5  PROCEEDS  →  version T+60
```

**Steady state: one wake-authored version per ~60 minutes instead of ~30.** The
cutoff then removes any wake in the last 25 minutes of a session on top of that.

This corrects my own earlier estimate to the owner ("three fewer plan versions
this morning"). Three is what today's counters caught; the ongoing effect is a
halving of the drumbeat, which is a different and better-bounded claim. **[A]**
— arithmetic from the timestamps above, not projection.

### 13.3 Scope, pinned both ways

`WakeCadenceGoverns` is the only place that decides what a wake is.
`level_event` and `structure_mss` are governed; `NY_scheduled_read`,
`LONDON_scheduled_read`, `death_replan`, `owner_reset`, `owner_reread` and
`sunday_weekly_read` are asserted NOT governed, so a later edit cannot widen it
without failing a test.

### 13.4 Two deliberate choices

- `HaveFlat` / `HaveLastWakeVersion` are separate fields from their numbers. An
  unreadable session window, or a session with no prior wake-authored version,
  must never manufacture a skip out of a zero — that is how a gate starts
  refusing things nobody decided to refuse (A24).
- Both boundaries belong to the wake: strictly `< cutoff` and `< cooldown`, so
  exactly 25 or exactly 30 still runs.

### 13.5 Unchanged

The counter keys and the recorded `n` — they now count real skips rather than
would-be skips, which the log line states, so today's WARN-era numbers and
tomorrow's enforce-era numbers form one series.
