/**
 * Landing page — Task 6.
 * Standalone (outside AppShell). No wallet required.
 *
 * Sections:
 *   1. Nav  — minimal wordmark + theme toggle + CTA link
 *   2. Hero — display headline + mock certificate artifact (the signature element) + primary CTA
 *   3. How it works — genuine 3-step sequence (issue → own → verify)
 *   4. Audience — who this is for, in prose not identical cards
 *   5. Footer strip — minimal
 *
 * DESIGN.md bans honored:
 *   ✓ No eyebrow-above-every-section
 *   ✓ No identical icon-card grid
 *   ✓ No gradient text
 *   ✓ No side-stripe borders
 *   ✓ No hero-metric template
 *   ✓ Fraunces serif used ONLY in the certificate artifact
 */

import { Link } from 'react-router-dom'
import { ThemeToggle } from '@/components/layout/ThemeToggle'

// Stable mock data — makes the cert artifact feel real without dynamic data
const MOCK_CERT = {
  title: 'Solidity Fundamentals',
  issuer: 'Base Developer DAO',
  recipient: 'alex.eth',
  tokenId: '0x0000...0042',
  issuedDate: '2024-12-01',
  contractSnippet: '0xAbC1…dEf9',
} as const

export default function LandingPage() {
  return (
    <div className="min-h-svh bg-bg text-ink flex flex-col">

      {/* ── Minimal landing nav ──────────────────────────────────────────── */}
      <header className="sticky top-0 z-10 border-b border-border bg-bg/90 backdrop-blur-sm">
        <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 flex h-14 items-center justify-between gap-4">
          {/* Logo mark + wordmark */}
          <Link
            to="/"
            className="flex shrink-0 items-center gap-2 rounded-lg px-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="SkillPass — home"
          >
            <img
              src="/logo.webp"
              alt=""
              aria-hidden="true"
              width={32}
              height={32}
              className="size-8 shrink-0"
            />
            <span className="font-sans text-base font-semibold tracking-tight text-ink">
              SkillPass
            </span>
          </Link>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            <Link
              to="/app"
              className="inline-flex items-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-ink min-h-[44px] hover:bg-primary/85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors"
            >
              Open app
            </Link>
          </div>
        </div>
      </header>

      <main>
        {/* ── Hero ──────────────────────────────────────────────────────── */}
        <section
          aria-label="Introduction"
          className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 pt-16 pb-20 md:pt-24 md:pb-28 grid md:grid-cols-2 gap-10 md:gap-16 items-center"
        >
          {/* Left: headline + CTA */}
          <div className="flex flex-col gap-6">
            <h1 className="font-sans text-[clamp(2rem,5vw,3.5rem)] font-semibold leading-[1.1] tracking-[-0.03em] text-ink text-wrap-balance">
              Credentials anyone can verify.
            </h1>
            <p className="font-sans text-base md:text-lg text-ink-muted leading-relaxed max-w-[52ch]">
              SkillPass issues soulbound certificates to wallets and anchors proof
              on Base. Anyone — recruiter, peer, community — can confirm a credential
              is real without an account.
            </p>
            <div className="flex flex-wrap gap-3 pt-2">
              <Link
                to="/app"
                className="inline-flex items-center rounded-lg bg-primary px-6 py-3 text-base font-medium text-primary-ink min-h-[48px] hover:bg-primary/85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors"
              >
                Issue a certificate
              </Link>
              <Link
                to="/app/my-certificates"
                className="inline-flex items-center rounded-lg border border-border bg-surface px-6 py-3 text-base font-medium text-ink min-h-[48px] hover:bg-surface-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors"
              >
                View mine
              </Link>
            </div>
          </div>

          {/* Right: mock certificate artifact — the signature element */}
          <div
            aria-hidden="true"
            className="relative select-none"
          >
            {/* Outer document container */}
            <div className="rounded-xl border border-border bg-surface shadow-sm p-6 md:p-8 flex flex-col gap-5">

              {/* Certificate header mark */}
              <div className="flex items-center gap-2.5">
                <span className="size-4 rounded-[2px] bg-primary shrink-0" />
                <span className="font-sans text-xs font-medium text-ink-muted tracking-wide uppercase">
                  SkillPass Certificate
                </span>
              </div>

              {/* Certificate title — the ONE place Fraunces appears */}
              <div>
                <p className="font-sans text-xs text-ink-muted mb-1">Certificate of Completion</p>
                <h2 className="font-serif text-[clamp(1.5rem,3vw,2rem)] font-semibold leading-snug text-ink text-wrap-balance">
                  {MOCK_CERT.title}
                </h2>
              </div>

              {/* Issuer / recipient row */}
              <dl className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="font-sans text-xs text-ink-muted mb-0.5">Issued by</dt>
                  <dd className="font-sans text-sm font-medium text-ink">{MOCK_CERT.issuer}</dd>
                </div>
                <div>
                  <dt className="font-sans text-xs text-ink-muted mb-0.5">Recipient</dt>
                  <dd className="font-sans text-sm font-medium text-ink">{MOCK_CERT.recipient}</dd>
                </div>
              </dl>

              {/* Divider */}
              <div className="border-t border-border" />

              {/* On-chain data — mono signals "exact, verifiable" */}
              <dl className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="font-sans text-xs text-ink-muted mb-0.5">Token ID</dt>
                  <dd className="font-mono text-xs text-ink">{MOCK_CERT.tokenId}</dd>
                </div>
                <div>
                  <dt className="font-sans text-xs text-ink-muted mb-0.5">Contract</dt>
                  <dd className="font-mono text-xs text-ink">{MOCK_CERT.contractSnippet}</dd>
                </div>
              </dl>

              {/* Verification affirmation */}
              <div className="flex items-center gap-2 rounded-lg bg-primary-weak px-3 py-2">
                <span className="text-primary text-sm font-medium" aria-hidden="true">✓</span>
                <span className="font-sans text-sm font-medium text-primary">
                  Verified on Base · block 14 208 903
                </span>
              </div>
            </div>
          </div>
        </section>

        {/* ── How it works — genuine ordered sequence ────────────────────── */}
        <section
          aria-label="How SkillPass works"
          className="border-t border-border bg-surface"
        >
          <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 py-16 md:py-20">
            <h2 className="font-sans text-2xl font-semibold text-ink mb-12">
              How it works
            </h2>

            {/* Steps as an ordered list — genuinely sequential, numbering is true here */}
            <ol className="grid md:grid-cols-3 gap-0 md:gap-px relative">
              {/* Horizontal connector line on md+ */}
              <li className="relative flex flex-col gap-4 pb-10 md:pb-0 md:pr-8 border-l-2 md:border-l-0 md:border-none border-border pl-8 md:pl-0">
                <StepConnector step={1} />
                <StepNumber n={1} />
                <div className="flex flex-col gap-2">
                  <h3 className="font-sans text-lg font-semibold text-ink">Issue</h3>
                  <p className="font-sans text-sm text-ink-muted leading-relaxed max-w-[36ch]">
                    Connect your wallet as an issuer and mint a certificate to any
                    wallet address. The credential is soulbound — it cannot be transferred.
                  </p>
                </div>
              </li>

              <li className="relative flex flex-col gap-4 pb-10 md:pb-0 md:px-8 border-l-2 md:border-l-0 border-border pl-8 md:pl-8">
                <StepConnector step={2} />
                <StepNumber n={2} />
                <div className="flex flex-col gap-2">
                  <h3 className="font-sans text-lg font-semibold text-ink">Own</h3>
                  <p className="font-sans text-sm text-ink-muted leading-relaxed max-w-[36ch]">
                    Recipients connect their wallet to see every certificate they hold,
                    with the issuer, date, and on-chain proof for each.
                  </p>
                </div>
              </li>

              <li className="relative flex flex-col gap-4 md:pl-8 border-l-2 md:border-l-0 border-border pl-8">
                <StepConnector step={3} last />
                <StepNumber n={3} />
                <div className="flex flex-col gap-2">
                  <h3 className="font-sans text-lg font-semibold text-ink">Verify</h3>
                  <p className="font-sans text-sm text-ink-muted leading-relaxed max-w-[36ch]">
                    Anyone with a certificate link can confirm it on-chain — no wallet,
                    no account, no friction. The proof points straight to the block explorer.
                  </p>
                </div>
              </li>
            </ol>
          </div>
        </section>

        {/* ── Who it's for ──────────────────────────────────────────────── */}
        <section
          aria-label="Who SkillPass is for"
          className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 py-16 md:py-20"
        >
          <h2 className="font-sans text-2xl font-semibold text-ink mb-10">
            Built for three roles, one credential
          </h2>

          {/* Prose description list — not identical cards */}
          <dl className="divide-y divide-border">
            <AudienceItem
              heading="Issuers"
              description="Bootcamps, course creators, developer communities, and open-source maintainers. You ran a program and want to hand participants a tamper-proof credential that outlasts any platform. Connect a wallet, fill out a form, mint. That's it."
            />
            <AudienceItem
              heading="Recipients"
              description="Participants, contributors, attendees. Connect your wallet to see every credential you have earned, with full issuer details and the on-chain record behind each one. Share a link — the proof travels with it."
            />
            <AudienceItem
              heading="Verifiers"
              description="Recruiters, peers, community members. You received a link. No account required. Open it, read the certificate, and follow the block explorer link to confirm the credential is real and belongs to who it claims."
            />
          </dl>
        </section>

        {/* ── CTA strip ─────────────────────────────────────────────────── */}
        <section
          aria-label="Get started"
          className="border-t border-border bg-surface"
        >
          <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 py-14 md:py-16 flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
            <div className="flex flex-col gap-1">
              <p className="font-sans text-xl font-semibold text-ink">
                Ready to issue your first certificate?
              </p>
              <p className="font-sans text-sm text-ink-muted">
                Connect a wallet and mint in under a minute. No backend required.
              </p>
            </div>
            <Link
              to="/app"
              className="inline-flex shrink-0 items-center rounded-lg bg-primary px-6 py-3 text-base font-medium text-primary-ink min-h-[48px] hover:bg-primary/85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-colors"
            >
              Open app
            </Link>
          </div>
        </section>
      </main>

      {/* ── Footer ──────────────────────────────────────────────────────── */}
      <footer className="border-t border-border mt-auto">
        <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8 py-6 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="size-4 rounded-[2px] bg-primary shrink-0" aria-hidden="true" />
            <span className="font-sans text-sm font-medium text-ink">SkillPass</span>
          </div>
          <p className="font-sans text-xs text-ink-muted">
            Certificates anchored on{' '}
            <a
              href="https://base.org"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline underline-offset-2 hover:text-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              Base
            </a>
            . Non-transferable. Permanent.
          </p>
        </div>
      </footer>
    </div>
  )
}

// ── Sub-components ────────────────────────────────────────────────────────────

/**
 * Numbered step circle — the step indicator for the "How it works" sequence.
 */
function StepNumber({ n }: { readonly n: number }) {
  return (
    <div
      className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary-weak text-primary font-sans text-sm font-semibold md:mb-2"
      aria-hidden="true"
    >
      {n}
    </div>
  )
}

/**
 * Horizontal connector line visible on md+ between steps.
 * Renders a subtle line from the step number rightward.
 * ponytail: pure CSS, no animation needed here
 */
function StepConnector({ step: _step, last = false }: { readonly step: number; readonly last?: boolean }) {
  if (last) return null
  return (
    <div
      className="hidden md:block absolute top-4 left-[calc(100%_-_2rem)] w-[calc(100%_-_2rem)] h-px bg-border"
      aria-hidden="true"
    />
  )
}

/**
 * Audience description item for the "Who it's for" section.
 * A definition list row: role term + description.
 * Deliberately NOT a card — prose layout is the differentiator.
 */
function AudienceItem({
  heading,
  description,
}: {
  readonly heading: string
  readonly description: string
}) {
  return (
    <div className="grid md:grid-cols-[10rem_1fr] gap-2 md:gap-8 py-6">
      <dt className="font-sans text-sm font-semibold text-ink pt-0.5">{heading}</dt>
      <dd className="font-sans text-sm text-ink-muted leading-relaxed">{description}</dd>
    </div>
  )
}
