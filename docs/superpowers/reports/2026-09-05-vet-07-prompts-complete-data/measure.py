from pathlib import Path
import json,re,csv,hashlib,statistics,tiktoken
D=Path('/home/hoang/nofx-vet-07-complete/docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data');S=Path('/home/hoang/nofx-analysis/vet-07-complete-0905')
P=(D/'planner-132-actual.txt').read_text(); current=P.replace('legal ONLY on fvg_entry|reject|breakdown_continue|breakup_continue (sweep_reclaim arms only via wait_confirm; breakout_retest|reclaim|hold|acceptance NEVER arm)','legal ONLY on reclaim|reject|fvg_entry|breakdown_continue|breakup_continue (sweep_reclaim arms only via wait_confirm; hold|acceptance|breakout_retest NEVER arm)').replace('≥ 1.0× the current 5m ATR','≥ 1.5× the current 5m ATR')
assert current!=P
(D/'planner-132-current-contract-replay.txt').write_text(current)
R=(S/'rewrite.txt').read_text();(D/'appendix-rewrite.txt').write_text(R)
# Every character belongs to a documented category. Static beliefs are I, not facts.
I=set(range(1,11))|{103,105,127,128,141,157,158,170,214,217}|set(range(229,239))|set(range(243,260))|{271}|set(range(274,279))
SCL=set(range(260,271))
splits={14:' — EVERY',15:' Write the',104:' — waterfall',108:' Stops tighter',111:' A breakdown',177:' (HARD',178:' (HARD',179:' (HARD',241:' — labels'}
def classify(txt,kind):
 out=[]
 for n,l in enumerate(txt.splitlines(keepends=True),1):
  cat='I' if n in I else ('S' if n in SCL else 'F')
  if kind=='planner' and n in splits:
   at=l.index(splits[n]); parts=[('F',l[:at]),('I',l[at:])]
  elif kind=='planner':parts=[(cat,l)]
  elif kind=='system':
   # Plan contents count as supplied contextual facts, including prior authored judgments.
   cat='F' if n in ({9,10,11}|set(range(35,55))|set(range(57,64))|{124,126,142,144,147}|set(range(127,139))|set(range(149,160))|set(range(161,166))) else 'I'
   if 87<=n<=100:cat='S'
   parts=[(cat,l)]
   if n==6:
    at=l.index(' — ALL');parts=[('F',l[:at]),('I',l[at:])]
  else:
   parts=[('I' if n==232 or l.startswith(('Note: an unresolved','Performance:')) else 'F',l)]
  for cat,s in parts:out.append({'line':n,'category':cat,'text':s})
 return out
rows=classify(current,'planner')
for name,txt in [('original',P),('current',current)]:
 rr=classify(txt,'planner');(D/f'planner-instructions-{name}.txt').write_text(''.join(x['text'] for x in rr if x['category']!='F'))
with (D/'prompt-boundaries.csv').open('w') as f:
 w=csv.DictWriter(f,fieldnames=['document','line','category','text']);w.writeheader()
 for name,kind in [('planner-132-actual.txt','planner'),('planner-132-current-contract-replay.txt','planner'),('executor-37768-system_prompt.txt','system'),('executor-37768-input_prompt.txt','input')]:
  for x in classify((D/name).read_text(),kind):w.writerow({'document':name,**x})
def shape(txt):
 paras=[x for x in re.split(r'\n\s*\n|\n(?=#)',txt) if x.strip()]
 ns=[len([s for s in re.split(r'(?<=[.!?])\s+(?=[A-Z`"(“])',p) if s.strip()]) for p in paras]
 return {'characters':len(txt),'words':len(txt.split()),'paragraphs':len(paras),'sentences_per_paragraph':ns,'mean_sentences':round(statistics.mean(ns),4),'max_sentences':max(ns),'uppercase_modals':{x:len(re.findall(r'\b'+x+r'\b',txt)) for x in ['MUST','NEVER','SHOULD']},'case_insensitive_modals':{x:len(re.findall(r'\b'+x+r'\b',txt,re.I)) for x in ['MUST','NEVER','SHOULD']}}
res={'tokenizer_note':'tiktoken o200k_base primary; cl100k_base sensitivity. Neither is DeepSeek provider billing. Same encoding used within each comparison. No model call.', 'tokenizers':{}}
for encname in ['o200k_base','cl100k_base']:
 e=tiktoken.get_encoding(encname);T=lambda s:len(e.encode(s));r={}
 for name,kind in [('planner-132-actual.txt','planner'),('planner-132-current-contract-replay.txt','planner'),('executor-37768-system_prompt.txt','system'),('executor-37768-input_prompt.txt','input')]:
  txt=(D/name).read_text();x=classify(txt,kind)
  # Tokenize concatenated groups; separate groups may have small BPE-boundary differences.
  cats={cat:T(''.join(a['text'] for a in x if a['category']==cat)) for cat in ['F','I','S']};r[name]={'whole_tokens':T(txt),'category_tokens':cats,'category_sum':sum(cats.values()),'facts_share':cats['F']/sum(cats.values()),'instruction_plus_schema_share':(cats['I']+cats['S'])/sum(cats.values())}
 orig=(D/'planner-instructions-current.txt').read_text();r['rewrite']={'original_tokens':T(orig),'rewrite_tokens':T(R),'ratio':T(R)/T(orig),'half_length_pass':T(R)<=T(orig)/2}
 rules=next(l for l in current.splitlines() if l.startswith('Rules:'));r['rules']={'tokens':T(rules),**shape(rules)};res['tokenizers'][encname]=r
res['shape']={n:shape((D/n).read_text()) for n in ['planner-132-actual.txt','executor-37768-system_prompt.txt','executor-37768-input_prompt.txt']}
res['rewrite_checks']={'numbered_lines':len(R.splitlines()),'all_lines_numbered':all(re.match(r'^\d+\. ',l) for l in R.splitlines()),'uppercase_words':re.findall(r'\b[A-Z][A-Z_]{1,}\b',R),'sha256':hashlib.sha256(R.encode()).hexdigest()}
(D/'measurements.json').write_text(json.dumps(res,indent=2)+'\n');print(json.dumps({k:v['rewrite'] for k,v in res['tokenizers'].items()},indent=2))
print(json.dumps(res['tokenizers']['o200k_base'],indent=2)[:4500])
