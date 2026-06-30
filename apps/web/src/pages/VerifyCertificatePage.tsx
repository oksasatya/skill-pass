import { useParams } from 'react-router-dom'

/**
 * Public verify — /certificates/:tokenId
 * Standalone: no AppShell chrome, optimized to be shared and read cold.
 * Full CertificateDetail + on-chain verification ships in Phase 2c.
 */
export default function VerifyCertificatePage() {
  const { tokenId } = useParams<{ tokenId: string }>()

  return (
    <div className="min-h-svh bg-bg text-ink flex flex-col items-center px-4 py-16">
      <div className="w-full max-w-xl space-y-8">
        {/* Minimal wordmark — no full nav, standalone per DESIGN.md */}
        <a
          href="/"
          className="inline-flex items-center gap-2 text-ink-muted hover:text-ink transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-lg"
          aria-label="SkillPass — home"
        >
          <span className="size-4 rounded-[3px] bg-primary" aria-hidden="true" />
          <span className="font-sans text-sm font-semibold tracking-tight">SkillPass</span>
        </a>

        {/* Placeholder certificate card */}
        <article
          className="rounded-xl border border-border bg-surface p-8 space-y-4"
          aria-label={`Certificate #${tokenId ?? '—'}`}
        >
          <p className="font-sans text-xs font-medium text-ink-muted uppercase tracking-widest">
            Certificate
          </p>
          <h1 className="font-serif text-[clamp(1.75rem,4vw,3rem)] leading-tight tracking-[-0.02em] text-ink">
            Verification pending
          </h1>
          <p className="font-sans text-sm text-ink-muted">
            Token ID:{' '}
            <span className="font-mono text-[0.9375rem] text-ink">{tokenId ?? '—'}</span>
          </p>
          <p className="font-sans text-sm text-ink-muted">
            On-chain verification ships in Phase 2c.
          </p>
        </article>
      </div>
    </div>
  )
}
