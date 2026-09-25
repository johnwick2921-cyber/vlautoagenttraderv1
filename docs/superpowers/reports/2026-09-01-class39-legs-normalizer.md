# CLASS 39 — normalize-don't-reject: legs on a non-sweep condition collapse to a single arm

Date: 2026-09-01 · Owner: hoang · Worktree `../nofx-class39` (branch `fix/class39-legs-normalizer`)
Evidence tiers: **[A]** directly verified · **[B]** inferred from strong evidence · **[C]** speculation.
All times CT (R8). Live rev at dispatch: `c0580011` (class 38), PID 2030083.

## STATUS

| Item | State |
|---|---|
| Code | **merged to dev @ `aeb11179`** (ff from `91c97dc6`); report `aeb11179`→`e31e9146`; marker `9df72702` |
| Build | clean clone `--no-local` at `aeb11179`, `vcs.modified=false`, built 2026-09-02T04:12:33Z, sha256 `df0a24d4256b9767…`, 70,930,632 B |
| Cutover | **DONE 00:01:01 CT 2026-09-02 on the owner's GO ("just go", overriding the live-arm HOLD)** — `🔐 BOOT INTEGRITY OK — rev aeb11179df5a · expected aeb11179df5a · goldens PASS` 00:01:07, PID 2089356; **marker committed AFTER the passed boot (A19)** — see §10 |
| Proof (A20) | **NOT YET OCCURRED** for 39 — the first live arm on a non-sweep condition carrying legs. (Class 38's proof DID land during this wave: 23:12:44 CT, attempt 1 of 3, ASIA v5 active, zero rejects — see the class-38 report §12.) |
| Lock (A2) | `~/nofx-main.lock` acquired 22:58 CT (no prior holder), released at closeout |
| Stop-lines | held: no leg synthesized · sweep_reclaim untouched · no second pass · validator logic unchanged beyond the C1 insertion · no prompt text · no retry semantics · no knob beyond the cap · 35/36/37/38 paths untouched |

---

## 1. The ruling (owner, 2026-09-01) — verbatim, applied literally [A]

> On a NON-sweep_reclaim condition, when the authored arm carries any `legs` array (one leg or
> many): 1. DROP the `legs` array. 2. RE-RUN the full arm validation on what remains (top-level
> entry / stop / target / wait_confirm / rule). 3. If VALID: proceed with the single arm; emit a
> WARN naming exactly what was dropped (leg count; each leg's entry/stop/target/rule) and the
> condition. 4. If STILL INVALID: REJECT UNCHANGED with the original reason. No second
> normalization pass. NEVER synthesize a leg. NEVER normalize the reverse: one leg on
> sweep_reclaim stays a reject. Cap: plannerRejectedCap 20 → 200.

Why it is worded "any legs array" and not "keep the matching leg": the only retained non-sweep
instance (row 69 S1) had ONE leg with rule `touch`, which is not in breakdown_continue's confirm
set, so a keep-the-matching-leg rule would still have rejected it — while its top-level arm
already carried a valid single arm. Dropping the array is deterministic: every non-sweep
condition arms single by contract.

**Scale (72h to 2026-09-01) [A]:** 35 of 121 validator rejects were legs on a non-sweep
condition (breakdown_continue 24, reject 11); 7 landed on attempt 3/3; two sessions fail-closed
directly on it (`planner_rejected_prompts` 69, 80). Class 38's schema qualifier should shrink
this; 39 catches the remainder without burning a retry.

## 2. Sites — verified before editing (A17) [A]

| Site | Verified at | Role |
|---|---|---|
| C1 `NormalizePlanDocRules` | `kernel/plan_doc.go:442`, called at `:549` inside `ValidatePlanDocWithCaps` — before `validateArmSpecs` (`:228`) reaches `ArmSpecValid` (`:126`) | the normalizer's home; runs on every parse/validate path |
| C2 the reject | `plan_doc.go:167` `arm_legs_sweep_reclaim_only` | **UNCHANGED** — still fires for sweep_reclaim shapes and for non-sweep arms still invalid after the drop |
| C3 WARN precedent | `trader/auto_trader_planner.go:1436` (level auto-collapse) | the house style the ⚖ line follows |
| C4 cap | `store/planner_rejected.go:49` `plannerRejectedCap = 200` | already on dev (`eaac141c`); rides in this build |
| C5 E8 writer | `trader/armed_executor.go:logShadowAB` → `store/ab_confirm.go` `AbConfirmStore.Upsert`; the plan doc is on `plan.Doc` (`kernel/plan_render.go:22`) | the stamp lands here |
| C6 guide | `web/src/guide/content/plays.ts:79` ARM SPLIT block | — |
| C7 checklist | class 38 at `:378`; 39 appended at `:412` | — |
| C8 samples | rows 69 (S1 breakdown_continue 1 leg rule=touch; S2 sweep_reclaim 1 leg) and 85 (S1 sweep_reclaim 1 leg) | arm JSON quoted in §6 |

## 3. The fix — file:line [A]

| File | Lines | Change |
|---|---|---|
| `kernel/plan_doc.go` | 274 | `PlanDoc.ArmNormalizations []ArmNormalization` (`json:"arm_normalizations,omitempty"`) — the record travels WITH the stored doc |
| `kernel/plan_doc.go` | 465 | `NormalizePlanDocRules` calls `normalizeArmLegs(d)` — the C1 insertion |
| `kernel/plan_doc.go` | 1131-1230 | `ArmNormalization` type · `normalizeArmLegs` (steps 1-4; `:1169` skips sweep_reclaim — D7; `:1176` "still invalid → continue unchanged" — step 4; `:1186` `sc.Arm.Legs = nil` — step 1) · `ArmNormalizationWarn` (D2 wording) · `ArmNormalizationFor` / `DroppedLegsJSON` (E8 stamp) |
| `trader/auto_trader_planner.go` | 1438-1446 | after `ParsePlanDocCapped` succeeds: `⚖` WARN per normalization + `store.IncArmsNormalized` (D2, D5) |
| `trader/armed_executor.go` | 561, 582 | E8 row: `Normalized: norm != nil, DroppedLegs: kernel.DroppedLegsJSON(norm)` (D4) |
| `store/ab_confirm.go` | 58-59, 123-124, 179 | `normalized` + `dropped_legs` columns: struct, DDL/migration, upsert map |
| `store/arm_normalized_counter.go` | new | `arms_normalized_class39` in system_config, one atomic UPSERT (D5) |
| `kernel/levels_volume_boot.go` | 46 | `⚖ arm normalizer: legs on non-sweep → single arm + WARN (class 39); sweep_reclaim contract unchanged; counter arms_normalized_class39 recorded in system_config` (D8) |
| `web/src/guide/content/plays.ts` · `status.ts` | 79-99 · 23 | ARM SPLIT block gains the class-39 rule; boot ledger gains the ⚖ line (A12) |
| `docs/superpowers/AUDIT-CHECKLIST.md` | 412 | CLASS 39 (A16; 33 unoccupied, 27/28/29 untouched) |

**D3 — the reject path, quoted.** `normalizeArmLegs` validates the de-legged copy FIRST
(`plan_doc.go:1176`); when that still fails it `continue`s without touching the scenario, so
`validateArmSpecs` (`:228`) emits the ORIGINAL error from `ArmSpecValid` (`:167` for the legs
branch). In the trader that error is `perr` at `auto_trader_planner.go:1413` → `lastErr` (`:1415`)
→ the repair prompt `BuildPlannerRepairPrompt(lastRaw, lastErr.Error())` (`:1396`) or the
re-author block `plannerRejectBlock(lastErr, …)` (`:1422`). No normalization text exists on that
path; `TestClass39RejectPathCarriesOriginalReason` asserts the retry prompt carries
`arm_legs_sweep_reclaim_only` and contains no "normaliz".

**Literal-reading note:** the ruling says "any legs array"; the normalizer therefore also drops
legs from a non-sweep arm whose `enabled` is false (the re-validation trivially passes). That
case never rejected before, so the only visible effect is the ⚖ WARN.

## 4. Tests — E1 and E4 quoted, then everything else (A8, E1-E10) [A]

**E1 RED, pre-fix tree** (`77c0c2b1`, `go test ./kernel -run TestClass39`):
```
--- FAIL: TestClass39PinRow69S1
    class39_pin_test.go:79: row 69 S1 (breakdown_continue, one leg rule=touch, top-level already
    mirrors it) must NORMALIZE and validate; got the reject: arm legs on breakdown_continue —
    arm_legs_sweep_reclaim_only (the split entry is the sweep_reclaim contract; other conditions arm single)
--- PASS: TestClass39ReversePin
```
**E1 GREEN, post-fix** (`c403e931`): `TestClass39PinRow69S1` PASS, legs dropped, single arm kept.

**E4 — the reverse pin.** Passes on the pre-fix tree AND the post-fix tree by design: it pins
what must NOT change. It cannot be "quoted failing" — a test that fails before the fix would mean
the reverse direction was already being normalized, which it never was.

| Test | Pins | Result |
|---|---|---|
| `kernel.TestClass39PinRow69S1` (E1) | row 69 S1 → normalized, top-level kept | red → **PASS** |
| `kernel.TestClass39WarnNamesDroppedLeg` (E1b) | ⚖ wording: condition, S1, `dropped legs[1]`, `#1 entry=… rule=touch`, kept arm + rule + wait_confirm; `ArmNormalizationFor`, `DroppedLegsJSON` | PASS |
| `kernel.TestClass39MultiLegNormalized` (E2) | 3 legs on breakdown_continue with a valid top-level → all dropped, proceeds, WARN lists #1 #2 #3 | PASS |
| `kernel.TestClass39InvalidTopLevelRejectsUnchanged` (E3 kernel) | legs on `reject` with target above entry (long) → error string IDENTICAL to `ArmSpecValid` on the authored scenario; no record; legs untouched | PASS |
| `trader.TestClass39RejectPathCarriesOriginalReason` (E3/D3) | attempt-2 repair prompt carries `arm_legs_sweep_reclaim_only`, no "normaliz"; counter 0 | PASS |
| `kernel.TestClass39ReversePin` (E4) | rows 69 S2 / 85 S1 still `needs EXACTLY 2 legs`, legs intact; a legal 2-leg split unchanged | PASS (pre and post) |
| `kernel.TestClass39NeverSynthesizeALeg` (E5) | fixture: legs only ever go to 0 or stay; every kept/dropped leg was authored (value-equal); total never grows · source guard: no non-test kernel file has a `PlanArmLeg{` literal or a `.Legs = append(`; scanned >50 files | PASS |
| `store.TestClass39E8RowCarriesNormalizationStamp` (E6) | `normalized`/`dropped_legs` survive create AND the upsert-update path | PASS |
| `store.TestClass39CounterPersistsAcrossRestart` (E7) | 2 increments → close → reopen → 2 → 3 | PASS |
| `store.TestClass39RejectedCapTrimsOldestAt201` (E8) | cap == 200; 201st insert trims r0; newest survives | PASS |
| `trader.TestClass39WriteSiteNormalizesAndRecords` (D2/D5) | legged non-sweep arm lands on attempt 1 (1 call), counter reads 1 | PASS |
| `kernel.TestClass39ReplayRetainedRows` (E9) | see §6 | PASS |

**E10:** goldens PASS · `tsc --noEmit` clean · `npx vitest run` **37 files / 295 tests passed** ·
`npm run build` ✓ (4.78 s) · full Go suite `go test ./... -count=1` → **27 packages ok, 0 FAIL, EXIT=0**

## 5. E8 stamp (D4) and counter (D5) [A]

`ab_confirm_log` gains `normalized INTEGER NOT NULL DEFAULT 0` and `dropped_legs TEXT NOT NULL
DEFAULT ''` (migration is the existing dup-tolerant `ALTER TABLE … ADD COLUMN` loop, so the live
table upgrades in place at boot). `logShadowAB` looks the scenario up in `plan.Doc.ArmNormalizations`
and stamps both. Counter: `system_config` key `arms_normalized_class39`, bumped by one atomic
`INSERT … ON CONFLICT DO UPDATE SET value = value+1` — no read-modify-write, survives restarts
(E7). Before this wave every refusal counter in the system was log-only or in-memory (Appendix L3
"fifth instance"); this one is recorded by construction.

## 6. E9 — replay of every retained rejected prompt with a legs array [A]

Sample honesty (A21/A24): the store holds 20 rows (ids 67-86). Only the REPAIR rows embed the
model's verbatim output; of those, exactly two carry a legs array — rows **69** and **85** (the
other "legs" hits in the table are the schema line of full prompts). Re-scanned at 23:09 CT before
the build: no new legged output had landed. n = 3 scenarios in 2 docs.

Replayed at the caps the live read resolved — `max_levels 12 · scenario_cap 5` from
`strategies.config` for `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` (A11; at the shipped default of 8
the replay rejects on level count before reaching the arm branch, which is what my first run did):

| Row | Scenario | Condition | Authored arm | New path |
|---|---|---|---|---|
| 69 | S1 | breakdown_continue | `{"enabled":true,"entry":29130,"stop":29168,"target":29040,"wait_confirm":true,"legs":[{"entry":29130,"stop":29168,"target":29040,"size":1,"wait_confirm":false,"rule":"touch"}]}` | **NORMALIZE-AND-PROCEED** — `⚖ arm normalized (class 39): breakdown_continue S1 — dropped legs[1] (#1 entry=29130.00 stop=29168.00 target=29040.00 rule=touch); single arm kept entry=29130.00 stop=29168.00 target=29040.00 rule=1x5m_close wait_confirm=true` |
| 69 | S2 | sweep_reclaim | `{"enabled":true,"entry":29082.75,"stop":29035,"target":29179,"wait_confirm":true,"legs":[{…"rule":"touch"}]}` (1 leg) | legs UNTOUCHED — doc **STILL REJECTS**: `arm on S2 needs EXACTLY 2 legs (split contract), got 1` |
| 85 | S1 | sweep_reclaim | `{"enabled":true,"entry":29048,"stop":29024,"target":29110,"wait_confirm":true,"legs":[{…"rule":"touch"}]}` (1 leg) | legs UNTOUCHED — doc **STILL REJECTS**: `arm on S1 needs EXACTLY 2 legs (split contract), got 1` |

Split: **1 normalize-and-proceed · 2 still-reject** (both sweep_reclaim, the reverse direction the
ruling protects). Note row 69's DOC still fails as a whole because its S2 is a one-leg sweep — class
39 fixes S1, and S2 is exactly the shape that must stay a reject. That is the correct outcome, not
a shortfall.

## 7. Cap (D6) [A]

`plannerRejectedCap = 200` (`store/planner_rejected.go:49`, from `eaac141c`, already on dev).
Storage projection from the live table (read-only, 23:0x CT): 20 rows, avg 20,261 B, max 25,939 B
→ **200 rows ≈ 3.86 MiB (worst case 4.95 MiB)** against a 653 MB database.

## 8. Build, stage, rollback (A4/A13) [A]

```
git clone --no-local ~/nofx <scratch>/clone-c39 && git checkout aeb11179df5a52b400e03c0cefbee52d5e4e67b9   (clone porcelain-clean)
go build -o nofx-bin .
	build	vcs.revision=aeb11179df5a52b400e03c0cefbee52d5e4e67b9
	build	vcs.time=2026-09-02T04:12:33Z
	build	vcs.modified=false
sha256 df0a24d4256b9767c4f14693…  (70,930,632 bytes) → staged ~/nofx/nofx-bin.next (slot was free)
goldens + class-39 guards re-run inside the built tree: ok
```
**Rollback (exact):**
```
cd ~/nofx && mv nofx-bin nofx-bin.bad.aeb11179 && cp nofx-bin.prev.boot nofx-bin \
  && printf 'c0580011b4ce4fdefa9d92566019a6d5789d5c1e' > deploy/RELEASE \
  && kill -9 $(systemctl show -p MainPID --value nofx) && git checkout -- deploy/RELEASE web/src/guide/types.ts
```
`nofx-bin.prev.boot` = the class-38 binary `c0580011`, also kept as `nofx-bin.old.c0580011`.

## 9. What the owner will STILL see wrong (A15)

- **Until cutover:** legs on a non-sweep condition still burn an attempt on `c0580011`.
- **After cutover:** a one-leg `sweep_reclaim` (rows 69 S2 / 85 S1) still rejects with
  `needs EXACTLY 2 legs` — by ruling. Class 38's prose now states the split contract; if the model
  keeps authoring one-leg sweeps, that is the next thing to look at, and it is NOT a normalization.
- The ⚖ WARN is loud but the plan card does not render the normalization; the stored doc carries
  `arm_normalizations` and the E8 row carries the stamp — forensics, not UI.
- Class 38's behavioural proof (first read authored on the new prompt) and class 37's (first read
  past 600 s) are still outstanding; no read has run since 22:23 CT.
- The guide drift banner shows until this build is cut over and its marker lands.

---

## 10. CUTOVER — DONE (owner GO, 2026-09-02 00:01 CT) [A]

**Cutover log (verbatim, in order):**
- 23:36:33 CT GATE → HOLD. A5 1-4 all clean (DB OPEN 0 · API positions [] · NT8 count=0 ×2 · API open-orders []), A6 clean (replan_in_flight false, 0/0 reads), BUT armed_orders row 29 = ASIA v6 S1 leg 0/2 state WORKING limit 29035.25 (📌 armed S1 → WORKING 23:21:03, signal d00abc07…). Owner rule: arm placed since 23:20 → HOLD. Watcher armed (60 s poll, ≤90 min) for arms=0 · open=0 · reads in flight=0.
- FINDING (report, A15/A23): /api/open-orders returned [] while a WORKING resting limit existed per the ledger and the 📌 line — flat-gate leg 4 is blind to at least this arm shape; the armed_orders ledger check is what held the swap.
- ROOT CAUSE of the leg-4 blindness: trader/ninjatrader/tcp_trader.go:1149-1151 `func (t *TCPTrader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) { return []types.OpenOrder{}, nil }` — a stub. /api/open-orders (api/server.go:373 → trader.GetOpenOrders) is therefore ALWAYS [] for NinjaTrader. Flat-gate leg 4 has been vacuous at the 35/36/37/38 cutovers; the working-order truth lives in armed_orders.state (WORKING) and the NT8 order_update frames. Recommendation (not this wave): the cutover rite's leg 4 should read `armed_orders WHERE state='working'` (or a real order-list frame once the AddOn sends one — Appendix L3 hygiene item). Class 33 candidate.
- 23:43:02 CT a level-wake read started (5th wake, OB(bear)·4h invalidated) → A6 hard-holds the swap regardless of the arm override; waiter armed for the landing.
- 23:47:13 CT CLASS-37 LIVE EVIDENCE: `ai_call model=deepseek-v4-pro duration_ms=250094 ok=false retries=1 ttfb_ms=510 reasoning_chars=54986 timeout_source=transport deadline_s=600 class=transport http_status=200 request_id="" err="stream interrupted: unexpected EOF"` → 23:47:15 `⚠️ AI API stream failed, retrying (2/2)` → `Request URL (stream idle=30s total=1200s)` re-sent with the IDENTICAL prompt. First live sighting of a class= token on a failure line and of the client-level transport retry on the 1200s stream. (request_id="" — DeepSeek returned none of the probed headers.)
- 23:47:33 CT retry ALSO hit `stream interrupted: unexpected EOF` (18.2 s in, retries=2) → `🛰 planner call FAILED class=transport provider_row=8ef641a7-…_deepseek model=deepseek-v4-pro http_status=200 request_id="" elapsed=270.3s idle=30s total=1200s — still failed after 2 retries` (first live sighting of the class-37 FAILED line) → `📐 planner attempt 1/3 failed` → `🧩 planner attempt 2/3 reauthor+block: prompt ~6783 tokens` — the re-author block carries the TRANSPORT text as the "validator reason" (owner ruling M4 2026-09-01: transport/deadline failures should retry the IDENTICAL prompt — NOT yet implemented; queued, not this wave). Two consecutive provider-side EOFs on the same read = DeepSeek stream instability at ~23:47 CT. A6 hard-hold continues until PLAN written / FAIL-CLOSED. arm 29 still WORKING.
- 23:52:41 CT THIRD cut on the same read: attempt 2/3 call 1 `ai_call … duration_ms=307957 ok=false retries=1 ttfb_ms=685 reasoning_chars=69503 class=transport http_status=200 request_id="" err="stream interrupted: unexpected EOF"`. Pattern: 3 provider-side mid-body EOFs in 6 min (250 s / 18 s / 308 s), all 200 OK at the head — DeepSeek stream instability, not our ceiling (no client_timeout/total_deadline class). The class-37 telemetry attributes every one; request_id="" throughout.
- 23:55:06 CT attempt 2/3 landed at the transport level (453.7 s incl. the 308 s cut + 2 s backoff + ~144 s retry) but the VALIDATOR rejected it: `S3 breakup_continue: a close came back across 29068.05 — the breakdown is void; author a \`reject\` play instead` — a facts rule (breakdown-void), not a leg-contract / token shape → not a class-38 counterexample. Attempt 3/3 (repair) begins; A6 hold continues.

- 23:55:06 CT attempt 3/3 (repair, ~986 tokens) landed 23:58:16 in 189.9 s → `🗓️ PLAN written 2026-09-01 ASIA v7 … lifecycle active` — the read is over; A6 clears.
- 00:00:19 CT first gate run tripped on MY OWN in-flight counter (`planner model` lines minus `planner call` lines = −3): retried calls emit one `planner call` line per attempt against one `planner model` line per read. Not a real read in flight (`replan_in_flight:false`, v7 just written). Re-keyed the check to *last model line older than last plan write*. Lesson recorded: an in-flight test must key on read boundaries, not call counts.

**Final gate, 00:01:01 CT — hard gates green, live arms advisory per the owner's override:**

| Gate | Value |
|---|---|
| A5 1/4 DB `status='OPEN'` | **0** |
| A5 2/4 `GET /api/positions` | **`[]`** |
| A5 3/4 NT8 positions snapshot | **`count=0`** Sim101 · **`count=0`** SimAccount1 |
| A5 4/4 `GET /api/open-orders` | `[]` — **stub, always `[]` on NinjaTrader** (`trader/ninjatrader/tcp_trader.go:1149`); see the finding below |
| A6 `replan_in_flight` | **false** · last read started 23:43:02 · last plan write 23:58:16 → **no read in flight** |
| A7 live arms | **2 WORKING** (ids 29, 30 — ASIA v6 S1 split legs 0/1, limits at 29035.25) — **owner override "just go"** at 23:4x CT, after the 23:36 HOLD |

**Swap:** `deploy/RELEASE` file written to `aeb11179…` at 00:01:01 (uncommitted) → `cp nofx-bin nofx-bin.prev.boot && mv nofx-bin nofx-bin.old.c0580011 && mv nofx-bin.next nofx-bin` → `kill -9 2030083` at 00:01:01 → systemd relaunched → PID **2089356** at 00:01:06.

**Boot checklist (A19 — one boot, then one marker) [A]:**
```
00:01:07 🔐 BOOT INTEGRITY OK — rev aeb11179df5a · built 2026-09-02T04:12:33Z · expected aeb11179df5a · goldens PASS
00:01:07 ⚖ arm normalizer: legs on non-sweep → single arm + WARN (class 39); sweep_reclaim contract unchanged;
         counter arms_normalized_class39 recorded in system_config                                  ← class 39
00:01:07 📜 prompt/validator contract: 17 restrictions, all stated in prompt (class 38 guard)        ← 38 intact
00:01:07 🧪 validator hints: 15 sites — … every rule token in its own field enum (class 34 + 38)     ← 34/38 intact
00:01:07 🚀 planner speed wave … stream_idle=30s stream_total=1200s (class 37 …)                    ← 37 intact
00:01:07 🗓 preflight: scheduled reads bypass freshness in halt/weekend (class 36)                   ← 36 intact
00:01:07 🛰 planner client: provider_row=8ef641a7-…_deepseek stream_idle=30s stream_total=1200s http_ceiling=600s …
00:01:07 ✅ Trader auto-started successfully · positions snapshot count=0 (both accounts) · zero error-level lines
```
**Marker `9df72702`** committed at 00:0x CT AFTER the checklist above — `deploy/RELEASE` + `GUIDE_BUILT_REV` = `aeb11179…`. The gap between the RELEASE file write and the marker commit was the attended swap window only.

**The two working arms across the restart:** ledger after boot → `29:working 30:working `. Arm lines since boot:
2026-09-02T00:01:07-05:00 🪢 netting-orphan wave (class 27, 2026-08-31): netting-flat cancels brackets (C# sweep + Go desync cancel_order) · exit r
2026-09-02T00:01:07-05:00 ⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=ON arm_rr=2.0 (gate-at-arm only; market-entry floor 2.0 u
2026-09-02T00:03:07-05:00 ⚔️ arm S1 leg 1 wait_confirm MET (touch) — arming
2026-09-02T00:05:07-05:00 ⚔️ arm S1 leg 1 wait_confirm MET (touch) — arming
NT8 positions stayed at count=0 through the restart; the resting limits belong to the broker and were not filled during the ~5 s gap.

**Proof (A20) — NOT YET OCCURRED for class 39.** The proving event is the first live arm on a
non-sweep condition that carries legs: quote the `⚖ arm normalized (class 39): …` WARN, the kept arm,
and whether it placed. Tonight's three reads on the class-38 prompt authored the contract correctly
(v6 S1 legal 2-leg sweep + S2 single-arm reject; v7 likewise), so the normalizer had nothing to do —
with 38 stating the contract, rare is the desired outcome. The counter `arms_normalized_class39` reads 0.
The persistent journal watch stays armed for the first ⚖ line.

**Live-surface findings from the cutover (A15/A23), not changed here:**
1. **Flat-gate leg 4 is structurally blind on NinjaTrader.** `TCPTrader.GetOpenOrders` returns an empty
   slice unconditionally (`tcp_trader.go:1149-1151`), so `/api/open-orders` was `[]` at every cutover
   today (35-39) while, tonight, two limits rested at the broker. The `armed_orders` ledger is what
   caught the arm. Recommendation: the cutover rite's leg 4 reads `armed_orders WHERE state='working'`
   until the AddOn sends a real order-list frame. Class-33 candidate (the "fifth flat-gate leg").
2. **Class 37 telemetry, first live sightings:** three provider-side mid-body EOFs on one read
   (250,094 ms / 18,167 ms / 307,957 ms — 54,986 / 3,073 / 69,503 reasoning chars received), every one
   stamped `class=transport http_status=200 request_id=""`; the client retry fired each time
   (`⏳ Waiting 2s` → `retrying (2/2)` → identical request re-sent on the 1200 s stream); the
   `🛰 planner call FAILED class=transport provider_row=… elapsed=270.3s` line fired once. DeepSeek
   returned none of the four request-id headers probed. Its status page shows no incident for the window.
3. **Owner ruling M4 (transport failures retry the identical prompt at the PLANNER level) is not yet
   implemented:** attempt 2/3 at 23:47:33 went out as `reauthor+block` carrying the transport text as
   the "validator reason". Queued, not this wave.
4. The in-flight counter artifact above — fixed in the rite, not in code.
