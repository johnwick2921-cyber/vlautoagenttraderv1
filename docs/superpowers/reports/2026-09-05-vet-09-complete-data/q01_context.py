import sqlite3,json,datetime,pathlib,urllib.request,subprocess
out=pathlib.Path('/home/hoang/nofx-analysis/vet-09-complete-0905'); root=pathlib.Path('/home/hoang/nofx-vet-09-complete')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
r={'captured_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'base_revision':subprocess.check_output(['git','rev-parse','origin/dev'],cwd=root,text=True).strip(),'queries':[],'source':{}}
queries=["SELECT id,entry_time,exit_time,plan_id,pnl_corrected,source FROM trader_positions WHERE entry_time >= 1786770000000 ORDER BY id", "SELECT name,sql FROM sqlite_master WHERE type='table' AND name LIKE '%calendar%'", "SELECT count(*) n FROM trade_excursions"]
for q in queries:r['queries'].append({'sql':q,'rows':[dict(x) for x in c.execute(q)]})
for table in [x['name'] for x in r['queries'][1]['rows']]:
 q='SELECT * FROM "'+table.replace('"','""')+'"';r['queries'].append({'sql':q,'rows':[dict(x) for x in c.execute(q)]})
c.rollback();c.close()
with urllib.request.urlopen('http://127.0.0.1:8080/api/health',timeout=10) as v:r['health']={'HTTP':v.status,'body':json.load(v)}
for path,start,end in [('kernel/levels_volume.go',159,220),('kernel/regime.go',11,101),('trader/auto_trader_session.go',96,131),('trader/auto_trader_calendar.go',152,243),('trader/auto_trader_clock.go',663,712),('trader/auto_trader_loop.go',305,355)]:
 a=(root/path).read_text().splitlines();r['source'][path]=[{'line':i+1,'text':a[i]} for i in range(start-1,min(end,len(a)))]
(out/'q01_context.json').write_text(json.dumps(r,indent=2)+'\n');print('Saved read-only context:',len(r['queries']),'queries, health',r['health']['HTTP']);print('Calendar tables:',[x['name'] for x in r['queries'][1]['rows']])
