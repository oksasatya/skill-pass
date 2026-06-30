/**
 * IssueForm — the admin certificate issuance form.
 *
 * Design rules (DESIGN.md / PRODUCT.md):
 *   - Real semantic elements: <form>, <label>, <input>, <textarea>, <button>
 *   - Mono for on-chain data (addresses, token ids, tx hashes)
 *   - Privacy disclosure + acknowledgement checkbox is MANDATORY before submit
 *   - Success: token id (mono) + tx hash (mono) + Basescan link + "view certificate" link
 *   - Error: surface revert/reject reason + retry
 *   - Pending/confirming: spinner + disabled, label stays
 *   - Sonar-TS: readonly props, no nested ternaries, real elements, globalThis, optional chaining
 */

import { useState, useId, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Loader2, CheckCircle2, AlertCircle, ExternalLink, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { FIELD_LIMITS, validateFields } from '@/lib/validateCertificate'
import type { CertificateFields, FieldErrors } from '@/lib/validateCertificate'
import type { UseIssueCertificateResult } from '@/hooks/useIssueCertificate'

type IssueFormProps = {
  readonly issuer: UseIssueCertificateResult
}

// ── Field component ────────────────────────────────────────────────────────────

type FieldProps = {
  readonly id: string
  readonly label: string
  readonly hint?: string
  readonly error?: string
  readonly required?: boolean
  readonly children: React.ReactNode
}

function Field({ id, label, hint, error, required = false, children }: FieldProps) {
  return (
    <div className="space-y-1.5">
      <label
        htmlFor={id}
        className="block text-sm font-medium text-ink"
      >
        {label}
        {required && (
          <span className="ml-1 text-danger" aria-hidden="true">*</span>
        )}
      </label>
      {hint && (
        <p id={`${id}-hint`} className="text-xs text-ink-muted">
          {hint}
        </p>
      )}
      {children}
      {error && (
        <p id={`${id}-error`} role="alert" className="flex items-center gap-1.5 text-xs text-danger">
          <AlertCircle className="size-3 shrink-0" aria-hidden="true" />
          {error}
        </p>
      )}
    </div>
  )
}

// ── Character counter ──────────────────────────────────────────────────────────

type CharCountProps = {
  readonly current: number
  readonly max: number
}

function CharCount({ current, max }: CharCountProps) {
  const over = current > max
  return (
    <span className={cn('text-xs tabular-nums', over ? 'text-danger' : 'text-ink-muted')}>
      {current}/{max}
    </span>
  )
}

// ── Copy button (for mono data in success state) ───────────────────────────────

function CopyButton({ value }: { readonly value: string }) {
  const [copied, setCopied] = useState(false)

  function handleCopy() {
    globalThis.navigator?.clipboard?.writeText(value).then(() => {
      setCopied(true)
      globalThis.setTimeout(() => setCopied(false), 1500)
    })
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label="Copy to clipboard"
      className="text-ink-muted hover:text-ink transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
    >
      <Copy className="size-3.5" aria-hidden="true" />
      {copied && <span className="sr-only">Copied</span>}
    </button>
  )
}

// ── Input style ────────────────────────────────────────────────────────────────

const inputCls = cn(
  'w-full rounded-lg border border-border bg-bg px-3 py-2',
  'text-sm text-ink placeholder:text-ink-muted/60',
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:border-primary',
  'aria-invalid:border-danger aria-invalid:ring-2 aria-invalid:ring-danger/30',
  'transition-colors duration-150',
  'min-h-[44px]', // touch target
)

// ── Main component ────────────────────────────────────────────────────────────

export function IssueForm({ issuer }: IssueFormProps) {
  const uid = useId()
  const id = (name: string) => `${uid}-${name}`

  const [fields, setFields] = useState<CertificateFields>({
    recipient: '',
    title: '',
    recipientName: '',
    issuerName: '',
    description: '',
    metadataURI: '',
  })
  const [errors, setErrors] = useState<FieldErrors>({})
  const [acknowledged, setAcknowledged] = useState(false)
  const [touched, setTouched] = useState(false)

  const { issue, status, tokenId, txHash, error: issueError, reset } = issuer

  const isSubmitting = status === 'pending' || status === 'confirming'
  const submitDisabled = isSubmitting || !acknowledged

  function updateField(name: keyof CertificateFields, value: string) {
    const next = { ...fields, [name]: value }
    setFields(next)
    if (touched) {
      setErrors(validateFields(next))
    }
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setTouched(true)
    const fieldErrors = validateFields(fields)
    setErrors(fieldErrors)
    if (Object.keys(fieldErrors).length > 0) return
    if (!acknowledged) return
    issue(fields)
  }

  function handleReset() {
    reset()
    setFields({ recipient: '', title: '', recipientName: '', issuerName: '', description: '', metadataURI: '' })
    setErrors({})
    setAcknowledged(false)
    setTouched(false)
  }

  // ── Success state ──────────────────────────────────────────────────────────
  if (status === 'success' && txHash) {
    const tokenIdStr = tokenId !== undefined ? tokenId.toString() : undefined
    return (
      <div
        role="status"
        aria-live="polite"
        className={cn(
          'rounded-xl border border-success/30 bg-success/5 p-6 space-y-5',
          'motion-safe:animate-in motion-safe:fade-in-0 motion-safe:slide-in-from-bottom-1 motion-safe:duration-200',
        )}
      >
        {/* Header */}
        <div className="flex items-start gap-3">
          <CheckCircle2 className="size-5 text-success shrink-0 mt-0.5" aria-hidden="true" />
          <div>
            <h2 className="text-base font-semibold text-ink">Certificate issued</h2>
            <p className="text-sm text-ink-muted mt-0.5">
              The soulbound token was minted to the recipient's wallet.
            </p>
          </div>
        </div>

        {/* On-chain facts */}
        <dl className="space-y-3 text-sm">
          {tokenIdStr && (
            <div>
              <dt className="text-xs text-ink-muted uppercase tracking-wide">Token ID</dt>
              <dd className="flex items-center gap-2 mt-0.5">
                <span className="font-mono text-ink">{tokenIdStr}</span>
                <CopyButton value={tokenIdStr} />
              </dd>
            </div>
          )}
          <div>
            <dt className="text-xs text-ink-muted uppercase tracking-wide">Transaction hash</dt>
            <dd className="flex items-center gap-2 mt-0.5 min-w-0">
              <span className="font-mono text-xs text-ink break-all">{txHash}</span>
              <CopyButton value={txHash} />
            </dd>
          </div>
        </dl>

        {/* Links */}
        <div className="flex flex-wrap gap-3 pt-1">
          <a
            href={`https://sepolia.basescan.org/tx/${txHash}`}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          >
            View on Basescan
            <ExternalLink className="size-3.5" aria-hidden="true" />
          </a>
          {tokenIdStr && (
            <Link
              to={`/certificates/${tokenIdStr}`}
              className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            >
              View certificate
              <ExternalLink className="size-3.5" aria-hidden="true" />
            </Link>
          )}
        </div>

        {/* Issue another */}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleReset}
          className="min-h-[44px]"
        >
          Issue another certificate
        </Button>
      </div>
    )
  }

  // ── Error state ────────────────────────────────────────────────────────────
  const hasIssueError = status === 'error' && issueError
  const errorMessage = extractErrorMessage(issueError)

  // ── Form ───────────────────────────────────────────────────────────────────
  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-label="Issue a certificate"
      className="space-y-6"
    >
      {/* Error banner */}
      {hasIssueError && (
        <div
          role="alert"
          aria-live="assertive"
          className="flex items-start gap-3 rounded-lg border border-danger/30 bg-danger/5 px-4 py-3"
        >
          <AlertCircle className="size-4 text-danger shrink-0 mt-0.5" aria-hidden="true" />
          <div className="space-y-1 min-w-0">
            <p className="text-sm font-medium text-danger">Transaction failed</p>
            <p className="text-xs text-danger/80 break-words">{errorMessage}</p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleReset}
            className="ml-auto shrink-0 min-h-[44px] text-danger hover:text-danger"
          >
            Retry
          </Button>
        </div>
      )}

      {/* Recipient */}
      <Field
        id={id('recipient')}
        label="Recipient wallet address"
        required
        error={errors.recipient}
        hint="The wallet that will receive this soulbound certificate."
      >
        <input
          id={id('recipient')}
          type="text"
          inputMode="text"
          autoComplete="off"
          spellCheck={false}
          placeholder="0x…"
          value={fields.recipient}
          onChange={(e) => updateField('recipient', e.target.value)}
          aria-required="true"
          aria-invalid={!!errors.recipient}
          aria-describedby={errors.recipient ? `${id('recipient')}-error` : `${id('recipient')}-hint`}
          className={cn(inputCls, 'font-mono text-xs')}
          disabled={isSubmitting}
        />
      </Field>

      {/* Title */}
      <Field
        id={id('title')}
        label="Certificate title"
        required
        error={errors.title}
      >
        <div className="space-y-1">
          <input
            id={id('title')}
            type="text"
            placeholder="Go Backend Bootcamp — Completion"
            value={fields.title}
            onChange={(e) => updateField('title', e.target.value)}
            aria-required="true"
            aria-invalid={!!errors.title}
            aria-describedby={errors.title ? `${id('title')}-error` : undefined}
            className={inputCls}
            disabled={isSubmitting}
          />
          <div className="flex justify-end">
            <CharCount current={fields.title.length} max={FIELD_LIMITS.title} />
          </div>
        </div>
      </Field>

      {/* Recipient name */}
      <Field
        id={id('recipientName')}
        label="Recipient name"
        required
        error={errors.recipientName}
        hint="Will appear on the certificate exactly as entered."
      >
        <div className="space-y-1">
          <input
            id={id('recipientName')}
            type="text"
            placeholder="Alice Nguyen"
            value={fields.recipientName}
            onChange={(e) => updateField('recipientName', e.target.value)}
            aria-required="true"
            aria-invalid={!!errors.recipientName}
            aria-describedby={errors.recipientName ? `${id('recipientName')}-error` : `${id('recipientName')}-hint`}
            className={inputCls}
            disabled={isSubmitting}
          />
          <div className="flex justify-end">
            <CharCount current={fields.recipientName.length} max={FIELD_LIMITS.recipientName} />
          </div>
        </div>
      </Field>

      {/* Issuer name */}
      <Field
        id={id('issuerName')}
        label="Issuer name"
        required
        error={errors.issuerName}
        hint="Your organization, program, or community name."
      >
        <div className="space-y-1">
          <input
            id={id('issuerName')}
            type="text"
            placeholder="Hacktiv8"
            value={fields.issuerName}
            onChange={(e) => updateField('issuerName', e.target.value)}
            aria-required="true"
            aria-invalid={!!errors.issuerName}
            aria-describedby={errors.issuerName ? `${id('issuerName')}-error` : `${id('issuerName')}-hint`}
            className={inputCls}
            disabled={isSubmitting}
          />
          <div className="flex justify-end">
            <CharCount current={fields.issuerName.length} max={FIELD_LIMITS.issuerName} />
          </div>
        </div>
      </Field>

      {/* Description */}
      <Field
        id={id('description')}
        label="Description"
        required
        error={errors.description}
      >
        <div className="space-y-1">
          <textarea
            id={id('description')}
            rows={4}
            placeholder="Awarded for successfully completing the 60-hour Go backend development program, covering REST APIs, hexagonal architecture, and Postgres."
            value={fields.description}
            onChange={(e) => updateField('description', e.target.value)}
            aria-required="true"
            aria-invalid={!!errors.description}
            aria-describedby={errors.description ? `${id('description')}-error` : undefined}
            className={cn(inputCls, 'resize-y min-h-[100px]')}
            disabled={isSubmitting}
          />
          <div className="flex justify-end">
            <CharCount current={fields.description.length} max={FIELD_LIMITS.description} />
          </div>
        </div>
      </Field>

      {/* Metadata URI (optional) */}
      <Field
        id={id('metadataURI')}
        label="Metadata URI"
        error={errors.metadataURI}
        hint="Optional. An IPFS or HTTPS link to the certificate's JSON metadata."
      >
        <div className="space-y-1">
          <input
            id={id('metadataURI')}
            type="url"
            placeholder="ipfs://… or https://…"
            value={fields.metadataURI}
            onChange={(e) => updateField('metadataURI', e.target.value)}
            aria-invalid={!!errors.metadataURI}
            aria-describedby={errors.metadataURI ? `${id('metadataURI')}-error` : `${id('metadataURI')}-hint`}
            className={inputCls}
            disabled={isSubmitting}
          />
          <div className="flex justify-end">
            <CharCount current={fields.metadataURI.length} max={FIELD_LIMITS.metadataURI} />
          </div>
        </div>
      </Field>

      {/* ── Privacy disclosure (MANDATORY — HARD GATE) ────────────────────── */}
      <div
        className={cn(
          'rounded-lg border px-4 py-4 space-y-3',
          acknowledged
            ? 'border-border bg-surface'
            : 'border-warning/40 bg-warning/5',
        )}
        role="group"
        aria-labelledby={id('privacy-heading')}
      >
        <div className="flex items-start gap-2">
          <AlertCircle
            className="size-4 text-warning shrink-0 mt-0.5"
            aria-hidden="true"
          />
          <div className="space-y-1.5">
            <p
              id={id('privacy-heading')}
              className="text-sm font-semibold text-ink"
            >
              This data will be permanent and public on-chain
            </p>
            <p className="text-sm text-ink-muted">
              The <strong>recipient name</strong>, <strong>title</strong>, and{' '}
              <strong>description</strong> you enter will be written to the Base Sepolia
              blockchain. Once confirmed, this data{' '}
              <strong className="text-ink">cannot be changed or removed</strong>. Anyone
              with the token ID or recipient address can read it.
            </p>
          </div>
        </div>
        <label className="flex items-start gap-2.5 cursor-pointer group">
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            className={cn(
              'mt-0.5 size-4 rounded border-border accent-primary',
              'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
              'cursor-pointer',
            )}
            aria-required="true"
            aria-describedby={id('privacy-heading')}
            disabled={isSubmitting}
          />
          <span className="text-sm text-ink group-hover:text-ink transition-colors">
            I understand this data is permanent and public on-chain, and I consent to
            issuing this certificate.
          </span>
        </label>
      </div>

      {/* Submit */}
      <Button
        type="submit"
        variant="default"
        disabled={submitDisabled}
        className="min-h-[44px] w-full sm:w-auto gap-2"
      >
        {isSubmitting && (
          <Loader2 className="size-4 animate-spin" aria-hidden="true" />
        )}
        {statusLabel(status)}
      </Button>
    </form>
  )
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function statusLabel(status: IssueStatus): string {
  if (status === 'pending') return 'Waiting for wallet…'
  if (status === 'confirming') return 'Confirming on-chain…'
  return 'Issue certificate'
}

type IssueStatus = 'idle' | 'pending' | 'confirming' | 'success' | 'error'

function extractErrorMessage(err: Error | null): string {
  if (!err) return 'Unknown error.'
  // Surface user-friendly messages for known revert reasons
  const msg = err.message
  if (msg.includes('ZeroRecipient')) return 'Recipient address cannot be the zero address.'
  if (msg.includes('StringTooLong')) return 'One or more fields exceed the on-chain character limit.'
  if (msg.includes('OwnableUnauthorizedAccount')) return 'Only the contract owner can issue certificates.'
  if (msg.includes('User rejected') || msg.includes('user rejected')) return 'Transaction rejected in wallet.'
  if (msg.includes('Soulbound')) return 'Certificate is soulbound and cannot be transferred.'
  // ponytail: truncate very long error messages from the wallet/RPC
  if (msg.length > 200) return `${msg.slice(0, 200)}…`
  return msg
}
