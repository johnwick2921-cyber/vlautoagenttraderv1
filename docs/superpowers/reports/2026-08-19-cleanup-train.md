# CLEANUP TRAIN — EVERYTHING REMAINING (2026-08-19)
**6 of 6 landed · DEPLOYED bb966a049d9e · boot 18:25:53 CT — `🔐 BOOT INTEGRITY OK — rev bb966a049d9e · expected bb966a049d9e · goldens PASS`**
## Item 1 — learning loop closed
- `decision_records` now gets plan_id/plan_version/overlay_version/cited_scenario on every decision write; MAE/MFE + adherence grade flow into session/daily digests and the review path. Backfill: prior rows keep ''/0/0 (attribution starts now — old rows only inferable via positions' plan links).
- Test: a closed trade produces MAE/MFE + a grade, both visible in the digest and linkable to its plan version.
## Item 2 — self-announcing errors
- Refusals are never stored as bare waits — any late action-rewrite records reason + type + cost (decision lost / trade lost / none); a test fails if a refusal is stored as a plain wait.
- One error surface: dashboard panel + alert feed shows per-session errors by type, counts, decisions lost, last occurrence.
- Daily digest line: "errors today: N (types: …), decisions lost: N".
- Sweep: the decision/order/planner paths emit structured events; **76 pre-existing bare-discard sites remain elsewhere (sized, not fixed — mostly non-critical json/http writes; listed in the sweep).**
## Item 3 — soft-alert guardrails
- Master stays OFF (owner-dated). `CheckSoft` now evaluates every configured limit and, when one WOULD trip, logs + records an error event + announces via the alert feed — never blocks anything.
## Item 4 — level_state trader scope
- `level_key` is now trader-scoped (`traderID|symbol|type|origin|bin`); additive migration backfills unscoped rows to the single day-plan trader only (ambiguous → rows stay cold, never shared). Two-trader isolation test passes.
## Item 5 — cosmetics
- `ctMinutesNow` uses `kernel.CTLocation()` (single timezone source); grid comment un-mangled. Already landed earlier: real per-scenario status (no stale "armed"), HandoverBanner deleted, PruneAckedOlderThan wired. Two uncommitted audit reports committed as-is.
## Item 6 — disease sweeps at the new head
- never-wired: all new symbols have callers. scope: 0 unscoped MakeLevelKey in production. config-shadow: guard tests green. captured-then-discarded: 76 pre-existing sites, none new from this week's fixes.
## Exit bar
go build/vet/test/-race green (kernel/trader/store/api) · tsc clean · vitest 247 pass, 1 pre-existing jsdom-canvas failure (untouched) · goldens 0 diffs.
## Not done / sized
- The 76 legacy bare-discard sites (mechanical sweep, ~2–3h, no new behavior).
- Decision-record backfill for pre-today rows (would need bar history the DB doesn't store — unattributable, left as-is).
