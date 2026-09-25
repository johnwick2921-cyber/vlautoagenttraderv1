# ALL 7 SHIPPED — the dashboard now reports what the bot actually did

W16, from the CTO final verification (`04475d66`). The gates were already right; the **reporting** lied. Seven items, one commit each, no change to any gate threshold or order-routing path. Exit bar green; goldens byte-identical.

| # | Commit | What was lying | What the owner now sees |
|---|---|---|---|
| R1 | `c4996495` | `scenario_status` had ONE writer in the repo — the sandbox seeder — so every play painted **"armed"** forever | Real per-scenario dots: waiting ◌ · armed ○ · triggered ● · invalidated ✕ · expired ○ |
| R2 | `0b8443f9` | Declining a proposal rendered **"Applied — card updated"** and persisted nothing | "You kept the plan as it was. The decision was recorded; nothing was changed." + a KPI row |
| R3 | `9e7b3259` | Gate-block counters had **zero** frontend consumers; `feed_down` still recorded refusals as successes | A "Refused this session" panel + the gate name on the decision row itself |
| R4 | `0504516f` | The session digest was keyed without trader scope but held **one** trader's P&L | Each trader's digest is its own; the other's day can't seed its planner |
| R5 | `4381c801` | A post-fill DB write failure logged at **INFO** and left an untracked live position | P0 alert + trader frozen for new entries until cleared |
| R6 | `5db9d3c9` | Planner dedupe spanned a full AI call — two traders both paid for the same read | One planner call per (trade_date, session) |
| R7 | `97e4f541` | The T22 stale/drift gate was **unreachable** — dead code posing as a safety net | Dead block removed; the gate that *does* run is pinned by tests |

## The two that needed judgement

**R1 — the anchor problem.** `PlanScenario` carries no price; trigger and invalid are free text. Deciding *which level* a scenario is about is a heuristic, and a confidently-wrong dot would replace a vague lie with a precise one. So the rule is **emit nothing when the anchor cannot be resolved** — the scenario is omitted, the card keeps its fallback. I deliberately did *not* implement "nearest level on the side implied by direction", because it always produces an answer. Refusal cases are tested: a price matching no level, text with no price, and `2x5m`/`15m` (kept out by a 4+ digit rule).

**A real bug my own fixture caught.** I first used the trade's direction and `!StillValid` for invalidation. `StillValid` reports acceptance through *either* side, so a resistance level price merely sat below all morning read as **invalidated** on a quiet open. Direction is now the way price moves to *kill* the trade (short → above, long → below), and invalidation is acceptance in that direction only.

**R7 — diagnosis, not relocation.** The guard `len(ctx.MarketDataMap) > 0` sat at `engine_analysis.go:202`; the map is filled ~70 lines *below*, in the same function, and a fresh Context is built each cycle. False on 100% of evaluations. I removed it rather than moving it: unreachable safety code reads as a live gate in review and rots silently. **Stale data still does not reach an entry** — `applyStaleDataBlock` (B4) runs post-fetch and turns stale opens into `wait`; three tests now pin that so it cannot rot the same way. Not replaced: suspicious-**drift** detection, which also never ran. Re-landing it needs timeframe-aware thresholds (the flat 90s/5m limits would spuriously hold higher-TF cycles) — **size M**, listed below.

## Verification

`go build` · `go vet` · `go test ./...` · **`-race` clean** on kernel/trader/store/api · `tsc` · `npm run build` · **goldens byte-identical** (`git diff 04475d66 -- kernel/testdata/` empty — R1 writes only a reporting surface and never touches the prompt). vitest **190/191**: the one failure (`RegistrationDisabled` "NoFx Logo") and the `e2e/gate.spec.ts` collection error are the same **pre-existing** pair, untouched.

New tests: 11 Go (scenario ladder, anchor refusals, fixture day, lifecycle projection) · 2 Go (planner guard, incl. a 32-goroutine race asserting exactly one winner) · 2 Go (freeze lifecycle, exits never trapped) · 1 Go (digest two-trader race) · 3 Go (stale coverage) · 1 Go (decline KPI) · 8 vitest (decline branches, gate-blocks panel).

**Playwright — partial, stated plainly.** R2 is verified in a real browser: the declined outcome renders and does **not** contain "applied" (`shots/2026-08-17-w16-decline-copy.png`). **R3's panel could not be browser-verified**: the sandbox is unauthenticated, the real call 401s, and `httpClient` redirects to `/login`; my XHR stub didn't satisfy axios's full adapter surface. Its rendering is covered by 5 component tests driving the same code path with real payloads — but that is not the browser check the dispatch asked for, so I am not claiming it.

## Remaining (not attempted here)

1. **Suspicious-drift detection** — never ran, not replaced. TF-aware re-land after the fetch. **M**
2. **Gate-block counters are in-memory** — the panel resets at the 17:00 CT roll and on restart. It says so on its face; persisting them is **M** (audit item 12).
3. **B4 fails open with no 1m/5m data** — documented and pinned by a test. **S**
4. **The freeze map is in-memory** — an R5 freeze does not survive a restart; the P0 alert row does. **S**
5. **Scenarios with unresolvable anchors show no status** — by design. Giving `PlanScenario` an explicit `anchor` field would fix it at the source, but changes the plan schema and the planner contract. **M**

## Deploy

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
git rev-parse HEAD > deploy/RELEASE     # MANDATORY — else the boot assertion refuses trading
sudo systemctl restart nofx
journalctl -u nofx --since '2 min ago' | grep 'BOOT INTEGRITY'   # expected must equal rev
cd web && npm run build && cd ..        # then hard-reload the browser
```

`day_plan_digests` gains a `trader_id` column via AutoMigrate on start — additive, and the live table has 0 rows so there is nothing to backfill. A plain index, not `uniqueIndex`, which would fail the whole migration at boot on any colliding row.
