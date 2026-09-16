// 6.2 (final-bundle 2026-08-19) — EDIT-PUT field audit, source-contract style.
// The cadence_mode drop class: TraderConfigModal collects a field, the edit
// handler silently omits it from the PUT body, and the toggle works on create
// only. This test diffs the modal's saveData keys against the edit request
// body so the NEXT added field cannot regress the same way.
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const read = (p: string) => readFileSync(resolve(__dirname, p), 'utf-8')

describe('trader edit PUT persists every modal field', () => {
  it('every saveData key appears in handleSaveEditTrader request body', () => {
    const modal = read('./TraderConfigModal.tsx')
    const page = read('./AITradersPage.tsx')

    const saveBlock = modal.slice(
      modal.indexOf('const saveData: CreateTraderRequest'),
      modal.indexOf('await onSave(saveData)')
    )
    const keys = [...saveBlock.matchAll(/^\s{8}(\w+):/gm)].map((m) => m[1])
    expect(keys.length).toBeGreaterThanOrEqual(8) // sanity: the block was found

    const editBlock = page.slice(
      page.indexOf('const handleSaveEditTrader'),
      page.indexOf('await api.updateTrader')
    )
    const missing = keys.filter((k) => !editBlock.includes(`${k}:`))
    expect(
      missing,
      `edit PUT drops modal fields: ${missing.join(', ')}`
    ).toEqual([])
  })
})
