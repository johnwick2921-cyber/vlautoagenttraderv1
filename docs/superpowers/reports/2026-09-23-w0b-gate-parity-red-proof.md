# W-EXEC-TRUTH W0b — the gate-parity matrix's RED proof

**Head:** `200b7e34953b0a5bec97d78d1b4bb5236abcab08` (branch `fix/exec-admission-gate-b`). **Script:** `scripts/w0b_gate_parity_mutations.py` (committed `200b7e34`). **Run:** 2026-09-23, sequential under the CTO's load ruling (`nice -n 19 ionice -c 3`, `-p 4`, `GOMAXPROCS=4`; one compile or one test binary at a time). The tree was clean after every restore (`git status --porcelain` empty, checked by the script after each mutation).

**What it proves [A].** `TestGateParityMatrix` drives the five producer entry points (`executeDecisionWithRecord`, `maybeManageArmedOrdersAt`, `runArmedPlacementAt`, `Evaluate` pre-claim, `Evaluate`→claim→`pictureHtfSend`) with one row per `<gate>@<path>`. For each mutation below ONE refusal is removed from production code; the matrix is re-run from a binary compiled against that mutation; the rows the table expects to go RED are compared with the rows that did. **29 of 30 CAUGHT with no expected row missing; 1 SURVIVED, and it is the expected one** — the arm send path's own hold pre-check is a redundant layer in front of `admitEntry`'s hold leg, which refuses with the same `maintenance_hold` class, so removing the pre-check ALONE changes nothing (removing both is mutation 27, CAUGHT).

**The one mutation with unexpected RED rows (27) [A].** With BOTH hold layers removed, the `B_arm_send` rig can no longer do what it relies on: it authors the arm under a maintenance hold so the arm rests unplaced, then hands the send point that pass's admitted set. Without any hold refusal the authoring pass sends, and every `B_arm_send` cell (and the G1 two-pass row `entry_gate@B_arm_pass`, which also parks its arm under a hold) fails its own fixture check — `gate_parity_matrix_test.go`: "fixture: the authoring pass under hold must not send". Those are fixture guards firing, not gates leaking; the rows the mutation targets (`maintenance_hold@A_decision/B_arm_pass/B_arm_send`) went RED as expected.

## Summary

| # | mutation | file(s) | expected RED rows | RED rows observed | result |
|---|---|---|---|---|---|
| 00 | trader_stopped | `entry_admission.go` | 2 | 2 | CAUGHT |
| 01 | day_plan_off | `entry_admission.go` | 2 | 2 | CAUGHT |
| 02 | feed_down | `entry_admission.go` | 5 | 5 | CAUGHT |
| 03 | dead_man | `entry_admission.go` | 5 | 5 | CAUGHT |
| 04 | frozen | `entry_admission.go` | 5 | 5 | CAUGHT |
| 05 | boot_integrity | `entry_admission.go` | 5 | 5 | CAUGHT |
| 06 | stop_until | `entry_admission.go` | 5 | 5 | CAUGHT |
| 07 | maintenance_hold | `entry_admission.go` | 1 | 1 | CAUGHT |
| 08 | contract_roll | `entry_admission.go` | 5 | 5 | CAUGHT |
| 09 | consecutive_loss(decision) | `entry_admission.go` | 1 | 1 | CAUGHT |
| 10 | session_risk(arm+picture) | `entry_admission.go` | 6 | 6 | CAUGHT |
| 11 | last_entry | `entry_admission.go` | 5 | 5 | CAUGHT |
| 12 | session_gate | `entry_admission.go` | 1 | 1 | CAUGHT |
| 13 | cme_closed | `entry_admission.go` | 3 | 3 | CAUGHT |
| 14 | plan_mode | `entry_admission.go` | 1 | 1 | CAUGHT |
| 15 | approval_required | `entry_admission.go` | 5 | 5 | CAUGHT |
| 16 | reentry_cooldown | `entry_admission.go` | 4 | 4 | CAUGHT |
| 17 | entry_gate(decision) | `entry_admission.go` | 1 | 1 | CAUGHT |
| 18 | entry_gate(picture) | `entry_admission.go` | 2 | 2 | CAUGHT |
| 19 | G1(arm_admission.go) | `arm_admission.go` | 1 | 1 | CAUGHT |
| 20 | arm admitEntry call (arm_admission.go) | `arm_admission.go` | 22 | 22 | CAUGHT |
| 21 | decision admitEntry call (auto_trader_orders.go) | `auto_trader_orders.go` | 13 | 13 | CAUGHT |
| 22 | picture pre-claim admitEntry call (picture_htf_evaluator.go) | `picture_htf_evaluator.go` | 15 | 15 | CAUGHT |
| 23 | picture send-time admitEntry call (picture_htf_send.go) | `picture_htf_send.go` | 14 | 14 | CAUGHT |
| 24 | LAYER arm pass-head session risk (armed_executor.go) alone | `armed_executor.go` | 2 | 2 | CAUGHT |
| 25 | LAYER arm pass-head + admitEntry session risk | `armed_executor.go, entry_admission.go` | 8 | 8 | CAUGHT |
| 26 | LAYER arm send hold precheck (armed_executor.go) alone | `armed_executor.go` | 0 | 0 | SURVIVED(expected: redundant layer) |
| 27 | LAYER arm send hold precheck + admitEntry hold | `armed_executor.go, entry_admission.go` | 3 | 17 | CAUGHT |
| 28 | LAYER picture pre-claim hold precheck + admitEntry hold | `entry_admission.go, picture_htf_evaluator.go` | 2 | 2 | CAUGHT |
| 29 | LAYER picture send hold precheck + admitEntry hold | `entry_admission.go, picture_htf_send.go` | 2 | 2 | CAUGHT |

## Per mutation

- **00 trader_stopped** — change: `trader/entry_admission.go: 'if !at.runningNow() {' -> 'if !at.runningNow() && false {'`
  - expected RED: trader_stopped@C_picture, trader_stopped@C_picture_send
  - observed RED: trader_stopped@C_picture, trader_stopped@C_picture_send
- **01 day_plan_off** — change: `trader/entry_admission.go: 'if !at.dayPlanEnabled() {' -> 'if !at.dayPlanEnabled() && false {'`
  - expected RED: day_plan_off@C_picture, day_plan_off@C_picture_send
  - observed RED: day_plan_off@C_picture, day_plan_off@C_picture_send
- **02 feed_down** — change: `trader/entry_admission.go: 'if down, status := at.ninjaFeedDown(); down {' -> 'if down, status := at.ninjaFeedDown(); down && false {'`
  - expected RED: feed_down@A_decision, feed_down@B_arm_pass, feed_down@B_arm_send, feed_down@C_picture, feed_down@C_picture_send
  - observed RED: feed_down@A_decision, feed_down@B_arm_pass, feed_down@B_arm_send, feed_down@C_picture, feed_down@C_picture_send
- **03 dead_man** — change: `trader/entry_admission.go: 'if at.deadMan.entriesBlocked() {' -> 'if at.deadMan.entriesBlocked() && false {'`
  - expected RED: dead_man@A_decision, dead_man@B_arm_pass, dead_man@B_arm_send, dead_man@C_picture, dead_man@C_picture_send
  - observed RED: dead_man@A_decision, dead_man@B_arm_pass, dead_man@B_arm_send, dead_man@C_picture, dead_man@C_picture_send
- **04 frozen** — change: `trader/entry_admission.go: 'if reason, frozen := discipline.IsFrozen(at.id); frozen {' -> 'if reason, frozen := discipline.IsFrozen(at.id); frozen && false {'`
  - expected RED: frozen@A_decision, frozen@B_arm_pass, frozen@B_arm_send, frozen@C_picture, frozen@C_picture_send
  - observed RED: frozen@A_decision, frozen@B_arm_pass, frozen@B_arm_send, frozen@C_picture, frozen@C_picture_send
- **05 boot_integrity** — change: `trader/entry_admission.go: 'if reason, refused := kernel.TradingRefused(); refused {' -> 'if reason, refused := kernel.TradingRefused(); refused && false {'`
  - expected RED: boot_integrity@A_decision, boot_integrity@B_arm_pass, boot_integrity@B_arm_send, boot_integrity@C_picture, boot_integrity@C_picture_send
  - observed RED: boot_integrity@A_decision, boot_integrity@B_arm_pass, boot_integrity@B_arm_send, boot_integrity@C_picture, boot_integrity@C_picture_send
- **06 stop_until** — change: `trader/entry_admission.go: 'if reason, paused := at.entryPausedAt(now); paused {' -> 'if reason, paused := at.entryPausedAt(now); paused && false {'`
  - expected RED: stop_until@A_decision, stop_until@B_arm_pass, stop_until@B_arm_send, stop_until@C_picture, stop_until@C_picture_send
  - observed RED: stop_until@A_decision, stop_until@B_arm_pass, stop_until@B_arm_send, stop_until@C_picture, stop_until@C_picture_send
- **07 maintenance_hold** — change: `trader/entry_admission.go: 'if reason, held := MaintenanceHeld(); held {' -> 'if reason, held := MaintenanceHeld(); held && false {'`
  - expected RED: maintenance_hold@A_decision
  - observed RED: maintenance_hold@A_decision
- **08 contract_roll** — change: `trader/entry_admission.go: 'if reason, blocked := at.entryBlockedByRoll(now); blocked {' -> 'if reason, blocked := at.entryBlockedByRoll(now); blocked && false {'`
  - expected RED: contract_roll@A_decision, contract_roll@B_arm_pass, contract_roll@B_arm_send, contract_roll@C_picture, contract_roll@C_picture_send
  - observed RED: contract_roll@A_decision, contract_roll@B_arm_pass, contract_roll@B_arm_send, contract_roll@C_picture, contract_roll@C_picture_send
- **09 consecutive_loss(decision)** — change: `trader/entry_admission.go: 'if reason, halted := at.consecutiveLossHaltedAt(now); halted {' -> 'if reason, halted := at.consecutiveLossHaltedAt(now); halted && false {'`
  - expected RED: consecutive_loss@A_decision
  - observed RED: consecutive_loss@A_decision
- **10 session_risk(arm+picture)** — change: `trader/entry_admission.go: 'if risk := at.sessionRiskGateAt(now); risk.Refuse {' -> 'if risk := at.sessionRiskGateAt(now); risk.Refuse && false {'`
  - expected RED: consecutive_loss@B_arm_send, consecutive_loss@C_picture, consecutive_loss@C_picture_send, no_trade_band@B_arm_send, no_trade_band@C_picture, no_trade_band@C_picture_send
  - observed RED: consecutive_loss@B_arm_send, consecutive_loss@C_picture, consecutive_loss@C_picture_send, no_trade_band@B_arm_send, no_trade_band@C_picture, no_trade_band@C_picture_send
- **11 last_entry** — change: `trader/entry_admission.go: 'if reason, blocked := at.entryBlockedByLastEntryAt(now); blocked {' -> 'if reason, blocked := at.entryBlockedByLastEntryAt(now); blocked && false {'`
  - expected RED: last_entry@A_decision, last_entry@B_arm_pass, last_entry@B_arm_send, last_entry@C_picture, last_entry@C_picture_send
  - observed RED: last_entry@A_decision, last_entry@B_arm_pass, last_entry@B_arm_send, last_entry@C_picture, last_entry@C_picture_send
- **12 session_gate** — change: `trader/entry_admission.go: 'if reason, blocked := at.sessionEntryBlockedAt(now); blocked {' -> 'if reason, blocked := at.sessionEntryBlockedAt(now); blocked && false {'`
  - expected RED: session_gate@A_decision
  - observed RED: session_gate@A_decision
- **13 cme_closed** — change: `trader/entry_admission.go: 'if closed, reason := kernel.CMEClosedReason(now); closed {' -> 'if closed, reason := kernel.CMEClosedReason(now); closed && false {'`
  - expected RED: cme_closed@B_arm_pass, cme_closed@B_arm_send, cme_closed@C_picture
  - observed RED: cme_closed@B_arm_pass, cme_closed@B_arm_send, cme_closed@C_picture
- **14 plan_mode** — change: `trader/entry_admission.go: 'if reason, blocked := at.planModeBlockedAt(in.Decision, now); blocked {' -> 'if reason, blocked := at.planModeBlockedAt(in.Decision, now); blocked && false {'`
  - expected RED: plan_mode@A_decision
  - observed RED: plan_mode@A_decision
- **15 approval_required** — change: `trader/entry_admission.go: 'if at.approvalRequired() && !at.approvalGranted(now) {' -> 'if at.approvalRequired() && !at.approvalGranted(now) && false {'`
  - expected RED: approval_required@A_decision, approval_required@B_arm_pass, approval_required@B_arm_send, approval_required@C_picture, approval_required@C_picture_send
  - observed RED: approval_required@A_decision, approval_required@B_arm_pass, approval_required@B_arm_send, approval_required@C_picture, approval_required@C_picture_send
- **16 reentry_cooldown** — change: `trader/entry_admission.go: 'in.Price, now.UnixMilli()); blocked {' -> 'in.Price, now.UnixMilli()); blocked && false {'`
  - expected RED: reentry_cooldown@B_arm_pass, reentry_cooldown@B_arm_send, reentry_cooldown@C_picture, reentry_cooldown@C_picture_send
  - observed RED: reentry_cooldown@B_arm_pass, reentry_cooldown@B_arm_send, reentry_cooldown@C_picture, reentry_cooldown@C_picture_send
- **17 entry_gate(decision)** — change: `trader/entry_admission.go: '\t\tif refused {\n\t\t\tif in.Record != nil {' -> '\t\tif refused && false {\n\t\t\tif in.Record != nil {'`
  - expected RED: entry_gate@A_decision
  - observed RED: entry_gate@A_decision
- **18 entry_gate(picture)** — change: `trader/entry_admission.go: 'if reason, refused := at.pictureEntryGate(in); refused {' -> 'if reason, refused := at.pictureEntryGate(in); refused && false {'`
  - expected RED: entry_gate@C_picture, entry_gate@C_picture_send
  - observed RED: entry_gate@C_picture, entry_gate@C_picture_send
- **19 G1(arm_admission.go)** — change: `trader/arm_admission.go: 'if admitted == nil || !admitted[armAdmitKey(r.PlanID, r.Scenario, r.LegIndex)] {' -> 'if false && (admitted == nil || !admitted[armAdmitKey(r.PlanID, r.Scenario, r.LegIndex)]) {'`
  - expected RED: entry_gate@B_arm_pass
  - observed RED: entry_gate@B_arm_pass
- **20 arm admitEntry call (arm_admission.go)** — change: `trader/arm_admission.go: '\t}); refused {\n\t\treturn false' -> '\t}); refused && false {\n\t\treturn false'`
  - expected RED: feed_down@B_arm_pass, dead_man@B_arm_pass, frozen@B_arm_pass, boot_integrity@B_arm_pass, stop_until@B_arm_pass, contract_roll@B_arm_pass, approval_required@B_arm_pass, last_entry@B_arm_pass, cme_closed@B_arm_pass, reentry_cooldown@B_arm_pass, feed_down@B_arm_send, dead_man@B_arm_send, frozen@B_arm_send, boot_integrity@B_arm_send, stop_until@B_arm_send, contract_roll@B_arm_send, approval_required@B_arm_send, consecutive_loss@B_arm_send, no_trade_band@B_arm_send, last_entry@B_arm_send, cme_closed@B_arm_send, reentry_cooldown@B_arm_send
  - observed RED: approval_required@B_arm_pass, approval_required@B_arm_send, boot_integrity@B_arm_pass, boot_integrity@B_arm_send, cme_closed@B_arm_pass, cme_closed@B_arm_send, consecutive_loss@B_arm_send, contract_roll@B_arm_pass, contract_roll@B_arm_send, dead_man@B_arm_pass, dead_man@B_arm_send, feed_down@B_arm_pass, feed_down@B_arm_send, frozen@B_arm_pass, frozen@B_arm_send, last_entry@B_arm_pass, last_entry@B_arm_send, no_trade_band@B_arm_send, reentry_cooldown@B_arm_pass, reentry_cooldown@B_arm_send, stop_until@B_arm_pass, stop_until@B_arm_send
- **21 decision admitEntry call (auto_trader_orders.go)** — change: `trader/auto_trader_orders.go: '\t\t}); refused {\n\t\t\tactionRecord.Success = false' -> '\t\t}); refused && false {\n\t\t\tactionRecord.Success = false'`
  - expected RED: feed_down@A_decision, dead_man@A_decision, frozen@A_decision, boot_integrity@A_decision, stop_until@A_decision, contract_roll@A_decision, approval_required@A_decision, maintenance_hold@A_decision, consecutive_loss@A_decision, last_entry@A_decision, session_gate@A_decision, plan_mode@A_decision, entry_gate@A_decision
  - observed RED: approval_required@A_decision, boot_integrity@A_decision, consecutive_loss@A_decision, contract_roll@A_decision, dead_man@A_decision, entry_gate@A_decision, feed_down@A_decision, frozen@A_decision, last_entry@A_decision, maintenance_hold@A_decision, plan_mode@A_decision, session_gate@A_decision, stop_until@A_decision
- **22 picture pre-claim admitEntry call (picture_htf_evaluator.go)** — change: `trader/picture_htf_evaluator.go: '\t}); refused {\n\t\treturn EvaluateResult{Stage: "watching", Reason: "admission refused: "' -> '\t}); refused && false {\n\t\treturn EvaluateResult{Stage: "watching", Reason: "admission refused: "'`
  - expected RED: trader_stopped@C_picture, day_plan_off@C_picture, feed_down@C_picture, dead_man@C_picture, frozen@C_picture, boot_integrity@C_picture, stop_until@C_picture, contract_roll@C_picture, approval_required@C_picture, consecutive_loss@C_picture, no_trade_band@C_picture, last_entry@C_picture, cme_closed@C_picture, reentry_cooldown@C_picture, entry_gate@C_picture
  - observed RED: approval_required@C_picture, boot_integrity@C_picture, cme_closed@C_picture, consecutive_loss@C_picture, contract_roll@C_picture, day_plan_off@C_picture, dead_man@C_picture, entry_gate@C_picture, feed_down@C_picture, frozen@C_picture, last_entry@C_picture, no_trade_band@C_picture, reentry_cooldown@C_picture, stop_until@C_picture, trader_stopped@C_picture
- **23 picture send-time admitEntry call (picture_htf_send.go)** — change: `trader/picture_htf_send.go: '\t}); refused {\n\t\treturn fmt.Errorf("picture_htf: send refused — %s", refusal)' -> '\t}); refused && false {\n\t\treturn fmt.Errorf("picture_htf: send refused — %s", refusal)'`
  - expected RED: trader_stopped@C_picture_send, day_plan_off@C_picture_send, feed_down@C_picture_send, dead_man@C_picture_send, frozen@C_picture_send, boot_integrity@C_picture_send, stop_until@C_picture_send, contract_roll@C_picture_send, approval_required@C_picture_send, consecutive_loss@C_picture_send, no_trade_band@C_picture_send, last_entry@C_picture_send, reentry_cooldown@C_picture_send, entry_gate@C_picture_send
  - observed RED: approval_required@C_picture_send, boot_integrity@C_picture_send, consecutive_loss@C_picture_send, contract_roll@C_picture_send, day_plan_off@C_picture_send, dead_man@C_picture_send, entry_gate@C_picture_send, feed_down@C_picture_send, frozen@C_picture_send, last_entry@C_picture_send, no_trade_band@C_picture_send, reentry_cooldown@C_picture_send, stop_until@C_picture_send, trader_stopped@C_picture_send
- **24 LAYER arm pass-head session risk (armed_executor.go) alone** — change: `trader/armed_executor.go: '\tif risk.Refuse {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+":session_risk", risk.Class) {' -> '\tif risk.Refuse && false {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+":session_risk", risk.Class) {'`
  - expected RED: consecutive_loss@B_arm_pass, no_trade_band@B_arm_pass
  - observed RED: consecutive_loss@B_arm_pass, no_trade_band@B_arm_pass
- **25 LAYER arm pass-head + admitEntry session risk** — change: `trader/armed_executor.go: '\tif risk.Refuse {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+":session_risk", risk.Class) {' -> '\tif risk.Refuse && false {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+":session_risk", risk.Class) {'`; `trader/entry_admission.go: 'if risk := at.sessionRiskGateAt(now); risk.Refuse {' -> 'if risk := at.sessionRiskGateAt(now); risk.Refuse && false {'`
  - expected RED: consecutive_loss@B_arm_pass, consecutive_loss@B_arm_send, consecutive_loss@C_picture, consecutive_loss@C_picture_send, no_trade_band@B_arm_pass, no_trade_band@B_arm_send, no_trade_band@C_picture, no_trade_band@C_picture_send
  - observed RED: consecutive_loss@B_arm_pass, consecutive_loss@B_arm_send, consecutive_loss@C_picture, consecutive_loss@C_picture_send, no_trade_band@B_arm_pass, no_trade_band@B_arm_send, no_trade_band@C_picture, no_trade_band@C_picture_send
- **26 LAYER arm send hold precheck (armed_executor.go) alone** — change: `trader/armed_executor.go: '\tholdReason, held := MaintenanceHeld()\n' -> '\tholdReason, held := MaintenanceHeld()\n\theld = held && false\n'`
  - expected RED: (none — redundant layer)
  - observed RED: (none)
- **27 LAYER arm send hold precheck + admitEntry hold** — change: `trader/armed_executor.go: '\tholdReason, held := MaintenanceHeld()\n' -> '\tholdReason, held := MaintenanceHeld()\n\theld = held && false\n'`; `trader/entry_admission.go: 'if reason, held := MaintenanceHeld(); held {' -> 'if reason, held := MaintenanceHeld(); held && false {'`
  - expected RED: maintenance_hold@A_decision, maintenance_hold@B_arm_pass, maintenance_hold@B_arm_send
  - observed RED: approval_required@B_arm_send, boot_integrity@B_arm_send, cme_closed@B_arm_send, consecutive_loss@B_arm_send, contract_roll@B_arm_send, dead_man@B_arm_send, entry_gate@B_arm_pass, feed_down@B_arm_send, frozen@B_arm_send, last_entry@B_arm_send, maintenance_hold@A_decision, maintenance_hold@B_arm_pass, maintenance_hold@B_arm_send, no_trade_band@B_arm_send, positive@B_arm_send, reentry_cooldown@B_arm_send, stop_until@B_arm_send
- **28 LAYER picture pre-claim hold precheck + admitEntry hold** — change: `trader/picture_htf_evaluator.go: '\tif reason, held := MaintenanceHeld(); held {\n\t\treturn e.refuseHeld(oppKey, reason, stall)' -> '\tif reason, held := MaintenanceHeld(); held && false {\n\t\treturn e.refuseHeld(oppKey, reason, stall)'`; `trader/entry_admission.go: 'if reason, held := MaintenanceHeld(); held {' -> 'if reason, held := MaintenanceHeld(); held && false {'`
  - expected RED: maintenance_hold@A_decision, maintenance_hold@C_picture
  - observed RED: maintenance_hold@A_decision, maintenance_hold@C_picture
- **29 LAYER picture send hold precheck + admitEntry hold** — change: `trader/picture_htf_send.go: '\tif reason, held := MaintenanceHeld(); held {\n\t\treturn fmt.Errorf("picture_htf: send refused — %s: %w", reason, ntTrader.ErrMaintenanceHold)' -> '\tif reason, held := MaintenanceHeld(); held && false {\n\t\treturn fmt.Errorf("picture_htf: send refused — %s: %w", reason, ntTrader.ErrMaintenanceHold)'`; `trader/entry_admission.go: 'if reason, held := MaintenanceHeld(); held {' -> 'if reason, held := MaintenanceHeld(); held && false {'`
  - expected RED: maintenance_hold@A_decision, maintenance_hold@C_picture_send
  - observed RED: maintenance_hold@A_decision, maintenance_hold@C_picture_send
