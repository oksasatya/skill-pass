/**
 * VerifyCertificatePage — /certificates/:tokenId
 *
 * Standalone public page. No AppShell chrome, no wallet required.
 * Optimized to be shared and read cold (DESIGN.md §Layout).
 *
 * States:
 *   loading   — skeleton matched to CertificateDetail shape
 *   not-found — invalid/non-existent tokenId (incl. non-numeric route param)
 *   error     — RPC / network failure, with retry
 *   success   — full CertificateDetail with on-chain proof
 *   no-contract — CONTRACT_ADDRESS not configured (dev guard)
 *
 * Non-numeric route param → parsed as null → not-found immediately (no RPC call).
 *
 * Sonar-TS: readonly props, real elements, globalThis, optional chaining, no nested ternaries.
 */

import { useParams, Link } from 'react-router-dom'
import { AlertCircle, FileX, RotateCcw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { parseTokenIdParam } from '@/hooks/useCertificate'
import { useCertificate } from '@/hooks/useCertificate'
import { CertificateDetail, CertificateDetailSkeleton } from '@/components/certificate/CertificateDetail'

// ── Wordmark (minimal header, standalone per DESIGN.md) ───────────────────────

function VerifyHeader() {
  return (
    <header>
      <Link
        to="/"
        className={cn(
          'inline-flex items-center gap-2',
          'text-ink-muted hover:text-ink transition-colors duration-150',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-lg',
        )}
        aria-label="SkillPass — home"
      >
        <span className="size-4 rounded-[3px] bg-primary shrink-0" aria-hidden="true" />
        <span className="font-sans text-sm font-semibold tracking-tight">SkillPass</span>
      </Link>
    </header>
  )
}

// ── Not-found state ───────────────────────────────────────────────────────────

function NotFound({ tokenId }: { readonly tokenId?: string }) {
  return (
    <div
      className="rounded-xl border border-border bg-surface p-8 text-center space-y-4"
      role="alert"
    >
      <FileX
        className="size-10 mx-auto text-ink-muted"
        aria-hidden="true"
      />
      <div className="space-y-1">
        <h1 className="font-sans text-lg font-semibold text-ink">Certificate not found</h1>
        <p className="font-sans text-sm text-ink-muted">
          {tokenId
            ? <>Token <span className="font-mono text-ink">#{tokenId}</span> does not exist on Base Sepolia.</>
            : 'The token ID in this URL is not valid.'}
        </p>
      </div>
      <p className="font-sans text-xs text-ink-muted">
        Check the URL or ask the issuer for the correct certificate link.
      </p>
    </div>
  )
}

// ── Error + retry state ───────────────────────────────────────────────────────

type ErrorStateProps = {
  readonly message: string
  readonly onRetry: () => void
}

function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div
      className="rounded-xl border border-danger/30 bg-danger/5 p-8 text-center space-y-4"
      role="alert"
    >
      <AlertCircle
        className="size-10 mx-auto text-danger"
        aria-hidden="true"
      />
      <div className="space-y-1">
        <h1 className="font-sans text-lg font-semibold text-ink">Could not load certificate</h1>
        <p className="font-sans text-sm text-ink-muted">{message}</p>
      </div>
      <button
        type="button"
        onClick={onRetry}
        className={cn(
          'inline-flex items-center gap-2',
          'font-sans text-sm font-medium text-primary',
          'hover:text-primary/80 transition-colors duration-150',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-md px-2 py-1',
        )}
      >
        <RotateCcw className="size-3.5" aria-hidden="true" />
        Try again
      </button>
    </div>
  )
}

// ── Contract not configured (dev guard) ───────────────────────────────────────

function ContractNotConfigured() {
  return (
    <div
      className="rounded-xl border border-warning/30 bg-warning/5 p-8 text-center space-y-2"
      role="alert"
    >
      <p className="font-sans text-sm font-semibold text-ink">Contract not configured</p>
      <p className="font-sans text-xs text-ink-muted">
        Set <span className="font-mono text-ink">VITE_CONTRACT_ADDRESS</span> to enable certificate verification.
      </p>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function VerifyCertificatePage() {
  const { tokenId: tokenIdParam } = useParams<{ tokenId: string }>()

  // Validate param early — non-numeric → immediate not-found (no RPC call)
  const tokenId = parseTokenIdParam(tokenIdParam)
  const isInvalidParam = tokenId === null

  const {
    certificate,
    isLoading,
    notFound,
    error,
    refetch,
    contractNotConfigured,
  } = useCertificate(tokenId)

  return (
    <div className="min-h-svh bg-bg text-ink">
      <div className="mx-auto max-w-xl px-4 py-10 md:py-16 space-y-8">
        <VerifyHeader />

        {/* Main content area */}
        <main id="main-content">
          {contractNotConfigured && <ContractNotConfigured />}

          {!contractNotConfigured && (isInvalidParam || notFound) && (
            <NotFound tokenId={tokenIdParam} />
          )}

          {!contractNotConfigured && !isInvalidParam && !notFound && error && (
            <ErrorState
              message={error.message ?? 'An unexpected error occurred.'}
              onRetry={refetch}
            />
          )}

          {!contractNotConfigured && !isInvalidParam && !notFound && !error && isLoading && (
            <CertificateDetailSkeleton />
          )}

          {!contractNotConfigured && !isInvalidParam && !notFound && !error && !isLoading && certificate && (
            <CertificateDetail certificate={certificate} />
          )}
        </main>

        {/* Footer — minimal, no chrome */}
        <footer className="text-center">
          <p className="font-sans text-xs text-ink-muted">
            Certificates on SkillPass are{' '}
            <span className="font-medium text-ink">soulbound</span>{' '}
            and verified directly on{' '}
            <a
              href="https://base.org"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary rounded"
            >
              Base
            </a>
            .
          </p>
        </footer>
      </div>
    </div>
  )
}
