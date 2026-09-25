import sqlite3, json, math, statistics as st, datetime as dt, collections, csv
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
c=sqlite3.connect(DB, uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (ctr-h, ctr+h)
def W(lbl,k,n):
    lo,hi=wilson(k,n); print(f"  {lbl}: {k}/{n} = {100*k/n:.1f}% [{100*lo:.1f}%, {100*hi:.1f}%]")
print("### A1 plan_band + side on armed rows / long-short splits (compliant 58)")
rows=c.execute("""SELECT id, source, plan_id, cited_scenario_id, plan_session, plan_band, pnl_corrected, upper(side), datetime(entry_time/1000,'unixepoch','-5 hours'), mae, mfe, entry_price FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL ORDER BY id""").fetchall()
comp=[r for r in rows if r[2]!='UNRESOLVABLE']; broad=rows
for r in rows:
    if r[1]!='system': print("   ",r[0],r[1],r[2][:12],r[3],r[4],"band=",r[5],r[6],r[7],r[8])
def summ(name, rs):
    n=len(rs); s=sum(r[6] for r in rs); Wn=[r for r in rs if r[6]>0]; Ln=[r for r in rs if r[6]<0]; F=[r for r in rs if r[6]==0]
    mean=s/n if n else 0; sd=st.stdev([r[6] for r in rs]) if n>1 else 0; se=sd/math.sqrt(n) if n else 0
    dec=len(Wn)+len(Ln); lo,hi=wilson(len(Wn),dec) if dec else (0,0)
    print(f"  {name}: n={n} sum={s:.2f} mean={mean:.2f} sd={sd:.2f} se={se:.2f} t={(mean/se if se else 0):.2f} W/L/F={len(Wn)}/{len(Ln)}/{len(F)} win={100*len(Wn)/dec if dec else 0:.1f}% [{100*lo:.1f}%,{100*hi:.1f}%] ids={[r[0] for r in rs]}")
for lbl,pop in (("BROAD65",broad),("COMPLIANT58",comp)):
    print(" ==",lbl)
    summ("armed_fill band (reconcile+armed_fill)",[r for r in pop if r[1]=='reconcile' and r[5]=='armed_fill'])
    summ("armed lineage any (reconcile|armed_entry)",[r for r in pop if r[1]!='system'])
    summ("non-system remainder not armed_fill",[r for r in pop if r[1]!='system' and r[5]!='armed_fill'])
    summ("LONG",[r for r in pop if r[7]=='LONG']); summ("SHORT",[r for r in pop if r[7]=='SHORT'])
    summ("LONG system",[r for r in pop if r[7]=='LONG' and r[1]=='system']); summ("LONG armed",[r for r in pop if r[7]=='LONG' and r[1]!='system'])
    summ("SHORT system",[r for r in pop if r[7]=='SHORT' and r[1]=='system']); summ("SHORT armed",[r for r in pop if r[7]=='SHORT' and r[1]!='system'])
    # t-tests
    s1=[r[6] for r in pop if r[3]=='S1']; s24=[r[6] for r in pop if r[3] in ('S2','S3','S4')]
    d=st.mean(s1)-st.mean(s24); se=math.sqrt(st.variance(s1)/len(s1)+st.variance(s24)/len(s24)); print(f"  S1 vs S2-S4: diff={d:.2f} se={se:.2f} t={d/se:.2f}")
    a=[r[6] for r in pop if r[4]=='ASIA']; ln=[r[6] for r in pop if r[4] in ('LONDON','NY')]
    d=st.mean(a)-st.mean(ln); se=math.sqrt(st.variance(a)/len(a)+st.variance(ln)/len(ln)); print(f"  ASIA vs LONDON+NY: diff={d:.2f} se={se:.2f} t={d/se:.2f}; ASIA one-sample t={st.mean(a)/(st.stdev(a)/math.sqrt(len(a))):.2f}; LONDON+NY mean CI=[{st.mean(ln)-1.96*st.stdev(ln)/math.sqrt(len(ln)):.2f},{st.mean(ln)+1.96*st.stdev(ln)/math.sqrt(len(ln)):.2f}]")
    sysp=[r[6] for r in pop if r[1]=='system']; print(f"  system one-sample: mean={st.mean(sysp):.2f} se={st.stdev(sysp)/math.sqrt(len(sysp)):.2f} t={st.mean(sysp)/(st.stdev(sysp)/math.sqrt(len(sysp))):.2f} CI=[{st.mean(sysp)-1.96*st.stdev(sysp)/math.sqrt(len(sysp)):.2f},{st.mean(sysp)+1.96*st.stdev(sysp)/math.sqrt(len(sysp)):.2f}]")

print("### A2 rejects classified by the STATED definition (void close-back / displacement 0.00 / BD_MIN_CLOSES) vs the q07 classifier")
rej=c.execute("SELECT id, trade_date, session, attempt, reject_reason FROM planner_rejected_prompts ORDER BY id").fetchall()
print("  total rejects", len(rej))
stated=[]; q07=[]; conf_na=[]; armlegs=[]; other=[]
for i,td,se,att,rr in rej:
    s=rr
    is_q07 = ('breakdown_continue' in s or 'breakup_continue' in s)
    is_void = ('came back across' in s) or ('close came back' in s)
    is_disp = ('0.00 pts' in s) or ('displacement 0.00' in s)
    is_bdmin = ('NO confirming close' in s) or ('BD_MIN_CLOSES' in s)
    is_notallowed = ('not allowed for' in s)
    is_armlegs = ('arm legs' in s)
    if is_q07: q07.append(i)
    if is_void or is_disp or is_bdmin: stated.append((i, 'void' if is_void else 'disp' if is_disp else 'bdmin'))
    if is_notallowed and is_q07: conf_na.append(i)
    if is_armlegs and is_q07: armlegs.append(i)
    if is_q07 and not (is_void or is_disp or is_bdmin): other.append((i, s[:110]))
print("  q07 classifier ('breakdown_continue' in s):", len(q07), q07)
print("  stated definition:", len(stated), [x[0] for x in stated])
print("    by kind:", collections.Counter(x[1] for x in stated))
print("  q07-but-not-stated:", len(other)); [print("     ",o) for o in other]
W("stated/64", len(stated), 64); W("q07/64", len(q07), 64)
fam=collections.Counter()
for i,td,se,att,s in rej:
    if ('came back across' in s) or ('0.00 pts' in s) or ('NO confirming close' in s): f='continuation law (stated def)'
    elif 'fade_requires_touch' in s: f='fade_requires_touch'
    elif 'not allowed for' in s or 'invalid (touch' in s: f='confirm-rule vocabulary/not-allowed'
    elif 'entry_mode=pullback' in s: f='arm requires entry_mode=pullback'
    elif 'arm on' in s or 'arm legs' in s: f='arm legs contract'
    elif 'gap-up' in s: f='gap-up trigger law'
    elif '503' in s or 'stream interrupted' in s or 'EOF' in s: f='TRANSPORT'
    elif 'too many levels' in s: f='level cap'
    elif 'unreachable' in s: f='retest distance'
    else: f='other: '+s[:50]
    fam[f]+=1
print("  families with stated def:"); [print("     ",k,v) for k,v in fam.most_common()]
# the 5 not-allowed rows and the arm legs row text
for i in (70,78,80,81,112,118):
    print("   id",i, c.execute("SELECT substr(reject_reason,1,160) FROM planner_rejected_prompts WHERE id=?",(i,)).fetchone()[0])

print("### A3 armed_orders: flap deaths ids 62-102; reaper; marketable; created_at CT")
def ct(s):
    if s is None: return None
    s=str(s)
    if s.endswith('+00:00') or s.endswith('Z'): base=dt.datetime.strptime(s[:19].replace('T',' '),'%Y-%m-%d %H:%M:%S'); return (base-dt.timedelta(hours=5)).strftime('%Y-%m-%d %H:%M:%S')
    if s.endswith('-05:00'): return s[:19].replace('T',' ')
    return s[:19].replace('T',' ')+'?'
arms=c.execute("SELECT id, state, state_reason, entry_px, scenario, condition, kind, side, placement_seq, created_at, updated_at, session, plan_id, version, signal_id FROM armed_orders ORDER BY id").fetchall()
flap=[a for a in arms if 62<=a[0]<=102]
print("  ids 62-102 count", len(flap), "with entry_px 29591.02:", sum(1 for a in flap if abs(a[3]-29591.02)<0.01))
print("  deaths in 62-102 (all):", collections.Counter(a[2] for a in flap))
print("  deaths in 62-102 @29591.02:", collections.Counter(a[2] for a in flap if abs(a[3]-29591.02)<0.01))
print("  other rows in range:", [(a[0],a[4],a[3],a[2]) for a in flap if abs(a[3]-29591.02)>=0.01])
print("  62-102 created CT min/max:", min(ct(a[9]) for a in flap), max(ct(a[9]) for a in flap), "; id 101:", ct([a for a in flap if a[0]==101][0][9]), "id 102:", ct([a for a in flap if a[0]==102][0][9]))
print("  placement_seq in 62-102 @29591.02:", sorted(a[8] for a in flap if abs(a[3]-29591.02)<0.01))
print("  id 38:", [(a[0],a[1],a[2],a[3],a[4],a[5],a[8],ct(a[9]),ct(a[10])) for a in arms if a[0]==38])
rr=[(a[0],ct(a[9]),ct(a[10])) for a in arms if a[2] and 'gate changed: rr' in a[2]]; print("  all 'gate changed: rr' rows:", rr)
stale=[(a[0],ct(a[9])[:10]) for a in arms if a[2] and 'stale window' in a[2]]; print("  stale-window rows:", len(stale), stale)
mkt=[(a[0],ct(a[9])) for a in arms if a[2] and 'marketable' in a[2]]; print("  marketable rows:", len(mkt), mkt)
boot=[(a[0],ct(a[9])[:10]) for a in arms if a[2] and 'boot_sweep' in a[2]]; print("  boot_sweep rows:", len(boot), boot)
test=[a[0] for a in arms if a[11] and a[11].startswith('TEST')]; print("  test-session arms:", test)
fills=[(a[0],a[2]) for a in arms if a[1]=='filled']; print("  filled:", fills)
print("  arm 6:", [(a[0],a[1],a[2],a[3],a[7]) for a in arms if a[0]==6])
print("  stop_px arm 6:", c.execute("SELECT id, entry_px, stop_px, target_px, fill_price, state_reason FROM armed_orders WHERE id=6").fetchone())
print("  kind x condition x side:", collections.Counter((a[6],a[5],a[7]) for a in arms if a[6]=='stop_entry'))

print("### A4 decision_records open intents 09-03 CT and since strict 10:43 CT")
ints=c.execute("""SELECT id, datetime(timestamp,'-5 hours'), substr(execution_log,1,90), substr(risk_check_error,1,60) FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-03' AND (decision_json LIKE '%open_long%' OR decision_json LIKE '%open_short%') ORDER BY id""").fetchall()
for r in ints: print("   ",r)
print("  count 09-03:", len(ints), "since 10:43:", sum(1 for r in ints if r[1]>='2026-09-03 10:43:00'))
print("  c8c90dcc:", end=' ')
import subprocess; print(subprocess.run(['git','-C','/home/hoang/nofx-vet-03','log','-1','--format=%h %ci %s','c8c90dcc'],capture_output=True,text=True).stdout.strip())

print("### A5 plans bias_label by date")
print("  by UTC date:", c.execute("SELECT date(created_at), COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1").fetchall())
print("  by CT date:", c.execute("SELECT date(created_at,'-5 hours'), COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1").fetchall())
print("  first:", c.execute("SELECT plan_id, version, trade_date, session, created_at FROM plans WHERE doc LIKE '%bias_label%' ORDER BY created_at LIMIT 1").fetchone())
print("  total plans:", c.execute("SELECT COUNT(*) FROM plans").fetchone(), "distinct versions with doc:", c.execute("SELECT COUNT(*) FROM plans WHERE doc IS NOT NULL").fetchone())
print("  by trade_date:", c.execute("SELECT trade_date, COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1").fetchall())

print("### A6 inter-version gaps since 09-01 (ASIA/LONDON/NY)")
pl=c.execute("SELECT trade_date, session, version, created_at FROM plans WHERE session IN ('ASIA','LONDON','NY') AND trade_date>='2026-09-01' ORDER BY trade_date, session, created_at").fetchall()
gaps=[]; prev=None
for r in pl:
    t=dt.datetime.strptime(r[3][:19].replace('T',' '),'%Y-%m-%d %H:%M:%S')
    if prev and prev[0]==r[0] and prev[1]==r[1]: gaps.append((t-prev[2]).total_seconds()/60)
    prev=(r[0],r[1],t)
gaps.sort(); n=len(gaps)
print(f"  n={n} versions={len(pl)} median(statistics)={st.median(gaps):.2f} sorted[32..35]={[round(x,2) for x in gaps[32:36]]} mean={st.mean(gaps):.1f} min={min(gaps):.1f} max={max(gaps):.1f}")

print("### A7 MFE>=2R on compliant 58: own authored stop; 568 via armed_orders state_reason fill@")
decs=c.execute("SELECT id, strftime('%s',timestamp)*1000, decision_json FROM decision_records WHERE decision_json LIKE '%open_%' AND date(timestamp,'-5 hours')>='2026-08-19'").fetchall()
def stops_from(dj):
    out=[]
    try: d=json.loads(dj)
    except Exception: return out
    items=d if isinstance(d,list) else d.get('decisions',[d]) if isinstance(d,dict) else []
    for x in items:
        if isinstance(x,dict) and str(x.get('action','')).startswith('open_'):
            out.append((x.get('action'), x.get('stop_loss') or x.get('stopLoss') or x.get('stop'), x.get('entry') or x.get('entry_price')))
    return out
armsf=c.execute("SELECT id, side, entry_px, stop_px, fill_price, state_reason FROM armed_orders WHERE state='filled'").fetchall()
pos=c.execute("SELECT id, entry_time, upper(side), entry_price, mfe, mae, pnl_corrected, source, plan_band, plan_id FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND plan_id<>'UNRESOLVABLE' AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL ORDER BY entry_time").fetchall()
res=[]; unres=[]
for pid,et,side,ep,mfe,mae,pnl,src,band,plid in pos:
    if mfe is None: unres.append((pid,'no mfe')); continue
    R=None; how=''
    if src=='system':
        best=None
        for did,dtm,dj in decs:
            if abs(dtm-et)<=180000:
                for a,sl,e in stops_from(dj):
                    if sl and a.endswith(side.lower()):
                        cand=abs(float(ep)-float(sl))
                        if best is None or abs(dtm-et)<best[0]: best=(abs(dtm-et),cand,did)
        if best: R=best[1]; how=f'decision {best[2]}'
    else:
        for aid,aside,aep,asp,afp,sr in armsf:
            fillpx=None
            if sr and 'fill@' in sr:
                try: fillpx=float(sr.split('fill@')[1].split()[0].rstrip(','))
                except: pass
            if aside.lower()==side.lower() and (abs(float(aep)-float(ep))<=1.0 or (fillpx and abs(fillpx-float(ep))<=1.0) or (afp and abs(float(afp)-float(ep))<=1.0)):
                R=abs(float(ep)-float(asp)); how=f'arm {aid}'; break
    if R and R>0: res.append((pid,side,ep,R,mfe,mfe/R,mae,pnl,how))
    else: unres.append((pid,'no stop'))
n=len(res); k2=sum(1 for r in res if r[5]>=2.0); k1=sum(1 for r in res if r[5]>=1.0)
print(f"  resolved n={n}; excluded {unres}")
W("MFE>=2R", k2, n); W("MFE>=1R", k1, n)
mr=sorted(r[5] for r in res)
def pct(a,p):
    k=(len(a)-1)*p; f=math.floor(k); cc=math.ceil(k); return a[f] if f==cc else a[f]+(a[cc]-a[f])*(k-f)
print(f"  mfe_R median={st.median(mr):.2f} p75(interp)={pct(mr,.75):.2f} p75(rank 3n//4)={mr[3*n//4]:.2f}")
print("  568 row:", [r for r in res if r[0]==568])
wins=[r for r in res if r[7]>0]; loss=[r for r in res if r[7]<0]
print(f"  winners n={len(wins)} MAE/R median={st.median([r[6]/r[3] for r in wins]):.3f} max={max(r[6]/r[3] for r in wins):.3f}; losers n={len(loss)} MAE/R median={st.median([r[6]/r[3] for r in loss]):.3f} min={min(r[6]/r[3] for r in loss):.3f}")
print("  losers with MAE/R<0.9:", [(r[0],round(r[6]/r[3],3)) for r in loss if r[6]/r[3]<0.9])
print("  R pts median:", st.median([r[3] for r in res]))
with open('r11_mfe_r_rows_compliant.csv','w',newline='') as f:
    w=csv.writer(f); w.writerow(['position_id','side','entry','R_pts','mfe_pts','mfe_R','mae_pts','pnl_corrected','stop_source']); w.writerows(res)
# fixed-R target EV
print("  fixed-R target counterfactual EV (per trade, R units): +T if mfe_R>=T; else -1 if mae_R>=1; else realized R")
for T in (0.5,0.67,1.0,1.5,1.79,2.0,3.0):
    ev=[]; reach=0
    for r in res:
        mfeR=r[5]; maeR=r[6]/r[3]; realR=r[7]/(r[3]*2.0)
        if mfeR>=T: ev.append(T); reach+=1
        elif maeR>=1: ev.append(-1)
        else: ev.append(realR)
    print(f"    T={T}: EV={st.mean(ev):+.3f}R reach={reach}/{n}")
print(f"  realized mean R={st.mean([r[7]/(r[3]*2.0) for r in res]):+.3f}")

print("### A8 saved MNQ strategy config knobs")
srow=c.execute("SELECT s.id, s.name FROM strategies s JOIN traders t ON t.strategy_id=s.id WHERE t.name='hoang' OR t.name LIKE '%hoang%'").fetchall(); print("  strategies bound:", srow)
for sid,name in srow:
    cfg=c.execute("SELECT config FROM strategies WHERE id=?",(sid,)).fetchone()[0]
    d=json.loads(cfg); dp=d.get('day_plan',{})
    print("  sessions_enabled:", dp.get('sessions_enabled'), "| replan_cap:", dp.get('replan_cap'), "| approval_required:", dp.get('approval_required'), "| plan_mode:", dp.get('plan_mode'))
    for s in dp.get('sessions',[]) or []: print("    session override:", {k:s.get(k) for k in ('session','enable','replan_cap')})
print("  traders scan_interval:", c.execute("SELECT name, scan_interval_minutes FROM traders").fetchall())

print("### A9 cycle_type x trigger since 08-27")
print(c.execute("SELECT cycle_type, cycle_trigger, COUNT(*) FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-08-27' GROUP BY 1,2").fetchall())
print("### A10 position 590 and cycle 26500")
print(c.execute("SELECT id, entry_price, exit_price, upper(side), pnl_corrected, mae, mfe, datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE id=590").fetchone())
print(c.execute("SELECT id, cycle_number, datetime(timestamp,'-5 hours'), substr(execution_log,1,120), substr(decision_json,1,300) FROM decision_records WHERE cycle_number=26500").fetchall())
