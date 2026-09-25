import pathlib,json,csv,math,sys,subprocess,re
R=pathlib.Path(__file__).resolve().parent;repo=pathlib.Path(sys.argv[1])
def rows(n):return list(csv.DictReader((R/n).open()))
s=json.loads((R/'extract_summary.json').read_text());assert s['era_n']==58;assert math.isclose(s['era_pnl_corrected'],-466.428572,abs_tol=1e-7);assert s['WLF']==[18,38,2];assert len(s['CME_days'])==12;assert s['query_only']==1;assert s['trade_excursions_count']==0
assert set(s['excluded_ids'])=={530,539,545,546,566,571,572,573,574,576,577,579,580}
t=rows('trades.csv');assert [int(r['id']) for r in t]==[587,588,589,590,591];assert sum(float(r['pnl_corrected']) for r in t)==-521.5
for r in t:
 pts=(float(r['exit_price'])-float(r['entry_price']))*(1 if r['side']=='LONG' else -1);assert pts*2==float(r['pnl_corrected'])
assert len(rows('arms.csv'))==39;assert len(rows('intents.csv'))==24;assert len(rows('refusals.csv'))==20
l=rows('opportunity_ledger.csv');assert len(l)==len({r['opportunity'] for r in l})==188
for fn,field in [('arms.csv','arm_ids'),('intents.csv','intent_ids'),('trades.csv','position_ids')]:
 assert {r['id'] for r in rows(fn)}=={s for r in l for s in r[field].split(';') if s}
for r in rows('replay_checkpoints.csv'):assert int(r['bar_open_ms'])+60000<=int(r['time_ms'])
r=(R/'replay.go').read_text();src=(repo/'trader/arm_stop_anchor.go').read_text();assert src[src.index('type StopComposition struct'):src.index('// armStopCompositionLine')] in r
src=(repo/'trader/armed_executor.go').read_text();assert src[src.index('func limitMarketableWrongSide'):src.index('func throughWord')] in r
b=json.loads((R/'bounds_summary.json').read_text());assert [(x['send_opportunities'],x['reachable_opportunities']) for x in b]==[(7,4),(6,4),(7,4),(6,4),(7,3),(6,3)]
a=json.loads((R/'analysis_summary.json').read_text());assert (a['local_pass_opps'],a['actual_send_opps'],a['corrected_send_opps'])==(13,7,6)
br=rows('broker_evidence.csv');zero=[x for x in br if 'Stop price=0' in x['text']];assert len(zero)==84;assert len({re.search("Name='([^']*)'",x['text'])[1] for x in zero})==21
assert any("New state='Filled'" in x['text'] and 'Stop price=29355' in x['text'] and 'Fill price=29355' in x['text'] for x in br)
q=json.loads((R/'q2_asof.json').read_text());assert q['RTH_bars']==90 and q['last_close']==29363.25 and q['known_net']==113.75
changed=subprocess.check_output(['git','diff','--name-only','b4376246'],cwd=repo,text=True).splitlines();assert all(x=='docs/superpowers/reports/2026-09-05-vet-08-stretch.md' or x.startswith('docs/superpowers/reports/2026-09-05-vet-08-stretch-data/') for x in changed),changed
print('PASS: corrected 58-row population, 12 CME days, five trades and point conversion; 188 unique opportunities with complete 39-arm/24-intent/5-position lineage; 20 refusals; no future minute bars in checkpoints; exact copied source functions; broker stop591 evidence; 7/6 submissions and 4/3 conditional reaches; docs-only tracked diff.')
print('NOT VERIFIED / NOT CLAIMED: complete current-rule broker replay, actual counterfactual fills/P&L, original intrabar receipt history, immutable initial risk, post-strict expectancy or a validated continuation edge.')
