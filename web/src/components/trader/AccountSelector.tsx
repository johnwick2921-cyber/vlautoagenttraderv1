import { useCallback, useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import useSWR from 'swr'
import { api } from '../../lib/api'
import { notify } from '../../lib/notify'
import { ChevronDown, Check } from 'lucide-react'
import { cn } from '../../lib/cn'
import type { AccountsResponse } from '../../types'

interface AccountSelectorProps {
  traderId: string
  onAccountChanged?: () => void
}

/**
 * AccountSelector - Displays all accounts in a dropdown
 * SIM accounts are selectable, live accounts are greyed out and disabled
 * Current account is highlighted with a checkmark
 */
export function AccountSelector({
  traderId,
  onAccountChanged,
}: AccountSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [isSelecting, setIsSelecting] = useState(false)
  const [triggerRect, setTriggerRect] = useState<DOMRect | null>(null)
  const [dropdownPos, setDropdownPos] = useState({ top: 0, left: 0, width: 0 })

  // Fetch accounts
  const {
    data: accountsData,
    isLoading,
    error,
    mutate,
  } = useSWR<AccountsResponse>(
    traderId ? `accounts-${traderId}` : null,
    () => {
      if (!traderId) return Promise.reject(new Error('No traderId'))
      return api.getAccounts(traderId).catch((err) => {
        console.error('AccountSelector: failed to fetch accounts', err)
        throw err
      })
    },
    { revalidateOnFocus: false, dedupingInterval: 5000 }
  )

  // The trader's PERSISTED chosen account (multi-account Stage 1). "" = none chosen
  // yet → the trader is GATED (does not trade). Show/highlight THIS, not the streamed
  // `current` (which can drift on the NT8 account_balance stream).
  const selectedAccount = accountsData?.selected || ''
  // `current` is the ONE account the shared NT8 connection is streaming (balance /
  // positions / PnL reflect it). It can differ from THIS trader's bound `selected`
  // account when another trader/panel last switched the connection — so the on-screen
  // numbers must be labeled with `current` to avoid masquerading as this trader's.
  const currentAccount = accountsData?.current || ''
  const accounts = accountsData?.accounts || []

  // Memoize ref callback to prevent infinite setState loop
  const handleTriggerRef = useCallback((el: HTMLButtonElement | null) => {
    if (el) setTriggerRect(el.getBoundingClientRect())
  }, [])

  // Update dropdown position when opened
  useEffect(() => {
    if (!isOpen || !triggerRect) return

    setDropdownPos({
      top: triggerRect.bottom + 8,
      left: triggerRect.left,
      width: triggerRect.width,
    })

    // Close dropdown when clicking outside
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('[data-account-selector]')) {
        setIsOpen(false)
      }
    }

    // Close dropdown on scroll
    const handleScroll = () => {
      setIsOpen(false)
    }

    document.addEventListener('mousedown', handleClickOutside)
    window.addEventListener('scroll', handleScroll, true)

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      window.removeEventListener('scroll', handleScroll, true)
    }
  }, [isOpen, triggerRect])

  const handleSelectAccount = async (accountName: string) => {
    if (accountName === selectedAccount) {
      setIsOpen(false)
      return
    }

    setIsSelecting(true)
    try {
      await api.selectAccount(traderId, accountName)
      notify.success(`Account switched to ${accountName}`)
      await mutate()
      onAccountChanged?.()
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : 'Failed to select account'
      notify.error(errorMsg)
    } finally {
      setIsSelecting(false)
      setIsOpen(false)
    }
  }

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 px-3 py-2 rounded border border-nofx-gold/20 bg-nofx-bg/30 text-sm text-nofx-text-muted"
        data-account-selector
      >
        <div className="h-4 w-20 bg-nofx-bg/50 rounded animate-pulse" />
      </div>
    )
  }

  if (error) {
    return (
      <div
        className="flex items-center gap-2 px-3 py-2 rounded border border-red-500/30 bg-red-500/5 text-xs text-red-400"
        data-account-selector
      >
        Failed to load accounts
      </div>
    )
  }

  return (
    <div data-account-selector className="relative">
      <button
        ref={handleTriggerRef}
        onClick={() => setIsOpen(!isOpen)}
        onFocus={(e) => setTriggerRect(e.currentTarget.getBoundingClientRect())}
        className={cn(
          'flex items-center gap-2 px-3 py-2 rounded border transition-all',
          'text-sm font-medium',
          isOpen
            ? 'border-nofx-gold/50 bg-nofx-gold/10 text-nofx-gold'
            : 'border-nofx-gold/20 bg-nofx-bg/30 text-nofx-text hover:border-nofx-gold/40 hover:bg-nofx-bg/50'
        )}
      >
        <span className="truncate">
          {selectedAccount ? (
            <>
              Account: <span className="font-semibold">{selectedAccount}</span>
            </>
          ) : (
            <span className="font-semibold text-nofx-gold">
              Select an account
            </span>
          )}
        </span>
        <ChevronDown
          className={cn(
            'w-3 h-3 shrink-0 opacity-60 transition-transform',
            isOpen && 'rotate-180'
          )}
        />
      </button>

      {/* Masquerade guard: the shared NT8 connection streams ONE account's balance /
          positions. If it isn't this trader's bound account, say so explicitly so the
          numbers can't be mistaken for this trader's account. */}
      {selectedAccount &&
        currentAccount &&
        currentAccount !== selectedAccount && (
          <div className="mt-1 px-2 py-1 rounded bg-amber-500/10 border border-amber-500/30 text-[11px] leading-tight text-amber-400">
            ⚠ Balance &amp; positions on screen are for{' '}
            <span className="font-semibold">{currentAccount}</span> (the
            connection&apos;s active account). This trader trades{' '}
            <span className="font-semibold">{selectedAccount}</span>.
          </div>
        )}

      {/* Dropdown Menu — Portal to body to escape parent overflow */}
      {isOpen &&
        typeof document !== 'undefined' &&
        document.body &&
        ReactDOM.createPortal(
          <div
            data-account-selector
            className="fixed z-[9999] rounded border border-nofx-gold/20 bg-[#0B0E11] shadow-xl shadow-black/50 max-h-64 overflow-y-auto"
            style={{
              top: `${dropdownPos.top}px`,
              left: `${dropdownPos.left}px`,
              minWidth: `${dropdownPos.width}px`,
            }}
          >
            {accounts.length === 0 ? (
              <div className="px-3 py-2 text-xs text-nofx-text-muted text-center">
                Waiting for accounts...
              </div>
            ) : (
              accounts.map((account) => (
                <button
                  key={account.name}
                  onClick={() => {
                    if (account.is_sim && !isSelecting) {
                      handleSelectAccount(account.name)
                    }
                  }}
                  disabled={!account.is_sim || isSelecting}
                  title={
                    !account.is_sim
                      ? 'Live accounts are not selectable — only SIM accounts can be auto-traded'
                      : ''
                  }
                  className={cn(
                    'w-full flex items-center gap-3 px-3 py-2 text-sm transition-colors border-b border-nofx-bg/30 last:border-b-0',
                    account.is_sim
                      ? cn(
                          'text-nofx-text-main hover:bg-nofx-bg/50 cursor-pointer',
                          account.name === selectedAccount
                            ? 'bg-nofx-gold/10 text-nofx-gold'
                            : ''
                        )
                      : 'text-nofx-text-muted opacity-50 cursor-not-allowed bg-nofx-bg/20'
                  )}
                >
                  <span className="flex-1 text-left truncate">
                    {account.name}
                    {!account.is_sim && (
                      <span className="text-xs ml-2 opacity-70">
                        (live — not selectable for auto-trade)
                      </span>
                    )}
                  </span>
                  {account.name === selectedAccount && (
                    <Check className="w-4 h-4 text-nofx-gold shrink-0" />
                  )}
                </button>
              ))
            )}
          </div>,
          document.body
        )}
    </div>
  )
}
