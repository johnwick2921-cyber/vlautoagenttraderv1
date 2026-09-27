-- q33: when does nt8_order_snapshots really begin? received_at_ms / emitted_at_ms in UTC and CT
SELECT COUNT(*) n,
       datetime(MIN(received_at_ms)/1000,'unixepoch') first_utc,
       datetime(MIN(received_at_ms)/1000,'unixepoch','-5 hours') first_ct,
       datetime(MAX(received_at_ms)/1000,'unixepoch','-5 hours') last_ct,
       datetime(MIN(emitted_at_ms)/1000,'unixepoch','-5 hours') first_emit_ct
FROM nt8_order_snapshots;
-- the 9-order window in CT (working_count >= 8)
SELECT MIN(datetime(received_at_ms/1000,'unixepoch','-5 hours')) first_ct, MAX(datetime(received_at_ms/1000,'unixepoch','-5 hours')) last_ct, COUNT(*) n
FROM nt8_order_snapshots WHERE working_count >= 8;
