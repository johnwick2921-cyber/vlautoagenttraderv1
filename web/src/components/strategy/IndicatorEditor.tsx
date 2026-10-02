import {
  Clock,
  Activity,
  TrendingUp,
  BarChart2,
  Info,
  Lock,
} from 'lucide-react'
import { useState } from 'react'
import useSWR from 'swr'
import { api } from '../../lib/api'
import type { IndicatorConfig } from '../../types'
import { indicator, ts } from '../../i18n/strategy-translations'

interface IndicatorEditorProps {
  config: IndicatorConfig
  onChange: (config: IndicatorConfig) => void
  disabled?: boolean
  language: string
  // Funding rate is a crypto-perpetual concept; CME futures have no funding, so
  // that toggle is hidden when the strategy's instrument is a futures contract.
  isFutures?: boolean
}

// All available timeframes
const allTimeframes = [
  { value: '1m', label: '1m', category: 'scalp' },
  { value: '3m', label: '3m', category: 'scalp' },
  { value: '5m', label: '5m', category: 'scalp' },
  { value: '15m', label: '15m', category: 'intraday' },
  { value: '30m', label: '30m', category: 'intraday' },
  { value: '1h', label: '1h', category: 'intraday' },
  { value: '2h', label: '2h', category: 'swing' },
  { value: '4h', label: '4h', category: 'swing' },
  { value: '6h', label: '6h', category: 'swing' },
  { value: '8h', label: '8h', category: 'swing' },
  { value: '12h', label: '12h', category: 'swing' },
  { value: '1d', label: '1D', category: 'position' },
  { value: '3d', label: '3D', category: 'position' },
  { value: '1w', label: '1W', category: 'position' },
]

// PeriodInput edits a multi-value indicator period field (EMA/RSI/ATR/BOLL,
// stored as number[]). It holds the RAW typed text in local state WHILE editing
// — so commas, clearing, and mid-string edits are all allowed — and parses →
// number[] ONLY on blur/commit (Enter blurs). The saved shape is unchanged:
// onCommit always receives a number[] (the default periods if left empty). When
// not editing (draft === null) the field is derived straight from the saved
// value, so a strategy switch / reset-to-default just works.
function PeriodInput({
  value,
  defaultPeriods,
  disabled,
  onCommit,
}: {
  value?: number[]
  defaultPeriods: string
  disabled?: boolean
  onCommit: (periods: number[]) => void
}) {
  const [draft, setDraft] = useState<string | null>(null)
  const saved = value && value.length > 0 ? value.join(',') : defaultPeriods
  const shown = draft ?? saved

  const parse = (raw: string) =>
    raw
      .split(',')
      .map((s) => parseInt(s.trim(), 10))
      .filter((n) => !isNaN(n) && n > 0)

  const commit = () => {
    if (draft === null) return
    let periods = parse(draft)
    if (periods.length === 0) periods = parse(defaultPeriods) // empty → default
    onCommit(periods)
    setDraft(null)
  }

  return (
    <input
      type="text"
      value={shown}
      onFocus={() => !disabled && setDraft(saved)}
      onChange={(e) => !disabled && setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
      }}
      disabled={disabled}
      placeholder={defaultPeriods}
      className="w-full px-2 py-1 rounded text-[10px] text-center"
      style={{
        background: '#1E2329',
        border: '1px solid #2B3139',
        color: '#EAECEF',
      }}
    />
  )
}

export function IndicatorEditor({
  config,
  onChange,
  disabled,
  language,
  isFutures = false,
}: IndicatorEditorProps) {
  // Capability list (single source of truth) — fetched from the backend
  // (store.SupportedTimeframes). Falls back to the local allTimeframes values
  // if the request fails, so the selector never goes empty. The local table
  // still supplies label + category (presentation); the backend decides which
  // values are actually selectable, so the menu can't drift from what the
  // engine/NT8 actually serves.
  const { data: tfCapability } = useSWR(
    'supported-timeframes',
    api.getSupportedTimeframes,
    {
      revalidateOnFocus: false,
      dedupingInterval: 600000,
      shouldRetryOnError: false,
    }
  )
  const capabilityValues =
    tfCapability?.timeframes ?? allTimeframes.map((t) => t.value)
  const softWarnAbove = tfCapability?.soft_warn_above ?? 6
  const availableTimeframes = [
    ...allTimeframes.filter((t) => capabilityValues.includes(t.value)),
    // Defensive: a capability value with no local label/category (shouldn't
    // happen — the Go parity test keeps the lists in lockstep).
    ...capabilityValues
      .filter((v) => !allTimeframes.some((t) => t.value === v))
      .map((v) => ({ value: v, label: v, category: 'position' as const })),
  ]

  // Get currently selected timeframes
  const selectedTimeframes = config.klines.selected_timeframes || [
    config.klines.primary_timeframe,
  ]

  // Toggle timeframe selection
  const toggleTimeframe = (tf: string) => {
    if (disabled) return
    const current = [...selectedTimeframes]
    const index = current.indexOf(tf)

    if (index >= 0) {
      if (current.length > 1) {
        current.splice(index, 1)
        const newPrimary =
          tf === config.klines.primary_timeframe
            ? current[0]
            : config.klines.primary_timeframe
        onChange({
          ...config,
          klines: {
            ...config.klines,
            selected_timeframes: current,
            primary_timeframe: newPrimary,
            enable_multi_timeframe: current.length > 1,
          },
        })
      }
    } else {
      // Soft advisory above the threshold — NEVER a hard block (this replaced
      // the old artificial max-4 cap). The timeframe is still added; we just
      // warn that more timeframes = a bigger AI prompt → slower, costlier
      // decisions. capped by the backend MaxTimeframes (the real capability).
      if (current.length >= softWarnAbove) {
        const toast = document.createElement('div')
        toast.textContent =
          language === 'zh'
            ? `已选 ${current.length + 1} 个时间维度：维度越多，AI 提示词越大，决策更慢、成本更高`
            : `${current.length + 1} timeframes selected — more means a bigger AI prompt: slower, costlier decisions`
        toast.className =
          'fixed top-4 left-1/2 -translate-x-1/2 px-4 py-2 rounded-lg text-sm z-50 shadow-lg'
        toast.style.cssText = 'background:#F0B90B;color:#000;'
        document.body.appendChild(toast)
        setTimeout(() => toast.remove(), 2600)
        // fall through — do NOT return; the timeframe is still selected
      }
      current.push(tf)
      onChange({
        ...config,
        klines: {
          ...config.klines,
          selected_timeframes: current,
          enable_multi_timeframe: current.length > 1,
        },
      })
    }
  }

  // Set primary timeframe
  const setPrimaryTimeframe = (tf: string) => {
    if (disabled) return
    onChange({
      ...config,
      klines: {
        ...config.klines,
        primary_timeframe: tf,
      },
    })
  }

  const categoryColors: Record<string, string> = {
    scalp: '#F6465D',
    intraday: '#F0B90B',
    swing: '#0ECB81',
    position: '#60a5fa',
  }

  // Ensure enable_raw_klines is always true
  const ensureRawKlines = () => {
    if (!config.enable_raw_klines) {
      onChange({ ...config, enable_raw_klines: true })
    }
  }

  // Call on mount if needed
  if (
    config.enable_raw_klines === undefined ||
    config.enable_raw_klines === false
  ) {
    ensureRawKlines()
  }

  return (
    <div className="space-y-5">
      {/* ============================================ */}
      {/* Section 1: Market Data (Required)           */}
      {/* ============================================ */}
      <div
        className="rounded-lg overflow-hidden"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div
          className="px-3 py-2 flex items-center gap-2"
          style={{ background: '#1E2329', borderBottom: '1px solid #2B3139' }}
        >
          <BarChart2 className="w-4 h-4" style={{ color: '#F0B90B' }} />
          <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
            {ts(indicator.marketData, language)}
          </span>
          <span className="text-xs" style={{ color: '#848E9C' }}>
            - {ts(indicator.marketDataDesc, language)}
          </span>
        </div>

        <div className="p-3 space-y-4">
          {/* Raw Klines - Required, Always On.
              DEAD lever on futures: enable_raw_klines is force-set true on mount
              (ensureRawKlines) and no futures reader gates on it, so this locked
              "Required" card is a switch that does nothing. Hidden on futures
              (the flag stays true underneath); crypto keeps it — the card is the
              honest "always-on" marker there. */}
          {!isFutures && (
            <div
              className="flex items-center justify-between p-3 rounded-lg"
              style={{
                background: 'rgba(240, 185, 11, 0.08)',
                border: '1px solid rgba(240, 185, 11, 0.2)',
              }}
            >
              <div className="flex items-center gap-3">
                <div
                  className="w-8 h-8 rounded-lg flex items-center justify-center"
                  style={{ background: 'rgba(240, 185, 11, 0.15)' }}
                >
                  <TrendingUp
                    className="w-4 h-4"
                    style={{ color: '#F0B90B' }}
                  />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span
                      className="text-sm font-medium"
                      style={{ color: '#EAECEF' }}
                    >
                      {ts(indicator.rawKlines, language)}
                    </span>
                    <span
                      className="px-1.5 py-0.5 rounded text-[10px] font-medium flex items-center gap-1"
                      style={{
                        background: 'rgba(240, 185, 11, 0.2)',
                        color: '#F0B90B',
                      }}
                    >
                      <Lock className="w-2.5 h-2.5" />
                      {ts(indicator.required, language)}
                    </span>
                  </div>
                  <p className="text-xs mt-0.5" style={{ color: '#848E9C' }}>
                    {ts(indicator.rawKlinesDesc, language)}
                  </p>
                </div>
              </div>
              <input
                type="checkbox"
                checked={true}
                disabled={true}
                className="w-5 h-5 rounded accent-yellow-500 cursor-not-allowed"
              />
            </div>
          )}

          {/* Timeframe Selection */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <Clock className="w-3.5 h-3.5" style={{ color: '#848E9C' }} />
                <span
                  className="text-xs font-medium"
                  style={{ color: '#EAECEF' }}
                >
                  {ts(indicator.timeframes, language)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-[10px]" style={{ color: '#848E9C' }}>
                  {ts(indicator.klineCount, language)}:
                </span>
                <input
                  type="number"
                  value={config.klines.primary_count}
                  onChange={(e) =>
                    !disabled &&
                    onChange({
                      ...config,
                      klines: {
                        ...config.klines,
                        primary_count: parseInt(e.target.value) || 30,
                      },
                    })
                  }
                  disabled={disabled}
                  min={10}
                  max={30}
                  className="w-16 px-2 py-1 rounded text-xs text-center"
                  style={{
                    background: '#1E2329',
                    border: '1px solid #2B3139',
                    color: '#EAECEF',
                  }}
                />
              </div>
            </div>
            <p className="text-[10px] mb-2" style={{ color: '#5E6673' }}>
              {ts(indicator.timeframesDesc, language)}
            </p>

            {/* Timeframe Grid */}
            <div className="space-y-1.5">
              {(['scalp', 'intraday', 'swing', 'position'] as const).map(
                (category) => {
                  const categoryTfs = availableTimeframes.filter(
                    (tf) => tf.category === category
                  )
                  return (
                    <div key={category} className="flex items-center gap-2">
                      <span
                        className="text-[10px] w-10 flex-shrink-0"
                        style={{ color: categoryColors[category] }}
                      >
                        {ts(indicator[category], language)}
                      </span>
                      <div className="flex flex-wrap gap-1">
                        {categoryTfs.map((tf) => {
                          const isSelected = selectedTimeframes.includes(
                            tf.value
                          )
                          const isPrimary =
                            config.klines.primary_timeframe === tf.value
                          return (
                            <button
                              key={tf.value}
                              onClick={() => toggleTimeframe(tf.value)}
                              onDoubleClick={() =>
                                setPrimaryTimeframe(tf.value)
                              }
                              disabled={disabled}
                              className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                                isSelected ? '' : 'opacity-40 hover:opacity-70'
                              }`}
                              style={{
                                background: isSelected
                                  ? `${categoryColors[category]}15`
                                  : 'transparent',
                                border: `1px solid ${isSelected ? categoryColors[category] : '#2B3139'}`,
                                color: isSelected
                                  ? categoryColors[category]
                                  : '#848E9C',
                                boxShadow: isPrimary
                                  ? `0 0 0 2px ${categoryColors[category]}`
                                  : undefined,
                              }}
                              title={
                                isPrimary ? `${tf.label} (Primary)` : tf.label
                              }
                            >
                              {tf.label}
                              {isPrimary && (
                                <span className="ml-0.5 text-[8px]">★</span>
                              )}
                            </button>
                          )
                        })}
                      </div>
                    </div>
                  )
                }
              )}
            </div>
          </div>
        </div>
      </div>

      {/* ============================================ */}
      {/* Section 2: Technical Indicators (Optional)  */}
      {/* ============================================ */}
      <div
        className="rounded-lg overflow-hidden"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div
          className="px-3 py-2 flex items-center gap-2"
          style={{ background: '#1E2329', borderBottom: '1px solid #2B3139' }}
        >
          <Activity className="w-4 h-4" style={{ color: '#0ECB81' }} />
          <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
            {ts(indicator.technicalIndicators, language)}
          </span>
          <span className="text-xs" style={{ color: '#848E9C' }}>
            - {ts(indicator.technicalIndicatorsDesc, language)}
          </span>
        </div>

        <div className="p-3">
          {/* Tip */}
          <div
            className="flex items-start gap-2 mb-3 p-2 rounded"
            style={{ background: 'rgba(14, 203, 129, 0.05)' }}
          >
            <Info
              className="w-3.5 h-3.5 mt-0.5 flex-shrink-0"
              style={{ color: '#0ECB81' }}
            />
            <p className="text-[10px]" style={{ color: '#848E9C' }}>
              {ts(indicator.aiCanCalculate, language)}
            </p>
          </div>

          {/* Indicator Grid */}
          <div className="grid grid-cols-2 gap-2">
            {[
              {
                key: 'enable_ema',
                label: 'ema',
                desc: 'emaDesc',
                color: '#F0B90B',
                periodKey: 'ema_periods',
                defaultPeriods: '20,50',
              },
              {
                key: 'enable_macd',
                label: 'macd',
                desc: 'macdDesc',
                color: '#a855f7',
              },
              {
                key: 'enable_rsi',
                label: 'rsi',
                desc: 'rsiDesc',
                color: '#F6465D',
                periodKey: 'rsi_periods',
                defaultPeriods: '7,14',
              },
              {
                key: 'enable_atr',
                label: 'atr',
                desc: 'atrDesc',
                color: '#60a5fa',
                periodKey: 'atr_periods',
                defaultPeriods: '14',
              },
              {
                key: 'enable_boll',
                label: 'boll',
                desc: 'bollDesc',
                color: '#ec4899',
                periodKey: 'boll_periods',
                defaultPeriods: '20',
              },
              {
                key: 'enable_svp',
                label: 'svp',
                desc: 'svpDesc',
                color: '#F0B90B',
              },
            ].map(({ key, label, desc, color, periodKey, defaultPeriods }) => (
              <div
                key={key}
                className="p-2.5 rounded-lg transition-all"
                style={{
                  background: config[key as keyof IndicatorConfig]
                    ? `${color}08`
                    : 'transparent',
                  border: `1px solid ${config[key as keyof IndicatorConfig] ? `${color}30` : '#2B3139'}`,
                }}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2">
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ background: color }}
                    />
                    <span
                      className="text-xs font-medium"
                      style={{ color: '#EAECEF' }}
                    >
                      {ts(indicator[label as keyof typeof indicator], language)}
                    </span>
                  </div>
                  <input
                    type="checkbox"
                    checked={
                      (config[key as keyof IndicatorConfig] as boolean) || false
                    }
                    onChange={(e) =>
                      !disabled &&
                      onChange({ ...config, [key]: e.target.checked })
                    }
                    disabled={disabled}
                    className="w-4 h-4 rounded accent-yellow-500"
                  />
                </div>
                <p className="text-[10px] mb-1.5" style={{ color: '#5E6673' }}>
                  {ts(indicator[desc as keyof typeof indicator], language)}
                </p>
                {periodKey && config[key as keyof IndicatorConfig] && (
                  <PeriodInput
                    value={
                      config[periodKey as keyof IndicatorConfig] as
                        | number[]
                        | undefined
                    }
                    defaultPeriods={defaultPeriods}
                    disabled={disabled}
                    onCommit={(periods) =>
                      onChange({ ...config, [periodKey]: periods })
                    }
                  />
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ============================================ */}
      {/* Section 3: Market Sentiment                 */}
      {/* ============================================ */}
      <div
        className="rounded-lg overflow-hidden"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div
          className="px-3 py-2 flex items-center gap-2"
          style={{ background: '#1E2329', borderBottom: '1px solid #2B3139' }}
        >
          <TrendingUp className="w-4 h-4" style={{ color: '#22c55e' }} />
          <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
            {ts(indicator.marketSentiment, language)}
          </span>
          <span className="text-xs" style={{ color: '#848E9C' }}>
            -{' '}
            {ts(
              isFutures
                ? indicator.marketSentimentDescFutures
                : indicator.marketSentimentDesc,
              language
            )}
          </span>
        </div>

        <div className="p-3">
          <div className="grid grid-cols-3 gap-2">
            {[
              {
                key: 'enable_volume',
                label: 'volume',
                desc: 'volumeDesc',
                color: '#c084fc',
                cryptoOnly: false,
              },
              {
                key: 'enable_oi',
                label: 'oi',
                desc: 'oiDesc',
                color: '#34d399',
                // Open Interest is the legacy crypto-perp feed (empty zeros on
                // CME futures). Hidden on futures like funding rate — no real
                // futures OI is wired (NT8 bridge carries OHLCV only).
                cryptoOnly: true,
              },
              {
                key: 'enable_funding_rate',
                label: 'fundingRate',
                desc: 'fundingRateDesc',
                color: '#fbbf24',
                cryptoOnly: true,
              },
            ]
              .filter(({ cryptoOnly }) => !cryptoOnly || !isFutures)
              .map(({ key, label, desc, color }) => (
                <div
                  key={key}
                  className="p-2.5 rounded-lg transition-all"
                  style={{
                    background: config[key as keyof IndicatorConfig]
                      ? `${color}08`
                      : 'transparent',
                    border: `1px solid ${config[key as keyof IndicatorConfig] ? `${color}30` : '#2B3139'}`,
                  }}
                >
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      <div
                        className="w-2 h-2 rounded-full"
                        style={{ background: color }}
                      />
                      <span
                        className="text-xs font-medium"
                        style={{ color: '#EAECEF' }}
                      >
                        {ts(
                          indicator[label as keyof typeof indicator],
                          language
                        )}
                      </span>
                    </div>
                    <input
                      type="checkbox"
                      checked={
                        (config[key as keyof IndicatorConfig] as boolean) ||
                        false
                      }
                      onChange={(e) =>
                        !disabled &&
                        onChange({ ...config, [key]: e.target.checked })
                      }
                      disabled={disabled}
                      className="w-4 h-4 rounded accent-yellow-500"
                    />
                  </div>
                  <p className="text-[10px]" style={{ color: '#5E6673' }}>
                    {ts(indicator[desc as keyof typeof indicator], language)}
                  </p>
                </div>
              ))}
          </div>
        </div>
      </div>
    </div>
  )
}
