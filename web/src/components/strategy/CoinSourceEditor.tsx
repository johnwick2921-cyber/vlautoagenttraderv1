import { useState } from 'react'
import { Plus, X, Database, List, Ban } from 'lucide-react'
import type { CoinSourceConfig } from '../../types'
import { coinSource, ts } from '../../i18n/strategy-translations'
import { isCMEFutures } from '../../lib/instrument'

interface CoinSourceEditorProps {
  config: CoinSourceConfig
  onChange: (config: CoinSourceConfig) => void
  disabled?: boolean
  language: string
}

export function CoinSourceEditor({
  config,
  onChange,
  disabled,
  language,
}: CoinSourceEditorProps) {
  const [newCoin, setNewCoin] = useState('')
  const [newExcludedCoin, setNewExcludedCoin] = useState('')

  const sourceTypes = [
    { value: 'static', icon: List, color: '#848E9C' },
  ] as const

  // CME futures (e.g. MNQ) only use the Static symbol list.
  const isFutures = isCMEFutures(config.static_coins?.[0])
  const visibleSourceTypes = isFutures
    ? sourceTypes.filter((s) => s.value === 'static')
    : sourceTypes
  const effectiveSourceType = isFutures ? 'static' : config.source_type

  // xyz dex assets (stocks, forex, commodities)
  const xyzDexAssets = new Set([
    // Stocks
    'TSLA',
    'NVDA',
    'AAPL',
    'MSFT',
    'META',
    'AMZN',
    'GOOGL',
    'AMD',
    'COIN',
    'NFLX',
    'PLTR',
    'HOOD',
    'INTC',
    'MSTR',
    'TSM',
    'ORCL',
    'MU',
    'RIVN',
    'COST',
    'LLY',
    'CRCL',
    'SKHX',
    'SNDK',
    // Forex
    'EUR',
    'JPY',
    // Commodities
    'GOLD',
    'SILVER',
    // Index
    'XYZ100',
  ])

  const isXyzDexAsset = (symbol: string): boolean => {
    const base = symbol.toUpperCase().replace(/^XYZ:/, '').replace(/USD$/, '')
    return xyzDexAssets.has(base)
  }

  // CME-futures recognizer (Phase 1) now lives in web/src/lib/instrument.ts so the
  // Strategy editors can condition crypto-only UI on it too — isCMEFutures imported.

  const MAX_STATIC_COINS = 10

  const showToast = (msg: string) => {
    const toast = document.createElement('div')
    toast.textContent = msg
    toast.className =
      'fixed top-4 left-1/2 -translate-x-1/2 px-4 py-2 rounded-lg text-sm z-50 shadow-lg'
    toast.style.cssText = 'background:#F6465D;color:#fff;'
    document.body.appendChild(toast)
    setTimeout(() => toast.remove(), 2000)
  }

  const handleAddCoin = () => {
    if (!newCoin.trim()) return

    const currentCoins = config.static_coins || []
    if (currentCoins.length >= MAX_STATIC_COINS) {
      showToast(
        language === 'zh'
          ? `最多添加 ${MAX_STATIC_COINS} 个品种`
          : `Maximum ${MAX_STATIC_COINS} symbols allowed`
      )
      return
    }

    const symbol = newCoin.toUpperCase().trim()

    // For xyz dex assets (stocks, forex, commodities), use xyz: prefix
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      // Remove xyz: prefix (case-insensitive) and any USD suffixes
      const base = symbol.replace(/^xyz:/i, '').replace(/USD$/i, '')
      formattedSymbol = `xyz:${base}`
    } else if (isCMEFutures(symbol)) {
      // CME futures root (ES, MNQ, NG, …) — recognized as a futures symbol.
      formattedSymbol = symbol
    } else {
      formattedSymbol = symbol
    }

    if (!currentCoins.includes(formattedSymbol)) {
      onChange({
        ...config,
        static_coins: [...currentCoins, formattedSymbol],
      })
    }
    setNewCoin('')
  }

  const handleRemoveCoin = (coin: string) => {
    onChange({
      ...config,
      static_coins: (config.static_coins || []).filter((c) => c !== coin),
    })
  }

  const handleAddExcludedCoin = () => {
    if (!newExcludedCoin.trim()) return
    const symbol = newExcludedCoin.toUpperCase().trim()

    // For xyz dex assets, use xyz: prefix
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      const base = symbol.replace(/^xyz:/i, '').replace(/USD$/i, '')
      formattedSymbol = `xyz:${base}`
    } else if (isCMEFutures(symbol)) {
      // CME futures root (ES, MNQ, NG, …) — recognized as a futures symbol.
      formattedSymbol = symbol
    } else {
      formattedSymbol = symbol
    }

    const currentExcluded = config.excluded_coins || []
    if (!currentExcluded.includes(formattedSymbol)) {
      onChange({
        ...config,
        excluded_coins: [...currentExcluded, formattedSymbol],
      })
    }
    setNewExcludedCoin('')
  }

  const handleRemoveExcludedCoin = (coin: string) => {
    onChange({
      ...config,
      excluded_coins: (config.excluded_coins || []).filter((c) => c !== coin),
    })
  }

  return (
    <div className="space-y-6">
      {/* Venue badge — Studio Phase 3: a futures strategy must SHOW its real
          venue instead of a crypto "Static List" source (Strategy-Studio plan).
          Display only; the backend infers the venue from the symbol. */}
      {isFutures && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg border border-vl-neo-gold/30 bg-vl-neo-gold/10">
          <Database className="w-4 h-4 text-vl-neo-gold" />
          <span className="text-xs font-medium text-vl-neo-gold">
            NinjaTrader · CME futures
          </span>
          <span className="text-[10px] text-vl-neo-text-muted ml-auto">
            real-time bars + SIM execution via the NT8 TCP bridge
          </span>
        </div>
      )}
      {/* Source Type Selector */}
      <div>
        <label className="block text-sm font-medium mb-3 text-vl-neo-text">
          {ts(coinSource.sourceType, language)}
        </label>
        <div className="grid grid-cols-4 gap-2">
          {visibleSourceTypes.map(({ value, icon: Icon, color }) => (
            <button
              key={value}
              onClick={() =>
                !disabled &&
                onChange({
                  ...config,
                  source_type: value as CoinSourceConfig['source_type'],
                })
              }
              disabled={disabled}
              className={`p-4 rounded-lg border transition-all ${
                effectiveSourceType === value
                  ? 'ring-2 ring-vl-neo-gold bg-vl-neo-gold/10'
                  : 'hover:bg-white/5 bg-vl-neo-bg'
              } border-vl-neo-gold/20`}
            >
              <Icon className="w-6 h-6 mx-auto mb-2" style={{ color }} />
              <div className="text-sm font-medium text-vl-neo-text">
                {ts(coinSource[value as keyof typeof coinSource], language)}
              </div>
              <div className="text-xs mt-1 text-vl-neo-text-muted">
                {ts(
                  coinSource[`${value}Desc` as keyof typeof coinSource],
                  language
                )}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Static Coins - only for static mode */}
      {effectiveSourceType === 'static' && (
        <div>
          <label className="block text-sm font-medium mb-3 text-vl-neo-text">
            {ts(coinSource.staticCoins, language)}
          </label>
          <div className="flex flex-wrap gap-2 mb-3">
            {(config.static_coins || []).map((coin) => (
              <span
                key={coin}
                className="flex items-center gap-1 px-3 py-1.5 rounded-full text-sm bg-vl-neo-bg-lighter text-vl-neo-text"
              >
                {coin}
                {!disabled && (
                  <button
                    onClick={() => handleRemoveCoin(coin)}
                    className="ml-1 hover:text-red-400 transition-colors"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </span>
            ))}
          </div>
          {!disabled && (
            <div className="flex gap-2">
              <input
                type="text"
                value={newCoin}
                onChange={(e) => setNewCoin(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddCoin()}
                placeholder="e.g. MNQ, ES"
                className="flex-1 px-4 py-2 rounded-lg bg-vl-neo-bg border border-vl-neo-gold/20 text-vl-neo-text"
              />
              <button
                onClick={handleAddCoin}
                className="px-4 py-2 rounded-lg flex items-center gap-2 transition-colors bg-vl-neo-gold text-black hover:bg-yellow-500"
              >
                <Plus className="w-4 h-4" />
                {ts(coinSource.addCoin, language)}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Excluded Coins */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <Ban className="w-4 h-4 text-vl-neo-danger" />
          <label className="text-sm font-medium text-vl-neo-text">
            {ts(coinSource.excludedCoins, language)}
          </label>
        </div>
        <p className="text-xs mb-3 text-vl-neo-text-muted">
          {ts(coinSource.excludedCoinsDesc, language)}
        </p>
        <div className="flex flex-wrap gap-2 mb-3">
          {(config.excluded_coins || []).map((coin) => (
            <span
              key={coin}
              className="flex items-center gap-1 px-3 py-1.5 rounded-full text-sm bg-vl-neo-danger/15 text-vl-neo-danger"
            >
              {coin}
              {!disabled && (
                <button
                  onClick={() => handleRemoveExcludedCoin(coin)}
                  className="ml-1 hover:text-white transition-colors"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </span>
          ))}
          {(config.excluded_coins || []).length === 0 && (
            <span className="text-xs italic text-vl-neo-text-muted">
              {ts(coinSource.excludedNone, language)}
            </span>
          )}
        </div>
        {!disabled && (
          <div className="flex gap-2">
            <input
              type="text"
              value={newExcludedCoin}
              onChange={(e) => setNewExcludedCoin(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAddExcludedCoin()}
              placeholder="e.g. MNQ, ES"
              className="flex-1 px-4 py-2 rounded-lg text-sm bg-vl-neo-bg border border-vl-neo-gold/20 text-vl-neo-text"
            />
            <button
              onClick={handleAddExcludedCoin}
              className="px-4 py-2 rounded-lg flex items-center gap-2 transition-colors text-sm bg-vl-neo-danger text-white hover:bg-red-600"
            >
              <Ban className="w-4 h-4" />
              {ts(coinSource.addExcludedCoin, language)}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
