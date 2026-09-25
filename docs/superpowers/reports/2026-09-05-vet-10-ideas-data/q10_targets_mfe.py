# q10: target-hit rates, MFE thresholds, sensitivity w/o 568/569 artifacts, hold-time x MFE, per-session R
import csv, sys
sys.path.insert(0,'/home/hoang/nofx-analysis/vet-10-0905'); from wilson import wilson, mean_ci
rows=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-10-0905/q07_canonical_trades.csv')))
def f(x): 
    try: return float(x)
    except: return None
R=[r for r in rows if f(r['risk_pts'])]
print("n with stop:",len(R))
def stats(S,label):
    S=[r for r in S if f(r['mfe_pts']) is not None]
    n=len(S); risk=lambda r:f(r['risk_pts']); mfe=lambda r:f(r['mfe_pts'])
    tp_hit=[r for r in S if f(r['tp']) and mfe(r)>=abs(f(r['tp'])-f(r['entry']))]
    for k in (0.5,1.0,1.5,2.0,3.0):
        c=sum(1 for r in S if mfe(r)>=k*risk(r)); lo,hi=wilson(c,n); print(f"  {label}: MFE>={k}R: {c}/{n}={c/n:.3f} wilson[{lo:.3f},{hi:.3f}]")
    c=len(tp_hit); lo,hi=wilson(c,n); print(f"  {label}: MFE>=planned target dist: {c}/{n}={c/n:.3f} wilson[{lo:.3f},{hi:.3f}] ids={[r['id'] for r in tp_hit]}")
    w=[r for r in S if f(r['pnl_c'])>0]; l=[r for r in S if f(r['pnl_c'])<0]
    print(f"  {label}: wins {len(w)} losses {len(l)}; avg planned RR {sum(f(r['planned_rr']) for r in S if f(r['planned_rr']))/len([r for r in S if f(r['planned_rr'])]):.2f}; median planned RR {sorted(f(r['planned_rr']) for r in S if f(r['planned_rr']))[len(S)//2]:.2f}")
    if w and l: print(f"  {label}: realized payoff avgW/avgL = {(sum(f(r['pnl_c']) for r in w)/len(w))/abs(sum(f(r['pnl_c']) for r in l)/len(l)):.2f}; avg win R {sum(f(r['realized_R']) for r in w)/len(w):.2f}; avg loss R {sum(f(r['realized_R']) for r in l)/len(l):.2f}")
    Rm=[f(r['realized_R']) for r in S]; m=mean_ci(Rm); print(f"  {label}: mean R {m[0]:.3f} CI [{m[1]:.3f},{m[2]:.3f}] n={len(Rm)}")
    rp=sorted(risk(r) for r in S); print(f"  {label}: risk pts min {rp[0]} p25 {rp[len(rp)//4]} median {rp[len(rp)//2]} p75 {rp[3*len(rp)//4]} max {rp[-1]}")
stats(R,"all")
S2=[r for r in R if r['id'] not in ('568','569')]
stats(S2,"excl 568/569")
print("--- 1R-target counterfactual (excl 568/569): if every trade exited at +1R when MFE>=1R else at actual outcome")
S=[r for r in S2 if f(r['mfe_pts']) is not None]
cf=[]; 
for r in S:
    if f(r['mfe_pts'])>=f(r['risk_pts']): cf.append(1.0)
    else: cf.append(f(r['realized_R']))
m=mean_ci(cf); print(f"  1R-exit counterfactual mean R {m[0]:.3f} CI [{m[1]:.3f},{m[2]:.3f}] n={len(cf)}; wins {sum(1 for x in cf if x>0)}")
print("  NOTE: MFE is a 1m-extreme; a resting 1R limit would need MFE>=1R+slip, and the 1m extreme may have been one tick — this is an UPPER bound, not a result")
for thr in (0.5,0.75):
    cf=[thr if f(r['mfe_pts'])>=thr*f(r['risk_pts']) else f(r['realized_R']) for r in S]; m=mean_ci(cf); print(f"  {thr}R-exit counterfactual mean R {m[0]:.3f} CI [{m[1]:.3f},{m[2]:.3f}]")
print("--- per session (excl 568/569)")
for s in ('ASIA','LONDON','NY',''):
    T=[r for r in S2 if r['session']==s]
    if not T: continue
    Rm=[f(r['realized_R']) for r in T]; m=mean_ci(Rm) if len(Rm)>1 else (Rm[0],0,0)
    w=sum(1 for x in Rm if x>0); print(f"  {s or 'off-plan'}: n={len(T)} wins={w} sumR={sum(Rm):.2f} meanR={m[0]:.3f} CI[{m[1]:.3f},{m[2]:.3f}] sum$={sum(f(r['pnl_c']) for r in T):.1f}")
print("--- per side (excl 568/569)")
for s in ('LONG','SHORT'):
    T=[r for r in S2 if r['side']==s]; Rm=[f(r['realized_R']) for r in T]; m=mean_ci(Rm); w=sum(1 for x in Rm if x>0)
    lo,hi=wilson(w,len(T)); print(f"  {s}: n={len(T)} wins={w} ({w/len(T):.3f} wilson[{lo:.3f},{hi:.3f}]) meanR={m[0]:.3f} CI[{m[1]:.3f},{m[2]:.3f}]")
print("--- losers by MFE bucket (excl 568/569): how far did losers go our way first")
L=[r for r in S2 if f(r['pnl_c'])<0 and f(r['mfe_pts']) is not None]
b={'<0.25R':0,'0.25-0.5R':0,'0.5-1R':0,'>=1R':0}
for r in L:
    x=f(r['mfe_pts'])/f(r['risk_pts'])
    k='<0.25R' if x<0.25 else '0.25-0.5R' if x<0.5 else '0.5-1R' if x<1 else '>=1R'; b[k]+=1
print(" ",b,"n=",len(L))
