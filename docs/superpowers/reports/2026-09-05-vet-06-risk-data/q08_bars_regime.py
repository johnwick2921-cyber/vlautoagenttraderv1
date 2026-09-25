#!/usr/bin/env python3
"""q08 — bars coverage + session-day range / ATR history (regime context for Q5). mode=ro."""
import sqlite3, datetime, zoneinfo, statistics as st, math
ct = zoneinfo.ZoneInfo("America/Chicago")
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
print("--- bars coverage by symbol/tf ---")
for r in con.execute("SELECT symbol, tf, COUNT(*), datetime(MIN(open_time_ms)/1000,'unixepoch','-5 hours'), datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours') FROM bars GROUP BY 1,2 ORDER BY 1,2"):
    print(r)
# session-day (17:00 CT open) OHLC from 5m bars for MNQ
rows = con.execute("SELECT open_time_ms, o, h, l, c FROM bars WHERE symbol='MNQ' AND tf='5m' ORDER BY open_time_ms").fetchall()
print("5m MNQ bars:", len(rows))
days = {}
for ms, o, h, l, c in rows:
    t = datetime.datetime.fromtimestamp(ms/1000, ct)
    sd = (t - datetime.timedelta(hours=17)).date().isoformat()
    d = days.setdefault(sd, dict(o=o, h=h, l=l, c=c, n=0))
    d["h"] = max(d["h"], h); d["l"] = min(d["l"], l); d["c"] = c; d["n"] += 1
keys = sorted(days)
print(f"session-days with 5m bars: {len(keys)}  {keys[0]}..{keys[-1]}")
# daily range, true range, ATR(14) on session-days with >= 200 bars (full-ish day: a full CME day is 276 5m bars)
full = [k for k in keys if days[k]["n"] >= 200]
print(f"full-ish days (>=200 of 276 5m bars): {len(full)}")
prev_c = None; tr = {}
for k in full:
    d = days[k]; rng = d["h"] - d["l"]
    t = rng if prev_c is None else max(rng, abs(d["h"]-prev_c), abs(d["l"]-prev_c))
    tr[k] = t; prev_c = d["c"]
ranges = [days[k]["h"]-days[k]["l"] for k in full]
print(f"session-day RANGE pts: n={len(ranges)} median={st.median(ranges):.1f} mean={st.mean(ranges):.1f} min={min(ranges):.1f} max={max(ranges):.1f}")
print("--- per session-day: range, TR, bars, close ---")
sample_days = {'2026-08-18','2026-08-19','2026-08-20','2026-08-23','2026-08-24','2026-08-25','2026-08-26','2026-08-27','2026-08-30','2026-08-31','2026-09-01','2026-09-02'}
for k in full:
    d = days[k]; mark = "*" if k in sample_days else " "
    print(f"{mark} {k} range={d['h']-d['l']:7.2f} TR={tr[k]:7.2f} bars={d['n']:3d} close={d['c']:.2f}")
sr = [days[k]["h"]-days[k]["l"] for k in full if k in sample_days]
nr = [days[k]["h"]-days[k]["l"] for k in full if k not in sample_days]
if sr: print(f"sample-day ranges: n={len(sr)} median={st.median(sr):.1f} mean={st.mean(sr):.1f}")
if nr: print(f"other-day ranges:  n={len(nr)} median={st.median(nr):.1f} mean={st.mean(nr):.1f}")
# 5m ATR(14) distribution over the whole history vs the sample window (regime proxy for the stop floor)
atr = []; prev=None; trs=[]
for ms,o,h,l,c in rows:
    t = h-l if prev is None else max(h-l, abs(h-prev), abs(l-prev)); trs.append(t); prev=c
    if len(trs)>=14: atr.append((ms, sum(trs[-14:])/14))
def q(a,p):
    a=sorted(a); i=(len(a)-1)*p; lo=math.floor(i); hi=math.ceil(i); return a[lo]+(a[hi]-a[lo])*(i-lo)
vals=[v for _,v in atr]
print(f"ATR5m(14) all history: n={len(vals)} p10={q(vals,.1):.2f} p50={q(vals,.5):.2f} p90={q(vals,.9):.2f} p99={q(vals,.99):.2f} max={max(vals):.2f}")
# RTH-only (08:30-15:00 CT) ATR5m
rth=[v for ms,v in atr if 8.5 <= (lambda t: t.hour + t.minute/60)(datetime.datetime.fromtimestamp(ms/1000, ct)) < 15]
print(f"ATR5m(14) RTH only: n={len(rth)} p10={q(rth,.1):.2f} p50={q(rth,.5):.2f} p90={q(rth,.9):.2f} p99={q(rth,.99):.2f}  -> 1.5x floor at p50={1.5*q(rth,.5):.1f} pts (${3*q(rth,.5):.0f}), at p90={1.5*q(rth,.9):.1f} pts (${3*q(rth,.9):.0f})")
