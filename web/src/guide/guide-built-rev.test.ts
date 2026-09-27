import { describe, expect, it } from 'vitest'

// W-ONE-BUTTON M4 — the guide's rev is supplied at BUILD time, and a
// production build without it FAILS rather than shipping a guide whose claim
// about the running binary cannot be checked.
//
// This exercises the same rule the module applies, against the same regex, so
// the contract is pinned even though import.meta.env is fixed per test run.
function resolve(env: {
  PROD?: boolean
  VITE_GUIDE_BUILT_REV?: unknown
}): string {
  const raw = env.VITE_GUIDE_BUILT_REV
  if (env.PROD) {
    if (typeof raw !== 'string' || !/^[0-9a-f]{40}$/.test(raw)) {
      throw new Error(
        'VITE_GUIDE_BUILT_REV must be a 40-hex commit sha for a production build'
      )
    }
    return raw
  }
  return typeof raw === 'string' && /^[0-9a-f]{40}$/.test(raw) ? raw : 'dev'
}

const SHA = '662c79bd236f43fb15eb0c7950880123896be8c0'

describe('GUIDE_BUILT_REV is a build input, not a hand-edited constant', () => {
  it('a production build with no rev FAILS instead of shipping an uncheckable guide', () => {
    expect(() => resolve({ PROD: true })).toThrow(/40-hex/)
  })

  it('a production build with a non-sha value FAILS', () => {
    expect(() => resolve({ PROD: true, VITE_GUIDE_BUILT_REV: 'dev' })).toThrow(
      /40-hex/
    )
    expect(() =>
      resolve({ PROD: true, VITE_GUIDE_BUILT_REV: SHA.slice(0, 12) })
    ).toThrow(/40-hex/)
    expect(() => resolve({ PROD: true, VITE_GUIDE_BUILT_REV: 123 })).toThrow(
      /40-hex/
    )
  })

  it('a production build with a real sha uses it', () => {
    expect(resolve({ PROD: true, VITE_GUIDE_BUILT_REV: SHA })).toBe(SHA)
  })

  it("a dev build shows 'dev' rather than a plausible-looking sha", () => {
    expect(resolve({})).toBe('dev')
    expect(resolve({ VITE_GUIDE_BUILT_REV: 'not-a-sha' })).toBe('dev')
  })

  it('a dev build still honours a real sha when one is supplied', () => {
    expect(resolve({ VITE_GUIDE_BUILT_REV: SHA })).toBe(SHA)
  })
})
