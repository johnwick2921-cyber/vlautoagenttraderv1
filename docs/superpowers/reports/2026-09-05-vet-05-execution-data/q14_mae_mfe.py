import sqlite3, csv, re, math, datetime
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor()
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (round(100*(ctr-h),1), round(100*(ctr+h),1))
def pct(xs,q):
    xs=sorted(xs); 
    if not xs: return None
    i=(len(xs)-1)*q; lo=int(i); hi=min(lo+1,len(xs)-1); return round(xs[lo]+(xs[hi]-xs[lo])*(i-lo),2)
era=1786770000000
rows=c.execute("""SELECT id, side, entry_price, exit_price, entry_time, exit_time, pnl_corrected, mae, mfe, source, close_reason, plan_band, cited_scenario_id FROM trader_positions
 WHERE entry_time>=? AND pnl_corrected IS NOT NULL AND source<>'e7_farside_test' AND close_reason NOT IN ('reconcile_flat','unresolved','e7_farside_test') ORDER BY id""",(era,)).fetchall()
excluded=c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time>=? AND (pnl_corrected IS NULL OR source='e7_farside_test' OR close_reason IN ('reconcile_flat','unresolved','e7_farside_test'))",(era,)).fetchone()[0]
# exit reasons from log lines
reasons=[]
for line in open("log_nt_closes.out"):
    m=re.search(r'^(\d\d-\d\d \d\d:\d\d:\d\d).*MNQ (LONG|SHORT) qty=[\d.]+ exit=([\d.]+) reason=(\w+) pnl=(-?[\d.]+)', line)
    if m: reasons.append((m.group(1),m.group(2),float(m.group(3)),m.group(4),float(m.group(5))))
def find_reason(side, exitpx, exit_ms, pnl):
    best=None
    for t,s,ex,rs,pn in reasons:
        if s!=side.upper() or abs(ex-exitpx)>0.01: continue
        tm=datetime.datetime.strptime("2026-"+t,"%Y-%m-%d %H:%M:%S").replace(tzinfo=datetime.timezone(datetime.timedelta(hours=-5)))
        dt=abs(tm.timestamp()*1000-exit_ms)
        if dt<5*60000 and (best is None or dt<best[0]): best=(dt,rs)
    return best[1] if best else 'unknown'
def atr5m_at(ms, n=14):
    b=c.execute("SELECT h,l,c FROM bars WHERE symbol='MNQ' AND tf='5m' AND open_time_ms<? ORDER BY open_time_ms DESC LIMIT ?",(ms,n+1)).fetchall()
    if len(b)<n+1: return None
    b=b[::-1]; trs=[]
    for i in range(1,len(b)):
        h,l,pc=b[i][0],b[i][1],b[i-1][2]; trs.append(max(h-l,abs(h-pc),abs(l-pc)))
    return sum(trs)/len(trs)
out=[]
for pid,side,ent,ex,et,xt,pnl,mae,mfe,src,cr,band,scen in rows:
    rs=find_reason(side,ex,xt,pnl)
    risk=abs(ent-ex) if rs=='sl' else None
    atr=atr5m_at(et)
    out.append(dict(id=pid,side=side,entry=ent,exit=ex,entry_ct=datetime.datetime.fromtimestamp(et/1000,datetime.timezone(datetime.timedelta(hours=-5))).strftime('%m-%d %H:%M'),pnl=pnl,mae=mae,mfe=mfe,reason=rs,risk_pts=risk,atr5m=round(atr,2) if atr else None,floor=round(1.5*atr,2) if atr else None,src=src,band=band,scen=scen))
w=csv.DictWriter(open("q14_mae_mfe.csv","w"),fieldnames=list(out[0].keys())); w.writeheader(); [w.writerow(r) for r in out]
n=len(out); print("era rows used n=",n,"(excluded NULL/test/flat/unresolved:",excluded,") ids:",[r['id'] for r in out])
win=[r for r in out if r['pnl']>0 and r['mae'] is not None]; los=[r for r in out if r['pnl']<0 and r['mae'] is not None]; scr=[r for r in out if r['pnl']==0]
print("winners",len(win),"losers",len(los),"scratch",len(scr))
print("win rate (ex-scratch) %d/%d = %.1f%% Wilson %s" % (len(win),len(win)+len(los),100*len(win)/(len(win)+len(los)),wilson(len(win),len(win)+len(los))))
for name,g in (("winners",win),("losers",los)):
    m=[r['mae'] for r in g if r['mae'] is not None]; f=[r['mfe'] for r in g if r['mfe'] is not None]
    print(f"{name}: MAE n={len(m)} p50={pct(m,.5)} p80={pct(m,.8)} p95={pct(m,.95)} | MFE n={len(f)} p50={pct(f,.5)} p80={pct(f,.8)} p95={pct(f,.95)}")
rc={}
for r in out: rc[r['reason']]=rc.get(r['reason'],0)+1
print("exit reasons:",rc)
sl=[r for r in out if r['reason']=='sl']; tp=[r for r in out if r['reason']=='tp']
print("stop-hit share %d/%d = %.1f%% Wilson %s; target-hit %d/%d = %.1f%% Wilson %s" % (len(sl),n,100*len(sl)/n,wilson(len(sl),n),len(tp),n,100*len(tp)/n,wilson(len(tp),n)))
print("stop-hit trades: risk_pts p50/p80/p95 =",pct([r['risk_pts'] for r in sl],.5),pct([r['risk_pts'] for r in sl],.8),pct([r['risk_pts'] for r in sl],.95), " ids:",[r['id'] for r in sl])
print("stop-hit losers' MFE before death: p50/p80 =",pct([r['mfe'] for r in sl],.5),pct([r['mfe'] for r in sl],.8), "; share with MFE>=10pts:",sum(1 for r in sl if r['mfe']>=10),"/",len(sl), " >=20:",sum(1 for r in sl if r['mfe']>=20), " >=30:",sum(1 for r in sl if r['mfe']>=30))
print("winners' MAE: share with MAE>=10:",sum(1 for r in win if r['mae']>=10),"/",len(win)," >=20:",sum(1 for r in win if r['mae']>=20)," >=30:",sum(1 for r in win if r['mae']>=30)," >=40:",sum(1 for r in win if r['mae']>=40))
# floor test
wf=[r for r in win if r['floor']]; lf=[r for r in los if r['floor']]
print("floor (1.5xATR5m at entry) available for winners",len(wf),"losers",len(lf))
print("winners with MAE > floor (floor would NOT have saved them, but a stop AT the floor would have killed them):",sum(1 for r in wf if r['mae']>r['floor']),"/",len(wf), " ids:",[r['id'] for r in wf if r['mae']>r['floor']])
print("winners MAE/floor ratio p50/p80/p95:",pct([r['mae']/r['floor'] for r in wf],.5),pct([r['mae']/r['floor'] for r in wf],.8),pct([r['mae']/r['floor'] for r in wf],.95))
slf=[r for r in sl if r['floor']]
print("stop-hit trades: risk_pts vs floor — risk<floor (pre-0B tighter-than-floor stops):",sum(1 for r in slf if r['risk_pts']<r['floor']),"/",len(slf),"; risk/floor p50:",pct([r['risk_pts']/r['floor'] for r in slf],.5))
print("ATR5m at entry p50/p80:",pct([r['atr5m'] for r in out if r['atr5m']],.5),pct([r['atr5m'] for r in out if r['atr5m']],.8), " floor p50:",pct([r['floor'] for r in out if r['floor']],.5))
# pre vs post 0B (2026-09-02 07:49 CT deploy)
ob=int(datetime.datetime(2026,9,2,7,49,tzinfo=datetime.timezone(datetime.timedelta(hours=-5))).timestamp()*1000)
post=[r for r in out if c.execute("SELECT entry_time FROM trader_positions WHERE id=?",(r['id'],)).fetchone()[0]>=ob]
print("post-0B trades:",[(r['id'],r['reason'],r['risk_pts'],r['floor'],r['pnl']) for r in post])
# payoff
aw=sum(r['pnl'] for r in win)/len(win); al=sum(r['pnl'] for r in los)/len(los)
print("avg win %.2f avg loss %.2f payoff %.2f expectancy/trade %.2f total %.2f" % (aw,al,aw/abs(al),sum(r['pnl'] for r in out)/n,sum(r['pnl'] for r in out)))
print("MFE>=2R? for each trade need R; using stop-hit risk median as R proxy:", pct([r['risk_pts'] for r in sl],.5))
Rm=pct([r['risk_pts'] for r in sl],.5)
print("share of all trades with MFE >= 2x median risk (%.1f pts): %d/%d" % (2*Rm, sum(1 for r in out if r['mfe'] is not None and r['mfe']>=2*Rm), n), " >=1x:",sum(1 for r in out if r['mfe'] is not None and r['mfe']>=Rm))
