# SUBSYSTEM B — VALIDATOR BELIEFS (census B1–B10) + D2 deliverable

Read-only lane, worktree `/home/hoang/nofx-conform` (HEAD `fb50903f`, base `492d2067`).
Running binary: rev `70af663d`, PID 878451, boot 2026-09-04 08:30:11 CT.
Resolved values quoted from the process's own boot block in
`/home/hoang/nofx/data/nofx_2026-09-04.log` (08:30:11), never from a file default.
`/api/config/resolved` and `/api/risk/gate-blocks` both return
`{"error":"Missing Authorization header"}` from this session — not used.

## Boot-8 lines this lane relies on (READ from the log, 08:30:11 CT) [A]

```
main.go:335  🛑 exits: stop=max(anchor+clr, 1.5×ATR5m) · anchor_max=3.0×ATR5m · BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)
main.go:338  📜 prompt feeds forward: void-levels=n/a (computed per read) · stop-floor=1.5×ATR5m (n/a — no ATR yet) · waterfall-displacement-floor=1.0×ATR5m (n/a — no ATR yet) (stated per level) · reject-block=top+tail (class 45)
main.go:430  🎯 arms: bias-coherent=warn · stop-entry=on(reclaim) · far-arm counter=on(3.0×ATR5m) · ledger append-only=on
main.go:431  🎛 entry law: bd_min_closes=1 bd_min_disp_atr=1.00 mss_min_disp_atr=0.50 accept_hold_min=10 stop_entry_offset_ticks=2 retest_wait_bars=6 stop_entry_seam=ON
auto_trader.go:43 🛑 min-sl guard: atr_mult=1.5 level_clearance=2tick(s)
auto_trader.go:43 ⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off arm_rr=2.0 (gate-at-arm only; market-entry floor 2.0 unchanged)
levels_volume_boot.go:17 📐 fvg_entry: on min_disp=1.5×ATR ce_width=20pt lookback=40 bars — advisory, zero gates
levels_volume_boot.go:19 🔧 S-wave: stale_confirm=2.0×ATR5m · eod_flat=session-end (NY 14:45 CT, R-A15)
levels_volume_boot.go:30 🔬 conditions: live [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim] · shadow [breakout_retest, fvg_entry]
levels_volume_boot.go:42 📜 prompt/validator contract: 19 restrictions, all stated in prompt (class 38 guard)
main.go:343  🛡 arm gate: invalidation-wired=on · armed-under surfaces=on
main.go:408  📜 void scope: session-day window · 1m×2000 · one resolver for prompt AND validator (parity)
```

Two lines the dispatch's boot block omitted are recovered above (`🎯 arms`, `🎛 entry law`);
they are the resolved source for BD_MIN_CLOSES=1 and the 2026-09-04 arm wave.

`.env` (24 keys) sets NONE of MIN_SL_ATR_MULT / STALE_CONFIRM_ATR / BD_* / FVG_* /
ARM_MIN_RR / ARM_FAR_ATR_MULT / ARM_STOP_ANCHOR_MAX_ATR / SHADOW_AB_ENABLED, so the
code defaults ARE the resolved values — corroborated by the boot lines above. [A]

## CSVs in this directory (this lane)

- `subsystemB_rules.csv` — one row per B rule + census D2
- `subsystemB_D2_invented_rejects.csv` — the D2 deliverable
- `subsystemB_planner_rejects.csv` — all 55 planner_rejected_prompts rows, CT-stamped
- `subsystemB_reject_class_tally.csv` — reject class × n × window
