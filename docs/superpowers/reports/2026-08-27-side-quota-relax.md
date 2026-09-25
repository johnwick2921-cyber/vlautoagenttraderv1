# P0-VALIDATOR AUDIT-THEN-RELAX — side-quota (≥3 per side) (2026-08-27)

Branch `fix/side-quota-relax` · base: running `e49c82e5` (feat/planner-playbook HEAD; note: `origin/dev` is still `b02461cf` and PR #81 is unmerged — the branch was cut from the RUNNING rev so the cutover rebuild matches the deployed tree).

Owner ruling: the ≥3-levels-per-side rule fail-closed the entire ASIA session on 2026-08-26 over a machine-map shortage. Relax per phases. AUDIT FIRST.

---

## PHASE 1 — AUDIT (read-only, before any edit)

### 1.1 The validator, verbatim + birth

`kernel/plan_doc.go:471-481` (inside `ValidatePlanDocWithFacts`):

```go
	// P0.1 — both-side minimum.
	below, above := 0, 0
	for _, l := range d.Levels {
		switch {
		case l.Price < facts.Price:
			below++
		case l.Price > facts.Price:
			above++
		}
	}
	if below < MinSideLevels {
		return fmt.Errorf("only %d levels below price %.2f — the plan must carry ≥%d on EACH side (add prior week/month lows, swing lows, round numbers or value-area edges below)", below, facts.Price, MinSideLevels)
	}
	if above < MinSideLevels {
		return fmt.Errorf("only %d levels above price %.2f — the plan must carry ≥%d on EACH side", above, facts.Price, MinSideLevels)
	}
```

The quota constant: `kernel/levels_score.go:720-722` — `const MinSideLevels = 3`.

**Birth:** `5408abd1` "P0 planner-quality: facts-validated plans — both-side levels, gap continuation, reachable targets (items 1c/2)", 2026-08-18 15:12 CT.
**The incident it protects against** (report `docs/superpowers/reports/2026-08-19-planner-quality-fix.md`): the **2026-08-18 NY one-sided map** — today-priority kinds (PDH/PDL/PDC/RTH/OR/ONH/ONL) sorted FIRST and filled all 8 seats; on a gap-down day they all sat above price (29680.75–30092 at price 29687). Stored plan = **1 below / 7 above**; the market broke down 110 points with zero downside levels in the plan, and the lone short (S3) required a 166-pt rally to trigger. The rule's job: a plan nobody can trade on one side.

### 1.2 The full fail-close validator chain (write site `runPlannerReadCoreWithFactsGrades`, trader/auto_trader_planner.go:853-1000 — ≤3 attempts each)

| # | Validator | Rejects | Mode | File:line |
|---|---|---|---|---|
| 1 | LLM `call()` error | transport/HTTP/402/timeout | retry ×2 | planner.go:860-864 |
| 2 | `ParsePlanDocCapped` → `ValidatePlanDocWithCaps` | no JSON / bad JSON / reasoning-empty / bias enums / conviction enum / death_condition missing / level count>cap / grade not A·B·C / price≤0 / scenario count 0 or >cap / S-id format / condition enum / direction enum / quality enum / target_chain≤0 / confirm{} rule·side·ref_price + prose match / death·flip structured enums + prose match | retry | plan_doc.go:234-300, 306-367 |
| 3 | P0.4 duplicates | two levels within cluster tolerance (2.13pt apart killed ASIA 08-24) | retry | plan_doc.go:463-471 |
| 4 | P0.4-G flip mandate | re-plan bias ≠ the prior flip's fired direction | retry | planner.go:891-896 |
| 5 | P0.4-H label provenance | plan level relabeled vs machine table's structural label (phantom PDH) | retry | planner.go:899-904, plan_doc.go:402-435 |
| 6 | **P0.1 side quota** | **plan carries < MinSideLevels(3) on either side of price** | retry → fail-close | **plan_doc.go:471-481 ← RELAXING THIS** |
| 7 | P0.2 gap continuation | price < PDL without a short / > PDH without a long | retry | plan_doc.go:483-492 |
| 8 | P0.2-c continuation reachable | gap-day continuation trigger on the wrong side of price (166-pt-rally S3) | retry | plan_doc.go:497-507 |
| 9 | P0.2b targets-in-band | any scenario target beyond 1.5×dATR of price | retry | plan_doc.go:509-521 |
| 10 | FVG entry model | 3-candle relation / gap floor / displacement / origin membership re-verified from stored bars | retry | planner.go:917-928 |
| 11 | C1 confirm{} grace | scenarios missing confirm{} after CONFIRM_GRACE_SESSIONS | reject (no retry) | planner.go:966-975 |
| — | attempts exhausted | fail-closed NO-TRADE write (wake reads: benign keep-active) | fail-close | planner.go:1010-1030 |

**Blast-radius statement:** exactly rule #6 changes. Rules #1-5, 7-11 are untouched. The seating-side rebalance (`seatBothSides`, levels_score.go:993-1014) is untouched — it already degrades gracefully ("when the in-band universe can supply them").

### 1.3 What the quota actually counts

`facts.PlanFacts` carries only `Price / DATR / PDH / PDL` (plan_doc.go:436-441). The loop at plan_doc.go:472-476 counts **`d.Levels` only** — the levels the AI wrote into the plan. It gives zero credit to:
- **HTF-section rows** (`PlannerInput.HTFZones`, planner.go:1390) — prompt-visible, merged into `machineLabels` purely for the grade-stamp + fvg-origin check (planner.go:757-764), never counted by the quota.
- **Owner sticky levels** — prepended into the seated list (planner.go:1258-1272, Kind `KindOwner`, grade A) so they ARE in `input.Levels`/the prompt, but the quota still only credits them if the AI copies them into `d.Levels`.

**Hypothesis CONFIRMED:** the validator can reject a plan whose side-shortage is a machine-map shortage, because it only sees the AI's copy, not the universe the AI was allowed to copy from. The full prompt-visible universe is available at the write site as `machineLabels` (price → label; seated incl. owner-sticky + HTF rows merged — planner.go:725-764).

### 1.4 History — has this rule bitten before?

Grep of every `data/nofx_*.log` (2026-08-16 → 08-26) for the rule's error strings and every `FAIL-CLOSED` line:

- **Side-quota fail-closes: exactly ONE, all time** — tonight, 2026-08-26 18:01:15 ASIA ("only 2 levels above price 29614.00"). It was the first.
- All-time fail-close inventory (13 `planner_fail_closed` plan rows): 08-17 ASIA no-JSON · 08-18 ASIA ×3 scenarios-count-0 · 08-23 ASIA ×3 quality-"C"-enum (fixed b2753f2d) · 08-24 LONDON quality-C · 08-24 ASIA duplicates-2.13pt · 08-25 ASIA ×2 context-deadline (timeout, fixed by env) · **08-26 ASIA side-quota (tonight)**.
- Tonight it also ate both in-flight reads (owner reset 17:46:20 + level wake 17:47:27) — 6 LLM attempts in ~15 min, all rejected on the same above-side count.

Verdict: the rule has NOT been quietly eating reads — tonight was its first kill, and it killed a whole session.

### 1.5 Did the ±92pt band (S1 retune) cause it?

Recompute against the persisted detector universe (`level_state`, 204 distinct prices):

| band around 29614 | above | below |
|---|---|---|
| OLD ±530pt (pre-S1) | **35** | 144 |
| NEW ±92pt (current) | **9** | 26 |

- At the old band the above-side was never short (35 candidates). The S1 retune took the above-side from 35 → 9 candidates — it SET the conditions.
- What finished it: price 29614.00 sat at the **top of the day's level stack** — the old plan's own "above" targets (weekly nPOC 29553.12 / 29596.88, OB 29611) had all fallen BELOW price by read time. The remaining above rows are low-grade RNs (29625, 29650) and a just-formed ONH (29628.75).
- The fail-time machine map (`plans` row ASIA v2 `no_trade` doc) carries 5 below / **3 above** — so by 18:01 the universe could again supply 3. The AI-vs-machine split at 17:46 read time is NOT forensically decidable from persisted data (rejected attempt drafts are not stored).

**Stop-check:** nothing above contradicts the fix design. The design handles both branches deterministically at write time by comparing the plan against the SAME `machineLabels` universe the prompt displayed. Proceeding to Phase 2.

---

## PHASE 2 — THE FIX

- **2.1 Count HTF + owner toward the quota.** The write site already merges the seated table (owner-sticky levels prepended, grade A) + HTF-section rows into `machineLabels` (price→label, planner.go:725-764). The new validator receives that same map — HTF and owner rows now count toward the machine side-counts that decide machine-caused vs AI-caused.
- **2.2 Machine-caused shortage → WARN + write.** `ValidatePlanDocWithFactsMachine` (`kernel/plan_doc.go`): when the plan carries < quota on a side AND the machine map itself has < quota on that side, the plan writes with `thin_side: "N above/below (machine map N)"` stamped onto the doc; the write site logs `📉 side-quota relaxed: …`. The card renders the note as an amber ⚖ chip (`SessionPlanCard`, testid `thin-side-note`).
- **2.3 Hard fail-close kept for the safety floor only:** 0 levels on a side (the 2026-08-18 one-sided-map pathology) and an empty machine map. Both still fail-closed.
- **2.4 AI-caused omission stays rejected:** map had ≥ quota but the plan carries fewer → error `"AI dropped levels the map supplied"` → retry → fail-closed.
- **2.5 Knob MIN_SIDE_LEVELS:** `kernel.SideQuotaFromEnv()` (env, default `DefaultSideQuota=2`) + `DayPlanConfig.min_side_levels` (Strategy Studio, base + per-session override, resolver `MinSideLevelsFor`). Precedence: config → env → 2. Old behavior = set 3. No literals (all named constants). JSON round-trip test covers both PUT paths (full config save + session override).

## PHASE 3 — TESTS

- `kernel/side_quota_relax_test.go` (8 tests): machine-thin WARN+proceed (above) · machine-thin BELOW (long/short symmetry) · AI-caused omission reject · zero-side fail (both sides) · empty-map fail · nil-map legacy hard behavior (quota 3) · **HTF+owner rows counted** (the reject proves the merged map counts them) · knob 2 vs 3 (2/2 map at quota 2 passes clean; quota 3 on a 2/2 map WARNs; quota 3 with a full map and a 2/2 plan rejects) · `SideQuotaFromEnv` parse matrix (unset/3/garbage/0).
- `store/side_quota_test.go`: `MinSideLevelsFor` resolution seam (default 2 · strategy 3 · session override · override-0 → unset · nil config) + JSON round-trip of base + session override (both PUT paths).
- `trader/auto_trader_planner_read_test.go`: write-site `TestRunPlannerReadMachineThinWritesWithNote` (machine-thin writes v1 active with `thin_side` stamped, trigger `owner_reset`) + `TestRunPlannerReadAIOmissionFailsClosed` (map had 3 above, plan carried 1 → fail-closed).
- FE: DayPlanEditor test — knob renders default 2 and writes `min_side_levels: 3` on change.
- Full regression: `go test ./...` 0 failures · tsc 0 errors · vitest 262/262 · `npm run build` green. Goldens untouched (validator is not golden-covered; futures executor goldens unaffected).

## PHASE 4 — CUTOVER + REVIVAL

- Flat check 18:38 CT (`GET /api/positions` → `[]`), one PID, ASIA plan was the fail-closed NO-TRADE v2 (sitting out — a restart loses nothing). Owner-ordered cutover executed inside the live ASIA window per dispatch.
- Deployed `92bf01edd0` (buildRevision embedded), `deploy/RELEASE` marker `0b0986f4`, SIGKILL restart, boot 18:38:38 CT:
  - `🔐 BOOT INTEGRITY OK — rev 92bf01edd043 +dirty · goldens PASS`
  - `⚖ side-quota (P0-relax 2026-08-27): min_side=cfg(default 2, MIN_SIDE_LEVELS env, 3 = old hard rule) · HTF+owner rows counted · machine-thin side=WARN+write(thin_side note) · 0-side/empty-map=fail-closed`
  - health: `revision 92bf01edd043` · one PID.

### REVIVAL — owner reset 18:38:57 CT → ASIA v3 active 19:02:08 CT

- Read took 3 attempts (~8.8 min + 7.4 min + 6.9 min of deepseek reasoning=max; curl's 15-min max died first at 18:53:57 — the read continued server-side, per design).
- **`PLAN written 2026-08-26 ASIA v3 (model deepseek-v4-pro, lifecycle active, trigger owner_reset)`.**
- **BIAS-TREE branch quoted verbatim** (A1 contract followed):
  `bias-tree: branch 5 premium (price at 376% of range; longs disallowed) — no PDH/PDL anchor computed, so no branch 1/2/3 call. 1h/15m are trending up but price is extended: ~260 pts above 15m EMA50, 15m RSI 76, …`
- **Level table (8 rows, quota 2 satisfied both sides):** above — 29611.00 OB(bull)·1h C, 29621.00 EQH·1h (HTF) A, 29628.75 ONH A, 29638.00 Demand·1h (HTF) A, 29654.00 Demand·4h (HTF) A · below — 29585.99 VWAP A, 29563.12 VWAP−1σ A, 29541.12 Demand·1h C. **HTF-section rows seated as real plan levels** — the 2.1 counting path visible in the wild.
- **thin_side note: correctly ABSENT** — the overnight map had regenerated 5 levels above price by read time, so no machine-thin WARN fired (the relaxed path simply wasn't needed tonight).
- Scenarios: S1 sweep_reclaim / S2 acceptance / S3 reject — all short on the premium branch; no fvg_entry → no chain_after expected.
- Session revived: the fail-closed NO-TRADE v2 is superseded by an ACTIVE v3.

### Follow-up (committed, NOT yet deployed — folds into the next cutover)

Attempts 1–2 of the revival read were rejected SILENTLY (the retry loop never logged per-attempt rejections). Added `📐 planner attempt N/3 …` WARN lines for every `continue` in the write loop (call error, parse/schema, flip mandate, label provenance, facts/side-quota, fvg) — zero behavior change, log-only observability.
