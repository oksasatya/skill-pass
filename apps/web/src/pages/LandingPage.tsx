/**
 * Landing page — Task 6 placeholder.
 * Real content (hero, CTA, product overview) ships in Task 6.
 */
export default function LandingPage() {
  return (
    <div className="min-h-svh bg-bg text-ink flex flex-col items-center justify-center px-4 py-20">
      <div className="max-w-md text-center space-y-4">
        <div className="inline-flex size-12 items-center justify-center rounded-xl bg-primary-weak mb-2">
          <span className="size-5 rounded-[3px] bg-primary" aria-hidden="true" />
        </div>
        <h1 className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink tracking-tight">
          SkillPass
        </h1>
        <p className="font-sans text-base text-ink-muted leading-relaxed">
          Verifiable on-chain credentials. Coming in Task 6.
        </p>
        <a
          href="/app"
          className="inline-flex items-center rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-ink min-h-[44px] hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors"
        >
          Go to dashboard
        </a>
      </div>
    </div>
  )
}
