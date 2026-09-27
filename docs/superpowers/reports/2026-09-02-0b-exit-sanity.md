# 0B — EXIT SANITY + RE-ARM AFTER BOOT SWEEP

Date: 2026-09-02 · Owner: hoang · Agent: Opus 5 (1M) · Worktree: `../nofx-0b` (branch `fix/0b-exit-sanity`) · Checklist: **CLASS 43**
Evidence tiers: **[A]** directly verified · **[B]** inferred from strong evidence · **[C]** speculation.

## STATUS

| Item | State |
|---|---|
| Code | **MERGED to dev** — `4657560b` (code) · `7c968238` (guide) · `4175e0b6` (owner rulings) · `34532c1e` (marker) |
| Build | clean clone `--no-local` at `4175e0b6`, `vcs.modified=false`, built 2026-09-02T12:44:37Z, sha256 `bf4ae24b55028f73` |
| Suites | Go **27 ok / 0 FAIL** · goldens PASS · vitest **38 files / 298 tests** · tsc clean |
| Cutover | **DONE 07:49:01 CT** on owner GO **with an explicit A7 override** (open position). Boot 07:49:06 `BOOT INTEGRITY OK — rev 4175e0b62de7 · goldens PASS`, PID 2674837, **0 ERRO** |
| Marker | `34532c1e` written AFTER the passed boot (A19); **RELEASE set BEFORE the swap** — the lesson from the P&L-truth wave |
| Sequencing | ROOT-FIX merged and booted first (rev `0d093c3b`, 07:32:15 CT); 0B rebased onto its marker, re-ran the suite, took the lock, booted on its own. Two lanes, two boots, two markers |
| Footprint (B) | zero edits to `kernel/planner_prompt.go`, `kernel/plan_doc.go`, `trader/auto_trader_planner.go`, `mcp/client.go`, `store/planner_rejected.go` — the min_sl authoring-WARN change stays DEFERRED |

---

## RESOLVED VALUES — BEFORE → AFTER (A11)

Every value read from its resolver, never a file literal.

| knob | resolver | before | after |
|---|---|---|---|
| min stop width | `kernel.MinSLATRMult()` (env `MIN_SL_ATR_MULT` unset) | **1.0×ATR5m** | **1.5×ATR5m** |
| dead-zone bound | `armStopAnchorMaxATR()` (env `ARM_STOP_ANCHOR_MAX_ATR` unset) | n/a (no anchoring) | **3.0×ATR5m** `[I] PROVISIONAL` |
| order size | `kernel.ResolveMaxContracts(0, 2)` | **2** | **1** (Stage-A ceiling) |
| arm-leg capacity | `splitLegCapacity(0)` | 1 | 1 (unchanged) |
| breakeven | `breakevenTrigger` + `exitMechsSuspended()` | firing (2 moves on 09-01) | **suspended** |
| ATR trail | `trailingConfig` + `exitMechsSuspended()` | firing (8 ratchets on 09-01) | **suspended** |
| R:R floor | `armMinRR()` (env `ARM_MIN_RR` unset) | **2.0** | **2.0 — VERIFIED, UNCHANGED (D7)** |

The size contradiction in C4 is confirmed [A]: the live strategy carries **no** `max_contracts_per_order`, so arm-leg capacity resolved 1 while order sizing resolved 2 through `maxFuturesContracts` — the boot line said capacity=1 while a market entry could size 2.

---

## FILES AND LINES TOUCHED (A17)

| file | line | change |
|---|---|---|
| `kernel/min_sl.go` | 34 | `MinSLATRMultDefault` 1.0 → **1.5**, with the citation and the note that THREE gates read it |
| `trader/arm_stop_anchor.go` | 35 / 38 | `ArmStopAnchorMaxATRDefault = 3.0` · `armStopAnchorMaxATR()` |
| | 71 | `composeArmStop` — the pure composition |
| | 154 | `armStopCompositionLine` — the per-arm log line |
| `trader/armed_executor.go` | 341 | composition wired at the head of the arm leg loop, before every downstream consumer |
| | 351 | `store.IncStopUnanchored` + the 🛑 dead-zone line with the running n |
| | 415 | `store.IncArmRefusal` + the running per-session refusal count on the refusal line |
| `trader/exit_mechs_suspend.go` | 35 / 49 / 61 / 77 / 95 | `exitMechsSuspended()` · `moveStopWire` (the wire seam) · `exitMechSuspendedRefuse` · `ExitPolicyBootLine` · `…Live` |
| `trader/auto_trader.go` | 157 / 176 | BE guard before any broker resolution; the send goes through `moveStopWire` |
| `trader/auto_trader_trailing.go` | 180 / 189 | trail guard hoisted ABOVE the broker resolution; the send goes through `moveStopWire` |
| `kernel/risk_limits.go` | 283 / 299 | `ResolveMaxContracts` ends in `ClampStageAContracts`; `StageAContractCapDefault = 1` |
| `store/armed_orders.go` | 181 / 187-188 / 274 | the same-version early return now exempts boot-swept rows; the loud ⚖ re-arm line; `BootSweepReasonPrefix` |
| `store/zerob_counters.go` | 24 / 28 / 47 | `StopUnanchoredKey` · `StopUnanchoredReviewN = 30` · `ArmRefusalKey` |
| `trader/class33_boot_sweep.go` | — | the sweep reason is now built FROM the prefix constant, so the contract cannot drift |
| `main.go` | 267 | the 🛑 boot line |
| guide | `settings.ts` (3 cards + 1 new) · `status.ts` · `faq.ts` (2 entries) · census 44 → 46 | |
| `docs/superpowers/AUDIT-CHECKLIST.md` | 589 | **CLASS 43** (highest at merge time was 42, appended by ROOT-FIX) |

**Cross-effect, stated plainly:** raising `MinSLATRMultDefault` also tightens the AI-entry decision gate (`kernel/engine_position.go`) and the planner's authoring WARNING (`trader/auto_trader_planner.go`, ROOT-FIX's file — read only, not edited). One floor everywhere is the intent.

---

## THE RULE (D2), exactly as implemented

```
stop = WIDEST OF:
        authored stop            (never tightened)
        anchor  = nearest seated level on the RISK side ± 2 ticks clearance
        floor   = entry ∓ 1.5 × ATR5m
anchor is skipped when the nearest risk-side level is further than 3.0 × ATR5m
        → stop_unanchored, the floor governs, and a level is NEVER invented
```

Per-arm line: `🛑 arm stop <session> <S#> leg <n> <side>: stop X (authored Y WIDENED) · anchor <label> P → beyond Q · atr_floor F (1.5×ATR5m A) · bound=anchor|atr_floor|authored`.

---

## TESTS (A8 — written before the change; each quoted)

**E1 · `TestZeroBPinStopFloor`** (`trader/zerob_exit_sanity_test.go:49`) — uses only pre-0B surfaces so it compiles on the old tree. **RED on `928e49d2`:**
```
--- FAIL: TestZeroBPinStopFloor (0.00s)
    zerob_exit_sanity_test.go:51: 0B: the resolved min-SL floor is 1.00×ATR5m, want 1.5
        (C1 owner ruling; env MIN_SL_ATR_MULT unset)
```
**GREEN on `4175e0b6`** — and a 1.2×ATR stop that the old floor accepted is now refused with `too close`, while 1.6×ATR still passes.

**E5 · `TestZeroBReArmAfterBootSweep`** (`:257`) — two pre-boot WORKING arms + one owner-cancelled row at the same plan version. The sweep cancels **both old broker orders at the wire** (recorder: `sig-old-1`, `sig-old-3`) and stamps the `boot_sweep` reason; re-authoring at the SAME version brings both back as `armed` with **empty signal id and empty reason** (fresh identity), while the owner's row **stays cancelled with its signal intact**; a NEW plan version still re-authorizes the owner's row. On the pre-0B tree the swept rows stay terminal — that is the C5 defect. PASS.

**E2** anchor/dead-zone/short-mirror/never-tighten/log-line (`:77`) · **E3** `TestZeroBSuspendedMechanismsNeverReachTheWire` (`:166`): the BE trigger fires (+60 on a 40-pt trigger) and the trail ratchet emits, and **zero move_stop frames reach `moveStopWire`**; `EXIT_MECHS_SUSPENDED=0` restores the send, proving suspended-not-deleted · **E3b** source pin (`:210`): neither file may call the broker directly · **E4** size (`:231`) · **E6** the R:R gate refuses R:R 1.0 after the widening and `armMinRR()` still resolves 2.0 (D7) · **counters** (`:370`): durability, class separation, session isolation, malformed → 0. All PASS.

**E7 / D6 — the corrected-column lint already exists.** Class 40 shipped `store/pnl_surface_guard_test.go:87`, scanning `{".", "../api", "../trader", "../agent", "../kernel"}` — the same five roots this dispatch asks for, with the writer/backfill allow-list and a self-test that fails on a deliberate raw aggregation. **Confirmed and NOT duplicated:** `TestPnlSurfaceGuardNoRawAggregation` PASS, `TestPnlSurfaceGuardCatchesRawAggregation` PASS.

**Three pre-existing pins updated** (they asserted the exact values 0B changes, and their intent was preserved, not weakened): `kernel/guardrails_test.go` verifies the precedence with the ceiling raised, then pins the ceiling itself; `kernel/min_sl_gate_test.go` widens its fixture to the new floor and ADDS a pin that the old 1.05×ATR width is now refused; `trader/caps_always_on_test.go` keeps its always-on intent at the Stage-A value.

---

## OWNER RULINGS (2026-09-02), as implemented

**1. The 3.0×ATR dead-zone bound is PROVISIONAL `[I]`, not a ruling on the number.** The Guide card carries `[R researched]` on the 1.5 floor and `[I] PROVISIONAL` on the 3.0 bound with the review trigger. Because class 35's law is that counters record events, the review is backed by a durable count: `store.IncStopUnanchored` (`arm_stop_unanchored_0b`), and every dead zone logs `🛑 stop_unanchored … Recorded n=<N> (provisional bound reviewed at n≥30)`.

**2. More arm refusals at R:R 2.0 is the intended trade — record the cost.** `store.IncArmRefusal` records per `(trader, session-day, session, class)`. **Definition, so the number can be quoted honestly:** one bump per DISTINCT refused arm-spec (deduped by plan:version:scenario:leg), never per cycle spent re-refusing the same arm. The refusal line prints the running count: `⚔️ arm REFUSED … · rr refusals this session: N`. Both counters read 0 on a malformed row — never a fabricated figure.

---

## CUTOVER (A5/A6/A7) — with an explicit owner override

The five-leg gate at **07:37 CT** read `ready:false` on leg 5 (a LONDON planner read in flight); legs 1-4 passed. The read landed and the bot **opened position 588** (LONG 1 @ 29082.50, cited S2, plan v3, 07:41:05 CT), so legs 1-3 then failed on an open position. The owner overrode the wait.

**Gate quoted at the swap, 07:49:01 CT [A]:**
```
ready: False
  leg 1 db_open_positions      FAIL | 1 open row(s)
  leg 2 api_positions          FAIL | 1 position(s)
  leg 3 nt8_positions_snapshot FAIL | count=1
  leg 4 working_orders         PASS | 0 non-terminal arm(s)
  leg 5 planner_in_flight      PASS | no planner read claimed
A6/A7: session LONDON v4 · replan_in_flight false · armed {}
```

**The exposure I could not close [A].** Position 588 was a MARKET entry; its protective bracket is created by the NT8 AddOn and lives at the broker, so it survives a Go restart by design. I could not OBSERVE it: `/api/open-orders?symbol=MNQ` returns `[]` and there is no NT8 order-frame source on the Go side — leg 4's own note says `(no NT8 order frame — F12 open)`. So the stop is **asserted by design, not verified**. The real cost of the override was a ~5-second unwatched window (no drawdown monitor, no reconcile). The position reconciled cleanly after boot: API and DB both show 588 open at the same entry, unrealized −51.50 at 07:49:34.

**Boot checklist, 07:49:06 CT — every line [A]:**
```
🔐 BOOT INTEGRITY OK — rev 4175e0b62de7 · built 2026-09-02T12:44:37Z · expected 4175e0b62de7 · goldens PASS
🛑 exits: stop=max(anchor+clr, 1.5×ATR5m) · anchor_max=3.0×ATR5m · BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)   ← NEW
🪢 netting-orphan (class 27) · 🧪 validator hints 15 sites (class 34+38) · 📜 prompt/validator 17 restrictions (class 38)
⚖ arm normalizer (class 39) · ✂ planner schema 9 fields (class 42) · 🛡 cutover safety legs=5 (class 33) · 🔬 shadow A/B OFF
🧮 replan budget: recorded-counter (class 35) · 🗓 preflight: halt/weekend bypass (class 36) · 🧾 P&L surfaces 12 strict (class 40)
```
One PID (2674837), `vcs.revision=4175e0b6`, **0 ERRO**, position-reconcile and the drawdown monitor both started.

**Rollback:** `cd /home/hoang/nofx && mv nofx-bin nofx-bin.bad.4175e0b6 && cp nofx-bin.prev.boot nofx-bin && printf '0d093c3b3a11fb6ea6cb19454ffa59a9f7bd9f8b' > deploy/RELEASE && kill -9 $(pgrep -f '^/home/hoang/nofx/nofx-bin$')` — `nofx-bin.prev.boot` and `nofx-bin.old.0d093c3b` are the ROOT-FIX binary. (Set RELEASE to whatever `git show 7e7556b9:deploy/RELEASE` holds if the literal above is stale.)

---

## PROOF (F) — what is proven and what is still owed (A20)

**Proven live:** the boot checklist above, the surviving class lines, zero errors, and the open position reconciling across the restart.

**NOT yet observed — owed:**
1. **The first arm authored after boot**, with its entry, anchor level, ATR floor, chosen stop, binding side and R:R. No arm has been authored since 07:49; a watcher is running for the first `🛑 arm stop` line.
2. **The first BE or trail trigger**, proving no `move_stop` frame is sent. Position 588 is open, so a BE trigger is plausible today; the fixture proves the wire is untouched, the live line will confirm it.
3. **The sweep → re-arm sequence** (D5), which needs a cutover with a resting arm. This cutover had `0 non-terminal arm(s)`, so it could not be shown.

Until those land, E1/E2/E5 are the fixture proof and this report says so.

### PROOF LANDED — first arms composed live, 08:03:06 CT [A]

```
🛑 arm stop LONDON S3 leg 1 long: stop 28926.75 (authored 28927.25 WIDENED) ·
   anchor ONL 28927.25 → beyond 28926.75 · atr_floor n/a (no ATR) · bound=anchor
🛑 arm stop LONDON S1 leg 1 long: stop 28975.16 (authored 28985.50 WIDENED) ·
   anchor RTH-L 29001.75 → beyond 29001.25 · atr_floor 28975.16 (1.5×ATR5m 26.02) · bound=atr_floor
```

Both legs of D2 are demonstrated on the live tape within 14 minutes of the boot:

| arm | authored | anchor | ATR floor | chosen | bound | moved |
|---|---|---|---|---|---|---|
| S3 long | 28927.25 | ONL 28927.25 → beyond 28926.75 (2-tick clearance) | n/a (ATR unavailable that cycle) | **28926.75** | **anchor** | 0.50 |
| S1 long | 28985.50 (28.69 pts) | RTH-L 29001.75 → beyond 29001.25 | 28975.16 (1.5 × 26.02 = 39.03 pts) | **28975.16** | **atr_floor** | **10.34** |

S1 is the wave's thesis in one line: the planner authored a 28.69-point stop that the OLD 1.0× floor would have ACCEPTED (26.02 pts required) and the new floor rejects — so the stop was widened to 39.03 points rather than the trade being refused, and the anchor sat on the wrong side of the entry to help. S3 shows the anchor binding with exactly the 2-tick clearance.

**The cross-effect is visible in the same tape** (the planner's authoring WARNING reads the same resolver — ROOT-FIX's file, read-only):

```
07:0x (old binary) ⚔️ arm feasibility: S1 arm stop 29040.25 too close (19.00 < 19.44 = 1.0×ATR5m)
08:01 (new binary) ⚔️ arm feasibility: S1 arm stop 28985.50 too close (28.69 < 38.46 = 1.5×ATR5m)
```

**The A7-override exposure is now CLOSED BY EVIDENCE [A].** I could not observe position 588's protective stop before the swap. It fired 2 min 32 s after the boot:

```
07:51:38 📕 NT position closed: MNQ LONG qty=1.00 exit=29050.00 reason=sl pnl=-65.00
```

Row 588: entry 29082.50 → exit 29050.00, `pnl_corrected` **−65.00**, exit 07:51:38 CT. The bracket was resting at the broker, survived the restart, executed on its own, and the Go process recorded the close. The stop was real; I simply had no way to see it beforehand — that is the F12 gap, and it is still worth closing.

**Counters at 08:05 CT:** `arm_stop_unanchored_0b` and `arm_refusals_0b:*` are both **absent (= 0)** — no dead zone and no refusal yet. Stated as a zero with its meaning, not as a result.

**Still owed:** a BE or trail trigger proving no `move_stop` frame (588 closed at a loss, so breakeven never armed), and a sweep → re-arm sequence (that needs a cutover with a resting arm).

### One honest finding from the live lines (A15)

S3 composed with **`atr_floor n/a (no ATR)`**. When ATR5m is unavailable on a cycle the floor leg is skipped (documented fail-open) — and the arm gate's min-SL leg is skipped on the same condition (`if atr5m > 0`). So on an ATR-less cycle a tight authored stop receives only the anchor's 2 ticks and nothing enforces the floor. This is PRE-EXISTING fail-open behaviour that 0B inherits rather than introduces, but it means the floor is only as strong as ATR availability. It deserves a follow-up ruling: refuse the arm when ATR is unavailable, or accept the gap explicitly.

---

## A15 — what the owner will still see wrong

- **Position 588 is running under the OLD stop rules.** Its stop was authored before the swap; 0B composes stops at ARM time, so nothing retro-fits an open trade. Its BE and trail are now suspended, so its stop will no longer move at all — that is the intended change, but it means the trade now runs to its original stop, target, or the EOD flat.
- **More arm refusals are coming**, by design. The count is recorded per session and printed on each refusal line.
- **The Guide drift banner** clears at this boot (`GUIDE_BUILT_REV` = the running rev), but the vite dev server serves the main tree, so the new cards appear only after the tree is at `4175e0b6` — it is.
- **The 3.0×ATR bound is `[I]`**, not researched. If dead zones are common, the ATR floor will be doing the work rather than structure — the recorded n is the signal to revisit.
- **Leg 4 still reads the ledger, not NT8** (F12 open). A broker order with no ledger row remains invisible to the gate; that is the same gap that left me unable to verify 588's stop.
- **`ResolveMaxContracts` is now clamped globally**, including for any non-futures path that calls it. In this SIM-only, MNQ-only deployment that is a no-op, but the clamp is not futures-scoped.

## Closeout

Commits: `4657560b` · `7c968238` · `4175e0b6` · `34532c1e` (marker) · this report. Lock released, worktree `../nofx-0b` removed, repo memory updated.
