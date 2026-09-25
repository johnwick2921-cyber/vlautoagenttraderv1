"""Extract only execution evidence from raw logs; never write source logs."""
from pathlib import Path
import re
root=Path('/home/hoang/nofx/data'); closes=[]; provenance=[];trace=[];guard=[]
for p in sorted(root.glob('nofx_2026-*.log')):
 for n,line in enumerate(p.open(errors='replace'),1):
  clean=re.sub(r'\x1b\[[0-9;]*m','',line).strip()
  if 'NT position closed: MNQ' in clean:
   closes.append(clean);provenance.append(f'{p}:{n}: {clean}')
  if clean.startswith('09-03 ') and any(clean.startswith('09-03 '+h) for h in ['08:45','09:00','09:02','09:03','09:05','09:06','09:15','09:20']):
   if any(t in clean for t in ['arm stop NY S1','armed S1','armed fill S1','MATERIALIZED','UNTRACKED','arm S1','scenario S1','MAE 75.00','29355.00','order_update summary','AI call','Requesting AI','position row not materialized','WIDENED']):trace.append(f'{p}:{n}: {clean}')
  if 'cancelled' in clean and ('marketable, never placed' in clean or 'already' in clean and 'trigger' in clean):guard.append(f'{p}:{n}: {clean}')
Path('log_nt_closes.out').write_text('\n'.join(closes)+'\n')
Path('raw_close_sources.out').write_text('\n'.join(provenance)+'\n')
Path('raw_arm35_sources.out').write_text('\n'.join(trace)+'\n')
Path('raw_guard_sources.out').write_text('\n'.join(guard)+'\n')
print('close lines',len(closes),'trace lines',len(trace),'guard lines',len(guard))
