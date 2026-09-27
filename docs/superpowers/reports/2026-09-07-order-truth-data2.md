# ORDER-TRUTH — DATA-2

Implementation `ae9bd136d1d20df072d4a5d95c30e0beb08eb749`, branch `fix/order-truth-data2-0907`, after the separate LABEL wave `d79cc7acb76b79e7564b8c541091216c3122311a`. This is tested source, not a claim of deployment. No execution, placement, cancellation, account binding or database schema behavior changes.

Spec freshness: `738ea9b72925f0061a3f16a2cb5818f4ab86d153 | 2026-09-07T21:21:45-05:00 | docs: complete chart provenance and final order-truth verification`, latest change to `2026-09-07-order-truth.md` at acceptance. Verification uses `docs/superpowers/AUDIT-CHECKLIST.md`; existing identity/provenance and fabricated-value classes apply, with no new class allocated.

[A] The actual plan/today handler now selects rows whose Version equals the displayed plan version, then highest PlacementSeq per scenario/leg (ID breaks ties defensively). It does not choose the first historical row or prefer an older working placement over a newer terminal one. Version is last authorization touch; immutable ArmedUnderVersion is shown separately. Split legs retain individual states; a mixed summary does not claim both are working. A current-version scenario without a matching arm remains UNKNOWN, even if an older version reused the same S#.

[A] Each scenario leg renders three entry/stop/target columns: intended terms from the resolved displayed plan including overlays; composed terms from the selected armed_orders row; current ACCEPTED terms from the fresh received broker book. The composed column is explicitly ledger-sourced, not a claim of wire transmission: stop-entry offset/rounding can still make its entry differ from the actual broker trigger. The accepted entry reads the broker stop slot for stop entries and limit slot for limits. Filled/missing entries stay UNKNOWN while separately matched live protective terms can remain visible.

[A] The read-only broker projection captures one frame and receipt atomically, scoped to the trader-bound account and symbol; account identity is not exposed. Missing book, future receipt or age above 2×snapshot interval prevents acceptance. Matching uses the selected signal and exact -sl/-tp names; ambiguous names, non-live states or missing prices yield null, rendered UNKNOWN. There is no fallback to intended/composed prices or historical accepted_risk rows, which the audit found can contain dying-order records. Receipt date, age and received build accompany the terms. These columns do not certify full bracket protection or cancel safety.

[A] Validation: full `go test ./...` PASS at the combined LABEL + DATA-2 source; web 48 files / 364 tests PASS; `npm run build` PASS; precommit eslint/prettier and whitespace checks PASS. Regression tests cover the actual armedMapFor production method with a temporary SQLite ledger containing reused scenario versions, multiple placements, mixed split legs and newer terminal history; three differing price sources; stop-entry trigger slot; stale/future/missing and wrong-account/symbol book rejection; ambiguous and non-live orders; rendered columns and provenance even when scenario activation is unevaluable. Test fixtures never write the production database. One initial test fixture violated the unique placement constraint; corrected to use a newer sequence, without weakening that constraint.

## F2: owner controls copy, compile and restart

**Update after the owner restart:** h1 receipt is now confirmed (`match=yes`); see [the h1 wire-proof watch](2026-09-07-nt8-h1-wire-proofs.md). The two adopted g2 protections still carried `tif=Day` in h1 snapshots. They were not GTC and would not survive the trading-session close if still working. They subsequently settled at 22:28:53 CT (target filled, stop cancelled). The pre-restart evidence below is retained as history.

[A] At 21:44:16.879 CT, trace.20260907.00001.txt:20 says exactly `UserDataDir='C:\Users\hoang\Documents\NinjaTrader 8\'`. The loaded source is therefore its bin/Custom/AddOns/VLTraderTCPClient.cs. Line 55 declares `2026-09-05-g2`, MD5 `34efc3f85d0a775247f6c2f2ea576224`, 140520 bytes.

[A] h1 is present at `/home/hoang/nofx-oco/ninjascript/VLTraderTCPClient.cs` (fix/bracket-oco-separation worktree) and byte-identically at `/home/hoang/nofx/ninjascript/VLTraderTCPClient.cs`. Both line 55 declarations are `2026-09-07-h1`; MD5 `d0a604d79163f36557af89edc9f40777`, 157510 bytes. The older lane scratchpad file HEAD_VLTraderTCPClient.cs is actually g2 and is not the source to copy. Today's NT8 native log/trace search found no compile errors; the recompiled DLL embeds g2.

Owner was given this ONE WSL command, not executed by this agent. It creates a fresh build-named backup of both g2 source and current g2 DLL before replacing the loaded source. Full F5 compilation and NT8 restart remain owner actions.

```bash
(
  nt8_custom='/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom' &&
  nt8_backup=$(mktemp -d /home/hoang/nofx-backups/nt8-addon/2026-09-05-g2.XXXXXX) &&
  cp -p "$nt8_custom/AddOns/VLTraderTCPClient.cs" "$nt8_backup/VLTraderTCPClient.2026-09-05-g2.cs" &&
  cp -p "$nt8_custom/NinjaTrader.Custom.dll" "$nt8_backup/NinjaTrader.Custom.2026-09-05-g2.dll" &&
  cp -p /home/hoang/nofx-oco/ninjascript/VLTraderTCPClient.cs "$nt8_custom/AddOns/VLTraderTCPClient.cs"
)
```

[A] Last observed hello: received 2026-09-07T21:44:22.179663-05:00, protocol_version=3, source=vltrader-addon, build_id=2026-09-05-g2. h1 match=NO. Snapshot10928 received22:07:52.383 CT (emitted22:07:52.368) still carries g2 and two protective orders. Earlier flatness at21:56 was only a sample, and no longer describes this book.

[A] Passive order evidence for signal `953e8986-3496-45ae-a78d-ebd6d863177e`, all g2:

- Snapshot10915 received22:05:28.329 CT: entry Working, limit29664.5, no oco field; native log.20260907.00003.txt:595 confirms Oco=''.
- Native fill at22:05:41.337 CT (:597), then snapshot10923 received22:05:41.356 CT: stop29640 Accepted and target29721.25 Working, both OCO `953e8986-3496-45ae-a78d-ebd6d863177e-exit`.
- The wire omits TIF. Native log at22:05:41.449 CT (:604–606) explicitly shows Time in force=DAY for both protections. Gtc is not proven.
- No unfilled-entry cancellation preserving a bracket was observed. No agent-induced placement, cancellation or restart occurred.

At the original 22:14 CT closeout, h1 receipt and its four wire proofs were pending. The linked follow-up confirms h1 receipt; new-arm behavior proofs remain pending. Existing Go entry-only cancellation exemptions are not certified by this report.
