# q07: re-author cost — for each rejected read cluster (trade_date, session, attempts within 20 min), seconds from first reject to the next accepted plans row; attempts per accepted read; fail-closed reads
import sqlite3, datetime, collections, statistics
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
def p(s):
    s=s.strip()
    if s.endswith('+00:00'): s=s[:-6]
    if s.endswith('-05:00'): 
        s=s[:-6]; base=datetime.datetime.strptime(s.split('.')[0].replace('T',' '),'%Y-%m-%d %H:%M:%S'); return base+datetime.timedelta(hours=5)
    return datetime.datetime.strptime(s.split('.')[0].replace('T',' '),'%Y-%m-%d %H:%M:%S')
rej=con.execute("SELECT trade_date, session, attempt, reject_reason, created_at FROM planner_rejected_prompts ORDER BY created_at").fetchall()
plans=con.execute("SELECT trade_date, session, version, trigger_reason, lifecycle, created_at FROM plans WHERE session IN ('ASIA','LONDON','NY') ORDER BY created_at").fetchall()
# cluster rejects by (trade_date, session) with gap <= 20 min
clusters=[]
for r in rej:
    t=p(r[4])
    if clusters and clusters[-1]['key']==(r[0],r[1]) and (t-clusters[-1]['last']).total_seconds()<=1200:
        clusters[-1]['rows'].append(r); clusters[-1]['last']=t
    else:
        clusters.append({'key':(r[0],r[1]),'first':t,'last':t,'rows':[r]})
print('## reject clusters (reads with ≥1 rejected attempt):', len(clusters))
costs=[]; fc=0
for c in clusters:
    nxt=[pl for pl in plans if (pl[0],pl[1])==c['key'] and p(pl[5])>=c['first']]
    if nxt:
        pl=nxt[0]; dt=(p(pl[5])-c['first']).total_seconds()
        fcd = pl[3]=='planner_fail_closed'
        fc+=fcd
        costs.append(dt)
        print(f"  {c['key']} first_reject_utc={c['first']} attempts={len(c['rows'])} -> next plan v{pl[2]} {pl[3]}/{pl[4]} after {dt:.0f}s  reasons={[x[3][:40] for x in c['rows']]}")
    else:
        print(f"  {c['key']} first_reject_utc={c['first']} attempts={len(c['rows'])} -> NO subsequent plan row")
print(f'## re-author cost seconds (first reject → next plan row): n={len(costs)} median={statistics.median(costs):.0f} min={min(costs):.0f} max={max(costs):.0f}; next row was planner_fail_closed in {fc}/{len(costs)}')
# planner_fail_closed rows overall + by session
print('## planner_fail_closed plans by date/session:', con.execute("SELECT trade_date||' '||session, COUNT(*) FROM plans WHERE trigger_reason='planner_fail_closed' GROUP BY 1").fetchall())
# accepted reads vs rejected attempts: reads = distinct (trade_date,session,version); attempts = reads + rejects
reads=len(plans); print(f'## plan rows (ASIA/LONDON/NY) = {reads}; rejected attempts = {len(rej)}; reject share of all authoring attempts = {len(rej)/(reads+len(rej)):.3f}')
# reject rate by rule family
fam=collections.Counter()
for r in rej:
    s=r[3]
    if 'breakdown_continue' in s or 'breakup_continue' in s: f='continuation law (void/displacement/BD_MIN_CLOSES)'
    elif 'fade_requires_touch' in s: f='fade_requires_touch'
    elif 'not allowed for' in s or 'invalid (touch' in s: f='confirm-rule vocabulary'
    elif 'entry_mode=pullback' in s: f='arm requires entry_mode=pullback (09-04 rule)'
    elif 'arm on' in s or 'arm legs' in s: f='arm legs contract'
    elif 'gap-up' in s: f='gap-up trigger law'
    elif '503' in s or 'stream interrupted' in s or 'EOF' in s: f='TRANSPORT (503/stream/EOF)'
    elif 'too many levels' in s: f='level cap'
    elif 'unreachable' in s: f='retest distance'
    else: f='other: '+s[:40]
    fam[f]+=1
print('## rejects by rule family:'); [print('  ',k,v) for k,v in fam.most_common()]
