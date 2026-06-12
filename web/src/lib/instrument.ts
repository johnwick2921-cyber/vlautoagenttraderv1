// Shared CME-futures instrument recognizer for the FE.
//
// Mirrors the Go market.cmeFuturesRoots / store.cmeFuturesRootsStore tables
// (Phase 1). Lifted out of CoinSourceEditor so the Strategy editors can condition
// their crypto-only UI (leverage tiers, USDT labels, funding-rate) on whether the
// active symbol is a CME futures contract — without each component re-declaring it.

// The 18 recognized CME roots (index + treasury resolve live; energy/metals are
// recognized but their front-month resolution is parked — Phase 2.5).
export const cmeFuturesRoots = new Set([
  'NQ',
  'MNQ',
  'ES',
  'MES',
  'RTY',
  'M2K',
  'YM',
  'MYM',
  'CL',
  'MCL',
  'NG',
  'GC',
  'MGC',
  'SI',
  'ZB',
  'ZN',
  'ZF',
  'ZT',
])

const cmeMonthCodes = 'FGHJKMNQUVXZ'

// isCMEFutures reports whether a symbol is a CME futures contract (bare root,
// Databento continuous "NQ.c.0", "ROOT "/"ROOT." qualified, or contract-code
// "MNQU6"). A blank/crypto symbol returns false.
export function isCMEFutures(symbol: string | undefined | null): boolean {
  if (!symbol) return false
  const s = symbol.toUpperCase().trim()
  if (s === '') return false
  if (s.includes('.C.')) return true // Databento continuous form (NQ.c.0, uppercased here)
  if (cmeFuturesRoots.has(s)) return true
  for (const root of cmeFuturesRoots) {
    if (s.startsWith(root + '.') || s.startsWith(root + ' ')) return true
    // contract-code form <ROOT><month-letter><year-digit>, e.g. MNQU6, NGF6
    if (
      s.length === root.length + 2 &&
      s.startsWith(root) &&
      cmeMonthCodes.includes(s[root.length]) &&
      /[0-9]/.test(s[root.length + 1])
    ) {
      return true
    }
  }
  return false
}
