# Adversarial re-verification — F4 weekly DOA breach-at-write guard

Verdict: **peer's non-conformance / DEAD finding CONFIRMED.** I could not refute it. Two
citation defects in their write-up are corrected below; neither changes the verdict.

## What I re-ran (all [A])

- `kernel/weekly_prompt.go:372-386` is exactly `ApplyWeeklyDOA` — line 372 `func`, line 386 `}`.
  `git diff 70af663d..HEAD -- kernel/weekly_prompt.go` is **empty**, so the deployed binary
  (rev 70af663d, boot 8, PID 878451) runs this exact body. `git log -1 -- kernel/weekly_prompt.go`
  = `882c2b7a 2026-09-02 21:38:05 -0500 fix(class50): weekly doc REFS ONLY …`.
- Caller search survived the method-level / indirect pass:
  - repo-wide `grep -rn ApplyWeeklyDOA . --exclude-dir=.git` → 9 hits: 6 test, 1 comment
    (`trader/auto_trader_weekly.go:326`), 2 the definition itself.
  - same grep **at the deployed rev** (`git grep -n ApplyWeeklyDOA 70af663d -- '*.go'`) → identical.
  - not a method, so no interface dispatch is possible; no func-value assignment
    (`grep '= ApplyWeekly\|(ApplyWeekly'` → none); no `go:linkname`; **one `go.mod`**, so no
    out-of-module importer of package `kernel` exists.
  - `WeeklyDoc.InvalidatedAt` has **zero production readers outside the dead function**
    (`kernel/weekly_prompt.go:187` struct tag, `:373`, `:384`). The `ScenarioInvalidatedAtKey`
    hits in `trader/invalidation_resolver.go:71` / `store/strategy.go:1340` are the scenario
    arm-gate subsystem, a different rule.
- Retirement is in the running binary: `git merge-base --is-ancestor 830717dd 70af663d` → **YES**.
- Their log quote is **verbatim real**: `/home/hoang/nofx/data/nofx_2026-09-02.log:50197`
  `09-02 19:06:24 [WARN] … 📅 WEEKLY READ 2026-08-31 stamped NEUTRAL AT WRITE (F5 DOA) —
  invalidation 29811.75 already crossed by a closed 1h bar`. The emitting code no longer exists
  (`git log -S "NEUTRAL AT WRITE" -- '*.go'` → added 59dc9460, removed 830717dd/654fd1da).
- DB corroborates the firing, and gives the n they omitted:
  `SELECT COUNT(*) FROM plans WHERE session='WEEKLY'` → **1** (n=1, the only weekly doc ever).
  That row: `bias='neutral'`, `invalidated_at='2026-09-02 19:06 CT'` (the `FormatCT` stamp written
  by `weekly_prompt.go:384`), `created_at=2026-09-03 00:06:24Z` = 19:06:24 CT — same second as the
  log line. No `shadow_bias` / `refs_only` fields, i.e. a pre-class-50 doc.

## Corrections (A17 — measured, not asserted)

1. **Wrong report:line (A9).** They ground F4 on `2026-09-02-belief-census.md:95`. Line 95 is
   **F2** — `| F2 | IPDA range 20/40/60 trailing days | kernel/weekly_bias.go:58-61 | [I] | advisory |`.
   F4 is at **:97**. Their adjacent F3 row cites `:94`, which is **F1**; F3 is at **:96**. Both are
   off by exactly two. This is not spec drift — the census is byte-identical to its pin ee64a494
   (`2026-09-02 08:50:38 -0500`). The same off-by-two is already published in their artifacts
   (`subsystem-F-weekly-reader.csv:4-5`, `.audit/f_weekly_reader_belief_census_f1_f5_disp.md:25-26`).
   Substance survives: :97 does say `[O] | gate`, so the "research value" is quoted, not invented.
2. **"2h31m before the class-50 boot" is measured to the COMMIT, not a boot.** 19:06:24 CT →
   `830717dd 2026-09-02 21:38:05 -0500` = 2h31m41s, which is the commit. The first *deployed* rev
   containing 830717dd booted at `09-02 22:37:38 🔐 BOOT INTEGRITY OK — rev 1cee77a87f1d`
   (nofx_2026-09-02.log:62012) — **3h31m14s** after the firing. Revs booted at 20:42 (575e9c05),
   21:19 (bb8b5419) and 21:32 (56904ec1) do **not** contain it.
3. **n was never quoted.** "Last live firing" rests on n=1 weekly doc row. Say so.

## Weakness inherited, not introduced

The `[O]` label's only provenance in the census is the string `59dc9460 F5 wave` in the "Where"
column — a commit hash, not an owner statement. The peer copied the census label faithfully; the
unverifiable owner-ruling is the census's defect. [B]
