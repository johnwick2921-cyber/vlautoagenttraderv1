# Root-fix: shorten the planner call — what the measurement actually said

**Dispatch:** ROOT-FIX (A) schema slim · (B) fast-mode shadow A/B · (C) provider fallback.
Owner hoang, 2026-09-02. Base: dev `bed0a96c` (class 33 already merged and booted).
**Evidence tiers:** [A] verified directly · [B] inferred from strong evidence · [C] speculation.

## 0. HEADLINE — Part A's premise does not hold (A23)

The dispatch scoped Part A at "40-50% fewer output tokens" by slimming the plan schema. **The
plan JSON is not the output.** Measured before touching anything:

| Measurement | Value | Source |
|---|---|---|
| Full-author calls sampled | n=67 | `AI call complete (stream)` lines, `prompt>9000`, 2026-08-31 → 09-02 [A] |
| Completion tokens p50 | **23,769** | same (min 7,499 · max 41,743 · mean 22,376) [A] |
| Mean wall | **349 s** | same [A] |
| Mean reasoning chars | 72,477 | same [A] |
| Stored plan doc size | **3,088 bytes ≈ 920 tokens** | `select avg(length(doc)) from plans where created_at>'2026-08-28'` → n=61 [A] |
| **Plan JSON share of output** | **≈ 3.9%** | 920 / 23,769 [A] |
| **Reasoning share** | **≈ 96%** | remainder [B] |

Per-field weight in 12 recent stored docs [A]: levels ~402 tok · scenarios ~237 · reasoning
~161 · no_trade ~42 · bias ~33 · death_condition ~18 · flip ~14 · death ~10 · day_type ~3.

**Deleting the entire schema saves under 4% of the output — about 14 s of a 349 s call.**
The 40-50% target is unreachable by a factor of ten. Per A23 the cut was NOT shipped; the
finding is pinned instead.

## 1. Part A — consumption audit (A-1) and the decision

Every one of the 9 top-level fields has at least one reader, so there is no removal candidate:

| Field | Reader (file:line) |
|---|---|
| `reasoning` | `api/handler_plan.go:1326` (Ask-Planner context) |
| `bias` | `kernel/plan_render.go:157` · executor prompt |
| `levels` | `kernel/plan_render.go:161` · `api/handler_plan.go:554` · `trader/auto_trader_planner.go:1128` (machine grade) |
| `scenarios` | `kernel/plan_render.go:167` · `kernel/scenario_state.go:96` · `kernel/scenario_facts.go:24` |
| `no_trade` | executor prompt render |
| `death_condition` | `kernel/plan_render.go:178` · `api/handler_plan.go:653,1329` |
| `day_type` | `api/handler_plan.go:652,695` (plan diff) |
| `death` / `flip` | machine death/flip evaluation each cycle |
| (scenario) `instruction`, `trigger`, `invalid`, `chain_after`, `machine_grade`, `consumed` | `kernel/realign.go:60` · `trader/auto_trader_planner.go:1734` · `kernel/planner_prompt.go:279` · `kernel/scenario_facts.go:21` |

**Decision: no schema change.** Capping the prose fields would save ~100-160 tokens ≈ 0.6% of
output ≈ 2 s per call, against the risk of touching a contract that classes 38 and 39 have just
stabilised. That is risk without reward. Two pins hold the line:

- `TestRootFixEveryPlanFieldHasAReader` — fails if any field ever loses its last reader (then it
  IS a removal candidate).
- `TestRootFixPlanJSONIsASmallFractionOfOutput` — fails if the JSON share ever exceeds 15%,
  i.e. if the premise ever becomes true.

Boot line: `✂ planner schema: 9 top-level fields, ALL consumed … plan JSON ~920 tokens of a
23,769-token p50 output (3.9%); reasoning is ~96%, so schema slimming CANNOT shorten the call`.

## 2. Part B — the fast-mode shadow A/B (shipped, dormant)

The reasoning mode is the only lever on the 96%. The instrument ships; the mode change does not.

| Piece | Location |
|---|---|
| Knob + target + counter | `trader/rootfix_shadow_ab.go` (`SHADOW_AB_ENABLED` default **OFF**, `SHADOW_AB_N` default 10) · `store/shadow_ab_counter.go` (`shadow_ab_calls_rootfix`, recorded) |
| Runner registration | `trader/auto_trader_planner.go` at the live call site (client, system prompt, budget, fast wire) |
| Trigger | `trader/auto_trader_planner.go`, immediately after `recordPlanWritePrice` — i.e. **after** the live read finished |
| Offline verdict | `shadowVerdictFor` replays the live chain in order: `ParsePlanDocCapped` → required bias → `MislabeledStructuralLevels` → `ValidatePlanDocWithFactsMachine` → `ValidateFvgEntryScenarios` (same origin universe) → `ValidateBreakdownContinueScenarios` |
| Telemetry | `mcp.LastCompletionTokens` / `LastReasoningChars` (new accessors) |
| B-1 facts | `store/planner_rejected.go` gains `Facts`; `SaveRejectedPromptWithFacts`; every reject site passes `FactsSnapshotJSON(facts)` |

Safety properties, each fixture-pinned: off by default · never fires with an empty prompt ·
one shadow at a time process-wide (`CompareAndSwap`) · never concurrent with a live stream
(fires only after the read completes — class 41 P3 has not excluded concurrency as a cut cause)
· writes no plan and spends no re-plan budget · stops firing at the target n · restores the live
wire on the shared client afterwards · panics are caught and logged, never propagated (A10).

The row it logs:

```
🔬 shadow A/B 3/10 (2026-09-02 NY): fast=ILLEGAL tokens=4210 wall=92.0s (21% of live) ·
   live max tokens=23769 wall=448.0s · reasons=… — SHADOW ONLY, no plan written, no replan budget
```

### B-3 — the criterion, registered BEFORE the data

Written into the Guide (`settings.ts`, new knob card) and into the boot line:

> Fast mode is promoted to live ONLY IF, at n≥10 shadow calls, its legal-plan rate is ≥ max
> mode's on the same prompts AND its median wall time is ≤50% of max's. Otherwise it stays
> shadow or is dropped. No promotion on narrative.

The earlier fast-mode rejection was n=1 and pre-dates the class-38 prompt contract, which now
states the split-arm rule the fast plan violated. That evidence is stale, not a verdict.

### B-4 — the knob, honestly

`AI_PLAN_REASONING` (default `max`) is resolved through `reasoningWire` at
`trader/auto_trader_loop.go:72`. It is an **environment variable read at call time**, so a
promotion is an env change plus a restart — **not** a Studio config click, as the dispatch
assumed. Correcting that assumption here rather than building a config surface nobody asked for.

## 3. Part C — provider fallback: REPORT-ONLY

No owner ruling for this is in chat, so per C-2 nothing was added: no provider row, no key, no
knob. What it would touch, for the ruling:

- `mcp/client.go` — the retry loop already classifies `class=transport` and exhausts
  `StreamTries`; the fallback hook belongs where `CallWithRequestStreamRetryDeadlines` returns
  its final error.
- `trader/auto_trader_planner.go` — the class-41 provider-failure branch already re-sends the
  identical prompt; the fallback would swap the provider row for that resend only.
- Resolver key it would add: `AI_PLAN_FALLBACK_ROW` (an `ai_models` row id), plus a recorded
  counter `planner_fallback_used`.
- The owner must name the row (a second DeepSeek key/endpoint, or another provider). **STOP.**

## 4. Tests

`TestRootFixEveryPlanFieldHasAReader` (D1) · `TestRootFixPlanJSONIsASmallFractionOfOutput` (D3,
reports the real share: **1.6% on the synthetic doc, 3.9% on real stored docs** — the dispatch's
≥40% delta is not achievable and this says so) · `TestRootFixShadowDisabledByDefault`,
`TestRootFixShadowTargetResolved`, `TestRootFixShadowOffFiresNothing`,
`TestRootFixShadowRefusesEmptyPrompt`, `TestRootFixShadowLineWording`,
`TestRootFixShadowLineBounded`, `TestRootFixShadowBootLineStatesCriterion` (D4) ·
`TestRootFixFactsSnapshotRoundTrip` (D5) · class-38 contract + hint guard still green (D2) ·
full `go test ./...` green · vitest guide 10/10 · `tsc --noEmit` clean (D7).

## 5. Cutover — staged, and a live proof arrived while staging

Staged: clean clone of dev @ `928e49d2` → `vcs.revision=928e49d297f97b9dacac8456da65a6b1ab9eccba`,
`vcs.modified=false`, copied to `nofx-bin.next`. Running: `8a756bba` PID 2065518. Rollback slot
`nofx-bin.prev.boot`.

**CLASS-33 LEG 5 FIRED, ON ITS FIRST REAL USE — and it stopped this cutover.** At 07:21:33 CT
`GET /api/cutover-gate` returned:

```
ready: False
  1 db_open_positions      True  | 0 open row(s)
  2 api_positions          True  | 0 position(s)
  3 nt8_positions_snapshot True  | count=0
  4 working_orders         True  | 0 non-terminal arm(s)
  5 planner_in_flight      False | planner read IN FLIGHT: 2026-09-02:LONDON:8d5c8af5_…_deepseek
```

The journal confirms a genuine read was mid-flight: stream opened 07:15:50, `🧠 planner call …
completed in 369.5s` at 07:21:59, `🗓️ PLAN written 2026-09-02 LONDON v3`. **All four of the OLD
legs passed.** A pre-class-33 cutover would have swapped and killed that chain at ~5.5 minutes
in — precisely the 2026-08-31 17:34 CT defect (class 33 defect B1) reproduced 24 minutes after
the fix shipped. Held per A6; re-quoted at 07:22:12 CT → `ready: True`, all five legs pass.

This is the live proof class 33's report listed as owed for leg 5. [A]

**Swap (A26 — the owner runs it):**

```
cd /home/hoang/nofx
echo 928e49d297f97b9dacac8456da65a6b1ab9eccba > deploy/RELEASE
cp nofx-bin nofx-bin.prev.boot && mv nofx-bin nofx-bin.old.8a756bba && mv nofx-bin.next nofx-bin
kill -9 <MainPID>
```

Re-quote the gate immediately before the kill: LONDON has been re-reading (v2 07:15, v3 07:21)
and the NY read fires at 08:00 CT. `ready:false` = HOLD.

## 6. What the owner will still see wrong (A15)

- **The call is still ~350-450 s.** This wave shipped no speed change — it shipped the
  measurement that says where the time goes and the instrument to test the one real lever.
  A live 451.4 s LONDON v2 read landed at 07:15 CT today, on the class-33 binary, mid-wave.
- **The shadow A/B is OFF.** Nothing measures fast mode until the owner sets
  `SHADOW_AB_ENABLED=1`. Each shadow call costs one extra provider call per session read.
- **Promotion is an env change plus a restart**, not a config click (B-4 above).
- **The facts column is empty for existing rows** — only rejects written after this boot carry
  the snapshot, so an offline A/B over older rows still runs the schema gate alone.
- **Part C is not wired.** After three transport tries fail, the attempt is still consumed
  against the same provider row.

## 7. Rollback

```
cp nofx-bin.prev.boot nofx-bin && echo 8a756bba4a21ab455beafac75bf6415e71de2fb9 > deploy/RELEASE && kill -9 <MainPID>
```

### 5.1 Boot record (owner GO 07:31 CT 2026-09-02)

Gate re-quoted at **07:31:57 CT** immediately before the swap — `ready: True`, all five legs
pass (leg 5 `no planner read claimed`). RELEASE written before the swap; marker after the boot.

Old process exited 07:32:10 (`status=9/KILL`); started 07:32:15; **PID 2461883, exactly one
nofx-bin process, 0 `[ERRO]` lines, 0 TradingRefused**:

```
🔐 BOOT INTEGRITY OK — rev 0d093c3b3a11 · built 2026-09-02T12:22:31Z · expected 0d093c3b3a11 · goldens PASS
✂ planner schema: 9 top-level fields, ALL consumed (audit 2026-09-02: levels~402tok scenarios~237
  reasoning~161 no_trade~42 bias~33 death_condition~18 flip~14 death~10 day_type~3) — plan JSON
  ~920 tokens of a 23,769-token p50 output (3.9%); reasoning is ~96%, so schema slimming CANNOT
  shorten the call — the reasoning MODE is the lever (root-fix part A: measured, no cut shipped)
🔬 shadow A/B (root-fix part B): OFF target_n=10 done=0 (SHADOW_AB_ENABLED, SHADOW_AB_N) —
  fast-mode plans validated offline, never written; promotion criterion: legal-rate ≥ max AND
  median wall ≤50% of max at n≥10
```

Surviving lines: 🪢 class 27 · 🧪 hints (34+38) · 📜 contract 17 (38) · ⚖ normalizer (39) ·
🗓 preflight (36) · 🧾 P&L 12/0 (40) · 🛰 planner client (37) · 🔁 stream policy (41) ·
🛡 cutover safety ×2 (33, `boot sweep cancelled 0 pre-boot arm(s)`).

**PROOF STATUS (A20).** Part A's measurement is PROVEN (n=67 calls, n=61 plans, quoted above)
and pinned by two tests. Part B's instrument is SHIPPED-DORMANT: `SHADOW_AB_ENABLED` is OFF, so
**no shadow row exists yet** — the first 🔬 row is owed once the owner enables it, and only at
n≥10 does the pre-registered criterion decide anything. Part C is REPORT-ONLY, unwired.
The wave shipped NO speed change; the next full-author read will still run ~350-450 s.
