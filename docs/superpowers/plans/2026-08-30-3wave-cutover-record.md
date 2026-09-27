# 3-Wave Cutover Record (news-hygiene + security-F1 + weekly-bias)

**Cutover:** 2026-08-30 09:41 CT · **Deployed rev:** `23243670` (week-anchoring fix) · PID 482741.

## Sequence
Merges (order per dispatch): fix/news-hygiene (5f0c31e5) → fix/security-hygiene (883482d4; AUDIT-CHECKLIST 21+22 conflict reconciled) → feat/weekly-bias (b84a96d6). Two cutover-found defects fixed before ship: (1) tz-guard law — `auto_trader_weekly.go` InvalidatedAt bare layout → `kernel.FormatCT` (416eb180); (2) week-anchoring — `weekStartMonday` anchored on the session-day, mapping Sunday morning one week back → wrong-week boot-backfill fired at first boot; calendar-anchored fix + regression test (23243670). Both cherry-picked to feat/weekly-bias. Marker v2: RELEASE=GUIDE_BUILT_REV=23243670 (dev 95b3edaa).

## Boot checklist (all quoted from journal)
- `🔐 BOOT INTEGRITY OK — rev 23243670af35 · expected 23243670 · goldens PASS` ✓
- `📜 scenario schema: 9 conditions […]` ✓
- `📅 calendar: fetched 13 events — 0 day slice(s) stored (src forexfactory)` — true-count form live ✓ (0 stored = frozen slices, honest)
- static T1 file: ISM Mfg 09-01 · ISM Svc 09-03 · NFP 09-04 · PPI 09-10 · CPI 09-11 · Retail Sales 09-16 · FOMC ×2 ✓ (fail-closed net armed; live feed working)
- AUDIT-CHECKLIST: THE 22 BUG CLASSES ✓ (classes 21+22 reconciled in merge)
- WEEKLY boot-backfill: **correctly DEFERRED** (boot 09:41 < Sunday 16:30 CT read time — no "WEEKLY READ starting" line). First-ever weekly read fires 16:30 CT via the scheduler; first session prompt (ASIA ~17:01) is the `## Candles` + `## Weekly Context` proof. (First boot's wrong-week read never wrote a row — no stale WEEKLY doc in DB.)
- watchdog 0 · ERRO 0 · flat-gate re-quoted flat.

## Notes
- NFP Friday runs on this binary.
- First-boot defect (wrong-week backfill) was caught at the cutover itself and shipped fixed in the same dispatch — the depth-guard thin_history/conviction-low proof arrives with tonight's 16:30 read.
