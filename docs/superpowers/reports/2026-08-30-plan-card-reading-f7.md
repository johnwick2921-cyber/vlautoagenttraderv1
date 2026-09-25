# F7 SEAM-WAVE — plan-card "writing" status sticks (store-derived fix)

2026-08-30 · branch `fix/clock-hold` (rides fix/move-seams with F1-F6) · NO deploy (parked).

## 1. Diagnosis — which layer lied

**The server layer lied.** The client polls correctly (SWR `refreshInterval`
15s, `usePlan.ts` `PLAN_REFRESH_MS`), but `/api/plan/today` derived `reading`
from the raw in-flight CLAIM (`at.PlannerReadInFlight(tradeDate, session)`),
not from the store. Planner wake re-reads hold that claim for 3-9 minutes each
(reasoning=max on deepseek-v4-pro), back-to-back all evening, so `reading:true`
was nearly continuous — the card's "The planner is writing a fresh plan" banner
sat over a committed, tradeable plan for hours.

Reproduced live at 21:35 CT (same moment a wake read held the claim):

```
API  GET /api/plan/today?trader_id=8d5c8af5_…&symbol=MNQ&session=ASIA
     {"found": true, "session": "ASIA", "reading": true, "version": 2,
      "lifecycle": "active"}

DB   plans: 2026-08-30:ASIA:8d5c8af5_… | version 2 | lifecycle active |
     trigger_reason owner_reset | created_at 2026-08-30 23:35:34+00:00
```

Journal, same evening — the wake-read chain that kept the claim hot:

```
21:29:12 🧠 planner call (reasoning=max wire=enabled/max cap=65536) completed in 552.0s
21:38:12 🧠 planner call (reasoning=max wire=enabled/max cap=65536) completed in 539.5s
21:43:14 🧠 planner call (reasoning=fast→low wire=enabled/low cap=65536) completed in 193.5s
```

Each holds the claim for its full duration; the card polled every 15s and saw
`reading:true` the whole way — over a plan row that was already written and
tradeable. The benign wake-read failures (read ends, nothing written) are the
same shape: the claim-keyed flag outlived the read outcome.

## 2. Fix — status derives from the STORE

**Server** (`api/handler_plan.go`): the row is fetched FIRST; the response
fields are now:

- `reading` = in-flight claim **only while NO plan row is committed** (the
  pre-first-plan "writing" state). With a row committed it is always false.
- `replan_in_flight` (new) = in-flight claim **while a row IS committed** — the
  plan renders normally; the claim is informational.

Pure helper `planReadingFields(hasRow, inFlight)`:
`no row + in-flight → reading:true` · `row + in-flight → replan_in_flight:true` ·
else both false. A failed read (benign wake failure, no write) therefore always
lands back on the kept plan.

**Client** (`web/src/components/plan/SessionPlanCard.tsx`):
- `found:false` + `reading:true` → renders the "writing a fresh plan" state
  panel (spinner + `readingBanner` text) instead of the bare no-plan card.
- `found:true` + `replan_in_flight:true` → subtle "Planner re-reading… this
  plan stays live." chip (`data-testid="replan-chip"`); the writing banner is
  gone from the committed-plan branch entirely.
- New `replanChip` i18n key (en/zh/id) in `plan-translations.ts`; `replan_in_flight`
  added to the `PlanToday` type.

Polling unchanged (SWR 15s) — the card transitions within one poll interval
because the response flips the moment the row commits.

## 3. Fixtures (all green)

- `api/handler_plan_test.go` — `TestPlanReadingFieldsDeriveFromStore` (4-case
  store-derived table).
- `web/src/components/plan/F7_reading_transition.test.tsx` —
  (a) in-flight + no plan → writing state; (b) same card re-rendered with the
  committed plan → plan renders, banner gone (the one-poll-interval transition);
  (c) read-failed → kept active plan renders, never a stuck writing state;
  (d) read running over a committed plan → subtle chip, no writing banner.
- `P7_no_trade_map.test.tsx` updated to the F7 semantics (`reading:true` over a
  committed plan no longer renders the writing banner).

## 4. Note

The claim/release pairing itself is sound (deferred release in the wrapper) —
this was a semantics bug: the flag's lifetime legitimately outlives the write,
and the UI treated it as the plan's state. Nothing in the planner loop changed.
