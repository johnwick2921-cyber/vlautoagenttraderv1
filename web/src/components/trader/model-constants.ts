// Constants for AI model and provider configuration
// CR-C: the legacy payment-provider constants left with the payment
// family (C4/C5); only the display helpers remain. crypto-era sections removed.

// Get friendly AI model display name
export function getModelDisplayName(modelId: string): string {
  switch (modelId.toLowerCase()) {
    case 'deepseek':
      return 'DeepSeek'
    case 'qwen':
      return 'Qwen'
    case 'claude':
      return 'Claude'
    default:
      return modelId.toUpperCase()
  }
}

// Extract name part after underscore
export function getShortName(fullName: string): string {
  const parts = fullName.split('_')
  return parts.length > 1 ? parts[parts.length - 1] : fullName
}

// Helper to truncate a long address for display (kept until its crypto callers
// are cut with the ModelConfigModal surgery; rows re-flagged KEEP in the table).
export function truncateAddress(address: string | undefined, len = 10): string {
  if (!address) return ''
  return address.length > len * 2
    ? `${address.slice(0, len)}...${address.slice(-len)}`
    : address
}

export function getExchangeDisplayName(
  exchangeId: string | undefined,
  exchanges: {
    id: string
    exchange_type?: string
    name: string
    account_name?: string
  }[]
): string {
  if (!exchangeId) return 'Unknown'
  const exchange = exchanges.find((e) => e.id === exchangeId)
  if (!exchange) return exchangeId.substring(0, 8).toUpperCase() + '...' // Show truncated UUID if not found
  const typeName = exchange.exchange_type?.toUpperCase() || exchange.name
  return exchange.account_name
    ? `${typeName} - ${exchange.account_name}`
    : typeName
}
