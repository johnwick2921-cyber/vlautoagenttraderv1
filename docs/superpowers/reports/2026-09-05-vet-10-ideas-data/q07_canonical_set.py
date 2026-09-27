# q07: canonical usable set aligned with /api/expectancy (excl. e7 test seam, pnl NULL, plan_link UNRESOLVABLE),
# plus the 7 off-plan rows shown separately; R-multiples from the opening decision's stop (decision path) or arm stop_px (armed path)
import sqlite3, json, sys, re
sys.path.insert(0,'/home/hoang/nofx-analysis/vet-10-0905'); from wilson import wilson, mean_ci
db=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); db.row_factory=sqlite3.Row
E=int(__import__('datetime').datetime(2026,8,15,5,0).timestamp()*1000)  # 2026-08-15 00:00 CT = 05:00 UTC
rows=db.execute("select * from trader_positions where entry_time>=? and source<>'e7_farside_test' and pnl_corrected is not null order by entry_time",(E,)).fetchall()
print("era rows with pnl, non-test:",len(rows))
unres=[r for r in rows if 'UNRESOLVABLE' in (r['plan_link_note'] or '') or 'UNRESOLVABLE' in (r['pnl_correction_note'] or '')]
print("UNRESOLVABLE plan_link rows:",[r['id'] for r in unres])
for r in unres: print("   ",r['id'],r['plan_link_note'])
canon=[r for r in rows if r not in unres]
print("canonical n=",len(canon),"sum=",round(sum(r['pnl_corrected'] for r in canon),2))
# arms by plan/scenario for stop lookup
arms={}
for a in db.execute("select plan_id, scenario, side, entry_px, stop_px, target_px, fill_price, state from armed_orders where state='filled'"):
    arms[(a['plan_id'],a['scenario'])]=a
out=[]
for r in canon:
    stop=None; src=None; tp=None
    if r['plan_band']=='armed_fill' or r['source']=='armed_entry':
        a=arms.get((r['plan_id'],r['cited_scenario_id']))
        if a: stop=a['stop_px']; tp=a['target_px']; src='arm'
    if stop is None:
        # nearest opening decision within 6 min before entry
        t0=r['entry_time']
        d=db.execute("select decision_json, timestamp from decision_records where trader_id=? and (strftime('%s',timestamp)*1000) between ? and ? and (decision_json like '%open_long%' or decision_json like '%open_short%') order by timestamp desc limit 1",(r['trader_id'],t0-6*60000,t0+60000)).fetchone()
        if d:
            try:
                j=json.loads(d['decision_json'])
                if isinstance(j,list): j=[x for x in j if str(x.get('action','')).startswith('open')][0]
                elif isinstance(j,dict) and 'decisions' in j: j=[x for x in j['decisions'] if str(x.get('action','')).startswith('open')][0]
                stop=float(j.get('stop_loss') or 0) or None; tp=float(j.get('take_profit') or 0) or None; src='decision'
            except Exception as e: src='parse_fail:'+str(e)[:30]
    risk=None; rr=None
    if stop:
        risk=abs(r['entry_price']-stop)
        if tp: rr=abs(tp-r['entry_price'])/risk if risk>0 else None
    Rm = (r['pnl_corrected']/2.0)/risk if risk else None
    out.append((r['id'],r['side'],r['entry_price'],r['exit_price'],round(r['pnl_corrected'],1),r['mae'],r['mfe'],stop,tp,round(risk,2) if risk else None,round(rr,2) if rr else None,round(Rm,2) if Rm is not None else None,src,r['plan_session'],r['cited_scenario_id']))
import csv
w=csv.writer(open('/home/hoang/nofx-analysis/vet-10-0905/q07_canonical_trades.csv','w'))
w.writerow(['id','side','entry','exit','pnl_c','mae_pts','mfe_pts','stop','tp','risk_pts','planned_rr','realized_R','stop_src','session','scenario'])
for o in out: w.writerow(o)
print("stop source counts:",{s:sum(1 for o in out if o[12]==s) for s in set(o[12] for o in out)})
have=[o for o in out if o[11] is not None]
print("rows with R:",len(have))
R=[o[11] for o in have]; print("mean R=%.3f  CI %s"%(sum(R)/len(R), tuple(round(x,3) for x in mean_ci(R))))
wins=[o for o in have if o[4]>0]; loss=[o for o in have if o[4]<0]
print("wins",len(wins),"avg win R %.2f"%(sum(o[11] for o in wins)/len(wins)),"avg win $ %.1f"%(sum(o[4] for o in wins)/len(wins)))
print("losses",len(loss),"avg loss R %.2f"%(sum(o[11] for o in loss)/len(loss)),"avg loss $ %.1f"%(sum(o[4] for o in loss)/len(loss)))
print("avg planned RR %.2f"%(sum(o[10] for o in have if o[10])/len([o for o in have if o[10]])))
print("realized reward:risk (avg win$/avg loss$) = %.2f"%((sum(o[4] for o in wins)/len(wins))/abs(sum(o[4] for o in loss)/len(loss))))
k=len(wins); n=len(wins)+len(loss); print("win rate %d/%d=%.3f wilson %s"%(k,n,k/n,tuple(round(x,3) for x in wilson(k,n))))
# stops hit fraction: loss with |pnl|/2 >= 0.9*risk
sh=[o for o in loss if o[9] and abs(o[4])/2>=0.9*o[9]]; print("losses that were full stops (>=0.9R):",len(sh),"of",len(loss))
# MFE give-back among losers
gb=[o for o in loss if o[6] is not None and o[9] and o[6]>=1.0*o[9]]; print("losers whose MFE >= 1R:",len(gb),[o[0] for o in gb])
gb2=[o for o in loss if o[6] is not None and o[9] and o[6]>=0.5*o[9]]; print("losers whose MFE >= 0.5R:",len(gb2))
# all with mfe: ratio mfe/risk
mr=[o[6]/o[9] for o in have if o[6] is not None and o[9]]; mr.sort(); print("MFE/risk median %.2f p75 %.2f n=%d"%(mr[len(mr)//2], mr[int(len(mr)*0.75)], len(mr)))
ma=[o[5]/o[9] for o in have if o[5] is not None and o[9]]; ma.sort(); print("MAE/risk median %.2f p75 %.2f"%(ma[len(ma)//2], ma[int(len(ma)*0.75)]))
# winners: MAE/risk (how close winners came to stop)
wm=[o[5]/o[9] for o in wins if o[5] is not None and o[9]]; wm.sort(); print("winners MAE/risk median %.2f p75 %.2f max %.2f"%(wm[len(wm)//2], wm[int(len(wm)*0.75)], wm[-1]))
# per-trade table
for o in out: print(o)
