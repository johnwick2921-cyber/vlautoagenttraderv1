# 0C — Shadow Demotion: fvg_entry + breakout_retest

Date: 2026-08-31 CT · Wave: 0C-shadow-demotion · Branch: `0c-shadow-demotion` (merged → `dev`)
**STATUS: SHIPPED — cutover 2026-08-31 17:34:21 CT, owner GO.**
Live rev now: `7004a7f1f7266a3d8c354afc7ee27f05b5fda2a4` (PID 1466535).
Rollback kept: `nofx-bin.prev.boot` = previous live `98a9b4cfb479197f55047b31f6cdacc1b565ec85` (PID 1391022).

## What shipped

Config-driven condition-status map; enforcement at the ARM SEAM only; inert
"shadowed" ledger state; E8 complete counterfactual rows; resolved-map boot line.

| # | Implementation item | Verdict |
|---|---|---|
| 1 | `ConditionStatus` precedence: session > base > env > defaults | PASS (kernel tests 7.x) |
| 2 | Defaults `{fvg_entry: shadow, breakout_retest: shadow}` | PASS |
| 3 | `SHADOW_CONDITIONS` / `LIVE_CONDITIONS` env composition | PASS |
| 4 | Enforcement at arm seam only (no gate at decision/sizing) | PASS (wire-loopback silence test) |
| 5 | Boot sweep: resting working/armed rows for shadowed scenarios cancelled (signal ids) | PASS |
| 6 | Inert `shadowed` ledger state — invisible to `ListNonTerminal`, visible to `ListForPlan`, never placed | PASS |
| 7 | `telemetry.armsRefusedShadowed` counter + deduped refusal log | PASS |
| 8 | E8 complete counterfactual rows (R/ATR units, time bars, net-of-friction, ambiguous) | PASS |
| 9 | Resolved-map boot line (once per trader per boot) | PASS |
| 10 | Guide callout + pre-registered promotion criterion (GUIDE_CONTENT_LAW) | PASS |
| 11 | Checklist: new classes 30 (GORM alias scan) + 31 (inert-but-visible state) appended same wave | PASS |

## Resolved condition map at cutover (all 9 known conditions)

| condition | resolved status | source |
|---|---|---|
| reclaim | live | default |
| hold | live | default |
| sweep_reclaim | live | default (docketed Sep-9, NOT shadowed) |
| reject | live | default |
| acceptance | live | default |
| breakout_retest | shadow | default |
| fvg_entry | shadow | default |
| breakdown_continue | live | default |
| breakup_continue | live | default |

(If the owner sets `SHADOW_CONDITIONS` / `LIVE_CONDITIONS` or a session/base
override, the resolved map recomputes at next boot; the boot line prints it.)

## Pre-registered promotion criterion (verbatim)

> A shadowed condition returns to LIVE only if, at n >= 30 shadow setups on our
> own tape, its net-of-friction expectancy LOWER CONFIDENCE BOUND exceeds zero.
> Otherwise it remains shadowed, or is deleted at the court's discretion.
> No promotion on narrative… No promotion on a point estimate without its
> interval.

## Evidence at cutover (all flat-gate legs quoted fresh)

- ASIA 16:30 CT read on the OLD binary: **zero rows authored** — no
  fvg_entry/breakout_retest exposure exists or existed under the old binary.
- PRE-cutover flat gate (17:33 CT): DB OPEN=0 (`trader_positions` 0 open/576
  CLOSED · `trader_orders` 0 open · `armed_orders` armed/working 0) · API
  positions `[]` · API open-orders MNQ `[]` · NT8 snapshots
  `account=Sim101 count=0` + `account=SimAccount1 count=0` @17:33:21.
- Window: 17:34 CT (forbidden 16:45–17:10 already passed; owner GO received).
- Cutover: 17:34:21 CT `kill -9 1391022` → systemd relaunched PID 1466535.
- **Boot 17:34:26 CT quoted:**
  - `🔐 BOOT INTEGRITY OK — rev 7004a7f1f726 · built 2026-08-31T22:12:55Z · expected 7004a7f1f726 · goldens PASS`
  - `🔬 conditions: live [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim] · shadow [breakout_retest, fvg_entry] (process-level: defaults+env; per-trader resolved map prints at first arm cycle)`
  - `📜 scenario schema: 9 conditions […]`
  - 0 `[ERRO]`, 0 panic since boot; balance frames immediate (equity
    52216.00 == pre-cutover).
- POST-cutover flat gate (17:35 CT): API positions `[]` · open-orders MNQ
  `[]` · DB armed nonterminal 0 · NT8 snapshots `count=0` ×2 @17:35:21.

## Live proof (first natural read)

- **Said plainly: no refusal line and no E8 counterfactual row have occurred
  yet.** The active ASIA v1 plan is `lifecycle=no_trade` (created 17:18:32 CT
  under the old binary) — zero scenarios authored, so the arm seam has had
  nothing to refuse. The per-trader resolved map line also awaits the first
  arm cycle by design. The shipped behavior is proven by the six 0C fixtures
  (wire-loopback silence, boot cancel, config flip, E8 counterfactual
  authoring, inert-row visibility) and the boot `🔬 conditions` line; the
  first live shadow refusal will land when a future read authors a shadowed
  scenario.

## Cutover runbook (used)

1. Swap: `mv nofx-bin nofx-bin.prev.boot` → `cp ~/nofx-staged/nofx-0c-bin
   nofx-bin` (stamp verified on the deployed file:
   `vcs.revision=7004a7f1f7266a3d8c354afc7ee27f05b5fda2a4`, `modified=false`).
2. `kill -9 1391022` → systemd relaunch → boot checklist quoted above.
3. Post-boot flat-gate re-quote (quoted above).

## Shipped vs deferred

- Shipped: items 1–11 above.
- Deferred: nothing structural. (C# AddOn from class-27 wave still awaiting
  owner F5 + NT8 restart — separate wave, no C# touched here.)

## Rollback

```
mv nofx-bin.prev.boot nofx-bin && <revert deploy/RELEASE> && kill -9 <PID>
```

`nofx-bin.prev.boot` holds `98a9b4cfb479197f55047b31f6cdacc1b565ec85` — the
pre-cutover live rev — so a single `mv` back + `kill -9` restores exactly
the pre-0C state.

## Anything the owner will still see wrong on screen

- Nothing. The guide page now matches the running binary (`GUIDE_BUILT_REV`
  `7004a7f1…` == health revision `7004a7f1f726…` — no drift banner).
- The shadow rule is enforced at the arm seam from the first cycle; the
  first visible refusal will arrive when a future plan authors an
  fvg_entry/breakout_retest scenario (ASIA v1 is no_trade).
