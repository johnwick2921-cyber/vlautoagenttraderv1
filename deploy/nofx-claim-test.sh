#!/usr/bin/env bash
# Pins for nofx-claim. The load-bearing one is REAL-2: the actual malformed
# claim from 2026-09-04 must FAIL. A checker that passes the message that
# caused the incident is decoration.
set -uo pipefail
CLAIM="$(dirname "$0")/nofx-claim.sh"
RE="$(grep -oP "^CLAIM_RE='\K.*(?='$)" "$CLAIM")"
pass=0; fail=0
t() { # t <name> <expect:ok|bad> <message>
  local name="$1" expect="$2" msg="$3" got
  if printf '%s' "$msg" | grep -qE "$RE"; then got=ok; else got=bad; fi
  if [ "$got" = "$expect" ]; then pass=$((pass+1)); echo "  PASS $name"
  else fail=$((fail+1)); echo "  FAIL $name — expected $expect, got $got"; echo "       msg: $msg"; fi
}

echo "REAL messages from 2026-09-04:"
# The message that caused the incident. MUST be rejected.
t REAL-1-reaper-no-session bad \
  "claim: reaper reads the snapshot, not order_update silence (PART 3 step 0)"
# nofx-47's claim. Well-formed under the 2026-09-04 rule, NOT under the
# 2026-09-07 routing amendment: "nofx-47" names a lane nobody can address.
t LEGACY-2-nofx47-bare-session bad \
  "claim: reaper reads the broker snapshot, not order_update silence — nofx-47, 2026-09-04T09:44:00-05:00"
# Same: the bare form is now rejected, which is the amendment working.
t LEGACY-3-bare-session bad \
  "claim: claim-identity enforcement + worktree prune — nofx-b3, 2026-09-04T10:10:00-05:00"

echo "routing pins (owner ruling 2026-09-07 — a claim must be ROUTABLE):"
# The amendment's own claim, and this wave's.
t ROUTE-1-composite-ok ok \
  "claim: PART 3 step 0: the claim carries BOTH identifiers — claimid-ee7f9468/nofx-db[ca9c60], 2026-09-07T10:12:25-05:00"
t ROUTE-2-takeover-ok ok \
  "claim: TAKEOVER of session-calendar — claimid-ee7f9468/nofx-db[ca9c60], 2026-09-07T10:20:00-05:00"
# A lane that cannot read its own ref says so, and is ACCEPTED.
t ROUTE-3-unlisted-ok ok \
  "claim: some wave — somewave-554049f5/nofx-2c[unlisted], 2026-09-07T09:47:55-05:00"
# The collision that produced the rule: a real claim naming a lane nobody could
# address. It must now be refused at the source.
t ROUTE-4-the-incident bad \
  "claim: a session calendar as DATA — session-calendar-554049f5, 2026-09-07T09:47:55-05:00"
# Half-composite forms are not enough.
t ROUTE-5-uuid-no-ref     bad "claim: w — wave-ee7f9468/nofx-db, 2026-09-07T10:00:00-05:00"
t ROUTE-6-ref-no-uuid     bad "claim: w — nofx-db[ca9c60], 2026-09-07T10:00:00-05:00"
t ROUTE-7-empty-ref       bad "claim: w — wave-ee7f9468/nofx-db[], 2026-09-07T10:00:00-05:00"

echo "shape pins:"
t no-session          bad "claim: some wave, 2026-09-04T09:44:00-05:00"
t no-timestamp        bad "claim: some wave — nofx-x"
t date-not-iso        bad "claim: some wave — nofx-x, Sep 4 2026"
t iso-utc-z           ok  "claim: some wave — wave-ee7f9468/nofx-x[ca9c60], 2026-09-04T09:44:00Z"
t iso-no-colon-offset ok  "claim: some wave — wave-ee7f9468/nofx-x[ca9c60], 2026-09-04T09:44:00-0500"
t not-a-claim         bad "fix(thing): unrelated commit"
t empty-session       bad "claim: some wave — , 2026-09-04T09:44:00-05:00"

echo "---- $pass passed, $fail failed"
[ "$fail" -eq 0 ]
