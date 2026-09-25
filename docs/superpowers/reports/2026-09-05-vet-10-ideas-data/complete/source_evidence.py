#!/usr/bin/env python3
"""Read source only; preserve numbered excerpts and revision under --out."""
import argparse,pathlib,subprocess
p=argparse.ArgumentParser();p.add_argument('--repo',required=True);p.add_argument('--out',required=True);a=p.parse_args();root=pathlib.Path(a.repo)
spans={'kernel/levels_intraday.go':[(124,145)],'kernel/armed.go':[(9,30)],'kernel/arm_kind.go':[(33,61)],'kernel/planner_prompt.go':[(284,301),(575,580),(612,620),(692,699),(719,735),(750,757)],'kernel/plan_doc.go':[(910,927)],'trader/entry_gate.go':[(157,191)],'trader/arm_stop_anchor.go':[(19,25),(68,96),(135,158)],'kernel/levels_volume.go':[(159,207)],'kernel/regime.go':[(11,28),(49,76)],'kernel/levels_score.go':[(595,616)],'trader/auto_trader_planner.go':[(2145,2150),(2373,2429),(2515,2563)],'kernel/detector_recorder.go':[(63,94)],'trader/detector_record.go':[(38,70)],'kernel/calendar_blackout.go':[(10,30)],'trader/auto_trader_clock.go':[(666,711)],'kernel/session_registry.go':[(85,119)],'kernel/risk_limits.go':[(358,382)],'trader/exit_mechs_suspend.go':[(29,45)],'ninjascript/VLTraderTCPClient.cs':[(967,981)],'trader/ninjatrader/tcp_trader.go':[(484,491)],'docs/superpowers/AUDIT-CHECKLIST.md':[(109,119),(1007,1032),(1080,1098),(1810,1821)]}
lines=['Base source revision: '+subprocess.check_output(['git','rev-parse','488ce82748ca570804240630677c90d3055f128e'],cwd=root,text=True).strip()]
for file,ranges in spans.items():
 text=(root/file).read_text().splitlines()
 for lo,hi in ranges:
  lines.append('\nSOURCE '+file)
  lines.extend(f'{file}:{i}: {text[i-1]}'.rstrip() for i in range(lo,min(hi,len(text))+1))
(pathlib.Path(a.out)/'source_evidence.txt').write_text('\n'.join(lines)+'\n')
