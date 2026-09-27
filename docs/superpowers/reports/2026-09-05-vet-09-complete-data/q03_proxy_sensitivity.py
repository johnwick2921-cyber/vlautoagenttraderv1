from pathlib import Path
import json, math, subprocess
root=Path('/home/hoang/nofx-vet-09-complete');d=root/'docs/superpowers/reports'
x=json.loads((d/'2026-09-05-vet-05-execution-data/complete/q31_verified.json').read_text())
def quant(a,p):
 a=sorted(a);i=(len(a)-1)*p;k=int(i);return a[k]+(a[min(k+1,len(a)-1)]-a[k])*(i-k)
w=[r for r in x['positions'] if r['pnl_corrected']>0 and r['id'] not in (569,584)]
f=[r for r in x['floor_rows'] if r['pnl_corrected']>0 and r['position_id'] not in (569,584) and r['age_ms']<=300000]
out={'purpose':'Sensitivity excludes two uncertain stored zero-MAE winners; not a validated excursion cohort. Raw primary P&L population remains58.', 'input':'2026-09-05-vet-05-execution-data/complete/q31_verified.json','excluded_rows':[r for r in x['positions'] if r['id'] in (569,584)], 'winner_ids':[r['id'] for r in w],'winner_mae_quantiles':{str(p):quant([r['mae'] for r in w],p) for p in (.5,.8,.95)},'floor_ids':[r['position_id'] for r in f],'floor_ratio_quantiles':{str(p):quant([r['mae_over_floor'] for r in f],p) for p in (.5,.8,.95)},'floor_exceeds':sum(r['mae_over_floor']>1 for r in f),'floor_n':len(f)}
out['source_excerpts']={}
for name,lo,hi in [('trader/auto_trader_clock.go',749,765),('docs/superpowers/reports/2026-08-17-cto-final-verification.md',10,14)]:
 lines=(root/name).read_text().splitlines();out['source_excerpts'][name]='\n'.join(f'{i+1}: {lines[i]}' for i in range(lo-1,hi))
(d/'2026-09-05-vet-09-complete-data/q03_proxy_sensitivity.json').write_text(json.dumps(out,indent=2)+'\n')
print(json.dumps({k:v for k,v in out.items() if k not in ('source_excerpts','excluded_rows')},indent=2))
