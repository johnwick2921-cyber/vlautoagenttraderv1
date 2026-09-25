import sqlite3, re, json
db = sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
c = db.cursor()
print("=== N1: planner_rejected_prompts classification by STATED definition ===")
rows = c.execute("SELECT id, reject_reason FROM planner_rejected_prompts ORDER BY id").fetchall()
print("total", len(rows))
void=[]; disp=[]; minclose=[]; mention=[]; other=[]
for i,r in rows:
    s=(r or '')
    sl=s.lower()
    if 'came back across' in sl or 'void' in sl: void.append(i)
    elif 'displacement' in sl and '0.00' in s: disp.append(i)
    elif 'no confirming close' in sl or 'bd_min_closes' in sl: minclose.append(i)
    elif 'breakdown_continue' in sl or 'breakup_continue' in sl: mention.append(i)
    else: other.append(i)
print("void:", len(void), void)
print("disp0:", len(disp), disp)
print("minclose:", len(minclose), minclose)
print("stated-def total:", len(void)+len(disp)+len(minclose))
print("mention-only (continuation named but not one of the three):", len(mention), mention)
for i,r in rows:
    if i in mention: print("   ", i, (r or '')[:160].replace('\n',' '))
print("other:", len(other))
# any-mention classifier
anym=[i for i,r in rows if ('breakdown_continue' in (r or '') or 'breakup_continue' in (r or ''))]
print("any-mention classifier:", len(anym))
# what the committed q07 classifier gives
print("\n=== reasons for the 'void' group (short) ===")
for i,r in rows:
    if i in void+disp+minclose: print("   ", i, (r or '')[:140].replace('\n',' '))

print("\n=== N2/C7: flap rows ids 62-102 ===")
q="""SELECT id, scenario, condition, entry_px, state, state_reason, placement_seq,
 datetime(strftime('%s',created_at),'unixepoch','-5 hours') ct
 FROM armed_orders WHERE id BETWEEN 62 AND 102 ORDER BY id"""
fl = c.execute(q).fetchall()
from collections import Counter
print("rows in 62-102:", len(fl))
print(Counter((r[3], r[1]) for r in fl))
s2 = [r for r in fl if r[3]==29591.02]
print("S2@29591.02 rows:", len(s2), "seq", sorted(r[6] for r in s2))
print(Counter(r[5] for r in s2))
print("ct range S2:", min(r[7] for r in s2), max(r[7] for r in s2))
print("all 62-102 reasons:", Counter(r[5] for r in fl))
for r in fl: print("  ", r)
print("\nid 38:", c.execute("SELECT id, scenario, condition, entry_px, state, state_reason, placement_seq, datetime(strftime('%s',created_at),'unixepoch','-5 hours'), datetime(strftime('%s',updated_at),'unixepoch','-5 hours') FROM armed_orders WHERE id=38").fetchall())

print("\n=== N3: stale window deaths all-time ===")
for r in c.execute("SELECT id, session, datetime(strftime('%s',created_at),'unixepoch','-5 hours'), datetime(strftime('%s',updated_at),'unixepoch','-5 hours'), state_reason FROM armed_orders WHERE state_reason LIKE '%stale window%' ORDER BY id"): print("  ", r)
print("\n=== N4: marketable guard all-time ===")
for r in c.execute("SELECT id, session, datetime(strftime('%s',created_at),'unixepoch','-5 hours'), datetime(strftime('%s',updated_at),'unixepoch','-5 hours'), state_reason FROM armed_orders WHERE state_reason LIKE '%marketable%' ORDER BY id"): print("  ", r)
print("\n=== boot_sweep all-time ===")
for r in c.execute("SELECT id, session, datetime(strftime('%s',created_at),'unixepoch','-5 hours'), state_reason FROM armed_orders WHERE state_reason LIKE '%boot_sweep%' ORDER BY id"): print("  ", r)
print("\n=== gate changed: rr all-time ===")
for r in c.execute("SELECT id, session, entry_px, datetime(strftime('%s',created_at),'unixepoch','-5 hours'), datetime(strftime('%s',updated_at),'unixepoch','-5 hours'), state_reason FROM armed_orders WHERE state_reason LIKE '%gate changed: rr%' ORDER BY id"): print("  ", r)

print("\n=== N5: open intents 09-03 CT ===")
for r in c.execute("""SELECT id, datetime(timestamp,'-5 hours'), substr(execution_log,1,90), substr(risk_check_error,1,60) FROM decision_records
 WHERE date(timestamp,'-5 hours')='2026-09-03' AND (decision_json LIKE '%open_long%' OR decision_json LIKE '%open_short%') ORDER BY id"""): print("  ", r)

print("\n=== N6: bias_label by created_at ===")
for r in c.execute("SELECT date(created_at), COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1"): print("  UTC", r)
for r in c.execute("SELECT date(created_at,'-5 hours'), COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1"): print("  CT", r)
for r in c.execute("SELECT trade_date, COUNT(*) FROM plans WHERE doc LIKE '%bias_label%' GROUP BY 1"): print("  trade_date", r)
print("  first:", c.execute("SELECT plan_id, version, trade_date, session, created_at FROM plans WHERE doc LIKE '%bias_label%' ORDER BY created_at LIMIT 1").fetchall())
print("  total plans:", c.execute("SELECT COUNT(*), COUNT(DISTINCT plan_id||'/'||version) FROM plans").fetchall())
