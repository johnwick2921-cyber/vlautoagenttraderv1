import sqlite3, csv, math, statistics as st, datetime
from collections import defaultdict, Counter
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (float('nan'),0,0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n)); return (p,(c-h)/d,(c+h)/d)
def pct(a,p):
    a=sorted(a)
    if not a: return None
    k=(len(a)-1)*p; f=int(k); c=min(f+1,len(a)-1); return round(a[f]+(a[c]-a[f])*(k-f),2)
def f(x,d=2): return 'None' if x is None else (f"{x:.{d}f}" if isinstance(x,float) else str(x))
# ---- 5m ATR14 (Wilder) from 1m MNQ bars
bars=con.execute("SELECT DISTINCT open_time_ms, o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
b5={}
for t,o,h,l,c in bars:
    k=(t//300000)*300000
    if k not in b5: b5[k]=[o,h,l,c]
    else:
        b5[k][1]=max(b5[k][1],h); b5[k][2]=min(b5[k][2],l); b5[k][3]=c
keys=sorted(b5); atr={}; prev=None; a=None; trs=[]
for i,k in enumerate(keys):
    o,h,l,c=b5[k]
    tr=(h-l) if prev is None else max(h-l,abs(h-prev),abs(l-prev)); prev=c
    trs.append(tr)
    if len(trs)==14: a=sum(trs)/14
    elif len(trs)>14: a=(a*13+tr)/14
    if a is not None: atr[k]=a
def atr_at(ms):
    # last CLOSED 5m bar before ms, require bar within 30 min (gap guard)
    k=((ms//300000)-1)*300000
    for j in range(0,6):
        kk=k-j*300000
        if kk in atr: return atr[kk]
    return None
# ---- trades
rows=list(csv.DictReader(open('q13_trades_enriched.csv')))
pos={int(r['id']):r for r in rows}
et={r['id']:r for r in con.execute("SELECT id, entry_time, exit_time FROM trader_positions WHERE id BETWEEN 521 AND 591")} if False else {}
for r in con.execute("SELECT id, entry_time, exit_time FROM trader_positions WHERE id BETWEEN 521 AND 591"):
    if r[0] in pos: pos[r[0]]['entry_ms']=r[1]; pos[r[0]]['exit_ms']=r[2]
T=[]
for r in rows:
    if r['source']=='e7_farside_test': continue
    if r['pnl']=='' : continue
    d=dict(r); d['id']=int(r['id']); d['pnl']=float(r['pnl']); d['pnl_pts']=float(r['pnl_pts']); d['hold']=float(r['hold_min'])
    d['stop_pts']=float(r['stop_pts']) if r['stop_pts'] else None
    d['mfe']=float(r['mfe']) if r['mfe'] else None; d['mae']=float(r['mae']) if r['mae'] else None
    d['atr']=atr_at(int(d['entry_ms']))
    d['R']=(d['pnl_pts']/d['stop_pts']) if d['stop_pts'] else None
    d['plan_rr']=float(r['plan_rr']) if r['plan_rr'] else None
    d['hour']=int(r['hour_ct'])
    T.append(d)
ids=[d['id'] for d in T]
print(f"USABLE n={len(T)} ids={ids[0]}..{ids[-1]} (excluded: e7 572-574; NULL pnl 576,577,579)")
print("with stop:", sum(1 for d in T if d['stop_pts']), " with ATR5m:", sum(1 for d in T if d['atr']), " no-stop ids:", [d['id'] for d in T if not d['stop_pts']], " no-ATR ids:", [d['id'] for d in T if not d['atr']])
def summary(name, S):
    n=len(S); W=[d for d in S if d['pnl']>0]; L=[d for d in S if d['pnl']<0]; F=[d for d in S if d['pnl']==0]
    s=sum(d['pnl'] for d in S); m=s/n if n else float('nan')
    sd=st.pstdev([d['pnl'] for d in S]) if n>1 else 0; se=sd/math.sqrt(n) if n else 0
    dec=len(W)+len(L); p,lo,hi=wilson(len(W),dec)
    aw=sum(d['pnl'] for d in W)/len(W) if W else 0; al=sum(d['pnl'] for d in L)/len(L) if L else 0
    pay=(aw/abs(al)) if al else float('nan')
    Rs=[d['R'] for d in S if d['R'] is not None]
    print(f"{name:22s} n={n:3d} W/L/F={len(W)}/{len(L)}/{len(F)} sum={s:8.2f} mean={m:7.2f} [{m-1.96*se:7.2f},{m+1.96*se:7.2f}] win={p:.3f} [{lo:.3f},{hi:.3f}] avgW={aw:7.2f} avgL={al:7.2f} payoff={pay:.2f} " + (f"meanR={sum(Rs)/len(Rs):.2f} medR={pct(Rs,.5)}" if Rs else "") + ("" if n>=30 else "  <n=30: no verdict>") + f"  ids={[d['id'] for d in S] if n<=12 else ''}")
print("\n=== BOOK, all usable ===")
summary("ALL", T)
print("\n=== by session (plan_session) ===")
for s in ('ASIA','LONDON','NY',''):
    summary(s or 'off-plan/NULL', [d for d in T if d['session']==s])
print("\n=== by side ==="); 
for s in ('long','short'): summary(s,[d for d in T if d['side']==s])
print("\n=== by condition ===")
for c,_ in Counter(d['cond'] for d in T).most_common(): summary(c or '(none)',[d for d in T if d['cond']==c])
print("\n=== by entry path ===")
summary("decision (system)",[d for d in T if d['source']=='system'])
summary("arm fill (armed/reconcile)",[d for d in T if d['source']!='system' and d['cond']])
summary("reconcile off-plan",[d for d in T if d['source']=='reconcile' and not d['cond']])
print("\n=== by entry hour CT ===")
for h in sorted(set(d['hour'] for d in T)): summary(f"hour {h:02d}",[d for d in T if d['hour']==h])
print("\n=== by hour bucket ===")
buckets={'ASIA 17-01':lambda h:h>=17 or h<2,'LDN 02-08':lambda h:2<=h<8,'NY AM 08-11':lambda h:8<=h<11,'NY MID 11-13':lambda h:11<=h<13,'NY PM 13-15':lambda h:13<=h<15}
for k,fn in buckets.items(): summary(k,[d for d in T if fn(d['hour'])])
print("\n=== by confirm rule ===")
for c,_ in Counter(d['rule'] for d in T).most_common(): summary(c or '(none)',[d for d in T if d['rule']==c])
print("\n=== by day_type ===")
for c,_ in Counter(d['day_type'] for d in T).most_common(): summary(c or '(none)',[d for d in T if d['day_type']==c])
print("\n=== by plan bias vs side ===")
summary("with-bias",[d for d in T if d['plan_bias'] and d['plan_bias']==d['side']])
summary("against-bias",[d for d in T if d['plan_bias'] and d['plan_bias'] not in ('',d['side'],'neutral')])
summary("neutral/none",[d for d in T if not d['plan_bias'] or d['plan_bias']=='neutral'])
print("\n=== by realized-vol tercile (ATR5m at entry) ===")
A=[d for d in T if d['atr']]; cuts=(pct([d['atr'] for d in A],1/3), pct([d['atr'] for d in A],2/3)); print("ATR5m terciles cut at", cuts)
summary("low ATR",[d for d in A if d['atr']<=cuts[0]]); summary("mid ATR",[d for d in A if cuts[0]<d['atr']<=cuts[1]]); summary("high ATR",[d for d in A if d['atr']>cuts[1]])
print("\n=== regime label (plans.doc.bias_label) coverage ===", Counter((d['regime'] or 'absent')[-12:] for d in T))
print("\n=== hold time (min) ===")
for nm,S in (('all',T),('winners',[d for d in T if d['pnl']>0]),('losers',[d for d in T if d['pnl']<0])):
    hs=[d['hold'] for d in S]; print(f"{nm:8s} n={len(hs)} p25={pct(hs,.25)} median={pct(hs,.5)} p75={pct(hs,.75)} max={max(hs)}")
print("\n=== stops ===")
S=[d for d in T if d['stop_pts']]; sp=[d['stop_pts'] for d in S]
print(f"stop pts n={len(sp)} p25={pct(sp,.25)} median={pct(sp,.5)} p75={pct(sp,.75)} min={min(sp)} max={max(sp)}")
SA=[d for d in S if d['atr']]; sa=[d['stop_pts']/d['atr'] for d in SA]
print(f"stop/ATR5m n={len(sa)} p25={pct(sa,.25)} median={pct(sa,.5)} p75={pct(sa,.75)}; share <1.0×: {sum(1 for x in sa if x<1.0)}, in [1.0,1.5): {sum(1 for x in sa if 1.0<=x<1.5)}, in [1.5,2.0): {sum(1 for x in sa if 1.5<=x<2.0)}, >=2: {sum(1 for x in sa if x>=2)}")
print("stop source counts:", Counter(d['stop_src'].split('#')[0] for d in T))
print("\n=== planned R:R (decision/arm target vs stop) vs realized ===")
pr=[d['plan_rr'] for d in T if d['plan_rr']]; print(f"planned RR at entry n={len(pr)} p25={pct(pr,.25)} median={pct(pr,.5)} p75={pct(pr,.75)}; in[2.0,2.3)={sum(1 for x in pr if 2.0<=x<2.3)} in[3.0,3.3)={sum(1 for x in pr if 3.0<=x<3.3)} <2={sum(1 for x in pr if x<2)}")
W=[d for d in T if d['pnl']>0]; L=[d for d in T if d['pnl']<0]
print("\n=== excursions (trader_positions.mae/mfe, pts) ===")
for nm,S in (('winners',W),('losers',L)):
    mf=[d['mfe'] for d in S if d['mfe'] is not None]; ma=[d['mae'] for d in S if d['mae'] is not None]
    print(f"{nm:8s} MFE n={len(mf)} p50={pct(mf,.5)} p80={pct(mf,.8)} p95={pct(mf,.95)} | MAE p50={pct(ma,.5)} p80={pct(ma,.8)} p95={pct(ma,.95)}")
print("\n=== MFE by condition (pts) p50/p80/p95 ===")
for c,_ in Counter(d['cond'] for d in T).most_common():
    mf=[d['mfe'] for d in T if d['cond']==c and d['mfe'] is not None]
    print(f"{(c or '(none)'):16s} n={len(mf):2d} p50={pct(mf,.5)} p80={pct(mf,.8)} p95={pct(mf,.95)}" + ("" if len(mf)>=30 else "  <n=30 descriptive>"))
print("\n=== MFE/MAE in STOP units (actual initial stop) ===")
SS=[d for d in T if d['stop_pts'] and d['mfe'] is not None]
for nm,S in (('winners',[d for d in SS if d['pnl']>0]),('losers',[d for d in SS if d['pnl']<0])):
    mfs=[d['mfe']/d['stop_pts'] for d in S]; mas=[d['mae']/d['stop_pts'] for d in S]
    print(f"{nm:8s} n={len(S)} MFE/stop p50={pct(mfs,.5)} p80={pct(mfs,.8)} max={round(max(mfs),2)} | MAE/stop p50={pct(mas,.5)} p80={pct(mas,.8)} max={round(max(mas),2)} | >=2.0R: {sum(1 for x in mfs if x>=2.0)} | winners max MAE/stop list: {sorted(round(x,2) for x in mas)}")
print("reached 2R off ACTUAL stop:", sum(1 for d in SS if d['mfe']>=2*d['stop_pts']), "of", len(SS), "ids", [d['id'] for d in SS if d['mfe']>=2*d['stop_pts']])
SM=[d for d in SS if d['atr']]
print("reached 2R off MIN stop (1.5×ATR5m):", sum(1 for d in SM if d['mfe']>=2*1.5*d['atr']), "of", len(SM), "ids", [d['id'] for d in SM if d['mfe']>=2*1.5*d['atr']], "| reached 1R off min stop:", sum(1 for d in SM if d['mfe']>=1.5*d['atr']))
print("winners MAE/floor(1.5xATR):", sorted(round(d['mae']/(1.5*d['atr']),2) for d in SM if d['pnl']>0))
print("losers  MAE/floor(1.5xATR):", sorted(round(d['mae']/(1.5*d['atr']),2) for d in SM if d['pnl']<0))
print("losers MAE >= 0.95×stop (stopped):", sum(1 for d in SS if d['pnl']<0 and d['mae']>=0.95*d['stop_pts']), "of", len([d for d in SS if d['pnl']<0]))
gb=[(d['mfe']-d['pnl_pts']) for d in W if d['mfe'] is not None]; print(f"winners giveback (MFE-realized pts) n={len(gb)} median={pct(gb,.5)} p75={pct(gb,.75)}; median MFE={pct([d['mfe'] for d in W if d['mfe'] is not None],.5)} median realized pts={pct([d['pnl_pts'] for d in W],.5)}")
print("\n=== stop-in-ATR counterfactual (in-sample, descriptive) ===")
for m in (0.75,1.0,1.25,1.5,2.0):
    surv=[d for d in SM if d['pnl']>0 and d['mae']<m*d['atr']]; cut=[d for d in SM if d['pnl']<0 and d['mae']>=m*d['atr']]
    print(f"  stop {m}×ATR5m: winners kept {len(surv)}/{len([d for d in SM if d['pnl']>0])}, losers stopped {len(cut)}/{len([d for d in SM if d['pnl']<0])}")
print("\n=== per-day ===")
byday=defaultdict(list)
for d in T: byday[d['entry_ct'][:5]].append(d['pnl'])
for k in sorted(byday): print(f"  {k} n={len(byday[k])} sum={sum(byday[k]):8.2f}")
print("days with trades:", len(byday), " trades/day:", round(len(T)/len(byday),2))
print("\n=== reject in NY (prompt line 737 claims 75% win +665 'this week') ===")
summary("reject×NY",[d for d in T if d['cond']=='reject' and d['session']=='NY'])
summary("reject×LONDON",[d for d in T if d['cond']=='reject' and d['session']=='LONDON'])
summary("reject×ASIA",[d for d in T if d['cond']=='reject' and d['session']=='ASIA'])
# mae/mfe spot check vs bars
print("\n=== spot-check stored mae/mfe vs 1m bars (3 trades) ===")
b1=[(t,o,h,l,c) for t,o,h,l,c in bars]
import bisect
ts=[b[0] for b in b1]
for pid in (532,589,591):
    d=pos[pid]; e=int(d['entry_ms']); x=int(d['exit_ms']); ep=float(d['entry']); side=d['side']
    i=bisect.bisect_left(ts,(e//60000)*60000); j=bisect.bisect_right(ts,x)
    seg=b1[i:j]
    if not seg: print(pid,'no bars'); continue
    hi=max(b[2] for b in seg); lo=min(b[3] for b in seg)
    mfe=(hi-ep) if side=='long' else (ep-lo); mae=(ep-lo) if side=='long' else (hi-ep)
    print(f"  {pid} {side} bars={len(seg)} recomputed MFE={mfe:.2f} MAE={mae:.2f} | stored MFE={d['mfe']} MAE={d['mae']}")
# write per-trade table
w=csv.writer(open('q21_trades_final.csv','w')); w.writerow(['id','source','session','side','entry_ct','hour','hold_min','cond','rule','quality','stop_src','stop_pts','atr5m','stop_atr','plan_rr','pnl_usd','pnl_pts','R','mfe','mae','mfe_over_stop','mae_over_stop','reached_2R_actual','reached_2R_minstop','plan_bias','day_type','grade'])
for d in T: w.writerow([d['id'],d['source'],d['session'],d['side'],d['entry_ct'],d['hour'],d['hold'],d['cond'],d['rule'],d['quality'],d['stop_src'],d['stop_pts'],round(d['atr'],2) if d['atr'] else None, round(d['stop_pts']/d['atr'],2) if (d['atr'] and d['stop_pts']) else None, d['plan_rr'], d['pnl'], d['pnl_pts'], round(d['R'],2) if d['R'] is not None else None, d['mfe'], d['mae'], round(d['mfe']/d['stop_pts'],2) if (d['mfe'] is not None and d['stop_pts']) else None, round(d['mae']/d['stop_pts'],2) if (d['mae'] is not None and d['stop_pts']) else None, (d['mfe']>=2*d['stop_pts']) if (d['mfe'] is not None and d['stop_pts']) else None, (d['mfe']>=3*d['atr']) if (d['mfe'] is not None and d['atr']) else None, d['plan_bias'], d['day_type'], d['grade']])
