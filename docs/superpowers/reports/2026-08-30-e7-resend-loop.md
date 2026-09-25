# E7 RESEND-LOOP INCIDENT — CLOSEOUT + F1 root cause

2026-08-30 · branch `fix/clock-hold` (rides fix/move-seams) · incident window
22:32–23:14 CT · kill 23:14:24 CT · silence proven 23:19:38 CT.

## A. Placement quotes + the re-placer

Every placement of the test/looping orders, from the journal (all in the
running E7 binary `06f1dc4e`, PID 848146):

```
22:32:55 📌 armed TEST-E7 → WORKING stop-entry 28700.00 signal=c15c48a6-a036-4991-b4d0-9c9e0b2295b3 (seam)
22:44:36 📌 armed S2 → WORKING limit 29371.50 signal=8115d2ec-738e-4de5-80c0-00b8051f2c81 (band ±100t)
22:50:36 📌 armed S2 → WORKING limit 29371.50 signal=6f0ce8fa-764f-4e6a-960e-ee10e1526a9b (band ±100t)
23:02:24 📌 armed S2 → WORKING limit 29371.50 signal=b6811ca0-2064-44ad-8df9-2e213dbebe6e (band ±100t)
23:04:41 📌 armed S2 → WORKING limit 29371.50 signal=e6eb0ec1-d725-42a9-84e7-e30aab70cf9f (band ±100t)
23:06:36 📌 armed S2 → WORKING limit 29371.50 signal=675f0022-6c5a-4e8a-94c0-1cfe70600c8f (band ±100t)
```
Fills in the same window: 22:34:36 TEST-E7 @29346.25 · 22:46:18 S2 @29347.25 ·
23:01:10 S2 @29350.25 · 23:03:14 S2 @29347.50 · 23:05:17 S2 @29349.00 ·
23:08:36 S2 @29348.50 — every S2 placement filled at market within ~90s.

**The re-placer: the armed authoring loop, `trader/armed_executor.go`
`maybeManageArmedOrders` (every cycle).** A filled ledger row is TERMINAL, so
`ListNonTerminal` no longer returns it → the loop authors a NEW row from the
plan's still-active Arm spec (`UpsertArm` on `row.ID==0`) → `runArmedPlacement`
places it. The TEST-E7 stop itself was a single seam placement (signal
c15c48a6, `api/handler_armed_seam.go` `place_stop`) — the LOOP was the S2 limit,
and the "re-appearing stop" the owner canceled was the same S2 limit chain
re-authored every cycle. NOT a seam retry, NOT reconnect replay.

## B. Why it looped (bug class) + why the kill is dead BY DESIGN

Two defects, both in the shipped armed manager:

1. **Wrong-side placement band.** `runArmedPlacement` used the symmetric
   `math.Abs(price-entry) <= band` — a LONG limit 24 pts ABOVE market sits
   inside a ±25pt band, is marketable, and fills instantly at market. A
   resting limit must sit on the unmarketable side only.
2. **Terminal rows don't consume the spec.** "Re-arm does NOT auto-re-arm"
   (armed_executor.go:160) held only for PLAN changes. At the row level the
   plan is the authorizer: after a fill the spec is re-authored next cycle —
   fresh AI authorization was NEVER actually required.

**Kill mechanics (dead by design, not luck):** the authoring loop re-reads the
plan lifecycle from the STORE every cycle
(`GetLatestPlanForTraderSession(...).Lifecycle != "active"` → `reason` →
`cancelArmedOrdersSync` + return). Setting the ASIA plan to `lifecycle=no_trade`
+ `trigger_reason=e7_incident_kill:manual_cancel_respected` flips the
executor's OWN cancel path — no code, no restart. The remaining non-terminal
rows were voided at the same moment, and `ARMED_TEST_SEAM=off` is now in
`.env` so no future boot re-enables the seam.

Before/after (23:14:24 CT):
```
BEFORE: plans ASIA v2 = active (owner_reset) · non-terminal arms: id=13 S3 armed 29414.21, id=14 S2 working 29371.5 (signal 58f77728)
AFTER:  plans ASIA v2 = no_trade (e7_incident_kill:manual_cancel_respected) · non-terminal arms = 0 · ARMED_TEST_SEAM=off
```
DB backed up first: `~/nofx-backups/e7-kill/data.db.pre-e7-kill` (605 MB,
VACUUM INTO).

## C. Ledger hygiene

`trader_positions` rows tagged (source AND close_reason):
`572 SHORT 29346.25`, `573 LONG 29347.00`, `574 LONG 29348.25` → all
`e7_farside_test` — excluded from strategy P&L.

## D. Zero-working proof (same minute, 23:19:38 CT)

```
23:19:21 tcp_server: positions snapshot account=Sim101 count=0
23:19:21 tcp_server: positions snapshot account=SimAccount1 count=0
ledger working=0 · positions_open=0 · placement/fill lines since 23:14:24 = 0
```
5-minute watch (23:14:38→23:19:38): **zero** placement/fill lines.

## E. F1 fix (this branch) — the money bug

- `armWrongSide(side, entry, price)`: long entry ≥ market / short entry ≤
  market → the armed row is VOIDED at placement time (`wrong-side arm: …`),
  never placed. The exact S2 shape tonight (long 29371.50 vs 29347.25) dies
  here.
- `armSpecConsumed`: a `filled` row — or a `wrong-side` void — at the SAME
  plan version consumes the spec; the authoring loop skips it. Fresh
  authorization = a NEW plan version (the 1.4 contract, now actually true).
- Fixtures (green): `TestArmWrongSide`, `TestArmSpecConsumed`,
  `TestFilledArmConsumesSpecAtSameVersion`,
  `TestWrongSideVoidConsumesSpecAtSameVersion`.
- AUDIT-CHECKLIST class 21 appended.

## Next (owner-gated, in order)

1. Owner: F5 C# file (md5 e17464a8) → NT8 copy → F5 compile → full NT8 restart.
2. Bot side stays parked on this branch until the F1 merge; the live bot runs
   with the DATA-level kill (no_trade plan) until then.
3. After F1 cutover + NT8 restart: 2.2 reconnect proof → far-side proof #2
   (stop rests at 28700, cancel ack) — only then re-enable anything.
