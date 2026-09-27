import sqlite3, json, datetime, collections, math, statistics
db=sqlite3.connect('file:data-copy.db?mode=ro', uri=True); db.row_factory=sqlite3.Row
TR='8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265'
ct=lambda ms: datetime.datetime.fromtimestamp(ms/1000).strftime('%m-%d %H:%M') if ms else '-'
BAND=3.0
GR={'A+':4,'A':3,'B':2,'C':1}
def cms(s): # '2026-09-10 12:52:00' CT -> ms
    return int(datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').timestamp()*1000)
W1=cms('2026-09-10 12:52:37'); WTF=cms('2026-09-10 17:11:34'); W2=cms('2026-09-10 18:47:07')
out={}
plans={ (p['plan_id'],p['version']):p for p in db.execute("SELECT plan_id, version, session, trade_date, doc, created_at FROM plans WHERE strategy_id=?", (TR,)) }
docs={}
for k,p in plans.items():
    try: docs[k]=json.loads(p['doc'])
    except Exception: pass
def anchor(s):
    c=s.get('confirm') or {}; a=s.get('arm') or {}
    return c.get('ref_price') or a.get('entry')
# ---------- C1 (price-proximity link, band 3.0)
arms=list(db.execute("SELECT * FROM armed_orders WHERE trader_id=?", (TR,)))
arms_by=collections.defaultdict(list)
for a in arms: arms_by[(a['plan_id'],a['version'])].append(a)
c1={'plans_ge2_armable':0,'first_touched_armed':[],'first_touched_not_armed':[],'no_touch_on_any_scenario':[],'no_ledger_row':[]}
for k,doc in docs.items():
    scs=[s for s in doc.get('scenarios',[]) if (s.get('arm') or {}).get('enabled') and anchor(s)]
    if len(scs)<2: continue
    c1['plans_ge2_armable']+=1
    rows=db.execute("SELECT level_price, opened_at_ms FROM touch_outcomes WHERE trader_id=? AND plan_id=? AND plan_version=? ORDER BY opened_at_ms",(TR,k[0],k[1])).fetchall()
    first=None
    for r in rows:
        hits=[s['id'] for s in scs if abs(r['level_price']-anchor(s))<=BAND]
        if len(hits)==1: first=(hits[0],r['opened_at_ms']); break
    armed={a['scenario'] for a in arms_by[k]}
    tag=(k[0][:16],k[1])
    if not armed: c1['no_ledger_row'].append(tag); continue
    if first is None: c1['no_touch_on_any_scenario'].append(tag); continue
    (c1['first_touched_armed'] if first[0] in armed else c1['first_touched_not_armed']).append((tag,first[0],ct(first[1]),sorted(armed)))
out['C1']={kk:(vv if isinstance(vv,int) else {'n':len(vv),'ids':vv}) for kk,vv in c1.items()}
# ---------- C2 pnl_corrected by condition
cells=collections.defaultdict(lambda: {'n':0,'sum':0.0,'ids':[],'null':0,'pnls':[]})
for q in db.execute("SELECT id, pnl_corrected, cited_scenario_id, plan_id, plan_version FROM trader_positions WHERE trader_id=? AND status='CLOSED'", (TR,)):
    cond=None
    ex=db.execute("SELECT condition FROM trade_excursions WHERE position_id=? AND condition!='' LIMIT 1",(q['id'],)).fetchone()
    if ex: cond=ex['condition']
    if not cond and q['cited_scenario_id'] and q['plan_id']:
        a=db.execute("SELECT condition FROM armed_orders WHERE plan_id=? AND scenario=? AND condition!='' LIMIT 1",(q['plan_id'],q['cited_scenario_id'])).fetchone()
        if a: cond=a['condition']
        if not cond and (q['plan_id'],q['plan_version']) in docs:
            for s in docs[(q['plan_id'],q['plan_version'])].get('scenarios',[]):
                if s.get('id')==q['cited_scenario_id']: cond=s.get('condition')
    c=cells[cond or 'UNLINKED']
    if q['pnl_corrected'] is None: c['null']+=1; continue
    c['n']+=1; c['sum']+=q['pnl_corrected']; c['ids'].append(q['id']); c['pnls'].append(q['pnl_corrected'])
out['C2']={k:{'n':v['n'],'sum_usd':round(v['sum'],2),'mean':round(v['sum']/v['n'],2) if v['n'] else None,'null_excluded':v['null'],'ids':v['ids']} for k,v in cells.items()}
# ---------- C3 best level near price at each scenario evaluation since W-TF boot (candidate_pool reads)
reads=db.execute("SELECT read_at_ms, plan_id, plan_version FROM candidate_pool WHERE trader_id=? AND read_at_ms>=? GROUP BY read_at_ms ORDER BY read_at_ms",(TR,WTF)).fetchall()
c3=[]
for r in reads:
    pool=db.execute("SELECT level_price, level_kind, label, grade, rank, seated, score FROM candidate_pool WHERE trader_id=? AND read_at_ms=?",(TR,r['read_at_ms'])).fetchall()
    # price at the read: last 1m close at/before read (contract-agnostic: bars are single-scale before 21:15 on 09-10; after, use 09-26 label)
    b=db.execute("SELECT c FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms<=? AND (contract='MNQ 09-26' OR contract='') AND (source='live' OR source='historical' OR source='') ORDER BY open_time_ms DESC LIMIT 1",(r['read_at_ms'],)).fetchone()
    price=b['c'] if b else None
    # merge within 3pt, strongest first
    cands=sorted(pool,key=lambda x:(-(x['score'] or 0),x['level_price']))
    merged=[]
    for c in cands:
        for m in merged:
            if abs(m['price']-c['level_price'])<=BAND: m['names'].append(c['label']); break
        else: merged.append({'price':c['level_price'],'grade':c['grade'],'names':[c['label']],'seated':c['seated']})
    if price is None: c3.append({'read':ct(r['read_at_ms']),'top':None,'why':'no price'}); continue
    # band: proximityK*dATR unknown here -> use all merged (all pool rows already passed the proximity filter at the read)
    ranked=sorted(merged,key=lambda m:(-GR.get((m['grade'] or '').upper(),0),abs(m['price']-price)))
    top=ranked[0] if ranked else None
    k=(r['plan_id'],r['plan_version']); doc=docs.get(k,{})
    armed=[(a['scenario'],a['state']) for a in arms_by[k]]
    on_top=[]
    for s in doc.get('scenarios',[]):
        if (s.get('arm') or {}).get('enabled') and anchor(s) and top and abs(anchor(s)-top['price'])<=BAND: on_top.append(s['id'])
    armed_on_top=[a for a in armed if a[0] in on_top]
    c3.append({'read':ct(r['read_at_ms']),'plan':(k[0][:16],k[1]),'price':price,'top':(top['price'],top['grade'],top['names'][:3]) if top else None,'scenarios_on_top':on_top,'armed':armed,'armed_on_top':armed_on_top,'n_merged':len(merged)})
out['C3']={'n_reads':len(c3),'reads':c3}
# ---------- C4 first live rows with fade_permitted since W2 boot
c4=db.execute("SELECT id, level_price, level_kind, entry_side, outcome, fade_permitted, fade_exclusions, datetime(fade_evaluated_ms/1000,'unixepoch','localtime') ev FROM touch_outcomes WHERE trader_id=? AND fade_evaluated_ms>=? ORDER BY id LIMIT 8",(TR,W2)).fetchall()
cnt=db.execute("SELECT SUM(fade_permitted=1), SUM(fade_permitted=0), SUM(fade_permitted IS NULL), COUNT(*) FROM touch_outcomes WHERE trader_id=? AND opened_at_ms>=?",(TR,W2)).fetchone()
excl=collections.Counter(r['fade_exclusions'] for r in db.execute("SELECT fade_exclusions FROM touch_outcomes WHERE trader_id=? AND fade_permitted=0 AND opened_at_ms>=?",(TR,W2)))
out['C4']={'first_rows':[dict(r) for r in c4],'since_W2':{'permitted':cnt[0],'excluded':cnt[1],'null':cnt[2],'n':cnt[3]},'exclusions':dict(excl)}
# ---------- C5 first obstacle on new scenarios + obstacle R distribution
obs={'scenarios':0,'with_obstacle':0,'obstacle_R':[],'ids_missing':[]}
for k,doc in docs.items():
    if plans[k]['created_at']<'2026-09-08': continue
    for s in doc.get('scenarios',[]):
        obs['scenarios']+=1
        o=((s.get('economics') or {}).get('first_obstacle') or {})
        a=s.get('arm') or {}
        if o.get('price'):
            obs['with_obstacle']+=1
            if a.get('entry') and a.get('stop') and a['entry']!=a['stop']:
                R=abs(o['price']-a['entry'])/abs(a['entry']-a['stop']); obs['obstacle_R'].append(round(R,2))
        else: obs['ids_missing'].append((k[0][:16],k[1],s.get('id')))
rs=sorted(obs['obstacle_R'])
out['C5']={'scenarios_since_0908':obs['scenarios'],'with_obstacle':obs['with_obstacle'],'missing':obs['ids_missing'][:20],'n_R':len(rs),'R_p25_p50_p75':[rs[len(rs)//4],rs[len(rs)//2],rs[3*len(rs)//4]] if rs else None,'R_lt1':sum(1 for x in rs if x<1),'R_lt2':sum(1 for x in rs if x<2)}
# ---------- C9 hold/break/ambiguous by entry side × level kind since W1 boot (valid rows only + all rows)
c9=collections.defaultdict(lambda: collections.Counter())
c9v=collections.defaultdict(lambda: collections.Counter())
for r in db.execute("SELECT entry_side, level_kind, outcome, validity FROM touch_outcomes WHERE trader_id=? AND opened_at_ms>=?",(TR,W1)):
    o='ambiguous' if r['outcome'].startswith('ambiguous') else r['outcome']
    c9[(r['entry_side'],r['level_kind'])][o]+=1
    if r['validity']=='valid': c9v[(r['entry_side'],r['level_kind'])][o]+=1
out['C9']={'all_rows':{f"{k[0]}|{k[1]}":dict(v) for k,v in sorted(c9.items())},'valid_only':{f"{k[0]}|{k[1]}":dict(v) for k,v in sorted(c9v.items())}}
side=collections.defaultdict(collections.Counter)
for (es,lk),v in c9.items():
    for o,n in v.items(): side[es][o]+=n
out['C9']['by_side_all']={k:dict(v) for k,v in side.items()}
json.dump(out,open('sectionC.json','w'),indent=1,default=str)
for k in ('C1','C2','C4','C5','C9'):
    print('=====',k); print(json.dumps(out[k],default=str)[:1800])
print('===== C3 n_reads',out['C3']['n_reads'])
for r in out['C3']['reads']: print(r)
