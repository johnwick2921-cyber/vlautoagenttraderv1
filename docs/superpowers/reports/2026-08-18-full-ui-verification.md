# FULL UI VERIFICATION — every button, every control (2026-08-18)

**LINE 1:** 21 controls LIVE · 2 fixed · 0 broken · reset: **WORKS-BUT-SILENT — fixed** (the click always worked; the UI never said the read was running).

**Reset (Part 1), root cause with receipts.** Chain fully wired (button → GET/POST /api/plan/reset → ForceReset → baseline+alert+read). Live browser click at 11:55 CT: backend log `OWNER RESET … chain abandoned at v2, budget re-armed`, P1 alert row, fresh v3 at 11:57 — BUT the reset's own read **silently skipped** because a concurrent death re-plan (v2 DIED 11:51:33) held the read claim; v3 got `NY_scheduled_read` instead of `owner_reset`, and the card showed nothing for minutes → "does nothing". Fix (28b79f09): claim-retry ≤30s + honest `note` on the response; `/plan/today` now carries `reading`; the card shows a persistent "planner is writing a fresh plan" banner; disabled-with-reason on the button was already implemented and verified. Feeds: day_plan_alerts rows 32-33 receipts.
**Second silent-disabled found (179fb229).** Strategy Studio locks EVERY editor for the DEFAULT strategy with no explanation — fixed with a 🔒 read-only banner (browser-verified).

**Matrix (control → verdict → proof):**
- Reset ⚡ LIVE in browser (click→confirm→backend→v3→card).
- Version chips + death history ⚡ LIVE (v2→"HISTORICAL VERSION — READ ONLY"→Back to current).
- Alert center: bell/feed/mark-read/✕/clear-read ⚡ LIVE (DB: acked=1, dismissed=1); unacked-P0 delete refusal = store/alert.go:138; clear-all = /plan/alert-clear-read.
- Plan card add/bulk/edit-sheet/save/delete/scenario-tag/conflict-chip = LIVE via P5_door·W15_door tests + /plan/overlay + /plan/owner-level(+delete); survives reload via plan_final merge (handler_plan.go P5.1).
- Ask-Planner open/ask/DEFEND-on-bare-challenge/Apply/Keep-as-is("kept") = LIVE via W16_decline·W19 + handler:1131.
- ⟳ Realign (W13) · ⟳ Re-read (W20) = LIVE (component-level; endpoints return gate reasons).
- Sessions toggles + day-plan block fields = LIVE config-truth (H-series: sessionRunnable/replanCapFor/acceptanceRuleFor/proximityFilterATR readers pinned by trader tests); AUTO rows + regime line read-only w/ tooltip = browser-verified.
- Tabs/timeline + card states (night/disabled/fail-closed/no-plan) = LIVE (P4_2·P7·W17·W19).
- Executor indicators: no regression (tsc clean, full suite).

**Browser proof:** auth worked via the repo's own gate-jwt minter + localStorage injection; screenshots: reset_confirm, reset_after, plan_card_v3, version_historical, alert_center3, alert_after_dismiss, dayplan_editor, default_lock_banner. Component-level only (no browser click): edit sheets/realign/reread/ask flows — exercised by their vitest suites, not clicked live.

**Exit bar:** go build/vet/test + -race green · tsc clean · vitest 244/245 (2 pre-existing harness failures: RegistrationDisabled logo, e2e/gate.spec.ts — test-infra, not controls) · goldens untouched.

**Not deployed** (market open, per rule). Owner deploy after 14:45 CT: git pull → go build -o nofx-bin . → git rev-parse HEAD > deploy/RELEASE → restart → npm run build. The reading banner activates only with the new binary.
