import React, { useState, useEffect } from 'react'
import type { Exchange } from '../../types'
import { t, type Language } from '../../i18n/translations'
import { getExchangeIcon } from '../common/ExchangeIcons'
import {
  WebCryptoEnvironmentCheck,
  type WebCryptoCheckStatus,
} from '../common/WebCryptoEnvironmentCheck'
import {
  Trash2,
  Key,
  Shield,
  ChevronLeft,
  Check,
  ArrowRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { getShortName } from './utils'

// Supported exchange templates
const SUPPORTED_EXCHANGE_TEMPLATES: {
  exchange_type: string
  name: string
  type: 'cex' | 'dex' | 'futures'
}[] = [
  {
    exchange_type: 'ninjatrader',
    name: 'NinjaTrader',
    type: 'futures' as const,
  },
]

interface ExchangeConfigModalProps {
  allExchanges: Exchange[]
  editingExchangeId: string | null
  onSave: (
    exchangeId: string | null,
    exchangeType: string,
    accountName: string,
    apiKey: string,
    secretKey?: string,
    passphrase?: string,
    testnet?: boolean,
    ntDataDir?: string,
    ntInstrumentName?: string,
    ntDefaultContractQty?: number
  ) => Promise<void>
  onDelete: (exchangeId: string) => void
  onClose: () => void
  language: Language
}

// Step indicator component
function StepIndicator({
  currentStep,
  labels,
}: {
  currentStep: number
  labels: string[]
}) {
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {labels.map((label, index) => (
        <React.Fragment key={index}>
          <div className="flex items-center gap-2">
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all"
              style={{
                background:
                  index < currentStep
                    ? '#0ECB81'
                    : index === currentStep
                      ? '#F0B90B'
                      : '#2B3139',
                color: index <= currentStep ? '#000' : '#848E9C',
              }}
            >
              {index < currentStep ? <Check className="w-4 h-4" /> : index + 1}
            </div>
            <span
              className="text-xs font-medium hidden sm:block"
              style={{ color: index === currentStep ? '#EAECEF' : '#848E9C' }}
            >
              {label}
            </span>
          </div>
          {index < labels.length - 1 && (
            <div
              className="w-8 h-0.5 mx-1"
              style={{
                background: index < currentStep ? '#0ECB81' : '#2B3139',
              }}
            />
          )}
        </React.Fragment>
      ))}
    </div>
  )
}

// Exchange card component
function ExchangeCard({
  template,
  selected,
  onClick,
  disabled,
}: {
  template: (typeof SUPPORTED_EXCHANGE_TEMPLATES)[0]
  selected: boolean
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      data-testid={`exchange-option-${template.exchange_type}`}
      className="flex flex-col items-center gap-2 p-4 rounded-xl transition-all hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
      style={{
        background: selected ? 'rgba(240, 185, 11, 0.15)' : '#0B0E11',
        border: selected ? '2px solid #F0B90B' : '2px solid #2B3139',
      }}
    >
      <div className="relative">
        {getExchangeIcon(template.exchange_type, { width: 48, height: 48 })}
        {selected && (
          <div
            className="absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center"
            style={{ background: '#0ECB81' }}
          >
            <Check className="w-3 h-3 text-black" />
          </div>
        )}
      </div>
      <span className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
        {getShortName(template.name)}
      </span>
      <span
        className="text-xs px-2 py-0.5 rounded-full"
        style={{
          background:
            template.type === 'cex'
              ? 'rgba(240, 185, 11, 0.2)'
              : template.type === 'dex'
                ? 'rgba(139, 92, 246, 0.2)'
                : 'rgba(14, 203, 129, 0.2)',
          color:
            template.type === 'cex'
              ? '#F0B90B'
              : template.type === 'dex'
                ? '#A78BFA'
                : '#0ECB81',
        }}
      >
        {template.type.toUpperCase()}
      </span>
    </button>
  )
}

export function ExchangeConfigModal({
  allExchanges,
  editingExchangeId,
  onSave,
  onDelete,
  onClose,
  language,
}: ExchangeConfigModalProps) {
  // Step: 0 = select exchange, 1 = configure
  const [currentStep, setCurrentStep] = useState(editingExchangeId ? 1 : 0)
  const [selectedExchangeType, setSelectedExchangeType] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [testnet, setTestnet] = useState(false)
  const [webCryptoStatus, setWebCryptoStatus] =
    useState<WebCryptoCheckStatus>('idle')
  // NinjaTrader fields
  const [ntDataDir, setNtDataDir] = useState('')
  const [ntInstrumentName, setNtInstrumentName] = useState('MNQ')
  const [ntDefaultContractQty, setNtDefaultContractQty] = useState(1)

  // Other state
  const [isSaving, setIsSaving] = useState(false)
  const [accountName, setAccountName] = useState('')

  const selectedExchange = editingExchangeId
    ? allExchanges?.find((e) => e.id === editingExchangeId)
    : null

  const selectedTemplate = editingExchangeId
    ? SUPPORTED_EXCHANGE_TEMPLATES.find(
        (t) => t.exchange_type === selectedExchange?.exchange_type
      )
    : SUPPORTED_EXCHANGE_TEMPLATES.find(
        (t) => t.exchange_type === selectedExchangeType
      )

  const currentExchangeType = editingExchangeId
    ? selectedExchange?.exchange_type
    : selectedExchangeType

  // Initialize form when editing
  useEffect(() => {
    if (editingExchangeId && selectedExchange) {
      setAccountName(selectedExchange.account_name || '')
      setApiKey(selectedExchange.apiKey || '')
      setSecretKey(selectedExchange.secretKey || '')
      setTestnet(selectedExchange.testnet || false)
    }
  }, [editingExchangeId, selectedExchange])

  const handleSelectExchange = (exchangeType: string) => {
    setSelectedExchangeType(exchangeType)
    setCurrentStep(1)
  }

  const handleBack = () => {
    if (editingExchangeId) {
      onClose()
    } else {
      setCurrentStep(0)
      setSelectedExchangeType('')
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isSaving) return
    if (!editingExchangeId && !selectedExchangeType) return

    const trimmedAccountName = accountName.trim()
    if (!trimmedAccountName) {
      toast.error(t('exchangeConfig.pleaseEnterAccountName', language))
      return
    }

    const exchangeId = editingExchangeId || null
    const exchangeType = currentExchangeType || ''

    setIsSaving(true)
    try {
      if (currentExchangeType === 'ninjatrader') {
        if (!ntDataDir.trim()) {
          toast.error('NT Data Directory is required')
          return
        }
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          '',
          '',
          '',
          testnet,
          ntDataDir.trim(),
          ntInstrumentName.trim() || 'MNQ',
          ntDefaultContractQty || 1
        )
      } else {
        if (!apiKey.trim() || !secretKey.trim()) return
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          apiKey.trim(),
          secretKey.trim(),
          '',
          testnet
        )
      }
    } finally {
      setIsSaving(false)
    }
  }

  const stepLabels = [
    t('exchangeConfig.selectExchange', language),
    t('exchangeConfig.configure', language),
  ]
  const cexExchanges = SUPPORTED_EXCHANGE_TEMPLATES.filter(
    (t) => t.type === 'cex'
  )
  const dexExchanges = SUPPORTED_EXCHANGE_TEMPLATES.filter(
    (t) => t.type === 'dex'
  )
  const futuresExchanges = SUPPORTED_EXCHANGE_TEMPLATES.filter(
    (t) => t.type === 'futures'
  )

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4 overflow-y-auto backdrop-blur-sm">
      <div
        className="rounded-2xl w-full max-w-2xl relative my-8 shadow-2xl"
        style={{
          background: 'linear-gradient(180deg, #1E2329 0%, #181A20 100%)',
          maxHeight: 'calc(100vh - 4rem)',
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 pb-2">
          <div className="flex items-center gap-3">
            {currentStep > 0 && !editingExchangeId && (
              <button
                type="button"
                onClick={handleBack}
                className="p-2 rounded-lg hover:bg-white/10 transition-colors"
              >
                <ChevronLeft className="w-5 h-5" style={{ color: '#848E9C' }} />
              </button>
            )}
            <h3 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
              {editingExchangeId
                ? t('editExchange', language)
                : t('addExchange', language)}
            </h3>
          </div>
          <div className="flex items-center gap-2">
            {editingExchangeId && (
              <button
                type="button"
                onClick={() => onDelete(editingExchangeId)}
                className="p-2 rounded-lg hover:bg-red-500/20 transition-colors"
                style={{ color: '#F6465D' }}
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-white/10 transition-colors"
              style={{ color: '#848E9C' }}
            >
              ✕
            </button>
          </div>
        </div>

        {/* Step Indicator */}
        {!editingExchangeId && (
          <div className="px-6">
            <StepIndicator currentStep={currentStep} labels={stepLabels} />
          </div>
        )}

        {/* Content */}
        <div
          className="px-6 pb-6 overflow-y-auto"
          style={{ maxHeight: 'calc(100vh - 16rem)' }}
        >
          {/* Step 0: Select Exchange */}
          {currentStep === 0 && !editingExchangeId && (
            <div className="space-y-6">
              {/* WebCrypto Check */}
              <div className="space-y-2">
                <div
                  className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: '#848E9C' }}
                >
                  <Shield className="w-4 h-4" />
                  {t('environmentSteps.checkTitle', language)}
                </div>
                <WebCryptoEnvironmentCheck
                  language={language}
                  variant="card"
                  onStatusChange={setWebCryptoStatus}
                />
              </div>

              {/* Exchange Grid */}
              <div className="space-y-4">
                <div
                  className="text-sm font-semibold"
                  style={{ color: '#EAECEF' }}
                >
                  {t('exchangeConfig.chooseExchange', language)}
                </div>

                {/* CEX */}
                <div className="space-y-3">
                  <div
                    className="text-xs font-medium uppercase tracking-wide"
                    style={{ color: '#F0B90B' }}
                  >
                    {t('exchangeConfig.centralizedExchanges', language)}
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
                    {cexExchanges.map((template) => (
                      <ExchangeCard
                        key={template.exchange_type}
                        template={template}
                        selected={
                          selectedExchangeType === template.exchange_type
                        }
                        onClick={() =>
                          handleSelectExchange(template.exchange_type)
                        }
                        disabled={
                          webCryptoStatus !== 'secure' &&
                          webCryptoStatus !== 'disabled'
                        }
                      />
                    ))}
                  </div>
                </div>

                {/* DEX */}
                <div className="space-y-3">
                  <div
                    className="text-xs font-medium uppercase tracking-wide"
                    style={{ color: '#A78BFA' }}
                  >
                    {t('exchangeConfig.decentralizedExchanges', language)}
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
                    {dexExchanges.map((template) => (
                      <ExchangeCard
                        key={template.exchange_type}
                        template={template}
                        selected={
                          selectedExchangeType === template.exchange_type
                        }
                        onClick={() =>
                          handleSelectExchange(template.exchange_type)
                        }
                        disabled={
                          webCryptoStatus !== 'secure' &&
                          webCryptoStatus !== 'disabled'
                        }
                      />
                    ))}
                  </div>
                </div>

                {/* Futures (CME via NinjaTrader) */}
                {futuresExchanges.length > 0 && (
                  <div className="space-y-3">
                    <div
                      className="text-xs font-medium uppercase tracking-wide"
                      style={{ color: '#0ECB81' }}
                    >
                      Futures (CME)
                    </div>
                    <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
                      {futuresExchanges.map((template) => (
                        <ExchangeCard
                          key={template.exchange_type}
                          template={template}
                          selected={
                            selectedExchangeType === template.exchange_type
                          }
                          onClick={() =>
                            handleSelectExchange(template.exchange_type)
                          }
                          disabled={
                            webCryptoStatus !== 'secure' &&
                            webCryptoStatus !== 'disabled'
                          }
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Step 1: Configure */}
          {(currentStep === 1 || editingExchangeId) && selectedTemplate && (
            <form onSubmit={handleSubmit} className="space-y-5">
              {/* Selected Exchange Header */}
              <div
                className="p-4 rounded-xl flex items-center gap-4"
                style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
              >
                {getExchangeIcon(selectedTemplate.exchange_type, {
                  width: 48,
                  height: 48,
                })}
                <div className="flex-1">
                  <div
                    className="font-semibold text-lg"
                    style={{ color: '#EAECEF' }}
                  >
                    {getShortName(selectedTemplate.name)}
                  </div>
                  <div className="text-xs" style={{ color: '#848E9C' }}>
                    {selectedTemplate.type.toUpperCase()} •{' '}
                    {selectedTemplate.exchange_type}
                  </div>
                </div>
              </div>

              {/* Account Name */}
              <div className="space-y-2">
                <label
                  className="flex items-center gap-2 text-sm font-semibold"
                  style={{ color: '#EAECEF' }}
                >
                  <Key className="w-4 h-4" style={{ color: '#F0B90B' }} />
                  {t('exchangeConfig.accountName', language)} *
                </label>
                <input
                  type="text"
                  value={accountName}
                  onChange={(e) => setAccountName(e.target.value)}
                  placeholder={t(
                    'exchangeConfig.accountNamePlaceholder',
                    language
                  )}
                  className="w-full px-4 py-3 rounded-xl text-base"
                  style={{
                    background: '#0B0E11',
                    border: '1px solid #2B3139',
                    color: '#EAECEF',
                  }}
                  required
                />
              </div>

              {currentExchangeType === 'ninjatrader' && (
                <>
                  <div
                    className="p-4 rounded-xl"
                    style={{
                      background: 'rgba(14, 203, 129, 0.1)',
                      border: '1px solid rgba(14, 203, 129, 0.3)',
                    }}
                  >
                    <div className="flex items-start gap-2">
                      <span style={{ fontSize: '16px' }}>📈</span>
                      <div>
                        <div
                          className="text-sm font-semibold mb-1"
                          style={{ color: '#0ECB81' }}
                        >
                          NinjaTrader CSV Bridge
                        </div>
                        <div className="text-xs" style={{ color: '#848E9C' }}>
                          CME futures (NQ/MNQ/ES/MES) via NT8 file bridge. No
                          API key needed — point to the WSL view of the NT data
                          directory.
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: '#EAECEF' }}
                    >
                      <Key className="w-4 h-4" style={{ color: '#0ECB81' }} />
                      NT Data Directory (WSL path) *
                    </label>
                    <input
                      type="text"
                      data-testid="ninjatrader-data-directory"
                      value={ntDataDir}
                      onChange={(e) => setNtDataDir(e.target.value)}
                      placeholder="/mnt/c/Users/<u>/VLTrader/data"
                      className="w-full px-4 py-3 rounded-xl font-mono text-sm"
                      style={{
                        background: '#0B0E11',
                        border: '1px solid #2B3139',
                        color: '#EAECEF',
                      }}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: '#EAECEF' }}
                    >
                      Instrument
                    </label>
                    <input
                      type="text"
                      data-testid="ninjatrader-instrument"
                      value={ntInstrumentName}
                      onChange={(e) => setNtInstrumentName(e.target.value)}
                      placeholder="MNQ or MNQ,ES,NQ"
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: '#0B0E11',
                        border: '1px solid #2B3139',
                        color: '#EAECEF',
                      }}
                    />
                    <p className="text-xs" style={{ color: '#848E9C' }}>
                      {/* P5.2 — comma list: first = TRADING symbol; the rest are
                          extra DATA feeds (bars only). A single symbol behaves
                          exactly as before. */}
                      First symbol = trading instrument; extras
                      (comma-separated) stream chart/data bars only.
                    </p>
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: '#EAECEF' }}
                    >
                      Default Contract Quantity
                    </label>
                    <input
                      type="number"
                      data-testid="ninjatrader-contract-qty"
                      min={1}
                      value={ntDefaultContractQty}
                      onChange={(e) =>
                        setNtDefaultContractQty(parseInt(e.target.value) || 1)
                      }
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: '#0B0E11',
                        border: '1px solid #2B3139',
                        color: '#EAECEF',
                      }}
                    />
                  </div>
                </>
              )}

              {/* Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={handleBack}
                  className="flex-1 px-4 py-3 rounded-xl text-sm font-semibold transition-all hover:bg-white/5"
                  style={{ background: '#2B3139', color: '#848E9C' }}
                >
                  {editingExchangeId
                    ? t('cancel', language)
                    : t('exchangeConfig.back', language)}
                </button>
                <button
                  type="submit"
                  data-testid={
                    currentExchangeType === 'ninjatrader'
                      ? 'ninjatrader-submit'
                      : 'exchange-submit'
                  }
                  disabled={isSaving || !accountName.trim()}
                  className="flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-xl text-sm font-bold transition-all hover:scale-[1.02] disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{ background: '#F0B90B', color: '#000' }}
                >
                  {isSaving ? (
                    t('saving', language)
                  ) : (
                    <>
                      {t('saveConfig', language)}{' '}
                      <ArrowRight className="w-4 h-4" />
                    </>
                  )}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
