# PRE-OPEN FIXES — the 3 conditions gating Monday's CONDITIONAL-GO

**LINE 1 — MONDAY CONDITIONS CLEARED.** P1 boot integrity ✓ · P2 dark-regime alert +
DEGRADED flag ✓ (livelock root-caused, fix queued for an NT8 window) · P3 session-end
drift ✓ · P4 calendar verified ✓. Exit bar green: 26 Go pkgs + `-race`, tsc + build +
51 FE tests, **zero golden changes**. Commits `a8af893b` `66ccb500` `542dfdc8`.

## STEP 0 — one miss, reported and cleared
Tree clean ✓ · HEAD on the `c3f4092a` lineage ✓ · bot PID 101785 running
`vcs.revision=c3f4092a` ✓ · live 8080/3000 + sandbox 3001 untouched ✓ ·
**ONE-SESSION: FAILED first pass** — a second Claude session (PID 100656, own cwd +
4 live MCP children incl. its own Playwright) was running. I **stopped and reported
rather than build**; the owner cleared it, and I proceeded with append-only
precautions (explicit `git add <paths>`, tree re-check before every commit, no
amend/rebase).

## Skills used — honest accounting
The named gstack skills (`/investigate`, `/plan-eng-review`, `/careful`,
`/context-save`) are **not installed** in this environment. I used the installed
equivalents and say so plainly: **superpowers:systematic-debugging** (P2 livelock,
P4 gap — evidence before conclusions), **verification-before-completion** (every
claim below has a receipt), and a self-run **adversarial review** in place of
commit-review. No subagents: each part was small and sequential; a fan-out would
have cost more context than it saved.

## P1 — BOOT INTEGRITY (the Knight-Capital control)
Startup now asserts the binary against the intended release **and** re-renders the
three prompt goldens **embedded in the same binary** (`go:embed`) — a right-revision
binary can still be wrong if a prompt builder drifted. Intended release =
`NOFX_EXPECTED_REVISION` or `deploy/RELEASE` (prefix match, so a short sha works).
Failure ⇒ **entries refused for every trader**, loud P0, everything else read-only.
Closes/holds are never gated (`isBootIntegrityGatedAction`, asserted by test) so a
refused process can still bring a position flat. **Declaring nothing is not a
failure** — it logs and proceeds, so this is inert until you fill `deploy/RELEASE`.
Proven on a **real binary**, both paths:
```
🔐 BOOT INTEGRITY REFUSED — rev 3155b766a7d8 · built 2026-08-16T12:38:13Z · expected deadbeefdead · goldens PASS
🔐 TRADING REFUSED — binary is revision "3155b766a7d8" but the intended release is "deadbeefdead" — a stale binary is running
🔐 BOOT INTEGRITY OK — rev 3155b766a7d8 · built 2026-08-16T12:38:13Z · expected 3155b76 · goldens PASS
```
The self-check earned its keep immediately: it caught **my own** mis-transcribed
KEY LEVELS fixture before it shipped.

## P2 — DARK REGIME (detection shipped; livelock fix queued)
Every read now names the unavailable regime fields, alerts (P1, or **P0 once
degraded**, deduped per session-day), and **stamps `dark_regime_count` + `degraded`
on the plan row**. >3 dark ⇒ the plan is written but flagged, and the card says
**⚠ DEGRADED 4/7**.
**Livelock root cause [A]:** `VLBarsSubscriptionManager`'s fast guard is **global** —
`anyDead` goes true if *any one* subscription was seeded without a live `.Update`
within 20s, and it then recreates **all 14** BarsRequests, resetting `LiveUpdateSeen`
on healthy ones too. The attempt counter re-arms only when *nothing* is dead, so a
timeframe that legitimately never ticks in 20s (3d, 1w, or anything during a quiet
window) keeps `anyDead` permanently true → healthy series get churned, the 3-attempt
budget burns out, and truly dead series wait for the 75-minute backstop. Starving
1h+1d is exactly the reported **4/7 dark**.
**Recommendation:** make the guard per-subscription (recreate only dead entries,
re-arm per entry), or scope it to the timeframes the kernel consumes (1m/5m/15m/1h/1d).
**Not fixed now** — it is a C# change needing copy → F5 → NT8 restart, which is not
safe hours before an open.

## P3 — SESSION-END DRIFT
The spec wrote NY as "08:30–15:45", **mixing CT with ET**; the registry then ended the
window at 15:00 CT while flatting at 14:45 CT — a **15-minute band where the gate
called NY open even though EOD-flat had already brought the book flat**. Contract
applied: NY ends **14:45 CT = 15:45 ET**, and the flat is that same instant.
**Single source of truth = `kernel.DefaultSessionRegistry()`.** A test now fails if
any mirror drifts: the registry values · the invariant `WindowEndCT == FlatCT` for
*every* session · the gate (14:44 in-window, 14:45 out) · the **TypeScript** table is
parsed and compared value-by-value · the spec no longer carries the mixed-unit phrase.
Net effect is more conservative: no entry can open after the flat.

## P4 — CALENDAR ROLL (verified, read-only)
Feed **rolled** (`source=forexfactory`): 08-18 (2 events) · 08-19 (3) · 08-20 (2) ·
08-21 (7). T1s in the week: GBP Claimant Count (08-18), GBP CPI (08-19 01:00 CT),
**USD FOMC Meeting Minutes 08-19 13:00 CT**. The NY currency filter was **proven on
the FOMC day** — GBP/EUR events are dropped for NY, leaving exactly the expected
**single T1 = FOMC Wed 13:00 CT**.
⚠ **Monday 2026-08-17 has no slice row** (the feed returned no events for it). Because
no row exists, the producer will keep retrying Monday morning (≤1/hr) and pick up
anything published late — which is why I did **not** write empty rows: an empty row
would make it "skip-fresh" and never refetch. Watch item, not a defect.

## Adversarial findings (self-review) — all resolved
**F1** boot refusal must not trap a position → closes are absent from the gated
switch (asserted by test). **F2** a per-cycle P0 storm would be worse than silence →
deduped per CME session-day. **F3** same for the dark-regime alert → deduped per
(trade_date, session). **F4** old plan rows must not read as degraded → both columns
default 0/false. **F5** any other session with end≠flat → none (invariant test).

## Screenshot index (`.playwright-mcp/audit/`, headless chromium)
| File | Verifies |
|---|---|
| `p2-degraded-badge.png` | P2 — "⚠ DEGRADED 4/7" beside ACTIVE; control card has none |
| `p3-session-end.png` | P3 — "NY window 08:30–14:45 CT · flat 14:45 CT" |
**Honest limitation:** these render the REAL components in headless chromium via a
temporary harness (deleted after), not the full logged-in app — I don't hold your
credentials and won't forge a token. DOM assertions accompanied both screenshots.

## DEPLOY HANDOFF
```bash
cd /home/hoang/nofx
git pull
go build -o nofx-bin . && echo BUILD OK
sudo systemctl restart nofx
go version -m ./nofx-bin | grep vcs.revision     # must match the sha you just pulled
```
**In the boot log you should now see exactly one new line:**
```
🔐 BOOT INTEGRITY OK — rev <sha> · built <time> · expected <unset> · goldens PASS
```
`expected <unset>` is correct today — the assertion is inert until you opt in. **To
arm it**, put the sha in `deploy/RELEASE` (or export `NOFX_EXPECTED_REVISION`) as part
of the deploy; a mismatch then prints `REFUSED` and blocks entries instead of trading
on a stale binary. Note `vcs.modified=true` is expected here (untracked scratch files)
and is not a failure.
