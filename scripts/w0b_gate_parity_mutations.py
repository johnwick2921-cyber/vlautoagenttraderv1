#!/usr/bin/env python3
"""W-EXEC-TRUTH W0b — the gate-parity matrix's RED proof (A21: the claim names
what it rests on).

For each mutation: remove ONE refusal from production code, compile the trader
test binary, restore the file with `git checkout --`, and verify the tree is
clean. Then run every binary's TestGateParityMatrix and compare the failing
<gate>@<path> rows with the rows the table expects to go RED.

    scripts/w0b_gate_parity_mutations.py <out-dir>      # out-dir OUTSIDE the repo

LOAD RULE (CTO 2026-09-23, while the bot is live): one process at a time,
nice -n 19 ionice -c 3, -p 4 / GOMAXPROCS=4, never a parallel fan-out — so
compiles AND runs are strictly sequential here. Refuses a dirty tree.
Output: <out-dir>/results.json + m<NN>.test.log per mutation; a summary line
per mutation on stdout (CAUGHT / SURVIVED / SURVIVED(expected: redundant layer)).
"""
import json, os, re, subprocess, sys

WT = subprocess.run(["git", "rev-parse", "--show-toplevel"], cwd=os.path.dirname(os.path.abspath(__file__)),
                    capture_output=True, text=True, check=True).stdout.strip()
if len(sys.argv) != 2:
    sys.exit("usage: w0b_gate_parity_mutations.py <out-dir outside the repo>")
OUT = os.path.abspath(sys.argv[1])
if OUT == WT or OUT.startswith(WT + os.sep):
    sys.exit("out-dir must be OUTSIDE the repo")
os.makedirs(OUT, exist_ok=True)
NICE = "nice -n 19 ionice -c 3"
ENV = dict(os.environ, GOMAXPROCS="4")

EA = "trader/entry_admission.go"
ALL = ["A_decision", "B_arm_pass", "B_arm_send", "C_picture", "C_picture_send"]

def rows(gate, paths):
    return [f"{gate}@{p}" for p in paths]

# rows the table marks via=admit, per path (for the call-site mutations)
ADMIT = {
 "A_decision": ["feed_down","dead_man","frozen","boot_integrity","stop_until","contract_roll","approval_required","maintenance_hold","consecutive_loss","last_entry","session_gate","plan_mode","entry_gate"],
 "B_arm_pass": ["feed_down","dead_man","frozen","boot_integrity","stop_until","contract_roll","approval_required","last_entry","cme_closed","reentry_cooldown"],
 "B_arm_send": ["feed_down","dead_man","frozen","boot_integrity","stop_until","contract_roll","approval_required","consecutive_loss","no_trade_band","last_entry","cme_closed","reentry_cooldown"],
 "C_picture": ["trader_stopped","day_plan_off","feed_down","dead_man","frozen","boot_integrity","stop_until","contract_roll","approval_required","consecutive_loss","no_trade_band","last_entry","cme_closed","reentry_cooldown","entry_gate"],
 "C_picture_send": ["trader_stopped","day_plan_off","feed_down","dead_man","frozen","boot_integrity","stop_until","contract_roll","approval_required","consecutive_loss","no_trade_band","last_entry","reentry_cooldown","entry_gate"],
}
def admit_rows(path):
    return [f"{g}@{path}" for g in ADMIT[path]]

M = [
 ("trader_stopped", EA, "if !at.runningNow() {", "if !at.runningNow() && false {", rows("trader_stopped", ["C_picture","C_picture_send"])),
 ("day_plan_off", EA, "if !at.dayPlanEnabled() {", "if !at.dayPlanEnabled() && false {", rows("day_plan_off", ["C_picture","C_picture_send"])),
 ("feed_down", EA, "if down, status := at.ninjaFeedDown(); down {", "if down, status := at.ninjaFeedDown(); down && false {", rows("feed_down", ALL)),
 ("dead_man", EA, "if at.deadMan.entriesBlocked() {", "if at.deadMan.entriesBlocked() && false {", rows("dead_man", ALL)),
 ("frozen", EA, "if reason, frozen := discipline.IsFrozen(at.id); frozen {", "if reason, frozen := discipline.IsFrozen(at.id); frozen && false {", rows("frozen", ALL)),
 ("boot_integrity", EA, "if reason, refused := kernel.TradingRefused(); refused {", "if reason, refused := kernel.TradingRefused(); refused && false {", rows("boot_integrity", ALL)),
 ("stop_until", EA, "if reason, paused := at.entryPausedAt(now); paused {", "if reason, paused := at.entryPausedAt(now); paused && false {", rows("stop_until", ALL)),
 ("maintenance_hold", EA, "if reason, held := MaintenanceHeld(); held {", "if reason, held := MaintenanceHeld(); held && false {", rows("maintenance_hold", ["A_decision"])),
 ("contract_roll", EA, "if reason, blocked := at.entryBlockedByRoll(now); blocked {", "if reason, blocked := at.entryBlockedByRoll(now); blocked && false {", rows("contract_roll", ALL)),
 ("consecutive_loss(decision)", EA, "if reason, halted := at.consecutiveLossHaltedAt(now); halted {", "if reason, halted := at.consecutiveLossHaltedAt(now); halted && false {", rows("consecutive_loss", ["A_decision"])),
 ("session_risk(arm+picture)", EA, "if risk := at.sessionRiskGateAt(now); risk.Refuse {", "if risk := at.sessionRiskGateAt(now); risk.Refuse && false {", rows("consecutive_loss", ["B_arm_send","C_picture","C_picture_send"]) + rows("no_trade_band", ["B_arm_send","C_picture","C_picture_send"])),
 ("last_entry", EA, "if reason, blocked := at.entryBlockedByLastEntryAt(now); blocked {", "if reason, blocked := at.entryBlockedByLastEntryAt(now); blocked && false {", rows("last_entry", ALL)),
 ("session_gate", EA, "if reason, blocked := at.sessionEntryBlockedAt(now); blocked {", "if reason, blocked := at.sessionEntryBlockedAt(now); blocked && false {", rows("session_gate", ["A_decision"])),
 ("cme_closed", EA, "if closed, reason := kernel.CMEClosedReason(now); closed {", "if closed, reason := kernel.CMEClosedReason(now); closed && false {", rows("cme_closed", ["B_arm_pass","B_arm_send","C_picture"])),
 ("plan_mode", EA, "if reason, blocked := at.planModeBlockedAt(in.Decision, now); blocked {", "if reason, blocked := at.planModeBlockedAt(in.Decision, now); blocked && false {", rows("plan_mode", ["A_decision"])),
 ("approval_required", EA, "if at.approvalRequired() && !at.approvalGranted(now) {", "if at.approvalRequired() && !at.approvalGranted(now) && false {", rows("approval_required", ALL)),
 ("reentry_cooldown", EA, "in.Price, now.UnixMilli()); blocked {", "in.Price, now.UnixMilli()); blocked && false {", rows("reentry_cooldown", ["B_arm_pass","B_arm_send","C_picture","C_picture_send"])),
 ("entry_gate(decision)", EA, "\t\tif refused {\n\t\t\tif in.Record != nil {", "\t\tif refused && false {\n\t\t\tif in.Record != nil {", rows("entry_gate", ["A_decision"])),
 ("entry_gate(picture)", EA, "if reason, refused := at.pictureEntryGate(in); refused {", "if reason, refused := at.pictureEntryGate(in); refused && false {", rows("entry_gate", ["C_picture","C_picture_send"])),
 # --- call-site / G1 mutations (outside entry_admission.go) ---
 ("G1(arm_admission.go)", "trader/arm_admission.go", "if admitted == nil || !admitted[armAdmitKey(r.PlanID, r.Scenario, r.LegIndex)] {", "if false && (admitted == nil || !admitted[armAdmitKey(r.PlanID, r.Scenario, r.LegIndex)]) {", rows("entry_gate", ["B_arm_pass"])),
 ("arm admitEntry call (arm_admission.go)", "trader/arm_admission.go", "\t}); refused {\n\t\treturn false", "\t}); refused && false {\n\t\treturn false", admit_rows("B_arm_pass") + admit_rows("B_arm_send")),
 ("decision admitEntry call (auto_trader_orders.go)", "trader/auto_trader_orders.go", "\t\t}); refused {\n\t\t\tactionRecord.Success = false", "\t\t}); refused && false {\n\t\t\tactionRecord.Success = false", admit_rows("A_decision")),
 ("picture pre-claim admitEntry call (picture_htf_evaluator.go)", "trader/picture_htf_evaluator.go", "\t}); refused {\n\t\treturn EvaluateResult{Stage: \"watching\", Reason: \"admission refused: \"", "\t}); refused && false {\n\t\treturn EvaluateResult{Stage: \"watching\", Reason: \"admission refused: \"", admit_rows("C_picture")),
 ("picture send-time admitEntry call (picture_htf_send.go)", "trader/picture_htf_send.go", "\t}); refused {\n\t\treturn fmt.Errorf(\"picture_htf: send refused — %s\", refusal)", "\t}); refused && false {\n\t\treturn fmt.Errorf(\"picture_htf: send refused — %s\", refusal)", admit_rows("C_picture_send")),
]

EA_HOLD = (EA, "if reason, held := MaintenanceHeld(); held {", "if reason, held := MaintenanceHeld(); held && false {")
EA_RISK = (EA, "if risk := at.sessionRiskGateAt(now); risk.Refuse {", "if risk := at.sessionRiskGateAt(now); risk.Refuse && false {")
ARM_HEAD = ("trader/armed_executor.go", "\tif risk.Refuse {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+\":session_risk\", risk.Class) {", "\tif risk.Refuse && false {\n\t\tif armRefusalChanged(&at.armRefusalLast, at.id+\":session_risk\", risk.Class) {")
ARM_HOLD = ("trader/armed_executor.go", "\tholdReason, held := MaintenanceHeld()\n", "\tholdReason, held := MaintenanceHeld()\n\theld = held && false\n")
PIC_HOLD = ("trader/picture_htf_evaluator.go", "\tif reason, held := MaintenanceHeld(); held {\n\t\treturn e.refuseHeld(oppKey, reason, stall)", "\tif reason, held := MaintenanceHeld(); held && false {\n\t\treturn e.refuseHeld(oppKey, reason, stall)")
SEND_HOLD = ("trader/picture_htf_send.go", "\tif reason, held := MaintenanceHeld(); held {\n\t\treturn fmt.Errorf(\"picture_htf: send refused — %s: %w\", reason, ntTrader.ErrMaintenanceHold)", "\tif reason, held := MaintenanceHeld(); held && false {\n\t\treturn fmt.Errorf(\"picture_htf: send refused — %s: %w\", reason, ntTrader.ErrMaintenanceHold)")
M += [
 ("LAYER arm pass-head session risk (armed_executor.go) alone", [ARM_HEAD], None, None, rows("consecutive_loss", ["B_arm_pass"]) + rows("no_trade_band", ["B_arm_pass"])),
 ("LAYER arm pass-head + admitEntry session risk", [ARM_HEAD, EA_RISK], None, None, rows("consecutive_loss", ["B_arm_pass","B_arm_send","C_picture","C_picture_send"]) + rows("no_trade_band", ["B_arm_pass","B_arm_send","C_picture","C_picture_send"])),
 ("LAYER arm send hold precheck (armed_executor.go) alone", [ARM_HOLD], None, None, []),
 ("LAYER arm send hold precheck + admitEntry hold", [ARM_HOLD, EA_HOLD], None, None, rows("maintenance_hold", ["A_decision","B_arm_pass","B_arm_send"])),
 ("LAYER picture pre-claim hold precheck + admitEntry hold", [PIC_HOLD, EA_HOLD], None, None, rows("maintenance_hold", ["A_decision","C_picture"])),
 ("LAYER picture send hold precheck + admitEntry hold", [SEND_HOLD, EA_HOLD], None, None, rows("maintenance_hold", ["A_decision","C_picture_send"])),
]

def sh(cmd, **kw):
    return subprocess.run(cmd, cwd=WT, shell=True, capture_output=True, text=True, env=ENV, **kw)

def edits_of(m):
    name, f, old, new, exp = m
    if isinstance(f, list):
        return name, f, exp
    return name, [(f, old, new)], exp

def compile_all():
    res = []
    for i, m in enumerate(M):
        name, edits, exp = edits_of(m)
        files = sorted({f for f, _, _ in edits})
        bad = None
        for f, old, new in edits:
            p = os.path.join(WT, f)
            src = open(p).read()
            n = src.count(old)
            if n != 1:
                bad = f"anchor count {n} in {f}"; break
            open(p, "w").write(src.replace(old, new))
        if bad:
            sh("git checkout -- " + " ".join(files))
            res.append({"name": name, "error": bad}); continue
        diff = sh("git diff --stat -- " + " ".join(files)).stdout.strip()
        b = os.path.join(OUT, f"m{i:02d}.test")
        c = sh(f"{NICE} go test -p 4 -c -o {b} ./trader/")
        restore = sh("git checkout -- " + " ".join(files))
        porcelain = sh("git status --porcelain").stdout.strip()
        res.append({"name": name, "files": files, "change": [f"{f}: {old!r} -> {new!r}" for f, old, new in edits], "diffstat": diff,
                    "compile_rc": c.returncode, "compile_err": c.stderr[-2000:], "bin": b,
                    "restored_rc": restore.returncode, "porcelain_after_restore": porcelain, "expected": exp})
        if porcelain:
            print("TREE NOT CLEAN after", name, porcelain, file=sys.stderr); sys.exit(2)
        print("compiled", i, name, "rc", c.returncode, "restored", restore.returncode, "porcelain", repr(porcelain), flush=True)
    return res

def run_one(r):
    if r.get("error") or r["compile_rc"] != 0:
        return r
    p = subprocess.run(f"{NICE} {r['bin']} -test.run '^TestGateParityMatrix$' -test.count=1 -test.v -test.timeout=20m",
                       cwd=os.path.join(WT, "trader"), shell=True, capture_output=True, text=True, env=ENV)
    fails = sorted(set(re.findall(r"--- FAIL: TestGateParityMatrix/(\S+)", p.stdout)))
    log = os.path.join(OUT, os.path.basename(r["bin"]) + ".log")
    open(log, "w").write(p.stdout + p.stderr)
    exp = sorted(set(r["expected"]))
    r.update({"run_rc": p.returncode, "failed_rows": fails,
              "missing_red": sorted(set(exp) - set(fails)),
              "unexpected_red": sorted(set(fails) - set(exp)),
              "result": ("CAUGHT" if set(exp) <= set(fails) else "SURVIVED") if exp else ("SURVIVED(expected: redundant layer)" if not fails else "CAUGHT(unexpected)"),
              "log": log})
    return r

if __name__ == "__main__":
    if sh("git status --porcelain").stdout.strip():
        sys.exit("refusing: the working tree is dirty (a mutation run restores files with git checkout --)")
    head = sh("git rev-parse HEAD").stdout.strip()
    print("HEAD", head, flush=True)
    compiled = compile_all()
    done = [run_one(r) for r in compiled]  # sequential: the load rule
    json.dump({"head": head, "results": done}, open(os.path.join(OUT, "results.json"), "w"), indent=1)
    for r in done:
        print(f"{r.get('result','ERR'):9s} {r['name']:60s} missing={r.get('missing_red')} unexpected={r.get('unexpected_red')}")
