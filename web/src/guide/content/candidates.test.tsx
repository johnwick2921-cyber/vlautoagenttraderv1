import { describe, expect, it } from 'vitest'
import { candidates } from './candidates'
import { GUIDE_SECTIONS } from '../GuidePage'
import { GUIDE_BUILT_REV } from '../types'

// W3 — the Guide section ships in the SAME wave as the code (GUIDE CONTENT LAW).
describe('candidates guide section', () => {
  it('is registered and stamped with the built rev', () => {
    expect(GUIDE_SECTIONS.some((s) => s.id === 'candidates')).toBe(true)
    // The stamp is what the drift banner compares against /api/health. A section
    // that forgets it silently claims to match every rev.
    expect(candidates.asBuiltRev).toBe(GUIDE_BUILT_REV)
  })

  it('says the map is kept whole and exclusion is not invalidation', () => {
    const text = JSON.stringify(candidates).toLowerCase()
    expect(text).toContain('exclusion is not invalidation')
    expect(text).toContain('stays in the map')
  })

  it('labels every unvalidated rule as unvalidated', () => {
    const text = JSON.stringify(candidates).toLowerCase()
    // Merging, the target minimum, the reachability order and two of the four
    // projection methods are inventions until E4 measures them. If a later edit
    // quietly drops the hedge, this fails.
    expect(text).toContain('unvalidated')
    expect(text).toContain('e4')
  })

  it('states that the score is NOT changed by the merge', () => {
    const text = JSON.stringify(candidates).toLowerCase()
    expect(text).toContain('does not change any score')
  })

  it('says a projection is never an entry and carries its method', () => {
    const text = JSON.stringify(candidates).toLowerCase()
    expect(text).toContain('never offered as an entry')
    expect(text).toContain('the word ')
    expect(text).toContain('projection')
  })

  it('records that the prior-week guard was NOT relaxed', () => {
    const text = JSON.stringify(candidates).toLowerCase()
    expect(text).toContain('was not relaxed')
    expect(text).toContain('daily bars')
  })
})
