import { describe, expect, it } from 'vitest'
import { expectancy } from './content/expectancy'

describe('Stage A research evidence', () => {
  it('explains the five objects, missing values, separate clocks and experiment input', () => {
    const text = JSON.stringify(expectancy.blocks)
    for (const phrase of [
      'market receipts',
      'candidate scores',
      'authoring attempts',
      'scenario permissions',
      'execution outcomes',
      'NULL',
      'receipt, publication and permission',
      'Only exported research bundles',
      'activation is not an order authorization',
    ]) {
      expect(text).toContain(phrase)
    }
  })
})
