/**
 * Dashboard — /app
 * Placeholder: My Certificates grid ships in Phase 2c.
 */
export default function DashboardPage() {
  return (
    <section className="space-y-6">
      <header>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          Dashboard
        </h1>
        <p className="mt-1 font-sans text-sm text-ink-muted">
          Your issued and received certificates — coming in Phase 2c.
        </p>
      </header>
      <div className="rounded-xl border border-border bg-surface px-6 py-10 text-center">
        <p className="font-sans text-sm text-ink-muted">No certificates yet.</p>
      </div>
    </section>
  )
}
