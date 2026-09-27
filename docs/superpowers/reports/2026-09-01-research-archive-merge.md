# RESEARCH ARCHIVE MERGE — stranded reports onto dev

Date: 2026-09-01 · Docs-only · No code, no binary, no boot, no knob writes.
Branch: `docs/research-archive-merge` (commit `741bfc2a` + this report), merged to dev.

## Prerequisites

- replan_in_flight at wave start: plan-today payload returned `reading: None / replan_in_flight: None` (fields absent — the in-flight ASIA repair read had resolved).
- Main tree porcelain-clean; lock acquired (owner=hoang pid=1437095 expiry=2026-09-01 10:54:14 CT, `kill -0` ALIVE per the liveness amendment).
- Live rev `fef656a4ee7c…` PID 1625428 — unchanged at close (verified 4.3; no build, no swap, no restart).

## Per-branch table

Method: for each branch, `git log dev..<branch> --name-only -- docs/superpowers/reports` → file-copy via `git checkout <branch> -- <file>` (docs paths ONLY — code is never touched, so "entangled" commits are moot: only the docs path is ever checked out).

| Branch | Tip sha | Reports merged | Status |
|---|---|---|---|
| docs/brand-census | 23582b2a | 2026-08-20-brand-census-docs-brand-census.md | merged (COLLISION — dev already carried a brand-census.md; branch version suffixed) |
| docs/cheap-five | 9298f9d4 | 2026-08-30-cheap-five-knob-verdicts.md | merged |
| docs/confirm-cost-0830 | 8f09aa84 | 2026-08-30-confirm-cost-forensics.md | merged |
| docs/controls-runtime-verify | 1522cfa2 | 2026-08-19-controls-runtime-verify.md | merged |
| docs/decision-anatomy | 2d4a706e | 2026-08-19-decision-anatomy.md | merged |
| docs/deep-verify-22 | 95049c0c | 2026-08-27-deep-verify-22.md | merged |
| docs/dress-rehearsal-0830 | b54a9bfc | 2026-08-30-pre-livefire-verify.md (rehearsal version) | merged (shares path with 0830 — both kept) |
| docs/dryrun-replay | 4b0ba249 | 2026-08-29-dryrun-replay-scope.md | merged |
| docs/e2e-verify | 03957e58 | 2026-08-28-e2e-verification.md | merged |
| docs/final-verify | 91995ad4 | 2026-08-28-final-verify.md | merged |
| docs/forensics-zerotrade-2026-08-19 | 765ac11a | 2026-08-19-zerotrade-forensics.md | merged |
| docs/grand-audit | 104f0d3d | 2026-08-28-grand-audit.md + -bcde-verdict.md | merged (2 files) |
| docs/knob-census | 39a0481e | 2026-08-30-knob-census.md | merged |
| docs/level-system-verify | d6aa9669 | 2026-08-27-level-system-verify.md | merged |
| docs/london-drought | 607b8861 | 2026-08-26-london-drought.md | merged |
| docs/london-drought-2026-08-27 | f962d648 | 2026-08-27-london-drought.md | merged (distinct filename) |
| docs/london-forensics | a5595503 | 2026-08-28-london-forensics.md | merged |
| docs/master-recheck | 4b65eeeb | 2026-08-27-master-recheck.md | merged |
| docs/mega-research-mnq | 2cea2029 | 2026-08-27-mega-research-mnq.md | merged |
| docs/missed-200pt | 673e9240 | 2026-08-28-missed-200pt.md + grand-audit-response-wave-docs-missed-200pt.md | merged (the latter COLLISION — branch's variant of a file already on dev, suffixed) |
| docs/planner-latency-autopsy | 168e5282 | 2026-08-31-planner-latency-autopsy.md | merged |
| docs/pre-livefire-verify | 85095811 | 2026-08-29-pre-livefire-verify.md | merged |
| docs/pre-livefire-verify-0830 | a290920e | 2026-08-30-pre-livefire-verify-docs-pre-livefire-verify-0830.md | merged (COLLISION — suffixed; rehearsal keeps the plain name) |
| docs/refusal-autopsy | 589f7865 | 2026-08-27-refusal-autopsy.md | merged |
| docs/research-import-shift-forensics | d070c932 | 2026-08-21-shift-day-loss-forensics.md | merged |
| docs/settings-week-audit | dda9777c | 2026-08-26-settings-census.md + -week-in-review.md | merged (2 files) |
| docs/strategy-controls-census | e12e3846 | 2026-08-19-strategy-controls-census.md | merged |
| docs/total-audit-15 | 35c3aad9 | 2026-08-29-total-audit-15.md | merged |
| docs/weekend-audit-p1 | c18bd3a2 | 2026-08-29-weekend-audit-p1.md | merged |
| docs/weekend-audit-p2 | b964dc8e | 2026-08-29-weekend-audit-p2.md | merged |
| docs/zone-math-verification | 6919aa8b | 2026-08-29-zone-math-total-verification.md | merged |
| origin/docs/massive-move-audit | 151ef42b | 2026-08-30-massive-move-audit.md | merged (origin-only branch) |
| fix/clock-hold | 40f5ba36 | 2026-08-30-e7-resend-loop.md | merged (docs only; code already on dev via db6f510a) |
| hotfix/breakeven-dead | 7b687b78 | 2026-08-19-breakeven-audit.md | merged (docs only) |
| fix/ledger-close-sep-risk | cc34308e | 2026-08-19-ledger-close-FINAL.md | merged (docs only) |
| feat/weekly-bias | f9da39e1 | — | nothing to copy: all its docs already on dev (content merged via b84a96d6) |

Skipped: none. Entangled code/docs: none encountered (file-copy scopes to docs paths; no commit was cherry-picked).
Collisions handled (3): brand-census, grand-audit-response-wave (missed-200pt variant), pre-livefire-verify (0830 vs rehearsal) — all suffixed, nothing overwritten. The two dev originals that momentarily conflicted were restored byte-identical.

## Verification

- Report count on dev: **130 before → 168 after** (delta = 38 = number of merged reports). INDEX.md not counted in the 38.
- Nothing but docs moved: `git diff --stat a552d2c5 HEAD -- . ':(exclude)docs/'` → **EMPTY**.
- Live binary/PID unchanged: rev `fef656a4ee7c…` PID 1625428 (quoted pre and post).
- No report content edited; every merged file already began with a `#` header, so no provenance line was needed inside files — provenance lives in the INDEX (branch + sha per row).
- Origin branches left in place (nothing deleted).

## Index

`docs/superpowers/research/INDEX.md` — one row per artifact (58 rows incl. dev-merged research), sorted by date: date · title · path on dev · original branch+sha · verdict · action (shipped wave / queued / NONE) · contradicts-or-duplicates. Orphans from inventory §B, duplicates §D, contradictions §E carried across.
