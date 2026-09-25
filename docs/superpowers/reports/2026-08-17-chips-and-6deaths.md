# The six deaths were ONE mechanism fed a poisoned input — and the chips were dead because the card has TWO chip rows and I wired only one

## ITEM 1 — why the chips were still dead

**It was my miss, not a stale build.** Ruled out first, with receipts: the running binary is HEAD (`deploy/RELEASE` = `359ace1c` = HEAD), `GET /api/plan/versions` returns **401** while a bogus path under the same group returns **404** (route live, not missing), and `web/dist` was rebuilt 19:44 and *does* contain the new chip code. So the owner was not on old code.

The card renders version chips **twice** — `SessionPlanCard` header **and** `PlanFooter`. `VersionChips` renders `disabled={!onSelect}`, and I wired only the header. The footer row was a set of disabled buttons; tapping those does nothing. Both rows now share one derivation (`newestVersion` / `versionCount` / `versionTitle`) so they cannot drift apart again (`403a9205`).

**Tap targets were also genuinely wrong**, which the last pass claimed to have fixed and had not: the 44px rule was keyed on `@media (pointer: coarse)` alone, so a narrow window or a touch laptop reporting a fine pointer still got **26×18** — and 18px misses even the WCAG 2.2 SC 2.5.8 **24×24** floor on desktop. Base is now ≥24×24 everywhere, 44×44 under `(pointer: coarse), (max-width: 640px)`.

**Verified in a real browser** — component-level, and labelled as such: the live app is behind auth and `httpClient` redirects to `/login` on 401, so it cannot be driven headlessly. `web/chips-harness.html` mounts the **real** module from the Vite dev server at n=1, 2, 6.

| viewport | chips | ellipsis | overlaps | min gap | tap target | disabled | header buttons |
|---|---|---|---|---|---|---|---|
| 390×844 | 1 / 2 / 6 — all rendered | none | **0** | +4.00px | **44×44** | 0 | inside the card |
| 1440×900 | 1 / 2 / 6 | none | **0** | +4.00px | 24×24 (AA) | 0 | inside the card |

Contrast: inactive **5.52:1**, active **7.48:1** (AA floor 4.5:1). Active also carries `font-weight:700`, a filled ground and `aria-current`, so it is not signalled by colour alone. A real `.click()` on v5 returned `"5"`. Screenshots: `assets/chips-390px.png`, `assets/chips-desktop.png`.

**What the owner can now click:** any version chip in **either** row, and any row of the new death-history panel. Each opens that version read-only, marked HISTORICAL, with its bias/levels/scenarios/rules as they were, the death reason (which condition, which price), a plain-language diff vs the version that replaced it, re-plans remaining, and a "← Back to current" button. The owner door is force-closed on a historical view, because every mutating endpoint writes the *latest* version and an edit made while reading v2 would land on v6.

## ITEM 2 — why six plans died

**One mechanism, compounded by a poisoned input.** Every version carried the identical prior-day trio **PDL 30146.75 / PDC 30147.50 / PDH 30148.25** — a **1.5-point cluster** ~45 points below spot — because Friday was 959 padded bars at exactly 30147.50. PDH and PDL are literally the high and low of the *one* real Friday bar (h30148.25 / l30146.75); PDC is the padding price itself. v1's own label says it out loud: **`PDC/RTH-H/RTH-L 30147.5`** — the planner was told Friday's close, RTH high and RTH low were all the same price. The death check then judged that cluster against a **33-hour** window on **1-minute** bars, where it is touched-and-consumed by construction.

The replay reconstructs the exact 2000-bar cache each check saw: 364 real pre-Friday bars (Thu 22:56Z → Fri 04:59Z), the single real Friday bar, Friday's 16-hour padded block, and the Sunday reopen (45 real bars). Bars captured live from `/api/klines`; the padded block reconstructed from the byte-decoded `.ncd`, not guessed.

| ver | born (UTC) | died | lvls | what killed it | price | window used | BEFORE | AFTER |
|---|---|---|---|---|---|---|---|---|
| v1 | 22:12:46 | 22:17:53 | 6 | all 6 touched+accepted; ONH 30200.50 consumed only via **Thursday's** range | 30197.50 | full ~33h, 1m | **dead** | alive |
| v2 | 22:19:02 | 22:20:58 | 6 | same, ONH 30203 | ~30193 | full ~33h, 1m | **dead** | alive |
| v3 | 22:23:34 | 22:25:11 | 6 | same | ~30193 | full ~33h, 1m | **dead** | alive |
| v4 | 22:26:19 | 22:31:20 | 6 | same | ~30193 | full ~33h, 1m | **dead** | alive |
| v5 | 22:32:28 | 22:37:29 | 6 | same; PDL had **997** consecutive 1m closes above it | 30193.50 | full ~33h, 1m | **dead** | alive |
| v6 | 22:37:29 | — | **0** | not a death — the levels-less NO-TRADE plan the exhausted budget produced | — | — | alive | alive |

**All five real deaths reproduce, all five survive the fix. No third cause in the death check.** A second test proves the padding refusal *alone* also breaks the loop: against real-bars-only history not one version dies, even under the old logic. So either fix would have prevented the outage; both are in.

**Methodology note worth keeping:** my first replay used today's `PlanIsDead` for the "before" column and reproduced only 4/5 deaths — because that function already carries the timeframe fix. The test's own "did the fixture reproduce the outage" guard caught it. "Before" now calls a local `planIsDeadPreFix` that mirrors the pre-`06f3343e` logic exactly.

**The guard (2d).** A death was five identical `DIED` lines with no condition and no price. `kernel.DescribePlanDeath` derives the explanation *from* the decision — same window, same timeframe — so verdict and explanation cannot disagree. Every death now logs the killing condition, the price, and per-level evidence (`ONH 30203.00 accepted below (4× 5m closes)`); reaching the cap emits a **P1** alert naming that condition; `writeNoTradePlan`'s stored reason carries it; and the card grows a **death-history panel** listing every superseded version with what killed it, each row opening that version.

## Incidental finding — report only, per instruction

The bar cache is clean: the captured Sunday-reopen window has **zero** flat bars, and Friday's real history is restored (1325 real bars between Thu 22:56Z and the reopen). Nothing to purge. The ingest refusal (`47dbb269`) remains as the guard.

## Exit bar

`go build` · `go vet` · `go test ./...` green · **`-race` clean** (kernel/trader/store/api) · `tsc` clean · `npm run build` OK · vitest **213/214** — the one failure and the `e2e/gate.spec.ts` collection error are the same pre-existing pair as the last four runs. **Goldens byte-identical**: `DescribePlanDeath` is new and read-only, no prompt path changed. No new config fields, so no config-truth step applies.

**Session check:** a second Claude session (`554049f5`) is open but **dormant** — no transcript write since 18:36, ~70 minutes before this run started; no repo state from it.

## Deploy

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
git rev-parse HEAD > deploy/RELEASE     # MANDATORY — else the boot assertion refuses trading
sudo systemctl restart nofx
cd web && npm run build && cd ..        # then HARD reload (Ctrl+Shift+R)
```

No NT8 steps required. Visual check: tap a version chip in either row — header or footer — and confirm the HISTORICAL banner with the death reason appears, then "← Back to current".
