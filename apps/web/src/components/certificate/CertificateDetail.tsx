/**
 * CertificateDetail — the public verify artifact.
 *
 * Spec (DESIGN.md): single focused column, cert as a document.
 * - Fraunces serif title
 * - Full field set via <dl>/<dt>/<dd>
 * - Distinct on-chain proof block: contract, tokenId, txHash, block
 *   each in mono with a copy button + Basescan link
 * - "✓ Verified on Base Sepolia" affirmation (text + icon, not color alone)
 *
 * Sonar-TS: readonly props, real elements (<dl>/<dt>/<dd>, <button>),
 * globalThis, optional chaining, no nested ternaries, no any.
 * WCAG AA: copy buttons keyboard-operable + announce live state.
 * Reduced-motion: no transform animations, opacity-only.
 */

import { useState, useCallback } from 'react'
import { CheckCircle, Copy, Check, ExternalLink, Award, ShieldCheck } from 'lucide-react'
import { cn } from '@/lib/utils'
import { truncateAddress } from '@/lib/format'
import {
  txExplorerUrl,
  nftExplorerUrl,
  contractExplorerUrl,
} from '@/hooks/useCertificate'
import type { CertificateOnChain } from '@/hooks/useCertificate'
import { CONTRACT_ADDRESS } from '@/lib/contract'

// ── Helpers ───────────────────────────────────────────────────────────────────

// ponytail: hoisted — Intl.DateTimeFormat construction is expensive; build once at module load
const dateFormatter = new Intl.DateTimeFormat('en-US', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
})

function formatDate(issuedAt: bigint): string {
  const ms = Number(issuedAt) * 1000
  if (ms === 0) return '—'
  return dateFormatter.format(new Date(ms))
}

function formatISODate(issuedAt: bigint): string {
  const ms = Number(issuedAt) * 1000
  if (ms === 0) return ''
  return new Date(ms).toISOString()
}

// ── CopyButton ────────────────────────────────────────────────────────────────

type CopyButtonProps = {
  readonly value: string
  readonly label: string
}

function CopyButton({ value, label }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await globalThis.navigator?.clipboard?.writeText(value)
      setCopied(true)
      globalThis.setTimeout?.(() => setCopied(false), 2000)
    } catch {
      // Clipboard API unavailable — silently skip (e.g. non-HTTPS context)
    }
  }, [value])

  return (
    <button
      type="button"
      onClick={() => { void handleCopy() }}
      aria-label={copied ? `${label} copied` : `Copy ${label}`}
      aria-live="polite"
      className={cn(
        'inline-flex items-center justify-center rounded-md',
        'size-7 shrink-0',
        'text-ink-muted hover:text-ink hover:bg-surface-2',
        'transition-colors duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1 focus-visible:ring-offset-bg',
      )}
    >
      {copied
        ? <Check className="size-3.5" aria-hidden="true" />
        : <Copy className="size-3.5" aria-hidden="true" />
      }
    </button>
  )
}

// ── MonoRow — a single on-chain fact row ──────────────────────────────────────

type MonoRowProps = {
  readonly label: string
  readonly value: string
  readonly copyValue?: string
  readonly href?: string
  readonly truncate?: boolean
}

function MonoRow({ label, value, copyValue, href, truncate = false }: MonoRowProps) {
  const displayValue = truncate && value.startsWith('0x') && value.length > 14
    ? `${value.slice(0, 10)}…${value.slice(-8)}`
    : value

  return (
    <div className="flex items-start justify-between gap-3 py-2.5 border-b border-border last:border-0">
      <dt className="font-sans text-xs text-ink-muted shrink-0 pt-0.5 w-28">{label}</dt>
      <dd className="flex items-center gap-1.5 min-w-0 flex-1">
        <span
          className="font-mono text-[0.8125rem] text-ink leading-snug break-all"
          title={value}
        >
          {displayValue}
        </span>
        <div className="flex items-center gap-0.5 shrink-0 ml-auto pl-1">
          {copyValue !== undefined && (
            <CopyButton value={copyValue} label={label} />
          )}
          {href && (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={`View ${label} on Basescan (opens in new tab)`}
              className={cn(
                'inline-flex items-center justify-center rounded-md',
                'size-7 shrink-0',
                'text-ink-muted hover:text-primary',
                'transition-colors duration-150',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1 focus-visible:ring-offset-bg',
              )}
            >
              <ExternalLink className="size-3.5" aria-hidden="true" />
            </a>
          )}
        </div>
      </dd>
    </div>
  )
}

// ── CertificateDetail ─────────────────────────────────────────────────────────

type CertificateDetailProps = {
  readonly certificate: CertificateOnChain
  readonly className?: string
}

export function CertificateDetail({ certificate, className }: CertificateDetailProps) {
  const {
    tokenId,
    title,
    recipientName,
    issuerName,
    description,
    metadataURI,
    issuedAt,
    recipient,
    txHash,
    blockNumber,
  } = certificate

  const issuedDateLabel = formatDate(issuedAt)
  const issuedDateISO = formatISODate(issuedAt)
  const contractAddr = CONTRACT_ADDRESS

  return (
    <article
      className={cn('space-y-6', className)}
      aria-label={`Certificate: ${title}`}
    >
      {/* ── Verified affirmation ──────────────────────────────────────────── */}
      <output
        className={cn(
          'flex items-center gap-2 px-3 py-2 rounded-lg',
          'bg-success/10 border border-success/20',
          'text-sm font-sans font-medium',
        )}
        aria-live="polite"
      >
        <ShieldCheck
          className="size-4 shrink-0 text-success"
          aria-hidden="true"
        />
        <span className="text-success">Verified on Base Sepolia</span>
      </output>

      {/* ── Certificate document ──────────────────────────────────────────── */}
      <div className="rounded-xl border border-border bg-surface p-6 md:p-8 space-y-6">

        {/* Document label — not an eyebrow kicker, just a quiet context line */}
        <p className="font-sans text-xs font-medium text-ink-muted">
          Certificate of Achievement
        </p>

        {/* Title — Fraunces serif, diploma gravitas */}
        <h1
          className={cn(
            'font-serif text-[clamp(1.75rem,4vw,3rem)]',
            'leading-tight tracking-[-0.02em] text-ink',
            'text-wrap-balance',
          )}
        >
          {title}
        </h1>

        {/* Core certificate fields */}
        <dl className="space-y-0 divide-y divide-border text-sm">
          <div className="flex items-baseline gap-3 py-2.5">
            <dt className="font-sans text-ink-muted shrink-0 w-32">Recipient</dt>
            <dd className="font-sans text-ink font-medium">{recipientName}</dd>
          </div>
          <div className="flex items-baseline gap-3 py-2.5">
            <dt className="font-sans text-ink-muted shrink-0 w-32">Issued by</dt>
            <dd className="font-sans text-ink font-medium">{issuerName}</dd>
          </div>
          <div className="flex items-baseline gap-3 py-2.5">
            <dt className="font-sans text-ink-muted shrink-0 w-32">Date issued</dt>
            <dd className="font-sans text-ink">
              {issuedDateISO
                ? <time dateTime={issuedDateISO}>{issuedDateLabel}</time>
                : issuedDateLabel}
            </dd>
          </div>
          {description && (
            <div className="flex items-start gap-3 py-2.5">
              <dt className="font-sans text-ink-muted shrink-0 w-32 pt-0">Description</dt>
              <dd className="font-sans text-ink leading-relaxed text-wrap-pretty">{description}</dd>
            </div>
          )}
          {metadataURI && (
            <div className="flex items-baseline gap-3 py-2.5">
              <dt className="font-sans text-ink-muted shrink-0 w-32">Metadata</dt>
              <dd className="font-mono text-[0.8125rem] text-ink break-all">{metadataURI}</dd>
            </div>
          )}
        </dl>

        {/* Soulbound badge */}
        <div className="flex items-center gap-1.5 text-ink-muted">
          <Award className="size-3.5 shrink-0" aria-hidden="true" />
          <span className="font-sans text-xs">Soulbound — non-transferable</span>
        </div>
      </div>

      {/* ── On-chain proof block ──────────────────────────────────────────── */}
      <div className="rounded-xl border border-border bg-surface p-6 space-y-1">
        <div className="flex items-center gap-2 mb-4">
          <CheckCircle className="size-4 shrink-0 text-primary" aria-hidden="true" />
          <h2 className="font-sans text-sm font-semibold text-ink">On-chain proof</h2>
        </div>

        <dl>
          {/* Recipient wallet */}
          <MonoRow
            label="Recipient wallet"
            value={recipient}
            copyValue={recipient}
            truncate
          />

          {/* Token ID */}
          <MonoRow
            label="Token ID"
            value={`#${tokenId.toString()}`}
            copyValue={tokenId.toString()}
            href={contractAddr ? nftExplorerUrl(contractAddr, tokenId) : undefined}
          />

          {/* Contract address */}
          {contractAddr && (
            <MonoRow
              label="Contract"
              value={truncateAddress(contractAddr)}
              copyValue={contractAddr}
              href={contractExplorerUrl(contractAddr)}
            />
          )}

          {/* Tx hash — only if event lookup succeeded */}
          {txHash && (
            <MonoRow
              label="Transaction"
              value={txHash}
              copyValue={txHash}
              href={txExplorerUrl(txHash)}
              truncate
            />
          )}

          {/* Block number */}
          {blockNumber !== undefined && (
            <MonoRow
              label="Block"
              value={blockNumber.toString()}
              copyValue={blockNumber.toString()}
            />
          )}
        </dl>
      </div>
    </article>
  )
}

// ── Skeleton (loading state matched to detail shape) ──────────────────────────

export function CertificateDetailSkeleton() {
  return (
    <div className="space-y-6" aria-busy="true" aria-label="Loading certificate">
      {/* Verified bar skeleton */}
      <div className="h-9 rounded-lg bg-surface-2 animate-pulse" aria-hidden="true" />

      {/* Document skeleton */}
      <div className="rounded-xl border border-border bg-surface p-6 md:p-8 space-y-6">
        <div className="h-3 w-32 rounded bg-surface-2 animate-pulse" />
        <div className="space-y-2">
          <div className="h-9 w-3/4 rounded bg-surface-2 animate-pulse" />
          <div className="h-7 w-1/2 rounded bg-surface-2 animate-pulse" />
        </div>
        <div className="space-y-3">
          {[80, 64, 48, 96].map((w) => (
            <div key={w} className="flex gap-3">
              <div className="h-4 w-28 rounded bg-surface-2 animate-pulse shrink-0" />
              <div className={`h-4 w-${w} rounded bg-surface-2 animate-pulse`} />
            </div>
          ))}
        </div>
      </div>

      {/* On-chain proof skeleton */}
      <div className="rounded-xl border border-border bg-surface p-6 space-y-3">
        <div className="h-4 w-28 rounded bg-surface-2 animate-pulse" />
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="flex justify-between gap-4 py-2">
            <div className="h-3 w-24 rounded bg-surface-2 animate-pulse" />
            <div className="h-3 w-40 rounded bg-surface-2 animate-pulse" />
          </div>
        ))}
      </div>
    </div>
  )
}
