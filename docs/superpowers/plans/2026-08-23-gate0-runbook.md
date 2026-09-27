# GATE 0 Runbook — Sunday 2026-08-23

Owner-facing checklist. Nothing here deploys code; every step is flat-window or
ruling-gated. The live binary stays on **rev 108dbaf0** until the Wave-1 deploy.

## 0. Pre-gate owner to-dos

- [ ] **Top up AI credits** — exhausted since 08-21 19:55 (402, alert 224).
      Without it the soak gets ZERO AI activity and the E2 replay stalls.
- [ ] Confirm NT8 is up (Tradovate feed + TCP bridge) — it is data AND execution.

## 1. Soak verdict (timer fires 16:55 CT)

- [ ] `systemctl --user list-timers` shows `e4-soak.timer` (armed, Sun 16:55 CT).
- [ ] Soak script: `~/soak-g7/e4-soak.sh`; results land in `~/soak-g7/`.
- [ ] Verdict rubric: zero unexpected AI 402s · zero NT8 disconnects · zero
      ATR-variant anomalies · cycle cadence within the flat expectations.
- [ ] Record the verdict in the soak log + the calibration ledger.

## 2. 08-22 E2 replay (3-variant G6)

Pre-computed deltas for the loss-streak gate: **A (shipped 4/60) = 0.00 ·
B (research 3/30) = 0.00 · C (≤0-reset) = −88.50**.

- [ ] Run the replay against the G6 variant table.
- [ ] Confirm the computed deltas match the pre-computed values (they are a
      pin, not a prediction — a mismatch is a replay-harness bug, stop and fix).
      **NOTE (08-23): the G6 loss-streak gate was REMOVED by owner order — the
      loss-streak variant of this replay is moot; skip it.**

## 3. Calibration pass — 7 owner decisions

| # | Decision | Shipped | Research | Note |
|---|---|---|---|---|
| 1 | swing window k | 2 | 10–20 | do NOT change before soak+replay |
| 2 | MSS FVG | on | keep/drop | ruling |
| 3 | HTF veto TF | 1h | 15m | ruling |
| 4 | min confidence | 60 | 65 | ruling |
| 5 | trail multiplier | 2.0 | 1.5 | gates wave-3 item 3.1 |
| 6 | loss streak | ~~4/60~~ **REMOVED by owner 08-23** | ~~3/30 (v5 C.6)~~ | moot — G6 deleted |
| 7 | target R:R | ≥3 (FULL-SPEC) | 2.5 (v5) | research-internal conflict |

- [ ] Log every ruling with the changed value + ledger line.
- [ ] Rulings 5 and 6 unlock wave-3 items **3.1** (trail lock) and **3.5**
      (apply calibration) respectively.

## 4. PR merges — STRICT stack order

```
#65 (ATR conformance) → #64 (regime wave) → #66 (audit V2) → #67 (DeepSeek
thinking) → #68 (read-through/plan) → #69 (wave 2) → #70 (wave 3) → #71 (4.5)
→ #72 (4.4) → #73 (4.2) → #74 (4.3)
```

- [ ] Merge only in this order (later PRs are stacked on earlier branches).
- [ ] #62/#63 (docs) may merge anytime, no order constraints.

## 5. Wave 1 deploy (Sunday night flat window)

- [ ] Flat window: 0 OPEN positions, no other agent running.
- [ ] Merge #67 → build → `git rev-parse HEAD > deploy/RELEASE` → `kill -9`
      (SIGTERM exits 0 and does NOT relaunch) → boot-line quote:
      `🔐 BOOT INTEGRITY OK — rev <rev> · goldens PASS`.
- [ ] Verify one PID + regime ledger lines.

## 6. 4.3 owner gate (NOT part of Sunday — whenever NT8 work is scheduled)

- [ ] Copy `ninjascript/VLTraderTCPClient.cs` to
      `C:\Users\hoang\Documents\NinjaTrader 8\bin\Custom\AddOns\` (WSL:
      `/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom/AddOns/`).
- [ ] F5 compile inside NT8 + **full NT8 restart** (AddOns do NOT hot-reload).
- [ ] Only then may `EOD_FLAT_LIMIT_TICKS` / `EOD_FLAT_MARKET_AFTER_SEC` be set.
      Until the AddOn is rebuilt, the knobs MUST stay 0 (old AddOn ignores the
      field, but the limit frame would never fill and the fallback still lands
      — no damage, but no benefit either).

## 7. Partner repo — already done 2026-08-22

- [ ] `vlautoagenttraderv1` main = `f9a6f001` (DeepSeek defaults + wave-4
      mirror: thinking knobs, attribution hardening, exit fills) — pushed ✓.

## End state

All 5 waves shipped or ruled. The next MASTER AUDIT triggers on first incident /
metric anomaly / gate fire after the soak (PR #66 verdict).
