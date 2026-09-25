# S-DISPATCH — HYGIENE + BIAS-TREE FACTS FIX (2026-08-27)

## 1. Merges + redeploy (hygiene)

- PR **#81** (planner contract wave) → dev: squash `8a35eeb0`.
- PR **#82** (side-quota relax) → dev: conflict in `deploy/RELEASE` / `kernel/levels_volume_boot.go` / `SessionPlanCard.tsx` after #81's squash — resolved (keep deployed rev `92bf01edd0`, keep BOTH boot lines, keep the thin-side block), branch re-pushed, merged: squash `a5f32f5f`.
- PR **#83** (this fix) → dev: squash `717acd34`.
- Regression at dev head: `go test ./...` 0 failures · FE tsc 0 · vitest 262/262 · `npm run build` green.
- **Redeploy from merged dev** (flat-checked: `GET /api/positions` → `[]` at 20:04 CT): rev **`717acd34e5`**, deploy marker `6a231a13`, SIGKILL restart, one PID.
- **Boot quote** (20:05:10 CT):
  `🔐 BOOT INTEGRITY OK — rev 717acd34e52b +dirty · built 2026-08-27T01:03:28Z · expected 717acd34e5 · goldens PASS`
  `📜 planner playbook (2026-08-26): playbook=v2 bias_tree=on chain_after=on no_trade_gates=on killzone_weights=on stop_doing=on — ALL ADVISORY, zero new gates`
  `⚖ side-quota (P0-relax 2026-08-27): min_side=cfg(default 2, MIN_SIDE_LEVELS env, 3 = old hard rule) · HTF+owner rows counted · machine-thin side=WARN+write(thin_side note) · 0-side/empty-map=fail-closed`
- **+dirty confirmation:** `git status` at deploy time shows EXACTLY one entry — the untracked `.env.bak.0825-2157`. No tracked modification. (Go's embedded `vcs.modified` counts untracked files, so `+dirty` is the backup file, nothing else.)
- The attempt-WARN lines (`📐 planner attempt N/3 …`) now ship in this binary (from #82).

## 2. BIAS-TREE FACTS — audit + fix

### (a) "no PDH/PDL anchor" on the 17:46/19:02 ASIA reads

**Trace.** `RenderBiasTree` (`kernel/planner_prompt.go`) read PDH/PDL/PDC only from `levels []ScoredLevel` — the SEATED table (top-N of the ±92pt in-band pool, `ScoreLevelsMinGrade` in `levels_assemble.go`). The day anchors are produced by `ExtractMultiDayLevels` (`levels_multiday.go:154-161`, prior CALENDAR day, coverage-guarded ≥900 bars) into the candidate universe — but seating drops out-of-band rows.

**The post-roll mechanism.** After the 17:00 CT session roll, the "prior calendar day" (08-25) anchors sit a full session-day back: PDH ≈ 30254 was **~640pt above price 29614** — outside the ±92pt band → never seated → `input.Levels` had no `KindPDH/KindPDL` rows → the tree rendered `PDH 0.00 · PDL 0.00` and no branch-1/2/3 match. Same root cause silently disabled the P0.2 gap-continuation rules (`facts.PDH/PDL = 0` = "unknown" in `runPlannerReadWithTriggerClaimedCtx`).

**Fix.** `BiasContext` now carries `PDH/PDL/PDC`; `ComputeBiasContext` scans the scored pool for all three; the write site stamps the FULL detector universe on top (`kernel.ExtractMultiDayLevels(bars, reg, now)` → new `kernel.ApplyUniverseDayAnchors`, seated values win). `RenderBiasTree` prefers the structured facts and falls back to the seated scan (legacy callers/tests). `facts.PDH/PDL` resolve from `input.BiasCtxFacts` when the seated loop missed them. Advisory only — zero new gates.

### (b) "376% of range" — wrong anchor

**Quoted range-selection code** (the old block, `planner_prompt.go`):

```go
	if bc.VAH > 0 && bc.VAL > 0 && price > 0 {
		lo, hi := bc.VAL, bc.VAH
		…
		pos := (price - lo) / (hi - lo)
```

Branch-5's premium/discount was computed against the **value area** (VAH–VAL). On the 19:02 read the VA profile was ~30pt wide while price sat ~114pt above its high → `(114/30) ≈ 376%`.

**Fix.** The anchor is now the **dealing range** — the prior-day swing hi/lo (PDH/PDL, which (a) makes always-resolve) — with the value area kept as the fallback when day anchors are unknown. Out-of-range prices are clamped and labelled:
`price BEYOND range high (extended) — longs disallowed by branch 5 (premium)` / `… BEYOND range low (extended) … (discount)`. No more >100% figures.

### Tests

- `TestBiasTreeDayAnchorsResolvePostRoll` — roll fixture: a seated table WITHOUT PDH/PDL + a universe WITH them → `ApplyUniverseDayAnchors` stamps → tree quotes `computed: PDH 30254.00 · PDL 29700.00 · PDC 30000.00`, `dealing range 29700.00–30254.00`, `BEYOND range low (extended)`, `facts match branch 1 mirror (close < PDL)` — anchors at ANY distance.
- `TestBiasTreeBeyondRangeHighClamped` — extended render never prints a >100% figure.
- `TestRenderBiasTreeBranches` updated: below-PDL now expects the extended label (deliberate behavior change); discount case re-anchored to below the dealing-range midpoint.

## 3. Proof — NY 08:25 bias-tree line

**Pending the scheduled NY read (2026-08-27 08:25 CT)** — the next read on this binary. Earlier opportunity: the LONDON scheduled read at 01:55 CT tonight. The ASIA v3 plan (written on the pre-fix binary at 19:02) quoted `bias-tree: branch 5 premium (price at 376% of range; longs disallowed)` — on the fixed binary the same market would render `bias-tree: … BEYOND range … (extended)` with real PDH/PDL anchors. Will be quoted when the NY read lands.
