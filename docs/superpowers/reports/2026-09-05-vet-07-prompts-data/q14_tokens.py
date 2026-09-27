import re, json, tiktoken
enc=tiktoken.get_encoding('o200k_base'); enc2=tiktoken.get_encoding('cl100k_base')
T=lambda s:(len(enc.encode(s)),len(enc2.encode(s)))
P=open('q04_planner_prompt_latest.txt',encoding='utf-8').read()
S=open('q06_exec_system_prompt_37768.txt',encoding='utf-8').read()
I=open('q06_exec_input_prompt_37768.txt',encoding='utf-8').read()
print("planner prompt (id 132) chars",len(P),"o200k/cl100k",T(P))
print("executor system (37768) chars",len(S),"o200k/cl100k",T(S))
print("executor input  (37768) chars",len(I),"o200k/cl100k",T(I))
print("executor total chars",len(S)+len(I),"o200k",T(S)[0]+T(I)[0])
# split planner on section headers
secs=re.split(r'\n(?=#{1,3} )', P)
FACT=['## Session','## Regime','## Candles','### 15m','### 1h','### 4h','### daily','## Weekly Context','## Indicators','### 1d','### 3m','### 5m','## VOID','## Minimum stop','## Waterfall displacement','## Measured displacement','## Ranked levels','## Consumed levels','## FRESH FVGs','## Structure','## HTF zones','## Calendar','## Recent context','## Prior plan invalidation','## Prior plan levels']
INSTR=['## CORRECTIONS','# DAY-PLAN READER','## Level roles','## Anchor roles','## BIAS-TREE','## Priority setup','## No-trade gates','## Killzone','## STOP-DOING','## OUTPUT']
rows=[]; fact_t=instr_t=0; fact_c=instr_c=0
for s in secs:
    head=s.strip().split('\n',1)[0][:40]
    t=T(s)[0]
    kind='?'
    if any(head.startswith(k) for k in FACT): kind='FACT'
    elif any(head.startswith(k) for k in INSTR): kind='INSTR'
    # mixed sections: FRESH FVGs header is an instruction sentence + a fact list; Level roles is instruction; bias_ctx line lives in Level roles section (fact)
    rows.append((head,kind,len(s),t))
    if kind=='FACT': fact_t+=t; fact_c+=len(s)
    elif kind=='INSTR': instr_t+=t; instr_c+=len(s)
    else: print("UNCLASSIFIED:",head)
print("\nsection | kind | chars | o200k")
for r in rows: print(f"{r[0]} | {r[1]} | {r[2]} | {r[3]}")
tot=fact_t+instr_t
print(f"\nFACT tokens {fact_t} ({fact_t/tot:.1%}) chars {fact_c}; INSTR tokens {instr_t} ({instr_t/tot:.1%}) chars {instr_c}")
# Rules paragraph
m=re.search(r'\nRules: .*?(?=\n\n|\n## CORRECTIONS)', P, re.S)
R=m.group(0).strip()
sents=[x for x in re.split(r'(?<=[.!?])\s+(?=[A-Z`"(“])', R) if x.strip()]
caps=re.findall(r'\b[A-Z]{3,}(?:[-_][A-Z]+)*\b',R)
print(f"\nRULES paragraph: chars {len(R)} words {len(R.split())} o200k {T(R)[0]} sentences {len(sents)} allcaps {len(caps)} distinct {len(set(caps))} share_of_prompt_tokens {T(R)[0]/T(P)[0]:.1%}")
print("MUST",len(re.findall(r'\bMUST\b',R)),"NEVER",len(re.findall(r'\bNEVER\b',R)),"SHOULD",len(re.findall(r'\bSHOULD\b',R)),"ONLY",len(re.findall(r'\bONLY\b',R)))
fvg=[s for s in sents if re.search(r'fvg',s,re.I)]
print("Rules sentences mentioning fvg:",len(fvg),"tokens",sum(T(s)[0] for s in fvg))
print("fvg mentions whole prompt (any case):",len(re.findall(r'fvg',P,re.I)))
# corrections header
m2=re.search(r'^## CORRECTIONS FROM THIS READ — read these FIRST.*?(?=\n# DAY-PLAN)', P, re.S)
print("corrections header tokens", T(m2.group(0))[0] if m2 else None)
m3=re.search(r'## CORRECTIONS FROM THIS READ \(repeated.*$', P, re.S)
print("corrections tail tokens", T(m3.group(0))[0] if m3 else None)
# OUTPUT schema block (before Rules)
m4=re.search(r'## OUTPUT.*?\n}\n', P, re.S)
print("OUTPUT schema block tokens", T(m4.group(0))[0] if m4 else None)
# executor system split: plan block vs rest
mp=re.search(r'# DAY PLAN.*?(?=\n# Available Data)', S, re.S)
ml=re.search(r'# Live map.*$', S, re.S)
print("\nexecutor: DAY PLAN block tokens", T(mp.group(0))[0], "chars", len(mp.group(0)))
print("executor: Live map + PLAN STATUS tokens", T(ml.group(0))[0], "chars", len(ml.group(0)))
owner=re.search(r'# Trading Frequency.*?(?=\n# DAY PLAN)', S, re.S).group(0)+re.search(r'# Personalized Strategy.*?(?=\n# Live map)', S, re.S).group(0)+re.search(r'# Entry Standards.*?(?=\n# Decision Process)', S, re.S).group(0)
print("executor: owner-typed boxes tokens", T(owner)[0])
# sentence stats executor
for name,txt in [('exec_sys',S),('exec_in',I)]:
    paras=[p for p in re.split(r'\n\s*\n|\n(?=#)', txt) if p.strip()]
    print(name,"paragraphs",len(paras))
