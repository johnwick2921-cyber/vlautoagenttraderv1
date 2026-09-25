#!/usr/bin/env python3
"""q12: GRADE-AT-SEAT vs OUTCOME over ALL plans (08-15..09-04). For every seated level in every plan version:
run D1' (k=3,H=12,close, delta trailing-5d) on the 1m tape from the plan's created_at to the end of its CME session day,
take the FIRST episode's outcome (and all episodes). Aggregate by grade, machine_grade, kind family, instruction keyword, session.
Also: seat utilisation = share of seated levels that were touched at all before the session-day end.
Also: candidate_pool (09-04) seated vs cut -> forward outcome from read_at."""
import sqlite3, json, math, collections, re, datetime
exec(open('/home/hoang/nofx-analysis/vet-02-0905/q11_replay.py').read().split("# group bars by session day")[0])  # reuse loaders + detect()
by_sd=collections.OrderedDict()
for b in bars: by_sd.setdefault(sess_day(b['t']),[]).append(b)
sdays=sorted(by_sd)
def delta_for(day):
    idx=sdays.index(day) if day in sdays else len(sdays)
    trail=[b for dd in sdays[max(0,idx-5):idx] for b in by_sd[dd]]
    if len(trail)<2: trail=by_sd[day]
    return sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/max(1,len(trail)-1)
def fam(label):
    l=label or ''
    for pre,f in (('VWAP','VWAP*'),('SWG-H','SWG-H'),('SWG-L','SWG-L'),('OB(','OB'),('Demand','DEMAND'),('Supply','SUPPLY'),('FVG','FVG'),('iFVG','IFVG'),('EQH','EQH'),('EQL','EQL'),('nPOC','nPOC'),('RN ','RN'),('IB','IB*')):
        if l.startswith(pre): return f
    return l
def utc_ms(s):
    s=s.replace('+00:00','').split('.')[0]
    return int(datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').replace(tzinfo=datetime.timezone.utc).timestamp()*1000)
con2=sqlite3.connect(DB, uri=True); con2.row_factory=sqlite3.Row
recs=[]
for r in con2.execute("SELECT plan_id, version, session, trade_date, created_at, doc FROM plans WHERE session<>'WEEKLY'"):
    t0=utc_ms(r['created_at']); day=sess_day(t0)
    if day not in by_sd: continue
    tape=[b for b in by_sd[day] if b['t']>=t0]
    if len(tape)<15: continue
    delta=delta_for(day)
    try: doc=json.loads(r['doc'])
    except Exception: continue
    for l in doc.get('levels') or []:
        try: px=float(l.get('price'))
        except Exception: continue
        eps=detect(tape, px, 3.0, delta, 12, 'close')
        first=eps[0]['outcome'] if eps else 'untouched'
        outs=[e['outcome'] for e in eps]
        recs.append(dict(plan=r['plan_id'],ver=r['version'],session=r['session'],day=r['trade_date'],label=l.get('label'),fam=fam(l.get('label')),grade=l.get('grade'),mgrade=l.get('machine_grade') or '',instr=(l.get('instruction') or '').lower(),px=px,first=first,n_eps=len(eps),hold=outs.count('hold'),brk=outs.count('break'),amb=len(eps)-outs.count('hold')-outs.count('break'),dist=abs(px-tape[0]['c'])))
def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (cen-half,cen+half)
def tab(name, keyf, rows=recs, minn=1):
    g=collections.defaultdict(lambda: dict(n=0,touched=0,f_hold=0,f_brk=0,f_amb=0,hold=0,brk=0,amb=0,dist=0.0))
    for x in rows:
        a=g[keyf(x)]; a['n']+=1; a['dist']+=x['dist']
        if x['first']!='untouched': a['touched']+=1
        if x['first']=='hold': a['f_hold']+=1
        elif x['first']=='break': a['f_brk']+=1
        elif x['first']!='untouched': a['f_amb']+=1
        a['hold']+=x['hold']; a['brk']+=x['brk']; a['amb']+=x['amb']
    out=[f"\n== {name}", f"{'key':22s} {'seats':>5s} {'touched%':>8s} {'1st:hold':>8s} {'1st:brk':>7s} {'1st:amb':>7s} {'p_hold1':>7s} {'wilson95':>16s} | {'all:hold':>8s} {'all:brk':>7s} {'p_hold_all':>10s} {'avg_dist':>8s}"]
    for k,a in sorted(g.items(), key=lambda kv:-kv[1]['n']):
        if a['n']<minn: continue
        d1=a['f_hold']+a['f_brk']; p1=a['f_hold']/d1 if d1 else float('nan'); lo,hi=wilson(a['f_hold'],d1)
        da=a['hold']+a['brk']; pa=a['hold']/da if da else float('nan')
        out.append(f"{str(k)[:22]:22s} {a['n']:5d} {100*a['touched']/a['n']:8.1f} {a['f_hold']:8d} {a['f_brk']:7d} {a['f_amb']:7d} {p1:7.3f} [{lo:.3f},{hi:.3f}] | {a['hold']:8d} {a['brk']:7d} {pa:10.3f} {a['dist']/a['n']:8.1f}")
    return "\n".join(out)
rep=[f"seated-level rows evaluated: {len(recs)} (plans with >=15 bars after created_at, non-WEEKLY)"]
rep.append(tab("by AI grade", lambda x:x['grade']))
rep.append(tab("by machine_grade", lambda x:x['mgrade'] or '(none)'))
rep.append(tab("by kind family", lambda x:x['fam'], minn=5))
rep.append(tab("by kind family x grade (n>=8)", lambda x:(x['fam'],x['grade']), minn=8))
rep.append(tab("by session", lambda x:x['session']))
def ik(s):
    for w in ('target only','target','fade','reject','react','flip','death','liquidity','sweep','confluence','magnet','pivot','trigger','support','resistance'):
        if w in s: return w
    return 'other'
rep.append(tab("by AI instruction keyword", lambda x:ik(x['instr'])))
rep.append(tab("by distance-at-seat bucket (pts)", lambda x: '<25' if x['dist']<25 else '<50' if x['dist']<50 else '<100' if x['dist']<100 else '<200' if x['dist']<200 else '200+'))
# candidate_pool 09-04: seated vs cut forward outcome from read_at
cp=[]
for r in con2.execute("SELECT plan_id, plan_version, session, read_at_ms, level_price, level_kind, label, seated, grade, score, cut_reason FROM candidate_pool"):
    day=sess_day(r['read_at_ms'])
    if day not in by_sd: continue
    tape=[b for b in by_sd[day] if b['t']>=r['read_at_ms']]
    if len(tape)<15: continue
    eps=detect(tape,float(r['level_price']),3.0,delta_for(day),12,'close'); outs=[e['outcome'] for e in eps]
    cp.append(dict(fam=r['level_kind'],grade=r['grade'] or '(cut)',seated=r['seated'],first=(eps[0]['outcome'] if eps else 'untouched'),hold=outs.count('hold'),brk=outs.count('break'),amb=len(eps)-outs.count('hold')-outs.count('break'),dist=abs(float(r['level_price'])-tape[0]['c']),instr='',n_eps=len(eps)))
rep.append(tab("candidate_pool 09-04: SEATED (1) vs CUT (0) — forward from read_at", lambda x:('seated' if x['seated'] else 'cut'), rows=cp))
rep.append(tab("candidate_pool 09-04: by kind x seated", lambda x:(x['fam'],'S' if x['seated'] else 'C'), rows=cp, minn=4))
txt="\n".join(rep); print(txt); open(OUT+"q12_grade_vs_outcome.txt","w").write(txt)
import csv
with open(OUT+"q12_plan_levels_forward.csv","w",newline="") as f:
    w=csv.DictWriter(f, fieldnames=list(recs[0].keys())); w.writeheader(); [w.writerow(x) for x in recs]
