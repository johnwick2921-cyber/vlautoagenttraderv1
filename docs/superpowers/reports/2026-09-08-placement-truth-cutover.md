# Placement truth: 98f4ec6e cutover and remaining feed-clock census

Owner GO received 2026-09-08. Lane: `placement-truth-96604090/root[unlisted]`, continuation of `fix/placement-truth-0907`. Evidence grades follow the canon: **A** directly observed source/output/frame; **B** inference; **C** unproven hypothesis. Clock classification is stated separately, so an evidence grade cannot masquerade as a clock choice.

## Scope and clock census, recorded before further helper changes

[A] Census at merged HEAD `9c2ab980986f5b9d07d7a7c21e833a9659ebe296`, using `git grep -n feedNowUTC` over all tracked files: **one remaining production call, one test call**. No helper behavior change in this cutover. The approved binary is `98f4ec6e499eb7dafb31e192460cb774c9a425df`; the merged HEAD differs only in this wave's existing report and `web/src/guide/types.ts`.

| Consumer / reference | Correct clock and actual behavior | Evidence and risk |
|---|---|---|
| `trader/ninjatrader/tcp_trader.go:677`, `MoveStopToBreakeven` → `MoveStopPayload.Timestamp` | **Wall clock: command creation / transport age.** Currently stamps the latest 1m bar close, else 5m close, else wall time, rounded to seconds. This is not a market-fact field merely because it is populated from a bar. | **[A]** Sole remaining production call. `SendMoveStop` (`provider/ninjatrader/tcp_server.go:1090`) writes immediately after its write lock without consulting this timestamp. h1 `HandleMoveStop` (`ninjascript/VLTraderTCPClient.cs:1982–2043`) reads signal id and new stop price, never `timestamp`. **[B]** Latent misleading command-age data; a future freshness reader would inherit the same stale-bar bug. **[C]** A current h1 move-stop stale-timestamp rejection is unproven and is not supported by this receiver source. |
| `trader/ninjatrader/placement_truth_test.go:31` | **Market fact, intentionally old bar close**, compared with wall time to prove the fixture's bar is at least 29 minutes old. Entry payload age is checked separately at receipt. | **[A]** Test-only consumer. Retain the bar clock here; replacing it with wall time would destroy the regression fixture. |
| `trader/ninjatrader/tcp_trader.go:279`, helper itself | **Market fact when a bar exists**, with a **wall-time fallback** when neither 1m nor 5m exists. Chooses 1m by preference, not newest across both timeframes. | **[A]** Definition, not another caller. **[B]** A caller using the result as market freshness can mistake missing data for current data because of the fallback. No helper-wide change authorized or made as part of this census. |
| `kernel/clock_drift.go:57`, related runtime warning at `:83`; `kernel/clock_drift_test.go:2` | Not calls. The old text says entries are feed-stamped. Actual `clockDriftMs` computes wall time minus bar close: **wall-versus-market comparison**, which must preserve both clocks. | **[A]** The feed-stamped claim is obsolete after 98f4ec6e. `applyClockDriftBlock` logs only. **[B]** Host clock skew remains a transport-age risk now that entry commands correctly use wall time. The existing F6 authoring hold is not a universal entry/send clock-synchronization guarantee. Text/guard follow-up is recorded, not silently folded into the approved binary. |
| Historical references: `2026-09-07-h1-stale-signals.md:56–57`, partner-only commits `:1`, partner-update runbook `:53` | Historical description / commit names; **no executable clock consumer**. | **[A]** Preserve as historical evidence. |

[A] The three entry call sites already fixed by this wave are `tcp_trader.go:398` (market), `:467` (limit), `:544` (stop entry): each uses command-creation wall UTC with millisecond precision. Go checks the actual payload timestamp before queueing and again under the write lock before sending; it does not refresh timestamps on retry. h1 source has `STALE_SIGNAL_AGE_SECONDS = 60` and compares `DateTime.UtcNow - sigTime.ToUniversalTime()` at `:814–815`.

## Deferred AddOn half and boot disclosure

**C# rejection-reason producer NOT LIVE until the next owner AddOn copy → F5 compile → full NT8 restart, followed by a received build receipt.** Go can receive and store the additive reason now; h1 does not send that reason. A missing reason stays `reason unavailable (NT8 frame omitted reason)`.

[A] The approved binary's compiled `PlaceConfirmBootLine(false)` suffix is `broker-reason: reason unavailable (NT8 frame omitted reason); C# reason producer deferred to next AddOn wave`. The copy/F5/restart wording above is the explicit operator cutover disclosure, not a fabricated quote of a process boot that has not happened. No NT8 source copy, compile or restart is performed by this lane.

## Ordered cutover evidence

[A] Main was clean on `dev` at `171bebee`, then fast-forwarded under this lane's lock to `9c2ab980`. The full `go test ./...` suite passed at that merged HEAD. The approved code revision was rebuilt from clean clone `/tmp/nofx-placement-build/nofx`, with `vcs.revision=98f4ec6e499eb7dafb31e192460cb774c9a425df`, `vcs.modified=false`, SHA-256 `316ed32ba10031d26bbc7d6df5d6d3d6bd3db3e4678fd1f406c6bb94eb086c4b`. GUIDE uses the full revision extracted from that binary before building dist. Validation logs are under `/tmp/placement-cutover-98f4/`.

[A] Initial independent gate, `2026-09-08T00:08:12.688878-05:00`, HTTP 200, `ready=true`:

1. `db_open_positions`: `0 open row(s)`, source `sqlite trader_positions`.
2. `api_positions`: `0 position(s)`, source `trader.GetPositions`.
3. `nt8_positions_snapshot`: `count=0`, source `NT8 positions frame`.
4. `working_orders`: `0 working order(s) at the broker (ledger agrees: 0)`, source `broker — NT8 order_snapshot frame (age 17s, build 2026-09-07-h1)`.
5. `planner_in_flight`: `no planner read claimed`, source `plannerReadInFlight claim`.

[A] The gate's trailing `note` still claims no NT8 working-order frame exists; that note is stale. The actual leg-4 source above is the received h1 broker snapshot. This receipt is an initial check, not permission to reuse an aging gate at swap time.

## Postboot proof contract — pending, not a win

The owner receives the kill command only after RELEASE → binary rename → independent VERIFY. This lane does not execute that kill. The independent lock keeper remains active while the owner performs the restart. After a verified boot, record the first organically placed entry's signal id, arm id, command payload timestamp, measured receipt age and the received live entry frame that promotes `place_pending` to `working`. A local send, an absence of rejection, a ledger row alone and the loopback regression test are not that proof. If h1 does not expose an exact successful-signal age, report that measurement as unavailable rather than substitute a row timestamp or infer milliseconds from acceptance.

Four protective-order proofs remain event-dependent: entry's own OCO; filled entry's stop and target sharing a different OCO; both protective legs Gtc; cancel of an unfilled entry leaving an existing bracket untouched. Adopted g2 orders were Day, not GTC; their earlier lifecycle does not prove h1 GTC behavior.

## Source freshness

- `trader/ninjatrader/tcp_trader.go`: 6310eaf8 2026-09-07T23:51:51-05:00 merge: reconcile placement confirmation with pre-send identity and owner ruling
- `provider/ninjatrader/tcp_server.go`: 7e0c5527 2026-09-07T23:39:11-05:00 fix: install receipt routing before entries and preserve rejection evidence
- `provider/ninjatrader/tcp_framing.go`: 6262bf42 2026-09-07T23:34:58-05:00 fix: stamp entry creation time and await received placement truth
- `ninjascript/VLTraderTCPClient.cs`: b4195e6f 2026-09-07T10:53:37-05:00 fix(exit): class 80 — a REJECTED limit exit no longer cancels the position's bracket
- `kernel/clock_drift.go`: 7b19e753 2026-09-03T23:00:08-05:00 style(kernel): gofmt the 10 files that were unformatted on dev — formatting ONLY
- `kernel/clock_drift_test.go`: b95e6125 2026-08-18T14:27:26-05:00 fix(P0): zero-trades root cause — WSL clock drift was blocking every entry
- `trader/ninjatrader/placement_truth_test.go`: 6262bf42 2026-09-07T23:34:58-05:00 fix: stamp entry creation time and await received placement truth
- `docs/superpowers/AUDIT-CHECKLIST.md`: 7e0c5527 2026-09-07T23:39:11-05:00 fix: install receipt routing before entries and preserve rejection evidence

## Final preparation gate and RELEASE

[A] The gate held at 00:13:33 CT because a planner wake begun at 00:10:58 CT was still in flight. No binary or active dist swap occurred during the hold. At `2026-09-08T00:17:48.810467-05:00`, this lane received HTTP 200, `ready=true`, with all five legs passing:

1. `0 open row(s)` — source `sqlite trader_positions`.
2. `0 position(s)` — source `trader.GetPositions`.
3. `count=0` — source `NT8 positions frame`.
4. `0 working order(s) at the broker (ledger agrees: 0)` — source `broker — NT8 order_snapshot frame (age 23s, build 2026-09-07-h1)`.
5. `no planner read claimed` — source `plannerReadInFlight claim`.

[A] RELEASE is now prepared as `98f4ec6e` after the successful merged-head suite, clean-clone binary build, binary-derived full GUIDE revision and successful production dist build. The gate will be read again before the binary rename; boot and broker placement proofs remain pending.

## Staged handoff — owner kill not executed

[A] Final single-response gate at `2026-09-08T00:19:38.229436-05:00`: `ready=true`; all five legs passed.

1. `0 open row(s)`; source `sqlite trader_positions`.
2. `0 position(s)`; source `trader.GetPositions`.
3. `count=0`; source `NT8 positions frame`.
4. `0 working order(s) at the broker (ledger agrees: 0)`; source `broker — NT8 order_snapshot frame (age 12s, build 2026-09-07-h1)`.
5. `no planner read claimed`; source `plannerReadInFlight claim`.

[A] RELEASE → binary rename → independent VERIFY completed at `2026-09-08T00:19:49.043308-05:00`. RELEASE file, `HEAD:deploy/RELEASE`, disk binary and GUIDE all resolve to **98f4ec6e**; disk `vcs.modified=false`, dist byte-matched the build. Running `/proc/3201079/exe` and `/api/health` still resolve to **317388e7**, intentionally awaiting the owner command `kill -9 3201079`. The command was printed, not executed by this lane. No new boot or broker acceptance is claimed.

[A] Preparation correction: the first dist staging path was an untracked sibling of `web/dist`; the pre-swap clean-tree assertion stopped that attempt. The owned staging directory was moved to `/tmp/placement-cutover-98f4/dist.next`, main was verified clean, and the gate was re-read before any binary rename. Backup dist is outside the main tree.

[A] Binary backup: `/home/hoang/nofx/nofx-bin.old.317388e7.placement-98f4ec6e`. Dist backup: `/tmp/placement-cutover-98f4/dist.old.317388e7`. Independent keeper log: `/tmp/placement-cutover-98f4/keeper.log`; received-frame watcher: `/tmp/placement-cutover-98f4/watch.jsonl`; full machine receipt: `/tmp/placement-cutover-98f4/verify.json`. The lock stays held with fresh heartbeats through the owner handoff; postboot verification and the postboot marker push must precede release.
