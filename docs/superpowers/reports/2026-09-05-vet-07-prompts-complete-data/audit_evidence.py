import pathlib,csv,json,re,math,sqlite3,datetime,zoneinfo,subprocess
D=pathlib.Path('/home/hoang/nofx-vet-07-complete/docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data'); W=D.parents[3]
def save(n,x):(D/n).write_text(json.dumps(x,indent=2,ensure_ascii=False)+'\n')
def wi(k,n):
 z=1.96;p=k/n;d=1+z*z/n;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n));return [round(100*(p+z*z/(2*n)-h)/d,2),round(100*(p+z*z/(2*n)+h)/d,2)]
patterns=[('transport',r'status 503|stream interrupted|context deadline|Server Overloaded|API error'),('void_reclaimed',r'came back across'),('minimum_confirming_closes',r'NO confirming close'),('minimum_displacement',r'measured displacement|displacement.*below'),('maximum_level_distance',r'BD_MAX_LEVEL_DIST|too far from|pts from price'),('arm_pullback_required',r'arm requires entry_mode=pullback'),('arm_split_shape',r'legs|split|EXACTLY 2|must equal leg 1|leg 2 must chain'),('fade_touch',r'fade_requires_touch'),('entry_rule_pair',r'not allowed for'),('confirm_enum',r'confirm2?\.rule .* invalid'),('gap_trigger_reachability',r'gap-(down|up)'),('level_cap',r'too many levels|0 levels (below|above)')]
rs=list(csv.DictReader((D/'rejects-seven-days.csv').open()));groups={}
for r in rs:
 hit=next((n for n,p in patterns if re.search(p,r['reject_reason'],re.I)),'UNCLASSIFIED');groups.setdefault(hit,[]).append(int(r['id']))
assert 'UNCLASSIFIED' not in groups,groups
save('reject-groups.json',[{'rule':k,'n':len(v),'share_of_64':round(100*len(v)/64,3),'wilson95_percent':wi(len(v),64),'ids':v} for k,v in sorted(groups.items(),key=lambda kv:-len(kv[1]))])
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('pragma query_only=on');c.execute('begin')
records=[]
for rr in c.execute("select id,timestamp,decision_json,execution_log,ai_request_duration_ms,system_prompt from decision_records where datetime(timestamp)>=datetime('2026-08-29 05:00:00') and datetime(timestamp)<datetime('2026-09-05 05:00:00')"):
 try: ds=json.loads(rr['decision_json'])
 except:continue
 if not isinstance(ds,list):continue
 for d in ds:
  if not isinstance(d,dict) or d.get('action') not in ('open_long','open_short'):continue
  r={k:rr[k] for k in ['id','timestamp','ai_request_duration_ms']};r.update(action=d['action'],cited_scenario=d.get('cited_scenario'),strict_refused='entry_gate: refused: strict' in rr['execution_log']);records.append(r)
  if r['strict_refused'] and rr['id']==37304:(D/'executor-37304-system-actual.txt').write_text(rr['system_prompt'])
save('open-proposals-seven-days.json',records)
strict=[r for r in records if r['strict_refused']];save('strict-refusals.json',{'ids':[r['id'] for r in strict],'n':len(strict),'cited':sorted(set(r['cited_scenario'] for r in strict)),'proposal_rows':strict})
row=c.execute("select plan_id,version,doc,created_at from plans where trade_date='2026-09-03' and session='ASIA' and version=5").fetchone();save('plan-asia-v5.json',dict(row) if row else {})
plans=[dict(r) for r in c.execute("select plan_id,version,trade_date,session,trigger_reason,created_at from plans where datetime(created_at)>=datetime('2026-08-29 05:00:00') and datetime(created_at)<datetime('2026-09-05 05:00:00') order by created_at")];save('plans-seven-days.json',plans)
# Verify all stored full prompt shortages freshly; ids retained.
prompts=[dict(r) for r in c.execute('select id,attempt,prompt_text from planner_rejected_prompts order by id')];full=[r for r in prompts if '# DAY-PLAN READER' in r['prompt_text']]
checks={
 'full_prompts':[r['id'] for r in full],
 'empty_fvg':[r['id'] for r in full if '(none fresh right now)' in r['prompt_text']],
 'daily_and_4h_structure_unavailable':[r['id'] for r in full if 'D: unavailable' in r['prompt_text'] and '4h: unavailable' in r['prompt_text']],
 'vix_unavailable':[r['id'] for r in full if 'VIX=n/a' in r['prompt_text']],
 'attempt1_with_corrections':[r['id'] for r in prompts if r['attempt']==1 and 'CORRECTIONS FROM THIS READ' in r['prompt_text']],
 'attempt1':[r['id'] for r in prompts if r['attempt']==1]}
save('prompt-population-checks.json',checks)
c.rollback();c.close()
# Each non-fact original span maps to rewrite line(s), then split Rules into every sentence.
p=(D/'planner-132-current-contract-replay.txt').read_text().splitlines();rr=list(csv.DictReader((D/'prompt-boundaries.csv').open()));m={1:[2],2:[2],3:[2],4:[2],5:[2],7:[2],9:[1],10:[1],14:[1],15:[1],103:[7],104:[7],105:[7],108:[26],111:[18],127:[3],128:[3],141:[10],157:[4],158:[4],170:[5],177:[11,28],178:[11,28],179:[11,28],214:[3],217:[3],229:[8],230:[8],232:[6],233:[6],234:[6],235:[6],236:[6],237:[6],238:[6],241:[6],243:[9],244:[9],246:[11],247:[11],248:[11],249:[11],250:[1,11],251:[11],253:[12],254:[12],256:[12],257:[12],259:[13],260:[13],261:[13],262:[13],263:[5,13],264:[14,15,17,18,21,22,24],265:[13,28],266:[13],267:[13,15],268:[13,15],269:[13],270:[13],274:[2],275:[2],276:[2],277:[2],278:[2]}
maprows=[]
for r in rr:
 if r['document']!='planner-132-current-contract-replay.txt' or r['category']=='F' or not r['text'].strip():continue
 n=int(r['line'])
 if n==271:continue
 assert n in m,n
 maprows.append({'unit':f'L{n}','source':f'planner-132-current-contract-replay.txt:{n}','original':r['text'].strip(),'rewrite_lines':m[n],'status':'preserved; duplicate wording consolidated where applicable'})
rules=re.split(r'(?<=[.!?])\s+(?=[A-Z`"(“])',p[270]);rm=[[5],[5],[14],[7],[7],[7],[6],[6],[9],[10],[10],[10,15],[15],[15],[15],[16],[15,20],[24],[24],[24],[21],[21],[21],[21],[21],[22,23],[22],[23],[19,23],[22,23],[21],[23],[23],[25],[25],[25],[25],[26],[26],[26],[27],[18,19],[5],[17],[17,28],[28],[28],[28],[3],[3],[20],[20]]
assert len(rm)==len(rules)==52
for i,(s,lines) in enumerate(zip(rules,rm),1):maprows.append({'unit':f'Rules-{i:02}','source':'planner-132-current-contract-replay.txt:271','original':s,'rewrite_lines':lines,'status':'preserved; original contradictions retained'})
# Instructions embedded in retained fact context remain literal and also map semantically.
maprows.append({'unit':'computed-premium-veto','source':'planner-132-current-contract-replay.txt:239','original':p[238],'rewrite_lines':[6],'status':'retained fact context; prohibition also covered'})
# Optional unrendered source branches are NOT silently claimed covered by the real-read rewrite.
maprows.extend([
 {'unit':'conditional-warming','source':'kernel/planner_prompt.go:416','original':'WARMING: ... (first-week honesty — narrate the machinery, not an edge).','rewrite_lines':[1],'status':'conditional current-source branch additionally preserved; absent from this real read'},
 {'unit':'conditional-nearest-1h-zone','source':'kernel/planner_prompt.go:690','original':'the nearest 1h supply/demand zone row in that section MUST be one of your included rows','rewrite_lines':[5],'status':'conditional current-source branch additionally preserved; absent from this real read'}])
save('constraint-map.json',maprows)
lines=['# Constraint coverage: current contract on real read 132','', 'This maps every instruction/schema span and each Rules sentence. Line references point to adjacent evidence files. Original errors are retained, not endorsed. Two conditional source branches absent from the real read are additionally included, without adding their original length to the compression denominator.','', '| Unit | Original location | Rewrite line(s) | Status | Original text |','|---|---|---|---|---|']
for r in maprows:
 esc=lambda s:s.replace('|','\\|').replace('\n',' ')
 lines.append('| '+ ' | '.join([r['unit'],r['source'],','.join(map(str,r['rewrite_lines'])) or '—',r['status'],esc(r['original'])])+' |')
(D/'constraint-map.md').write_text('\n'.join(lines)+'\n')
print('reject groups',groups);print('strict',len(strict));print('coverage units',len(maprows));print('full shortage counts',{k:len(v) for k,v in checks.items()});print('plans',len(plans))
