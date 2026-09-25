# RESEARCH-SNAPSHOT RECORDER VOLUME — dispatch 103 (DS-103, 2026-09-16)

Branch `fix/research-recorder-volume`, base origin/dev 054e97e5. Footprint:
researchsnapshot/*, main.go:77 wiring, telemetry/ (new counters), the three
docs, this report. logger/provider/trader/store/api/web untouched.

## RED → GREEN (quoted)

| # | RED line (before) | status |
|---|---|---|
| 1 | volume_test.go:81 — "per-fact narration must be gone: 20 line(s) still contain 'research snapshot written:'" | GREEN |
| 2 | volume_test.go:127 — "drop notices must be rate-limited to one line per minute: got 5 line(s)" | GREEN |
| 3 | volume_test.go:142 — "recorder must NOT start when RESEARCH_SNAPSHOT=0; Active() is non-nil" | GREEN |
| 4 | volume_test.go:183 — "rows older than RESEARCH_RETAIN_DAYS must be pruned at boot; 1 remain" | GREEN |
| 5 | (dropped= in rollup, asserted after 1) — "rollup must carry the REAL dropped count (3 forced)" | GREEN |

## What changed

- `researchsnapshot/recorder.go` — per-fact INFO narration removed; row counting
  per object; rollup ticker (RESEARCH_LOG_EVERY_S, default 60s) emitting
  `🗄 research snapshot rollup: rows=N objects={market=X candidate=Y …} drops=D queue=Q`;
  drop notices coalesced (200 ms window) to ONE WARN line per minute with the
  delta; final rollup + drop flush on stop and on the Flush barrier; info/warn
  sinks split (SetInfoLog).
- `researchsnapshot/volume.go` — envDur, coalesceDrop/emitDropWarn/emitRollup,
  pruneOldFacts (DELETE by captured_ms; NEVER auto-VACUUM — a ~77 GB VACUUM
  rewrites the file on the trading DB's disk; the owner decides), retainDays,
  status note.
- `researchsnapshot/runtime.go` — `Start(path, log, warn)`; RESEARCH_SNAPSHOT
  gate (unset/0 → not started, boot line "research snapshot: OFF
  (RESEARCH_SNAPSHOT unset)"); retention at boot + 24 h ticker; CurrentBootLine
  OFF text when inactive.
- `main.go:77` — wiring passes a WARN-level func for drop notices.
- `telemetry/research_snapshot.go` — drops + rows counters (new file only).
- Tests: `researchsnapshot/volume_test.go` (5 pins, fixed-clock offers per
  seam-walk law); runtime_test.go updated for the new signatures + OFF line.
  Full package green; trader seam-walk green; go build ./... clean.

## Measurement (required)

One-hour slice of the live log `data/nofx_2026-09-16.log`, 15:00–16:00 CT
(read-only):

- BEFORE: **324,807** "research snapshot written:" lines/hour — **88.8%** of the
  365,678 log lines in the slice — plus 39,428 drop-warn lines/hour.
- AFTER bound: ≤60 rollup lines + ≤60 drop-warn lines + boot/stop lines per hour.
- Rows/s unchanged: the archive Save path is untouched; every fact still lands
  in research_facts.

## Boot line (every resolved value, no literals)

- OFF: `research snapshot: OFF (RESEARCH_SNAPSHOT=0)` · not-started: `research
  snapshot: n/a` · archive broken: `research snapshot: OFF (archive unavailable)`.
- ON: `research snapshot: ON (default)` when the env is unset (opt-out gate, review B1).
  The existing boot line still prints schema/objects/rows-today/dropped/latency live;
  the rollup carries the real dropped= continuously.

## Owed to the owner

- Decision on the existing ~77 GB archive (keep / compress / delete) and a boot
  in a flat window; env values in .env are his to set.
