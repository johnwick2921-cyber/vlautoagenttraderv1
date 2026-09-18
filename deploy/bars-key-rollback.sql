-- W-BARS-CONTRACT-KEY rollback (2026-09-18) — RESTORE.md Option A as one transaction.
-- Executed by deploy/bars-key-rollback.sh, which substitutes @OLD@ (the
-- bars_pre_contract_key_<date> table found in sqlite_master) and @NEW@ (the
-- name the migrated table is parked under). Run by hand only with both names
-- substituted; the wrapper takes a VACUUM INTO backup first.
--
-- The new-shape indexes are dropped from the parked copy so their (global)
-- names are free for a later re-migration; the parked table is a discard copy,
-- the backup file holds the full database.
BEGIN;
ALTER TABLE bars RENAME TO "@NEW@";
DROP INDEX IF EXISTS idx_bars_v2_time;
DROP INDEX IF EXISTS idx_bars_v2_source;
ALTER TABLE "@OLD@" RENAME TO bars;
COMMIT;
