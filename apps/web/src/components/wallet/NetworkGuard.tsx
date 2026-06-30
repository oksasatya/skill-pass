import { useAccount, useSwitchChain } from 'wagmi'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useIsCorrectNetwork } from '@/hooks/useIsCorrectNetwork'
import { baseSepolia } from '@/lib/chains'
import { cn } from '@/lib/utils'

interface NetworkGuardProps {
  /**
   * Content to render when the network is correct (or wallet not connected).
   * When connected + wrong network, children are NOT rendered — writes are gated.
   * ponytail: children prop is optional so NetworkGuard can be used standalone as a banner.
   */
  readonly children?: React.ReactNode
}

/**
 * NetworkGuard — shows a --danger banner when connected to the wrong network.
 *
 * Design rules (DESIGN.md):
 *   - Status by text + icon (⚠ + text), NEVER color alone.
 *   - --danger token for banner bg/text.
 *   - Keyboard-focusable (role="alert" + button is natively focusable).
 *   - prefers-reduced-motion: motion-safe prefix on transitions.
 *
 * Usage:
 *   <NetworkGuard>
 *     <WriteForm />   ← only rendered on correct network
 *   </NetworkGuard>
 *
 *   or standalone banner (no children) for pages that need the guard visible
 *   but handle the gate themselves via useIsCorrectNetwork().
 */
export function NetworkGuard({ children }: NetworkGuardProps): React.ReactElement {
  const { isConnected } = useAccount()
  const { isCorrectNetwork } = useIsCorrectNetwork()
  const { switchChain, isPending } = useSwitchChain()

  // Not connected or on the right network — render children (or nothing)
  if (!isConnected || isCorrectNetwork) {
    return <>{children}</>
  }

  // Connected + wrong network → danger banner; children withheld (writes gated)
  return (
    <>
      <div
        role="alert"
        aria-live="assertive"
        aria-atomic="true"
        className={cn(
          // Banner layout
          'flex flex-wrap items-center justify-between gap-3 rounded-lg px-4 py-3',
          // --danger token for bg (10% opacity) + border (30%) + text
          'bg-danger/10 border border-danger/30 text-danger',
          // Motion: slide-in only when motion is OK
          'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:slide-in-from-top-1',
          'motion-safe:duration-200',
        )}
      >
        {/* Left: icon + text — DESIGN.md: status NEVER color alone */}
        <div className="flex items-center gap-2">
          <AlertTriangle
            className="size-4 shrink-0"
            aria-hidden="true"
          />
          <span className="text-sm font-medium">
            Wrong network — please switch to Base Sepolia
          </span>
        </div>

        {/* Right: switch action */}
        <Button
          type="button"
          variant="destructive"
          size="sm"
          disabled={isPending}
          onClick={() => switchChain({ chainId: baseSepolia.id })}
          className="min-h-[44px] shrink-0"
        >
          {isPending && (
            <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
          )}
          {isPending ? 'Switching…' : 'Switch to Base Sepolia'}
        </Button>
      </div>

      {/*
        Children intentionally withheld on wrong network.
        Phase 2b can gate writes by wrapping <IssueForm /> in <NetworkGuard />.
        Consumers can also import useIsCorrectNetwork() directly for finer control.
      */}
    </>
  )
}

// Re-export the hook so 2b can import from one place
export { useIsCorrectNetwork } from '@/hooks/useIsCorrectNetwork'
