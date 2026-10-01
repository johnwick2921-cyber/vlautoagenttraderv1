import { describe, expect, it } from 'vitest'
import { translations, t } from './i18n/translations'
import { PRODUCT_NAME, PERSONA_NAME } from './constants/branding'

// The pre-rename token, built at runtime so this guard file holds no
// occurrence of it (the census scans every tracked file).
const oldName = 'NO' + 'FX'

const values = (v: unknown): string[] =>
  typeof v === 'string'
    ? [v]
    : v && typeof v === 'object'
      ? Object.values(v).flatMap(values)
      : []

describe('visible brand languages', () => {
  it('resolves the canonical shared names', () => {
    expect(PRODUCT_NAME).toBe('VL Intelligent')
    expect(PERSONA_NAME).toBe('VL')
  })
  it.each(['en', 'zh', 'id'] as const)(
    'keeps %s product prose canonical without rewriting technical identifiers',
    (lang) => {
      expect(t('appTitle', lang)).toBe('VL')
      expect(t('footerTitle', lang)).toContain('VL')
      for (const value of values(translations[lang])) {
        // URLs, env keys and commands are not product prose.
        const prose = value
          .replace(/https?:\/\/[^\s"']+/g, '')
          .replace(new RegExp('NO' + 'FX_' + '[A-Z_]+', 'g'), '')
        expect(prose).not.toMatch(
          new RegExp(oldName + 'i|\\b' + oldName + '\\b|VL Trader')
        )
      }
    }
  )
})
