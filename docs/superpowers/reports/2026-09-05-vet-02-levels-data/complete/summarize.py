#!/usr/bin/env python3
"""Offline summaries from audit.py snapshots; no trading-system access."""
import json,pathlib,math,collections,datetime
from zoneinfo import ZoneInfo
p=pathlib.Path(__file__).resolve().parent
load=lambda f:json.loads((p/f).read_text());rs=load('candidate_rows.json');ts=load('touch_rows_quarantined.json');obs=load('forward_rows.json');ps=load('plan_rows.json');su=load('summary.json');pos=load('positions.json')
ct=lambda t:datetime.datetime.fromtimestamp(t/1000,ZoneInfo('America/Chicago')).isoformat()
def wi(k,n):
 if not n:return 'UNMEASURABLE n=0'
 z=1.959963984540054;v=k/n;d=1+z*z/n;m=(v+z*z/(2*n))/d;h=z*math.sqrt(v*(1-v)/n+z*z/(4*n*n))/d
 return f'{k}/{n}={v:.1%} [{m-h:.1%},{m+h:.1%}]'
# No ever-seated grouping: status at FIRST appearance is the as-of exposure.
first=[];seen=set()
for r in obs:
 key=(r['level_kind'],r['price'])
 if key not in seen:seen.add(key);first.append(r)
lines=[]
lines.append('E01 population eligible IDs='+','.join(str(r['id']) for r in pos['eligible'])+'; n=58; pnl_corrected='+str(pos['pnl_corrected'])+'; W/L/F=18/38/2; CME days='+str(pos['cme_days'])+'; post strict='+str(pos['post_strict']))
lines.append('E02 excluded positions='+str([(r['id'],r['plan_id'],r['source'],r['pnl_corrected']) for r in pos['excluded']]))
lines.append('E03 candidate IDs 1..360, 15 reads on 2026-09-04 00:30:54.769..12:01:01.727 CT; seated 179 cuts181; each read24; 14 reads12matched seats and 01:30read11; cuts all max_levels12; components360 {}; cut scores181 zeros; count and rank are not independent trials.')
lines.append('E04 raw touches677; distinct (kind,price,opened)423; conflicting keys='+str(su['touch_conflicts']))
rth=[r for r in ts if r['level_kind']=='RTH-L'];keys=collections.defaultdict(list)
for r in rth:keys[(r['level_price'],r['opened_at_ms'])].append(r)
lines.append('E05 RTH-L rawIDs='+','.join(str(r['id']) for r in rth)+'; raw H/B/A=42/82/16; hold '+wi(42,124)+'; break '+wi(82,124)+'; amb '+wi(16,140))
for (px,t),rows in sorted(keys.items()):lines.append('E06 RTH-L key '+str(px)+' '+str(t)+' '+ct(t)+' IDs='+','.join(str(r['id']) for r in rows)+' outcomes='+str(dict(collections.Counter(r['outcome'] for r in rows))))
lines.append('E07 bars MNQ1m bound='+str(su['bars_bounds'])+'; last='+ct(su['bars_bounds'][-1])+'; first exposure levels='+str(len(first))+' complete='+str(sum(r['status']=='complete_60m' for r in first))+'; remaining right censored.')
for group in ['first_exposure','matched']:
 for k,v in su[group].items():lines.append('E08 '+group+' '+k+' '+json.dumps(v))
lines.append('E09 matched pairs='+str(su['matched_n'])+'; unique touched (price,time)='+str(su['matched_distinct_price_touch_keys'])+'; all ONE CME day; no clustered CI estimable from one day.')
for id in [10,26,50,55,76,121,122,193,195,337,358,360]:
 lines.append('E10 candidate '+json.dumps(next(r for r in rs if r['id']==id)))
for id in [121,122,193,195]:lines.append('E11 forward '+json.dumps(next(r for r in obs if r['candidate_id']==id)))
for r in ps:
 if r['row_id'] in [252,253,254]:lines.append('E12 plan '+json.dumps({k:r[k] for k in ['row_id','plan_id','version','created_at']})+' levels='+json.dumps(r['doc'].get('levels')))
lines.append('E13 grade count='+str(dict(collections.Counter(r['grade'] for r in rs if r['seated'])))+'; stored score min/max='+str((min(r['score'] for r in rs if r['seated']),max(r['score'] for r in rs if r['seated']))))
for kind in sorted(set(r['level_kind'] for r in rs+ts)):
 a=[r for r in rs if r['level_kind']==kind];t=[r for r in ts if r['level_kind']==kind];f=[r for r in first if r['level_kind']==kind and r['status']=='complete_60m'];touch=[r for r in f if r['touch_ms']];res=[r for r in f if r['outcome'] in ['hold','break']]
 lines.append('E14 '+kind+' poolIDs='+str([r['id'] for r in a])+' seatedIDs='+str([r['id'] for r in a if r['seated']])+' seat='+wi(sum(r['seated'] for r in a),len(a))+' rawtouchIDs='+str([r['id'] for r in t])+' forwardIDs='+str([r['candidate_id'] for r in f])+' touch='+wi(len(touch),len(f))+' H/B/A/censor='+str([sum(r['outcome']==x for r in f) for x in ['hold','break','ambiguous_horizon','censored_12bar']])+' hold='+wi(sum(r['outcome']=='hold' for r in res),len(res))+' break='+wi(sum(r['outcome']=='break' for r in res),len(res))+' amb='+wi(sum(r['outcome']=='ambiguous_horizon' for r in touch),len(touch)))
(p/'evidence.txt').write_text('\n'.join(lines)+'\n')
# One complete kind inventory, with grouped same-definition kinds only when all absent.
defs=[
('PDH','prior CT calendar-day high; >=900 closed 1m bars; ring2000','kernel/levels_multiday.go:146','L1,T,H'),('PDL','same bucket low','kernel/levels_multiday.go:154','L1,T,H'),('PDC','same bucket final close','kernel/levels_multiday.go:159','L1,T,H'),('RTH-H','prior calendar-day NY high; 08:30–14:45 CT registry','kernel/levels_multiday.go:161','L1,T'),('RTH-L','same NY-window low','kernel/levels_multiday.go:164','L1,T'),
('ONH','current CME-day AS+London high, develops until08:30','kernel/levels_multiday.go:183','L.85,T'),('ONL','same overnight low','kernel/levels_multiday.go:188','L.85,T'),('AS-H','current AS 17:00–02:00 high','kernel/levels_multiday.go:171','L.70'),('AS-L','same AS low','kernel/levels_multiday.go:174','L.70'),('LDN-H','current London02:00–08:30 high','kernel/levels_multiday.go:177','L.70'),('LDN-L','same London low','kernel/levels_multiday.go:180','L.70'),
('PWH','prior CT week high; guard4320 bars impossible on2000 ring','kernel/levels_multiday.go:198','L1,H,t'),('PWL','same prior-week low','kernel/levels_multiday.go:202','L1,H,t'),('PMH','prior month high; guard10080 bars','kernel/levels_multiday.go:205','L1,H,t'),('PML','same month low','kernel/levels_multiday.go:209','L1,H,t'),
('OR-H','first5min high; DEVELOPING emitted before08:35','kernel/levels_intraday.go:143','L.70,T'),('OR-L','same first5min low','kernel/levels_intraday.go:166','L.70,T'),('IB-H','first60min high plus1.5x/2x extensions; DEVELOPING before09:30','kernel/levels_intraday.go:172','L.70'),('IB-L','same low and lower extensions','kernel/levels_intraday.go:179','L.70'),
('VWAP','17:00 CME anchor; >=2 closed bars, typical-price volume weighting; ±1σ same kind','kernel/levels_volume.go:35','L.90,V'),('VWAP±2σ','same developing VWAP±2σ','kernel/levels_volume.go:58','L.85,V'),('eVWAP','last15:00 CT anchor; evolving','kernel/levels_volume.go:97','L.85,V'),('pdVWAP','previous CME24h bucket; >=2bars','kernel/levels_volume.go:318','L.85,V'),
('POC','prior CME-day120-bin CLOSE-bin volume proxy max','kernel/levels_volume.go:162','L.90,V'),('VAH','upper boundary70% proxy value area','kernel/levels_volume.go:206','L.80,V,t'),('VAL','lower boundary70% proxy value area','kernel/levels_volume.go:206','L.80,V,t'),('nPOC','untouched prior POC; up to10sessions, ring+durable extras','kernel/levels_volume.go:257','L.85,V,t'),('SETT','prior24h final available1m close; NOT verified official settlement','kernel/levels_volume.go:342','L.80,V,t'),('MID-O','current overnight midpoint','kernel/levels_volume.go:366','L.60,V'),
('RN','100/50/25 multiples inside proximity band','kernel/levels_intraday.go:21','L.55'),('GAP','unfilled1m gap edge >=ATR14 multiplier1','kernel/levels_intraday.go:57','L.55'),('EQH','k2 pivot clusters3ticks; HTF tolerance max3ticks,.15TFATR','kernel/levels_zones.go:34','L.70,H if tagged'),('EQL','same pivot-low clusters','kernel/levels_zones.go:34','L.70,H if tagged'),('SWG-H','recent5m/15m fractal swings; k/minmove; lookbacks144/96;3perTF/side','kernel/levels_swing.go:38','L.85,V'),('SWG-L','same swing lows','kernel/levels_swing.go:38','L.85,V'),
('SUPPLY','base<=6bars, bodies<=.5ATR, departure>=1.5ATR;1m and configured HTF','kernel/levels_zones.go:103','Z'),('DEMAND','same demand base; confirmed departure birth','kernel/levels_zones.go:147','Z'),('FVG','3bar imbalance;1m floor max2ticks,2pt; HTF calls gap floor=TFATR','kernel/levels_zones.go:213','Z'),('IFVG','inverse filled FVG, same detector; excluded separate HTFzone section','kernel/levels_zones.go:260','Z'),('OB','opposing candle within8bars before displacement;1m/configuredHTF','kernel/levels_zones.go:296','Z_OB'),('OWNER','active manually-set sticky level; appended after cap','trader/auto_trader_planner.go:2185','A fixed')]
out=['| Kind | Definition and window · source | Grade/seat terms | Pool seats/candidates: rate [Wilson95] | First-exposure forward: touched/exposures; H/B/A/censored; hold & break [Wilson95]; ambiguous share [Wilson95] |','|---|---|---|---|---|']
for k,de,src,g in defs:
 a=[r for r in rs if r['level_kind']==k];f=[r for r in first if r['level_kind']==k and r['status']=='complete_60m'];touch=[r for r in f if r['touch_ms']];res=[r for r in f if r['outcome'] in ['hold','break']];h=sum(r['outcome']=='hold' for r in f);b=sum(r['outcome']=='break' for r in f);amb=sum(r['outcome']=='ambiguous_horizon' for r in f);cen=sum(r['outcome']=='censored_12bar' for r in f)
 out.append('| '+k+' | '+de+'; `'+src+'` | '+g+' | '+wi(sum(r['seated'] for r in a),len(a))+' | '+('No eligible forward observation; UNMEASURABLE' if not f else 'touch '+wi(len(touch),len(f))+f'; {h}/{b}/{amb}/{cen}; H '+wi(h,len(res))+'; B '+wi(b,len(res))+'; A '+wi(amb,len(touch)))+' |')
(p/'inventory.md').write_text('\n'.join(out)+'\n')
print('evidence lines',len(lines),'inventory kinds',len(defs))
