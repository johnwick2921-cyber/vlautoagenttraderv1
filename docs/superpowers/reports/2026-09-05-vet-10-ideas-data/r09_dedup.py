import sqlite3, math
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (0.0,0.0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); m=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return ((c-m)/d,(c+m)/d)
rows=con.execute("select level_kind, session, ordinal, level_price, opened_at_ms, outcome from touch_outcomes").fetchall()
ep={}
for k,s,o,p,t,out in rows:
    key=(k,p,t)
    ep.setdefault(key,{'sess':set(),'ord':set(),'out':set()})
    ep[key]['sess'].add(s); ep[key]['ord'].add(o); ep[key]['out'].add(out)
def collapse(outs):
    return sorted(outs)[0]   # ambiguous_horizon < break < hold
print(f"rows={len(rows)} episodes={len(ep)}  conflicting-outcome episodes={sum(1 for v in ep.values() if len(v['out'])>1)}")
agg={}
for (k,p,t),v in ep.items():
    out=collapse(v['out']); s=sorted(v['sess'])[0]
    agg.setdefault((k,'ALL'),[0,0,0]); agg.setdefault((k,s),[0,0,0])
    for key in ((k,'ALL'),(k,s)):
        agg[key][0]+= out=='hold'; agg[key][1]+= out=='break'; agg[key][2]+= out=='ambiguous_horizon'
print(f"{'kind/session':<22}{'hold':>5}{'brk':>5}{'amb':>5}{'res':>5}  p(hold) [Wilson]           p(break)")
for key in sorted(agg, key=lambda x:(-sum(agg[x]),x)):
    h,b,a=agg[key]; res=h+b
    if h+b+a<8: continue
    lo,hi=wilson(h,res) if res else (0,0); lb,hb=wilson(b,res) if res else (0,0)
    print(f"{key[0]+'/'+key[1]:<22}{h:>5}{b:>5}{a:>5}{res:>5}  {h/res if res else 0:.3f} [{lo:.3f},{hi:.3f}]   {b/res if res else 0:.3f} [{lb:.3f},{hb:.3f}]")
# first touch dedup
ft={'hold':0,'break':0,'ambiguous_horizon':0}
for (k,p,t),v in ep.items():
    if 1 in v['ord']: ft[collapse(v['out'])]+=1
res=ft['hold']+ft['break']
lo,hi=wilson(ft['hold'],res)
print(f"\nfirst-touch (ordinal 1 in any read), deduplicated: hold {ft['hold']} break {ft['break']} amb {ft['ambiguous_horizon']} → p(hold) {ft['hold']/res:.3f} [{lo:.3f},{hi:.3f}] n={res}")
print(f"   ambiguity-bounded p(hold) in [{ft['hold']/(res+ft['ambiguous_horizon']):.3f},{(ft['hold']+ft['ambiguous_horizon'])/(res+ft['ambiguous_horizon']):.3f}]")

DYN={'VWAP','eVWAP'}
ftS={'hold':0,'break':0,'ambiguous_horizon':0}; allS={'hold':0,'break':0,'ambiguous_horizon':0}
for (k,p,t),v in ep.items():
    if k in DYN: continue
    allS[collapse(v['out'])]+=1
    if 1 in v['ord']: ftS[collapse(v['out'])]+=1
for nm,d in (('static-kind first-touch',ftS),('static-kind all episodes',allS)):
    res=d['hold']+d['break']; lo,hi=wilson(d['hold'],res)
    print(f"{nm}: hold {d['hold']} break {d['break']} amb {d['ambiguous_horizon']} → p(hold) {d['hold']/res:.3f} [{lo:.3f},{hi:.3f}] n={res}")
print("episodes by kind class: dynamic(VWAP/eVWAP)=%d static=%d"%(sum(1 for k in ep if k[0] in DYN), sum(1 for k in ep if k[0] not in DYN)))
# distinct prices per static kind
import collections
c=collections.Counter((k[0],k[1]) for k in ep if k[0] not in DYN)
pk=collections.defaultdict(set)
for k in ep:
    if k[0] not in DYN: pk[k[0]].add(k[1])
print("distinct prices per static kind:", {k:len(v) for k,v in sorted(pk.items(), key=lambda x:-len(x[1]))})
