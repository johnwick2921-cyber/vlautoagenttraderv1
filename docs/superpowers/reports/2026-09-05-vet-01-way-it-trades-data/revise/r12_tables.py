import csv,datetime
from lib import *
UNRES={530,539,545,546,566,571,580}
rows=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-01-0905/q21_trades_final.csv')))
for r in rows:
    r['id']=int(r['id']); r['pnl']=float(r['pnl_usd'])
    r['Rv']=float(r['R']) if r['R'] not in ('','None') else None
    r['atr']=float(r['atr5m']) if r['atr5m'] else None
    # entry_ct like '08-19 03:22'
    hh,mm=r['entry_ct'].split(' ')[1].split(':'); r['minofday']=int(hh)*60+int(mm)
comp=[r for r in rows if r['id'] not in UNRES]
def blk(name,rs):
    p=[r['pnl'] for r in rs]; w=len([x for x in p if x>0]);l=len([x for x in p if x<0]);f=len([x for x in p if x==0])
    R=[r['Rv'] for r in rs if r['Rv'] is not None]
    mr=f"{mean(R):+.3f}" if R else "n/a"
    md=f"{pct(R,0.5):+.3f}" if R else "n/a"
    print(f"  {name:22s} n {len(rs):2d}  {w}/{l}/{f}  Σ {sum(p):+9.2f}  meanR {mr} (nR {len(R)}) medR {md}")
for lab,pop in (("n58 COMPLIANT",comp),("n65 sensitivity",rows)):
    print(f"\n#### {lab}")
    print("-- by session")
    for s in ('ASIA','LONDON','NY','','None'):
        rs=[r for r in pop if r['session']==s]
        if rs: blk(s or '(off-plan)',rs)
    print("-- by hour bucket (CT)")
    buckets=[("17:00-02:00",lambda m: m>=17*60 or m<2*60),("02:00-08:00",lambda m: 2*60<=m<8*60),
             ("08:00-11:00",lambda m: 8*60<=m<11*60),("11:00-13:00",lambda m: 11*60<=m<13*60),
             ("13:00-15:00",lambda m: 13*60<=m<15*60)]
    for nm,fn in buckets:
        rs=[r for r in pop if fn(r['minofday'])]
        if rs: blk(nm,rs)
    print("-- prompt's privileged window, rendered")
    for nm,fn in (("07:30-10:00 CT",lambda m: 7*60+30<=m<10*60),("08:30-11:00 CT",lambda m: 8*60+30<=m<11*60),("09:00-10:00 CT",lambda m: 9*60<=m<10*60)):
        rs=[r for r in pop if fn(r['minofday'])]
        blk(nm,rs)
    print("-- ATR5m tercile (cuts from THIS population)")
    withatr=[r for r in pop if r['atr']]
    vals=sorted(r['atr'] for r in withatr)
    c1=pct(vals,1/3); c2=pct(vals,2/3)
    print(f"   cuts {c1:.2f} / {c2:.2f}  (n with ATR {len(withatr)})")
    blk("low",[r for r in withatr if r['atr']<=c1]); blk("mid",[r for r in withatr if c1<r['atr']<=c2]); blk("high",[r for r in withatr if r['atr']>c2])
    print("-- by side")
    for s in ('short','long'): blk(s,[r for r in pop if r['side']==s])
    print("-- session x side")
    for s in ('ASIA','LONDON','NY'):
        for sd in ('short','long'):
            rs=[r for r in pop if r['session']==s and r['side']==sd]
            if rs: blk(f"{s} {sd}",rs)
    print("-- by entry path (stop_src)")
    for k in ('arm','decision','plan'):
        rs=[r for r in pop if r['stop_src'].startswith(k)]
        if rs: blk(k,rs)
    print("-- by confirm rule")
    rl={}
    for r in pop: rl.setdefault(r['rule'] or '(none)',[]).append(r)
    for k,rs in sorted(rl.items(),key=lambda kv:-len(kv[1])): blk(k,rs)
    print("-- reject x session")
    for s in ('ASIA','LONDON','NY'):
        rs=[r for r in pop if r['cond']=='reject' and r['session']==s]
        if rs: blk(f"reject {s}",rs)
    print("-- plan bias vs side")
    bs={}
    for r in pop:
        b=(r['plan_bias'] or 'none').lower(); sd=r['side']
        if b in ('up','long','bullish'): k='with' if sd=='long' else 'against'
        elif b in ('down','short','bearish'): k='with' if sd=='short' else 'against'
        else: k='neutral/none'
        bs.setdefault(k,[]).append(r)
    for k,rs in bs.items(): blk(k,rs)
    print("-- day_type")
    dt={}
    for r in pop: dt.setdefault(r['day_type'] or '(none)',[]).append(r)
    for k,rs in sorted(dt.items(),key=lambda kv:-len(kv[1])): blk(k,rs)
