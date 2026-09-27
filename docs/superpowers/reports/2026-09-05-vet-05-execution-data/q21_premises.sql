.mode list
.headers on
-- epoch ms cutoffs: 2026-08-15 00:00 CT = 2026-08-15 05:00 UTC
SELECT 'cut_0815_ms' k, strftime('%s','2026-08-15 05:00:00')*1000 v;
SELECT 'rows_entry_ge_0815' k, COUNT(*) v FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-15 05:00:00')*1000;
SELECT 'rows_entry_ge_0815_pnlcorr_notnull', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-15 05:00:00')*1000 AND pnl_corrected IS NOT NULL;
SELECT 'rows_entry_ge_0815_plan_id', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-15 05:00:00')*1000 AND plan_id IS NOT NULL AND plan_id<>'';
SELECT 'rows_entry_ge_0815_cited', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-15 05:00:00')*1000 AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>'';
SELECT 'rows_entry_ge_0815_mae_notnull', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-15 05:00:00')*1000 AND mae IS NOT NULL;
SELECT 'rows_mae_notnull_alltime', COUNT(*) FROM trader_positions WHERE mae IS NOT NULL;
SELECT 'rows_pnlcorr_notnull_alltime', COUNT(*) FROM trader_positions WHERE pnl_corrected IS NOT NULL;
SELECT 'rows_pnlcorr_null_alltime', COUNT(*) FROM trader_positions WHERE pnl_corrected IS NULL;
SELECT 'rows_plan_id_alltime', COUNT(*) FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'';
SELECT 'rows_cited_alltime', COUNT(*) FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'';
-- hunt: what cutoff gives 227?
SELECT 'created_ge_0815', COUNT(*) FROM trader_positions WHERE created_at >= '2026-08-15';
SELECT 'created_ge_0801', COUNT(*) FROM trader_positions WHERE created_at >= '2026-08-01';
SELECT 'entry_ge_0801', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-01 05:00:00')*1000;
SELECT 'entry_ge_0805', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-05 05:00:00')*1000;
SELECT 'entry_ge_0810', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-10 05:00:00')*1000;
SELECT 'entry_ge_0812', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-12 05:00:00')*1000;
SELECT 'entry_ge_0813', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-13 05:00:00')*1000;
SELECT 'entry_ge_0814', COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-14 05:00:00')*1000;
SELECT 'typeof_entry_time', typeof(entry_time), COUNT(*) FROM trader_positions GROUP BY 2;
SELECT 'source', source, COUNT(*) FROM trader_positions GROUP BY 2;
SELECT 'plan_session', COALESCE(plan_session,'NULL'), COUNT(*) FROM trader_positions GROUP BY 2;
SELECT 'status', status, COUNT(*) FROM trader_positions GROUP BY 2;
SELECT 'first_pnlcorr_entry_ct', datetime(MIN(entry_time)/1000,'unixepoch','-5 hours') FROM trader_positions WHERE pnl_corrected IS NOT NULL;
SELECT 'pnlcorr_notnull_by_month', strftime('%Y-%m', entry_time/1000,'unixepoch','-5 hours'), COUNT(*) FROM trader_positions WHERE pnl_corrected IS NOT NULL GROUP BY 2;
