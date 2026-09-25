# VETERAN REVIEW — PART C · THE PROMPTS + THE SYSTEM

Sub-agent C · 2026-09-05 · section 7 · read-only.

---

## 7.0 EVIDENCE BASIS — what I could and could not reach

Read the amendment before you read a number in this report.

**What I reached, and used.**

- The prompt source is fully present: `kernel/planner_prompt.go` (770 lines),
  `kernel/plan_doc.go` (1296), `kernel/engine_prompt.go` (1097),
  `kernel/engine_prompt_futures.go` (454), `kernel/engine_prompt_observer.go` (164).
- **Go builds and tests run in this environment.** Every size and modal count in
  §7.2 is real command output from prompts I RENDERED here, not an estimate off
  the source. I rendered them with `go test -overlay=` and a scratch test that
  lives only under my scratchpad — no file in the tree was created or changed
  except this report.
- `tiktoken` installed cleanly, so token counts are from a real BPE
  (`o200k_base`, cross-checked against `cl100k_base`). **Caveat stated once and
  meant: the planner runs on DeepSeek** (`docs/.../2026-09-01-full-system-audit.md:250`,
  `model=deepseek-v4-pro`), whose tokenizer is neither of these. Treat my token
  figures as a consistent proxy; where the PROVIDER's own count exists I quote it
  and say so.
- The executor prompt did not need rendering guesswork: `kernel/testdata/futures_mnq_plan.golden`
  is the verbatim executor system prompt with an active plan, and
  `kernel/golden_selfcheck.go:94-121` re-renders and byte-compares it **at boot**.
  I re-rendered it here and got the same 5,477 bytes.
- A rendered planner prompt is also committed:
  `docs/superpowers/reports/2026-09-04-research-conformance-data/A-rendered-planner-prompt.txt`.
- The reject data survives as a committed artifact:
  `.../2026-09-04-research-conformance-data/subsystemB_planner_rejects.csv` (55 rows)
  and `subsystemB_reject_class_tally.csv`.
- `docs/superpowers/AUDIT-CHECKLIST.md` (2021 lines, 79 numbered bug classes in PART 1,
  **80 `**Law:**` lines**) is complete here.

**What I could not reach.**

- **No running engine.** `/api/health`, `/api/expectancy`, `/api/config/resolved`
  are unreachable. For the settings surface (§7.10) I read the config-resolution
  code and the knob registry instead — `api/config_resolved.go`,
  `store/knob_registry.go`, `store/knob_registry_table.go`,
  `store/resolve_source.go` — and the committed
  `knob_registry_labels.csv` / `knoblive.txt`. **That is what I did; no number in
  §7.10 came from a live endpoint.**
- **No SQLite store.** The dispatch's `planner_rejected_prompts` query is
  unexecutable. §7.7 states the exact SQL I would have run, marks it BLOCKED, and
  answers from code plus the committed CSVs.
- `~/nofx-analysis/` does not exist. `docs/superpowers/plans/VL-MASTER-PLAN-v2.md`
  does not exist.
- **`docs/superpowers/research/` contains only `INDEX.md`.** Rounds 1–9 are not
  in this tree. So when the prompt cites "2,827-day NQ sample" or "40k-sample
  null", I can trace the citation to a *report that repeats it*
  (`2026-08-26-winrate-packA-implementation.md:22`,
  `2026-08-27-planner-contract-wave.md:10`) and no further. I say so in §7.4
  rather than pretending the primary source was checked.
- I ran **no git command**, per the amendment. I claim no branch and no merge.

No secret, key, token, `.env` value or account name appears below.

---

## 7.1 SUMMARY — the verdict, up front

I have read a lot of junior traders' checklists. This one is the work of a
careful person who has been adding a rule every time something went wrong for
three weeks and has never once taken one away. It is now **80% instruction and
20% facts by token**, and **half of it is a single unbroken paragraph**. That is
the shape of a checklist that has stopped being read.

Six things, in the order I would fix them:

1. **The output contract is one paragraph of 10,090 characters / 2,511 tokens /
   52 sentences with 157 ALL-CAPS emphasis tokens and no structure whatsoever.**
   It is 46.6% of the whole prompt. Nine `MUST`s and four `SHOULD`s live inside
   it. When class 50 found the correction block "sat at the TAIL of a ~6,600-token
   prompt, the position most likely to be skimmed" (`AUDIT-CHECKLIST.md:864`), it
   diagnosed the position and never asked about the shape.
2. **The prompt contradicts itself inside that same paragraph, twice, on numbers
   that decide whether a trade happens.** It states the minimum stop as
   `1.5×ATR5m` and, ~2,000 characters later, as `1.0×ATR5m`
   (`planner_prompt.go:733` vs `:752`). It lists `fvg_entry` as a condition an
   arm is legal on, and then says do not arm `fvg_entry`.
3. **The executor prompt does not know the machine is in `plan_mode=strict`.**
   Word "strict" appears **zero times** in the executor prompt, the plan block or
   the planner prompt. The executor is still told a valid off-plan setup may be
   traded and is handed `open_long`/`open_short` in its own worked example —
   while `trader/entry_gate.go:160-163` refuses every decision-path market entry
   outright.
4. **The prompt asserts market beliefs as fact that nobody has ever tested.**
   "Conviction: down on Monday, up Thursday/Friday" (`:656`) is [I] with, in the
   system's own words, "no citation anywhere in docs/ — pure prompt doctrine"
   (`2026-09-02-belief-census.md:120`). Bias-tree branch 5 forbids longs in
   premium; **17 of 58 plans (29.3%) stamped the forbidden bias and nothing
   rejected any of them.**
5. **Roughly 20% of the fixed instruction payload is spent on a condition that
   cannot place an order.** `fvg_entry` is `shadow` by default
   (`kernel/condition_status.go:27`); 32 FVG mentions, 10 of 51 Rules-paragraph
   sentences (3,207 chars / 850 tokens), plus a whole `## Priority setup` section.
6. **The single biggest historical reject class was created by a prompt sentence,
   and killing that sentence killed the rejects.** `B3 breakdown_void_reclaimed`
   was 21 of 55 rejects (38.2%). Fifteen of the twenty-one fell on 2026-09-02.
   After the class-45 VOID block landed: **zero on 09-03 and 09-04.** That is the
   only clean before/after in the whole reject record and it should be the
   template for everything else in §7.8.

The good news, and I will say it plainly because it is unusual: this prompt is
one of the few I have seen where the machine *feeds forward what it will refuse
with* — `RenderVoidBreakdownLevels` calls `BreakdownContinueState`, the actual
validator, so the prompt cannot hold a second opinion (`planner_prompt.go:459`,
`kernel/class45_feeds_forward.go:12-27`). That pattern works. It is applied to
one rule out of nineteen.

---

## 7.2 THE MEASUREMENT

### Method

I did not eyeball this. Commands and output:

```
# render the real prompts (scratch overlay; nothing written into the tree)
$ VET_OUT=$SCRATCH/out go test -overlay=$SCRATCH/overlay.json ./kernel/ -run TestVeteranDump -v
--- PASS: TestVeteranDump (0.00s)
ok  nofx/kernel 0.013s
```

The overlay test calls the production builders directly:
`BuildPlannerPrompt(...)`, `plannerOutputContract(0,0,...)`,
`selfCheckPlanEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)` — the same
function the boot self-check asserts against
(`kernel/golden_selfcheck.go:100-105`).

- `planner_min.txt` — the committed sample fixture (`planner_prompt_test.go:8-27`),
  a data-POOR prompt: 2 levels, no candles, no HTF zones.
- `planner_full.txt` — every optional block populated (candles, HTF zones,
  indicators, void levels, displacement, prior-plan carry-over, weekly). My
  candle rows are elided placeholders, so the FACTS side here is still thin
  relative to live; the INSTRUCTION side is exact.
- `executor_plan.txt` — byte-identical to `kernel/testdata/futures_mnq_plan.golden`.

### Sizes (real output)

```
file                              chars   words  lines   o200k  cl100k
planner_min.txt                   17078    2641     84    4520    4532
planner_full.txt                  19933    3104    131    5394    5422
planner_contract_default.txt      12637    1931     14    3301    3311
planner_contract_htf.txt          12927    1982     14    3365    3375
executor_plan.txt                  5477     854     94    1506    1504
executor_empty.txt                 4309     680     70    1096    1097
plan_block.txt                      610      92     10     207     205
```

**Live size, from the provider's own counter** (not mine):
`prompt` tokens `9,507–9,974` for a full-author planner call, `1,231–1,817` for a
repair — `2026-09-02-deepseek-e2e-audit.md:346`. A full re-author prompt is
**≈25.8k characters** (`2026-09-01-full-system-audit.md:272`).

So: my fixed instruction payload is **16,480 characters** (§ below). Against a
live 25.8k-character prompt in the same unit, **instruction is ≈64% of what the
model reads on a real read.** In my data-poor fixture it is 80.3% of tokens.
Either number tells the same story.

### The MUST / NEVER / SHOULD count

All-caps form only (`MUST` excludes `MUST NOT`; `SHOULD` excludes `SHOULD NOT`):

```
file                              MUST  NEVER  SHOULD  REQUIRED  ONLY
planner_min.txt                     11      7       5         4    14
planner_full.txt                    14      8       5         4    14
planner_contract_default.txt         9      5       4         4    10
executor_plan.txt                    3      0       0         1     1
```

Any casing, which is the honest count of how many imperatives the model is under:

```
file                              must  never  should  only  "do not"
planner_full.txt                    20     27       6    22         9
planner_contract_default.txt        15     17       5    13         6
executor_plan.txt                    8      2       1     5         6
```

**Twenty musts, twenty-seven nevers, twenty-two onlys and nine do-nots in one
prompt.** No human checklist survives that. Note also `MUST NOT` = **0** and
`SHOULD NOT` = **0** — the prompt has no vocabulary for a soft prohibition, so
every prohibition is written as `NEVER` and every one of them reads equally
loud. That is why the 27 `never`s carry no ranking: the reader cannot tell the
one that costs money from the one that is style.

### Facts vs instruction

I split the rendered prompt on `## ` headers and classified each section as
FACTS (Go-computed data the model reads) or INSTRUCTION (rules, playbooks,
schema, the header, the bias-tree branch list):

```
INSTRUCTION       4,332 o200k tokens   16,480 chars
FACTS + data      1,062 o200k tokens    3,425 chars
instruction share of tokens: 80.3%
instruction share of chars : 82.8%
```

The output contract alone is **3,301 tokens = 73.0% of the entire minimal
prompt.** The class-38 wave grew it 10,219 → 11,284 chars (+4%)
(`2026-09-02-deepseek-e2e-audit.md:199`); it now renders at **12,637** at shipped
caps and **12,927** when the HTF rules fire. It has grown 24% since that
measurement and nothing in the tree bounds it.

### The executor prompt

1,506 tokens with an active plan; 1,096 empty. **This is the right size.** It
reads like a checklist a person would actually follow. Everything wrong with it
(§7.6) is about what it says, not how much.

---

## 7.3 THE ONE-PARAGRAPH PROBLEM

`plannerOutputContract` (`kernel/planner_prompt.go:683-763`) ends by
concatenating every rule the system has ever learned into ONE string with no
newline. Measured:

```
Rules paragraph: chars 10090  words 1646  o200k tokens 2511
sentences (naive split): 52
inline rule tags: A1, A2, A2b, A2c, D2, F1, G5, GAR-F4, class 38
ALL-CAPS emphasis tokens: 157 (97 distinct)
as pct of full prompt chars : 50.6%
as pct of full prompt tokens: 46.6%
```

Half the prompt is one paragraph. It opens `Rules: levels chosen ONLY from the
ranked table above...` and does not break for ten thousand characters. Inside it,
in order, unsignposted: the level rules, the quality ladder, the scenario mix,
the gap-direction MUST, A1, A2, A2b, A2c, death/flip, confirm, ENTRY LAW,
target_chain, ARM SPLIT vs SINGLE, ARMED ORDERS, ARMS FOLLOW THE BIAS, WHICH
CONDITIONS CAN BE ARMED, ENTRY TYPE, CHAINED ARMS, FEASIBILITY CONTRACT,
WATERFALL ARMS, condition×session guidance, WATERFALL PLAY, NOISE FILTER, FVG
ENTRY, no_trade, candles, weekly.

Twenty-six distinct rule blocks. Every one of them is a heading in the author's
head and none of them is a heading on the page. Nine of the fourteen `MUST`s in
the whole prompt are in here.

The 157 ALL-CAPS tokens are the tell. When everything is shouted, nothing is.
`MUST` appears 9 times, `ONLY` 7, and then `THE`, `ONE`, `EXACT`, `BELOW`,
`FRESH`, `ENTRY`, `ARM`, `ARMS` — capitalisation used as prose emphasis, which
destroys its value as a signal for the four or five rules that actually decide
whether money is lost.

**What a trader does with a checklist this shape: skims for the shape of the
answer and writes it.** That is exactly the failure class 50 documented — the
correction block ignored at the tail — and moving the correction to the top
(`plannerRejectHeader`, `AUDIT-CHECKLIST.md:874`) treated the position without
touching the reason the position mattered.

The fix is not a wave. It is `strings.Builder` and newlines: ten headed sections
of ~250 tokens each, ordered by what they cost. Byte-identical content,
navigable. If nothing else in this report is done, do that.

---

## 7.4 THE PROMPT'S OWN BELIEFS — the research law, applied to the prompt text

This is the section I was told is most valuable to the owner, and I agree. A
prompt that instructs a model to believe something about the market is asserting
a market belief. Here is what this one asserts, labelled.

| # | The prompt says | file:line | Label | What backs it |
|---|---|---|---|---|
| 1 | "Conviction: down on Monday, up Thursday/Friday." | `planner_prompt.go:656` | **[I]** | Nothing. `2026-09-02-belief-census.md:27` labels it `[I] — no own-tape test`; `:120` says A2/A3 "carry no citation anywhere in docs/ — pure prompt doctrine". |
| 2 | "NY AM 08:30–11:00 ET is the primary window; 10:00–11:00 ET is the premium FVG window" | `:654-655` | **[I]** | Same census row A2, `[I]`. And it disagrees with the engine — see below. |
| 3 | Bias-tree branch 5: "longs ONLY below the 50% mark of the dealing range, shorts ONLY above it" | `:162`, veto rendered `:183-189` | **[I]** dressed as a machine branch | `adv-refute-A1b-branch5.csv` measured it: **own tape n=100 of 251 stored plans cite "branch 5"; 58 cite a side-specific veto; 17 of 58 (29.3%) stamped the FORBIDDEN bias — 12 of 43 premium plans stamped long, 5 of 15 discount plans stamped short — and nothing rejected any of them.** The holdout: `n=21, p=0.4762, Wilson [0.2834, 0.6763]` — a coin. |
| 4 | The whole bias tree (branches 1–4, 6) | `:157-163` | **[I]** | `belief-census.md:23` labels A1 `[I]`. |
| 5 | "ONH/ONL … broken intraday ~94% of days (2,827-day NQ sample)" | `:605` | **[R]-by-reference, unverifiable here** | Traced to `2026-08-26-winrate-packA-implementation.md:22` ("research 94.2% broken (2,827-day NQ)"). The primary study is not in this tree — `docs/superpowers/research/` holds only `INDEX.md`. It is repeated, not sourced. **[T]** support does exist alongside it: week ONH 14 entries · 21.4% win · −131. |
| 6 | "raw-FVG carries no standalone edge (40k-sample null)" | `:627` | **[R]-by-reference, unverifiable here** | Traced to `2026-08-27-planner-contract-wave.md:10` ("Cites the raw-FVG null (40k sample)"). No primary source in tree. |
| 7 | "NQ gap sweet spot 20–80pts; 1h+ fill ~70–80%" | `:756` | **[R]-by-reference** | `belief-census.md` B5 marks the family `[R] in-code citation` (`kernel/fvg_entry.go:25-46`). Not independently checkable here. |
| 8 | "reject-based setups are best in NY RTH (75% win, +665 this week); acceptance 0% win; sweep_reclaim 0% win" | `:737` | **[T] FROZEN AND NOW FALSE** | These are literals from the week ending 2026-08-26. `A-prompt-rule-inventory.csv` X5 re-measured on own tape: **reject 45.2% / +586, n=31 (NY 58.3% n=12, which is BELOW LONDON's 62.5% n=8); acceptance 16.7% n=6; sweep_reclaim 16.7% n=6.** The prompt is telling the model a 75% win rate that the tape now puts at 45.2%, and telling it NY is the best session when the same tape says London. |
| 9 | "acceptance entries WITHOUT a prior sweep + displacement are 0% win evidence" | `:660` | **[T] with a dead n** | `belief-census.md:28` labels it `[T]`. `A-prompt-rule-inventory.csv` A4: own tape acceptance **n=6, win 16.7%, sum +4.6** — not 0%, and n=6 is not evidence of anything. Class 16's own law: "every verdict carries n; no crowns on small n" (`AUDIT-CHECKLIST.md:123`). This line carries a rate with no n and no interval, in a prompt, in a system whose checklist forbids exactly that. |
| 10 | "opening gap >1.2×ATR or open outside the prior range → NEVER fade; the gap is a target" | `:638` | **[I]** | No citation in the tree. |
| 11 | "no A/B zone in reach AND no pool swept by 09:30 CT → declare the skip" | `:639` | **[I]** | No citation. |
| 12 | "Candles are ground truth for structure … On conflict, trust the candles" | `:760` | **[O]** owner ruling (W2b wave) | Fine as an owner ruling. Not a market claim. |

**Count: of the twelve market claims the planner prompt makes, six are [I] with
no citation anywhere in the repository, three are [R]-by-reference to studies
that are not in this tree, two are [T] with numbers the tape has since refuted,
and one is an owner ruling.** Not one of them is labelled inside the prompt. The
model reads all twelve in the same voice, at the same volume, as the Go-computed
level table.

Two of these have teeth I can prove:

**(a) The killzone the prompt names is not the killzone the grader scores
against.** `ETtoCT` subtracts an hour (`kernel/no_trade_band.go:243-249`), so
`:654` renders "NY AM **07:30–10:00 CT** is the primary window". The engine's own
NY session starts at **08:30 CT** (`kernel/session_registry.go:106`) and its
`ny_am` killzone is **08:30–11:00 CT** (`:111`). The registry's own test confirms
08:15 CT is still LONDON (`session_registry_test.go:63`). So the prompt's
"primary window" opens a full hour before the engine thinks NY exists.
`adv-refute-A2-killzone.csv` has the receipts: entries **588** (2026-09-02 07:41,
plan_session LONDON) and **558** (2026-08-25 07:56, LONDON) were graded **A → B,
step_down_cause `!InKillzone`**, note "*inside prompt primary window 07:30-10:00
CT; outside registry ny_am 08:30-11:00 CT*". Two entries downgraded for obeying
the prompt. That is class 52's law — "a rule with more than one definition has
none" (`AUDIT-CHECKLIST.md:941`) — live in the prompt.

**(b) Branch 5 is an instruction the machine ignores and the model half-obeys.**
The prompt writes it as an absolute (`longs ONLY below the 50% mark`), renders a
per-cycle veto line ("*PREMIUM — longs disallowed by branch 5*",
`planner_prompt.go:187`), and then, four rendered lines later, prints "*bias: AI
yours · tree neutral · regime up — labels only, no MUST on either*" (`:618-619`).
The prompt states an absolute and then withdraws it. `adv-refute-A1b-branch5.csv`
confirms branch 5 "appear[s] in NO validator/gate/entry-law file (grep whole
tree, non-test)". The measured result of an absolute nobody enforces and the
prompt itself undercuts is 29.3% non-compliance. That is what you get.

And the compliance rate for the one bias-tree instruction that IS phrased as a
`MUST` — "your reasoning MUST open by naming the bias-tree branch you took"
(`:715`): `adv-A1-biastree-compliance.csv`, **163 plans, 2026-08-26 → 2026-09-04,
120 named a branch = 73.6%** (ASIA 83%, NY 76%, **LONDON 60%**). A `MUST` with a
26% miss rate and no validator behind it (`A-prompt-rule-inventory.csv` A1a:
"NO validator anywhere (grep bias-tree = prompt only)", verdict `overclaim`).

**My own view, labelled [I] — my experience, untested here:** I have never seen a
day-of-week conviction tilt survive a live book on an index future, and I have
watched three people lose money holding one. If you keep line 1 in the prompt,
keep it as `[I]` in the prompt's own text so the model discounts it. If you are
not willing to write `[I]` next to it, you should not be willing to send it.

---

## 7.5 WHAT NO ONE COULD OBEY — contradictions inside the prompt

These are not judgement calls. They are two statements in one document that
cannot both be followed.

**1. Two minimum stops, 2,000 characters apart, in the same paragraph.**

`planner_prompt.go:733` renders from the resolver:

> "…the stop distance must be ≥ **1.5×** the current 5m ATR (the facts list the
> session ATR5m — cite the live value; a 10-point stop when ATR5m is ~16 is an
> instant refuse)"

`planner_prompt.go:752`, hardcoded as a string literal:

> "…runs the FULL gate chain (**min-SL ≥ 1.0×ATR5m**, R:R ≥ min_risk_reward_ratio,
> min-conf, HTF veto)"

`kernel/min_sl.go:34`: `const MinSLATRMultDefault = 1.5`. The `:733` figure is
correct because it calls `MinSLATRMult()`. The `:752` figure is a stale literal
from before 0B raised the floor. `grep` on the rendered prompt returns both:

```
$ grep -o "1.5× the current 5m ATR\|min-SL ≥ 1.0×ATR5m" out/planner_full.txt
1.5× the current 5m ATR
min-SL ≥ 1.0×ATR5m
```

A model authoring a waterfall play reads the nearer, more specific-sounding
number and sizes a stop that is refused. Class 43's law: "a knob that decides
money carries a citation or a suspension — never a number someone once typed"
(`AUDIT-CHECKLIST.md:648`). This is a number someone once typed, sitting six
lines from the resolver that would fix it.

**2. `fvg_entry` is both armable and forbidden to arm — same paragraph.**

The schema line renders from `ArmableConditionsPipe()` (`kernel/arm_kind.go:89`,
which delegates to `kernel/armed.go:17-27`):

> "arm{} is OPTIONAL and legal ONLY on **reclaim|reject|fvg_entry|
> breakdown_continue|breakup_continue**"

~4,000 characters later, from `ArmableConditionsLine(ResolvedConditionStatuses(...))`:

> "armable but SHADOWED — **do not arm these: fvg_entry**. NOT armable at all:
> acceptance, breakout_retest, hold."

Both are generated. Both are correct in their own terms. Together they are a
trap, and the second one is 4,000 characters downstream of the first in an
unbroken block. Class 78's law says "the prompt states the armable AND live
vocabulary, generated from the same table" (`:1808`) — it does, twice, from two
different tables, and never reconciles them into one sentence.

**3. The prompt demands a scenario that cannot reach the market.**

Chain the three statements:

- `:483` — "if NON-empty and a candidate's direction agrees with your bias, an
  fvg_entry is **EXPECTED** unless you state why not in one line"; `:716` A2c
  repeats it as a `SHOULD`.
- `:730` — "**ARMS FOLLOW THE BIAS**: with the decision path closed, a RESTING
  ORDER IS THE ONLY WAY INTO THE MARKET — a scenario with no arm **cannot
  trade**, however well argued."
- `kernel/condition_status.go:26-29` — `"fvg_entry": ConditionShadow` by default,
  and a shadowed condition "**MUST NOT place any order at the arm seam**" (`:11`).

So: write an fvg_entry (expected) → it cannot carry a live arm (shadowed) → a
scenario with no arm cannot trade (the prompt's own words). The model is told to
author a play that is dead on arrival, and told in the same paragraph that it is
dead on arrival. `INDEX.md` already flags this as a live contradiction:
"08-26 fvg-entry model … **no tradeable edge after honest costs** … shipped 0C
shadow demotion … →CONTRADICTION FVG-demand prompts (0C ruling)".

The cost, measured:

```
fvg mentions in the rendered prompt (any case): 32
Rules-paragraph sentences mentioning fvg_entry: 10 of 51
  their weight: 3,207 chars / 850 o200k tokens
plus: ## FRESH FVGs section, ## Priority setup — THE CHAIN (555 chars),
      the fvg{} schema object, A2/A2b/A2c, the FVG ENTRY 6-line playbook
```

Call it **≈4,100 characters and ~1,100 tokens — about 20% of the fixed
instruction payload — spent on a condition that has been ruled out and cannot
place an order.**

**4. The prompt cites a knob that has been deleted.**

`:733`: "R:R = |target−entry| ÷ |stop−entry| must be ≥ 2.0 (**ARM_MIN_RR**)".
`trader/armed_executor.go:60-61`: "**ARM_MIN_RR is DELETED, env and code default
alike.**" The test suite enforces the deletion:
`trader/settings_r123_test.go:35-37` sets `ARM_MIN_RR=9.9` and fails if a second
floor resurrects. The value 2.0 is right; the source name is a ghost.

**5. Six unresolved symbol names are shown to a model that cannot resolve them.**

```
$ grep -o "ACCEPT_HOLD_MIN\|BD_MIN_DISP_ATR\|BD_MIN_CLOSES\|ARM_MIN_RR\|min_risk_reward_ratio" out/planner_full.txt | sort | uniq -c
      1 ACCEPT_HOLD_MIN
      1 ARM_MIN_RR
      2 BD_MIN_CLOSES
      1 BD_MIN_DISP_ATR
      1 min_risk_reward_ratio
```

`BD_MIN_CLOSES` at least renders `(BD_MIN_CLOSES=1)` once. `ACCEPT_HOLD_MIN`,
`BD_MIN_DISP_ATR` and `min_risk_reward_ratio` are named with no value at all.
"price holds beyond ref for **ACCEPT_HOLD_MIN** minutes" is not an instruction, it
is a variable reference in a document with no variables. Two lines away, the
system renders the stop floor and the displacement floor with their live values
(`RenderStopFloorLine`, `RenderDisplacementFloorLine`, `:460-463`) — it already
knows how to do this correctly.

---

## 7.6 WHAT THE PROMPT DOES NOT SAY THAT THE MACHINE ENFORCES

**1. `plan_mode=strict` is invisible to both prompts.**

```
$ grep -c "strict\|plan_mode" out/executor_plan.txt out/executor_empty.txt out/plan_block.txt
0  (every file, both terms)
$ grep -o "strict[a-z]*" out/planner_full.txt
stricter        # a different sentence entirely
```

`plan_mode` resolves to **strict** on this build (`knob_registry_labels.csv`:
`plan_mode,KnobLive,trader/auto_trader_planconfig.go,[O],knob-census.md:95,strict,yes`).
`trader/entry_gate.go:160-163`:

```go
if in.PlanMode == "strict" {
    if in.Path != "arm" {
        return fmt.Sprintf("entry_gate: refused: strict — plan_mode=strict executes plan scenarios on the ARM path only, and this is a %s-path market entry", in.Path), true
    }
```

Now read what the executor is actually told, verbatim from the boot-verified
golden:

> `# DAY PLAN (NY) — preferred: follow it · a valid off-plan setup may still be
> traded (cite "off-plan")`
> …
> `- `cited_scenario`: REQUIRED on every open when a DAY PLAN is shown — the plan
> scenario id ("S1"…) you are trading, or **"off-plan" for a valid non-plan
> setup**.`
> …
> `[{"symbol": "MNQ", "action": "open_long", … "confidence": 80, "cited_scenario": "S1"}]`

Under strict, **every one of those is refused at leg 0** — the decision path is
not the arm path, so the `open_long` in the worked example is dead, and
`"off-plan"` is dead twice over (wrong path AND no cited scenario). The header
line is hardcoded with no mode awareness at `kernel/plan_render.go:156`.

This is the executor's version of class 50's exact defect: "the prompt withheld
what the validator enforces". Class 50 was raised against the PLANNER prompt and
fixed there. Nobody walked next door.

**What it costs:** the executor spends every 2-minute cycle reasoning toward an
action it cannot take, and its `wait` reasoning — which the adherence grader and
the F4 `no breakdown scenario authored` counter both read — is produced by a
model operating on a false model of its own authority.

**2. Two R:R floors from one unset config.**

`kernel/engine_prompt_futures.go:64-72`:

```go
minConf := rc.MinConfidence
if minConf <= 0 {
    // 6.1 — the SAME constant the gate's clamp uses (store.SafeDefaultMinConfidence):
    // prompt promise and gate threshold can no longer diverge for unset strategies.
    minConf = store.SafeDefaultMinConfidence
}
minRR := rc.MinRiskRewardRatio
if minRR <= 0 {
    minRR = 1.5
}
```

The min-confidence fallback does it right, and the comment states the principle
in the system's own words. **The R:R fallback four lines below does exactly what
that comment forbids.** With the field unset:

- the prompt tells the model **1.50** (`engine_prompt_futures.go:71`, rendered
  into the golden as "reward must be at least 1.50x the risk"),
- the gate enforces **3.0** (`kernel/engine_position.go:151-154`, `effRR = 3.0`),
- `/api/config/resolved` narrates **3.0** (`store/strategy.go:76`
  `SafeDefaultMinRiskReward = 3.0`, via `store.ResolveMinRiskReward` at
  `api/config_resolved.go:86`).

Three answers to one question. It does not bite today because the bound strategy
carries 2.0 (`knob_registry_labels.csv` line 3) — so I am labelling this
**latent, not live**. It fires on any strategy shipped with the field unset,
which is the default state. Class 48's law: "a gate with two ATRs is two gates,
and one of them is lying" (`AUDIT-CHECKLIST.md:787`). Same shape, three gates.

**3. The executor is still told to ignore crypto sections.**

> "This is a FUTURES contract, NOT a crypto perpetual: there is NO funding rate
> and NO crypto-style open interest. **Ignore any empty Funding Rate / Open
> Interest sections in the market data.**"

That sentence exists because the market data still contains those sections. See
§7.10 — 32 of 167 registry knobs are crypto/grid-shaped and marked live, and four
of them (`BTCETHMaxLeverage`, `AltcoinMaxLeverage`, and both position ratios) are
passed straight into `parseFullDecisionResponse` for every futures decision
(`kernel/engine_analysis.go:551-554`).

---

## 7.7 REJECTS PER RULE

**BLOCKED — NO STORE IN THIS ENVIRONMENT.** The query I would have run:

```sql
-- last 7 days, by rule, with n
sqlite3 -readonly data/data.db "
SELECT
  CASE
    WHEN reject_reason LIKE '%came back across%'      THEN 'B3 breakdown_void_reclaimed'
    WHEN reject_reason LIKE '%legs%'                  THEN 'ArmSpecValid legs'
    WHEN reject_reason LIKE '%not allowed for%'       THEN 'entry_law confirm-rule'
    WHEN reject_reason LIKE '%fade_requires_touch%'   THEN 'B4 fade_requires_touch'
    WHEN reject_reason LIKE '%displacement%'          THEN 'B1 BD_MIN_DISP_ATR'
    WHEN reject_reason LIKE '%stream interrupted%'
      OR reject_reason LIKE '%status 5%'              THEN 'TRANSPORT (not a validator)'
    ELSE 'other'
  END AS rule,
  COUNT(*) AS n,
  MIN(datetime(created_at,'-5 hours')) AS first_ct,
  MAX(datetime(created_at,'-5 hours')) AS last_ct
FROM planner_rejected_prompts
WHERE created_at >= datetime('now','-7 days')
GROUP BY rule ORDER BY n DESC;"
```

**What the code says can reject.** The reject path is
`store/planner_rejected.go:60-90` `SaveRejectedPrompt` /
`SaveRejectedPromptWithFacts`, called from `trader/auto_trader_planner.go:1293`,
capped at **200 rows** (`plannerRejectedCap`, `:56` — raised 20→200 on
2026-09-01 precisely because at 20 the class-38 forensics found n=1 of the defect
it was investigating). The nineteen enforced restrictions are enumerated with
their validator sites in `A-prompt-contract-19.csv`.

**What the committed data says.** `subsystemB_planner_rejects.csv`, **n=55**,
interval **2026-09-01 11:59:57 → 2026-09-04 08:41:25 CT** (a 3-day window inside
the 7 days to today; 55 « the 200-row cap, so nothing was trimmed). Sessions:
ASIA 33 · NY 16 · LONDON 6. Attempts: 1 → 27, 2 → 17, 3 → 11.

Per rule (`subsystemB_reject_class_tally.csv`, which I re-derived independently
and agree with on the headline):

| rule | n | % of 55 | first CT | last CT |
|---|---|---|---|---|
| **B3 breakdown_void_reclaimed** | **21** | **38.2%** | 09-01 12:52:48 | 09-02 21:46:34 |
| ArmSpecValid family | 7 | 12.7% | 09-01 11:59:57 | 09-04 08:28:32 |
| TRANSPORT (not a validator) | 6 | 10.9% | 09-01 12:33:07 | 09-02 01:15:57 |
| entry_law confirm-rule not-allowed | 5 | 9.1% | 09-01 12:06:32 | 09-04 08:07:20 |
| B4-family fade_requires_touch | 5 | 9.1% | 09-01 17:15:51 | 09-02 19:35:38 |
| B1 BD_MIN_DISP_ATR | 3 | 5.5% | 09-03 00:07:05 | 09-04 07:52:37 |
| confirm-rule enum invalid | 2 | 3.6% | 09-01 17:53:33 | 09-01 21:05:17 |
| P0.2-c continuation reachability | 2 | 3.6% | 09-04 00:37:17 | 09-04 08:41:25 |
| B2 BD_MIN_CLOSES | 2 | 3.6% | 09-04 08:09:03 | 09-04 08:25:34 |
| B2 BD_MAX_LEVEL_DIST_ATR | 1 | 1.8% | 09-02 12:57:32 | — |
| level cap (12) | 1 | 1.8% | 09-02 13:56:32 | — |

**The finding a trader cares about is the date column, not the count column.**

```
$ python3 ... Counter(B3 rejects by date)
Counter({'2026-09-02': 15, '2026-09-01': 6})
```

Twenty-one B3 rejects, all of them on 09-01 and 09-02, **zero on 09-03 and
09-04** — over which the store recorded 10 further rejects of other classes, so
the instrument was still writing. The class-45 wave, which added
`RenderVoidBreakdownLevels` calling the validator's own
`BreakdownContinueState` (`planner_prompt.go:459`,
`kernel/class45_feeds_forward.go`) and changed `:714`'s `MUST` from naming a
CONDITION to naming a DIRECTION, **removed the single largest reject class
entirely.**

That is the whole thesis of this section in one before/after. 38.2% of all
planner rejects over three days were caused by one sentence ordering a play the
validator would void, and the cure was to render the validator's own verdict into
the prompt. **Nineteen restrictions are enforced; exactly one of them feeds its
verdict forward this way.** The other eighteen are still statements of law
written twice, in two places, by two waves.

Two more notes on the reject record:

- **6 of 55 (10.9%) are not validator rejects at all** — three `stream
  interrupted: context deadline exceeded` and three `API error (status 503):
  Server Overloaded`. Class 41's law says "a provider failure is retried, a
  validator reject is repaired — never append a transport error to a prompt"
  (`AUDIT-CHECKLIST.md:587`). These rows are transport text sitting in a table
  whose purpose is offline A/B of the validator. The 09-01 pair (ids 71, 72) were
  fed back to the model as "the validator reason"
  (`2026-09-01-full-system-audit.md:272`).
- **The re-author correction is 1.5% of the prompt it corrects.**
  `2026-09-02-deepseek-e2e-audit.md:152`: "the reject block is ~92–127 tokens
  against a ~6,341–6,691-token prompt, i.e. ~1.5%, and the standing MUST at :589
  is still inside it. The model is told 'you MUST write a continuation short' in
  the body and 'do not write the continuation you just wrote' in a footnote." In
  a 52-sentence single paragraph, a 1.5% footnote is not a correction. It is a
  rounding error.

---

## 7.8 WHAT I WOULD DELETE

A prompt is a position. You size it. Here is my sizing, with the token cost of
each cut measured, not guessed.

| # | Cut | Saves | Why |
|---|---|---|---|
| 1 | **Break the Rules paragraph into ~10 headed sections.** Zero content change. | 0 tokens | §7.3. The single highest-return edit in the file. It is a `strings.Builder` change. |
| 2 | **Delete every `fvg_entry` demand line** (`:483` "EXPECTED", `:716` A2c, `:756` the 6-line playbook, `:623-627` THE CHAIN as an FVG chain). Keep the schema field and the FRESH FVGs list so a future un-shadowing is one line. | **≈850–1,100 tokens** | §7.5.3. The condition is shadowed, has no edge after costs (`INDEX.md` 08-26 row), and cannot place an order. You are paying 20% of your instruction budget to be told about it. |
| 3 | **Delete `:656`** ("Conviction: down on Monday, up Thursday/Friday"). | ~15 tokens | §7.4 #1. [I], zero citation, zero counter, and it will bias a plan on a Monday. Cheap to cut, and every uncited [I] you leave in teaches the model the rest are also opinions. |
| 4 | **Replace `:737`'s frozen week literals** with the current tape numbers **and their n and interval**, rendered — or delete the line. | ~60 tokens | §7.4 #8. "75% win, +665 this week" against a measured 45.2% / n=31 is not stale, it is wrong, and the prompt says "this week" about a week that ended ten days ago. |
| 5 | **Fix `:752`'s `1.0×ATR5m` to render `MinSLATRMult()`** like `:733` does. | 0 tokens | §7.5.1. Six lines from the resolver. |
| 6 | **Reconcile the two armable lists into one generated sentence.** | ~40 tokens | §7.5.2. |
| 7 | **Render `ACCEPT_HOLD_MIN`, `BD_MIN_DISP_ATR`, `min_risk_reward_ratio` as values**; drop the dead `ARM_MIN_RR` name. | ~0 | §7.5.4-5. The renderers already exist. |
| 8 | **Reconcile `:654-655`'s killzone with `session_registry.go:111`** — one definition, rendered from the registry. | ~10 tokens | §7.4(a). Two live entries already graded down for obeying the prompt. |
| 9 | **State `plan_mode` in BOTH prompts.** Under strict, delete "a valid off-plan setup may still be traded" from `plan_render.go:156` and the `"off-plan"` option from the executor's `cited_scenario` line; replace the `open_long` example with the truth. | +~40 tokens (a net ADD) | §7.6.1. The one place I would spend tokens rather than save them. |
| 10 | **Give the FEASIBILITY CONTRACT, the ENTRY LAW and the ARM SPLIT contract their own headed section at the TOP**, above the facts. | 0 tokens | These three are what the 55 rejects are actually made of (ArmSpec 7 + entry_law 5 + B4 5 + B1 3 + B2 3 = 23 of 55 = 42%). They currently sit in the middle of a ten-thousand-character run-on. |

Net: **≈1,000 tokens out, ~50 in, and the remainder restructured.** That is a 20%
cut to the instruction payload with no rule lost that the machine enforces.

**What I would NOT cut**, because it is the best thing in the file: every block
that renders a machine verdict — `RenderVoidBreakdownLevels`,
`RenderStopFloorLine`, `RenderDisplacementFloorLine`, `RenderDisplacementLines`,
the FRESH FVGs candidate list, the ranked level table, the consumed-levels list,
the prior-plan carry-over. Those are facts computed by the same code that will
judge the answer. That is how you write a checklist. Do more of it and less
prose.

---

## 7.9 THE LAWS — which protect P&L, which are hygiene, which are missing

`docs/superpowers/AUDIT-CHECKLIST.md` carries **79 numbered bug classes in PART 1 and 80
`**Law:**` lines** (`grep -c '\*\*Law:\*\*'` → 80; 79 distinct class numbers,
some renumbered at merge, which is why two classes share a law line and one law
sits under the class-75 SYSTEM-MAP contract at `:2002`). I read all 80 and sorted them.
My method, stated so you can disagree with the boundaries: a law is **money-path**
if breaching it can change a fill, a size, a direction, or leave an order alive at
the broker; **evidence** if breaching it corrupts a number a decision is later made
on; **craft** if it governs the repo, the build, the logs or the process.

| Bucket | n | The ones that matter most |
|---|---|---|
| **Money-path** | **≈37** | 15 fantasy-R · 20 broken clock · 24 no panic in the trading loop · 25 MANUAL-CANCEL-WINS · 27 the ledger's model of the broker is a claim · 33 no process leaves orders alive for a successor · 34 a hint is an instruction · 36 never authoring during a halt · **38 the prompt states every restriction its validator enforces** · 41 provider failure retried / validator reject repaired · 43 a knob that decides money carries a citation or a suspension · **44 every retry path shows the model the same vocabulary the validator judges by** · 48a a protection on one order path is a suggestion · 48b a gate with two ATRs is two gates and one is lying · **50 the prompt must state every rule the validator will enforce, in the validator's own words** · 51 a directional label is evidence or noise · **52 a rule with more than one definition has none** · 67 a gate leg must be answered by the system it asserts about · 74 swap-verify-kill · **78 the prompt states the armable AND live vocabulary, generated** · **79 silence is not evidence** |
| **Evidence** | **≈27** | 16 every verdict carries n, no crowns on small n · 22 counters log what they name · 29 an aggregate reports what it excluded · 35/77 counters record, they do not infer, and reporting never resets · 40 the model never reads a fabricated track record · 49 an instrument that cannot disagree with the code is not evidence · 56 a column that cannot say "unknown" cannot input a ruling · 62 populated ≠ computed · 63 a boot line is a claim · 64 calibrate every rate-producing instrument against noise of its own scale · 65 synthetic rows excluded at the write · 69 a claim of being wired is a testable assertion |
| **Craft / hygiene** | **≈14** | 4 LoadOrStore · 13 WORKTREE LAW · 17 baks · 18 GUIDE CONTENT · 21 binaries never tracked · 23 supply chain · 28 one canonicalizer · 70/71/72/73/75/76 lock, stash, flaky gate, hook install, build dir, target-not-on-dev |

**This is a good body of law.** Five of the six laws I would have written for a
prompt-driven trading system are already in it (38, 44, 50, 52, 78). The
checklist is not naive. What it is missing is the class of defect this section
found — and every one of the gaps below is a defect I could point at in §7.3–7.6.

### The laws that are missing

**M1 — Nothing bounds the prompt's SIZE or SHAPE.** Class 50 diagnosed the
correction block's *position* ("the TAIL of a ~6,600-token prompt, the position
most likely to be skimmed", `:864`) and fixed the position. No law says a rule
must be findable. Result: a 10,090-character, 52-sentence, 157-ALL-CAPS
paragraph carrying 9 of the prompt's 14 `MUST`s, unbroken. **Proposed law:**
*every rule the model must obey is stated ONCE, under a heading, in a section
with a stated token budget; a prompt section that grows past its budget loses a
rule before it gains one.*

**M2 — Nothing requires a prompt assertion to carry its evidence label.**
`2026-09-02-belief-census.md` labels every belief `[R]/[T]/[I]/[O]` — in a
report. The labels never reach the prompt. The model reads "Conviction: down on
Monday" (`[I]`, uncited), "reject 75% win" (`[T]`, refuted), and the Go-computed
level table in the same voice. **Proposed law:** *a market claim in a prompt
renders its label and, for [T], its n and interval — or it is not sent. A [T]
literal frozen from a past week is a [T] with an expiry, and it renders the date
of the week it came from.*

**M3 — Nothing forces a rule to be REMOVED when its subject dies.** The 0C
ruling shadowed `fvg_entry` on 2026-08-31. On 2026-09-05 the prompt still spends
~20% of its instruction budget demanding it. `ARM_MIN_RR` was deleted; the prompt
still cites it. Class 58/61's law covers user-selectable options ("an option a
user can select is a promise") — nothing covers a rule the prompt still teaches.
**Proposed law:** *a wave that shadows a condition, deletes a knob or moves a
floor deletes its prompt text in the SAME commit; a contract test greps the
rendered prompt for the retired token.* (Class 75's SYSTEM-MAP contract at
`:2002` already does exactly this for boot lines. Point it at the prompt.)

**M4 — Class 50's law is scoped to the PLANNER prompt and was never applied to
the EXECUTOR prompt.** "The prompt must state every rule the validator will
enforce" (`:877`) is written about the plan validator. `plan_mode=strict` is
enforced at `entry_gate.go:160` and appears in no prompt at all. **Proposed law:**
*every prompt states the mode it is operating under, rendered from the resolver
that enforces it; a prompt that offers an action the gate refuses by construction
is a defect of the same class as a validator rule the prompt withheld.*

**M5 — Nothing requires a stated threshold to be RENDERED from its resolver.**
Class 8's law ("a knob without a shared resolver is decoration", `:67`) is about
the Studio card. The prompt hardcodes `1.0×ATR5m` next to a resolver-rendered
`1.5×`, and hardcodes `1.5` for R:R next to a `SafeDefaultMinConfidence` call
that gets it right. **Proposed law:** *a number a prompt states about a gate is
rendered by calling that gate's resolver; a numeric literal in prompt text about
an enforced threshold fails the build.* A `grep` for `[0-9]+\.[0-9]×ATR` in the
prompt builders would have caught both.

**M6 — Nothing measures whether the model OBEYS.** `adv-A1-biastree-compliance.csv`
(73.6% on a `MUST`, n=163) and `adv-refute-A1b-branch5.csv` (29.3% violating an
`ONLY`, n=58) exist because an adversarial reviewer went looking. Nothing routine
produces them. **Proposed law:** *every `MUST` in a prompt has either a validator
that rejects its breach or a counter that records its compliance rate with n; a
`MUST` with neither is downgraded to prose at the next wave.* By this test, of
the 14 all-caps `MUST`s in the planner prompt, `A-prompt-rule-inventory.csv`
already marks four `overclaim` — A1a, A1b, A7 and X3 — including X3 (`ARMS FOLLOW
THE BIAS … a long plan with no long arm is invalid`) whose enforcer,
`kernel/arms_bias_coherent.go:74 BiasArmWarning`, has **0 production callers**.
The prompt says "invalid"; the machine does nothing.

---

## 7.10 THE SETTINGS SURFACE — which knobs a trader would actually turn

**`GET /api/config/resolved` is unreachable here.** I read the handler
(`api/config_resolved.go`), the registry (`store/knob_registry.go`,
`store/knob_registry_table.go`), the resolvers (`store/resolve_source.go`) and
the committed `knob_registry_labels.csv` / `knoblive.txt` instead. Every number
below is from those files.

**The registry, counted from source:**

```
$ python3 (parse store/knob_registry_table.go)
entries: 167
  KnobLive        144
  KnobCandidate    16   (no consumer found by a FIELD grep — NOT dead, unverified)
  KnobIneffective   7   (read, does not take effect)
```

The two-label design (`api/config_resolved.go:8-16`) is genuinely good work:
"no consumer" and "reads it but cannot take effect" are different findings and
stay legible as different findings. Keep it.

Now the trader's question — **which of these would I actually turn?** Three
findings.

**(1) The endpoint serves classification, not values.** By design:
`api/config_resolved.go:18-19` — "*A25: KnobEntry holds no values, and this
payload adds none.*" The only actual saved→resolved values it returns are **three
fields**, from `buildResolvedFields` (`:79-126`): `min_risk_reward_ratio`,
`day_plan.plan_mode`, `regime.htf_veto`. So a trader asking "what is my stop
floor right now?" gets a list of 167 knob *statuses* and three numbers, none of
which is the stop floor.

**(2) The knobs that actually shape trading are env vars, and not one of them is
in the registry.**

```
$ grep -rn "os.Getenv(" kernel/ trader/ --include=*.go | grep -v _test | grep -oP 'os\.Getenv\("\K[A-Z0-9_]+' | sort -u | wc -l
81
$ (for each: grep the lowercased name in store/knob_registry_table.go)
count=0
```

**Eighty-one environment knobs read by `kernel/` and `trader/`; zero of them
appear in the settings registry.** They include every number a discretionary
trader would reach for first: `MIN_SL_ATR_MULT`, `BD_MIN_DISP_ATR`,
`BD_MIN_CLOSES`, `BD_MAX_PULLBACK`, `BD_MAX_LEVEL_DIST_ATR`, `ACCEPT_HOLD_MIN`,
`ARM_PLACE_TICKS`, `ARM_WORKING_STALE_MIN`, `ARM_FAR_ATR_MULT`,
`ARM_STOP_ANCHOR_MAX_ATR`, `TOUCH_BAND_TICKS`, `STALE_CONFIRM_ATR`,
`NOCHASE_MAX_DIST_ATR`, `FLIP_CONFIRM_CLOSES`, `HTF_VETO_MODE`, `HTF_VETO_TF`,
`STRUCTURE_MIN_SWING_ATR`, `MSS_MIN_DISP_ATR`, `FVG_CE_WIDTH_PTS`,
`SHADOW_CONDITIONS`, `LIVE_CONDITIONS`, `WEEKLY_READ_CT`, `VOID_SCOPE_BARS`.

The settings page shows 167 knobs, of which ~32 govern an asset class this desk
does not trade (below), and **hides the ~25 that govern how a trade actually gets
entered.**

**(3) The endpoint claims to report env shadows, and the counter is never
written.** `store/knob_registry.go:125-126` declares
`KnobSummary.EnvShadows` and `EnvShadowPaths`. They are read at
`api/config_resolved.go:175-176` and printed on the boot line at
`store/knob_registry.go:181` as `env-shadows=%d`.

```
$ grep -rn "EnvShadows\s*=\|EnvShadows:\|EnvShadowPaths\s*=\|EnvShadowPaths:" --include=*.go .
./api/config_resolved.go:175:  EnvShadows:     sum.EnvShadows,
./api/config_resolved.go:176:  EnvShadowPaths: paths,
```

Two read sites. **Zero write sites.** `KnobStatusSummary()` (`:141-163`) counts
seven statuses and never touches either field. So the boot line prints
`env-shadows=0` and the API returns `env_shadows: 0, env_shadow_paths: []`
**every time, unconditionally**, against 81 real env knobs. That is class 22's
law ("counters log what they name", `:185`) and class 69's law ("a claim of being
wired is a testable assertion", `:1501`) broken *inside the endpoint the audit
dispatch sends a trader to for the truth*. It is a one-line fix and it should be
this week's.

**(4) Nineteen percent of the live registry is for an asset class this desk does
not trade.**

```
$ (regex: altcoin|btc_eth|funding|hyper|netflow|oi_|ai500|coin|grid|leverage|notional|maker_only|quant)
crypto/grid-shaped: 32 of 167
  ai500_limit · altcoin_max_leverage · altcoin_max_position_value_ratio ·
  btc_eth_max_leverage · btc_eth_max_position_value_ratio · enable_funding_rate ·
  enable_netflow_ranking · enable_oi_ranking · enable_quant_data ·
  enable_quant_netflow · enable_quant_oi · excluded_coins · grid_config ·
  grid_count · hyper_main_limit · leverage · max_notional_leverage ·
  netflow_ranking_duration · netflow_ranking_limit · notional_cap_enabled ·
  oi_low_limit · oi_ranking_duration · oi_ranking_limit · oi_top_limit ·
  quant_data · static_coins · use_ai500 · use_hyper_all · use_hyper_main ·
  use_maker_only · use_oi_low · use_oi_top
```

Thirty-one of the thirty-two are `KnobLive` — a consumer really does read them.
And four of them touch every futures decision:
`kernel/engine_analysis.go:551-554` passes `BTCETHMaxLeverage`,
`AltcoinMaxLeverage`, `BTCETHMaxPositionValueRatio` and
`AltcoinMaxPositionValueRatio` into `parseFullDecisionResponse` for MNQ.
`knob_registry_labels.csv` says as much in its own note: "*crypto-shaped; still
clamps futures decisions via parseFullDecisionResponse*". This is also why the
executor prompt has to spend a line telling the model to ignore empty Funding
Rate sections (§7.6.3).

### The knobs I would actually turn, in order

Labelled. Nothing here is a recommendation to change a value — it is a
recommendation about which dial should be on the front panel.

| # | Knob | Where it lives now | Why a trader reaches for it |
|---|---|---|---|
| 1 | `MIN_SL_ATR_MULT` (resolved **1.5**, `kernel/min_sl.go:34`) | **env only** | It is the single number standing between an authored setup and a refused arm, and the demotion queue already ranks it #1 as an `[I]` belief with REJECT teeth (`belief-census.md` demotion queue rank 1). **[I]** — my own view: 1.5×ATR5m on MNQ is a *wide* floor for a level-based fade, and it is the reason arms get refused after the composer widens the stop and the R:R gate then judges the wider stop. That interaction (class 50(b), `:853-857`) is the most expensive mechanism in this system and it is controlled by an invisible env var. |
| 2 | `plan_mode` (resolved **strict**) | registry, `[O]` | It decides whether the AI executor can trade at all. It is one of the three values `/api/config/resolved` actually returns — correctly. Neither prompt mentions it (§7.6.1). |
| 3 | `min_risk_reward_ratio` (live DB **2.0**; unset default **3.0**) | registry, `[O]` | Governs both the arm seam and the decision path since R1. Three different fallbacks in three files (§7.6.2). |
| 4 | `condition_status` / `SHADOW_CONDITIONS` | registry + env | Decides which of the nine conditions can place an order. Today it silently kills `fvg_entry` and `breakout_retest` while the prompt still teaches both. |
| 5 | `sessions_enabled` | registry — and **it does not work** | `knob_registry_labels.csv`: live value `["NY"]`, conforms **NO**, "*per-session `enable:true` OVERRIDES it — ASIA+LONDON still run*". The reject record backs it: **33 of 55 rejects (60%) came from ASIA**, a session the owner's top-level knob says is off. A trader who turns this knob off and walks away is still trading Asia. |
| 6 | `min_grade` (B), `max_levels` (12), `scenario_cap` (5), `proximity_filter_atr` (**1.0**, census says 0.3 — conforms NO) | registry | These shape the map the planner sees. `proximity_filter_atr` disagreeing with its own census by 3.3× is exactly class 8's "a knob without a shared resolver is decoration". |
| 7 | `breakeven_enabled` / `trailing_enabled` | registry — **both read `true` and are refused at the wire** | `knob_registry_labels.csv`: "*DB true but 0B refuses at the wire; boot BE=off*". The settings page says your breakeven is on. It is not. Class 58's law ("an option a user can select is a promise") applies verbatim. |
| 8 | `guardrails_enabled` = **false** | registry | With the master off, `daily_loss_limit_usd` (450), `daily_profit_target_usd` (900), `max_daily_trades` (3) and `max_contracts_per_order` (2) are all inert. Four numbers on the settings page that do nothing. `2026-09-03-mc-drawdown.md` (per INDEX) already measured that the 3-trade cap forfeits $24.54/day and the $450 limit is inert — the report exists; the knob still reads as live. |

**The one-line summary of the settings surface:** the page shows 167 knobs of
which ~32 are for the wrong asset class, ~7 are promises the wire refuses, and it
hides the 81 env knobs that decide how a trade is entered — while reporting
`env-shadows=0` from a counter that has no writer.

---

## 7.11 SURPRISES (included, not acted on)

1. **The boot-time prompt golden self-check is genuinely excellent and I have not
   seen it elsewhere.** `kernel/golden_selfcheck.go:13-25` embeds the goldens in
   the binary and re-renders all three at startup, so it checks the code against
   *the contract it was built with*, not whatever is on disk. That is the
   Knight-Capital control and it is correctly reasoned. It covers the executor
   prompt only — the planner prompt has no equivalent.
2. **The `facts` column shipped and its data is empty.**
   `2026-09-02-deepseek-e2e-audit.md:336`: `Facts` is declared
   (`store/planner_rejected.go:28`) and `SaveRejectedPromptWithFacts` is wired,
   but `AutoMigrate` is lazy and at audit time all 26 stored rows were
   facts-less, so the offline A/B still could not re-run a single fact-dependent
   validator. Not mine to act on; worth knowing before anyone plans an A/B on
   this table.
3. **`plannerRejectedCap` was raised 20 → 200 for exactly the reason this review
   needed it** (`store/planner_rejected.go:50-56`): at 20, the class-38 forensics
   found n=1 of the very defect they were investigating. Somebody learned the
   right lesson from that and wrote it in the comment.
4. **`RenderBiasTree` carries a `%` bug fix in a comment that reveals a bigger
   one.** `:165-170`: "*the 17:46 read printed '376% of range' off a ~30pt VA*".
   The fix was to switch the anchor to PDH/PDL with the value area as fallback.
   But branch 5 is still stated as an absolute over an anchor that can silently
   change from the dealing range to a 30-point value area, in the same prompt,
   with no note to the model about which one it got.
5. **The repair prompt is a different vocabulary from the re-author prompt.**
   `2026-09-01-full-system-audit.md:272`: attempt-2 repair prompts (4.3–4.7k
   chars) **do not** carry `LiveConditionsLine`; full re-authors (≈25.8k chars)
   do. Class 44's law is "every retry path shows the model the same vocabulary
   the validator will judge it by" (`:682`). The repair path still does not.

---

## 7.12 WHAT I COULD NOT EXECUTE, AND WHY

| Box instruction | Status | Why |
|---|---|---|
| `GET /api/expectancy` | **not executed** | No engine running; connection refused. Not needed for section 7. |
| `GET /api/config/resolved` (token `cmd/gate-jwt`) | **not executed** | No engine running. Substituted with `api/config_resolved.go`, `store/knob_registry*.go`, `store/resolve_source.go` and the committed `knob_registry_labels.csv` / `knoblive.txt`. Stated in §7.10. No token was read or printed. |
| `planner_rejected_prompts` rejects per rule, last 7 days | **BLOCKED — no store** | Exact SQL given in §7.7. Answered from `store/planner_rejected.go` (the writer, the cap, the 19 restrictions) plus `subsystemB_planner_rejects.csv` (n=55, 09-01 → 09-04 CT) and `subsystemB_reject_class_tally.csv`. |
| Query `trader_positions`, `armed_orders`, `plans`, `touch_outcomes`, `decision_records`, `nt8_order_snapshots`, `bars` | **BLOCKED — no store** | Not required by section 7; noted so the lead can see the boundary. |
| Replay the tape · read the NT8 logs · run `~/nofx-analysis/` scripts | **not executed** | No tape, no NT8 logs, no `~/nofx-analysis/` in this environment. |
| `docs/superpowers/plans/VL-MASTER-PLAN-v2.md` | **absent from the tree** | Confirmed missing. |
| `docs/superpowers/research/` rounds 1–9 | **absent from the tree** | Only `INDEX.md` is present. This is why §7.4 rows 5–7 are labelled *[R]-by-reference, unverifiable here* rather than [R]. |
| Real DeepSeek token counts | **substituted** | No DeepSeek tokenizer available. Used `o200k_base` + `cl100k_base` (both reported), and quoted the PROVIDER's own `prompt=` counts from the journal where they exist (`2026-09-02-deepseek-e2e-audit.md:346`). |
| Claim a branch / run `deploy/nofx-claim.sh` / merge | **not executed** | Amendment rule 5: the lead owns all git in this session. I ran no git command. |

Every other measurement in this report is command output I produced in this
environment, or a `file:line` / `report:line` I read here.

---

*Sub-agent C · read-only · no file in the tree changed except this report.*
