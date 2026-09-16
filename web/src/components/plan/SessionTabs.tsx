// P4.2 — session tabs (role=tablist). States: active (gold underline + pulse
// dot) · inactive · reading · disabled (night / not-enabled). Keyboard: arrows
// move between enabled tabs, Enter/Space selects. Auto-advance is the parent's
// job (it flips `selected` when the backend active session changes).

import type { KeyboardEvent } from 'react'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { SessionName } from './sessionConfig'

export type TabState = 'active' | 'inactive' | 'reading' | 'disabled'

export interface SessionTab {
  name: SessionName
  state: TabState
}

interface Props {
  tabs: SessionTab[]
  selected: SessionName
  onSelect: (name: SessionName) => void
  language: Language
}

export function SessionTabs({ tabs, selected, onSelect, language }: Props) {
  const label = (name: SessionName) =>
    tp(
      name === 'ASIA'
        ? 'sessionAsia'
        : name === 'LONDON'
          ? 'sessionLondon'
          : 'sessionNY',
      language
    )

  const enabledNames = tabs
    .filter((t) => t.state !== 'disabled')
    .map((t) => t.name)

  const onKeyDown = (
    e: KeyboardEvent<HTMLButtonElement>,
    name: SessionName
  ) => {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
    e.preventDefault()
    const i = enabledNames.indexOf(name)
    if (i < 0) return
    const next =
      e.key === 'ArrowRight'
        ? (i + 1) % enabledNames.length
        : (i - 1 + enabledNames.length) % enabledNames.length
    onSelect(enabledNames[next])
  }

  return (
    <div
      role="tablist"
      aria-label="Sessions"
      className="flex items-center gap-1"
    >
      {tabs.map((tab) => {
        const isSelected = tab.name === selected
        const disabled = tab.state === 'disabled'
        const reading = tab.state === 'reading'
        const isActive = tab.state === 'active'
        const color = disabled
          ? 'var(--vl-faint)'
          : isSelected
            ? 'var(--vl-gold)'
            : 'var(--vl-muted)'
        return (
          <button
            key={tab.name}
            role="tab"
            aria-selected={isSelected}
            aria-disabled={disabled}
            disabled={disabled}
            tabIndex={isSelected ? 0 : -1}
            onClick={() => !disabled && onSelect(tab.name)}
            onKeyDown={(e) => onKeyDown(e, tab.name)}
            className="relative inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold uppercase tracking-wide transition-colors"
            style={{
              color,
              fontFamily: 'var(--vl-font-ui)',
              borderBottom: isSelected
                ? '2px solid var(--vl-gold)'
                : '2px solid transparent',
              opacity: disabled ? 0.5 : 1,
              cursor: disabled ? 'not-allowed' : 'pointer',
            }}
            title={
              disabled
                ? tp('tabNight', language)
                : reading
                  ? tp('tabReading', language)
                  : label(tab.name)
            }
          >
            {isActive && (
              <span
                aria-hidden
                className="vl-pulse"
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: 999,
                  background: 'var(--vl-gold)',
                  display: 'inline-block',
                }}
              />
            )}
            {reading && (
              <span
                aria-hidden
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: 999,
                  border: '1px solid var(--vl-muted)',
                  display: 'inline-block',
                }}
              />
            )}
            <span>{label(tab.name)}</span>
            {reading && (
              <span
                className="text-[9px] lowercase"
                style={{ color: 'var(--vl-faint)' }}
              >
                {tp('tabReading', language)}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
