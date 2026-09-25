# 0A-2 Sync-Row Audit + Test-Row Quarantine & Ledger-Surface Honesty — 2026-08-31

**Rev:** `98a9b4cfb479197f55047b31f6cdacc1b565ec85` (Go, boot 15:36:04 CT, PID 1391022, goldens PASS)
**FE:** committed + `npm run build` (served by the frontend host; no boot needed)
**Rollback kept:** `nofx-bin.prev.boot` = `a0c7ff0b118769cabfb39c03d965637190461768`
**DB backups:** `~/nofx-backups/class27-backfill/data.db.pre-backfill-151037` · `~/nofx-backups/0a2/data.db.pre-0a2-153344`

---

## LEDGER-SURFACE HONESTY (UI only, no boot)

`web/src/components/trader/PositionHistory.tsx` — every row is now classified by a
single deterministic rule (`classifyPosition`, exported + fixture-tested):

| State | Rendering |
|---|---|
| `unresolved` (`close_reason='unresolved'`) | shown · P&L "—" · badge **exit unknown** |
| `duplicate` (`reconcile_flat` row sharing `entry_order_id` with a real close) | hidden by default · toggle **duplicates: hidden/shown** in the filter bar · greyed + badge **duplicate of #N** (577→578, 576→575) |
| `hidden` (`reconcile_flat` with no duplicate evidence + `e7_farside_test` test-seam) | excluded from the list entirely |
| `normal` | P&L = `pnl_corrected ?? realized_pnl` (corrections win — mirrors the store) |

- **Day total** (`Today (CT)` in the footer) uses the SAME rule as the ledger's
  `GetSessionDayActivity`: unknown/test-seam/duplicate excluded, corrections win,
  `pnl_corrected IS NOT NULL` (the A-2 rule), CT day boundary. The visible footer
  total only ever sums `normal` rows → **visible rows and visible totals agree**.
- **Fixtures per state:** `web/src/components/trader/PositionHistory.fixtures.test.ts`
  (5 tests, incl. today-recompute = +164.00 from 575/578/580 with 576/577/579/573/574 excluded).
- `pnl_corrected` + `pnl_correction_note` added to `HistoricalPosition`.

## 0A-2 PART 1 — the six sync rows

| row | date | side | entry | reconstruction (quoted BEFORE writing) | write |
|---|---|---|---|---|---|
| 110 | 06-04 | SHORT | 30512.25 | no opposite-side fill after entry (only SELLs all day) → no derivable exit | **unresolved** (exit 0, note) |
| 249 | 06-12 | LONG | 30000.25 | next SELL is 29917.75 @19:42 (row 250's own entry, 1h52m later) — no exit fill at the 17:52 close | **unresolved** |
| 256 | 06-24 | LONG | 30114.75 | fills after entry: BUY 30116.75 (row 257's entry, same-side) — no SELL ever | **unresolved** |
| 288 | 06-28 | SHORT | 29567.75 | fill 204 BUY **29652.50** @23:24:38 = row 289's LONG entry (netting handoff) → **exit 29652.50, P&L = −84.75 pts × $2 = −169.50** | **exit 29652.50 / −169.50 (sync)** |
| 542 | 08-21 | SHORT | 29310.00 | NT8 trace 09:27:42 UTC: `price=29310 marketPosition=Long operation=Add` → Flat Remove — close filled BUY **@ 29310.00 exactly** | **NO write — genuine scratch, $0 real** (note pinned) |
| 551 | 08-25 | SHORT | 29163.00 | fill 378 BUY 29163.0 `nt8-exit-551` — real priced exit at entry | **NO write — genuine scratch** (note pinned) |

Evidence sources: `trader_fills` (ids 49-57, 164-167, 169-177, 203-204, 362-363, 376-382),
`trader_positions` neighbors (250/257/289), NT8 trace `trace.20260821.00000.txt`.

## 0A-2 PART 2 — which rulings ate them

The four named samples (week-1 money table n=9 · ARM_MIN_RR n=18 · gates net-saving
· killzone in/out) are **not defined by row ids in any repo artifact** (reports,
plans, session store all checked). Mapping by date/provenance:

- **Week-1 money table (n=9):** only **110** (Jun 4) can fall inside a May-28→Jun-5
  "week 1" window. Its recomputed figure: **excluded / "—"** (was a fake $0).
  Rows 249/256/288 are weeks 3-4; 542/551 are August — outside.
- **ARM_MIN_RR n=18 set:** June rows predate the armed system entirely; 542/551 are
  `source=system` rows with no `armed_orders` entry (`entry_order_id` NULL) → **none
  of the six is in that sample. No recompute.**
- **Gates net-saving figure:** the refusal autopsy runs **Sep 3** (per
  `2026-08-26-bar-persistence.md` §10) — not computed yet; these six are closed
  positions, not gate refusals, so they're outside that sample by construction.
- **Killzone in/out comparison:** no shipped artifact defines this sample (killzones
  are grading-only metadata, `kernel/adherence.go`). Needs the owner's sample
  definition to map — flagged.
- The ONLY recomputed number from this audit: **288: 0.00 → −169.50.**

## 0A-2 PART 3 — test-row quarantine (the broken promise, now kept)

`store`: new `CloseReasonTestSeam = "e7_farside_test"`; `UnknownPnLReason` now covers
reconcile_flat + unresolved + test-seam. Every real-P&L aggregator excludes it:
`GetPositionStats` (also fixed a pre-existing GORM scan-mapping bug — `TotalPnL`
aliased to `total_pn_l` and silently read 0), `GetFullStats` (count+P&L),
`GetSessionDayActivity`, `CountConsecutiveLossesSince`, `GetSymbolStats`,
`GetDirectionStats`, history `RecentPnL` + streaks. Fixture:
`store/test_seam_exclusion_test.go` — +6/−1 test-seam rows + a real +32.5 row →
every surface returns only the real trade.

**Still open (flagged, out of scope):** `GetSessionDayActivity`'s ENTRIES count
includes test rows (the max-daily-trades guardrail saw `today=10` partly inflated by
the E7 rows 572-574). Fixing it changes guardrail behavior → deferred to the owner.

## 0A-2 PART 4 — era forensics (one paragraph)

June 2-4 (37 rows) and Aug 10-13 (19 rows) came from two dead paths: June was the
pre-TCP era where the AI "Close" path wrote `exit_order_id='Close'` and stamped
exit=entry when no close fill was captured (no NT8 traces exist before Aug 1, and
`trader_fills` has no opposite fill for those rows — the fabricated-$0 class); Aug
10-13 was the TCP reconcile orphan-close writing the `reconcile_flat` entry-as-exit
placeholder. Both producing paths are dead today: no current code writes
`exit_order_id='Close'` (grep-proven), and class-27 FIX 2 replaced the orphan-close
with netting-fill reconstruction or `unresolved` — exit=entry is never written again.

## Gates

- `go test ./...` → all packages ok, 0 FAIL (goldens PASS)
- web `npx vitest run` → 36 files / **292 tests** passed · `npx tsc --noEmit` clean · `npm run build` ✓
- Boot: `🔐 BOOT INTEGRITY OK — rev 98a9b4cfb479 · goldens PASS` (15:36:04 CT)
- Flat at boot: DB open rows = 0 · NT8 `positions snapshot account=Sim101 count=0`
