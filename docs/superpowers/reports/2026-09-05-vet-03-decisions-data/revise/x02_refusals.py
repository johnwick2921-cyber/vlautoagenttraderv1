import csv,collections,sqlite3
P="/home/hoang/nofx-vet-03/docs/superpowers/reports/2026-09-04-two-day-audit-data/refusals.csv"
rows=list(csv.DictReader(open(P)))
print("rows",len(rows))
by=collections.defaultdict(list)
for r in rows: by[r['leg']].append(r)
tot=0
for leg,rs in sorted(by.items()):
    s=sum(float(x['cf_usd'] or 0) for x in rs); tot+=s
    print(f"{leg}: n={len(rs)} cf_session_flat={s:+.2f}")
print("TOTAL",f"{tot:+.2f}")
# min_sl validateDecision: did the same cycle end in a taken open_*?
db=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
ms=[r for r in rows if r['leg'].startswith('min_sl (validateDecision')]
print("\nmin_sl validateDecision n=",len(ms))
taken=[];nott=[]
for r in ms:
    cyc=r['decision_cycle']
    if not cyc: nott.append(r); continue
    q=db.execute("SELECT id,execution_status,execution_log FROM decision_records WHERE cycle_number=? AND trader_id LIKE '8d5c8af5%'",(cyc,)).fetchall()
    ok=any('succeeded' in (x[2] or '') and ('open_long' in (x[2] or '') or 'open_short' in (x[2] or '')) for x in q)
    (taken if ok else nott).append(r)
print("  refusal whose own cycle went on to open a position:",len(taken),[(r['ts_ct'],r['decision_cycle'],r['cf_usd']) for r in taken])
print("  refusal after which no position opened that cycle:",len(nott))
print("  cf sum taken:",sum(float(r['cf_usd'] or 0) for r in taken))
print("  cf sum not-taken:",sum(float(r['cf_usd'] or 0) for r in nott))
w=sorted(nott,key=lambda r: float(r['cf_usd'] or 0))
print("  biggest saved loss among not-taken:",[(r['ts_ct'],r['session'],r['cf_usd'],r['entry_px'],r['stop_px'],r['cf_outcome'],r['cf_time_ct'],r['decision_cycle']) for r in w[:3]])
print("  biggest missed win among not-taken:",[(r['ts_ct'],r['session'],r['cf_usd'],r['cf_outcome']) for r in w[-3:]])
for cyc in ('26472','26500'):
    print("  positions from cycle",cyc,db.execute("SELECT id,entry_price,exit_price,side,pnl_corrected,datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE id IN (589,590)").fetchall() if cyc=='26472' else '')
print(db.execute("SELECT id,entry_price,exit_price,side,pnl_corrected,round(abs(entry_price-exit_price),2),datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE id IN (589,590)").fetchall())
# dedup ledger
legs={'min_sl (validateDecision A3) 31 not-taken':sum(float(r['cf_usd'] or 0) for r in nott),
      'min_sl (EntryGate leg 6)':sum(float(r['cf_usd'] or 0) for r in by['min_sl (EntryGate leg 6)']),
      'rr_at_fill (EntryGate leg 5)':sum(float(r['cf_usd'] or 0) for r in by['rr_at_fill (EntryGate leg 5)']),
      'rr (armGateVerdictFor)':sum(float(r['cf_usd'] or 0) for r in by['rr (armGateVerdictFor)']),
      'strict deduped to one position':-43.50,
      'last_entry_cutoff':sum(float(r['cf_usd'] or 0) for r in by['last_entry_cutoff (P2.3)']),
      'leg 3 invalidation (09-04, mine)':428.00}
print("\nDEDUPED REFUSED-SET LEDGER")
t=0
for k,v in legs.items(): t+=v; print(f"  {k}: {v:+.2f}")
print(f"  TOTAL forgone: {t:+.2f}")
print("  with all 34 min-SL rows instead of 31:", f"{t - legs['min_sl (validateDecision A3) 31 not-taken'] + sum(float(r['cf_usd'] or 0) for r in ms):+.2f}")
