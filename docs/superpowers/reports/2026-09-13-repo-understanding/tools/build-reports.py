#!/usr/bin/env python3
"""Deterministically package existing audit Markdown; never invent final stamps.
Run from any directory: python3 path/to/tools/build-reports.py [--check]
Only generated CTO-REPORT.md, FULL-AUDIT.md and report-package.json are written.
Original source reports are preserved. No external tools/network/data are used.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
from urllib.parse import quote, unquote, urlsplit

ROOT = Path(__file__).resolve().parent.parent
BASE = '63968be62e44db2fb07a92883e02127b9064b0be'
CORE = ['PACKAGING-UPDATES.md', 'CTO-TRADING-LOGIC.md', 'REPAIR-STATUS.md',
        'CORE-TRACE.md', 'CHECKPOINT.md']
REVIEWS = [f'reviews/{i:02}/report.md' for i in range(1, 31)]
INLINE = re.compile(r'(\]\()(<[^>]+>|[^\s)]+)')
REFERENCE = re.compile(r'^(\s*\[[^\]]+\]:\s*)(<[^>]+>|\S+)')


def digest(data):
    return hashlib.sha256(data).hexdigest()


def rebase(target, source):
    angle = target.startswith('<') and target.endswith('>')
    raw = target[1:-1] if angle else target
    parts = urlsplit(raw)
    if parts.scheme or raw.startswith(('/', '//')):
        return target
    # Fragment-only links must keep pointing to the original document because
    # heading anchors can collide when thirty independent reports are combined.
    path = source if not parts.path else source.parent / unquote(parts.path)
    relative = os.path.relpath(path.resolve(), ROOT)
    result = quote(relative, safe='/.-_~')
    if parts.query:
        result += '?' + parts.query
    if parts.fragment:
        result += '#' + parts.fragment
    return '<' + result + '>' if angle else result


def body(name):
    source = ROOT / name
    lines = source.read_text().splitlines()
    output = []
    fence = None
    for line in lines:
        marker = re.match(r'^\s*(`{3,}|~{3,})', line)
        if marker:
            token = marker.group(1)
            if fence is None:
                fence = token
            elif token[0] == fence[0] and len(token) >= len(fence):
                fence = None
            output.append(line)
            continue
        if fence is None:
            line = INLINE.sub(lambda m: m[1] + rebase(m[2], source), line)
            line = REFERENCE.sub(lambda m: m[1] + rebase(m[2], source), line)
            line = re.sub(r'^(#{1,5}) ', r'#\1 ', line)
        output.append(line)
    return '\n'.join(output).rstrip() + '\n'


def section(name, index, appendix=False):
    notice = ('Historical baseline review at ' + BASE + '. '
              'Findings and line references retain their original scope; later repair dispositions override only named findings.')
    if name.startswith(('reviews/29/', 'reviews/30/')):
        notice = ('Historical bounded independent cross-review of repair revision 99a06543, '
                  'with baseline 63968be. It adds no primary source coverage and is not a review of all later repairs.')
    if not appendix:
        notice = ('Preserved source document; its own revision/checkpoint statements govern. '
                  'Read current repair disposition before interpreting historical review findings.')
    return (f'\n---\n\n<a id="section-{index}"></a>\n\n'
            f'> Original: [{name}]({name}). {notice}\n\n' + body(name))


def build(full=False):
    names = CORE + (REVIEWS if full else [])
    title = 'Full repository audit — editable evidence report' if full else 'CTO repository and trading-process report'
    preface = f'''# {title}

Generated from preserved Markdown sources by `tools/build-reports.py`. This is an editable assembled report; make durable corrections in the linked originals and rebuild. Baseline source scope is **1,049 files / 250,582 lines at {BASE}**, across 28 primary reviews plus two bounded independent reviews. Source review is not runtime verification, deployment approval, or profitability evidence.

**Final verification is recorded only in [CHECKPOINT.md](CHECKPOINT.md#final-verification).** Packaging is not an additional audit or test pass. Current disposition includes committed ordered-execution and entry-receipt lifetime repairs; original numbered appendices remain historical.

Relative links have been rebased to the original artifacts; fragment links point to the original document to avoid duplicate-heading ambiguity. Code fences and original source files are preserved. Machine-readable baseline consistency evidence remains in [coverage-validation.json](coverage-validation.json) and [publication-validation.json](publication-validation.json); the checkpoint describes its limits.

## Contents

'''
    preface += '\n'.join(f'- [{name}](#section-{i})' for i, name in enumerate(names, 1)) + '\n'
    return preface + ''.join(section(name, i, name in REVIEWS) for i, name in enumerate(names, 1))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true', help='fail if generated outputs differ; write nothing')
    args = parser.parse_args()
    missing = [name for name in CORE + REVIEWS if not (ROOT / name).is_file()]
    if missing:
        raise SystemExit('Missing inputs: ' + ', '.join(missing))
    output = {name: build(full).encode() for name, full in [('CTO-REPORT.md', False), ('FULL-AUDIT.md', True)]}
    manifest = {
        'schema': 1, 'baseline_revision': BASE, 'review_appendices': 30,
        'scope': 'Document packaging only; original evidence limits and revision scopes preserved.',
        'builder_sha256': digest(Path(__file__).read_bytes()),
        'inputs': {name: digest((ROOT / name).read_bytes()) for name in CORE + REVIEWS},
        'outputs': {name: {'bytes': len(data), 'words_whitespace': len(data.decode().split()), 'sha256': digest(data)} for name, data in output.items()},
    }
    output['report-package.json'] = (json.dumps(manifest, indent=2) + '\n').encode()
    stale = []
    for name, data in output.items():
        path = ROOT / name
        if args.check:
            if not path.exists() or path.read_bytes() != data:
                stale.append(name)
        else:
            path.write_bytes(data)
    if stale:
        raise SystemExit('Stale generated outputs: ' + ', '.join(stale))
    print(json.dumps(manifest['outputs'], indent=2))


if __name__ == '__main__':
    main()
