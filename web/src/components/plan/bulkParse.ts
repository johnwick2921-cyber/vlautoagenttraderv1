// P5.2 / P5.7 — the bulk-add + blind-mark grammar parser. One level per line:
//   <price>[-<price2>] [type] [note…]
// e.g.  "30246 D-zone strong 4h OB"  ·  "30150-30152 S-zone"  ·  "30000 RN magnet"
// The SAME parser backs the bulk-add sheet and the owner's blind-mark hour, so a
// line that parses for calibration also parses for a live add. Values stay
// canonical (English tokens); a price is required, type + note are optional.

import { LEVEL_TYPES } from './vocab'

export interface ParsedLevel {
  price: number
  priceHi?: number // when a range "a-b" was given
  type?: string // a canonical LEVEL_TYPES token, when the 2nd word matched one
  note: string
  raw: string
}

const TYPE_TOKENS = new Set(LEVEL_TYPES.map((t) => t.value.toLowerCase()))

// parseBulkLevels parses free multi-line text into staged rows, skipping blank
// lines and any line without a valid leading price. Never throws.
export function parseBulkLevels(text: string): ParsedLevel[] {
  const out: ParsedLevel[] = []
  for (const line of text.split('\n')) {
    const raw = line.trim()
    if (!raw) continue
    const parts = raw.split(/\s+/)
    const priceTok = parts[0]
    const [loStr, hiStr] = priceTok.split('-')
    const price = parseFloat(loStr)
    if (Number.isNaN(price) || price <= 0) continue // no valid price → skip
    const priceHi = hiStr ? parseFloat(hiStr) : undefined

    let rest = parts.slice(1)
    let type: string | undefined
    if (rest.length && TYPE_TOKENS.has(rest[0].toLowerCase())) {
      // restore the canonical-cased token
      const canon = LEVEL_TYPES.find(
        (t) => t.value.toLowerCase() === rest[0].toLowerCase()
      )
      type = canon?.value
      rest = rest.slice(1)
    }
    out.push({
      price,
      priceHi: priceHi && !Number.isNaN(priceHi) ? priceHi : undefined,
      type,
      note: rest.join(' '),
      raw,
    })
  }
  return out
}
