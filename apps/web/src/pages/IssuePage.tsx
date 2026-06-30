/**
 * IssuePage — /app/issue
 *
 * Three gate states (in order):
 *   1. Not connected          → prompt to connect wallet
 *   2. Connected, not owner   → "Only the contract owner can issue" (shows owner address)
 *   3. Connected + owner      → IssueForm wrapped in NetworkGuard
 *
 * Contract not configured (no VITE_CONTRACT_ADDRESS) → surface clearly, do not crash.
 *
 * Sonar-TS: readonly props, real elements, no nested ternaries, globalThis, optional chaining.
 */

import { useAccount } from 'wagmi'
import { ShieldAlert, Wallet, AlertTriangle } from 'lucide-react'
import { NetworkGuard } from '@/components/wallet/NetworkGuard'
import { IssueForm } from '@/components/certificate/IssueForm'
import { useIsAdmin } from '@/hooks/useIsAdmin'
import { useIssueCertificate } from '@/hooks/useIssueCertificate'
import { cn } from '@/lib/utils'

export default function IssuePage() {
  const { isConnected } = useAccount()
  const { isAdmin, owner, isLoading, contractNotConfigured } = useIsAdmin()
  const issuer = useIssueCertificate()

  // ── Contract not configured ──────────────────────────────────────────────
  if (contractNotConfigured) {
    return (
      <PageShell>
        <div
          role="alert"
          className="flex items-start gap-3 rounded-lg border border-warning/40 bg-warning/5 px-4 py-4"
        >
          <AlertTriangle className="size-5 text-warning shrink-0 mt-0.5" aria-hidden="true" />
          <div className="space-y-1">
            <p className="text-sm font-semibold text-ink">Contract not configured</p>
            <p className="text-sm text-ink-muted">
              Set <code className="font-mono text-xs bg-surface-2 px-1 py-0.5 rounded">VITE_CONTRACT_ADDRESS</code> in{' '}
              <code className="font-mono text-xs bg-surface-2 px-1 py-0.5 rounded">.env</code> and
              restart the dev server.
            </p>
          </div>
        </div>
      </PageShell>
    )
  }

  // ── Not connected ─────────────────────────────────────────────────────────
  if (!isConnected) {
    return (
      <PageShell>
        <div className="flex flex-col items-center justify-center gap-4 rounded-xl border border-dashed border-border bg-surface px-6 py-12 text-center">
          <div className="flex size-12 items-center justify-center rounded-full bg-primary-weak">
            <Wallet className="size-6 text-primary" aria-hidden="true" />
          </div>
          <div className="space-y-1.5 max-w-xs">
            <p className="text-sm font-semibold text-ink">Connect your wallet to continue</p>
            <p className="text-sm text-ink-muted">
              Issuing a certificate requires signing a transaction with the contract owner's wallet.
            </p>
          </div>
        </div>
      </PageShell>
    )
  }

  // ── Loading owner ─────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <PageShell>
        <div className="flex items-center gap-2 text-sm text-ink-muted py-4">
          <span
            className="size-4 rounded-full border-2 border-border border-t-primary animate-spin"
            aria-hidden="true"
          />
          Checking owner…
        </div>
      </PageShell>
    )
  }

  // ── Connected but not the owner ───────────────────────────────────────────
  if (!isAdmin) {
    return (
      <PageShell>
        <div
          role="alert"
          className={cn(
            'flex items-start gap-3 rounded-lg border border-danger/30 bg-danger/5 px-4 py-4',
            'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:duration-200',
          )}
        >
          <ShieldAlert className="size-5 text-danger shrink-0 mt-0.5" aria-hidden="true" />
          <div className="space-y-2">
            <p className="text-sm font-semibold text-danger">
              Only the contract owner can issue certificates
            </p>
            {owner && (
              <p className="text-xs text-ink-muted">
                Owner:{' '}
                <span className="font-mono text-ink break-all">{owner}</span>
              </p>
            )}
            <p className="text-xs text-ink-muted">
              Connect the owner wallet to issue certificates.
            </p>
          </div>
        </div>
      </PageShell>
    )
  }

  // ── Owner: show the form, gated behind NetworkGuard ───────────────────────
  return (
    <PageShell>
      <NetworkGuard>
        <IssueForm issuer={issuer} />
      </NetworkGuard>
    </PageShell>
  )
}

// ── Layout wrapper ────────────────────────────────────────────────────────────

type PageShellProps = {
  readonly children: React.ReactNode
}

function PageShell({ children }: PageShellProps) {
  return (
    <section className="space-y-6 max-w-2xl">
      <header>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          Issue a certificate
        </h1>
        <p className="mt-1 font-sans text-sm text-ink-muted">
          Mint a soulbound credential to a recipient's wallet. The certificate is
          permanent and non-transferable.
        </p>
      </header>
      {children}
    </section>
  )
}
