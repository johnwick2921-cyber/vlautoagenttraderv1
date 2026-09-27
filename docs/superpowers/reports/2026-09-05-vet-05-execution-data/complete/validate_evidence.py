from pathlib import Path
import ast,json,csv,sqlite3,math
x=json.load(open('q31_verified.json'));y=json.load(open('q35_complete.json'));z=json.load(open('q34_integration.json'))
expected=set(range(521,592))-{530,539,545,546,566,571,580,572,573,574,576,577,579}
assert set(x['performance']['ids'])==expected and len(expected)==58
assert len(x['performance']['winner_ids'])==18 and len(x['performance']['loser_ids'])==38 and len(x['performance']['scratch_ids'])==2
assert math.isclose(x['performance']['sum'],-466.428572,abs_tol=1e-8)
assert z['cme_days']==12 and z['calendar_days']==12 and not z['positions_after_cut']
assert z['slippage_pts_vs_authorized_broker_stop']==0
assert len(x['exit_reasons']['sl']['ids'])==40 and len(x['exit_reasons']['tp']['ids'])==11
assert sum(len(r['ids']) for r in x['exit_reasons'].values())==58
assert len(x['floor_winners']['ids'])==10 and x['floor_winners']['exceeded']['k']==0
assert len(list(csv.DictReader(open('eligible_fill_audit.csv'))))==116
for leg in ['entry','exit']:
 g=[r for r in x['fill_rows'] if r['leg']==leg];assert len(g)==58 and sum(bool(r['bar']) for r in g)==55
 assert x['fill_summary'][leg]['inside']['rate']['k']==54
f=y['funnel'];assert [len(f[k]) for k in ['enabled_keys','armed_keys','placed_keys','reached_proxy_keys','filled_keys']]==[22,7,4,2,1]
assert len(f['stop_entry_sent_ids'])==21
assert y['arm35_geometry']['max_adverse_entry_pts_at_2R']<.25
assert y['arm35_geometry']['rr_one_adverse_tick']<2
# Independent raw SQL using an explicit whitelist, not the application or q31 function.
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.execute('pragma query_only=ON')
r=c.execute('select count(*),sum(pnl_corrected),sum(pnl_corrected>0),sum(pnl_corrected<0),sum(pnl_corrected=0) from trader_positions where id in ('+','.join('?' for _ in expected)+')',sorted(expected)).fetchone()
assert r[0]==58 and abs(r[1]+466.428572)<1e-8 and r[2:]==(18,38,2)
for p in Path('.').glob('q*.py'):
 ast.parse(p.read_text(),str(p))
 if 'sqlite3.connect' in p.read_text():assert 'mode=ro' in p.read_text() and 'query_only' in p.read_text().lower(),p
print('PASS: exact58-id whitelist; independent SQL count/P&L18W38L2flat;12CME/12CT days;strict n0;116leg rows/55bars each;40stop/11target;10floor proxies;22→7→4→2proxy→1→0 funnel;21stop attempts;591brokerstopslip0;RRtick geometry;Python syntax/read-only settings.')
