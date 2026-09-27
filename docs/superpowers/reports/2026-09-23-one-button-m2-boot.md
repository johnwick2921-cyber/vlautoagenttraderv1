# W-ONE-BUTTON M2 boot — 2026-09-23 00:46 CT (CTO, owner GO, SIM)

- **Binary**: `0e490e44827995be0f0da4c01617e62ebbfcb351` (merge of PR #182), built from a
  clean clone, `vcs.modified=false`, md5 `a119201d927e378f858f97b23b52ce36`.
- **Proof before the kill**: independent `go build ./... && go vet ./...` exit 0 and
  `go test -race ./...` 37 ok / 0 fail / 0 race / exit 0 at c4823d77 (tree-identical to the
  merge). Lane's own run at c4823d77 matched. Three adversarial review rounds: 5 must-fixes,
  all in before merge.
- **Cutover**: `cutover-auto-rollback-v3.sh` from live `a2bac00d`, deploy tree `dbf06dc7`
  (guide bump), kill of PID 75597 at 00:46:26 CT; systemd relaunched PID 79574;
  `🔐 BOOT INTEGRITY OK — rev 0e490e448279 · expected 0e490e448279 · goldens PASS` at +7 s;
  `/api/health` revision `0e490e448279`. Rollback binary kept as `nofx-bin.old.a2bac00d`.
- **Flat gate**: open positions 0 · placed/pending arms 0 (14 superseded rows only) ·
  picture rows 0 · broker positions snapshot count=0 · no planner read in flight (last plan
  write 21:14 CT).
- **New boot lines (READ, not literal)**:
  `🔒 maintenance: hold=clear job=n/a since=n/a addon_ack=n/a` — no hold file, OFF state.
  `🖥 ui: served-by=go-static bundle=index-CqKdtrm3.js bundle-rev=0e490e44 matches the binary`.
  `📷 picture-htf: mode=on … addon=proven (build="2026-09-20-p1", need ≥ 2026-09-20-p1)`.
  `🔌 nt8 addon: build_id=2026-09-20-p1 expected=2026-09-22-m2 match=NO` — EXPECTED: the
  m2 AddOn is in the tree and NOT deployed (owner F5, gated on M2.1 merging first). Every
  behaviour floor is a separate constant; nothing lost.
- **Behaviour change live**: none. With no hold file the wire and every entry path are
  byte-identical to a2bac00d. The hold, permit, frames, gate and endpoints exist and are inert.
- **Pre-existing, seen during the boot, NOT from this change**: `🔭 desk strip: line 10
  (planner) panicked and was contained: nil pointer` every scan (logged at 00:44:20 and
  00:46:20 on the OLD binary before the kill, and after). Display-only, contained. To fix in
  its own small change.
- **Also merged behind the boot**: PR #180 (M1 feasibility docs) → dev `df51f74c`.
- **Not done tonight**: the m2 AddOn F5 (owner, after M2.1); the pre-existing trader test
  race in CI's coverage job (owner-held); the queued-AI-entry phantom-position class (owner's
  call, checklist entry in #182).
- **Deploy tree**: the owner's preserved `store/armed_orders.go` import reorder was stashed
  for the cutover (the script requires a clean tree) and restored after the marker.
