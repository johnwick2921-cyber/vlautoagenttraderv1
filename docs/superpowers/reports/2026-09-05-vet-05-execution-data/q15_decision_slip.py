import sqlite3, json, re, datetime, math
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor()
def pct(xs,q):
    xs=sorted(xs); 
    if not xs: return None
    i=(len(xs)-1)*q; lo=int(i); hi=min(lo+1,len(xs)-1); return round(xs[lo]+(xs[hi]-xs[lo])*(i-lo),2)
era=1786770000000
rows=c.execute("SELECT id, side, entry_price, entry_time, pnl_corrected FROM trader_positions WHERE entry_time>=? AND source='system' AND pnl_corrected IS NOT NULL ORDER BY id",(era,)).fetchall()
out=[]
for pid, side, fill, et, pnl in rows:
    # decision records within [-4 min, +30 s] of the entry time (timestamp text is UTC)
    lo=datetime.datetime.fromtimestamp(et/1000-240,datetime.timezone.utc).strftime('%Y-%m-%d %H:%M:%S')
    hi=datetime.datetime.fromtimestamp(et/1000+30,datetime.timezone.utc).strftime('%Y-%m-%d %H:%M:%S')
    best=None
    for did, ts, dj, el in c.execute("SELECT id, timestamp, decision_json, execution_log FROM decision_records WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp DESC",(lo,hi)):
        try: d=json.loads(dj) if dj else None
        except: d=None
        cands=[]
        if isinstance(d,list): cands=d
        elif isinstance(d,dict): cands=[d] + (d.get('decisions') or [])
        for x in cands:
            if not isinstance(x,dict): continue
            act=str(x.get('action','')).lower()
            if act in ('open_long','open_short','buy','sell') and x.get('entry') or x.get('entry_price'):
                e=float(x.get('entry') or x.get('entry_price') or 0)
                if e>0: best=(did, ts, act, e); break
        if best: break
    if best:
        did,ts,act,e=best
        raw=(fill-e)/0.25
        adverse = raw if side.upper()=='LONG' else -raw   # + = paid more (long) / got less (short)
        out.append((pid,side,fill,e,round(raw,1),round(adverse,1),did,ts,pnl))
    else:
        out.append((pid,side,fill,None,None,None,None,None,pnl))
m=[r for r in out if r[3] is not None]
print("system rows:",len(rows),"matched to a decision entry price:",len(m))
adv=[r[5] for r in m]
print("adverse slippage ticks vs decision price: p50=%s p80=%s p95=%s mean=%.2f | favorable (neg) count=%d, zero=%d, adverse=%d" % (pct(adv,.5),pct(adv,.8),pct(adv,.95),sum(adv)/len(adv),sum(1 for a in adv if a<0),sum(1 for a in adv if a==0),sum(1 for a in adv if a>0)))
print("in points: p50=%.2f mean=%.2f ; in $ per contract (x$0.50/tick): mean=$%.2f" % (pct(adv,.5)/4, sum(adv)/len(adv)/4, sum(adv)/len(adv)*0.5))
print("rows (id, side, fill, decision_entry, raw_ticks, adverse_ticks, decision_id, ts_utc, pnl):")
for r in m: print("  ",r)
print("unmatched ids:",[r[0] for r in out if r[3] is None])
