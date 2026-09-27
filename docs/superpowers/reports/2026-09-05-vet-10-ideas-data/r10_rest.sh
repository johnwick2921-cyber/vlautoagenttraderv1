DB="file:/home/hoang/nofx/data/data.db?mode=ro"
echo "== N2 armed rows 104/105 =="
sqlite3 "$DB" "select id, leg_index, leg_count, kind, condition, entry_px, state, version, armed_under_version, scenario from armed_orders where id in (104,105,102,101,99);"
echo "== N4 refused open_short 09-03 20:35-21:13 =="
sqlite3 "$DB" "select count(*), min(datetime(timestamp,'-5 hours')), max(datetime(timestamp,'-5 hours')) from decision_records where datetime(timestamp,'-5 hours') between '2026-09-03 20:35:00' and '2026-09-03 21:13:00' and execution_log like '%open_short refused%';"
echo "== N5 08-27 gaps =="
sqlite3 "$DB" "select prev, cur, round(gap,1) from (select datetime(lag(timestamp) over (order by timestamp),'-5 hours') prev, datetime(timestamp,'-5 hours') cur, (julianday(timestamp)-julianday(lag(timestamp) over (order by timestamp)))*1440 gap from decision_records where date(timestamp,'-5 hours')='2026-08-27') where gap>30;"
echo "== N6 plans lifecycle =="
sqlite3 "$DB" "select count(*), sum(lifecycle='active') from plans;"
sqlite3 "$DB" "select count(*), sum(lifecycle='active') from plans where trade_date<='2026-09-03';"
sqlite3 "$DB" "select trade_date, count(*), sum(lifecycle='active') from plans where trade_date in ('2026-08-15','2026-08-23','2026-08-30') group by 1;"
echo "== N8 09-03 trade 591 and plan versions =="
sqlite3 "$DB" "select id, side, datetime(entry_time/1000,'unixepoch','-5 hours'), source, cited_scenario_id, plan_version from trader_positions where id=591;"
sqlite3 "$DB" "select version, datetime(created_at,'-5 hours'), substr(doc,instr(doc,'\"direction\"'),40) from plans where plan_id like '2026-09-03:NY%' order by version;"
echo "== N10 model-hours per CT day post-0B =="
sqlite3 "$DB" "select date(timestamp,'-5 hours') d, count(*), round(sum(ai_latency_ms)/3600000.0,2) from decision_records where timestamp >= '2026-09-02 12:49:00' group by 1;"
echo "== N14 weekdays 09-08..10-16 =="
python3 -c "
import datetime as dt
a=dt.date(2026,9,8); b=dt.date(2026,10,16); n=0
while a<=b:
  if a.weekday()<5: n+=1
  a+=dt.timedelta(days=1)
print('weekdays 09-08..10-16 =',n)
a=dt.date(2026,9,7); n=0
while a<=b:
  if a.weekday()<5: n+=1
  a+=dt.timedelta(days=1)
print('weekdays 09-07..10-16 =',n, ' 09-07 is',dt.date(2026,9,7).strftime('%A'))"
sqlite3 "$DB" "select substr(value,instr(value,'half_days'),120) from system_config where key='session_registry';"
echo "== N17 mae/mfe nonzero =="
sqlite3 "$DB" "select count(*), sum(mae is not null and mfe is not null), sum(mae!=0), sum(mae!=0 and mfe!=0) from trader_positions where entry_time>=strftime('%s','2026-08-15')*1000;"
echo "== trade hold minutes, compliant =="
sqlite3 "$DB" "select count(*), round(avg((exit_time-entry_time)/60000.0),1) from trader_positions where entry_time>=strftime('%s','2026-08-15')*1000 and source!='e7_farside_test' and pnl_corrected is not null and plan_id!='UNRESOLVABLE';"
sqlite3 "$DB" "select round((exit_time-entry_time)/60000.0,1) from trader_positions where entry_time>=strftime('%s','2026-08-15')*1000 and source!='e7_farside_test' and pnl_corrected is not null and plan_id!='UNRESOLVABLE' order by (exit_time-entry_time) limit 1 offset 28;"
sqlite3 "$DB" "select sum((exit_time-entry_time)/60000.0 > 13) , count(*) from trader_positions where entry_time>=strftime('%s','2026-08-15')*1000 and source!='e7_farside_test' and pnl_corrected is not null and plan_id!='UNRESOLVABLE';"
echo "== C19 touch_outcomes created_at day =="
sqlite3 "$DB" "select date(created_at,'-5 hours'), count(*) from touch_outcomes group by 1;"
