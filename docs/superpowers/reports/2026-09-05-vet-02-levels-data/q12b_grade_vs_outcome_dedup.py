#!/usr/bin/env python3
"""q12b: same as q12 but ONE row per (session-day, label, price): the EARLIEST plan version that seated it starts the scan.
Removes the re-plan duplication (249 versions over ~51 plan-sessions)."""
import sqlite3, json, math, collections, datetime, csv
exec(open('/home/hoang/nofx-analysis/vet-02-0905/q11_replay.py').read().split("# group bars by session day")[0])
by_sd=collections.OrderedDict()
for b in bars: by_sd.setdefault(sess_day(b['t']),[]).append(b)
sdays=sorted(by_sd)
def delta_for(day):
    idx=sdays.index(day); trail=[b for dd in sdays[max(0,idx-5):idx] for b in by_sd[dd]]
    if len(trail)<2: trail=by_sd[day]
    return sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/max(1,len(trail)-1)
def fam(label):
    l=label or ''
    for pre,f in (('VWAP','VWAP*'),('SWG-H','SWG-H'),('SWG-L','SWG-L'),('OB(','OB'),('Demand','DEMAND'),('Supply','SUPPLY'),('FVG','FVG'),('iFVG','IFVG'),('EQH','EQH'),('EQL','EQL'),('nPOC','nPOC'),('RN ','RN'),('IB','IB*')):
        if l.startswith(pre): return f
    return l
def utc_ms(s):
    s=s.replace('+00:00','').split('.')[0]; return int(datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').replace(tzinfo=datetime.timezone.utc).timestamp()*1000)
con2=sqlite3.connect(DB, uri=True); con2.row_factory=sqlite3.Row
first={}  # (day, label, px) -> earliest record
psess=set()
for r in con2.execute("SELECT plan_id, version, session, trade_date, created_at, doc FROM plans WHERE session<>'WEEKLY' ORDER BY created_at"):
    t0=utc_ms(r['created_at']); day=sess_day(t0); psess.add((r['trade_date'],r['session']))
    if day not in by_sd: continue
    try: doc=json.loads(r['doc'])
    except Exception: continue
    for l in doc.get('levels') or []:
        try: px=round(float(l.get('price')),2)
        except Exception: continue
        k=(day,l.get('label'),px)
        if k not in first: first[k]=dict(t0=t0,session=r['session'],day=r['trade_date'],label=l.get('label'),fam=fam(l.get('label')),grade=l.get('grade'),mgrade=l.get('machine_grade') or '',instr=(l.get('instruction') or '').lower(),px=px)
recs=[]
for k,x in first.items():
    day=k[0]; tape=[b for b in by_sd[day] if b['t']>=x['t0']]
    if len(tape)<15: continue
    eps=detect(tape,x['px'],3.0,delta_for(day),12,'close'); outs=[e['outcome'] for e in eps]
    x.update(first=(eps[0]['outcome'] if eps else 'untouched'),n_eps=len(eps),hold=outs.count('hold'),brk=outs.count('break'),amb=len(eps)-outs.count('hold')-outs.count('break'),dist=abs(x['px']-tape[0]['c']),mfe1=(eps[0]['mfe'] if eps else 0),mae1=(eps[0]['mae'] if eps else 0)); recs.append(x)
def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (cen-half,cen+half)
def tab(name,keyf,rows=recs,minn=1):
    g=collections.defaultdict(lambda: dict(n=0,touched=0,f_hold=0,f_brk=0,f_amb=0,hold=0,brk=0,amb=0,dist=0.0,days=set()))
    for x in rows:
        a=g[keyf(x)]; a['n']+=1; a['dist']+=x['dist']; a['days'].add(x['day'])
        if x['first']!='untouched': a['touched']+=1
        if x['first']=='hold': a['f_hold']+=1
        elif x['first']=='break': a['f_brk']+=1
        elif x['first']!='untouched': a['f_amb']+=1
        a['hold']+=x['hold']; a['brk']+=x['brk']; a['amb']+=x['amb']
    out=[f"\n== {name}", f"{'key':22s} {'seats':>5s} {'days':>4s} {'touch%':>6s} {'1st:h':>5s} {'1st:b':>5s} {'1st:a':>5s} {'p_hold1':>7s} {'wilson95':>16s} | {'all:h':>5s} {'all:b':>5s} {'p_all':>6s} {'dist':>6s}"]
    for k,a in sorted(g.items(), key=lambda kv:-kv[1]['n']):
        if a['n']<minn: continue
        d1=a['f_hold']+a['f_brk']; p1=a['f_hold']/d1 if d1 else float('nan'); lo,hi=wilson(a['f_hold'],d1)
        da=a['hold']+a['brk']; pa=a['hold']/da if da else float('nan')
        out.append(f"{str(k)[:22]:22s} {a['n']:5d} {len(a['days']):4d} {100*a['touched']/a['n']:6.1f} {a['f_hold']:5d} {a['f_brk']:5d} {a['f_amb']:5d} {p1:7.3f} [{lo:.3f},{hi:.3f}] | {a['hold']:5d} {a['brk']:5d} {pa:6.3f} {a['dist']/a['n']:6.1f}")
    return "\n".join(out)
rep=[f"DEDUPED seated levels (one per session-day x label x price): {len(recs)} from {len(psess)} plan-sessions"]
rep.append(tab("by AI grade", lambda x:x['grade']))
rep.append(tab("by machine_grade", lambda x:x['mgrade'] or '(none)'))
rep.append(tab("by kind family", lambda x:x['fam'], minn=3))
rep.append(tab("by kind family x grade (n>=6)", lambda x:(x['fam'],x['grade']), minn=6))
rep.append(tab("by session", lambda x:x['session']))
rep.append(tab("by distance bucket", lambda x: '<25' if x['dist']<25 else '<50' if x['dist']<50 else '<100' if x['dist']<100 else '<200' if x['dist']<200 else '200+'))
rep.append(tab("kind family x session (n>=5)", lambda x:(x['fam'],x['session']), minn=5))
txt="\n".join(rep); print(txt); open(OUT+"q12b_grade_vs_outcome_dedup.txt","w").write(txt)
with open(OUT+"q12b_plan_levels_forward_dedup.csv","w",newline="") as f:
    w=csv.DictWriter(f, fieldnames=list(recs[0].keys())); w.writeheader(); [w.writerow(x) for x in recs]
