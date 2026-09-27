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
  expect(text).toContain('The structural-stop research candidate')
})

it('W2 A3/A4: the guide says identity≠price and the obstacle chain are refused at write', () => {
  const text = JSON.stringify(planCard)
  expect(text).not.toContain('A disagreement is recorded; it does not change')
  expect(text).not.toContain(
    'accepted with WARN and recorded counters for the first two boots'
  )
  expect(text).toContain('A disagreement is REFUSED at write')
  expect(text).toContain('is REFUSED at write and re-authored')
  expect(text).toContain('Obstacle chain (refused at write, new plans only)')
  expect(text).toContain('S4 omits SWG-H·5m 31043.00 (4.00 pts from entry)')
})
