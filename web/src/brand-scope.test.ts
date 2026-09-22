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
// Bar-feed baseline advanced 2026-09-16 for dispatch 101 (PRs #131/#132,
// fix/nt8-history-and-chart-depth), re-pinned in W-brandscope after PR #140's
// CI found the red — the wave that changed the file never re-pinned it:
//   provider/ninjatrader/tcp_server.go  @ 0252afe4 (dev 1e3ad705, sha256
//     0a757488…): 65f93f6b added the histAtSub field + one
//     s.histAtSub.note(...) call in enqueueBarHistorical (E1, the 🧯 history-at-
//     subscribe count); 0252afe4 added the histReplay field (D1'(3a), the
//     once-per-boot re-request; RequestHistoryReplayAt lives in
//     history_rerequest.go). Additive; the struct block re-aligned by gofmt
//     (addr / listener lines moved, not removed). The guards this pin protects
//     are intact on dev: SubscribeBarsHistoryFor (:421), the
//     bars_history_request write (:470), the bars_history_data / _error
//     fan-out (:1985 / :2008). Verified by diff 0aea0c2e..1e3ad705 on the file.
// go.mod baseline advanced 2026-09-17 for W-CGOFREE-SQLITE-UPSTREAM
// (fix/cgofree-sqlite-upstream): two require lines ADDED —
//   github.com/glebarez/go-sqlite v1.22.0 and github.com/glebarez/sqlite v1.11.0,
// the pure-Go backend behind `go build -tags cgofree` for machines without a C
// compiler (the partner mirror). Nothing removed or bumped: modernc.org/sqlite
// stays v1.40.0, libc stays v1.66.10, gorm.io/driver/sqlite stays v1.6.0. The
// Go security guard the pin protects (patched toolchain/deps) is intact.
// tcp_server.go baseline advanced 2026-09-21 for W-PICTURE-HTF (owner GO,
// merged 23050993): the baseline hash was last pinned at 42c35e2d (the picture
// branch's own 5/6 commit); the merged-HEAD delta vs that pin is EXACTLY the
// 111 inserted lines of the CTO-reviewed coordinated order-update fan-out —
// TCPServer fields (ouFanMu/ouFanNext/ouFanouts) + ListenOrderUpdates +
// runOrderUpdateFanout + the two test feed hooks. ADDITIVE only: zero
// removals, no identifier renamed, no guard removed; the bar-feed guards this
// pin protects (SubscribeBarsHistoryFor, bars_history_request write,
// bars_history_data/_error fan-out) are byte-untouched by that delta.
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

it('preserves every existing TypeScript import target in changed files', async (ctx) => {
  const { execFileSync } = await import('node:child_process')
  const ts = await import('typescript')
  const base = '954f11b15f2e7615678f7d2b708c47895faebf1e'
  const git = (args: string[]) =>
    execFileSync('git', args, {
      cwd: resolve('..'),
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    })
  // The base is a nofx commit. A mirror clone (the VL partner repo) does not
  // carry nofx history, so the pin cannot be evaluated there: skip with the
  // reason stated instead of failing on `git diff` (bad object). In nofx itself
  // the commit exists and the check runs unchanged. TypeScript twin of the Go
  // skip in branding/scope_test.go (TestExistingGoImportTargetsPreserved).
  let baseIsPresent = true
  try {
    git(['cat-file', '-e', `${base}^{commit}`])
  } catch {
    baseIsPresent = false
  }
  if (!baseIsPresent)
    ctx.skip(
      `base commit ${base.slice(0, 8)} is not in this repository (mirror clone) — import-target pin not evaluable here`
    )
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
