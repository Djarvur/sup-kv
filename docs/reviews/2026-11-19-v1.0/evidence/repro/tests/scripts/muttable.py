#!/usr/bin/env python3
import json, sys
W='<scratch>/work-tests'
rows=[json.loads(l) for l in open(W+'/out/mutations.jsonl')]
def tops(x, n=3):
    ts=[t for t in x.get('tests',[]) if '/' not in t]
    if not ts and x.get('panics'): return 'panic'
    s=', '.join('`'+t+'`' for t in ts[:n])
    if len(ts)>n: s+=f' +{len(ts)-n}'
    return s
def verdict(r):
    d=r.get('darwin',{}).get('status','?'); l=r.get('linux',{}).get('status','?')
    dk = d=='KILLED'; lk = l=='KILLED'
    if dk and lk: return 'killed'
    if lk and not dk: return '**killed on Linux only**'
    if dk and not lk: return '**killed on darwin only**'
    return '**SURVIVED**'
out=[]
out.append('| # | Mutation (file:line — one-line diff) | Verdict | Killed by (top-level tests; darwin = Linux unless noted) |')
out.append('|---|---|---|---|')
stats={'killed':0,'linux':0,'surv':0}
for r in rows:
    v=verdict(r)
    d=r.get('darwin',{}); l=r.get('linux',{})
    if v=='killed':
        stats['killed']+=1
        dset=[t for t in d.get('tests',[]) if '/' not in t]; lset=[t for t in l.get('tests',[]) if '/' not in t]
        extra=sorted(set(lset)-set(dset))
        k=tops(d)
        if extra: k+= ' ; Linux adds ' + ', '.join('`'+t+'`' for t in extra[:3]) + (f' +{len(extra)-3}' if len(extra)>3 else '')
    elif 'Linux only' in v:
        stats['linux']+=1
        k='Linux: '+tops(l,4)+' ; darwin: none'
    else:
        stats['surv']+=1
        k='—'
    esc=lambda t: t.replace('|','\\|')
    out.append(f"| {r['id']} | `{r['file']}:{r['line']}` — {esc(r['desc'])}. *{esc(r['diff'])}* | {v} | {k} |")
print('\n'.join(out))
print(stats, len(rows), file=sys.stderr)
