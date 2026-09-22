# W-PICTURE-HTF activation — 2026-09-22 (owner-attended, SIM)

- **Cutover**: 00:48:10 CT — binary swap + SIGKILL; systemd relaunched PID
  75597. Boot: `🔐 BOOT INTEGRITY OK — rev a2bac00dc2ab · expected a2bac00dc2ab
  · goldens PASS`.
- **AddOn receipt**: `nt8 addon: build_id=2026-09-20-p1 expected=2026-09-20-p1
  match=yes` (received on the wire; folder md5s 66e8265c…/58feee90…; backup
  ~/nofx-backups/addon/20260922-002056).
- **Flat gate before the kill**: open positions 0 · non-terminal arms 0 ·
  broker book 0 orders · fresh snapshot (build 2026-09-20-p1). Arm 173 (S1)
  filled earlier and its position was manually closed by the owner (615,
  CLOSED exit 30811.00 reason manual); its protective stop was cancelled —
  book verified empty, no naked-short risk. Arm 174 (S2) settled cancelled by
  the bot (book-confirmed).
- **Native 4H readiness**: evaluator cache (BarCache, live+seed, ring 2500)
  holds 428 native 4h bars for the resolved contract (`MNQ 4h asked=500
  served=428 span=2441h gaps=0`, oldest ~101 days) — ≥ the 120 pivot-window
  requirement. Completion/provenance evidence = final+emitted_at markers,
  guaranteed by the proven AddOn capability. The earlier `4h=95` was a LEVEL
  count from the dayplan engine, not the evaluator's bar count.
- **Enable**: 08:12:15 CT — PUT /api/strategies/a5b7662e-… (only
  `day_plan.picture_htf.enabled` changed; min_rr=2.0 and plan_mode=strict
  preserved). Runtime: `📷 picture-htf: mode=on rule=v1 SIM-only data=native
  NT8 bars (final+emitted_at) addon=proven (build="2026-09-20-p1", need ≥
  2026-09-20-p1)`.
- **Entry/fill/protection proof**: PENDING — no natural qualifying setup has
  occurred; reported when received. Historical backfill never triggers the
  mode (live-only sink by design; no import performed).
