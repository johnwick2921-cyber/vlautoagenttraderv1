# Guide Docs-Refresh Wave (content-only, 2026-08-28)

- **Branch:** `docs/guide-refresh` (commit `fa2fd9b0`) → merged to dev (`72c8bfd8`). Content-only: 6 guide files + 1 constant + FAQ tagline. Zero engine/logic changes. Rides the next binary build — no dedicated cutover.
- **Gates:** vitest `GuidePage.test.tsx` **10/10** · `tsc --noEmit` **0 errors** · `go build ./...` **OK**.

## Changes (per the audit list)

1. `types.ts` — `GUIDE_BUILT_REV '717acd34' → '8666db0b'` (drift banner clears for all 12 sections).
2. `plays.ts` — tagline 7→8 · `breakdown_continue` card (two-leg confirm, displacement ≥1.0×ATR5m, pullback/immediate modes, born from the −347pt NY crash: v4 active, S2 declined on the retest rule, no plan-legal continuation short) · `breakup_continue` long mirror · A2c FVG-demand line on `fvg_entry`.
3. `planCard.ts` — scenario-row condition list → 8 conditions + two-leg confirm render wording ("leg 1/2 MET · leg 2/2 NOT MET → overall NOT MET") · new item 10 armed chips (⏳ armed / 📌 working / ⚡ filled / ✕ cancelled) · item 11 dormant 😴 + auto-rearm (0.5×ATR14, 2 closes).
4. `guards.ts` — HTF-veto row now documents MODE (1h|cross|4h, LIVE=cross, the $352/0 one-liner) · new ARM-floors row (2.0 gate-at-arm vs 3.0 market-entry and why they differ).
5. `settings.ts` — proximity ⭐ live tense (0.3 since 08-28 11:59, ×DailyRangeProxy ≈±85pt) · min-conf Sep-9 note (65 deferred, 60–64 judged at full n) · scenario cap 3→5 (live) · min-side 2→4 (live) · plan-mode ⭐ STRICT ×3 (owner ruling) · session rows → strict ×3 / 7-10-10 / replan 4 / min_side 4 · new "Env-only knobs" callout listing all 9 (ARM_MIN_RR 2.0 · HTF_VETO_MODE cross · HTF_VETO_TF 1h · FAST_MARKET_ATR 1.5 · FAST_MARKET_REASONING fast · BD_MIN_DISP_ATR 1.0 · FVG_ENTRY_MIN_DISP_ATR 1.5 · INGEST_QUEUE_CAP 1024 · AI_PLAN_MAX_TOKENS 65536).
6. `faq.ts` — +2 Qs (why cross mode; what happens on a fast-market wake) · tagline 12→14.

**Canon added:** GUIDE CONTENT LAW — any wave changing a knob/play/chip/gate/default must update `web/src/guide/content/*` in the same PR with the `GUIDE_BUILT_REV` bump (recorded in `CLAUDE.md` + repo memory). The drift banner is a failsafe, not a maintenance strategy.

**Gate gotcha:** two rewritten ⭐ lines initially contained the word "default", tripping the lint test's unique `/default/i` match inside knob cards — reworded to "code fallback" (content-only fix, no test change).
