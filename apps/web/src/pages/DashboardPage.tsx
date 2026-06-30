/**
 * DashboardPage — /app
 *
 * Post-connect hub: wallet + network status, certificate count, quick links.
 * Not connected → prompt to connect.
 *
 * Quick links:
 *   - My Certificates (always)
 *   - Issue (only if useIsAdmin returns true)
 *
 * Sonar-TS: readonly props, real elements, no nested ternaries, optional chaining.
 */

import { Link } from 'react-router-dom'
import { useAccount } from 'wagmi'
import { Award, PlusCircle, ArrowRight, Wallet, CheckCircle2, AlertTriangle } from 'lucide-react'
import { useIsAdmin } from '@/hooks/useIsAdmin'
import { useCertificates } from '@/hooks/useCertificates'
import { useIsCorrectNetwork } from '@/hooks/useIsCorrectNetwork'
import { truncateAddress } from '@/lib/format'
import { cn } from '@/lib/utils'
import { CONTRACT_ADDRESS } from '@/lib/contract'

// ── Quick link card ───────────────────────────────────────────────────────────

type QuickLinkProps = {
  readonly to: string
  readonly icon: React.ReactNode
  readonly label: string
  readonly description: string
}

function QuickLink({ to, icon, label, description }: QuickLinkProps) {
  return (
    <Link
      to={to}
      className={cn(
        'group flex items-center gap-4 rounded-xl border border-border bg-surface px-5 py-4',
        'transition-all duration-150 ease-out',
        'motion-safe:hover:-translate-y-px motion-safe:hover:shadow-xs',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-bg',
      )}
    >
      <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary-weak text-primary">
        {icon}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-ink">{label}</p>
        <p className="text-xs text-ink-muted mt-0.5">{description}</p>
      </div>
      <ArrowRight
        className="size-4 text-ink-muted shrink-0 transition-transform duration-150 group-hover:translate-x-0.5"
        aria-hidden="true"
      />
    </Link>
  )
}

// ── Wallet status row ─────────────────────────────────────────────────────────

type WalletStatusProps = {
  readonly address: `0x${string}`
  readonly isCorrectNetwork: boolean
}

function WalletStatus({ address, isCorrectNetwork }: WalletStatusProps) {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-xl border border-border bg-surface px-5 py-4">
      <div className="flex items-center gap-2 min-w-0">
        <span className="font-mono text-sm text-ink tracking-tight truncate">
          {truncateAddress(address)}
        </span>
      </div>

      <div className="flex items-center gap-1.5 ml-auto">
        {isCorrectNetwork ? (
          <>
            <CheckCircle2 className="size-4 text-success shrink-0" aria-hidden="true" />
            <span className="text-xs font-medium text-success">Base Sepolia</span>
          </>
        ) : (
          <>
            <AlertTriangle className="size-4 text-danger shrink-0" aria-hidden="true" />
            <span className="text-xs font-medium text-danger">Wrong network</span>
          </>
        )}
      </div>
    </div>
  )
}

// ── Connect prompt ────────────────────────────────────────────────────────────

function ConnectPrompt() {
  return (
    <div className="flex flex-col items-center gap-4 rounded-xl border border-dashed border-border bg-surface px-6 py-14 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-primary-weak">
        <Wallet className="size-6 text-primary" aria-hidden="true" />
      </div>
      <div className="space-y-1.5 max-w-xs">
        <p className="text-sm font-semibold text-ink">Connect your wallet to continue</p>
        <p className="text-sm text-ink-muted">
          Use the Connect button above to link your wallet and access your dashboard.
        </p>
      </div>
    </div>
  )
}

// ── Certificate count stat ────────────────────────────────────────────────────

type CertCountProps = {
  readonly count: number
  readonly isLoading: boolean
}

function CertCount({ count, isLoading }: CertCountProps) {
  return (
    <div className="rounded-xl border border-border bg-surface px-5 py-4 flex items-center gap-4">
      <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary-weak text-primary">
        <Award className="size-5" aria-hidden="true" />
      </div>
      <div>
        <p className="text-xs text-ink-muted">Certificates held</p>
        {isLoading ? (
          <div className="mt-1 h-6 w-8 rounded bg-surface-2 animate-pulse" aria-label="Loading count" />
        ) : (
          <p className="text-2xl font-semibold text-ink leading-none mt-0.5">{count}</p>
        )}
      </div>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const { isConnected, address } = useAccount()
  const { isAdmin } = useIsAdmin()
  const { isCorrectNetwork } = useIsCorrectNetwork()
  const { certificates, isLoading: certsLoading } = useCertificates()

  const contractNotConfigured = !CONTRACT_ADDRESS

  return (
    <section className="space-y-6 max-w-2xl">
      <header>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          Dashboard
        </h1>
        <p className="mt-1 font-sans text-sm text-ink-muted">
          Your SkillPass activity at a glance.
        </p>
      </header>

      {!isConnected && <ConnectPrompt />}

      {isConnected && address && (
        <div className="space-y-4">
          {/* Wallet + network status */}
          <WalletStatus address={address} isCorrectNetwork={isCorrectNetwork} />

          {/* Certificate count — skipped when contract not configured */}
          {!contractNotConfigured && (
            <CertCount count={certificates.length} isLoading={certsLoading} />
          )}

          {/* Quick links */}
          <div className="space-y-2 pt-2">
            <p className="text-xs font-medium text-ink-muted">Quick links</p>
            <div className="space-y-2">
              <QuickLink
                to="/app/my-certificates"
                icon={<Award className="size-5" aria-hidden="true" />}
                label="My Certificates"
                description="View credentials issued to your wallet"
              />
              {isAdmin && (
                <QuickLink
                  to="/app/issue"
                  icon={<PlusCircle className="size-5" aria-hidden="true" />}
                  label="Issue a certificate"
                  description="Mint a soulbound credential to a recipient"
                />
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}
