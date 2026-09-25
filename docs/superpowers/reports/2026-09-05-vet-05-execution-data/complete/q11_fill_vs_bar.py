import sqlite3, csv, glob, re, datetime
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor(); c.execute('PRAGMA query_only=ON')
print("-- bars tf values:", c.execute("SELECT tf, COUNT(*), MIN(datetime(open_time_ms/1000,'unixepoch','-5 hours')), MAX(datetime(open_time_ms/1000,'unixepoch','-5 hours')) FROM bars WHERE symbol='MNQ' GROUP BY tf").fetchall())
era=1786770000000
rows=c.execute("""SELECT id, side, entry_price, entry_time, source, plan_band, cited_scenario_id, entry_order_id, pnl_corrected FROM trader_positions
 WHERE entry_time>=? AND entry_time<1788584400000 AND pnl_corrected IS NOT NULL AND upper(trim(coalesce(plan_id,'')))<>'UNRESOLVABLE' AND source<>'e7_farside_test' AND close_reason NOT IN ('reconcile_flat','unresolved','e7_farside_test') ORDER BY id""",(era,)).fetchall()
# NT8 log fill times by order name
ntfill={}
for f in glob.glob("/mnt/c/Users/hoang/Documents/NinjaTrader 8/log/log.2026*.en.txt"):
    try:
        for line in open(f, errors='ignore'):
            if "New state='Filled'" in line and "Name='" in line:
                m=re.search(r"^(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d):(\d+)\|.*Name='([^']+)' New state='Filled'.*?Type='([^']+)'", line)
                if m:
                    t=datetime.datetime.strptime(m.group(1),"%Y-%m-%d %H:%M:%S").replace(tzinfo=datetime.timezone(datetime.timedelta(hours=-5)))
                    ntfill.setdefault(m.group(3), int(t.timestamp()*1000)+int(m.group(2)))
    except Exception as e: print("skip",f,e)
print("-- NT8 log Filled names indexed:", len(ntfill))
# armed_orders signal ids by fill price
arms=c.execute("SELECT id, signal_id, fill_price, side, entry_px FROM armed_orders WHERE state='filled' AND signal_id<>''").fetchall()
out=[]; fill_times={}
for pid, side, entry, et, src, band, scen, eoid, pnl in rows:
    tfill=None; how=None
    if src=='system':
        r=c.execute("SELECT created_at, price FROM trader_fills WHERE side=? AND ABS(price-?)<0.01 AND ABS(created_at-?)<600000 ORDER BY ABS(created_at-?) LIMIT 1",
                    ('BUY' if side.upper()=='LONG' else 'SELL', entry, et, et)).fetchone()
        if r: tfill=r[0]; how='trader_fills'
        else: tfill=et; how='entry_time'
    else:
        cand=[a for a in arms if abs(a[2]-entry)<0.01 and a[3].upper()==side.upper()]
        for a in cand:
            if a[1] in ntfill and abs(ntfill[a[1]]-et)<15*60000: tfill=ntfill[a[1]]; how='nt8_log:'+a[1][:8]; break
        if tfill is None and eoid and eoid!='<nil>' and eoid in ntfill: tfill=ntfill[eoid]; how='nt8_log:'+eoid[:8]
        if tfill is None: tfill=et; how='entry_time(materialized,late)'
    fill_times[pid]=tfill
    b=c.execute("SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms<=? AND open_time_ms>?-60000 ORDER BY open_time_ms DESC LIMIT 1",(tfill,tfill)).fetchone()
    if not b: out.append((pid,side,entry,how,None,None,None,None,None,None,pnl)); continue
    ot,o,h,l,cl=b; rng=h-l
    pos=(entry-l)/rng if rng>0 else None
    at_hi = abs(entry-h)<0.126; at_lo=abs(entry-l)<0.126
    adverse = (side.upper()=='LONG' and at_hi) or (side.upper()=='SHORT' and at_lo)
    favorable = (side.upper()=='LONG' and at_lo) or (side.upper()=='SHORT' and at_hi)
    inside = l-0.001<=entry<=h+0.001
    out.append((pid,side,entry,how,datetime.datetime.utcfromtimestamp(ot/1000-5*3600).strftime('%m-%d %H:%M'),o,h,l,cl,round(pos,3) if pos is not None else None,pnl,inside,at_hi,at_lo,adverse,favorable))
w=csv.writer(open("q11_fill_vs_bar.csv","w"))
w.writerow(["pos_id","side","entry","fill_time_source","bar_open_ct","o","h","l","c","pos_in_range","pnl_corrected","inside_bar","at_high","at_low","adverse_extreme","favorable_extreme","fill_time_ms"])
for r in out: w.writerow(list(r)+[None]*(16-len(r))+[fill_times[r[0]]])
print('Eligible entry audits',len(out),'see CSV; no inference of order type from source')
