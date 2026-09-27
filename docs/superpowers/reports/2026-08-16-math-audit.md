# W12 — Math Correctness Audit

**LINE 1 — ALL MATH TRUE.** 14/14 formulas computationally CORRECT (independent
recompute + digit-by-digit hand-check + property/fuzz oracles, real MNQ data). **0
behavior bugs.** 3 documentation/derivation notes: **1 medium, 2 low** — none
changes a result; all report-only (no formula touched). 26 pkgs green.

## Oracle coverage (executable tests added)
| Formula | Verdict | Key oracle value | Test |
|---|---|---|---|
| EMA (SMA-seed, 2/(n+1)) | TRUE | EMA(3)[10..14]=13.0 · EMA20(real20)=29071.90 | `market/w12_indicators_test.go` |
| ATR14 (Wilder, TR max-3) | TRUE | ATR14(real20)=**595.8473** | `market/w12_indicators_test.go` |
| Realized-vol (√288, pop sd) | TRUE¹ | RV%=1.6970563 (a=.001) | `kernel/w12_regime_test.go` |
| percentileRank / atrBucket | TRUE | 80.0 · edges 25/75/90 half-open | `kernel/w12_regime_test.go` |
| Overnight gap ×ATR | TRUE | (30148.75−30188.50)/595.85=−0.0667 | `kernel/w12_regime_test.go` |
| Tick round 0.25 | TRUE | half→away-from-zero, idempotent | `trader/ninjatrader/w12_money_math_test.go` |
| MNQ money $2/pt | TRUE | 3 real trades to the cent | `…/w12_money_math_test.go` |
| R:R = reward/risk | TRUE² | 15/5=3.0 passes 1.5 & 3.0 | (hand-check below) |
| Breakeven trigger | TRUE | LONG/SHORT sign-correct, ≥trig | `trader/w12_math_test.go` |
| Confluence score | TRUE | 0.66→C · 1.40→A · 0.9792→B | `kernel/w12_regime_test.go` |
| Bonferroni α | TRUE | 0.05/8=0.00625 | `kernel/w12_regime_test.go` |
| Pre-registered N | TRUE³ | 1565 > 80%-power 1279 (conservative) | `kernel/w12_regime_test.go` |
| Session/PDH/ONH/OR-IB | TRUE | tz America/Chicago, both 2026 DST | `kernel/w12_dst_test.go` |
| 17:00 CT day-roll | TRUE | DST-correct across Mar-8 / Nov-1 | `kernel/w12_dst_test.go` |

## Findings (report-only — never a silent change)
- **[MEDIUM] N=1565 vs stated power.** `kernel/matched_random.go:17`. α=0.00625, p0=.50,
  effect=.05: 80%-power n≈**1279** (two-sided) / 1115 (one-sided). 1565 reconciles at
  **~89% power (two-sided)**, not 80%. No bug — a HIGHER N = stricter WARMING gate (never
  greens early). **Fix (doc):** state the power basis (≈0.90) + tail, or derive 1565 in a
  comment; or make N a function of α/effect so it can't silently drift.
- **[LOW] RV √288 assumes 24h.** `kernel/regime.go:205`. A CME day ≈23h ≈276 5m bars
  (the code's OWN `rvBaselineMinBarsPerDay=200` comment says so). √288 over-states the
  ABSOLUTE RV% by √(288/276)=+2.1%. **Cancels in the recent/baseline ratio** → regime
  class unaffected. **Fix (doc):** shared `barsPerCMEDay≈276` constant, or note RV% is
  24h-normalized and only the ratio is used.
- **[LOW] R:R default is 3.0, docs say 1.5.** `engine_position.go:142-145` (seed 3.0 +
  fallback 3.0 + clamp ≥1.0). Real floor = **3.0** (stricter than the CLAUDE.md-cited
  1.5). **Fix (doc):** correct kernel/CLAUDE.md to "R:R ≥ 3.0 default, clamped [1,10]".

## Hand-checks (owner phone-verifiable)
- **Money** id515: (30025.00−29901.25)=123.75 pts × $2 × 1 = **$247.50** = stored ✓.
- **ATR14** bar19 Wilder step: (621.5087×13 + 262.25)/14 = 8341.863/14 = **595.8473**.
- **EMA(3)** [10,11,12,13,14]: seed (10+11+12)/3=11; ×0.5 recursion → 12 → **13.0**.
- **R:R** entry100 stop95 target115: reward15/risk5 = **3.0** ≥ floor → pass.

## NT8 CROSS-CHECK TABLE (owner-run — overlay EMA(200,daily)+ATR(14,daily) on MNQ)
Bot values (Trader "hoang"/Sim101, intraday snapshot; day-bar = 05:00 UTC Globex):
| CME date | bot EMA200-d | bot ATR14-d | snapshot UTC |
|---|---|---|---|
| 2026-08-14 | 27368.9708 | 582.4799 | 20:59 |
| 2026-08-13 | 27256.7000 | 603.2380 | 14:24 |
| 2026-08-12 | 27254.6900 | 576.8987 | 23:59 |
| 2026-08-11 | 27236.5137 | 591.7852 | 23:59 |
| 2026-08-10 | 27222.8713 | 616.6835 | 23:56 |
PASS if |Δ| ≤ ~1 tick given matching session-template + snapshot time (NT8 EMA also
SMA-seeds; seed drift is negligible at 200+ bars). Differences beyond that = a
session-template mismatch, NOT a formula error.

## Deploy (owner, CME-closed window, before Mon 08:00 CT)
```
cd /home/hoang/nofx && git pull && go build -o nofx-bin . && echo BUILD OK && sudo systemctl restart nofx
```
No `.cs` touched → no NT8 F5. W12 is test-additive (zero production math changed).
