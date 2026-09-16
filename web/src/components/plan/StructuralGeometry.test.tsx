import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { StructuralGeometry } from './StructuralGeometry'

describe('recorded structural geometry', () => {
  it('renders the refused frozen prices and zero quantity', () => {
    render(<StructuralGeometry language="en" rows={[{ scenario: 'S1', leg: 1, entry: 29010, stop: 28987, target: 29014, stop_source: 'zone_edge', quantity: 0, reason: 'rr', detail: 'risk=23 gain=4' }]} />)
    expect(screen.getByText(/Refused · 0 MNQ/)).toBeTruthy()
    expect(screen.getByText(/28987.00.*29014.00/)).toBeTruthy()
  })
  it('does not invent numbers for an uncomputed legacy plan', () => {
    const { container } = render(<StructuralGeometry language="en" />)
    expect(container.textContent).toBe('')
  })
})
