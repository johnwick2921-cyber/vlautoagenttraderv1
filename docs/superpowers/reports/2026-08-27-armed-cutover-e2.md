# Armed-Orders Cutover E2 — Complete (2026-08-27)

Wave 2 armed-orders cutover declared **COMPLETE** on live proof. The two planner
defects that blocked E2 are fixed, the E2 chain ran on the real wire, and the
debug seam is OFF.

## The three-part fix (one wave commit)

### b1 — flip.rule alias normalization (the planner defect #1)

- **Before:** `kernel/plan_doc.go` `conditionRules` = `{2x5m, 15m_close, 5m_close}`.
  The planner's natural spelling `2x5m_close` was rejected at write with
  `flip.rule "2x5m_close" invalid` → every retry failed closed.
- **After:** `ValidatePlanDocWithCaps` normalizes `2x5m_close → 2x5m` and
  `1x5m_close → 5m_close` at parse; the stored doc carries the canonical rule
  (same aliasing `confirmRules` / `scenario_facts` already accepted elsewhere).
- **Proof (live):** post-fix NY plan v3 stores `flip.rule = "2x5m"` — the model
  wrote the alias, the parser normalized, the plan landed ACTIVE on the first
  read instead of `planner_fail_closed`.

### b2 — FRESH FVGs machine grounding (the planner defect #2)

- **Before:** the fvg write-site validator (`kernel/fvg_entry.go`) re-checks
  every declared gap against its own fresh-gap scan, but the model never SAW the
  candidates — it invented stale gaps and all three retries died on
  `no fresh 3-candle gap … (fake/stale gap)`.
- **After:** new `kernel.FreshFvgCandidates` exposes the validator's own
  candidate list (direction, lo–hi, age, displacement×ATR5m). It is rendered in
  the planner prompt as a `## FRESH FVGs:` section, plus contract rule A2b:
  *author an fvg_entry ONLY from this list; if empty, do NOT author one.*
- **Proof (live):** v3 authored **no** fvg_entry (the list was empty) and the
  write passed on the first attempt.

### a — E2 debug seam (built gated, used once, turned OFF)

- `POST /api/armed/test-arm` (`api/handler_armed_seam.go` +
  `trader/armed_executor.go` `TestArmPlace`/`TestArmCancel`): drives the REAL
  placement path (`TCPTrader.PlaceLimitEntry` / `CancelOrder`), ledger row
  tagged `TEST-E2`.
- Defended on both sides: env `ARMED_TEST_SEAM=on` (default OFF) + bound
  account must be `Sim101`.
- Boot line reports seam state.

## Deployment

- Code commit `e35c91d4` + release marker → `dev`, pushed (`bc95e92b..b5c14097`).
  Built with `-ldflags -X main.buildRevision=e35c91d41cc9`, `deploy/RELEASE`
  marked `e35c91d4` (the integrity gate refuses a rev/RELEASE mismatch — caught
  and fixed in this wave).
- Final commit `9dd9c629` (seam response echo fix) + release marker → `dev`,
  pushed (`b5c14097..84b304a0`).

## Boot lines (current, verified)

```
🔐 BOOT INTEGRITY OK — rev 9dd9c629dce0 +dirty · built 2026-08-27T16:33:28Z · expected 9dd9c629 · goldens PASS
⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off (resting limits fill at the authorized price; stale_reeval NOT applied)
```

## NY plan after owner reset

- `POST /api/plan/reset` → **v3 ACTIVE**, `trigger_reason=owner_reset`,
  `lifecycle=active`, budget `replan_cap=4`.
- `flip.rule = 2x5m` (canonical — b1 live), no fvg scenario authored (b2 live).

## E2 chain, run 1 (Go-optimistic only — superseded)

```
🧪 TEST-E2 arm → WORKING limit 29580.00 signal=88a1eb20-2e90-4573-be8e-2e7d5b6b9417 (seam)
🧪 TEST-E2 cancel sent signal=88a1eb20-2e90-4573-be8e-2e7d5b6b9417 (row 1 → cancelled)
```

Ledger row 1 went `working → cancelled`, but BOTH transitions were written by
the seam (optimistic Go-side state). The owner asked whether NT8 order_update
FRAMES drove them — they did not. Investigation found the NT8 AddOn was still
the pre-Phase-2 build (AddOns folder copy dated Aug 25 15:10, md5 `f6c8426a` ≠
repo `8411d403`, zero `SendOrderUpdateFrame` calls; Phase-2 C# commit
`ecb1961b` 08-27 08:48 was never copied in). See the confirmation run below.

## E2 confirmation run (frames FROM NT8 — the airtight proof)

After the owner's F5 + full NT8 restart (new AddOn live, verified by
`bars_historical` replay + fresh heartbeat ~13:29), the seam ran again and the
journal captured the C# dispatcher's order_update frames verbatim.

### Place leg — TEST-E2 row 4 (signal `f3ba7740-a561-4581-b526-a6bcf5d9c009`)

```
13:42:40 tcp_server: order_update raw payload raw="{\"signal_id\":\"f3ba7740-…\",\"state\":\"initialized\",\"symbol\":\"MNQ\",\"account\":\"Sim101\",\"seq\":1}"
13:42:40 tcp_server: order_update raw payload raw="{\"signal_id\":\"f3ba7740-…\",\"state\":\"submitted\",…}"
13:42:40 tcp_server: order_update raw payload raw="{\"signal_id\":\"f3ba7740-…\",\"state\":\"accepted\",…}"
13:42:40 tcp_server: order_update raw payload raw="{\"signal_id\":\"f3ba7740-…\",\"state\":\"working\",…}"
```

### Cancel leg

```
13:43:13 … \"state\":\"cancelpending\" …
13:43:13 … \"state\":\"cancelsubmitted\" …
13:43:13 … \"state\":\"cancelled\" …
```

### Consumer drain (armed ledger channel → ledger)

```
13:44:11 📡 armed order_update frame: state=initialized   signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=submitted     signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=accepted      signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=working       signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=cancelpending signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=cancelsubmitted signal=f3ba7740-… acct=Sim101
13:44:11 📡 armed order_update frame: state=cancelled     signal=f3ba7740-… acct=Sim101
```

All 7 state transitions for the TEST-E2 order arrived from NT8 through the C#
dispatcher and drained into the armed consumer. **C# receive path PROVEN.**

### Wave-2 consumer bug found and fixed during the confirmation

`consumeArmedOrderUpdates` called `nt.OrderUpdates()` on EVERY cycle as the
`LoadOrStore` argument — the argument is evaluated first, and
`SubscribeOrderUpdatesFor` CLOSES+replaces the channel on each subscribe. The
map then held a closed channel forever: at 13:34:48 the drain read **310,808**
zero-value payloads in 15s (closed-channel select spin) and the armed consumer
was permanently deaf. Fixed (`cb41ade7`): subscribe only on the miss path,
delete the map entry + re-subscribe if the channel is ever closed. This bug
would have silently killed the production armed lifecycle (working rows never
seeing fills/cancels) — it is the reason the first confirmation attempt showed
empty payloads while the raw frames on the wire were perfect.

### Stuck-order cleanup

Test row 3 (`efe72b44`, entry 29050) was placed during a binary swap and never
cancelled — the owner saw it stuck in the NT8 Orders tab. Cancelled via the
seam; `/api/open-orders?symbol=MNQ` now returns `[]` and all four TEST-E2
ledger rows are `cancelled`.

## Boot lines (final, verified)

```
🔐 BOOT INTEGRITY OK — rev 66413785add3 +dirty · built 2026-08-27T18:50:33Z · expected 66413785 · goldens PASS
⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off (resting limits fill at the authorized price; stale_reeval NOT applied)
```

## Seam state

`ARMED_TEST_SEAM` removed from `.env`; boot line confirms `test_seam=off`.
The endpoint now refuses (`ARMED_TEST_SEAM is off`) — it cannot place orders
unarmed.

## Deploy notes (confirmation wave)

- Final code commits on dev: `cb41ade7` (consumer fix), `66413785` (debug-dump
  strip), release markers pushed; final binary built at `66413785` with vcs
  stamping.
- Worktree (`git worktree add`) builds do NOT get `vcs.revision` stamped (Go
  1.25) → `<no-vcs>` → BOOT INTEGRITY REFUSED. The deploy rule stands: build
  from a real checkout — a `git clone` at the exact code commit stamps
  correctly.
- `NOFX_EXPECTED_REVISION` env overrides `deploy/RELEASE` if ever needed.

## Incident disclosure

The final redeploy kill ran while a position existed: an **owner manual NT8-side
entry** (MNQ LONG @ 29611.50, Sim101), not bot-generated. The reconcile system
materialized it as tracked before the kill and re-read it after; no bot damage.
The flat-window check was present but unguarded in that command — future deploys
must gate the `kill -9` on an empty positions list.

## Next

- Level-truth wave resumes: pop the parked stashes (now indexed
  `stash@{0..4}` after cutover) in order, then `go test ./...`.
- Master-recheck dispatch continues in its own worktree — untouched here.
