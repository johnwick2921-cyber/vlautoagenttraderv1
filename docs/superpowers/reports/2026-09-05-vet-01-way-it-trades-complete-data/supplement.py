"""Offline sensitivity/provenance only; no production access."""
import runpy,io,contextlib,json,math,statistics as st,datetime as dt,csv
from pathlib import Path
with contextlib.redirect_stdout(io.StringIO()):a=runpy.run_path('audit.py')
T=a['T'];raw=a['raw'];P=a['P'];bars=a['bars'];pct=a['pct'];wil=a['wil'];out={};write=a['writecsv']
# Read finite held bars only for measurement coverage; never invent ordering within a minute.
rows=[]
for t in T:
    m=t['entry_time']//300000*300000-300000;blocks=[];ids=[]
    # Up to 100 consecutive CLOSED complete 5m blocks; restart warmup at any gap.
    for i in range(100):
        bb=[bars.get(m-i*300000+j*60000) for j in range(5)]
        if not all(bb):break
        blocks.append((min(x['l'] for x in bb),max(x['h'] for x in bb),bb[-1]['c']));ids.extend(x['row_id'] for x in bb)
    blocks.reverse();atr=None
    if len(blocks)>=15:
        tr=[max(blocks[i][1]-blocks[i][0],abs(blocks[i][1]-blocks[i-1][2]),abs(blocks[i][0]-blocks[i-1][2])) for i in range(1,len(blocks))]
        atr=st.mean(tr[:14])
        for v in tr[14:]:atr=(13*atr+v)/14
    start=(t['entry_time']+59999)//60000*60000;end=t['exit_time']//60000*60000
    wanted=list(range(start,end,60000));missing=[m for m in wanted if m not in bars]
    rows.append(dict(id=t['id'],atr5m_closed_proxy=atr,closed_5m_blocks=len(blocks),atr_bar_ids=','.join(map(str,ids)),mfe_position_proxy=t['mfe'],proxy_reach_2x_floor=t['mfe']>=3*atr if atr and t['mfe'] is not None else None,held_complete_minutes=len(wanted),missing_inside_minutes=len(missing),ambiguous_boundary_minutes='entry and exit minute not sequenced',initial_R=t['realized_R']))
write('floor_path_sensitivity.csv',rows)
known=[r for r in rows if r['proxy_reach_2x_floor'] is not None];hit=[r['id'] for r in known if r['proxy_reach_2x_floor']]
out['floor_proxy']={'n':len(known),'hits':hit,'wilson':wil(len(hit),len(known)),'missing_ids':[r['id'] for r in rows if r['proxy_reach_2x_floor'] is None],'warning':'not actual entry ATR/floor and not ordered target-before-stop'}
ps=a['PS'];pr=[]
for r in ps:
    p=P[(r['plan_id'],r['version'])];sign=1 if r['target']>r['entry'] else -1
    distances=[(lv['price']-r['entry'])*sign for lv in p['doc']['levels'] or [] if (lv['price']-r['entry'])*sign>.25]
    nearest=min(distances) if distances else None
    pr.append(dict(plan_row_id=r['plan_row_id'],version=r['version'],scenario=r['scenario'],entry=r['entry'],target=r['target'],nearest_plan_level_distance=nearest,target_distance=abs(r['target']-r['entry']),beyond_nearest=bool(nearest is not None and abs(r['target']-r['entry'])>nearest+.25),nearest_R=nearest/abs(r['stop']-r['entry']) if nearest else None))
write('target_obstacles.csv',pr)
for label,pred in [('all',lambda p:True),('Sep01_Sep02_CT',lambda p:'2026-09-01'<=a['ct'](a['time_ms'](p['created_at'])).date().isoformat()<='2026-09-02')]:
    sub=[p for p in ps if pred(p)];out['planned_'+label]={'n':len(sub),'median_rr':pct([p['rr'] for p in sub],.5),'rr_2_to_2_3':sum(2<=p['rr']<2.3 for p in sub),'wilson_cluster':wil(sum(2<=p['rr']<2.3 for p in sub),len(sub)),'identities':[[p['plan_row_id'],p['version'],p['scenario']] for p in sub]}
out['target_provenance']={'level_matches':sum(x['target_at_plan_level'] for x in ps),'level_wilson':wil(sum(x['target_at_plan_level'] for x in ps),len(ps)),'chain_matches':sum(x['target_in_chain'] for x in ps),'chain_wilson':wil(sum(x['target_in_chain'] for x in ps),len(ps)),'beyond_nearest':sum(x['beyond_nearest'] for x in pr),'nearest_available_n':sum(x['nearest_plan_level_distance'] is not None for x in pr),'beyond_wilson':wil(sum(x['beyond_nearest'] for x in pr),sum(x['nearest_plan_level_distance'] is not None for x in pr))}
# Formation specific: RTH-L observations attached to Sep04 LONDON/NY use Sep03 RTH extrema; count before Sep03 RTH even began.
rth=[x for x in raw['touches'] if x['level_kind']=='RTH-L'];out['rth_formation']={'raw_n':len(rth),'prices':sorted(set(x['level_price'] for x in rth)),'keys':len(set((x['level_price'],x['opened_at_ms']) for x in rth)),'plan_keys':sorted(set((P.get((x['plan_id'],x['plan_version']),{}).get('row_id',-1),x['plan_version']) for x in rth)),'before_Sep03_0830_raw_ids':[x['id'] for x in rth if x['opened_at_ms']<int(dt.datetime(2026,9,3,8,30,tzinfo=a['CT']).timestamp()*1000)]}
for group,ts in [('winners',[t for t in T if t['pnl_corrected']>0]),('losers',[t for t in T if t['pnl_corrected']<0])]:
    out[group]={'n':len(ts),'ids':[t['id'] for t in ts],'hold_p50':pct([t['hold_min'] for t in ts],.5),'mfe_p50':pct([t['mfe'] for t in ts if t['mfe'] is not None],.5),'mae_p50':pct([t['mae'] for t in ts if t['mae'] is not None],.5)}
# rejected inference: midpoint or final arm stops may differ from initially accepted stops
brows=[]
for t in T:
    if t['realized_R'] is None:continue
    arms=[x for x in raw['arms'] if x['signal_id']==t['broker_signal']]
    stopfills=[b for b in raw['broker'] if b['state']=='Filled' and abs(b['ms']-t['exit_time'])<180000 and abs((b['fill'] or 0)-t['exit_price'])<1e-6 and b['name']==t['broker_signal']+'-sl']
    tpfill=[b for b in raw['broker'] if b['state']=='Filled' and abs(b['ms']-t['exit_time'])<180000 and abs((b['fill'] or 0)-t['exit_price'])<1e-6 and b['name']==t['broker_signal']+'-tp']
    brows.append(dict(id=t['id'],entry=t['entry_price'],initial_stop=t['broker_stop_initial'],initial_target=t['broker_target_initial'],R=t['realized_R'],ledger_arm_ids=','.join(str(x['id']) for x in arms),ledger_final_stops=','.join(str(x['stop_px']) for x in arms),exit_cause_by_order='broker_stop' if stopfills else 'broker_target' if tpfill else 'unresolved',stop_fill_price=stopfills[0]['fill'] if stopfills else None,stop_price_at_fill=stopfills[0]['stop'] if stopfills else None,stop_slippage=abs(stopfills[0]['fill']-stopfills[0]['stop']) if stopfills else None,exit_ref=f"{(stopfills or tpfill)[0]['path']}:{(stopfills or tpfill)[0]['line']}" if stopfills or tpfill else ''))
write('broker_R.csv',brows)
out['broker_R']={'n':len(brows),'mean':st.mean(r['R'] for r in brows),'median':pct([r['R'] for r in brows],.5),'p25':pct([r['R'] for r in brows],.25),'p75':pct([r['R'] for r in brows],.75),'ids':[r['id'] for r in brows],'exit_cause_counts':dict(a['Counter'](r['exit_cause_by_order'] for r in brows))}
out['path_coverage']={'missing_inside_ids':[r['id'] for r in rows if r['missing_inside_minutes']>0],'boundary_note':'All 58 require entry/exit intra-minute ordering; proxies do not show the counterfactual post-exit path.'}
# Two historical reconcile winner zeros are unverified, not measured absence of adverse movement.
uncertain={569,584}
assert all(t['source']=='reconcile' and t['mae']==0 and t['pnl_corrected']>0 for t in T if t['id'] in uncertain)
z=[]
for group,ts in [('winners',[t for t in T if t['pnl_corrected']>0]),('reject',[t for t in T if t['condition']=='reject'])]:
    for mode,ss in [('raw_position_proxy',ts),('exclude_uncertain_zero_MAE',[t for t in ts if t['id'] not in uncertain])]:
        xs=[t['mae'] for t in ss if t['mae'] is not None]
        z.append(dict(group=group,mode=mode,n=len(xs),mae_p50=pct(xs,.5),mae_p80=pct(xs,.8),mae_p95=pct(xs,.95),ids=','.join(str(t['id']) for t in ss),interpretation='UNVERIFIED measurement; sensitivity is not a corrected excursion series'))
write('zero_mae_sensitivity.csv',z)
out['zero_mae_sensitivity']={'uncertain_ids':[569,584],'rows':z,'rule':'Preserve primary 58 and raw proxies; exclude only these uncertain zero MAEs in sensitivity. No inference of no-adverse-excursion or safe stop tightening.'}
Path('supplement.json').write_text(json.dumps(out,indent=2))
with open('supplement.txt','w') as f:
    for key,value in out.items():f.write(key+' '+json.dumps(value)+'\n')
print(json.dumps({k:v for k,v in out.items() if not k.startswith('planned_')},indent=2))
