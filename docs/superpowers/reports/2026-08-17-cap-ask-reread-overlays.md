# Six versions existed because the cap worked exactly as specified — v1–v5 were the five real plans a cap of 4 allows, and v6 is the NO-TRADE marker, which consumes a version number because the plans table is append-only

## ITEM 1 — replan cap / v6 semantics

**Semantics, settled.** The cap counts **re-plans**, not versions and not deaths. v1 is the session's first read and costs nothing; each later REAL version is one re-plan. So `replan_cap = N` ⇒ real versions v1…v(N+1), and the (N+1)th death writes a NO-TRADE marker. **No off-by-one.** The arithmetic against the real session (gate `used = version-1`):

| ver | used | used ≥ cap(4) | action | card "re-plans left" |
|---|---|---|---|---|
| v1 | 0 | no | re-plan → v2 | 4 |
| v2 | 1 | no | re-plan → v3 | 3 |
| v3 | 2 | no | re-plan → v4 | 2 |
| v4 | 3 | no | re-plan → v5 | 1 |
| v5 | 4 | **yes** | **NO-TRADE**, written as v6 | 0 |
| v6 | — | — | *this row IS the marker* | 0 |

**1c.** The NO-TRADE write **does** consume a version number and does **not** check the cap — it *is* the exhausted branch. A NO-TRADE row is legitimately **not** a re-plan.

**1d — THREE-WAY CHECK: NO DIVERGENCE.** DB `day_plan.sessions[ASIA].replan_cap = 4` (strategy level 2, override wins) · enforcer `at.replanCapFor("ASIA")` = 4 · card `ReplanCapFor(session) − (version−1)` · executor prompt `storedReplanCap`. All three read config; the CTO run's hardcoded `2-(version-1)` literal is gone (`8b24c85e`). One caveat worth keeping: the owner **raised ASIA 2→4 mid-session at 17:25:11**, which retroactively extended an in-flight loop — under a constant cap of 2 it would have ended at v4.

**Audit-query note.** `day_plan` is **top-level** in the config, **not** under `ai_config` where `risk_control` lives. My first query used the `ai_config` path and returned empty — the same trap that produced a false headline in the P0 risk report. I walked the JSON instead of guessing.

**Fixed:** one definition of the budget (`store.ReplansUsed / MayReplan / ReplansLeftFor`) that all three consumers now call; `/plan/today` returns the resolved `replan_cap`; and the marker is presented as one — chips render **⛔ NO-TRADE** instead of "v6" in both rows, with a banner reading "NO-TRADE — re-read budget exhausted … after **4 of 4** re-plans were spent". (Deriving that count from the version number would have said 5 of 5.) A source-scan test now **fails** if hand-rolled budget arithmetic reappears in `trader/api/kernel/store`.

## What the owner can now do that he couldn't

**Ask with no plan (ITEM 2).** The thread was hard-gated on an ACTIVE plan — 400 on night/disabled, 404 when no row existed — so it was locked at exactly the moment it is most useful. It now degrades through three labelled contexts: **active** (unchanged), **historical** (the most recent stored plan, even dead, carrying *why* that version was written, the plan's own reasoning, and its stated death condition), and **no-plan** (live market facts only). A NO-TRADE row for the live session resolves *historical*, never active. Patching outside an active plan is **code-enforced off** — the handler strips `reply.Patch` and downgrades PROPOSE-MERGE before storage, so Apply is structurally absent rather than present-and-erroring. `plan_qa` rows now carry `context_type`. On the FE the 💬 exists in the night and not-yet-armed states, where previously there was no affordance at all.

**Force a re-read (ITEM 3).** A ⟳ control that spends **one re-read from the same budget** the automatic path uses, so it cannot talk the bot past its own limits, and states the cost before spending because it is a real API call. Refusals are named, never silent: day-plan off · no active session · session not runnable · market closed · already NO-TRADE · "the re-read budget for NY is spent (4 of 4 used)". The server re-checks on POST rather than trusting a possibly-stale client gate. The new version is written through the normal path (retries, fail-closed, alerts all inherited) with `trigger_reason = "owner_reread"`. Screenshot of both states: `assets/reread-states.png`.

**Keep his edits across a re-plan (ITEM 4).** Owner levels are now **re-anchored by price identity, never by array index** — `/levels/3` means "the fourth element", and the fourth element of a re-planned doc is a different level, so replaying that patch would move an edit onto the *wrong* price, which is worse than losing it. The carry therefore works on the resolved level set, emitting `add /levels/-`. What cannot be re-anchored is surfaced for review, never dropped and never mis-applied: structural edits, a delete the planner has undone, and a price where the new plan has its own different level. The card shows "N edit(s) could not carry into this version — review", one line each, plus a P1.

**Clear the alert feed (ITEM 5).** Per-row ✕ and "Clear read", trader-scoped exactly like the ack (cross-trader → 404, no IDOR). An **unacknowledged P0 refuses** — a halt cannot be swiped away unseen — while an acknowledged one clears normally; the ✕ stays rendered and the refusal explains itself rather than leaving a missing button. **Audit, not amnesia: soft-delete.** Rows are flagged `dismissed` + `dismissed_at` and leave the feed and the bell badge, but nothing is deleted. `PruneAckedOlderThan` ships for acknowledged P2/digest noise (P0/P1 never auto-prune) — **no scheduler calls it yet**, so nothing prunes on its own until a caller is wired.

**See why it waited (ITEM 6).** Not a parser bug and nothing was lost in transit: `cot_trace` held the full text all along (1042 chars on the newest row). What was empty is the decision JSON's *own* `reasoning`, because the model writes its analysis in `<reasoning>` and leaves that field blank — most consistently on `wait`, the action the owner most wants explained — and the UI renders exactly that field. A decision with an empty reasoning now inherits the cycle's `<reasoning>`; a decision that stated its own keeps it verbatim. The backfill runs **before** validation so *rejected* decisions keep their reason too — the refusals panel is where "why was this refused?" gets asked. `DecisionCard` already renders `action.reasoning`, so it surfaces with no FE change.

## Exit bar

`go build` · `go vet` · `go test ./...` green · **`-race` clean** (kernel/trader/store/api) · `tsc` clean · `npm run build` OK · vitest **235/236** — the one failure and the `e2e/gate.spec.ts` collection error are the same pre-existing pair as the last five runs. **Goldens byte-identical**; no prompt path was touched (ITEM 6 is parsing, `DescribePlanDeath` and the carry are read-only/additive). **Config-truth on `replan_cap`, 4 steps:** save through the codec → row JSON `$.day_plan.replan_cap` = 2 → reload → the ENFORCING path reads ASIA=4 (override), NY=2, and `MayReplan(5, 4) = false`.

**One test replaced, deliberately:** `TestReplanOrphanWarningFiresOnlyWhenOverlaysExist` pinned the interim warn-only contract (a P1 counting edits about to be stranded). That was the honest stopgap while the rebase was unbuilt; ITEM 4 supersedes it, and it is replaced by a test asserting the owner's level actually lands on the new version by identity.

**Observation, not acted on:** ASIA/LONDON `min_grade` and `max_trades` currently read **B / 3** in the live config, not the A / 1 the earlier `dayplan-sessions` run applied. Most likely a later UI save rewrote the whole config block. Flagged only — changing risk settings was not in this dispatch.

**Session check:** the second Claude session (`554049f5`) stayed dormant throughout — last transcript write 18:36, ~2h before this run.

## Deploy

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
git rev-parse HEAD > deploy/RELEASE     # MANDATORY — else the boot assertion refuses trading
sudo systemctl restart nofx
cd web && npm run build && cd ..        # then HARD reload (Ctrl+Shift+R)
```

Visual check: the alert feed now has ✕ per row and "Clear read"; a ⟳ Re-read sits above the plan card; the 💬 works at night; and a decision row shows its reasoning on the next `wait`.
