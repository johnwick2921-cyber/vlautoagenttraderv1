import { expect, it } from 'vitest'
import { planCard } from './planCard'
it('C5 guide table follows the independently worked break-even equation', () => {
  const table = planCard.blocks.find(
    (b) => b.kind === 'table' && b.title === 'Break-even reference arithmetic'
  )
  expect(table?.kind).toBe('table')
  if (table?.kind !== 'table') throw new Error('reference table missing')
  expect(table.rows).toEqual(
    [0.5, 1, 2, 3].map((b) => [
      `${b}R`,
      `${(100 / (1 + b)).toFixed(2)}%`,
      `${(104 / (1 + b)).toFixed(2)}%`,
    ])
  )
  const text = JSON.stringify(planCard)
  expect(text).toContain('E[net R] = p*b − (1−p) − c')
  expect(text).toContain('Legacy scenarios retain UNKNOWN')
  expect(text).toContain(
    'The structural-stop research candidate'
  )
})
