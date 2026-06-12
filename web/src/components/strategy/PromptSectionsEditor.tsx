import { useState } from 'react'
import { ChevronDown, ChevronRight, RotateCcw, FileText } from 'lucide-react'
import type { PromptSectionsConfig } from '../../types'
import {
  promptSections as promptSectionsI18n,
  ts,
} from '../../i18n/strategy-translations'

interface PromptSectionsEditorProps {
  config: PromptSectionsConfig | undefined
  onChange: (config: PromptSectionsConfig) => void
  disabled?: boolean
  language: string
  // CME futures (e.g. MNQ): the Role + Decision DEFAULTS shown for an empty box
  // become the instrument-aware CME role (matching the prompt builder's futures
  // fallback), instead of the crypto/neutral text.
  isFutures?: boolean
}

// Role + Decision defaults shown when a box is EMPTY, on the futures path —
// mirrors the quality of engine_prompt_futures.go's fixed CME role. (A user's
// typed box always overrides; this is the default/placeholder only.)
const futuresRoleDecision: Pick<
  PromptSectionsConfig,
  'role_definition' | 'decision_process'
> = {
  role_definition: `# You are a professional CME index-futures trading AI

You trade CME index futures (e.g. MNQ / ES) with disciplined technical analysis and risk management. You size by contract count and tick-aligned stops — NOT crypto leverage.`,
  decision_process: `# Decision Process

1. If a position is open → hold, or close (close_long/close_short) for profit/stop?
2. If flat → do the multi-timeframe bars + indicators show a high-confidence setup?
3. Write your reasoning first, then output the structured JSON.`,
}

// Default prompt sections (same as backend defaults)
const defaultSections: PromptSectionsConfig = {
  role_definition: `# 你是专业的交易AI

你专注于技术分析和风险管理，基于市场数据做出理性的交易决策。
你的目标是在控制风险的前提下，捕捉高概率的交易机会。`,

  trading_frequency: `# ⏱️ 交易频率认知

- 优秀交易员：每天2-4笔 ≈ 每小时0.1-0.2笔
- 每小时>2笔 = 过度交易
- 单笔持仓时间≥30-60分钟
如果你发现自己每个周期都在交易 → 标准过低；若持仓<30分钟就平仓 → 过于急躁。`,

  entry_standards: `# 🎯 开仓标准（严格）

只在多重信号共振时开仓：
- 趋势方向明确（EMA排列、价格位置）
- 动量确认（MACD、RSI协同）
- 波动率适中（ATR合理范围）
- 量价配合（成交量支持方向）

避免：单一指标、信号矛盾、横盘震荡、刚平仓即重启。`,

  decision_process: `# 📋 决策流程

1. 检查持仓 → 是否该止盈/止损
2. 扫描市场 + 多时间框 → 是否存在强信号
3. 评估风险回报比 → 是否满足最小要求
4. 先写思维链，再输出结构化JSON`,
}

export function PromptSectionsEditor({
  config,
  onChange,
  disabled,
  language,
  isFutures = false,
}: PromptSectionsEditorProps) {
  // Market-aware defaults: futures shows the CME Role/Decision; crypto keeps the
  // neutral text. Frequency + Entry are market-neutral (shared).
  const defaults: PromptSectionsConfig = isFutures
    ? { ...defaultSections, ...futuresRoleDecision }
    : defaultSections
  // Boxes start COLLAPSED — click a header to expand. (When expanded, the
  // textarea auto-grows to fit the full text.)
  const [expandedSections, setExpandedSections] = useState<
    Record<string, boolean>
  >({
    role_definition: false,
    trading_frequency: false,
    entry_standards: false,
    decision_process: false,
  })

  const sections = [
    {
      key: 'role_definition',
      label: ts(promptSectionsI18n.roleDefinition, language),
      desc: ts(promptSectionsI18n.roleDefinitionDesc, language),
    },
    {
      key: 'trading_frequency',
      label: ts(promptSectionsI18n.tradingFrequency, language),
      desc: ts(promptSectionsI18n.tradingFrequencyDesc, language),
    },
    {
      key: 'entry_standards',
      label: ts(promptSectionsI18n.entryStandards, language),
      desc: ts(promptSectionsI18n.entryStandardsDesc, language),
    },
    {
      key: 'decision_process',
      label: ts(promptSectionsI18n.decisionProcess, language),
      desc: ts(promptSectionsI18n.decisionProcessDesc, language),
    },
  ]

  const currentConfig = config || {}

  const updateSection = (key: keyof PromptSectionsConfig, value: string) => {
    if (!disabled) {
      onChange({ ...currentConfig, [key]: value })
    }
  }

  const resetSection = (key: keyof PromptSectionsConfig) => {
    if (!disabled) {
      onChange({ ...currentConfig, [key]: defaults[key] })
    }
  }

  const toggleSection = (key: string) => {
    setExpandedSections((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const getValue = (key: keyof PromptSectionsConfig): string => {
    return currentConfig[key] || defaults[key] || ''
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2 mb-4">
        <FileText className="w-5 h-5 mt-0.5" style={{ color: '#a855f7' }} />
        <div>
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {ts(promptSectionsI18n.promptSections, language)}
          </h3>
          <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
            {ts(promptSectionsI18n.promptSectionsDesc, language)}
          </p>
        </div>
      </div>

      <div className="space-y-2">
        {sections.map(({ key, label, desc }) => {
          const sectionKey = key as keyof PromptSectionsConfig
          const isExpanded = expandedSections[key]
          const value = getValue(sectionKey)
          const isModified =
            currentConfig[sectionKey] !== undefined &&
            currentConfig[sectionKey] !== defaults[sectionKey]

          return (
            <div
              key={key}
              className="rounded-lg overflow-hidden"
              style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
            >
              <button
                onClick={() => toggleSection(key)}
                className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-white/5 transition-colors text-left"
              >
                <div className="flex items-center gap-2">
                  {isExpanded ? (
                    <ChevronDown
                      className="w-4 h-4"
                      style={{ color: '#848E9C' }}
                    />
                  ) : (
                    <ChevronRight
                      className="w-4 h-4"
                      style={{ color: '#848E9C' }}
                    />
                  )}
                  <span
                    className="text-sm font-medium"
                    style={{ color: '#EAECEF' }}
                  >
                    {label}
                  </span>
                  {isModified && (
                    <span
                      className="px-1.5 py-0.5 text-[10px] rounded"
                      style={{
                        background: 'rgba(168, 85, 247, 0.15)',
                        color: '#a855f7',
                      }}
                    >
                      {ts(promptSectionsI18n.modified, language)}
                    </span>
                  )}
                </div>
                <span className="text-[10px]" style={{ color: '#848E9C' }}>
                  {value.length} {ts(promptSectionsI18n.chars, language)}
                </span>
              </button>

              {isExpanded && (
                <div className="px-3 pb-3">
                  <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
                    {desc}
                  </p>
                  <textarea
                    ref={(el) => {
                      // Auto-grow to fit the full content (no inner scrollbar /
                      // clipping) — height follows scrollHeight, floored at 120px.
                      if (el) {
                        el.style.height = 'auto'
                        el.style.height = `${el.scrollHeight}px`
                      }
                    }}
                    value={value}
                    onChange={(e) => {
                      e.target.style.height = 'auto'
                      e.target.style.height = `${e.target.scrollHeight}px`
                      updateSection(sectionKey, e.target.value)
                    }}
                    disabled={disabled}
                    rows={6}
                    className="w-full px-3 py-2 rounded-lg resize-y font-mono text-xs"
                    style={{
                      background: '#1E2329',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                      minHeight: '120px',
                    }}
                  />
                  <div className="flex justify-end mt-2">
                    <button
                      onClick={() => resetSection(sectionKey)}
                      disabled={disabled || !isModified}
                      className="flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors hover:bg-white/5 disabled:opacity-30"
                      style={{ color: '#848E9C' }}
                    >
                      <RotateCcw className="w-3 h-3" />
                      {ts(promptSectionsI18n.resetToDefault, language)}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
