#!/usr/bin/env python3
"""Reproduce descriptive tables from extract.py outputs. No trading code executed."""
import bisect
import collections as co
import csv
import datetime as dt
import itertools
import json
import math
import pathlib
import re
import statistics as st
import zoneinfo

P = pathlib.Path(__file__).parent
Z = zoneinfo.ZoneInfo('America/Chicago')
load = lambda n: json.loads((P/(n+'.json')).read_text())
def stamp(s):
    # SQLite strings retain offsets, including mixed UTC and CT rows.
    return int(dt.datetime.fromisoformat(s.replace('Z','+00:00')).timestamp()*1000)
def ct(t): return dt.datetime.fromtimestamp(t/1000,Z).isoformat()
def save(n,x): (P/(n+'.json')).write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
def csvout(n,rows):
    if not rows:return
    keys=list(dict.fromkeys(k for r in rows for k in r))
    with (P/(n+'.csv')).open('w') as f:
        w=csv.DictWriter(f,fieldnames=keys,lineterminator="\n");w.writeheader()
        for r in rows:w.writerow({k:json.dumps(v,ensure_ascii=False) if isinstance(v,(list,dict)) else v for k,v in r.items()})
def quantile(v,p):
    v=sorted(v); q=(len(v)-1)*p; a=int(q);return v[a]+(v[min(a+1,len(v)-1)]-v[a])*(q-a)
def stats(rows,key,idkey='scenario_key'):
    good=[r for r in rows if isinstance(r.get(key),(int,float)) and math.isfinite(r[key])]
    v=[r[key] for r in good]
    return {'n':len(v),'missing':len(rows)-len(v),'ids':[r[idkey] for r in good], **({'min':min(v),'p25':quantile(v,.25),'median':st.median(v),'p75':quantile(v,.75),'max':max(v)} if v else {})}
def groups(rows,keys,idkey='scenario_key'):
    g=co.defaultdict(list)
    for r in rows:g[tuple(r.get(k,'unknown') for k in keys)].append(r[idkey])
    return [{**dict(zip(keys,k)),'n':len(v),'ids':v} for k,v in sorted(g.items(),key=lambda kv:str(kv[0]))]
def kind(label):
    l=label.upper().strip()
    # Descriptive taxonomy; exact label retained. Composite labels are not forced into one kind.
    if '/' in l and not l.startswith('VWAP'):return 'composite'
    for prefix in ['PDVWAP','EVWAP','VWAP','NPOC','POC','RTH-H','RTH-L','EQH','EQL','IB-H','IB-L','OR-H','OR-L','AS-H','AS-L','LDN-H','LDN-L','SUPPLY','DEMAND','IFVG','FVG','OB','SWG-H','SWG-L','GAP','RN']:
        if l.startswith(prefix):return prefix
    if l in ['PDH','PDL','PDC','ONH','ONL','PWH','PWL','PMH','PML','VAH','VAL','SETT','MID-O']:return l
    return 'other'
def family(s):
    cond=s.get('condition',''); tr=s.get('trigger','').lower()
    if tr.strip() in ('none','n/a',''):return 'stand-aside'
    if cond in ('reject','sweep_reclaim'):return 'fade'
    if cond in ('reclaim','breakout_retest','acceptance','breakdown_continue','breakup_continue'):return 'follow'
    # HOLD is overloaded: retain unresolved rather than count every hold as a trade/fade.
    if cond=='hold':
        if any(x in tr for x in ['continuation short','confirms downside break','confirms the 1h ob loss','accept the break','break below','short on acceptance','pdc is reclaimed']):return 'follow'
        if any(x in tr for x in ['pullback','dip','retest','re-test','first touch','first-touch','rallies into','after the eql sweep','after the 01:00 ct sweep','after an onh rejection']):return 'fade'
    return 'ambiguous'

plans=load('plans'); arms=load('arms'); positions=load('positions'); facts=load('facts')
cut=stamp(load('queries')['read_ct']); bars=[b for b in load('bars') if b['tf']=='1m']; bt=[b['open_time_ms'] for b in bars]
pm={}; planrows=[]; levels=[]; scenarios=[]
for p in plans:
    d=json.loads(p['doc']); p['parsed']=d; p['ms']=stamp(p['created_at']); pm[(p['plan_id'],p['version'])]=p
    block=re.search(r'(?:^|\n)### 5m\n(.*?)(?=\n### |\Z)',p['indicators_block'],re.S)
    atr=re.search(r'ATR14:\s*([\d.]+)',block[1]) if block else None
    p['atr']=float(atr[1]) if atr else None
    price=re.search(r'\b(?:current\s+)?price\s*(?:is\s+|at\s+|=\s*|:\s*)?(\d{4,5}(?:\.\d+)?)\b',d.get('reasoning',''),re.I)
    p['narrative_price']=float(price[1]) if price else None
    p['price_quote']=price[0] if price else ''
    idx=bisect.bisect_right(bt,p['ms']-60000)-1
    pub=bars[idx] if idx>=0 and p['ms']-(bt[idx]+60000)<=120000 else None
    p['pub_price']=pub['c'] if pub else None; p['pub_bar']=pub['open_time_ms'] if pub else None
    raw=d.get('day_type','weekly'); norm='trend' if raw.startswith('trend') else 'balance' if raw.startswith('balance') else raw
    p['day_type_group']=norm
    # Regime text explicitly stored on the accepted plan only; no temporal joining of unbound read-facts.
    biaslabel=d.get('bias_label',''); reg=re.search(r'\bregime\s+([^·]+)',biaslabel)
    p['regime']=reg[1].strip() if reg else 'not-recorded'
    planrows.append({'plan_row':p['row_id'],'plan_key':p['plan_id']+':v'+str(p['version']),'session':p['session'],'date':p['trade_date'],'created_ct':ct(p['ms']),'day_type':raw,'day_type_group':norm,'bias':d.get('bias',{}).get('direction','unknown') if isinstance(d.get('bias'),dict) else d.get('bias'),'regime':p['regime'],'scenario_n':len(d.get('scenarios') or []),'atr5m_saved':p['atr'],'narrative_price':p['narrative_price'],'price_quote':p['price_quote'],'publication_price_proxy':p['pub_price'],'publication_bar_ms':p['pub_bar']})
    for i,l in enumerate(d.get('levels') or []):
        lr={'level_key':f"p{p['row_id']}:L{i}",'plan_row':p['row_id'],'session':p['session'],'date':p['trade_date'],'label':l['label'],'kind':kind(l['label']),'grade':l.get('grade'),'machine_grade':l.get('machine_grade'),'price':l['price']}
        for src,px in [('narrative',p['narrative_price']),('publication',p['pub_price'])]:
            lr[src+'_distance_pts']=abs(l['price']-px) if px else None
            lr[src+'_distance_atr']=lr[src+'_distance_pts']/p['atr'] if px and p['atr'] else None
        levels.append(lr)

# Position exclusions are disjoint and fully enumerated; no return/edge statistic is computed.
eligible=[];excluded=[]
for x in positions:
    reason=''
    if x['source']=='e7_farside_test' or x['close_reason']=='e7_farside_test':reason='e7_farside_test'
    elif x['plan_id']=='UNRESOLVABLE':reason='UNRESOLVABLE'
    elif x['pnl_corrected'] is None:reason='NULL pnl_corrected'
    elif x['status']!='CLOSED':reason='not-closed'
    x['entry_ct']=ct(x['entry_time']);x['exit_ct']=ct(x['exit_time']) if x['exit_time'] else None
    x['hold_minutes']=(x['exit_time']-x['entry_time'])/60000 if x['exit_time'] else None
    x['entry_hour_ct']=dt.datetime.fromtimestamp(x['entry_time']/1000,Z).hour
    (excluded if reason else eligible).append({**x,'exclusion':reason})
save('position-cohort',{'eligible':eligible,'excluded':excluded})

for p in plans:
    d=p['parsed']; base=[q['ms'] for q in plans if q['plan_id']==p['plan_id'] and q['ms']>p['ms']]
    date=dt.datetime.fromisoformat(p['trade_date']).replace(tzinfo=Z)
    end={'ASIA':date+dt.timedelta(days=1,hours=2),'LONDON':date+dt.timedelta(hours=8,minutes=30),'NY':date+dt.timedelta(hours=14,minutes=45)}.get(p['session'])
    ends=[cut]+base+([int(end.timestamp()*1000)] if end else [])
    ends += [stamp(l['at']) for l in load('lifecycle') if l['plan_id']==p['plan_id'] and l['version']==p['version'] and stamp(l['at'])>=p['ms'] and l['event'] in ('dormant','invalidated','expired')]
    until=min(ends); window=bars[bisect.bisect_left(bt,p['ms']):bisect.bisect_right(bt,until-60000)] if until>p['ms'] else []
    expected=max(0,(until//60000)-math.ceil(p['ms']/60000))
    complete=bool(expected and len(window)==expected)
    for i,s in enumerate(d.get('scenarios') or []):
        key=f"p{p['row_id']}:{s['id']}:{i}"; ref=(s.get('confirm') or {}).get('ref_price'); anchor_source='confirm.ref_price'
        if not isinstance(ref,(int,float)) or ref<=0:
            nums=[float(n) for n in re.findall(r'(?<![\d.])\d{4,5}(?:\.\d+)?',s['trigger'])]
            refs=[n for n in nums if any(abs(n-l['price'])<=.125 for l in d.get('levels') or [])]
            ref=refs[0] if refs else None;anchor_source='first trigger price matching a seated level' if ref else 'unresolved'
        matches=[(j,l) for j,l in enumerate(d.get('levels') or []) if ref and abs(l['price']-ref)<=.125]
        kinds=sorted(set(kind(l['label']) for _,l in matches)); grades=sorted(set(l.get('grade','unknown') for _,l in matches))
        matcharms=[a for a in arms if a['plan_id']==p['plan_id'] and (a['armed_under_version'] or a['version'])==p['version'] and a['scenario']==s['id']]
        matchpos=[x for x in eligible if x['plan_id']==p['plan_id'] and x['plan_version']==p['version'] and x['cited_scenario_id']==s['id']]
        a=s.get('arm') or {}; e=a.get('entry');stop=a.get('stop');target=a.get('target'); valid=all(isinstance(v,(int,float)) and v>0 for v in (e,stop,target)) and e!=stop
        touched=[b['open_time_ms'] for b in window if ref and b['l']<=ref<=b['h']]
        fam=family(s)
        r={'scenario_key':key,'plan_row':p['row_id'],'plan_id':p['plan_id'],'version':p['version'],'scenario':s['id'],'date':p['trade_date'],'session':p['session'],'session_day':p['trade_date']+':'+p['session'],'created_ct':ct(p['ms']),'hour_ct':dt.datetime.fromtimestamp(p['ms']/1000,Z).hour,'day_type':d.get('day_type'),'day_type_group':p['day_type_group'],'bias':d.get('bias',{}).get('direction','unknown'),'regime':p['regime'],'condition':s['condition'],'family':fam,'side':s['direction'].lower(),'quality':s['quality'],'trigger':s['trigger'],'anchor':ref,'anchor_source':anchor_source,'level_keys':[f"p{p['row_id']}:L{j}" for j,_ in matches],'level_labels':[l['label'] for _,l in matches],'level_kind':'+'.join(kinds) if kinds else 'unresolved','level_grade':'+'.join(grades) if grades else 'unresolved','arm_enabled':bool(a.get('enabled')),'arm_ids':[a['id'] for a in matcharms],'has_arm_evidence':bool(matcharms),'placed_arm_ids':[a['id'] for a in matcharms if a['signal_id']],'position_ids':[x['id'] for x in matchpos],'complete_geometry':valid,'entry':e,'stop':stop,'target':target,'stop_pts':abs(e-stop) if valid else None,'target_R':abs(target-e)/abs(e-stop) if valid else None,'first_target_R':abs(s['target_chain'][0]-e)/abs(e-stop) if valid and s.get('target_chain') else None,'atr5m_saved':p['atr'],'stop_atr':abs(e-stop)/p['atr'] if valid and p['atr'] else None,'reached':'observed' if touched else 'not observed (complete window)' if complete and ref else 'unknown','reach_bar_ids':touched[:1],'window_start_ct':ct(p['ms']),'window_end_ct':ct(until),'window_bars_n':len(window),'expected_bars_n':expected,'window_complete':complete}
        for src,px in [('narrative',p['narrative_price']),('publication',p['pub_price'])]:
            r[src+'_distance_pts']=abs(ref-px) if ref and px else None
            r[src+'_distance_atr']=r[src+'_distance_pts']/p['atr'] if ref and px and p['atr'] else None
        scenarios.append(r)

real=[s for s in scenarios if s['family']!='stand-aside']; empty=[s for s in scenarios if s['family']=='stand-aside']; geom=[s for s in real if s['complete_geometry']]
csvout('scenarios',scenarios);csvout('plan-inventory',planrows);csvout('levels',levels)
tables={}
for name,keys in [('play_session',['session','condition']),('play_day_type',['day_type_group','condition']),('play_day_type_raw',['day_type','condition']),('family_session',['session','family']),('family_day_type',['day_type_group','family']),('sides_session',['session','side']),('sides_day_type',['day_type_group','side']),('sides_regime',['regime','side']),('scenario_hour',['hour_ct','condition']),('scenario_levels_session',['session','level_kind']),('scenario_levels_grade',['level_grade','level_kind']),('arm_presence',['has_arm_evidence','condition']),('arm_presence_quality',['has_arm_evidence','quality']),('reached',['reached']),('bias_vs_side',['bias','side'])]:
    tables[name]=groups(real,keys)
tables['all_retained_condition']=groups(scenarios,['condition']);tables['stand_aside']=groups(empty,['session'])
tables['seated_levels']=groups(levels,['session','kind','grade'],'level_key')
tables['eligible_position_side']=groups(eligible,['plan_session','side'],'id')
tables['eligible_position_hour']=groups(eligible,['entry_hour_ct'],'id')
daily=[]
for day,ss in itertools.groupby(sorted(real,key=lambda x:x['session_day']),key=lambda x:x['session_day']):
    ss=list(ss); fs=co.Counter(s['family'] for s in ss)
    daily.append({'session_day':day,'session':ss[0]['session'],'n':len(ss),'plan_n':len(set(s['plan_row'] for s in ss)),'plan_rows':sorted(set(s['plan_row'] for s in ss)),'ids':[s['scenario_key'] for s in ss],'day_types':dict(co.Counter(s['day_type_group'] for s in ss)),'conditions':dict(co.Counter(s['condition'] for s in ss)),'families':dict(fs),'fade_follow_ratio':fs['fade']/fs['follow'] if fs['follow'] else None,'sides':dict(co.Counter(s['side'] for s in ss)),'level_kinds':dict(co.Counter(s['level_kind'] for s in ss)),'authored_with_arm_evidence_n':sum(s['has_arm_evidence'] for s in ss),'arm_ids':sorted(set(a for s in ss for a in s['arm_ids'])),'placed_arm_ids':sorted(set(a for s in ss for a in s['placed_arm_ids'])),'filled_scenario_n':sum(bool(s['position_ids']) for s in ss),'position_ids':sorted(set(a for s in ss for a in s['position_ids'])),'reached':dict(co.Counter(s['reached'] for s in ss))})
tables['session_days']=daily
calendar=[]
for day in sorted(set(p['trade_date'] for p in plans)):
    ss=[s for s in real if s['date']==day];pp=[p for p in plans if p['trade_date']==day]
    calendar.append({'date':day,'plan_rows':[p['row_id'] for p in pp],'scenario_n':len(ss),'scenario_ids':[s['scenario_key'] for s in ss],'arm_ids':sorted(set(a for s in ss for a in s['arm_ids'])),'authored_with_arm_evidence_n':sum(s['has_arm_evidence'] for s in ss),'filled_scenarios_n':sum(bool(s['position_ids']) for s in ss),'position_ids':sorted(set(a for s in ss for a in s['position_ids'])),'reached':dict(co.Counter(s['reached'] for s in ss))})
tables['calendar_days']=calendar
dist={}
for prefix,rs,idkey in [('scenario',real,'scenario_key'),('seated',levels,'level_key'),('with_arm',[s for s in real if s['has_arm_evidence']],'scenario_key'),('without_arm',[s for s in real if not s['has_arm_evidence']],'scenario_key')]:
    for metric in ['narrative_distance_pts','narrative_distance_atr','publication_distance_pts','publication_distance_atr']:dist[prefix+'_'+metric]=stats(rs,metric,idkey)
for key in ['stop_pts','stop_atr','target_R','first_target_R']:dist[key]=stats(geom,key)
dist['hold_minutes']=stats(eligible,'hold_minutes','id')
excs=[e for e in load('excursions') if e['position_id'] in {x['id'] for x in eligible}]
for e in excs:
    e['initial_stop_pts']=abs(e['entry_px']-e['stop_px_initial']) if e['stop_px_initial']>0 else None
    e['initial_stop_atr']=e['initial_stop_pts']/e['atr5m_at_entry'] if e['initial_stop_pts'] and e['atr5m_at_entry']>0 else None
dist['filled_initial_stop_pts']=stats(excs,'initial_stop_pts','id');dist['filled_initial_stop_atr']=stats(excs,'initial_stop_atr','id')
tables['trade_excursion_stop_changes']=[{'excursion_id':e['id'],'position_id':e['position_id'],'initial':e['stop_px_initial'],'final':e['stop_px_final'],'source':e['source']} for e in excs if e['stop_px_final'] is not None and e['stop_px_initial']!=e['stop_px_final']]

# Paired session-days, same session, >=6 real scenarios and >=2 versions.
# Equal-weight mean of marginal TV distances: condition, side, level kind.
# Marginals avoid declaring all sparse joint tuples different despite shared plays.
vectors={d['session_day']:{field:co.Counter(s[field] for s in real if s['session_day']==d['session_day']) for field in ('condition','side','level_kind')} for d in daily}
pairs=[]
for a,b in itertools.combinations(daily,2):
    if a['session']!=b['session'] or min(a['n'],b['n'])<6 or min(a['plan_n'],b['plan_n'])<2:continue
    components={}
    for field in ('condition','side','level_kind'):
        va,vb=vectors[a['session_day']][field],vectors[b['session_day']][field]
        components[field]=sum(abs(va[k]/a['n']-vb[k]/b['n']) for k in va.keys()|vb.keys())/2
    tv=sum(components.values())/3
    pairs.append({'a':a['session_day'],'b':b['session_day'],'a_n':a['n'],'b_n':b['n'],'a_plan_rows':a['plan_rows'],'b_plan_rows':b['plan_rows'],'components':components,'tv':tv})
pairs.sort(key=lambda x:(x['tv'],x['a'],x['b']));tables['similarity_pairs']=pairs
summary={'snapshot_ct':load('queries')['read_ct'],'plan_n':len(plans),'intraday_plan_n':sum(p['session']!='WEEKLY' for p in plans),'weekly_plan_ids':[p['row_id'] for p in plans if p['session']=='WEEKLY'],'all_scenario_n':len(scenarios),'stand_aside_n':len(empty),'real_scenario_n':len(real),'real_scenario_ids':[s['scenario_key'] for s in real],'conditions':dict(co.Counter(s['condition'] for s in real)),'families':dict(co.Counter(s['family'] for s in real)),'eligible_positions_n':len(eligible),'eligible_position_ids':[p['id'] for p in eligible],'excluded_position_groups':groups(excluded,['exclusion'],'id'),'geometry_n':len(geom),'geometry_ids':[s['scenario_key'] for s in geom],'session_days_n':len(daily),'saved_atr_plans_n':sum(bool(p['atr']) for p in plans),'narrative_price_plans_n':sum(bool(p['narrative_price']) for p in plans),'publication_price_plans_n':sum(bool(p['pub_price']) for p in plans),'mapped_level_scenarios_n':sum(bool(s['level_keys']) for s in real),'facts_without_plan_link_ids':[f['id'] for f in facts if not f['plan_id']],'dist':dist,'closest_pair':pairs[0] if pairs else None,'farthest_pair':pairs[-1] if pairs else None}
save('tables',tables);save('summary',summary)
print(json.dumps({k:v for k,v in summary.items() if k not in ['dist','real_scenario_ids','geometry_ids','facts_without_plan_link_ids']},indent=2))
print('distributions',json.dumps({k:{a:b for a,b in v.items() if a!='ids'} for k,v in dist.items()},indent=2))
