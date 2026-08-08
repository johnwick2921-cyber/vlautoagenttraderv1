import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import {
  User,
  Cpu,
  Building2,
  MessageCircle,
  Eye,
  EyeOff,
  ChevronRight,
  Plus,
  Pencil,
  Trash2,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { api } from '../lib/api'
import { ExchangeConfigModal } from '../components/trader/ExchangeConfigModal'
import { TelegramConfigModal } from '../components/trader/TelegramConfigModal'
import { ModelConfigModal } from '../components/trader/ModelConfigModal'
import type { Exchange, AIModel } from '../types'

type Tab = 'account' | 'models' | 'exchanges' | 'telegram'

function configBadge(label: string, active: boolean) {
  return (
    <span
      className={`text-[11px] px-2 py-0.5 rounded-full ${
        active
          ? 'bg-emerald-500/10 text-emerald-300'
          : 'bg-zinc-800 text-zinc-500'
      }`}
    >
      {label}
    </span>
  )
}

export function SettingsPage() {
  const { user } = useAuth()
  const { language } = useLanguage()
  const [activeTab, setActiveTab] = useState<Tab>('account')

  // Account state
  const [newPassword, setNewPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [changingPassword, setChangingPassword] = useState(false)

  // AI Models state
  const [configuredModels, setConfiguredModels] = useState<AIModel[]>([])
  const [supportedModels, setSupportedModels] = useState<AIModel[]>([])
  const [showModelModal, setShowModelModal] = useState(false)
  const [editingModel, setEditingModel] = useState<string | null>(null)

  // "Add another entry" inline form state (keyed by provider of the row it
  // hangs off). A provider can now hold multiple rows, so this lets the user
  // add a SECOND named key without touching the existing row.
  const [addEntryProvider, setAddEntryProvider] = useState<string | null>(null)
  const [addEntryName, setAddEntryName] = useState('')
  const [addEntryKey, setAddEntryKey] = useState('')
  const [addEntrySaving, setAddEntrySaving] = useState(false)

  // Exchanges state
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [showExchangeModal, setShowExchangeModal] = useState(false)
  const [editingExchange, setEditingExchange] = useState<string | null>(null)

  // Telegram state
  const [showTelegramModal, setShowTelegramModal] = useState(false)

  const refreshModelConfigs = async () => {
    const [configs, supported] = await Promise.all([
      api.getModelConfigs(),
      api.getSupportedModels(),
    ])
    setConfiguredModels(configs)
    setSupportedModels(supported)
  }

  const refreshExchangeConfigs = async () => {
    const refreshed = await api.getExchangeConfigs()
    setExchanges(refreshed)
  }

  // Fetch data when tabs are visited
  useEffect(() => {
    if (activeTab === 'models') {
      refreshModelConfigs().catch(() => toast.error('Failed to load AI models'))
    }
    if (activeTab === 'exchanges') {
      refreshExchangeConfigs().catch(() =>
        toast.error('Failed to load exchanges')
      )
    }
  }, [activeTab])

  useEffect(() => {
    const handleRefresh = () => {
      refreshModelConfigs().catch(() => {})
      refreshExchangeConfigs().catch(() => {})
    }
    window.addEventListener('agent-config-refresh', handleRefresh)
    return () =>
      window.removeEventListener('agent-config-refresh', handleRefresh)
  }, [])

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword.length < 8) {
      toast.error('Password must be at least 8 characters')
      return
    }
    setChangingPassword(true)
    try {
      const res = await fetch('/api/user/password', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
        },
        body: JSON.stringify({ new_password: newPassword }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(data.error || 'Failed to update password')
      }
      toast.success('Password updated successfully')
      setNewPassword('')
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to update password'
      )
    } finally {
      setChangingPassword(false)
    }
  }

  const handleSaveModel = async (
    modelId: string,
    apiKey: string,
    customApiUrl?: string,
    customModelName?: string,
    name?: string
  ) => {
    try {
      const existingModel = configuredModels.find((m) => m.id === modelId)
      const modelTemplate = supportedModels.find((m) => m.id === modelId)
      const modelToUpdate = existingModel || modelTemplate
      if (!modelToUpdate) {
        toast.error('Model not found')
        return
      }

      // NEW-ENTRY GUARD (fixes "+ Add Model overwrites an existing provider"):
      // when adding via the catalog (not editing a specific row) for a provider that
      // ALREADY has a configured entry, CREATE a new named row instead of sending a
      // bare-provider key that the backend's legacy-match would use to overwrite the
      // existing entry. Wallet providers (claw402/blockrun) keep their reconfigure flow.
      const provider = modelToUpdate.provider || ''
      const isWalletProvider =
        provider === 'claw402' || provider.startsWith('blockrun')
      const providerAlreadyConfigured =
        !existingModel && configuredModels.some((m) => m.provider === provider)
      if (providerAlreadyConfigured && !isWalletProvider) {
        const sameProviderCount = configuredModels.filter(
          (m) => m.provider === provider
        ).length
        const baseName = modelToUpdate.name || provider
        await api.createModelEntry({
          provider,
          name: `${baseName} ${sameProviderCount + 1}`,
          api_key: apiKey,
          custom_api_url: customApiUrl || undefined,
          custom_model_name: customModelName || undefined,
          enabled: true,
        })
        toast.success('Model entry added')
        await refreshModelConfigs()
        setShowModelModal(false)
        setEditingModel(null)
        return
      }

      let updatedModels: AIModel[]
      if (existingModel) {
        updatedModels = configuredModels.map((m) =>
          m.id === modelId
            ? {
                ...m,
                apiKey,
                customApiUrl: customApiUrl || '',
                customModelName: customModelName || '',
                enabled: true,
                name: name?.trim() ? name.trim() : m.name,
              }
            : m
        )
      } else {
        updatedModels = [
          ...configuredModels,
          {
            ...modelToUpdate,
            apiKey,
            customApiUrl: customApiUrl || '',
            customModelName: customModelName || '',
            enabled: true,
          },
        ]
      }

      const request = {
        models: Object.fromEntries(
          updatedModels.map((m) => [
            // Key by row id so editing one entry updates exactly that row
            // (a provider can now have multiple rows). Newly-configured
            // providers whose row doesn't exist yet still send a bare-provider
            // id from the supportedModels template — the backend's scoped
            // legacy path handles bare ids.
            m.id,
            {
              name: m.name || '',
              enabled: m.enabled,
              api_key: m.apiKey || '',
              custom_api_url: m.customApiUrl || '',
              custom_model_name: m.customModelName || '',
            },
          ])
        ),
      }
      await api.updateModelConfigs(request)
      toast.success('Model config saved')
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
    } catch {
      toast.error('Failed to save model config')
    }
  }

  const handleDeleteModel = async (modelId: string) => {
    if (!window.confirm(t('confirmDeleteModel', language))) return
    try {
      // Real row delete (not a soft-blank). Backend returns 409 when a trader
      // is bound to this entry — surface that message.
      await api.deleteModelEntry(modelId)
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
      toast.success('Model config removed')
    } catch (err) {
      const msg = err instanceof Error ? err.message : ''
      // Server 409 message contains the trader-in-use reason; fall back to the
      // localized "in use by traders" string when the message is generic.
      toast.error(msg || t('cannotDeleteModelInUse', language))
    }
  }

  const openAddEntry = (provider: string) => {
    setAddEntryProvider(provider)
    setAddEntryName('')
    setAddEntryKey('')
  }

  const cancelAddEntry = () => {
    setAddEntryProvider(null)
    setAddEntryName('')
    setAddEntryKey('')
  }

  const handleCreateEntry = async () => {
    if (!addEntryProvider || !addEntryName.trim() || !addEntryKey.trim()) return
    setAddEntrySaving(true)
    try {
      await api.createModelEntry({
        provider: addEntryProvider,
        name: addEntryName.trim(),
        api_key: addEntryKey.trim(),
        enabled: true,
      })
      toast.success('Model entry added')
      cancelAddEntry()
      await refreshModelConfigs()
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to add model entry'
      )
    } finally {
      setAddEntrySaving(false)
    }
  }

  const handleSaveExchange = async (
    exchangeId: string | null,
    exchangeType: string,
    accountName: string,
    apiKey: string,
    secretKey?: string,
    passphrase?: string,
    testnet?: boolean,
    hyperliquidWalletAddr?: string,
    asterUser?: string,
    asterSigner?: string,
    asterPrivateKey?: string,
    lighterWalletAddr?: string,
    lighterPrivateKey?: string,
    lighterApiKeyPrivateKey?: string,
    lighterApiKeyIndex?: number,
    ntDataDir?: string,
    ntInstrumentName?: string,
    ntDefaultContractQty?: number
  ) => {
    try {
      if (exchangeId) {
        const request = {
          exchanges: {
            [exchangeId]: {
              enabled: true,
              api_key: apiKey || '',
              secret_key: secretKey || '',
              passphrase: passphrase || '',
              testnet: testnet || false,
              hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
              aster_user: asterUser || '',
              aster_signer: asterSigner || '',
              aster_private_key: asterPrivateKey || '',
              lighter_wallet_addr: lighterWalletAddr || '',
              lighter_private_key: lighterPrivateKey || '',
              lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
              lighter_api_key_index: lighterApiKeyIndex || 0,
              nt_data_dir: ntDataDir || '',
              nt_instrument_name: ntInstrumentName || '',
              nt_default_contract_qty: ntDefaultContractQty || 0,
            },
          },
        }
        await api.updateExchangeConfigsEncrypted(request)
        toast.success('Exchange config updated')
      } else {
        const createRequest = {
          exchange_type: exchangeType,
          account_name: accountName,
          enabled: true,
          api_key: apiKey || '',
          secret_key: secretKey || '',
          passphrase: passphrase || '',
          testnet: testnet || false,
          hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
          aster_user: asterUser || '',
          aster_signer: asterSigner || '',
          aster_private_key: asterPrivateKey || '',
          lighter_wallet_addr: lighterWalletAddr || '',
          lighter_private_key: lighterPrivateKey || '',
          lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
          lighter_api_key_index: lighterApiKeyIndex || 0,
          nt_data_dir: ntDataDir || '',
          nt_instrument_name: ntInstrumentName || '',
          nt_default_contract_qty: ntDefaultContractQty || 0,
        }
        await api.createExchangeEncrypted(createRequest)
        toast.success('Exchange account created')
      }
      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
    } catch {
      toast.error('Failed to save exchange config')
    }
  }

  const handleDeleteExchange = async (exchangeId: string) => {
    try {
      await api.deleteExchange(exchangeId)
      toast.success('Exchange account deleted')
      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
    } catch {
      toast.error('Failed to delete exchange account')
    }
  }

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    { key: 'account', label: 'Account', icon: <User size={16} /> },
    { key: 'models', label: 'AI Models', icon: <Cpu size={16} /> },
    { key: 'exchanges', label: 'Exchanges', icon: <Building2 size={16} /> },
    { key: 'telegram', label: 'Telegram', icon: <MessageCircle size={16} /> },
  ]

  return (
    <div
      className="min-h-screen pt-20 pb-12 px-4"
      style={{ background: '#0B0E11' }}
    >
      <div className="max-w-2xl mx-auto">
        <h1 className="text-xl font-bold text-white mb-6">Settings</h1>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 bg-zinc-900/60 border border-zinc-800 rounded-xl p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all
                ${
                  activeTab === tab.key
                    ? 'bg-nofx-gold text-black'
                    : 'text-zinc-400 hover:text-white'
                }`}
            >
              {tab.icon}
              <span className="hidden sm:inline">{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Tab Content */}
        <div className="bg-zinc-900/60 backdrop-blur-xl border border-zinc-800/80 rounded-2xl p-6">
          {/* Account Tab */}
          {activeTab === 'account' && (
            <div className="space-y-6">
              <div>
                <p className="text-xs text-zinc-500 mb-1">Email</p>
                <p className="text-sm text-white font-medium">{user?.email}</p>
              </div>

              <div className="border-t border-zinc-800 pt-6">
                <h3 className="text-sm font-semibold text-white mb-4">
                  Change Password
                </h3>
                <form onSubmit={handleChangePassword} className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-zinc-400 mb-2">
                      New Password
                    </label>
                    <div className="relative">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        className="w-full bg-zinc-950/80 border border-zinc-700/80 rounded-xl px-4 py-3 pr-11 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-nofx-gold/60 focus:ring-1 focus:ring-nofx-gold/30 transition-all"
                        placeholder="At least 8 characters"
                        required
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300 transition-colors"
                      >
                        {showPassword ? (
                          <EyeOff size={16} />
                        ) : (
                          <Eye size={16} />
                        )}
                      </button>
                    </div>
                  </div>
                  <button
                    type="submit"
                    disabled={changingPassword || newPassword.length < 8}
                    className="w-full bg-nofx-gold hover:bg-yellow-400 active:scale-[0.98] text-black font-semibold py-3 rounded-xl text-sm transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {changingPassword ? 'Updating...' : 'Update Password'}
                  </button>
                </form>
              </div>
            </div>
          )}

          {/* AI Models Tab */}
          {activeTab === 'models' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-zinc-400">
                  {configuredModels.length} model
                  {configuredModels.length !== 1 ? 's' : ''} configured
                </p>
                <button
                  onClick={() => {
                    setEditingModel(null)
                    setShowModelModal(true)
                  }}
                  className="flex items-center gap-1.5 text-xs font-medium bg-nofx-gold/10 hover:bg-nofx-gold/20 text-nofx-gold px-3 py-1.5 rounded-lg transition-colors"
                >
                  <Plus size={14} />
                  Add Model
                </button>
              </div>

              {configuredModels.length > 0 && (
                <p className="text-xs text-zinc-500 -mt-2">
                  {t('multiKeyHint', language)}
                </p>
              )}

              {configuredModels.length === 0 ? (
                <div className="text-center py-8 text-zinc-600 text-sm">
                  No AI models configured yet
                </div>
              ) : (
                <div className="space-y-2">
                  {configuredModels.map((model) => (
                    <div key={model.id}>
                      <div className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-zinc-800/50 hover:bg-zinc-800 border border-zinc-700/50 transition-colors group">
                        {/* Clickable edit region */}
                        <button
                          onClick={() => {
                            setEditingModel(model.id)
                            setShowModelModal(true)
                          }}
                          className="flex items-center gap-3 flex-1 min-w-0 text-left"
                        >
                          <div className="w-8 h-8 rounded-lg bg-zinc-700 flex items-center justify-center shrink-0">
                            <Cpu size={14} className="text-zinc-300" />
                          </div>
                          <div className="text-left min-w-0">
                            <p className="text-sm font-medium text-white truncate">
                              {model.name}
                            </p>
                            <div className="flex flex-wrap items-center gap-1.5 mt-1">
                              <p className="text-xs text-zinc-500">
                                {model.provider}
                              </p>
                              {configBadge('API Key', !!model.has_api_key)}
                              {model.customModelName
                                ? configBadge('Custom Model', true)
                                : null}
                              {model.customApiUrl
                                ? configBadge('Base URL', true)
                                : null}
                            </div>
                          </div>
                        </button>
                        <div className="flex items-center gap-2 shrink-0 ml-2">
                          <span
                            className={`text-xs px-2 py-0.5 rounded-full ${model.enabled ? 'bg-emerald-500/10 text-emerald-400' : 'bg-zinc-700 text-zinc-500'}`}
                          >
                            {model.enabled ? 'Active' : 'Inactive'}
                          </span>
                          <button
                            onClick={() => openAddEntry(model.provider)}
                            title={t('addModelEntry', language)}
                            className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium text-nofx-gold border border-nofx-gold/40 hover:bg-nofx-gold/10 transition-colors shrink-0"
                          >
                            <Plus size={12} />
                            {t('addKeyShort', language)}
                          </button>
                          <button
                            onClick={() => handleDeleteModel(model.id)}
                            title={t('deleteModelEntry', language)}
                            className="p-1.5 rounded-lg text-zinc-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                          >
                            <Trash2 size={14} />
                          </button>
                          <Pencil
                            size={14}
                            className="text-zinc-600 group-hover:text-zinc-400 transition-colors"
                          />
                        </div>
                      </div>

                      {/* Inline "add another entry" form for this provider */}
                      {addEntryProvider === model.provider && (
                        <div className="mt-2 mb-1 mx-1 p-3 rounded-xl bg-zinc-900/70 border border-zinc-700/60 space-y-2">
                          <p className="text-xs text-zinc-400">
                            {t('addModelEntry', language)} — {model.provider}
                          </p>
                          <input
                            type="text"
                            value={addEntryName}
                            onChange={(e) => setAddEntryName(e.target.value)}
                            placeholder={t('entryName', language)}
                            className="w-full bg-zinc-950/80 border border-zinc-700/80 rounded-lg px-3 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-nofx-gold/60"
                          />
                          <input
                            type="password"
                            value={addEntryKey}
                            onChange={(e) => setAddEntryKey(e.target.value)}
                            placeholder={t('enterAPIKey', language)}
                            className="w-full bg-zinc-950/80 border border-zinc-700/80 rounded-lg px-3 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-nofx-gold/60"
                          />
                          <div className="flex justify-end gap-2 pt-1">
                            <button
                              onClick={cancelAddEntry}
                              className="px-3 py-1.5 text-xs rounded-lg bg-zinc-800 text-zinc-300 hover:bg-zinc-700 transition-colors"
                            >
                              {t('cancel', language)}
                            </button>
                            <button
                              onClick={handleCreateEntry}
                              disabled={
                                addEntrySaving ||
                                !addEntryName.trim() ||
                                !addEntryKey.trim()
                              }
                              className="px-3 py-1.5 text-xs rounded-lg bg-nofx-gold text-black font-medium hover:bg-yellow-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                              {addEntrySaving
                                ? t('saving', language)
                                : t('addModelEntry', language)}
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Exchanges Tab */}
          {activeTab === 'exchanges' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-zinc-400">
                  {exchanges.length} account{exchanges.length !== 1 ? 's' : ''}{' '}
                  connected
                </p>
                <button
                  onClick={() => {
                    setEditingExchange(null)
                    setShowExchangeModal(true)
                  }}
                  className="flex items-center gap-1.5 text-xs font-medium bg-nofx-gold/10 hover:bg-nofx-gold/20 text-nofx-gold px-3 py-1.5 rounded-lg transition-colors"
                >
                  <Plus size={14} />
                  Add Exchange
                </button>
              </div>

              {exchanges.length === 0 ? (
                <div className="text-center py-8 text-zinc-600 text-sm">
                  No exchange accounts connected yet
                </div>
              ) : (
                <div className="space-y-2">
                  {exchanges.map((exchange) => {
                    const isNinjaTrader =
                      (exchange.exchange_type || exchange.type) ===
                      'ninjatrader'
                    return (
                      <button
                        key={exchange.id}
                        onClick={() => {
                          setEditingExchange(exchange.id)
                          setShowExchangeModal(true)
                        }}
                        className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-zinc-800/50 hover:bg-zinc-800 border border-zinc-700/50 transition-colors group"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-lg bg-zinc-700 flex items-center justify-center">
                            <Building2 size={14} className="text-zinc-300" />
                          </div>
                          <div className="text-left">
                            <p className="text-sm font-medium text-white">
                              {exchange.account_name || exchange.name}
                            </p>
                            <div className="flex flex-wrap items-center gap-1.5 mt-1">
                              <p className="text-xs text-zinc-500 capitalize">
                                {exchange.exchange_type || exchange.type}
                              </p>
                              {isNinjaTrader ? (
                                configBadge('TCP Bridge', true)
                              ) : (
                                <>
                                  {configBadge(
                                    'API Key',
                                    !!exchange.has_api_key
                                  )}
                                  {configBadge(
                                    'Secret',
                                    !!exchange.has_secret_key
                                  )}
                                  {exchange.has_passphrase
                                    ? configBadge('Passphrase', true)
                                    : null}
                                  {exchange.hyperliquidWalletAddr
                                    ? configBadge('Wallet', true)
                                    : null}
                                  {exchange.has_aster_private_key
                                    ? configBadge('Aster Key', true)
                                    : null}
                                  {exchange.has_lighter_private_key ||
                                  exchange.has_lighter_api_key_private_key
                                    ? configBadge('Lighter Key', true)
                                    : null}
                                </>
                              )}
                            </div>
                          </div>
                        </div>
                        <ChevronRight
                          size={14}
                          className="text-zinc-600 group-hover:text-zinc-400 transition-colors"
                        />
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          )}

          {/* Telegram Tab */}
          {activeTab === 'telegram' && (
            <div className="space-y-4">
              <p className="text-sm text-zinc-400">
                Connect a Telegram bot to receive trading notifications and
                interact with your traders.
              </p>
              <button
                onClick={() => setShowTelegramModal(true)}
                className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-zinc-800/50 hover:bg-zinc-800 border border-zinc-700/50 transition-colors group"
              >
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-[#0088cc]/20 flex items-center justify-center">
                    <MessageCircle size={14} className="text-[#0088cc]" />
                  </div>
                  <span className="text-sm font-medium text-white">
                    Configure Telegram Bot
                  </span>
                </div>
                <ChevronRight
                  size={14}
                  className="text-zinc-600 group-hover:text-zinc-400 transition-colors"
                />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* AI Model Modal */}
      {showModelModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm px-4">
          <ModelConfigModal
            allModels={supportedModels}
            configuredModels={configuredModels}
            editingModelId={editingModel}
            onSave={handleSaveModel}
            onDelete={handleDeleteModel}
            onClose={() => {
              setShowModelModal(false)
              setEditingModel(null)
            }}
            language={language}
          />
        </div>
      )}

      {/* Exchange Modal */}
      {showExchangeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm px-4">
          <ExchangeConfigModal
            allExchanges={exchanges}
            editingExchangeId={editingExchange}
            onSave={handleSaveExchange}
            onDelete={handleDeleteExchange}
            onClose={() => {
              setShowExchangeModal(false)
              setEditingExchange(null)
            }}
            language={language}
          />
        </div>
      )}

      {/* Telegram Modal */}
      {showTelegramModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm px-4">
          <TelegramConfigModal
            onClose={() => setShowTelegramModal(false)}
            language={language}
          />
        </div>
      )}
    </div>
  )
}
