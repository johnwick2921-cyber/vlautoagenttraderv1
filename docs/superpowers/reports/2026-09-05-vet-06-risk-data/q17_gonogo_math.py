#!/usr/bin/env python3
"""q17 — the go/no-go arithmetic. Inputs are the measured sample (q04/q05/q06): n=65, mean -8.676, sd 101.162;
days D=12, day mean -46.99, day sd 256.60. Wilson z=1.96, power z=0.8416."""
import math
za, zb = 1.959963985, 0.8416212336
n, mu, sd = 65, -8.676, 101.162
D, dmu, dsd = 12, -46.99, 256.60
print("== Stage-A lower-CI clause: the per-trade MEAN it demands at sd=%.1f (lower 95%% CI > 0  <=>  mean > 1.96*sd/sqrt(n)) ==" % sd)
for k in (30, 50, 65, 100, 200, 400, 800):
    need = za*sd/math.sqrt(k)
    print(f"  n={k:4d}: mean must exceed ${need:6.2f}/trade  (= {need/2:5.1f} MNQ pts/contract)  -- at n=65 the measured mean is {mu:+.2f}")
print("\n== n required (alpha .05 two-sided, power .80) at hypothetical TRUE edges, sd=%.1f ==" % sd)
for edge in (5, 10, 15, 20, 30, 36.2):
    nreq = ((za+zb)*sd/edge)**2
    print(f"  true mean +${edge:5.1f}/trade: n_req = {nreq:7.0f} trades  -> at 5.4/day (pre-strict) {nreq/5.4:6.0f} session-days; at 1.5/day (arm-only) {nreq/1.5:6.0f}; at 0.5/day (09-03/09-04 observed) {nreq/0.5:6.0f}")
print("\n== day-level: n_days required at the measured day effect ==")
nd = ((za+zb)*dsd/abs(dmu))**2
print(f"  day mean {dmu:+.2f}, day sd {dsd:.2f}: n_days_req = {nd:.0f} session-days (~{nd/5:.0f} weeks) to resolve the CURRENT effect; 95% t-CI at D=12 is [-210.03, +116.04]")
for dedge in (25, 50, 100):
    print(f"  at a true +${dedge}/day edge (sd {dsd:.0f}): n_days_req = {((za+zb)*dsd/dedge)**2:.0f}")
print("\n== time to n>=30 post-0B (n=3 today, strict since 2026-09-03 20:35 CT) ==")
for rate in (0.5, 1.0, 1.5, 2.0):
    print(f"  at {rate}/session-day: {27/rate:.0f} more session-days (~{27/rate/5:.1f} weeks)")
print("\n== dollar risk per trade as it ran (armed_orders stop_pts, $2/pt) ==")
stops=[17.25,18.0,19.0,20.5,20.55,20.99,21.0,21.5,21.96,22.25,23.21,24.0,24.05,24.12,24.7,25.79,26.5,27.0,27.0,30.0,32.7,35.06,35.06,40.0,40.0,40.0,40.0,40.0,44.25,45.0,45.0,47.26,48.75,49.75,50.0,50.0,50.33,54.23,54.56,54.56,54.6,63.12,66.63,67.88,150.0,150.0]  # de-duplicated 54.23 churn run collapsed to one (q07 lists 20 copies of one re-placed arm)
stops_sorted=sorted(stops)
def q(a,p):
    i=(len(a)-1)*p; lo=math.floor(i); hi=math.ceil(i); return a[lo]+(a[hi]-a[lo])*(i-lo)
print(f"  distinct-arm stop pts: n={len(stops)} p25={q(stops_sorted,.25):.1f} p50={q(stops_sorted,.5):.1f} p75={q(stops_sorted,.75):.1f} p90={q(stops_sorted,.9):.1f} max={max(stops):.1f}  -> $ at 1 lot: p50=${2*q(stops_sorted,.5):.0f} p90=${2*q(stops_sorted,.9):.0f} max=${2*max(stops):.0f}")
print(f"  post-0B filled arm (id 35): 66.63 pts = $133; worst realised trade in sample: id 589 -$155 (77.5 pts)")
print("\n== daily limit vs per-trade risk ==")
print(f"  $450 limit / $100 median stop = {450/100:.1f} median stops; / $133 (post-0B fill) = {450/133:.1f}; / $155 worst = {450/155:.1f}")
print(f"  worst-case realised day under a $450 REALIZED-ONLY trip with one-open-position: trip at <= -450 then the open trade runs to its stop -> floor ~ -450 - one stop; at p90 stop ($127) ~ -$577; at the 150-pt max ($300) ~ -$750")
