# q06b: trigger-fired with each version's IN-FORCE window [created, min(next version created, session end)], all versions; plus FIRST version per session-day (window to session end)
import sqlite3, json, collections, datetime, bisect, math, csv
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
rows=con.execute("SELECT plan_id, version, trade_date, session, lifecycle, trigger_reason, doc, created_at FROM plans WHERE session IN ('ASIA','LONDON','NY') ORDER BY trade_date, session, version").fetchall()
CT=datetime.timezone(datetime.timedelta(hours=-5))
def parse_ts(s):
    s=s.strip()
    if s.endswith('+00:00'): s=s[:-6]
    if '.' in s: s=s.split('.')[0]
    return datetime.datetime.strptime(s.replace('T',' '), '%Y-%m-%d %H:%M:%S').replace(tzinfo=datetime.timezone.utc)
SESS_END_CT={'ASIA':(1,30,1),'LONDON':(8,30,0),'NY':(15,0,0)}
bars=con.execute("SELECT open_time_ms, h, l FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
times=[b[0] for b in bars]
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; den=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n)); return (round((c-h)/den,3),round((c+h)/den,3))
groups=collections.defaultdict(list)
for r in rows: groups[(r[2],r[3])].append(r)
def run(mode):
    fired=collections.Counter(); bycond=collections.defaultdict(collections.Counter); noref=collections.Counter(); nobars=0; out=[]
    for (td,sess),vs in sorted(groups.items()):
        y,m,dd=map(int,td.split('-')); hh,mm,plus=SESS_END_CT[sess]
        end=datetime.datetime(y,m,dd,hh,mm,tzinfo=CT)+datetime.timedelta(days=plus)
        vs=sorted(vs,key=lambda r:r[1])
        sel = vs if mode=='inforce' else vs[:1]
        for i,r in enumerate(sel):
            try: d=json.loads(r[6])
            except Exception: continue
            t0=parse_ts(r[7])
            t1=end
            if mode=='inforce' and i+1<len(vs): t1=min(end, parse_ts(vs[i+1][7]))
            t0ms=int(t0.timestamp()*1000); t1ms=int(t1.timestamp()*1000)
            i0=bisect.bisect_left(times,t0ms); i1=bisect.bisect_right(times,t1ms)
            seg=bars[i0:i1]
            for s in d.get('scenarios') or []:
                c=s.get('confirm') or {}; ref=c.get('ref_price')
                if not ref: noref[s.get('condition')]+=1; continue
                ref=float(ref)
                if not seg: nobars+=1; continue
                hit=next((b[0] for b in seg if b[2]<=ref<=b[1]), None)
                st='FIRED' if hit is not None else 'NOT_FIRED'
                fired[st]+=1; bycond[s.get('condition')][st]+=1
                out.append((td,sess,r[1],s.get('id'),s.get('condition'),s.get('direction'),ref,st, datetime.datetime.fromtimestamp(hit/1000,CT).strftime('%m-%d %H:%M') if hit else '', t0.astimezone(CT).strftime('%m-%d %H:%M'), t1.astimezone(CT).strftime('%m-%d %H:%M'), round((t1ms-t0ms)/60000)))
    n=sum(fired.values()); k=fired['FIRED']
    print(f'## mode={mode}: fired {k}/{n} = {k/n:.3f} Wilson {wilson(k,n)} · no ref_price: {sum(noref.values())} {dict(noref)} · no bars: {nobars}')
    for c,cc in sorted(bycond.items(), key=lambda x:-sum(x[1].values())):
        t=sum(cc.values()); print(f'   {c}: {cc["FIRED"]}/{t} Wilson {wilson(cc["FIRED"],t)}')
    with open(f'/home/hoang/nofx-analysis/vet-03-0905/q06b_trigger_{mode}.csv','w',newline='') as f:
        w=csv.writer(f); w.writerow(['trade_date','session','version','scenario','condition','direction','ref_price','fired','first_touch_ct','window_start_ct','window_end_ct','window_min']); w.writerows(out)
    return out
o=run('inforce'); run('first')
# median in-force window length
import statistics
w=[x[11] for x in o]; print('in-force window minutes: median', statistics.median(w), 'p25', sorted(w)[len(w)//4], 'p75', sorted(w)[3*len(w)//4], 'n', len(w))
