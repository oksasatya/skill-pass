import { useAccount, useConnect, useDisconnect } from 'wagmi'
import { Wallet, ChevronDown, Loader2, LogOut, PlugZap } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { truncateAddress } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useState } from 'react'

/**
 * ConnectButton — wallet slot for the Nav.
 *
 * States:
 *   not connected → connector picker (injected, Coinbase Wallet)
 *   pending       → spinner + "Connecting…"
 *   connected     → truncated address (mono) + chain label + disconnect option
 *
 * Design: DESIGN.md — mono for address, primary button for connect,
 * secondary for connected state, ghost/destructive for disconnect.
 * Keyboard-focusable; no color-only status.
 */

// Friendly label map — connector name → display string
function connectorLabel(name: string): string {
  if (name.toLowerCase().includes('coinbase')) return 'Coinbase Wallet'
  if (name.toLowerCase().includes('inject')) return 'Browser Wallet'
  return name
}

export function ConnectButton(): React.ReactElement {
  const { address, isConnected, isConnecting } = useAccount()
  const { connect, connectors, isPending } = useConnect()
  const { disconnect } = useDisconnect()

  // ponytail: local open state for the dropdown — no portal/popover lib needed for 2 items
  const [open, setOpen] = useState(false)
  const [disconnectOpen, setDisconnectOpen] = useState(false)

  const loading = isConnecting || isPending

  // ── Not connected: connector picker ──────────────────────────────────────
  if (!isConnected) {
    return (
      <div className="relative">
        <Button
          type="button"
          variant="default"
          size="sm"
          aria-haspopup="true"
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
          className="min-h-[44px] gap-1.5"
          disabled={loading}
        >
          {loading
            ? <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
            : <Wallet className="size-3.5" aria-hidden="true" />}
          {loading ? 'Connecting…' : 'Connect'}
          {!loading && <ChevronDown className="size-3 opacity-60" aria-hidden="true" />}
        </Button>

        {open && !loading && (
          <>
            {/* Click-away backdrop */}
            <div
              className="fixed inset-0 z-20"
              aria-hidden="true"
              onClick={() => setOpen(false)}
            />
            {/* Dropdown panel */}
            <div
              role="menu"
              aria-label="Choose a wallet"
              className={cn(
                'absolute right-0 top-full z-30 mt-1 min-w-[180px]',
                'rounded-lg border border-border bg-surface',
                'shadow-xs py-1',
                // ponytail: motion.reduce → no fade, just show
                'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:zoom-in-95',
                'motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95',
              )}
            >
              {connectors.map((connector) => (
                <button
                  key={connector.uid}
                  type="button"
                  role="menuitem"
                  className={cn(
                    'flex w-full items-center gap-2.5 px-3 py-2',
                    'text-sm font-medium text-ink',
                    'hover:bg-surface-2 focus-visible:bg-surface-2',
                    'focus-visible:outline-none transition-colors duration-150',
                  )}
                  onClick={() => {
                    setOpen(false)
                    connect({ connector })
                  }}
                >
                  <PlugZap className="size-3.5 text-ink-muted" aria-hidden="true" />
                  {connectorLabel(connector.name)}
                </button>
              ))}
            </div>
          </>
        )}
      </div>
    )
  }

  // ── Connected: address + disconnect ──────────────────────────────────────
  const displayAddress = address ? truncateAddress(address) : '…'

  return (
    <div className="relative">
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-haspopup="true"
        aria-expanded={disconnectOpen}
        onClick={() => setDisconnectOpen((v) => !v)}
        className="min-h-[44px] gap-1.5"
      >
        {/* Mono address — DESIGN.md: on-chain data in JetBrains Mono */}
        <span
          className="font-mono text-xs tracking-tight"
          aria-label={`Connected: ${address ?? ''}`}
        >
          {displayAddress}
        </span>
        <span className="text-ink-muted text-xs hidden sm:inline">· Base Sepolia</span>
        <ChevronDown className="size-3 opacity-60" aria-hidden="true" />
      </Button>

      {disconnectOpen && (
        <>
          <div
            className="fixed inset-0 z-20"
            aria-hidden="true"
            onClick={() => setDisconnectOpen(false)}
          />
          <div
            role="menu"
            aria-label="Wallet options"
            className={cn(
              'absolute right-0 top-full z-30 mt-1 min-w-[160px]',
              'rounded-lg border border-border bg-surface',
              'shadow-xs py-1',
              'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:zoom-in-95',
            )}
          >
            <button
              type="button"
              role="menuitem"
              className={cn(
                'flex w-full items-center gap-2.5 px-3 py-2',
                'text-sm font-medium text-danger',
                'hover:bg-surface-2 focus-visible:bg-surface-2',
                'focus-visible:outline-none transition-colors duration-150',
              )}
              onClick={() => {
                setDisconnectOpen(false)
                disconnect()
              }}
            >
              <LogOut className="size-3.5" aria-hidden="true" />
              Disconnect
            </button>
          </div>
        </>
      )}
    </div>
  )
}
