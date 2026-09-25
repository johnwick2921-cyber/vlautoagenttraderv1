from pathlib import Path
import json,csv,re,hashlib,subprocess,gzip
r=Path('/home/hoang/nofx-vet-09-complete');d=r/'docs/superpowers/reports';out=r.parent/'nofx-analysis/vet-09-complete-0905'
paths=['2026-09-05-vet-01-way-it-trades-complete-data/trades.csv','2026-09-05-vet-03-decisions-data/complete/population58.csv','2026-09-05-vet-06-risk-data/complete/trade_sample.csv','2026-09-05-vet-07-prompts-complete-data/eligible-positions.csv','2026-09-05-vet-08-stretch-data/complete-0905/era.csv','2026-09-05-vet-10-ideas-data/complete/population.csv']
exclude={530,539,545,546,566,571,580,572,573,574,576,577,579};canonical=None;checks=[]
for name in paths:
 rows=list(csv.DictReader((d/name).open()));rows=[z for z in rows if int(z['id']) not in exclude]
 ids={int(z['id']) for z in rows};assert len(rows)==len(ids)==58,(name,len(rows))
 if canonical is None:canonical=ids
 assert ids==canonical,name
 vals=[float(z['pnl_corrected']) for z in rows];assert abs(sum(vals)+466.428572)<1e-7,name
 assert (sum(v>0 for v in vals),sum(v<0 for v in vals),sum(v==0 for v in vals))==(18,38,2)
 checks.append({'file':name,'n':58,'sum':sum(vals),'ids_match':True})
report_files=[]
for n in range(1,11):
 p=next(d.glob(f'2026-09-05-vet-{n:02d}-*.md'));s=p.read_text();assert re.search('coverage',s,re.I),p
 assert not re.search(r'I (?:ran a discretionary|blew up an account|have traded|am the reviewer.*thirty years)',s),p
 assert not re.search(r'^<{7}|^={7}|^>{7}',s,re.M),p
 report_files.append({'section':n,'path':str(p.relative_to(r)),'lines':len(s.splitlines()),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()})
qs=json.loads((out/'requirements-final.json').read_text());assert len(qs)==47
for q in qs:
 lines=(d/q['report']).read_text().splitlines();assert lines[q['line']-1]=='## '+q['heading'],q
# Every local markdown target exists; exact source evidence code strings are intentionally not parsed as links.
missing=[]
for p in list(d.glob('2026-09-05-vet-*.md')):
 for target in re.findall(r'\]\(([^)]+)\)',p.read_text()):
  target=target.split('#')[0].split(' "')[0]
  if not target or re.match(r'[a-z]+://',target):continue
  if not (p.parent/target).exists():missing.append((p.name,target))
assert not missing,missing
changed=subprocess.check_output(['git','diff','--name-only','b4376246'],cwd=r,text=True).splitlines()+subprocess.check_output(['git','ls-files','--others','--exclude-standard'],cwd=r,text=True).splitlines()
assert all(p.startswith('docs/superpowers/reports/') for p in changed)
secret=[]
for name in set(changed):
 p=r/name
 if p.is_file():
  s=gzip.decompress(p.read_bytes()).decode(errors='replace') if p.suffix=='.gz' else p.read_text(errors='replace')
  if re.search(r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----|\bgh[pousr]_[A-Za-z0-9]{30,}\b|\bsk-[A-Za-z0-9_-]{32,}\b|\beyJ[A-Za-z0-9_-]{20,}\.eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}',s):secret.append(name)
assert not secret,'Credential-shaped content found; matches withheld'
result={'status':'PASS','cohort_crosschecks':checks,'canonical_ids':sorted(canonical),'reports':report_files,'numbered_requirement_groups':47,'docs_only':True,'local_markdown_links_exist':True,'credential_shape_scan':'PASS','limits':'Audit evidence and artifact checks; no runtime acceptance, live expectancy, model behavioral equivalence or exact counterfactual broker fills established.'}
(out/'final-validation.json').write_text(json.dumps(result,indent=2)+'\n');print('PASS: six independent cohort exports, 58 identical IDs and corrected P&L; ten report coverage tables; 47 exact answer links; docs-only scope; local links and credential-shape checks.')
