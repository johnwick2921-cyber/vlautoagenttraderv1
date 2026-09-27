from pathlib import Path
import json,csv,re,subprocess,hashlib,shutil
W=Path('/home/hoang/nofx-vet-07-complete');D=W/'docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data';S=Path('/home/hoang/nofx-analysis/vet-07-complete-0905')
m=json.loads((D/'measurements.json').read_text());p=json.loads((D/'population.json').read_text());r=(D/'appendix-rewrite.txt').read_text();cm=json.loads((D/'constraint-map.json').read_text())
assert p['n']==58 and abs(p['sum_pnl_corrected']+466.428572)<1e-8
assert (p['wins'],p['losses'],p['flats'])==(18,38,2) and len(p['days_17ct'])==12
assert p['post_strict_boot_positions']==[] and p['trade_excursions_n']==0
for e,v in m['tokenizers'].items():assert v['rewrite']['half_length_pass'],e
assert m['rewrite_checks']['uppercase_words']==[] and m['rewrite_checks']['all_lines_numbered'] and m['rewrite_checks']['numbered_lines']==28
assert len(cm)==120 and all(x['rewrite_lines'] for x in cm)
assert all(all(1<=i<=28 for i in x['rewrite_lines']) for x in cm)
rejects=json.loads((D/'reject-groups.json').read_text());ids=[i for x in rejects for i in x['ids']];assert sorted(ids)==list(range(69,133))
parts=list(csv.DictReader((D/'prompt-boundaries.csv').open()))
for doc in {x['document'] for x in parts}:assert ''.join(x['text'] for x in parts if x['document']==doc)==(D/doc).read_text(),doc
report=(W/'docs/superpowers/reports/2026-09-05-vet-07-prompts.md').read_text();assert r.rstrip() in report
for n in range(1,7):assert re.search('^## '+str(n)+r'\.',report,re.M)
assert 'Appendix B — concise executor decision flow' in report and 'noarm' in report and 'position management' in report.lower()
assert json.loads((D/'plan-asia-v5.json').read_text())['doc']['scenarios'][0].get('arm') is None
ny=json.loads((D/'executor-37768-plan.json').read_text());assert ny['version']==6 and ny['doc']['scenarios'][0]['arm']['target']==29503.38
assert 'wait' in json.loads((D/'executor-37768-result.json').read_text())['decision_json']
# No credential payloads in generated JSON; redacted indicator API key is permitted.
def creds(x):
 if isinstance(x,dict):
  for k,v in x.items():
   if re.search('password|secret|api_key|access_token|refresh_token',k,re.I):assert v in ('<redacted>',None),k
   creds(v)
 elif isinstance(x,list):
  for v in x:creds(v)
for f in D.glob('*.json'):creds(json.loads(f.read_text()))
for f in D.iterdir():
 if f.is_file():assert not re.search(r'eyJ[A-Za-z0-9_-]{15,}\.[A-Za-z0-9_-]{15,}\.[A-Za-z0-9_-]{15,}',f.read_text(errors='replace')),f.name
allowed=['docs/superpowers/reports/2026-09-05-vet-07-prompts.md','docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/']
changed=subprocess.check_output(['git','diff','--name-only','b4376246'],cwd=W,text=True).splitlines()+subprocess.check_output(['git','ls-files','--others','--exclude-standard'],cwd=W,text=True).splitlines()
assert all(x==allowed[0] or x.startswith(allowed[1]) for x in changed),changed
prose=['docs/superpowers/reports/2026-09-05-vet-07-prompts.md']+[str(f.relative_to(W)) for f in D.iterdir() if f.suffix in ('.py','.md') and f.name!='policy-cuts.md']
subprocess.run(['git','diff','--check','b4376246','--',*prose],cwd=W,check=True)
checks={'status':'PASS','base':'b4376246','eligible_population':58,'eligible_pnl_corrected':p['sum_pnl_corrected'],'cme_days':12,'rewrite_o200k':m['tokenizers']['o200k_base']['rewrite'],'rewrite_cl100k':m['tokenizers']['cl100k_base']['rewrite'],'mapped_constraint_units':120,'all_source_spans_reconstruct':True,'reject_ids_classified':len(ids),'uppercase_words':0,'numbered_lines':28,'executor_flow_included':True,'joined_real_executor_plan_and_result':True,'docs_only_scope':True,'credential_scan':'PASS','whitespace_scope':'report prose and scripts PASS; raw prompt/source/CSV/exact-cut quotation whitespace intentionally preserved','limits':'Artifact/content checks only; no production model calls, behavioral parity, validator changes or trading mutations. Parent owns merge.'}
(D/'validation.json').write_text(json.dumps(checks,indent=2)+'\n');shutil.copyfile(S/'validate_artifacts.py',D/'validate_artifacts.py')
print(json.dumps(checks,indent=2))
