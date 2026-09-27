# C1–C6 measured at the running rev — 2026-09-09, session nofx-66

Running rev verified three ways: `/api/health` → `954f11b15f2e` · `/proc/438/exe` →
`/home/hoang/nofx/nofx-bin`, `vcs.revision=954f11b15f2e7615678f7d2b708c47895faebf1e`,
`vcs.modified=false` · `deploy/RELEASE` = `954f11b1`. All agree.

SPEC-FRESHNESS: the dispatch's pins `f5927cdc` (range-fade) and `982091d4` (trading-policy)
are the MERGE commits; `git log -1 -- <file>` shows the authoring commits `b2ba8c21` / `0ec5bd2c`.
`git diff <pin> HEAD -- <file>` is EMPTY for both — no drift, specs read as current.

| # | premise | verdict | measured |
|---|---|---|---|
| C1 | max_levels=12; 12/12 in 14 of 15 reads; 181 cut rows all score 0 | **PARTLY** | resolved `max_levels`=**12** ✓ (file const `DefaultMaxLevels=8`, `levels_score.go:54` — the boot line prints the const, not the resolver). Live reads show **both** `seated 12/…` (111×) **and** `seated 24/…` (34+32+16×). Cut reasons now: **581** `max_levels: 12 seated, cap 12` + **8** min-grade. The "181 all score 0" was the **09-04 sample**; per-day zeros: 09-04 181, 09-06 36, 09-07 48, 09-08 240, **09-09 84 rows, ALL real scores, zero zeros** — Stage A works, no A24 regression. Today's cap-cut rows include **grade A at score 1.344–1.36**. |
| C2 | 22% of seats >100 pts; touched 31% vs 79% | **NOT REPRODUCED — worse** | today n=84 seats (ids 1009–1176): ≤25pt **14** · 25–50 **20** · 50–100 **25** · **>100pt 25 = 29.8%** (avg 150.1 pt). `touch_outcomes` n=3931 (ids 1–3931): hold 1627 (41.4%) · break 1477 (37.6%) · ambiguous 827 (21.0%). |
| C3 | confluence counts names, not independent evidence | **CONFIRMED + SHARPENED** | `conf` counts **distinct families** within `confBand` (`levels_score.go:462-476`); same-family already excluded (`seenFamilies`). Credit `(1 + 0.20*effConf)`, `ConfluenceCap()`=3 (`:194-200`), applied at `:502`/`:508`. **`confBand := 0.10 * dATR` (`:427`) ≈ ±20.3 pt today, while the cluster COLLAPSE width is fixed 3.00 pt (`LevelClusterTicks=12`, `:717`, used `:570`) — a ~6.8× mismatch, and BOTH are commented "cluster tolerance".** The collapse (`:751-779`) runs **after** scoring and its own override note says `"cluster confluence display %d -> %d; score unchanged"`. |
| C4 | OB 8/117 (6.8%), SUPPLY 5/23, DEMAND 22/59, EQH 0/4 | **NOT REPRODUCED** | **OB 34/187 = 18.2%** · SUPPLY 12/36 = 33.3% · DEMAND 30/75 = 40.0% · **EQH 17/146 = 11.6%** (not 0/4). Tier-1 anchors seat 100%: ONH 34/34, ONL 40/40, PDL 35/35, PDC 33/33; VWAP 64/68 = 94.1%. |
| C5 | nothing exists beyond the map | **CONFIRMED** | grep counts (kernel/+trader/, excl. tests): measured-move **0** · ATR-projected extreme **0** · prior-week 8 · round-number 15 · "projection" 2. No projector exists. |
| C6 | PWH/PWL can never seat (4,320-bar guard on a 33h ring) | **CONFIRMED + SHARPENED** | `priorWeekMinBars = 4320` (`levels_multiday.go:224`), gate at `:198` via `pwCovered`. **The const block's own comment (`:219-221`) says: "with the ~33h 1m ring these can never pass, which is CORRECT: those anchors must come from a multi-day source, not the ring."** `candidate_pool` rows for PWH/PWL/PMH/PML: **0, ever**. A daily source EXISTS: `bars` tf=`1d` n=1901 through 2026-09-08. |

---

## C1 addendum — the boot-line A11 literal is ALREADY FIXED AND DEPLOYED (correction to my own finding)

The owner ruled: *"the boot line printing the file const 8 while max_levels resolves to 12 is an A11
literal — fix it in this wave."* **It needs no fix: it was fixed on 2026-09-04 and is in the running
binary.** My earlier claim rested on a stale log line and is withdrawn.

```
commit 0e016635  2026-09-04 09:32:21 -0500
        fix(prompt/boot): read the floor, read the seats, say the refusal, derive the set
git show 954f11b1:kernel/levels_volume_boot.go   → contains the FIXED VolumeWaveBootLine
```

Current code (`kernel/levels_volume_boot.go:53-61`), whose own doc comment names the exact defect:

> *"Every number is READ from a constant or resolver (A24): it used to print seats=8 from a package
> default while the bound strategy's max_levels was 12 … Per-trader values are LABELLED rather than
> printed as if the boot process knew them."*

Live line from **today's** boot of the running rev:

```
09-09 13:02:06 kernel/levels_volume_boot.go:15
🎛 volume wave: detectors=on · seats=per-trader (default 8, hard cap 12) · proximity=per-trader
   (default 1.5×dATR) · family-confluence(cap=3) · role-overridden=false
```

`seats=8` appears **only** in `nofx_2026-09-03.log` — a pre-fix boot. 09-04, 09-08 and 09-09 all read
`seats=per-trader`. My earlier grep spanned `nofx_2026-09-0*.log` and quoted the 09-03 line as current.

`DefaultMaxLevels = 8` (`levels_score.go:54`) and `PlanHardMaxLevels = 12` (`plan_doc.go:365`) are a
package default and a hard ceiling; the bound strategy's resolved `max_levels` is 12. The line now
labels the per-trader value instead of asserting one. **W3 changes nothing here** — touching it would
be claiming another lane's shipped work (A24).

## C1 addendum — "seated 24/…" vs "seated 12/…", quoted

```go
// kernel/levels_score.go:646  (inside ScoreLevelsMinGradeFull)
pool := scoreLevelsPool(levels, price, dATR, freshness, eff*2, proximityK)
```

`eff = maxLevels` (12), so the pool is scored at **`eff*2` = 24**. Both passes emit the same line at
`levels_score.go:611`. So **24 is the 2× PRE-SEAT POOL, 12 is the seated table — not a second cap.**
The pool's purpose is stated at `:631-636`: a level the model copies from the prompt that LOST the
seat race must still get its machine grade stamped, so the stamp map records every graded candidate.
