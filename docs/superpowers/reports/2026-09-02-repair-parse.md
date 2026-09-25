# Repair-parse — the default retry was judged against a vocabulary it was never shown

**Dispatch:** REPAIR-PARSE, lane 1. Owner hoang, 2026-09-02. Base: dev `4175e0b6` (after lane 2).
**Evidence tiers:** [A] verified directly · [B] inferred · [C] speculation.

## 0. HEADLINE — the rate is real, the diagnosis was not (A23)

C1 said "repair output unparseable 59%". The rate is real. The cause is not packaging.

| Measurement | Value |
|---|---|
| Repair attempts (2026-09-01 → 09-02, journals) | **n=28** |
| Rejected at the parse/schema step | **18 (64%)** |
| Accepted (landed a plan) | 8 |
| Rejected later, by a fact-dependent validator | 2 |
| Of the 18: **packaging** failures | **1** |
| Of the 18: parsed cleanly, **rejected on field values** | **17** |

The one packaging failure (09-01 04:24:17) was a TYPE error, not fences or prose:
`plan JSON unmarshal: json: cannot unmarshal number 0.5 into Go struct field
PlanArmLeg.scenarios.arm.legs.size of type int` — the model wrote a fractional contract size. [A]

**Zero markdown fences. Zero prose-wrapped documents. Zero fragments. Zero truncation.** The
reason is in the code: `extractJSONObject` (kernel/plan_doc.go:933) already scans to the first
`{` and walks to its matching `}`, tracking strings and escapes — fences and surrounding prose
were ALREADY tolerated. **E2 as scoped (a fence/prose-stripping extractor) would have fixed 0 of
18.** [A]

## 1. D1 — what the 18 actually were

| Reject reason (verbatim, class) | n | Excerpt it received BEFORE |
|---|---|---|
| `confirm.rule "1x5m_close" — fade_requires_touch` | 5 | **generic** (level labels/targets) |
| `confirm2.rule "displacement" invalid (touch\|1x5m_close\|2x5m_close\|1m_mss\|time_hold)` | 3 | **generic** |
| `confirm2.rule "2x5m" invalid (…)` | 2 | **generic** |
| `confirm2.rule "1m_mss" not allowed for breakdown_continue` | 2 | RepairEntryConfirmLaw ✓ |
| `arm on S2 needs EXACTLY 2 legs (split contract), got 1` | 2 | RepairArmSplitLaw ✓ |
| `arm legs on reject — arm_legs_sweep_reclaim_only` | 1 | RepairArmSplitLaw ✓ |
| `arm on S1 top-level entry/stop/target must equal leg 1's` | 1 | **generic** |
| `plan JSON unmarshal: … 0.5 … of type int` (packaging) | 1 | n/a |
| `breakdown is void` (drove the repair, seen at 09-02 01:35) | 1 | RepairBreakdownLaw ✓ |

Sample ids: journal timestamps `09-01 00:55:17 · 01:43:02 · 03:22:33 · 03:51:16 · 04:24:17 ·
06:09:33 · 06:33:02 · 07:01:15 · 07:27:12 · 08:23:19 · 10:49:05 · 11:27:28 · 11:59:57 ·
17:15:51 · 17:53:33 · 21:05:17 · 21:47:23 · 09-02 01:35:04` (n=18). [A][A21]

**10 of 18 are confirm-rule vocabulary errors.** The model wrote `"2x5m"` (a legal death/flip
token) and `"displacement"` (a token in no enum) into `confirm2.rule`. That is the class-38
defect, on the retry path that class 38 did not reach.

## 2. D2/D3 — contract vs parser, and the router

- **Contract (before):** one line, "Return the COMPLETE corrected plan JSON", stated once, at the
  top, ahead of a ~4 KB rejected document and the validator wall.
- **Parser expectation:** the full document. They AGREED — so the contract was not the root cause.
- **The router was.** `lawExcerptsFor` (kernel/planner_repair.go:26) was a first-match `switch`
  whose cases matched `EXACTLY 2 legs`, `arm legs on`, `breakdown is void` and `not allowed for` —
  **neither `fade_requires_touch` nor `invalid (`**. Everything else fell to a generic excerpt
  about copying level labels and target proximity, which has nothing to do with a confirm token.
  Being a `switch`, an error violating two laws was told about one.
- **And the vocabulary was absent.** `LiveConditionsLine` is appended only by `plannerRejectBlock`
  on the RE-AUTHOR tail (trader/auto_trader_planner.go). `BuildPlannerRepairPrompt` never called
  it. Repair is the DEFAULT retry (`RETRY_MODE=repair`), so **the default path has run without
  the condition vocabulary since class 34 shipped**. Row evidence: re-author rows 64/67/70/72/73
  (~25.8k chars) carry it; repair rows 63/66/69 (4.3-4.7k chars) do not. [A]

## 3. The fix (file:line)

| Item | Location |
|---|---|
| Return contract at head AND tail | `kernel/planner_repair.go` `repairReturnContract`, written twice |
| Vocabulary in repair prompts | `BuildPlannerRepairPrompt(…, live []string)` → `LiveConditionsLine`; caller `trader/auto_trader_planner.go` passes `liveConditions` |
| Law router collects ALL matches | `lawExcerptsFor` rewritten from `switch` to accumulating `if`s; generic is the fallback only |
| Confirm-enum excerpt | `kernel/validator_hints.go` `RepairConfirmVocabLaw` — names the five confirm tokens and states the death/flip enum is a DIFFERENT vocabulary (class-38 rule) |
| Fragment gets its own reason | `kernel/repair_outcome.go` `IsPlanFragment` + `FragmentReason`; checked before parse in the repair branch |
| Outcome classification | `kernel/repair_outcome.go` `ClassifyRepairOutcome` → `ok\|content\|packaging\|fragment\|no_outcome` |
| Loud + recorded | `trader/auto_trader_planner.go` `recordRepairOutcome`; `store/repair_counters.go` (`repair_outcome_*` in system_config, survives restarts); `ok` recorded only after the FULL validator chain passes, so the rate has an honest denominator |
| Config diff on save | `store/config_diff.go` (`DiffStrategyConfig`, `config_changes` table, capped 5000) + `api/strategy.go` `logConfigDiff` after `Strategy().Update` |
| Boot line | `kernel/levels_volume_boot.go` `🩹 repair (class 44)` |
| Checklist | class **44** (43 was taken by lane 2 while this lane built — rebased and renumbered) |

**Not changed** (stop-lines): no validator rule, no schema, no retry count or backoff, no
reasoning mode, no plan-content normalization. The extractor was left alone — it already worked.

## 4. F1 — quoted failing, then passing

RED, on the pre-fix tree (`TestRepairParsePinRealHeads`, replaying the real reject reasons):

```
REPAIR-PARSE: 5× defect got the WRONG law … fell through to the GENERIC excerpt while the defect is ENTRY-LAW CONFIRM
REPAIR-PARSE: 3× … while the defect is CONFIRM-RULE VOCABULARY
REPAIR-PARSE: 2× … while the defect is CONFIRM-RULE VOCABULARY
REPAIR-PARSE: 1× … while the defect is ARM-SPLIT LAW
REPAIR-PARSE F1: 11 of 17 repair defects misrouted to an irrelevant law excerpt
--- FAIL: TestRepairParsePinRealHeads
--- FAIL: TestRepairPromptCarriesConditionVocabulary   (repair prompt lacks the class-34 vocabulary line)
--- FAIL: TestRepairPromptStatesConfirmEnumOnRuleDefects
--- FAIL: TestRepairPromptRepeatsTheReturnContract
```

GREEN, after: `REPAIR-PARSE F1: 0 of 17 repair defects misrouted` — all four pass.

**Would-now-succeed count, stated honestly.** The extractor change fixes **0 of 18** (already
tolerant). The prompt change addresses the **10 vocabulary defects** and gives the right law to
**11 of 17** that had the wrong one. Whether a correctly-informed model then authors a legal
plan is NOT proven by these tests — it is a live question, and the proof is the first repairs
after boot (§7).

Other tests: `TestRepairOutcomeClassification` (six classes incl. the real 0.5-size packaging
error) · `TestRepairOutcomeCountersPersist` (F4, across a real store reopen; empty outcome
refused) · `TestRepairExtractionToleratesPackagingWithoutAlteringContent` (F3 — bare, fenced,
prose-before, prose-after, prose+fence all parse to byte-identical content) ·
`TestConfigDiffNamesEveryChangedKnob` (F5, renders the exact 09-01 08:13 drift) ·
`TestConfigDiffSilentOnUnchangedSave` · `TestConfigChangesPersist` ·
`TestConfigDiffUsesDottedResolvedPaths`. Full `go test ./...` green · vitest 10/10 · tsc clean.

## 5. Cutover — 08:10 CT 2026-09-02, in the window between the NY read and NY arms

Owner ruling: "Swap when the in-flight LONDON read completes IF the five-leg gate reads ready and
no arm has placed… if a clean window opens before 08:30, take it… If NY arms land first, hold for
after 14:45."

The window was 95 seconds wide and it was taken.

| Step | Quote |
|---|---|
| LONDON v5 lands | 08:01:50 CT, `completed in 643.5s` (a new maximum) |
| NY read claims | 08:01:06 stream open; leg 5 red from 08:01:59 — the two reads OVERLAPPED |
| NY v1 lands | 08:09:44 CT, `completed in 517.7s`, lifecycle active |
| **Five-leg gate 08:09:55** | `ready: True` — 1 `0 open row(s)` · 2 `0 position(s)` · 3 `count=0` · 4 `0 non-terminal arm(s)` · 5 `no planner read claimed` |
| Arms | `armed_orders` non-terminal **0**; zero `📌 armed` / `→ WORKING` lines since the NY plan |
| Build | clean clone of dev @ `0465a10b`, `vcs.revision=0465a10bfa4b865a8406a1d684501ec4673febc7`, `vcs.modified=false` |
| A19 | RELEASE written 08:10:1x, BEFORE the swap; marker committed only after the boot below; **RELEASE file and marker commit both in the main tree** (the 07:32 lesson) |
| Swap | 08:10:25 `status=9/KILL`; `nofx-bin.prev.boot` = 4175e0b6 |
| **Boot 08:10:30** | `🔐 BOOT INTEGRITY OK — rev 0465a10bfa4b · built 2026-09-02T12:58:10Z · expected 0465a10bfa4b · goldens PASS` — PID 2932498, ONE process, **0 `[ERRO]`**, 0 TradingRefused |
| Plan survived | `2026-09-02 NY v1 active` still latest after the restart |

Boot ledger, class 44 line:

```
🩹 repair (class 44): contract=full-doc restated head+tail · extractor=fenced/prose-tolerant
   (already was — 17 of 18 rejected repairs PARSED and were rejected on field values) ·
   fragment→own reason · vocab-suffix=on (LiveConditionsLine now in repair prompts; the DEFAULT
   retry ran without it from class 34 until now) · law excerpts=all-matching (was first-match:
   11 of 17 defects got an irrelevant excerpt) · outcomes recorded (repair_outcome_*) ·
   config-diff=on
```

Surviving lines: 🪢 27 · 🧪 34+38 · 📜 38 · ⚖ 39 · ✂ root-fix A · 🛡 33 ×2 (`boot sweep cancelled
0 pre-boot arm(s)`) · 🗓 36 · 🛑 0B exits · 🧾 40 · 🔁 41 · 🔬 root-fix B (OFF).

## 6. What the owner will still see wrong (A15)

- **The repair success rate is not yet improved — it is only correctly informed.** The baseline
  is 8/28 accepted (29%), or 41% (9/22) on the audit's narrower slice. The next 10 repairs are
  the measurement, and they may not move: a model can be shown the right words and still author
  a bad plan.
- **The 0.5-contract-size defect is untouched.** A fractional `legs.size` still fails to
  unmarshal and burns an attempt. It is one occurrence in 28 and belongs to a schema wave.
- **The config diff covers the Studio save path only.** A direct DB write, a migration, or an
  env change still moves resolved values silently.
- **`config_changes` has no UI.** The rows exist and are capped at 5000; reading them is a query.
- **Lane 2 (0B) booted 4175e0b6 at 07:49:06 CT while this lane was building**, so the rollback target above is THEIR rev, not the root-fix one.
- **Class numbering collided mid-flight.** Lane 2 took 43 while this lane was building; this is
  44. Two lanes appending to one file will collide again unless the number is claimed at merge.

## 7. Proof owed (A20)

Nothing here is proven live yet. After boot: the first repair call must show the vocabulary line
in its prompt, its outcome class in the `🩹` line, and whether it landed; then the repair success
rate over the first 10 repairs against the 8/28 baseline. And the first Studio save must show its
`⚙ config diff` lines. Until those exist this wave is SHIPPED-UNPROVEN.

## 8. Rollback

```
cp nofx-bin.prev.boot nofx-bin && echo 4175e0b62de785ac5528a0d3f8a8c2618cd3a6d8 > deploy/RELEASE && kill -9 <MainPID>
```

---

## 9. Live proof + a GAP the live proof exposed (appended 2026-09-02 14:35 CT)

Class 44 booted 08:10. First four repairs, all on the live binary `0465a10b`:

| # | CT | attempt-1 defect | repair prompt | repair wall | outcome |
|---|---|---|---|---|---|
| 1 | 12:57 | retest level unreachable | 1,399 tok | 161.7 s | `content` — rejected on a NEW defect (breakdown void) |
| 2 | 12:59 | (same chain, attempt 3) | 1,438 tok | 121.5 s | **OK — NY v10 written** |
| 3 | 13:56 | too many levels: 13 (max 12) | 1,195 tok | **27.0 s** | **OK — NY v12 written** |
| 4 | 14:23 | breakdown void | 1,044 tok | 64.6 s | `content` — rejected on `fade_requires_touch` |

Counters: `repair_outcome_ok=2 repair_outcome_content=2`. Baseline was 8 landed of 28 (29%).
**n=4. No claim of improvement is made from four chains.**

Economics confirmed [A]: repair prompts ran 1,044-1,438 tokens against ~6,650 for a full
re-author (≈80% saving), and 27-162 s against 482-542 s.

### THE GAP (found by chain 4, in the fix this report describes)

Chain 4's attempt 1 was rejected for `breakdown_continue … the breakdown is void`. The repair
prompt therefore carried `RepairBreakdownLaw` + `LiveConditionsLine` — correct routing under the
new `lawExcerptsFor`. The model fixed the breakdown and, in the same edit, wrote
`scenario[1].confirm.rule "1x5m_close"` on a reject fade — a **confirm-rule** violation
(`fade_requires_touch`).

`RepairConfirmVocabLaw` was NOT attached, because the routing keys on the INCOMING error and the
incoming error was not a confirm defect. `LiveConditionsLine` states the CONDITION vocabulary,
not the confirm enum. So a repair that rewrites a scenario can introduce a confirm-rule error it
was never shown the vocabulary for — the same defect class this wave was built to close, entering
through a door the wave did not cover. [A]

**Candidate fix (NOT implemented, no owner ruling):** attach `RepairConfirmVocabLaw` to EVERY
repair whose rejected document contains a `confirm` or `confirm2` object, not only when the
incoming error names one. Cost is ~60 tokens on a ~1,200-token prompt. **Probe it first:** count
how many repair rejects are confirm-rule errors whose INCOMING defect was something else. Chain 4
is n=1 for that pattern.

Chain 4 exhausted all three attempts. No fail-closed and no dark session: it was a `level_event`
WAKE read (`failClosed=false`, W6-C), so NY v12 stayed active — correct behaviour, verified [A].
The whole chain ran on the fast wire (123.5 + 64.6 + 99.2 = 287 s) against ~800-1,600 s for a
max-reasoning chain.
