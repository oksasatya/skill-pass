/**
 * My Certificates — /app/my-certificates
 * Placeholder: certificate grid + wallet-gated view ships in Phase 2c.
 */
export default function MyCertificatesPage() {
  return (
    <section className="space-y-6">
      <header>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          My certificates
        </h1>
        <p className="mt-1 font-sans text-sm text-ink-muted">
          Credentials issued to your wallet — coming in Phase 2c.
        </p>
      </header>
      <div className="rounded-xl border border-dashed border-border bg-surface px-6 py-10 text-center">
        <p className="font-sans text-sm text-ink-muted">Connect your wallet to see certificates.</p>
      </div>
    </section>
  )
}
