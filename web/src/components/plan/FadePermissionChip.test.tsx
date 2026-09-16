import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FadePermissionChip, fadeChipText } from './FadePermissionChip'

// W2 D4 — the chip's three states. The one that matters is the third: an
// ABSENT label renders "not evaluated", never "permitted".
describe('FadePermissionChip', () => {
  it('renders an absent label as NOT EVALUATED, never permitted', () => {
    render(<FadePermissionChip id="S1" v={undefined} />)
    const el = screen.getByTestId('fade-S1')
    expect(el.getAttribute('data-fade')).toBe('not evaluated')
    expect(el.textContent).toContain('fade: not evaluated')
    expect(el.textContent).not.toContain('permitted')
  })

  it('renders permitted', () => {
    render(<FadePermissionChip id="S2" v={{ label: 'permitted' }} />)
    expect(screen.getByTestId('fade-S2').getAttribute('data-fade')).toBe(
      'permitted'
    )
  })

  it('renders EVERY exclusion that fired, with its measured detail', () => {
    const txt = fadeChipText({
      label: 'excluded',
      exclusions: ['ib_held', 'or_wide'],
      detail: {
        or_wide: 'OR 154.00 vs 1.28x median (n=13)',
        ib_held: 'price 29490.00 held beyond IB 29375.25',
      },
    })
    expect(txt).toContain('or_wide OR 154.00 vs 1.28x median (n=13)')
    expect(txt).toContain('ib_held price 29490.00 held beyond IB 29375.25')
  })

  it('surfaces UNKNOWN beside a permitted label rather than hiding it', () => {
    render(
      <FadePermissionChip
        id="S3"
        v={{ label: 'permitted', unknown: ['t1_news'] }}
      />
    )
    expect(screen.getByTestId('fade-S3').textContent).toContain(
      'unknown: t1_news'
    )
  })
})
