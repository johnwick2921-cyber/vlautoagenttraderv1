import { useState } from 'react'
import { Plus, X, Database, TrendingUp, List, Ban, Layers } from 'lucide-react'
import type { CoinSourceConfig } from '../../types'
import { coinSource, ts } from '../../i18n/strategy-translations'
import { VlSelect } from '../ui/select'
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
    { value: 'hyper_all', icon: Database, color: '#F0B90B' },
    { value: 'hyper_main', icon: TrendingUp, color: '#0ECB81' },
    { value: 'mixed', icon: Layers, color: '#a855f7' },
  ] as const

  // CME futures (e.g. MNQ) only use the Static symbol list — Hyperliquid coin
  // sources are crypto-only data feeds (and would make the engine fetch crypto
  // data instead of trading MNQ). Hide them on futures (DISPLAY only — saved
  // data is untouched) and treat the displayed type as Static there.
  const isFutures = isCMEFutures(config.static_coins?.[0])
  const visibleSourceTypes = isFutures
    ? sourceTypes.filter((s) => s.value === 'static')
    : sourceTypes
  const effectiveSourceType = isFutures ? 'static' : config.source_type

  // xyz dex assets (stocks, forex, commodities) - should NOT get USDT suffix
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
    const base = symbol
      .toUpperCase()
      .replace(/^XYZ:/, '')
      .replace(/USDT$|USD$|-USDC$/, '')
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

    // For xyz dex assets (stocks, forex, commodities), use xyz: prefix without USDT
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      // Remove xyz: prefix (case-insensitive) and any USD suffixes
      const base = symbol
        .replace(/^xyz:/i, '')
        .replace(/USDT$|USD$|-USDC$/i, '')
      formattedSymbol = `xyz:${base}`
    } else if (isCMEFutures(symbol)) {
      // CME futures root (ES, MNQ, NG, …) — recognized as a futures symbol, NOT
      // crypto; keep the bare root (no USDT suffix).
      formattedSymbol = symbol
    } else {
      formattedSymbol = symbol.endsWith('USDT') ? symbol : `${symbol}USDT`
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

    // For xyz dex assets, use xyz: prefix without USDT
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      const base = symbol
        .replace(/^xyz:/i, '')
        .replace(/USDT$|USD$|-USDC$/i, '')
      formattedSymbol = `xyz:${base}`
    } else if (isCMEFutures(symbol)) {
      // CME futures root (ES, MNQ, NG, …) — recognized as a futures symbol, NOT
      // crypto; keep the bare root (no USDT suffix).
      formattedSymbol = symbol
    } else {
      formattedSymbol = symbol.endsWith('USDT') ? symbol : `${symbol}USDT`
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
                placeholder="e.g. MNQ, ES, BTC, ETH"
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
              placeholder="e.g. MNQ, ES, BTC, ETH"
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

      {/* Hyperliquid All options — for hyper_all or mixed */}
      {(effectiveSourceType === 'hyper_all' ||
        effectiveSourceType === 'mixed') && (
        <div className="p-4 rounded-lg bg-vl-neo-gold/5 border border-vl-neo-gold/20">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <Database className="w-4 h-4 text-vl-neo-gold" />
              <span className="text-sm font-medium text-vl-neo-text">
                {ts(coinSource.hyperAll, language)}{' '}
                {ts(coinSource.dataSourceConfig, language)}
              </span>
            </div>
          </div>
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={config.use_hyper_all}
              onChange={(e) =>
                !disabled &&
                onChange({ ...config, use_hyper_all: e.target.checked })
              }
              disabled={disabled}
              className="w-5 h-5 rounded accent-vl-neo-gold"
            />
            <span className="text-vl-neo-text">
              {ts(coinSource.useHyperAll, language)}
            </span>
          </label>
          <p className="text-xs pl-8 text-vl-neo-text-muted mt-1">
            {ts(coinSource.hyperAllDesc, language)}
          </p>
        </div>
      )}

      {/* Hyperliquid Main options — for hyper_main or mixed */}
      {(effectiveSourceType === 'hyper_main' ||
        effectiveSourceType === 'mixed') && (
        <div className="p-4 rounded-lg bg-vl-neo-success/5 border border-vl-neo-success/20">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-vl-neo-success" />
              <span className="text-sm font-medium text-vl-neo-text">
                {ts(coinSource.hyperMain, language)}{' '}
                {ts(coinSource.dataSourceConfig, language)}
              </span>
            </div>
          </div>
          <div className="space-y-3">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={config.use_hyper_main}
                onChange={(e) =>
                  !disabled &&
                  onChange({ ...config, use_hyper_main: e.target.checked })
                }
                disabled={disabled}
                className="w-5 h-5 rounded accent-vl-neo-success"
              />
              <span className="text-vl-neo-text">
                {ts(coinSource.useHyperMain, language)}
              </span>
            </label>
            {config.use_hyper_main && (
              <div className="flex items-center gap-3 pl-8">
                <span className="text-sm text-vl-neo-text-muted">
                  {ts(coinSource.hyperMainLimit, language)}:
                </span>
                <VlSelect
                  value={config.hyper_main_limit || 20}
                  onChange={(val) =>
                    !disabled &&
                    onChange({
                      ...config,
                      hyper_main_limit: parseInt(val) || 20,
                    })
                  }
                  disabled={disabled}
                  options={[5, 10, 15, 20, 30, 50].map((n) => ({
                    value: n,
                    label: String(n),
                  }))}
                  className="px-3 py-1.5 rounded bg-vl-neo-bg border border-vl-neo-gold/20 text-vl-neo-text"
                />
              </div>
            )}
            <p className="text-xs text-vl-neo-text-muted">
              {ts(coinSource.hyperMainDesc, language)}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
