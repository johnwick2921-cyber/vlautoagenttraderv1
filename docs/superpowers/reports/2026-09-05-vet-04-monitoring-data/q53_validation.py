import json,pathlib,subprocess,math
root=pathlib.Path('/home/hoang/nofx-analysis/vet-04-complete-0905');d=pathlib.Path('docs/superpowers/reports/2026-09-05-vet-04-monitoring-data');r=pathlib.Path('docs/superpowers/reports/2026-09-05-vet-04-monitoring.md');s=r.read_text();o=json.loads((d/'q51_complete.json').read_text());e=json.loads((d/'q50_eod_verified.json').read_text());h=json.loads((d/'q52_receipts.json').read_text())
assert all('## Q'+str(i)+' —' in s for i in range(1,6))
assert all('### 1.'+str(i)+' ' in s for i in range(1,11))
assert 'Requirement coverage' in s and 'Implementation category' in s
assert o['query_only']==1 and o['era_ms']==1786770000000
assert o['eligible']['n']==58 and abs(o['eligible']['sum_pnl_corrected']+466.428572)<1e-8
assert len(o['cme_days_exit_time'])==12 and o['post_strict_exit']['n']==0
assert o['exclusions']['UNRESOLVABLE']==[530,539,545,546,566,571,580]
assert o['pos591_stop']['broker_stop_slippage_pts']==0
assert e['days']['2026-09-03'][0]['summary']['ids']==[591] and e['days']['2026-09-04'][0]['summary']['n']==0
assert e['days']['2026-09-03'][-1]['counts']['BOOT INTEGRITY']==4 and e['days']['2026-09-04'][-1]['counts']['BOOT INTEGRITY']==4
assert (29355-29301.25)*2==107.5 and (29355-29301.25)/.25==215
assert (29301.25-29144.5)/.25==627 and (29355-29285)*2==140
assert any('15:06:45' in x['text'] and 'ingest' in x['text'] for x in h['receipts'])
assert any('14:18:26' in x['text'] and 'BOOT INTEGRITY' in x['text'] for x in h['receipts'])
assert 'HTTP 200' in h['public_health']['output']
for f in d.glob('q5*.py'):compile(f.read_text(),str(f),'exec')
changed=subprocess.check_output(['git','diff','--name-only'],text=True).splitlines()+subprocess.check_output(['git','ls-files','--others','--exclude-standard'],text=True).splitlines()
assert all(p==str(r) or p.startswith(str(d)+'/') for p in changed),changed
result={'result':'PASS','checked':['all 5 questions and 10 inventory subsections','coverage and recommendation fields','58 population, exact exclusions, corrected P&L, 12 CME days, strict n0','591 accepted-stop slippage and illustrative stop/target arithmetic','15:00 EOD IDs/cutoffs/4 boot receipts each day','fresh recovery log receipts and unauthenticated HTTP200','evidence-script syntax, query_only and exclusive docs scope'],'changed_paths':changed,'limitations':'Documentation/evidence validation, no runtime trading test or deployment.'}
(root/'validation.json').write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
