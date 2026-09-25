#!/usr/bin/env python3
"""q10: the LIVE touch_outcomes corpus, deduplicated on (kind, price, opened_at_ms), with Wilson 95% and a static/dynamic flag.
Dynamic kinds (VWAP*, eVWAP, SWG-*, OR-* before 08:35, POC of the read day) are re-emitted at a NEW price per read and scanned
over bars that PREDATE the level's existence -> lookahead-contaminated; static prior-day anchors are the only clean rows."""
import sqlite3, math, collections
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (cen-half,cen+half)
DYN={'VWAP','VWAP±2σ','eVWAP','SWG-H','SWG-L','POC','OR-H','OR-L','OB','DEMAND','SUPPLY','FVG','EQL'}
rows=con.execute("""WITH d AS (SELECT level_kind k, level_price p, opened_at_ms t, MIN(outcome) o, COUNT(*) c FROM touch_outcomes GROUP BY 1,2,3)
SELECT k, COUNT(*) eps, SUM(c) raw, COUNT(DISTINCT p) prices, SUM(o='hold') h, SUM(o='break') b, SUM(o LIKE 'ambiguous%') a FROM d GROUP BY k ORDER BY eps DESC""").fetchall()
print(f"{'kind':10s} {'raw':>4s} {'eps':>4s} {'prices':>6s} {'hold':>4s} {'brk':>4s} {'amb':>4s} {'p_hold':>6s} {'wilson95':>16s} {'amb%':>5s} class")
T=dict(raw=0,eps=0,h=0,b=0,a=0)
for k,eps,raw,prices,h,b,a in rows:
    n=h+b; p=h/n if n else float('nan'); lo,hi=wilson(h,n)
    print(f"{k:10s} {raw:4d} {eps:4d} {prices:6d} {h:4d} {b:4d} {a:4d} {p:6.3f} [{lo:.3f},{hi:.3f}] {100*a/eps:5.1f} {'DYNAMIC/lookahead' if k in DYN else 'static'}")
    T['raw']+=raw; T['eps']+=eps; T['h']+=h; T['b']+=b; T['a']+=a
n=T['h']+T['b']; lo,hi=wilson(T['h'],n)
print(f"{'TOTAL':10s} {T['raw']:4d} {T['eps']:4d} {'':6s} {T['h']:4d} {T['b']:4d} {T['a']:4d} {T['h']/n:6.3f} [{lo:.3f},{hi:.3f}] {100*T['a']/T['eps']:5.1f}")
st=[r for r in rows if r[0] not in DYN]; h=sum(r[4] for r in st); b=sum(r[5] for r in st); a=sum(r[6] for r in st); n=h+b; lo,hi=wilson(h,n)
print(f"STATIC-ONLY pooled: eps={sum(r[1] for r in st)} hold={h} brk={b} amb={a} p_hold={h/n:.3f} [{lo:.3f},{hi:.3f}]")
# the prior review's D5b snapshot numbers, for the record
print("\nD5b snapshot (research-conformance-data, 09-04 ~08:30 CT): RTH-L 20/63 =", f"{20/63:.4f}", wilson(20,63), "| dedup today: RTH-L 4/12")
# how many reads re-recorded the same RTH-L episode
print(con.execute("SELECT level_price, COUNT(DISTINCT plan_id||'/'||plan_version) reads, COUNT(*) rows, COUNT(DISTINCT opened_at_ms) episodes FROM touch_outcomes WHERE level_kind='RTH-L'").fetchall())
print("which day's RTH-L is 29199.25? bars RTH 08:30-15:00 CT low per calendar day:")
for r in con.execute("SELECT date((open_time_ms/1000)-5*3600,'unixepoch') d, MIN(l) FROM bars WHERE symbol='MNQ' AND tf='1m' AND ((open_time_ms/1000)-5*3600)%86400 BETWEEN 30600 AND 53999 AND d BETWEEN '2026-09-01' AND '2026-09-03' GROUP BY 1"): print(r)
