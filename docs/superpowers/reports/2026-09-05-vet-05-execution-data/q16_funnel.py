import sqlite3, json, math, datetime
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor()
def wilson(k,n,z=1.96):
    if n==0: return "n/a"
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return "%.1f%% [%.1f, %.1f]" % (100*p, 100*(ctr-h), 100*(ctr+h))
# authored
sv=0; arm_enabled=0; specs=set(); arm_specs=set(); by_cond={}
for pid,ver,doc,sess in c.execute("SELECT plan_id, version, doc, session FROM plans WHERE trade_date>='2026-09-02'"):
    try: d=json.loads(doc)
    except: continue
    for s in d.get('scenarios') or []:
        sv+=1; specs.add((pid,s.get('id')))
        a=s.get('arm')
        cond=s.get('condition'); by_cond.setdefault(cond,[0,0]); by_cond[cond][0]+=1
        if a and a.get('enabled'): arm_enabled+=1; arm_specs.add((pid,s.get('id'))); by_cond[cond][1]+=1
print("plans since 09-02:", c.execute("SELECT COUNT(*) FROM plans WHERE trade_date>='2026-09-02'").fetchone()[0], "versions;", c.execute("SELECT COUNT(DISTINCT plan_id) FROM plans WHERE trade_date>='2026-09-02'").fetchone()[0], "plan ids")
print("authored scenario-versions:",sv,"| distinct (plan,scenario):",len(specs))
print("authored with arm.enabled:",arm_enabled,"| distinct (plan,scenario) with arm:",len(arm_specs))
print("by condition (authored, armed):", by_cond)
# ledger
rows=c.execute("""SELECT id, plan_id, version, scenario, lower(side), entry_px, state, state_reason, signal_id, kind, leg_index,
  strftime('%s',created_at)*1000, strftime('%s',updated_at)*1000 FROM armed_orders WHERE datetime(strftime('%s',created_at),'unixepoch','-5 hours') >= '2026-09-02' ORDER BY id""").fetchall()
n=len(rows); placed=[r for r in rows if r[8]]; 
specs_led=set((r[1],r[3],r[10]) for r in rows); specs_placed=set((r[1],r[3],r[10]) for r in placed)
print("ledger rows:",n,"| distinct (plan,scenario,leg):",len(specs_led),"| placed rows (signal_id):",len(placed),"| distinct specs placed:",len(specs_placed))
print("placed by kind:", {k:sum(1 for r in placed if r[9]==k) for k in set(r[9] for r in placed)})
# reached: 1m bars between created and updated (for created_at tz: CT offset text -> strftime('%s') handles tz? check)
reached=[]; 
for r in rows:
    rid,pid,ver,sc,side,ent,state,reason,sig,kind,li,cms,ums=r
    # created_at carries a CT offset (-05:00) and updated_at +00:00; strftime('%s') respects the offset -> both UTC ms
    if ums<=cms: ums=cms+120000
    if kind=='stop_entry': hit=c.execute("SELECT MAX(h) FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms BETWEEN ?-60000 AND ?",(cms,ums)).fetchone()[0]; ok = hit is not None and hit>=ent-0.5
    elif side=='short': hit=c.execute("SELECT MAX(h) FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms BETWEEN ?-60000 AND ?",(cms,ums)).fetchone()[0]; ok = hit is not None and hit>=ent
    else: hit=c.execute("SELECT MIN(l) FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms BETWEEN ?-60000 AND ?",(cms,ums)).fetchone()[0]; ok = hit is not None and hit<=ent
    reached.append((rid,ok,hit))
rk={r[0]:r[1] for r in reached}
placed_reached=[r for r in placed if rk[r[0]]]
all_reached=[r for r in rows if rk[r[0]]]
filled=[r for r in rows if r[6]=='filled']
print("reached (price touched entry while the row lived): all rows",len(all_reached),"/",n,"| placed rows",len(placed_reached),"/",len(placed))
print("filled:",len(filled), [r[0] for r in filled])
# won
won=0
for r in filled:
    p=c.execute("SELECT id, pnl_corrected FROM trader_positions WHERE plan_band='armed_fill' AND entry_time>=1788325200000 AND ABS(entry_price-?)<0.01",(r[5],)).fetchone()
    print("   filled arm",r[0],"-> position",p); 
    if p and p[1] and p[1]>0: won+=1
print("won:",won)
# terminal reasons for unplaced vs placed
import collections
print("-- never placed (no signal): reasons", collections.Counter(r[7][:45] for r in rows if not r[8]))
print("-- placed: terminal reasons", collections.Counter((r[6]+':'+r[7][:45]) for r in placed))
print("-- reached-but-not-filled rows:", [(r[0],r[3],r[9],r[6],r[7][:40]) for r in all_reached if r[6]!='filled'])
print("-- marketable-guard cancels since 09-02:", [(r[0],r[3],r[5]) for r in rows if 'marketable' in r[7]])
# rates
print("RATES: armed→placed", wilson(len(specs_placed),len(specs_led)), "(distinct specs); rows", wilson(len(placed),n))
print("       placed→reached", wilson(len(placed_reached),len(placed)))
print("       reached→filled (placed)", wilson(len([r for r in placed_reached if r[6]=='filled']),len(placed_reached)))
print("       filled→won", wilson(won,len(filled)))
print("       authored-with-arm(distinct)→ledger spec", wilson(len(specs_led),len(arm_specs)))
# marketable guard all-time: how far through was price at the cancel?
print("-- marketable guard all-time rows with distance (last 1m close at cancel vs entry)")
for rid,sc,side,ent,cms,ums,reason in c.execute("SELECT id, scenario, lower(side), entry_px, strftime('%s',created_at)*1000, strftime('%s',updated_at)*1000, state_reason FROM armed_orders WHERE state_reason LIKE '%marketable%' ORDER BY id"):
    b=c.execute("SELECT c FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms<=? ORDER BY open_time_ms DESC LIMIT 1",(ums,)).fetchone()
    px=b[0] if b else None
    dist=(px-ent) if (px and side=='short') else ((ent-px) if px else None)
    print("   ",rid,sc,side,ent,"close@cancel",px,"through by",round(dist,2) if dist is not None else None,"pts")
