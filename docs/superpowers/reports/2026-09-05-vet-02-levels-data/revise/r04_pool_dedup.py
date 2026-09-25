#!/usr/bin/env python3
"""r04: candidate_pool 09-04 seated vs cut, DEDUPLICATED per distinct (kind, price):
classify each distinct level by its status at its FIRST appearance (and by ever-seated), scan forward from that first read_at."""
import sqlite3, math, collections
exec(open('/home/hoang/nofx-analysis/vet-02-0905/q11_replay.py').read().split("# group bars by session day")[0])
by_sd=collections.OrderedDict()
for b in bars: by_sd.setdefault(sess_day(b['t']),[]).append(b)
sdays=sorted(by_sd)
def delta_for(day):
    idx=sdays.index(day); trail=[b for dd in sdays[max(0,idx-5):idx] for b in by_sd[dd]]
    return sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/max(1,len(trail)-1)
def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (cen-half,cen+half)
con2=sqlite3.connect(DB, uri=True); con2.row_factory=sqlite3.Row
lv={}
for r in con2.execute("SELECT read_at_ms, level_price, level_kind, seated FROM candidate_pool ORDER BY read_at_ms"):
    k=(r['level_kind'], float(r['level_price']))
    if k not in lv: lv[k]=dict(first_read=r['read_at_ms'], first_seated=r['seated'], ever=0, reads=0)
    lv[k]['reads']+=1; lv[k]['ever']=max(lv[k]['ever'], r['seated'])
print("distinct (kind,price):", len(lv), "first-seated:", sum(1 for v in lv.values() if v['first_seated']), "ever-seated:", sum(1 for v in lv.values() if v['ever']))
res=[]
for (kind,px),v in lv.items():
    day=sess_day(v['first_read']); tape=[b for b in by_sd[day] if b['t']>=v['first_read']]
    eps=detect(tape, px, 3.0, delta_for(day), 12, 'close')
    res.append(dict(kind=kind,px=px,first_seated=v['first_seated'],ever=v['ever'],touched=bool(eps),first=(eps[0]['outcome'] if eps else 'untouched'),dist=abs(px-tape[0]['c'])))
def tab(name, keyf):
    g=collections.defaultdict(lambda: dict(n=0,t=0,h=0,b=0,a=0,d=0.0))
    for x in res:
        a=g[keyf(x)]; a['n']+=1; a['d']+=x['dist']; a['t']+=x['touched']
        if x['first']=='hold': a['h']+=1
        elif x['first']=='break': a['b']+=1
        elif x['touched']: a['a']+=1
    print(f"\n== {name}")
    for k,a in g.items():
        n1=a['h']+a['b']; p=a['h']/n1 if n1 else float('nan'); lo,hi=wilson(a['h'],n1)
        print(f"{k:12s} levels {a['n']:3d} touched {a['t']:3d} ({100*a['t']/a['n']:.1f}%) 1st hold/brk/amb {a['h']}/{a['b']}/{a['a']} p={p:.3f} [{lo:.3f},{hi:.3f}] n={n1} mean_dist {a['d']/a['n']:.1f}")
tab("status at FIRST appearance", lambda x:'seated' if x['first_seated'] else 'cut')
tab("EVER seated in any read", lambda x:'seated' if x['ever'] else 'cut')
