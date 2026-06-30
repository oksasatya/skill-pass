/**
 * CertificateCard — per DESIGN.md spec.
 *
 * Layout: serif title, issuer + recipient (sans), issued date, token id (mono).
 * No side-stripe, no nested cards. Full border + surface elevation.
 * Hover lift ~2px (transform). Reduced-motion: instant.
 * Links to /certificates/:tokenId.
 *
 * Sonar-TS: readonly props, real elements, optional chaining, no nested ternaries.
 */

import { Link } from 'react-router-dom'
import { Award } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { CertificateView } from '@/hooks/useCertificates'

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatIssuedDate(issuedAt: bigint): string {
  const ms = Number(issuedAt) * 1000
  // issuedAt = 0 means unknown (fallback from mapMulticallResult)
  if (ms === 0) return '—'
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(ms))
}

// ── Component ─────────────────────────────────────────────────────────────────

type CertificateCardProps = {
  readonly certificate: CertificateView
  readonly className?: string
  readonly style?: React.CSSProperties
}

export function CertificateCard({ certificate, className, style }: CertificateCardProps) {
  const { tokenId, title, issuerName, recipientName, issuedAt } = certificate
  const href = `/certificates/${tokenId.toString()}`
  const dateLabel = formatIssuedDate(issuedAt)

  return (
    <Link
      to={href}
      style={style}
      aria-label={`View certificate: ${title} — issued by ${issuerName}`}
      className={cn(
        // Base surface
        'group block rounded-xl border border-border bg-surface',
        // Padding
        'px-5 py-6',
        // Hover lift — transform only, reduced-motion collapses to nothing
        'transition-all duration-200 ease-out',
        'motion-safe:hover:-translate-y-0.5 motion-safe:hover:shadow-sm',
        // Focus ring
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-bg',
        className,
      )}
    >
      {/* Certificate title — Fraunces serif, diploma gravitas */}
      <h2
        className={cn(
          'font-serif text-[clamp(1.125rem,2vw,1.375rem)] font-semibold leading-snug tracking-[-0.01em]',
          'text-ink text-wrap-balance',
          'mb-4',
        )}
      >
        {title}
      </h2>

      {/* Meta grid */}
      <dl className="space-y-2 text-sm">
        <div className="flex items-baseline gap-2">
          <dt className="text-ink-muted shrink-0 w-[5.5rem]">Issued by</dt>
          <dd className="text-ink font-medium truncate">{issuerName}</dd>
        </div>

        <div className="flex items-baseline gap-2">
          <dt className="text-ink-muted shrink-0 w-[5.5rem]">Recipient</dt>
          <dd className="text-ink truncate">{recipientName}</dd>
        </div>

        <div className="flex items-baseline gap-2">
          <dt className="text-ink-muted shrink-0 w-[5.5rem]">Issued</dt>
          <dd className="text-ink">
            <time dateTime={dateLabel !== '—' ? new Date(Number(issuedAt) * 1000).toISOString() : undefined}>
              {dateLabel}
            </time>
          </dd>
        </div>
      </dl>

      {/* Token ID footer — mono, subtle */}
      <footer className="mt-5 pt-4 border-t border-border flex items-center justify-between gap-3">
        <div className="flex items-center gap-1.5 text-ink-muted">
          <Award className="size-3.5 shrink-0" aria-hidden="true" />
          <span className="text-xs">Soulbound</span>
        </div>
        <span
          className="font-mono text-xs text-ink-muted tracking-tight"
          aria-label={`Token ID ${tokenId.toString()}`}
        >
          #{tokenId.toString()}
        </span>
      </footer>
    </Link>
  )
}

// ── Skeleton (loading state matched to card shape) ─────────────────────────────

export function CertificateCardSkeleton() {
  return (
    <div
      aria-hidden="true"
      className="rounded-xl border border-border bg-surface px-5 py-6 animate-pulse"
    >
      {/* Title shimmer */}
      <div className="h-6 w-3/4 rounded bg-surface-2 mb-4" />

      {/* Meta rows */}
      <div className="space-y-2">
        <div className="flex gap-2">
          <div className="h-4 w-20 rounded bg-surface-2" />
          <div className="h-4 w-28 rounded bg-surface-2" />
        </div>
        <div className="flex gap-2">
          <div className="h-4 w-20 rounded bg-surface-2" />
          <div className="h-4 w-24 rounded bg-surface-2" />
        </div>
        <div className="flex gap-2">
          <div className="h-4 w-20 rounded bg-surface-2" />
          <div className="h-4 w-20 rounded bg-surface-2" />
        </div>
      </div>

      {/* Footer shimmer */}
      <div className="mt-5 pt-4 border-t border-border flex justify-between">
        <div className="h-3 w-16 rounded bg-surface-2" />
        <div className="h-3 w-10 rounded bg-surface-2" />
      </div>
    </div>
  )
}
