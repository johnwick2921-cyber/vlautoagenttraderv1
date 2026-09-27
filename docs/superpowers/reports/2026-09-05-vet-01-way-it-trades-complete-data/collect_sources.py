from pathlib import Path
import subprocess,json
root=Path('/home/hoang/nofx-vet-01-complete')
spans={'trader/entry_gate.go':[(146,198),(259,306),(325,337)],'trader/arm_stop_anchor.go':[(20,25),(71,88),(91,96),(138,149)],'kernel/session_registry.go':[(83,115)],'kernel/arm_kind.go':[(36,63)],'kernel/armed.go':[(10,29)],'kernel/plan_doc.go':[(97,114),(130,163)],'kernel/condition_status.go':[(25,29)],'trader/exit_mechs_suspend.go':[(30,43)],'trader/auto_trader_clock.go':[(476,485),(503,526),(750,768)],'kernel/mae_mfe.go':[(22,50)],'trader/trade_excursion_hook.go':[(39,68)],'docs/superpowers/reports/2026-08-17-cto-final-verification.md':[(7,15)],'trader/detector_record.go':[(38,73)],'kernel/levels_multiday.go':[(88,99),(146,166)],'kernel/planner_prompt.go':[(721,722),(733,733)],'market/data_indicators.go':[(86,116)],'docs/superpowers/AUDIT-CHECKLIST.md':[(113,130),(1010,1032),(1089,1097),(1820,1842),(1845,1863)]}
with open('source_evidence.txt','w') as f:
 f.write('CODE BASE b4376246; excerpts from own isolated tree; line labels are repository lines.\n')
 for path,ranges in spans.items():
  lines=(root/path).read_text().splitlines()
  for start,end in ranges:
   for n in range(start,end+1):f.write(f'{path}:{n}: {lines[n-1]}\n')
 for path,nums in [('data/nofx_2026-09-02.log',[16710,16755]),('data/nofx_2026-09-03.log',[3658,3696])]:
  lines=(Path('/home/hoang/nofx')/path).read_text(errors='replace').splitlines()
  for n in nums:f.write(f'/home/hoang/nofx/{path}:{n}: {lines[n-1]}\n')
 # Verify executable strict gate in the exact enforced boot revision, without running it.
 r=subprocess.run(['git','show','f478ed880dc9:trader/entry_gate.go'],cwd=root,text=True,capture_output=True,check=True)
 for n,line in enumerate(r.stdout.splitlines(),1):
  if 'in.PlanMode == "strict"' in line or 'Path !=' in line or 'ARM path only' in line:f.write(f'BOOT f478ed880dc9 trader/entry_gate.go:{n}: {line}\n')
 r=subprocess.run(['curl','--silent','--show-error','--max-time','10','--write-out','\nHTTP %{http_code}\n','http://localhost:8080/api/health'],text=True,capture_output=True)
 f.write('READ-ONLY HEALTH '+r.stdout+'\n')
