import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { OneSetupChip, oneSetupChipText } from './OneSetupChip'

// ONE SETUP — the chip's states. The one that matters: an ABSENT verdict
// renders "not evaluated", never "allowed".
describe('OneSetupChip', () => {
  it('renders an absent verdict as NOT EVALUATED, never allowed', () => {
    render(<OneSetupChip id="S1" os={undefined} />)
    const el = screen.getByTestId('one-setup-S1')
    expect(el.getAttribute('data-one-setup')).toBe('not evaluated')
    expect(el.textContent).toContain('one-setup: not evaluated')
    expect(el.textContent).not.toContain('ALLOWED')
  })

  it('renders OFF when the switch is off, whatever the record says', () => {
    render(
      <OneSetupChip
        id="S1"
        os={{
          enabled: false,
          scenarios: {
            S1: { allowed: true, level: 'ok', play: 'ok', permission: 'ok' },
          },
        }}
      />
    )
    expect(
      screen.getByTestId('one-setup-S1').getAttribute('data-one-setup')
    ).toBe('off')
  })

  it('renders ALLOWED with the target choice', () => {
    const txt = oneSetupChipText(
      {
        enabled: true,
        scenarios: {
          S1: {
            allowed: true,
            level: 'ok',
            play: 'ok',
            permission: 'ok',
            target: 'first_obstacle@29520.00',
          },
        },
      },
      'S1'
    )
    expect(txt).toBe('one-setup: ALLOWED · target=first_obstacle@29520.00')
  })

  it('renders a decline with ALL THREE verdicts', () => {
    const txt = oneSetupChipText(
      {
        enabled: true,
        scenarios: {
          S2: {
            allowed: false,
            level: 'level_not_best:ONH@29500(B)',
            play: 'play_not_reject:sweep_reclaim',
            permission: 'day_excluded(ib_held)',
          },
        },
      },
      'S2'
    )
    expect(txt).toContain('level=level_not_best:ONH@29500(B)')
    expect(txt).toContain('play=play_not_reject:sweep_reclaim')
    expect(txt).toContain('permission=day_excluded(ib_held)')
  })

  it('renders second_setup_waiting for an allowed scenario that waits', () => {
    render(
      <OneSetupChip
        id="S3"
        os={{
          enabled: true,
          scenarios: {
            S3: {
              allowed: true,
              level: 'ok',
              play: 'ok',
              permission: 'ok',
              waiting: true,
            },
          },
        }}
      />
    )
    expect(
      screen.getByTestId('one-setup-S3').getAttribute('data-one-setup')
    ).toBe('waiting')
  })
})
