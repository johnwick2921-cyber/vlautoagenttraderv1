import csv,math
from lib import *
UNRES={530,539,545,546,566,571,580}
rows=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-01-0905/q21_trades_final.csv')))
for r in rows:
    r['id']=int(r['id']); r['pnl']=float(r['pnl_usd'])
    r['Rv']=float(r['R']) if r['R'] not in ('','None') else None
comp=[r for r in rows if r['id'] not in UNRES]
print("csv rows",len(rows),"compliant",len(comp))
for name,pop in (("n65",rows),("n58",comp)):
    p=[r['pnl'] for r in pop]
    w=[x for x in p if x>0]; l=[x for x in p if x<0]; f=[x for x in p if x==0]
    m=mean(p); sp=sd_pop(p); ss=sd_samp(p)
    sep=sp/math.sqrt(len(p)); ses=ss/math.sqrt(len(p))
    wr=len(w)/(len(w)+len(l))
    lo,hi=wilson(len(w),len(w)+len(l))
    payoff=mean(w)/abs(mean(l))
    be=abs(mean(l))/(mean(w)+abs(mean(l)))
    print(f"\n== {name}: n {len(p)} W {len(w)} L {len(l)} F {len(f)} sum {sum(p):.2f} mean {m:.4f}")
    print(f"   sd_pop {sp:.4f} sd_samp {ss:.4f}  CI_pop [{m-1.96*sep:.2f},{m+1.96*sep:.2f}]  CI_samp [{m-1.96*ses:.2f},{m+1.96*ses:.2f}]")
    print(f"   win {len(w)}/{len(w)+len(l)} = {wr:.4f} [{lo:.3f},{hi:.3f}]  avgW {mean(w):.2f} avgL {mean(l):.2f} payoff {payoff:.4f} BE {be:.4f}")
    print(f"   under BE by {100*(be-wr):.2f} pts")
    # required n for CI on mean to exclude zero (95% sig only) and at 80% power
    print(f"   n@95%sig-only pop {(1.96*sp/abs(m))**2:.0f} samp {(1.96*ss/abs(m))**2:.0f} ; n@80%power pop {((1.96+0.8416)*sp/abs(m))**2:.0f} samp {((1.96+0.8416)*ss/abs(m))**2:.0f}")
# by side
print("\n== by side (n58)")
for s in ('short','long'):
    p=[r['pnl'] for r in comp if r['side']==s]
    w=len([x for x in p if x>0]);l=len([x for x in p if x<0]);f=len([x for x in p if x==0])
    print(f"  {s}: n {len(p)} {w}/{l}/{f} sum {sum(p):.2f}")
# by condition
print("\n== by condition (n58) dollars + mean R")
conds={}
for r in comp: conds.setdefault(r['cond'] or '(none)',[]).append(r)
for c,rs in sorted(conds.items(),key=lambda kv:-len(kv[1])):
    p=[r['pnl'] for r in rs]; w=len([x for x in p if x>0]);l=len([x for x in p if x<0]);f=len([x for x in p if x==0])
    Rs=[r['Rv'] for r in rs if r['Rv'] is not None]
    lo,hi=wilson(w,w+l) if w+l else (0,0)
    print(f"  {c:18s} n {len(rs):2d}  {w}/{l}/{f}  sum {sum(p):+9.2f}  win {100*w/(w+l) if w+l else 0:5.1f} [{100*lo:.1f},{100*hi:.1f}]  meanR {mean(Rs) if Rs else None} (nR={len(Rs)})")
