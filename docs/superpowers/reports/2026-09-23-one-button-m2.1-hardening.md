# W-ONE-BUTTON M2.1: hardening after the M2 adversarial review

**Lane:** Claude-101 · **Branch:** `feat/one-button-m2.1-hardening` · **Base:** the M2 head `18a157cf`. M2 itself later merged as #182 (`0e490e44`), and M2.1 has merged `origin/dev` since; no rebase. **Dispatch:** CTO rulings `1790139907454`, `1790140189916`, `1790141392189` (+ correction `1790141420995`), `1790142485221`.

**Rule (CTO):** M2.1 merges **before any hold is ever written** (by the CLI, by ops, or by M4) and **before the owner's F5** of the new AddOn.

## What changed, in the CTO's order

Every item was built RED first, or the behaviour was already correct and only the proof was missing. In every case the named mutation fails the test.

| Item | What | Commit | Mutation caught |
|---|---|---|---|
| review 3 F4 (false pass) | the gate's `addon_ack` FAILS the moment the AddOn disconnects: real server + TCPTrader, no stubbed view | `ad8e71f3` | M03 view forces Connected · M04 record always connected |
| review 3 F3 | a queue drop after the permit logs that the arm was **retired** (never sent), not "stays armed" | `1647100c` | wording reverted |
| review 2 F1 | the reconnect flush re-checks the hold **with the writer lock held** | `a326843d` | re-check removed |
| (e) / CTO first | drop sinks keyed by the owning trader id, so a reload replaces the sink instead of leaking it | `7ea82f87` | per-instance key · wrong owner wired |
| (a) | an enumerated empty census stays `[]` through `ConnectionRecord` (absent ≠ [] both ways) | `3a4dfae6` | old copy (RED run) |
| (b) | a registry-only non-NT8 trader is informational; a RUNNING one still fails `traders_nt8` | `3f512a75` | both directions |
| (e) | boot-line ack fields print `n/a`; a doc comment back on its function | `916d7dd0` | job not n/a |
| (d) Go | census `settled`; a transitional connection fails the gate | `4e4c858f` | — |
| (d) C# | census snapshots each NT8 collection under its OWN lock (never nested) · `settled` · `source_hash` at activation · VL_BUILD_ID `2026-09-23-m21` (lockstep) · PROTOCOL.md | `5830ac31` | 4 parity pins: reset removed · hash at first hello · nested lock · settled not written |
| review 3 F5 | cutover legs 2 and 4 reach the gate through the real seam | `bb51970d` | M05 leg 1 only |
| review 3 F6 | a stat failure other than not-exist HOLDS | `e404be93` | M10 stat error → not held |
| review 3 F7 | the stop path's hold refusal never cancels a sibling arm, proven by behaviour in the race | `98eafd78` | M08 `&& held` · refusal not reported |
| review 3 F8 | configured data dir + no hold file: behaviour unchanged (Picture submits, the planner reads) | `6a445311` | M14 / M14b refuse whenever configured |
| review 3 F9 | a hold landing mid-flush drops the batch's tail, reported and never attempted | `2acbc098` | M06 both in-loop checks |
| review 3 F10 | a never-sent Picture drop settles `refused`, naming the job | `aff2327a` | M15 Picture settle skipped |
| review 3 F11 | permit-before-B3 in all four entry functions | `56bca16c` | M07 PlaceLimitEntry permit after B3 |
| review 3 F12 | the permit re-reads a hold no reader has seen yet | `a6123065` | M09 permit without re-read |
| review 2 N4 + N5 | the CLI refuses an `--install-dir` without the bot's database (and names the path it expected), and refuses to run as root; `status` reports `db` and `db_present` | `f084532a` | each check removed |
| CTO tail | the desk strip's planner line no longer dereferences a reference level's nil formation close (live "line 10 panicked" on every scan) | `74adc540` | deref restored |
| review 2 N2 | a drop inside the entry's own send is `Own`, so there is no phantom-row ERROR + P1 | `55b99391` | Own never set · handler ignores Own · every drop treated as own |
| review F15 / N7 | a hold refusal never starts the flip / death re-read launch clock | `fcf1c90f` | flip · death early refusal off |
| review 3 F13 | the hold-writer scan now confines the PATH and fails on unparseable files. It found a production writer (`writeRaw` in `maintenance_gate.go`), now moved to a test file | `20045da7` | raw `os.Remove` of the hold path from api |
| review N1 / F16 | a reader whose read predates a hold never releases a barrier engaged after it (generation check) | `36d4dff7` | unconditional release |
| review 3 F17 | test hygiene: locked barrier reset; planner-hold dedupe cleared; the api test releases the barrier | `5b2db215` | — |
| L5 | Guide: the CLI's refusals, the transitional-connection leg, registry-only traders | `95c8e5e1` | Go guide pin + GuidePage suite |

## Notes on the evidence

- **Equivalent or coupled mutants, stated rather than counted.**
  - (a) `a := *rec.Ack` already shares the empty slice header, so skipping the copy for `[]` is equivalent. The real bug, the `append(nil, …)` reassignment, failed the RED run.
  - F9: the mid-flush test flips the hold on a call count, so removing only one in-loop check also fails it by shifting the count. That shows the tail is guarded, not that each check is independently needed.
- **The C# is compile-checked, not deployed.** All `ninjascript/*.cs` were built against the installed NinjaTrader.Core/Gui DLLs (dotnet net48, C# 7.3): 0 errors, 0 warnings. The harness rejects a bogus `ConnectionStatus` member, so `ConnectionStatus.Disconnected` is a real NT8 member.
- **Deferred, stated:**
  - `EntryBarrier.Hold(ctx)` has no production caller (M4 may use it); it stays tested.
  - CSV transport has no send permit (review 2 N6). The gate fails closed on CSV traders, and the live install runs TCP.
  - Under contention the trader package approaches the 10-minute default `go test` timeout.

## What was NOT done

- No merge, no deploy, no restart, no lock taken. `/home/hoang/nofx` untouched.
- The C# is not copied to AddOns, not compiled in NT8, and NT8 is not restarted; that is the owner's F5 step after M2.1 merges.
- GUIDE_BUILT_REV not bumped. No live DB writes.
