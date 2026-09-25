import re, sys, json
def measure(path, split_hdr=True):
    t=open(path,encoding='utf-8').read()
    words=len(t.split()); chars=len(t)
    try:
        import tiktoken
        enc=tiktoken.get_encoding('o200k_base'); tok=len(enc.encode(t))
        enc2=tiktoken.get_encoding('cl100k_base'); tok2=len(enc2.encode(t))
    except Exception as e:
        tok=tok2=None
    caps=lambda w: len(re.findall(r'\b'+w+r'\b', t))
    anyc=lambda w: len(re.findall(r'\b'+w+r'\b', t, re.I))
    out={'file':path,'chars':chars,'words':words,'lines':t.count('\n')+1,'o200k':tok,'cl100k':tok2,
         'MUST':caps('MUST'),'MUST NOT':len(re.findall(r'\bMUST NOT\b',t)),'NEVER':caps('NEVER'),'SHOULD':caps('SHOULD'),'ONLY':caps('ONLY'),'REQUIRED':caps('REQUIRED'),
         'must_any':anyc('must'),'never_any':anyc('never'),'should_any':anyc('should'),'only_any':anyc('only'),'do_not_any':len(re.findall(r'\bdo not\b',t,re.I)),'dont_any':len(re.findall(r"\bdon't\b",t,re.I)),
         'allcaps_tokens':len(re.findall(r'\b[A-Z]{3,}(?:[-_][A-Z]+)*\b',t)),
         'allcaps_distinct':len(set(re.findall(r'\b[A-Z]{3,}(?:[-_][A-Z]+)*\b',t)))}
    # paragraphs
    paras=[p for p in re.split(r'\n\s*\n|\n(?=#)', t) if p.strip()]
    plen=[]
    for p in paras:
        s=[x for x in re.split(r'(?<=[.!?])\s+(?=[A-Z`"(“])', p.strip()) if x.strip()]
        plen.append((len(p), len(s)))
    out['paragraphs']=len(paras)
    out['sentences_total']=sum(n for _,n in plen)
    out['sent_per_para_mean']=round(sum(n for _,n in plen)/len(plen),2)
    big=max(plen,key=lambda x:x[0])
    out['longest_para_chars']=big[0]; out['longest_para_sentences']=big[1]
    return out, t, paras
if __name__=='__main__':
    for p in sys.argv[1:]:
        o,t,paras=measure(p)
        print(json.dumps(o))
