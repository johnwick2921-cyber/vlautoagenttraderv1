import { describe, expect, it } from 'vitest'
import { faqCategories } from './faqData'
import { t } from '../i18n/translations'

// D2 item 9 (A4-S4 P2-3): the FAQ keys were renamed mechanically with their
// definitions; t() returns the raw key on a miss and tsc cannot see a
// mismatch, so every questionKey/answerKey must RESOLVE in all three locales.
describe('faq key lockstep', () => {
  const locales = ['en', 'zh', 'id'] as const

  for (const category of faqCategories) {
    for (const item of category.items) {
      for (const locale of locales) {
        it(`${item.questionKey} resolves in ${locale}`, () => {
          expect(t(item.questionKey, locale)).not.toBe(item.questionKey)
        })
        it(`${item.answerKey} resolves in ${locale}`, () => {
          expect(t(item.answerKey, locale)).not.toBe(item.answerKey)
        })
      }
    }
  }
})
