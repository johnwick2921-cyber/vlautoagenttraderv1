# VL SYSTEM — DEFINITIVE VERIFICATION CHECKLIST
*Every item, how to verify it, what "done" looks like. No guessing.*
*v1 · 2026-08-16 · check items off as they pass*

---

## HOW TO USE THIS

Every row has three columns: **what**, **how to verify** (a command, a screen, or an observation), and **what "done" looks like**. If you can't tick a row, it isn't done — no matter what any report says.

Four states:
- ✅ **VERIFIED** — you personally saw the proof
- 🔵 **AGENT-VERIFIED** — a report claims it with receipts, you haven't seen it
- ⏳ **PENDING** — needs live market time to prove
- ⬜ **NOT DONE**

The rule that ends guessing: **a system is finished when every SECTION A and B row is ✅, and every SECTION C row is either ✅ or a dated decision to skip.**

---

## SECTION A — THE MACHINE IS WHAT YOU THINK IT IS
*(deployment integrity — the Knight Capital class)*

| # | What | How to verify | Done looks like |
|---|---|---|---|
| A1 | Running binary = intended code | `journalctl -u nofx --since "5 min ago" \| grep "BOOT INTEGRITY"` | `expected <sha> == rev <sha> · goldens PASS` |
| A2 | RELEASE armed after every deploy | `cat deploy/RELEASE` then `git rev-parse HEAD` | The two strings match |
| A3 | One process, no ghosts | `pgrep -fa nofx-bin` | Exactly one PID |
| A4 | Frontend bundle current | Hard-reload (Ctrl+Shift+R), check a recently-added control exists | The newest feature is visible |
| A5 | Wire protocol handshake | `journalctl -u nofx --since "5 min ago" \| grep hello` | `protocol_version=3 source=vltrader-addon` |
| A6 | Clock synced | `timedatectl` | `System clock synchronized: yes` · `NTP service: active` |
| A7 | Disk headroom | `df -h /` and `journalctl --disk-usage` | Free space comfortable; journal ≤ 2G cap |
| A8 | Backup fresh + restorable | `ls -lht ~/nofx-backups/auto/daily/ \| head -3` | A backup from today; RESTORE.md exists |
| A9 | Rollback rehearsed | Read the documented rollback steps once; confirm they include the RELEASE re-arm | You could do it under stress without thinking |

---

## SECTION B — THE SAFETY SPINE
*(these are non-negotiable; any ⬜ here means do not run)*

| # | What | How to verify | Done looks like |
|---|---|---|---|
| B1 | SIM lock unbreakable | CTO verification §5.2 receipts | No code path can route to a non-Sim account |
| B2 | Gate order proven | CTO verification §5.1 receipts | armor → conf → R:R → session → last-entry → blackout → plan → guardrails → size cap, in that order |
| B3 | Plan cannot bypass gates | §5.1 fixture: plan "demands" an entry violating each gate | Every one refused; plan-mode sits 8th of 9 |
| B4 | Owner overlay cannot bypass gates | Same fixture with an owner level | Refused identically |
| B5 | Size cap master-independent | Guardrails master OFF (your learning mode) yet cap holds | Max 2 contracts regardless |
| B6 | Hold-lock: AI can't exit by opinion | §5.3 fixtures | Only OCO / breakeven / EOD / session-flat / manual exit |
| B7 | Session flat bypasses hold-lock | §5.3 receipt | Routes directly through the trader close path |
| B8 | Fail-closed on every dependency | §5.7 enumeration | LLM, calendar, wire, bars, clock, disk → no-trade + alert, never a guess |
| B9 | Stale data can't reach a decision | B4 gate tests (3 pinned) | Stale opens become "wait" |
| B10 | Post-fill write failure is loud | R5 fixture | P0 alert + freeze, not an INFO log |
| B11 | Money math correct | Math audit: $2/pt, $0.50/tick, one real trade to the cent | 14/14 formulas verified, 3 oracles each |
| B12 | Loopback binds hold | `ss -lntp \| grep -E "8080\|3000"` | Both show `127.0.0.1`, not `0.0.0.0` |
| B13 | Destructive routes locked | Security P0 report | reset-account 401 · reset-password 410 · crypto/decrypt 401 |
| B14 | No secrets in logs or bundle | `grep -ri "sk-" data/*.log dist/ 2>/dev/null` | No hits |
| B15 | API keys rotated after the log leak | DeepSeek console | Old keys revoked, new keys in Studio |

---

## SECTION C — EVERY PIPELINE CONNECTS
*(the dead-wire class — the thing that kept biting)*

| # | Pipeline | How to verify | Done looks like |
|---|---|---|---|
| C1 | bars → detectors → scorer → KEY LEVELS → prompt | Query the latest decision's prompt for `KEY LEVELS` | Block present with real levels |
| C2 | bars → regime → planner input → plan → PLAN BLOCK → prompt | CTO §P-B trace | COMPLETE, no break |
| C3 | calendar → slice → planner input → T1 blackout → gate blocks | CTO §P-C + the FOMC Wed fixture | Entry inside the window refused |
| C4 | owner edit → overlay → resolved plan → prompt → W13 proposal → Apply → executor | CTO §P-D | COMPLETE |
| C5 | every config field → dual codec → row → hot-read → consumer | CTO §P-E, config-truth 4-step each | Every field reaches its consumer |
| C6 | fill → bracket/breakeven → exit → MAE/MFE + adherence + level-state → digest → next read | CTO §P-F | COMPLETE |
| C7 | registry → scheduler → entry gate → session flat → night mode | CTO §P-G | COMPLETE |
| C8 | alert emit → feed → P0 banner → ack | CTO §P-H + your own eye | You see an alert appear and clear it |

---

## SECTION D — EVERY CONTROL WORKS
*(check these yourself in the browser — 10 minutes, once)*

| # | Control | Test | Done looks like |
|---|---|---|---|
| D1 | Session toggles (all three rows) | Flip ASIA on → hard reload | Still on; chips show A🔸 and max_trades 1🔸 |
| D2 | Day Plan fields | Change proximity 1.5→1.6, save, reload, change back | Value persists both ways |
| D3 | ＋ Add level | Tap → sheet opens → save a level | Level appears with 👤 📝 and a version bump |
| D4 | Bulk add | Paste 3 lines → preview → save | All three appear; ONE realign call, not three |
| D5 | Level row tap → edit sheet | Tap an existing row | Sheet opens prefilled |
| D6 | Delete a level | Delete the test level | Gone; version bumps |
| D7 | ＋ new play (if built) | Create a scenario from a level | Play appears with direction and targets |
| D8 | Conflict chip | Add a level opposing an AI level | ⚡ chip; AI ghosted; you win |
| D9 | 💬 Ask-Planner: bare challenge | Ask "are you sure?" | Verdict = DEFEND (not a patch) |
| D10 | Ask-Planner: new info | Give a real HTF reason | CONCEDE or PROPOSE-MERGE with a patch |
| D11 | Decline a proposal | Tap "Keep as-is" | Says "You kept the plan as it was" — not "Applied" |
| D12 | Apply a proposal | Tap Apply | Card re-renders, version bumps |
| D13 | W13 auto-realign | Save any level edit | "⟳ planner reviewing…" then a proposal or no-change chip |
| D14 | Scenario status dots | Look at the card | Dots differ from each other (not all "armed") |
| D15 | "Refused this session" panel | Look after a refused entry | Panel lists the reason and blocking gate |
| D16 | Version chips (v1/v2) | Tap an old version | Shows that plan and why it died |
| D17 | Alert center | Trigger or wait for any alert | Bell badge → feed → ack clears it |
| D18 | Session tabs | Click ASIA / LONDON / NY | Card switches; disabled shows night |
| D19 | Executor indicators section | Toggle one indicator | Still fully working (no regression) |
| D20 | AUTO rows are read-only | Tap a planner AUTO row | Not interactive; tooltip explains |

---

## SECTION E — SETTINGS AT RESEARCHED VALUES
*(from the CTO verification's defaults table — confirm the LIVE DB column, not just the code default)*

| Setting | Should be | Verified |
|---|---|---|
| Proximity | 1.5 × dATR | ⬜ |
| Max levels | 8 | ⬜ |
| Max scenarios | 3 | ⬜ |
| Max re-plans | 2 per session | ⬜ |
| Acceptance | 2×5m | ⬜ |
| Approval required | OFF | ⬜ |
| Evening digest | ON | ⬜ |
| Plan mode | advisory | ⬜ |
| NY session | ON | ⬜ |
| ASIA / LONDON | your decision, min_grade A, Asia max_trades 1 | ⬜ |
| Last entry | 13:00 CT | ⬜ |
| EOD flat | 14:45 CT (single-sourced after the P3 fix) | ⬜ |
| Activation window | 1.5 × ATR | ⬜ |
| Re-arm cooldown | 20 min | ⬜ |
| Freshness floor | C | ⬜ |
| R:R gate | ≥ 3.0 | ⬜ |
| Confidence gate | ≥ 65 | ⬜ |
| Size cap | 2 contracts | ⬜ |
| Re-entry cooldown | 20 min | ⬜ |
| Stats gate | n≈1565, Bonferroni α≈0.006, weekly cadence | ⬜ |
| Digest chain | current sessions + 3 full dailies + days 4–7 one-liners | ⬜ |
| Planner model | exact pinned id (never an alias) | ⬜ |
| Guardrails master | OFF — **your dated learning-mode decision**, ON before real capital | ⬜ |

---

## SECTION F — LIVE PROOF
*(⏳ these can only be ticked by watching the market. This is where "finished building" becomes "finished proving.")*

| # | What | How you'll know | Status |
|---|---|---|---|
| F1 | A plan writes on schedule | A row appears with today's date at the session read time | ⏳ |
| F2 | Fail-closed works live | If a read ever fails: NO-TRADE plan + alert, never a stale plan | 🔵 (proven once by accident) |
| F3 | KEY LEVELS in a live decision | Query a decision's prompt after the open | ⏳ |
| F4 | The executor cites a scenario | A decision row names S1/S2/off-plan | ⏳ |
| F5 | An entry passes all gates and fills | Order → fill → bracket resting at the exchange | ⏳ |
| F6 | Breakeven moves automatically | Watch a winner reach +50% | ⏳ (needs ×10 to graduate) |
| F7 | Last-entry cutoff holds | No new entries after 13:00 CT | ⏳ |
| F8 | EOD flat executes | Position = 0 on both NT8 and DB at 14:45 | ⏳ |
| F9 | Session flat at a boundary | If Asia/London enabled: flat at 02:00 / 08:30 | ⏳ |
| F10 | A digest writes and feeds the next read | Evening digest exists; next plan references it | ⏳ |
| F11 | MAE/MFE + adherence grade recorded | A closed trade has both | ⏳ |
| F12 | Reconciliation holds | NT8 position = DB position, checked daily | ⏳ |
| F13 | A refused entry shows its reason | Refusal appears in the panel with the gate named | ⏳ |
| F14 | Guardrail trips correctly (when re-armed) | Halt fires at the limit, with an alert | ⏳ |
| F15 | Restart mid-session recovers | Restart once during a quiet session | ⏳ |

---

## SECTION G — TRUST EARNED (WEEKS, NOT DAYS)
*(the system isn't "finished" until these are underway — they cannot be rushed)*

| # | Milestone | Requirement |
|---|---|---|
| G1 | Blind-mark calibration done | ~10 historical days marked by you first; misses become detector rule fixes. Removes UNCALIBRATED |
| G2 | Detector warm-up complete | naked-POC reaches 10/10 sessions. Removes WARMING |
| G3 | Match-rate measured | ≥100 decisions with scenario citations |
| G4 | Stats gate progressing | Raw counts accumulating toward n≈1565 per level type; **no green verdict before then** |
| G5 | Regime coverage | Trend, range, news, and gap days all sampled |
| — | 100+ closed trades | The first point at which expectancy means anything (G6 loss-streak pause REMOVED 2026-08-23, owner override — commit 5126e57c) |
| G7 | Advisory → Direction | Only after G1, G3, and 2 clean weeks |
| G8 | Direction → Strict | Only after G4, G5, and 200–300 trades |
| G9 | SIM → real capital | 200–500 trades, multiple regimes, guardrails master ON, slippage measured |

---

## SECTION H — KNOWN OPEN ITEMS
*(these are tracked, not forgotten — tick when closed)*

| Item | Size | Status |
|---|---|---|
| AddOn watchdog livelock (4/7 regime fields dark) — needs C# + F5 + NT8 restart | M | ⬜ |
| Scenario anchor level missing from schema (dots sometimes blank) | S | ⬜ |
| Stale-data drift half needs TF-aware thresholds | M | ⬜ |
| Force re-read button | M | ⬜ |
| Plan history browser component | S–M | ✅ BUILT 2026-08-17 — `GET /plan/versions` + `?version=N` on `/plan/today`; chips are buttons that open the version they name, marked HISTORICAL with the death reason and a diff vs the successor (745f84f7, f630ceea) |
| Killzone gate (currently grading-only) | S–M | ⬜ |
| R3 panel unverified in a real browser (auth blocks Playwright) | — | verify by eye |
| approval_required has no grant UI | M | deliberately deferred |
| 7 low-severity security items (local-only behind loopback) | S | ⬜ |
| MAE/MFE visualization, per-scenario expectancy | M | v1.1 shelf |
| Telegram/external notifications | M | v1.1 shelf (your decision: in-app only) |

---

## THE FINISH LINE, STATED PLAINLY

**Building is finished when:** every row in A, B, C, D, E is ✅.

**Proving is finished when:** every row in F is ✅ and G1–G5 are complete
(G6 loss-streak pause REMOVED 2026-08-23, owner override — commit 5126e57c).

**Ready for real money when:** G7, G8, G9 are met, the guardrails master is ON, and Section H has no unresolved item marked as a safety concern.

Anything discovered after that should be a **feature request or a tuning decision** — not a dead wire. If a dead wire appears after Section C is fully ✅, it means a pipeline trace was wrong, and that trace gets re-run.

---

*Print this. Tick it with a pen. A system you've personally verified is worth more than a system someone told you was verified.*
