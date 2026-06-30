# Product

## Register

product

## Users

Three audiences share one onchain artifact — the certificate.

- **Issuers / Admins** — bootcamps, course creators, developer communities, open-source maintainers, internal orgs. Context: they have just finished running a program and want to award a credible, tamper-proof credential to participants. They work from a desktop, deliberately, a few certificates at a time. Primary job: issue a certificate to a recipient's wallet and walk away confident it landed.
- **Recipients / Users** — course participants, GitHub contributors, community members, event attendees. Context: they connect a wallet to see what they have earned. Often mobile, often Web3-curious rather than Web3-native. Primary job: see my certificates and open one to read it.
- **Verifiers / Visitors** — anyone handed a certificate link (a recruiter, a peer, a community). Context: no wallet, no account, often skeptical. Primary job: confirm this credential is real, on-chain, and belongs to who it claims.

## Product Purpose

SkillPass issues **soulbound (non-transferable) certificates** to wallets and lets anyone view and verify them on-chain. It proves a platform can mint a credential, anchor the proof on a public blockchain (Base Sepolia), and surface it in a dashboard and a public verification page — without a backend in this phase. Success: an issuer mints a certificate in under a minute and a stranger can verify it from a link with zero friction and full confidence.

## Brand Personality

Trustworthy, precise, quietly modern. Three words: **credible, clear, exact.** The voice is plain and confident — it states facts (token id, issuer, block) without hype. It should feel closer to a passport or a diploma than to a crypto app: the value proposition is trust, and the interface earns it by being legible and honest, never by being loud.

## Anti-references

- Generic Web3 / crypto-exchange aesthetics: neon-on-dark-navy, glowing coins, casino energy, gradient text, "to the moon" maximalism.
- The cream / sand / beige "editorial-warm" AI default body background.
- Templated SaaS dashboards: identical stat-card grids, big-number hero metrics, an uppercase tracked eyebrow above every section.
- Anything that makes a permanent credential feel disposable or speculative (price tickers, token imagery, marketplace framing).

## Design Principles

1. **Trust through clarity.** Legibility and honest information hierarchy are the brand. If a fact matters for verification (issuer, recipient, token id, on-chain proof), it is unambiguous and easy to find.
2. **The certificate is the hero.** Every screen exists to present or confirm the credential. The cert artifact gets the craft; chrome stays quiet.
3. **Verifiable by anyone.** The public path works with no wallet, no login, and always offers the on-chain proof (contract, token id, explorer link). Never ask the verifier to trust us — point them at the chain.
4. **Restraint over flash.** One accent, generous whitespace, no decorative effects that don't carry meaning. The product register: design serves the task.
5. **Fast and accessible.** Loading, error, and empty states are designed, not afterthoughts. WCAG AA, keyboard-complete, reduced-motion honored, status never conveyed by color alone.

## Accessibility & Inclusion

WCAG 2.2 AA. Body text ≥ 4.5:1 contrast; status communicated by text + icon, not color alone (color-blind safe). Full keyboard operability with visible focus rings. `prefers-reduced-motion` honored on every transition. Mobile-first: touch targets ≥ 44px, inputs ≥ 16px (no iOS zoom), wallet/network prompts reachable in the thumb zone.
