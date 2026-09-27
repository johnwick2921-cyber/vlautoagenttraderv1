# 11 DEAD CONTROLS FOUND — 11 FIXED, 0 LEFT DEAD (6 missing *features* sized, not built)

122 controls walked (6 auditors + my own re-verification of every claim). Commits `bc360a38` · `d396201e` · `e943f9c3`. All evidence **[A]** = read the file / ran it.

## THE REPORTED BUG — two defects, not one

The session rows had no enable toggle, and it would not have worked if they had: the read scheduler and the entry gate ANDed the **hardcoded** `DefaultSessionRegistry` flag (ASIA/LONDON `Enabled=false`), so a strategy-level "turn ASIA on" was vetoed forever by a compiled-in constant. Fixed with one resolver — `sessionRunnable()` — used by BOTH gates: explicit `sessions[].enable` wins 🔸, else inherit registry ∧ `sessions_enabled` ⚪. The registry still owns the CLOCK; only enablement became overridable. Enabling a session does **not** backfill a past read (the read window still decides *when*).

## CONTROL MATRIX — every defect (verdict → file:line → what it does now)

| # | Control | Was | File | Fix |
|---|---|---|---|---|
| 1 | `sessions[].enable` ×3 rows | **MISSING + dead on arrival** | `DayPlanEditor.tsx:560` · `auto_trader_planconfig.go:104` | toggle + resolver; gates honor it at the next read |
| 2 | `sessions[].acceptance_rule` | **DISPLAY-ONLY** (0 Go readers) | `store/strategy.go:918` | all 3 consumers resolve per-session (level-state, plan-death, executor prompt) |
| 3 | `sessions[].max_trades` | **DISPLAY-ONLY** (0 Go readers) | `auto_trader_session.go:77` | per-session entry cap, counted from *this* session instance (wrap-aware for ASIA) |
| 4 | `last_entry_ct` | **MISSING** control (Go read it) | `DayPlanEditor.tsx:498` | time input; blocks late entries |
| 5 | `eod_flat_ct` | **MISSING** control (Go read it) | `DayPlanEditor.tsx:506` | time input + ⚠ when set past NY's window end (the P3 drift band) |
| 6 | `realign_cap` | **MISSING** control *and* FE type | `DayPlanEditor.tsx:523` | numeric 0–10 |
| 7–9 | plan card: acceptance rule / mode / re-plan budget | **HARDCODED** `"2x5m"`, `"advisory"`, `2-(v-1)` at 5 sites | `api/handler_plan.go:60` | one `planRules()` resolver; card now narrates the rulebook the gates run |
| 10 | session tabs | **DISPLAY-ONLY** — no session in the request | `handler_plan.go:82` `resolveRequestedSession` | tab drives the fetch; `is_active` marks live vs sibling |
| 11 | tab enablement | **HARDCODED** FE `SESSION_BANDS` | `PlanCard.tsx:106` `computeSessionTabs` | server-told `runnable_sessions`, same resolver as the gates |

Hardcoded vs configured: items 7–9 and 11 were pure hardcodes — a strategy on `15m-close`/`strict`/`replan_cap=4` ran the real values in the executor while the card showed defaults, including `SessionPlanCard` gating its advisory affordances on that literal.

**Two auditor verdicts I disproved:** "Keep as is" and the ＋new-play chip are **LIVE** — the realign proposal is local state from an explicit call (nothing re-delivers it), and `scenario_tag` reaches the DB, the card, and the re-align prompt.

## NOT BUILT — missing *features*, with sizes (no dead controls remain)

Plan-history view **S–M** (endpoint + api client exist, no component) · force re-read button **M** (needs an endpoint + cost guard) · Day-Plan AUTO rows: per-TF structure, overnight auction, econ calendar **M–L** · per-session window/read/flat times on the row **S** · `external_data_sources` control **S** (outside day-plan scope) · `sessions_enabled` multiselect **deliberately skipped** — the per-session toggle supersedes it.

## VERIFY

Config-truth 4-step on all 6 new fields — they survive **both** halves of the hand-rolled codec (`day_plan.sessions[].enable` etc.) and reach their consumers. **8 new Go tests + 9 new vitest**; full Go suite green; FE 177/178 — the single failure (`RegistrationDisabled` "NoFx Logo" alt text) and the `e2e/gate.spec.ts` collection error are **pre-existing**, untouched here. Goldens unchanged (no prompt-shape edit). Playwright, after a **hard reload**: all three toggles render with accessible names, ASIA persists on with the 🔸 chip, LONDON stays off, `last_entry=12:15`, `eod_flat=15:30` with the drift warning firing — `shots/2026-08-17-w15-controls-after-reload.png`. (The 🔸 renders as tofu in headless Chrome; same glyph the shipped `OverrideRow` uses.)

Defaults are unchanged everywhere: no override → the shipped value, no cap → no block, no `?session=` → the live session.

## OWNER — DEPLOY

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && cd web && npm run build && cd ..
kill -9 $(pgrep -f '^\./nofx-bin$')        # systemd Restart=on-failure relaunches
git rev-parse HEAD > /tmp/rel && { cat deploy/RELEASE | grep '^#'; cat /tmp/rel; } > deploy/RELEASE.new && mv deploy/RELEASE.new deploy/RELEASE
journalctl -u nofx --since '2 min ago' | grep 'BOOT INTEGRITY'
```

Re-arming `deploy/RELEASE` **after** deploying is required — the boot assertion refuses trading on a mismatch. Then flip the ASIA/LONDON toggles only if you actually want those sessions; they stay off otherwise.
