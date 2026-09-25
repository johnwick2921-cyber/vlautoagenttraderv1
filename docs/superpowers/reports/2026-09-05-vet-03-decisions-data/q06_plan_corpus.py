# q06: full-corpus plan parse + trigger-fired join (plans ⋈ bars)
# Corpus: LATEST version per (trade_date, session) to avoid version double-counting; also ALL versions reported.
import sqlite3, json, collections, statistics, datetime, csv, sys
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
rows=con.execute("SELECT plan_id, version, trade_date, session, lifecycle, trigger_reason, doc, created_at FROM plans WHERE session IN ('ASIA','LONDON','NY') ORDER BY trade_date, session, version").fetchall()
def parse_ts(s):
    # plans.created_at is UTC RFC3339 with nanos and +00:00
    s=s.strip()
    if s.endswith('+00:00'): s=s[:-6]
    if '.' in s: s=s.split('.')[0]
    return datetime.datetime.strptime(s.replace('T',' '), '%Y-%m-%d %H:%M:%S').replace(tzinfo=datetime.timezone.utc)
# session end (CT) per session: ASIA read 16:30 → ends 01:30 next day; LONDON 01:30→08:30; NY 08:00→14:45 flat (use 15:00)
SESS_END_CT={'ASIA':(1,30,1),'LONDON':(8,30,0),'NY':(15,0,0)}
CT=datetime.timezone(datetime.timedelta(hours=-5))
latest={}
for r in rows: latest[(r[2],r[3])]=r
allv=rows
def stats(rowset, label):
    n_plans=len(rowset); sc_n=[]; cond=collections.Counter(); dirc=collections.Counter(); qual=collections.Counter()
    arm_by_dir=collections.Counter(); arm_en=collections.Counter(); bias_dir=collections.Counter(); ai_tree=collections.Counter(); rr=[]
    day_type=collections.Counter(); armcond=collections.Counter()
    for r in rowset:
        try: d=json.loads(r[6])
        except Exception: continue
        scs=d.get('scenarios') or []
        sc_n.append(len(scs))
        b=d.get('bias') or {}
        bias_dir[b.get('direction')]+=1
        day_type[d.get('day_type')]+=1
        bl=d.get('bias_label') or ''
        ai=tree=None
        if 'AI ' in bl and 'tree ' in bl:
            try:
                parts=bl.split('·'); ai=parts[0].split('AI')[-1].strip(); tree=parts[1].split('tree')[-1].strip()
            except Exception: pass
        ai_tree[(ai,tree)]+=1
        for s in scs:
            cond[s.get('condition')]+=1; dirc[s.get('direction')]+=1; qual[s.get('quality')]+=1
            a=s.get('arm') or {}
            en=bool(a.get('enabled'))
            arm_en[en]+=1
            arm_by_dir[(s.get('direction'),en)]+=1
            if en: armcond[s.get('condition')]+=1
            if en and a.get('entry') and a.get('stop') and a.get('target'):
                e,st,t=float(a['entry']),float(a['stop']),float(a['target'])
                if s.get('direction')=='long' and e>st: rr.append((t-e)/(e-st))
                elif s.get('direction')=='short' and st>e: rr.append((e-t)/(st-e))
    print(f'## {label}: plans={n_plans} scenarios={sum(sc_n)} sc/plan dist={collections.Counter(sc_n)}')
    print('  condition:', cond.most_common()); print('  direction:', dirc.most_common()); print('  quality:', qual.most_common())
    print('  arm_enabled:', dict(arm_en), ' by dir:', dict(arm_by_dir), ' arm cond:', armcond.most_common())
    print('  bias.direction:', bias_dir.most_common(), ' day_type:', day_type.most_common())
    print('  bias_label (AI,tree):', ai_tree.most_common())
    if rr: print(f'  planned R:R (arm-enabled, n={len(rr)}): median={statistics.median(rr):.2f} min={min(rr):.2f} max={max(rr):.2f}')
stats(list(latest.values()), 'LATEST version per session-day')
stats(allv, 'ALL versions')
# ---- trigger-fired join (latest version only): a scenario's trigger "fires" if any 1m bar after plan creation and before session end has low<=ref<=high
bars=con.execute("SELECT open_time_ms, h, l FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
if not bars:
    print('no 1m MNQ bars; tf values:', con.execute("SELECT symbol, tf, COUNT(*) FROM bars GROUP BY 1,2").fetchall()); sys.exit()
import bisect
times=[b[0] for b in bars]
out=[]; fired=collections.Counter(); fired_by_cond=collections.Counter(); tot_by_cond=collections.Counter(); nobars=0
for (td,sess),r in sorted(latest.items()):
    try: d=json.loads(r[6])
    except Exception: continue
    t0=parse_ts(r[7])
    y,m,dd=map(int,td.split('-'))
    hh,mm,plus=SESS_END_CT[sess]
    end=datetime.datetime(y,m,dd,hh,mm,tzinfo=CT)+datetime.timedelta(days=plus)
    t0ms=int(t0.timestamp()*1000); t1ms=int(end.timestamp()*1000)
    i0=bisect.bisect_left(times,t0ms); i1=bisect.bisect_right(times,t1ms)
    seg=bars[i0:i1]
    for s in d.get('scenarios') or []:
        c=s.get('confirm') or {}; ref=c.get('ref_price')
        if not ref: continue
        ref=float(ref)
        if not seg:
            nobars+=1; out.append((td,sess,r[1],s.get('id'),s.get('condition'),s.get('direction'),ref,'NO_BARS','' )); continue
        hit=None
        for b in seg:
            if b[2]<=ref<=b[1]: hit=b[0]; break
        tot_by_cond[s.get('condition')]+=1
        if hit is not None:
            fired['fired']+=1; fired_by_cond[s.get('condition')]+=1
            out.append((td,sess,r[1],s.get('id'),s.get('condition'),s.get('direction'),ref,'FIRED',datetime.datetime.fromtimestamp(hit/1000,CT).strftime('%m-%d %H:%M')))
        else:
            fired['not_fired']+=1
            out.append((td,sess,r[1],s.get('id'),s.get('condition'),s.get('direction'),ref,'NOT_FIRED',''))
n=sum(fired.values())
import math
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; den=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n)); return ((c-h)/den,(c+h)/den)
k=fired['fired']
print(f'## trigger-fired (LATEST version, confirm.ref_price touched by a 1m bar between plan creation and session end): {k}/{n} = {k/n:.3f} Wilson {wilson(k,n)}  (scenarios w/o bars coverage: {nobars})')
for c in tot_by_cond: print(f'   {c}: {fired_by_cond[c]}/{tot_by_cond[c]}  Wilson {tuple(round(x,3) for x in wilson(fired_by_cond[c],tot_by_cond[c]))}')
with open('/home/hoang/nofx-analysis/vet-03-0905/q06_trigger_fired.csv','w',newline='') as f:
    w=csv.writer(f); w.writerow(['trade_date','session','version','scenario','condition','direction','ref_price','fired','first_touch_ct']); w.writerows(out)
print('bars coverage:', datetime.datetime.fromtimestamp(times[0]/1000,CT), '→', datetime.datetime.fromtimestamp(times[-1]/1000,CT), 'n', len(times))
