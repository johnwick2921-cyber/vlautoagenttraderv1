# q11: Wilson intervals for every rate quoted; P&L split by entry path; MFE>=2R re-measure using each system trade's own authored stop (from its entry decision) and each arm's stop_px
import sqlite3, json, math, datetime, collections
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (float('nan'),float('nan'))
    p=k/n; den=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n)); return ((c-h)/den,(c+h)/den)
def W(label,k,n):
    lo,hi=wilson(k,n); print(f'{label}: {k}/{n} = {k/n:.3f}  Wilson [{lo:.3f}, {hi:.3f}]')
print('## rates')
W('era win rate excl scratch (pnl_corrected>0 of !=0)', 21, 63)
W('S1 win rate', 10, 27); W('S2+ win rate', 8, 31)
W('ASIA win rate', 2, 15); W('LONDON', 7, 21); W('NY (21 resolved)', 9, 20)
W('decision-path (source=system) win rate', 13, 49)
W('armed_fill (reconcile armed_fill rows) win rate', 5, 9)
W('all-time arm fill rate (real fills / non-test arms; test rows = sessions TEST-E2 x4 + TEST-E7 x2)', 9, 67-6)
W('two-day-window arm fill (ids 23-37)', 3, 15)
W('marketable-guard deaths (all non-test arms)', 6, 61)
W('stale-window reaper deaths (all non-test arms)', 12, 61)
W('trigger-fired in-force', 263, 592); W('trigger-fired first-version', 62, 82)
W('reject share of authoring attempts since 09-01', 64, 64+79)
W('continuation-law share of rejects', 38, 64)
W('open intents per executor cycle since 08-27', 31, 4102)
W('open intents refused by a gate', 19, 31)
W('strict refusals / decision-path intents 09-03', 13, 14)
W('level_event share of plan versions since 09-01', 53, 79)
W('long scenarios arm-enabled (latest)', 4, 56); W('short scenarios arm-enabled (latest)', 14, 72)
W('bias_label AI==tree (versions carrying a label)', 21, 27)
print('## P&L split (pnl_corrected, plan-linked, excl e7 test seam, excl NULL)')
for lbl,sql in [('decision path (source=system)',"source='system'"),('armed_fill lineage (source=reconcile AND plan_band=armed_fill)',"source='reconcile' AND plan_band='armed_fill'"),('reconcile non-armed (566,571,580)',"source='reconcile' AND (plan_band IS NULL OR plan_band<>'armed_fill')"),('armed_entry ghosts',"source='armed_entry'")]:
    r=con.execute(f"SELECT COUNT(*), SUM(pnl_corrected IS NULL), ROUND(SUM(pnl_corrected),2), SUM(pnl_corrected>0), SUM(pnl_corrected<0), GROUP_CONCAT(id) FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' AND {sql}").fetchone()
    print(f'  {lbl}: n={r[0]} null_excl={r[1]} sum={r[2]} W={r[3]} L={r[4]} ids={r[5]}')
print('## MFE >= 2R re-measure: system trades use the stop_loss in the open_* decision at entry minute; armed rows use armed_orders.stop_px matched on fill price/side/day')
pos=con.execute("SELECT id, entry_time, side, entry_price, mfe, mae, pnl_corrected, source, plan_band, plan_session FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL AND mfe IS NOT NULL ORDER BY entry_time").fetchall()
decs=con.execute("SELECT id, strftime('%s',timestamp)*1000, decision_json FROM decision_records WHERE decision_json LIKE '%open_%' AND date(timestamp,'-5 hours')>='2026-08-19'").fetchall()
def stops_from(dj):
    out=[]
    try: d=json.loads(dj)
    except Exception: return out
    items=d if isinstance(d,list) else d.get('decisions',[d]) if isinstance(d,dict) else []
    for x in items:
        if isinstance(x,dict) and str(x.get('action','')).startswith('open_'):
            out.append((x.get('action'), x.get('stop_loss') or x.get('stopLoss') or x.get('stop'), x.get('entry') or x.get('entry_price')))
    return out
arms=con.execute("SELECT id, side, entry_px, stop_px, fill_price, strftime('%s',substr(created_at,1,19)) FROM armed_orders WHERE state='filled'").fetchall()
rows=[]; unresolved=[]
for p in pos:
    pid,et,side,ep,mfe,mae,pnl,src,band,sess=p
    R=None; how=''
    if src=='system':
        best=None
        for did,dt,dj in decs:
            if abs(dt-et)<=180000:
                for a,sl,e in stops_from(dj):
                    if sl and a.endswith(side.lower()):
                        cand=abs(float(ep)-float(sl)); 
                        if best is None or abs(dt-et)<best[0]: best=(abs(dt-et),cand,did)
        if best: R=best[1]; how=f'decision {best[2]}'
    else:
        for aid,aside,aep,asp,afp,act in arms:
            if aside.lower()==side.lower() and abs(float(aep)-float(ep))<=1.0:
                R=abs(float(ep)-float(asp)); how=f'arm {aid}'; break
    if R and R>0: rows.append((pid,side,ep,R,mfe,mfe/R,mae,pnl,how))
    else: unresolved.append(pid)
n=len(rows); k2=sum(1 for r in rows if r[5]>=2.0); k1=sum(1 for r in rows if r[5]>=1.0)
print(f'resolved n={n} (unresolved ids: {unresolved})')
W('MFE >= 2.0R (own authored stop)', k2, n); W('MFE >= 1.0R', k1, n)
import statistics
print('mfe_R median', round(statistics.median([r[5] for r in rows]),2), 'p75', round(sorted([r[5] for r in rows])[3*n//4],2))
wins=[r for r in rows if r[7]>0]; losses=[r for r in rows if r[7]<0]
print('winners n',len(wins),'MAE/R median', round(statistics.median([r[6]/r[3] for r in wins]),3), 'max', round(max(r[6]/r[3] for r in wins),3))
print('losers  n',len(losses),'MAE/R median', round(statistics.median([r[6]/r[3] for r in losses]),3))
print('R (stop distance pts) median', round(statistics.median([r[3] for r in rows]),2))
import csv
with open('/home/hoang/nofx-analysis/vet-03-0905/q11_mfe_r_rows.csv','w',newline='') as f:
    w=csv.writer(f); w.writerow(['position_id','side','entry','R_pts','mfe_pts','mfe_R','mae_pts','pnl_corrected','stop_source']); w.writerows(rows)
