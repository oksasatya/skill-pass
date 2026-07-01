/**
 * MyCertificatesPage — /app/my-certificates
 *
 * States:
 *   not connected        → prompt to connect wallet
 *   gatewayNotConfigured → clean "gateway not configured" notice
 *   loading              → skeleton grid (card-matched)
 *   error                → error state with retry
 *   empty                → designed empty state (what certs are + next step)
 *   data                 → grid of CertificateCard with stagger entrance
 *
 * Grid: repeat(auto-fill, minmax(300px, 1fr)) per DESIGN.md.
 * Stagger: each card fades+rises ~8px, 40ms offset. Reduced-motion: instant.
 *
 * Sonar-TS: readonly props, real elements, no nested ternaries, optional chaining.
 */

import { useAccount } from 'wagmi'
import { Wallet, Award, AlertCircle, RefreshCw, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { CertificateCard, CertificateCardSkeleton } from '@/components/certificate/CertificateCard'
import { useCertificates } from '@/hooks/useCertificates'
import { cn } from '@/lib/utils'

// ── Skeleton grid ─────────────────────────────────────────────────────────────

function SkeletonGrid() {
  return (
    <div
      aria-busy="true"
      aria-label="Loading certificates"
      className="grid gap-4"
      style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))' }}
    >
      {/* ponytail: 3 skeletons cover the common case without over-engineering count logic */}
      {[0, 1, 2].map((i) => (
        <CertificateCardSkeleton key={i} />
      ))}
    </div>
  )
}

// ── Empty state ───────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <div className="flex flex-col items-center gap-5 rounded-xl border border-dashed border-border bg-surface px-6 py-14 text-center">
      <div className="flex size-14 items-center justify-center rounded-full bg-primary-weak">
        <Award className="size-7 text-primary" aria-hidden="true" />
      </div>
      <div className="space-y-1.5 max-w-xs">
        <p className="text-base font-semibold text-ink">No certificates yet</p>
        <p className="text-sm text-ink-muted text-wrap-pretty">
          Certificates issued to your wallet address will appear here.
          Each one is a soulbound token — permanent and non-transferable.
        </p>
      </div>
    </div>
  )
}

// ── Not connected ─────────────────────────────────────────────────────────────

function ConnectPrompt() {
  return (
    <div className="flex flex-col items-center gap-4 rounded-xl border border-dashed border-border bg-surface px-6 py-12 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-primary-weak">
        <Wallet className="size-6 text-primary" aria-hidden="true" />
      </div>
      <div className="space-y-1.5 max-w-xs">
        <p className="text-sm font-semibold text-ink">Connect your wallet to see your certificates</p>
        <p className="text-sm text-ink-muted">
          Your credentials are tied to your wallet address. Connect to view them.
        </p>
      </div>
    </div>
  )
}

// ── Error state ───────────────────────────────────────────────────────────────

type ErrorStateProps = {
  readonly onRetry: () => void
}

function ErrorState({ onRetry }: ErrorStateProps) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center gap-4 rounded-xl border border-danger/30 bg-danger/5 px-6 py-12 text-center"
    >
      <AlertCircle className="size-8 text-danger" aria-hidden="true" />
      <div className="space-y-1.5 max-w-xs">
        <p className="text-sm font-semibold text-danger">Could not load certificates</p>
        <p className="text-sm text-ink-muted">
          There was a problem reading from the blockchain. Check your connection and try again.
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        onClick={onRetry}
        className="gap-2"
      >
        <RefreshCw className="size-3.5" aria-hidden="true" />
        Try again
      </Button>
    </div>
  )
}

// ── Contract not configured ───────────────────────────────────────────────────

function NotConfigured() {
  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-lg border border-warning/40 bg-warning/5 px-4 py-4"
    >
      <AlertTriangle className="size-5 text-warning shrink-0 mt-0.5" aria-hidden="true" />
      <div className="space-y-1">
        <p className="text-sm font-semibold text-ink">Gateway not configured</p>
        <p className="text-sm text-ink-muted">
          Set{' '}
          <code className="font-mono text-xs bg-surface-2 px-1 py-0.5 rounded">
            VITE_GATEWAY_URL
          </code>{' '}
          in{' '}
          <code className="font-mono text-xs bg-surface-2 px-1 py-0.5 rounded">.env</code>{' '}
          and restart the dev server.
        </p>
      </div>
    </div>
  )
}

// ── Certificate grid with stagger entrance ────────────────────────────────────

type CertGridProps = {
  readonly certificates: ReturnType<typeof useCertificates>['certificates']
}

function CertGrid({ certificates }: CertGridProps) {
  return (
    <div
      className="grid gap-4"
      style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))' }}
    >
      {certificates.map((cert, i) => (
        <CertificateCard
          key={cert.tokenId.toString()}
          certificate={cert}
          className={cn(
            // Stagger entrance: each card fades+rises, 40ms offset per DESIGN.md
            'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:slide-in-from-bottom-2',
            'motion-safe:duration-300',
          )}
          // ponytail: inline style for stagger delay — no JS animation library needed for 40ms offset
          style={{ animationDelay: `${i * 40}ms` }}
        />
      ))}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function MyCertificatesPage() {
  const { isConnected } = useAccount()
  const { certificates, isLoading, error, refetch, gatewayNotConfigured } = useCertificates()

  return (
    <section className="space-y-6">
      <header>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          My certificates
        </h1>
        <p className="mt-1 font-sans text-sm text-ink-muted">
          Soulbound credentials issued to your wallet address.
        </p>
      </header>

      {gatewayNotConfigured && <NotConfigured />}

      {!gatewayNotConfigured && !isConnected && <ConnectPrompt />}

      {!gatewayNotConfigured && isConnected && isLoading && <SkeletonGrid />}

      {!gatewayNotConfigured && isConnected && !isLoading && !!error && (
        <ErrorState onRetry={refetch} />
      )}

      {!gatewayNotConfigured && isConnected && !isLoading && !error && certificates.length === 0 && (
        <EmptyState />
      )}

      {!gatewayNotConfigured && isConnected && !isLoading && !error && certificates.length > 0 && (
        <>
          <p className="text-sm text-ink-muted">
            {certificates.length === 1 ? '1 certificate' : `${certificates.length} certificates`}
          </p>
          <CertGrid certificates={certificates} />
        </>
      )}
    </section>
  )
}
