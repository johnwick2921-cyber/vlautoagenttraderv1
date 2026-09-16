// Dispatch 102 freezes load-bearing identifiers, including their surrounding guards.
// Lock baseline advanced after the separately authorized lock-keeper wave:
// deploy/nofx-lock.sh @ ace51598 (fix/lock-defects-release-meta-halfbuilt),
// following keeper @ 97a6525cb6d10d6c8898b2d277c0fe7581872c24.
// Only its recorded hash changes; protected-file mutation checks remain enforced.
// Bar-feed baselines advanced 2026-09-10 for two owner-dispatched waves that
// touched the protected files without renaming an identifier:
//   provider/ninjatrader/tcp_server.go  @ a53359ce (fix/contract-roll: the
//     subscribed ACK now routes through observeContract; rollMu/lastNamed/rolls
//     fields) — the roll wave's vitest ran in the MAIN tree, whose cwd-relative
//     read of this file was the pre-roll copy, so its 421/421 never saw this
//     change (class 110: a green suite is a claim about an environment).
//   provider/ninjatrader/tcp_framing.go @ c9b224a6 (fix/bar-source: Bar.Source,
//     Go-side only, json:"-").
//   ninjascript/VLTraderTCPClient.cs @ ed6bac8b (owner-ordered front-month fix,
//     2026-09-11: the three GetInstrument sites route through VLInstrumentLookup;
//     no identifier renamed). Advanced in cleanup batch 2 — the red was
//     pre-existing on dev since that commit.
// Bar-feed baselines advanced 2026-09-11 for wave 101 (fix/historical-backfill):
//   provider/ninjatrader/tcp_server.go  — bars_history_request/data/error fan-out
//     + SubscribeBarsHistoryFor (additive; no identifier renamed).
//   provider/ninjatrader/tcp_framing.go — the three new frame types + payloads
//     (additive; no identifier renamed).
//   ninjascript/VLTraderTCPClient.cs   — historyPulls wiring + the
//     bars_history_request dispatch (additive; no identifier renamed).
import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { expect, it } from 'vitest'
import baseline from './test/brand-scope-baseline.json'

function verifyScope(path: string, bytes: Buffer, expected: string) {
  if (createHash('sha256').update(bytes).digest('hex') !== expected) {
    throw new Error(`Dispatch 102 protected file changed: ${path}`)
  }
}
it.each(Object.entries(baseline))(
  'preserves %s byte for byte',
  (path, hash) => {
    expect(() =>
      verifyScope(path, readFileSync(resolve('..', path)), hash)
    ).not.toThrow()
  }
)
it('rejects a removed protected guard, instead of only checking the issuer word', () => {
  const path = 'auth/auth.go'
  const original = readFileSync(resolve('..', path), 'utf8')
  const removed = original.replace('&& token.Valid', '')
  expect(removed).not.toBe(original)
  expect(() => verifyScope(path, Buffer.from(removed), baseline[path])).toThrow(
    'protected file changed'
  )
})

it('preserves every existing TypeScript import target in changed files', async () => {
  const { execFileSync } = await import('node:child_process')
  const ts = await import('typescript')
  const base = '954f11b15f2e7615678f7d2b708c47895faebf1e'
  const git = (args: string[]) =>
    execFileSync('git', args, {
      cwd: resolve('..'),
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    })
  const targets = (source: string) => {
    const file = ts.createSourceFile(
      'source.tsx',
      source,
      ts.ScriptTarget.Latest,
      true,
      ts.ScriptKind.TSX
    )
    return file.statements.flatMap((node) =>
      ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier)
        ? [node.moduleSpecifier.text]
        : []
    )
  }
  for (const path of git(['diff', '--name-only', base, '--', '*.ts', '*.tsx'])
    .trim()
    .split('\n')
    .filter(Boolean)) {
    let old: string
    try {
      old = git(['show', `${base}:${path}`])
    } catch {
      continue
    }
    const current = targets(readFileSync(resolve('..', path), 'utf8'))
    for (const target of targets(old))
      expect(current, `${path}: existing import target ${target}`).toContain(
        target
      )
  }
})
