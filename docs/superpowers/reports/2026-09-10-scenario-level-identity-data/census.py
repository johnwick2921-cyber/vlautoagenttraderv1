#!/usr/bin/env python3
"""Dispatch 105 read-only census; run from the pinned running-source worktree.

The two database snapshots are individually consistent, not an atomic pair.
No order-state predicates or credentials are used. Output contains row IDs.
"""
import collections
import json
from pathlib import Path
import sqlite3
import sys
import time
from datetime import datetime
from zoneinfo import ZoneInfo

CT = ZoneInfo('America/Chicago')
W1 = datetime(2026, 9, 10, 16, 51, 28, tzinfo=CT)
WTF = datetime(2026, 9, 10, 17, 10, 33, tzinfo=CT)


def connect(path):
    db = sqlite3.connect('file:' + path + '?mode=ro', uri=True)
    db.row_factory = sqlite3.Row
    db.execute('PRAGMA query_only=ON')
    deadline = time.monotonic() + 120
    db.set_progress_handler(lambda: int(time.monotonic() > deadline), 10000)
    db.execute('BEGIN')
    return db


def groups(rows, field):
    out = collections.defaultdict(list)
    for row in rows:
        value = row[field]
        out['NULL' if value is None else str(value)].append(row['id'])
    return {key: {'n': len(ids), 'ids': ids} for key, ids in sorted(out.items())}


def formation(rows, field):
    result = {'total': len(rows)}
    for label, predicate in (
        ('sql_non_null', lambda v: v is not None),
        ('positive', lambda v: v is not None and v > 0),
        ('zero', lambda v: v == 0),
        ('null', lambda v: v is None),
        ('negative', lambda v: v is not None and v < 0),
    ):
        ids = [r['id'] for r in rows if predicate(r[field])]
        result[label] = {'n': len(ids), 'ids': ids}
    return result


def since(rows, cutoff):
    return [r for r in rows if datetime.fromisoformat(r['created_at']) >= cutoff]


out = {'measured_at_ct': datetime.now(CT).isoformat(),
       'source_revision': '770e2297d2188d09de0dcf76c3722e19e022e4c6',
       'cutoffs_ct': {'w1': W1.isoformat(), 'wtf': WTF.isoformat()}}
db = connect('/home/hoang/nofx/data/data.db')
columns = ('id,level_kind,level_price,formed_at_ms,scenario_nearest,'
           'scenario_link_basis,plan_id,plan_version,created_at')
rows = [dict(r) for r in db.execute('SELECT ' + columns + ' FROM touch_outcomes ORDER BY id')]
out['touch_all'] = {'formation': formation(rows, 'formed_at_ms'),
                    'basis': groups(rows, 'scenario_link_basis'),
                    'by_kind': {k: formation([r for r in rows if r['level_kind'] == k], 'formed_at_ms')
                                for k in sorted({r['level_kind'] for r in rows})},
                    'latest': rows[-5:]}
for label, cutoff in (('w1', W1), ('wtf', WTF)):
    selected = since(rows, cutoff)
    out['touch_since_' + label] = {'n': len(selected), 'ids': [r['id'] for r in selected],
                                  'basis': groups(selected, 'scenario_link_basis'),
                                  'formation': formation(selected, 'formed_at_ms')}
out['table_columns'] = {name: [r['name'] for r in db.execute('PRAGMA table_info(' + name + ')')]
                        for name in ('touch_outcomes', 'candidate_pool', 'armed_orders', 'plans')}
plans = [dict(r) for r in db.execute('SELECT plan_id,version,doc FROM plans ORDER BY created_at')]
out['plans'] = {'n': len(plans), 'ids': [[r['plan_id'], r['version']] for r in plans],
                'scenario_count': 0, 'scenario_level_id_count': 0, 'level_identity_key_counts': {}}
for row in plans:
    doc = json.loads(row['doc'])
    scenarios = doc.get('scenarios') or []
    out['plans']['scenario_count'] += len(scenarios)
    out['plans']['scenario_level_id_count'] += sum('level_id' in s for s in scenarios)
    for level in doc.get('levels') or []:
        for key in ('id', 'level_id', 'kind', 'lo', 'hi', 'origin_date', 'tf', 'formed_at_ms'):
            if key in level:
                counts = out['plans']['level_identity_key_counts']
                counts[key] = counts.get(key, 0) + 1
db.rollback()
db.close()

archive = connect('/home/hoang/nofx/data/data.db.research.db')
out['archive_snapshot_at_ct'] = datetime.now(CT).isoformat()
archive_rows = [dict(r) for r in archive.execute(
    'SELECT id,writer_revision,event,captured_ms,fields_json FROM research_facts '
    'INDEXED BY research_captured WHERE captured_ms>=? AND object=? ORDER BY id',
    (int(WTF.timestamp() * 1000), 'candidate'))]
facts = []
for row in archive_rows:
    fields = json.loads(row.pop('fields_json'))
    origin = fields.get('raw_origin') or {}
    row.update(timeframe=fields.get('timeframe'), formation_ms=fields.get('formation_ms'),
               kind=origin.get('kind'), raw_formed_at_ms=origin.get('formed_at_ms'),
               stable_id_present=fields.get('stable_id') is not None)
    facts.append(row)
out['archive_candidates_since_wtf'] = {
    'n': len(facts), 'ids': [r['id'] for r in facts], 'events': groups(facts, 'event'),
    'writers': groups(facts, 'writer_revision'),
    'by_tf': {tf: formation([r for r in facts if (r['timeframe'] or 'UNKNOWN') == tf], 'formation_ms')
              for tf in sorted({r['timeframe'] or 'UNKNOWN' for r in facts})},
    'rows': facts,
}
out['archive_candidates_since_wtf']['by_event'] = {}
for event in sorted({r['event'] for r in facts}):
    event_rows = [r for r in facts if r['event'] == event]
    out['archive_candidates_since_wtf']['by_event'][event] = {
        'n': len(event_rows), 'ids': [r['id'] for r in event_rows],
        'formation': formation(event_rows, 'formation_ms'),
        'by_tf': {tf: formation([r for r in event_rows if (r['timeframe'] or 'UNKNOWN') == tf], 'formation_ms')
                  for tf in sorted({r['timeframe'] or 'UNKNOWN' for r in event_rows})},
        'by_kind': {kind: formation([r for r in event_rows if (r['kind'] or 'UNKNOWN') == kind], 'formation_ms')
                    for kind in sorted({r['kind'] or 'UNKNOWN' for r in event_rows})},
    }
archive.rollback()
archive.close()
Path(sys.argv[1]).write_text(json.dumps(out, separators=(',', ':')) + '\n')
print(json.dumps({
    'measured_at_ct': out['measured_at_ct'],
    'touch_formation': {k: (v['n'] if isinstance(v, dict) else v)
                        for k, v in out['touch_all']['formation'].items()},
    'since_w1': out['touch_since_w1']['n'], 'since_wtf': out['touch_since_wtf']['n'],
    'archive_candidates_since_wtf': len(facts),
    'archive_by_tf': {k: {n: v['n'] if isinstance(v, dict) else v for n, v in f.items()}
                      for k, f in out['archive_candidates_since_wtf']['by_tf'].items()},
    'plans': {k: v for k, v in out['plans'].items() if k != 'ids'},
}, indent=2))
