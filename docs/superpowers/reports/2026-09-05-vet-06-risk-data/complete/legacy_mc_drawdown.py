#!/usr/bin/env python3
"""1E — Monte Carlo drawdown rig. Read-only, seeded, re-runnable.

Re-run:  cd ~/nofx-analysis/mc-drawdown && python3 mc_drawdown.py
Inputs:  trade_sample.csv (built read-only from data/data.db, mode=ro)
Outputs: drawdown_paths.csv, day_sim.csv, and the tables printed below.

Pre-registered 2026-09-03 (commit 2eb846e2) BEFORE this ran.
"""
import csv, math, statistics as st
from collections import defaultdict
import numpy as np

SEED, B = 20260903, 10_000
rng = np.random.default_rng(SEED)

rows = list(csv.DictReader(open('trade_sample.csv')))
pnl = np.array([float(r['pnl_corrected']) for r in rows])
ids = [r['id'] for r in rows]
days = defaultdict(list)
for r in rows:
    days[r['session_day_ct']].append(float(r['pnl_corrected']))
day_keys = sorted(days)
n = len(pnl)

def maxdd(path):
    """Max peak-to-trough drawdown of a cumulative equity path (positive $)."""
    eq = np.cumsum(path)
    peak = np.maximum.accumulate(np.concatenate(([0.0], eq)))[1:]
    return float(np.max(peak - eq))

def iid_paths(h, b=B):
    return rng.choice(pnl, size=(b, h), replace=True)

def block_paths(h, b=B, mean_block=5):
    """Stationary bootstrap (Politis-Romano): geometric block lengths, wrapped."""
    out = np.empty((b, h))
    p = 1.0 / mean_block
    for i in range(b):
        seq, idx = [], rng.integers(n)
        while len(seq) < h:
            seq.append(pnl[idx % n])
            idx = rng.integers(n) if rng.random() < p else idx + 1
        out[i] = seq[:h]
    return out

def q(a, p):
    return float(np.percentile(a, p))

print("=" * 78)
print(f"SAMPLE  n={n}  ids {ids[0]}..{ids[-1]}  sum={pnl.sum():+.2f}  mean={pnl.mean():+.3f}  sd={pnl.std(ddof=1):.3f}")
wins = int((pnl > 0).sum()); losses = int((pnl < 0).sum()); flats = int((pnl == 0).sum())
pw = wins / (wins + losses)
print(f"        wins={wins} losses={losses} flat={flats}  p(win)={pw:.4f} (flats excluded from p(win))")

# ── Q1 ───────────────────────────────────────────────────────────────────────
print("\nQ1 — MAX DRAWDOWN distribution ($, 1 contract, $2/pt)")
print(f"{'horizon':>8} {'method':>6} {'p50':>9} {'p90':>9} {'p95':>9} {'p99':>9} {'worst':>10}")
dd_rows = []
for h in (20, 50, 100):
    for label, gen in (('iid', iid_paths), ('block', block_paths)):
        dd = np.array([maxdd(p) for p in gen(h)])
        print(f"{h:>8} {label:>6} {q(dd,50):>9.0f} {q(dd,90):>9.0f} {q(dd,95):>9.0f} {q(dd,99):>9.0f} {dd.max():>10.0f}")
        dd_rows.append(dict(horizon=h, method=label, p50=q(dd,50), p90=q(dd,90),
                            p95=q(dd,95), p99=q(dd,99), worst=float(dd.max()), B=B))
with open('drawdown_paths.csv','w',newline='') as f:
    w = csv.DictWriter(f, fieldnames=list(dd_rows[0])); w.writeheader(); w.writerows(dd_rows)

# ── Q2 ───────────────────────────────────────────────────────────────────────
print("\nQ2 — P(losing streak >= k within 50 trades)")
print(f"{'k':>3} {'exact':>10} {'bootstrap':>11}")
def max_streak(path):
    best = cur = 0
    for x in path:
        cur = cur + 1 if x < 0 else 0
        best = max(best, cur)
    return best
paths50 = iid_paths(50)
streaks = np.array([max_streak(p) for p in paths50])
pl = 1 - pw
for k in (4, 6, 8, 10):
    # exact-ish: P(no run of k in N Bernoulli trials) via the standard recursion
    N = 50
    dp = [0.0]*(N+1); dp[0] = 1.0
    # f[i] = P(no run of k losses in first i trades)
    f = [1.0]*(k)
    f = [0.0]*(N+1); f[0] = 1.0
    for i in range(1, N+1):
        s = 0.0
        for j in range(1, min(k, i)+1):
            s += (pl**(j-1)) * pw * (f[i-j] if i-j >= 0 else 0.0)
        if i >= k:
            f[i] = s
        else:
            f[i] = s + pl**i
    exact = 1 - f[N]
    emp = float((streaks >= k).mean())
    print(f"{k:>3} {exact:>10.4f} {emp:>11.4f}")

# ── Q3 ───────────────────────────────────────────────────────────────────────
print("\nQ3 — guardrail counterfactual (day resample, B=%d days-sequences)" % B)
day_lists = [days[d] for d in day_keys]
print(f"        realized session-days: {len(day_lists)}  trades/day: "
      f"min={min(len(d) for d in day_lists)} median={st.median([len(d) for d in day_lists])} max={max(len(d) for d in day_lists)}")
sim_rows = []
for name, limit_usd, max_trades in (('daily_loss_450', -450.0, None), ('max_daily_trades_3', None, 3), ('both', -450.0, 3)):
    trips = forfeited = kept = 0
    tot_days = 0
    for _ in range(B):
        d = day_lists[rng.integers(len(day_lists))]
        tot_days += 1
        run = 0.0; tripped = False
        for i, x in enumerate(d):
            if tripped:
                forfeited += x
                continue
            if max_trades is not None and i >= max_trades:
                tripped = True; trips += 1; forfeited += x; continue
            run += x; kept += x
            if limit_usd is not None and run <= limit_usd:
                tripped = True; trips += 1
        if not tripped:
            pass
    print(f"  {name:>18}: trips on {100*trips/tot_days:5.1f}% of days · "
          f"P&L kept {kept/tot_days:+8.2f}/day · forfeited {forfeited/tot_days:+8.2f}/day "
          f"· net effect {(-forfeited)/tot_days:+8.2f}/day")
    if forfeited == 0 and trips > 0:
        print(f"  {'':>18}  (forfeited 0: every trip landed on the day's LAST trade — nothing followed it)")
    sim_rows.append(dict(rule=name, trip_pct=100*trips/tot_days, kept_per_day=kept/tot_days,
                         forfeited_per_day=forfeited/tot_days, net_effect_per_day=(-forfeited)/tot_days, B=B))
with open('day_sim.csv','w',newline='') as f:
    w = csv.DictWriter(f, fieldnames=list(sim_rows[0])); w.writeheader(); w.writerows(sim_rows)

# ── Q4 ───────────────────────────────────────────────────────────────────────
print("\nQ4 — expectancy per trade")
mu, sd = float(pnl.mean()), float(pnl.std(ddof=1))
se = sd / math.sqrt(n)
lo, hi = mu - 1.96*se, mu + 1.96*se
t = mu / se
print(f"  mean={mu:+.3f}  sd={sd:.3f}  se={se:.3f}  95% CI [{lo:+.3f}, {hi:+.3f}]  t={t:+.3f}")
za, zb = 1.959963985, 0.8416212336
if abs(mu) < 1e-9:
    print("  n_required -> infinity at the current expectancy (mu == 0)")
else:
    nreq = ((za + zb) * sd / abs(mu))**2
    print(f"  n_required = ((z_a+z_b)*sd/mu)^2 = (({za:.3f}+{zb:.3f})*{sd:.2f}/{abs(mu):.3f})^2 = {nreq:,.0f} trades")

# ── Q5 ───────────────────────────────────────────────────────────────────────
print("\nQ5 — era split (0B cut over 2026-09-02 07:49 CT)")
CUT_MS = 1788353340000  # 2026-09-02 07:49 CT — the 0B cutover, by TIMESTAMP.
# (Session-day would be wrong here: the CME day rolls at 17:00 CT, so 0B's
# 07:49 cutover sits INSIDE session-day 2026-09-01. Cutting on session_day put
# every row pre-0B and reported n=0 post — corrected before publishing.)
pre  = np.array([float(r['pnl_corrected']) for r in rows if int(r['created_ms']) <  CUT_MS])
post = np.array([float(r['pnl_corrected']) for r in rows if int(r['created_ms']) >= CUT_MS])
post_ids = [r['id'] for r in rows if int(r['created_ms']) >= CUT_MS]
for label, a in (('pre-0B', pre), ('post-0B', post)):
    if len(a) == 0:
        print(f"  {label:>8}: n=0"); continue
    if len(a) < 10:
        print(f"  {label:>8}: n={len(a)} (ids {post_ids}) values={[f'{x:+.0f}' for x in a]} — FAR TOO FEW for any distribution claim; descriptive only")
        continue
    d20 = np.array([maxdd(p) for p in rng.choice(a, size=(B, 20), replace=True)])
    print(f"  {label:>8}: n={len(a):>3}  mean={a.mean():+8.3f}  sum={a.sum():+9.2f}  "
          f"maxDD@20 p50={q(d20,50):>6.0f} p95={q(d20,95):>6.0f}")

# ── M6 ───────────────────────────────────────────────────────────────────────
print("\nM6 — sensitivity: drop the single largest win AND the single largest loss")
trimmed = np.delete(pnl, [int(pnl.argmax()), int(pnl.argmin())])
print(f"  full   : n={n}  mean={pnl.mean():+.3f}  sum={pnl.sum():+.2f}")
print(f"  trimmed: n={len(trimmed)}  mean={trimmed.mean():+.3f}  sum={trimmed.sum():+.2f}  "
      f"(dropped id {ids[int(pnl.argmax())]} {pnl.max():+.0f} and id {ids[int(pnl.argmin())]} {pnl.min():+.0f})")
for h in (20, 50):
    a = np.array([maxdd(p) for p in rng.choice(pnl, size=(B, h), replace=True)])
    b = np.array([maxdd(p) for p in rng.choice(trimmed, size=(B, h), replace=True)])
    print(f"  maxDD@{h:<3} full p50={q(a,50):>6.0f} p95={q(a,95):>6.0f}  |  trimmed p50={q(b,50):>6.0f} p95={q(b,95):>6.0f}")
print("=" * 78)
