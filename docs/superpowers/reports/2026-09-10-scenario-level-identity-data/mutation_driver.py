from pathlib import Path
import subprocess, json
root=Path(__file__).resolve().parents[4]
data=root/'docs/superpowers/reports/2026-09-10-scenario-level-identity-data'
mutations=[
 ('E2-alias','levelidentity/identity.go','identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s|%d", in.Symbol, in.Kind, *in.Lo, *in.Hi, in.OriginDate, in.TF, *in.FormedCloseMs)','identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s", in.Symbol, in.Kind, *in.Lo, *in.Hi, in.OriginDate, in.TF)', './kernel','^TestIdentityE2'),
 ('E3-refusal','trader/auto_trader_planner.go','identityWarnings := at.stampPlanIdentity(doc, facts.IdentityMap)','identityWarnings := at.stampPlanIdentity(doc, facts.IdentityMap)\n if identityWarnings.Unnamed+identityWarnings.Unresolved>0 {return 0, "identity_refused", fmt.Errorf("mutation: identity refused")}', './trader','^TestIdentityE1AuthoringToEpisodeAndE3WarnProduction$'),
 ('E4-authority','kernel/scenario_level_identity.go','r.Level = &l','l.Price = anchor\n r.Level = &l','./kernel','^TestIdentityE3WarnOnlyAndE4EvaluatorUnchanged$'),
 ('E8-episode-wire','trader/detector_record.go','row.LevelID = kernel.EpisodeLevelID(lv.DetectedLevel, identityDoc)','_ = kernel.EpisodeLevelID(lv.DetectedLevel, identityDoc)\n row.LevelID = nil','./trader','^TestIdentityE1AuthoringToEpisodeAndE3WarnProduction$'),
 ('E8-stamp-wire','trader/auto_trader_planner.go','identityWarnings := at.stampPlanIdentity(doc, facts.IdentityMap)','identityWarnings := kernel.IdentityWarnings{}','./trader','^TestIdentityE1AuthoringToEpisodeAndE3WarnProduction$'),
 ('E8-boot-wire','trader/auto_trader.go','at.logLevelIdentityBootAt(time.Now())','// mutation removed level identity boot call','./trader','^TestIdentityProductionWiring$'),
 ('E8-formation-wire','kernel/levels_intraday.go','out[i] = WithFormationClose(out[i], closeMs, lookback, "window_close", now)','unused','./kernel','^TestIdentityE5OpeningRangeKeepsTradingFields$'),
]
results=[]
for name,path,before,after,pkg,test in mutations:
 p=root/path; original=p.read_text()
 if name=='E8-formation-wire':
  # actual output seam; retain the old trading level unchanged
  line=next(x for x in original.splitlines() if 'out[i] = WithFormationClose(' in x)
  before=line.strip();after='_ = WithFormationClose'+before.split('WithFormationClose',1)[1]
 if original.count(before)!=1:raise RuntimeError((name,'match count',original.count(before)))
 try:
  changed=original.replace(before,after)
  p.write_text(changed)
  confirmed=(p.read_text()==changed and changed!=original)
  # The REFUSE mutation retains the original statement before its new branch.
  if name=='E3-refusal':confirmed=after in p.read_text() and 'mutation: identity refused' in p.read_text()
  build=subprocess.run(['go','build',pkg],cwd=root,capture_output=True,text=True)
  if build.returncode:raise RuntimeError((name,'build failed',build.stderr))
  run=subprocess.run(['go','test',pkg,'-run',test,'-count=1'],cwd=root,capture_output=True,text=True)
  log=f'{name}\npath: {path}\nBEFORE:\n{before}\nAFTER:\n{after}\nreplacement confirmed: {confirmed}\ngo build {pkg}: PASS\ntest rc: {run.returncode}\n'+run.stdout+run.stderr
  (data/(name+'.txt')).write_text(log)
  if run.returncode==0 or not confirmed:raise RuntimeError((name,'mutation survived',log))
  results.append({'name':name,'replacement_confirmed':confirmed,'build':'PASS','mutation':'KILLED'})
 finally:p.write_text(original)
(data/'mutations.json').write_text(json.dumps(results,indent=2)+'\n')
print(json.dumps(results))
