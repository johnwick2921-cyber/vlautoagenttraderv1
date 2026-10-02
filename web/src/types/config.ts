export interface AIModel {
  id: string
  name: string
  provider: string
  enabled: boolean
  has_api_key?: boolean
  apiKey?: string
  customApiUrl?: string
  customModelName?: string
  thinkingMode?: string
  reasoningEffort?: string
}

export interface TelegramConfig {
  token_masked: string // Masked token like "123456:ABC***XYZ"
  is_bound: boolean // Whether a user has sent /start
  bound_chat_id?: number // The bound chat ID (if any)
  model_id?: string // AI model selected for Telegram replies
}

export interface Exchange {
  id: string // UUID (empty for supported exchange templates)
  exchange_type: string
  account_name: string // User-defined account name
  name: string // Display name
  enabled: boolean
  has_api_key?: boolean
  has_secret_key?: boolean
  has_passphrase?: boolean
  apiKey?: string
  secretKey?: string
  passphrase?: string
  testnet?: boolean
}

export type ExchangeAccountStatus =
  | 'ok'
  | 'disabled'
  | 'missing_credentials'
  | 'invalid_credentials'
  | 'permission_denied'
  | 'unavailable'

export interface ExchangeAccountState {
  exchange_id: string
  status: ExchangeAccountStatus
  display_balance?: string
  asset?: string
  total_equity?: number
  available_balance?: number
  checked_at: string
  error_code?: string
  error_message?: string
}

export interface ExchangeAccountStateResponse {
  states: Record<string, ExchangeAccountState>
}

export interface CreateExchangeRequest {
  exchange_type: string
  account_name: string // User-defined account name
  enabled: boolean
  api_key?: string
  secret_key?: string
  passphrase?: string
  testnet?: boolean
}

export interface CreateTraderRequest {
  name: string
  ai_model_id: string
  exchange_id: string
  strategy_id?: string // 策略ID（新版，使用保存的策略配置）
  scan_interval_minutes?: number
  cadence_mode?: string // P10: 'interval' (default) | 'bar_close' (legacy)
  position_mode?: string // Phase 3: 'ai_watch' (default) | 'bracket_only' (legacy skip)
  is_cross_margin?: boolean
  show_in_competition?: boolean // 是否在竞技场显示
  // 以下字段为向后兼容保留，新版使用策略配置
  btc_eth_leverage?: number
  altcoin_leverage?: number
  trading_symbols?: string
  custom_prompt?: string
  override_base_prompt?: boolean
  system_prompt_template?: string
}

export interface UpdateModelConfigRequest {
  models: {
    [key: string]: {
      name?: string
      enabled: boolean
      api_key: string
      custom_api_url?: string
      custom_model_name?: string
      thinking_mode?: string
      reasoning_effort?: string
    }
  }
}

export interface UpdateExchangeConfigRequest {
  exchanges: {
    [key: string]: {
      enabled: boolean
      api_key: string
      secret_key: string
      passphrase?: string
      testnet?: boolean
      // NinjaTrader specific
      nt_data_dir?: string
      nt_instrument_name?: string
      nt_default_contract_qty?: number
    }
  }
}

export interface BeginnerOnboardingResponse {
  address: string
  private_key: string
  chain: string
  asset: string
  provider: string
  default_model: string
  configured_model_id: string
  env_saved: boolean
  env_path?: string
  reused_existing: boolean
  env_warning?: string
}
