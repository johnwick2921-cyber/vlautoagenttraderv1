import pathlib,subprocess,datetime,sqlite3,json,glob,re
root=pathlib.Path('/home/hoang/nofx-vet-05-complete')
print('Read at',datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=-5))).isoformat())
print('Pinned source base b4376246; historical running rev 36648655 from original health artifact; no new deployment assertion')
ranges={'ninjascript/VLTraderTCPClient.cs':[(970,980),(1338,1385),(1425,1433),(1819,1829)],'trader/armed_executor.go':[(935,945),(978,996),(1250,1290)],'kernel/risk_limits.go':[(151,175),(300,327)],'kernel/engine_analysis.go':[(180,202)],'trader/entry_gate.go':[(145,162)],'trader/exit_mechs_suspend.go':[(10,46)],'kernel/min_sl.go':[(25,43)],'trader/f12_leg4.go':[(199,210)],'store/armed_orders.go':[(190,210)]}
for p,spans in ranges.items():
 lines=(root/p).read_text().splitlines();print('\nSOURCE',p)
 for lo,hi in spans:
  for n in range(lo,hi+1): print(f'{n}: {lines[n-1]}')
for args in [['git','diff','--name-only','2a66d91c','b4376246','--','trader/armed_executor.go','ninjascript/VLTraderTCPClient.cs','provider/ninjatrader/tcp_framing.go','trader/ninjatrader'],['rg','-n','SlippageTicks','--glob','*.go','provider','trader']]:
 r=subprocess.run(args,cwd=root,text=True,capture_output=True);print('COMMAND',args,'EXIT',r.returncode);print(r.stdout)
print('\nNT8 selected source logs (line numbers original)')
slip=0;files=[]
for fn in sorted(glob.glob('/mnt/c/Users/hoang/Documents/NinjaTrader 8/log/log.2026090[34]*.txt')):
 files.append(fn)
 for n,line in enumerate(open(fn,errors='replace'),1):
  if 'slippage' in line.lower():slip+=1
  if any(s in line for s in ['f2b1eb20','931a761a','e38f1774']) and any(s in line for s in ["New state=",'submitted entry','routed to account']): print(pathlib.Path(fn).name+':'+str(n)+': '+line.strip())
print('Slippage text occurrence count',slip,'files',files)
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON')
print('Strategy schema',[(r[1],r[2]) for r in c.execute('pragma table_info(strategies)')])
print('Known bound trader mapping',[(r['id'],r['strategy_id']) for r in c.execute("select id,strategy_id from traders where strategy_id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8'")])
r=c.execute("select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8'").fetchone()
d=json.loads(r[0]); risk=(d.get('ai_config') or {}).get('risk_control') or d.get('risk_control') or {}
print('Bound strategy selected risk fields',json.dumps({k:risk.get(k) for k in ['guardrails_enabled','daily_loss_enabled','daily_loss_limit_usd','breakeven_enabled','trailing_enabled']}))
