import sqlite3, json, re, datetime
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor()
def pct(xs,q):
    xs=sorted(xs)
    if not xs: return None
    i=(len(xs)-1)*q; lo=int(i); hi=min(lo+1,len(xs)-1); return round(xs[lo]+(xs[hi]-xs[lo])*(i-lo),2)
era=1786770000000
rows=c.execute("SELECT id, side, entry_price, entry_time, pnl_corrected FROM trader_positions WHERE entry_time>=? AND source='system' AND pnl_corrected IS NOT NULL ORDER BY id",(era,)).fetchall()
out=[]
for pid, side, fill, et, pnl in rows:
    lo=datetime.datetime.fromtimestamp(et/1000-300,datetime.timezone.utc).strftime('%Y-%m-%d %H:%M:%S')
    hi=datetime.datetime.fromtimestamp(et/1000+60,datetime.timezone.utc).strftime('%Y-%m-%d %H:%M:%S')
    rec=None
    for did, ts, el in c.execute("SELECT id, timestamp, execution_log FROM decision_records WHERE timestamp BETWEEN ? AND ? AND (execution_log LIKE '%open_long succeeded%' OR execution_log LIKE '%open_short succeeded%') ORDER BY timestamp DESC LIMIT 1",(lo,hi)):
        rec=(did,ts,el)
    if not rec: out.append((pid,side,fill,None,None,None,None,None,pnl)); continue
    did,ts,el=rec
    m=re.search(r'AI call duration: (\d+) ms', el or ''); dur=int(m.group(1)) if m else None
    tsms=int(datetime.datetime.fromisoformat(ts.replace('+00:00','')[:26]).replace(tzinfo=datetime.timezone.utc).timestamp()*1000)
    start=tsms-(dur or 0)
    b=c.execute("SELECT c FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms<=? ORDER BY open_time_ms DESC LIMIT 1",(start,)).fetchone()
    if not b: out.append((pid,side,fill,did,dur,None,None,None,pnl)); continue
    p0=b[0]; raw=(fill-p0)/0.25; adverse= raw if side.upper()=='LONG' else -raw
    out.append((pid,side,fill,did,dur,p0,round(raw,1),round(adverse,1),pnl))
m=[r for r in out if r[7] is not None]
adv=[r[7] for r in m]; durs=[r[4] for r in m if r[4]]
print("system entries:",len(rows),"with cycle-start price:",len(m))
print("AI call duration ms: p50=%s p80=%s p95=%s max=%s" % (pct(durs,.5),pct(durs,.8),pct(durs,.95),max(durs)))
print("adverse drift (fill vs 1m close at cycle start), ticks: p50=%s p80=%s p95=%s mean=%.2f | favorable=%d zero=%d adverse=%d" % (pct(adv,.5),pct(adv,.8),pct(adv,.95),sum(adv)/len(adv),sum(1 for a in adv if a<0),sum(1 for a in adv if a==0),sum(1 for a in adv if a>0)))
print("in points: mean %.2f, p80 %.2f; $/contract mean $%.2f" % (sum(adv)/len(adv)/4, pct(adv,.8)/4, sum(adv)/len(adv)*0.5))
print("rows (id, side, fill, decision_id, ai_ms, p_cycle_start, raw_ticks, adverse_ticks, pnl):")
for r in out: print("  ",r)
