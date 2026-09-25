import sqlite3,json,datetime
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.execute('PRAGMA query_only=ON');c.row_factory=sqlite3.Row
allow={'sessions','sessions_enabled','plan_mode','min_risk_reward_ratio','min_sl_atr_mult','eod_flat_offset_min','daily_loss_enabled','daily_loss_limit','max_contracts','condition_status'}
with open('binding.txt','w') as f:
 f.write('mode=ro; query_only=ON; captured '+datetime.datetime.now().isoformat()+' host local time\n')
 f.write('SQL SELECT t.id,t.strategy_id,t.scan_interval_minutes,s.config FROM traders t JOIN strategies s ON s.id=t.strategy_id\n')
 for r in c.execute('SELECT t.id,t.strategy_id,t.scan_interval_minutes,s.config FROM traders t JOIN strategies s ON s.id=t.strategy_id'):
  f.write(json.dumps({k:r[k] for k in ['id','strategy_id','scan_interval_minutes']})+'\n')
  def visit(x,path=''):
   if isinstance(x,dict):
    for k,v in x.items():
     if k in allow:f.write(path+'/'+k+' '+json.dumps(v)+'\n')
     elif isinstance(v,(dict,list)):visit(v,path+'/'+k)
   elif isinstance(x,list):
    for i,v in enumerate(x):visit(v,path+'/'+str(i))
  visit(json.loads(r['config']))
c.close()
