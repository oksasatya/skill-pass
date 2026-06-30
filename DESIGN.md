# Design

> Visual system for SkillPass. Register: **product** (design serves the task). Personality: **clean, trustworthy, exact**. Color strategy: **restrained** — tinted neutrals + a single accent. All colors in OKLCH. Anchor: Base-blue `#0052FF` (the network's own color — a deliberate, meaningful primary, carried as the one accent, never as a glow).

## Theme

Light is the default (a credential is read in daylight, on a recruiter's laptop, in a community Discord — bright, neutral, document-like). Dark mode is a first-class peer, toggled and persisted, for the Web3-native recipient. Both share one token contract; only the values swap. Neutrals carry a faint **cool** tint toward the brand hue (264) — never warm/cream (the AI default), never generic web3 navy.

## Color Palette (OKLCH)

Base-blue `#0052FF` ≈ `oklch(0.55 0.24 264)`. Confirm the exact conversion in code; treat this as the primary anchor.

### Light (`:root`)
```css
--bg:            oklch(0.992 0.002 264);  /* page */
--surface:       oklch(0.975 0.004 264);  /* cards, raised */
--surface-2:     oklch(0.955 0.005 264);  /* inset, hover */
--border:        oklch(0.915 0.006 264);
--ink:           oklch(0.235 0.020 264);  /* body text — ≥ 4.5:1 on bg */
--ink-muted:     oklch(0.505 0.018 264);  /* secondary — verify ≥ 4.5:1 on surface */
--primary:       oklch(0.550 0.240 264);  /* Base-blue, the one accent */
--primary-ink:   oklch(0.992 0.010 264);  /* text on primary */
--primary-weak:  oklch(0.950 0.030 264);  /* tinted info/selected bg */
--success:       oklch(0.560 0.130 150);  /* verified */
--danger:        oklch(0.560 0.200 25);   /* error, wrong network */
--warning:       oklch(0.720 0.150 80);
--focus:         var(--primary);
```

### Dark (`.dark`)
```css
--bg:            oklch(0.185 0.015 264);  /* refined near-black, NOT #0a0e27 navy */
--surface:       oklch(0.225 0.018 264);
--surface-2:     oklch(0.265 0.020 264);
--border:        oklch(0.310 0.020 264);
--ink:           oklch(0.960 0.010 264);
--ink-muted:     oklch(0.720 0.015 264);
--primary:       oklch(0.640 0.215 264);  /* lifted for dark contrast */
--primary-ink:   oklch(0.165 0.020 264);
--primary-weak:  oklch(0.300 0.060 264);
--success:       oklch(0.700 0.140 150);
--danger:        oklch(0.660 0.190 25);
--warning:       oklch(0.800 0.140 80);
```

**Contrast rules (hard):** body text ≥ 4.5:1, large ≥ 3:1, placeholders 4.5:1. `--ink-muted` is for secondary labels only; never body copy on a tinted surface if it dips below 4.5:1 — bump toward `--ink`. **Status is never color alone** — pair every state with text + an icon (✓ verified, ⚠ wrong network, ✕ error).

## Typography

A three-role system on a deliberate contrast axis (sans + serif + mono) — never two similar sans.

- **Sans — UI & body:** `Inter` (variable). All controls, labels, body, nav. Weights 400/500/600. `text-wrap: pretty` on prose.
- **Serif — certificate display only:** `Fraunces` (variable, optical sizing). Used *only* for the certificate title on the card/detail/verify artifact — it lends diploma gravitas. Never in chrome. `text-wrap: balance` on the title.
- **Mono — on-chain data:** `JetBrains Mono`. Every address, token id, tx hash, block number, contract address. Mono signals "exact, verifiable" and makes hashes scannable.

**Scale (clamp, display ceiling ≤ 6rem):** display/cert title `clamp(1.75rem, 4vw, 3rem)`, h1 `clamp(1.5rem, 3vw, 2.25rem)`, h2 `1.5rem`, body `1rem` (16px floor — inputs too, no iOS zoom), small `0.875rem`, mono-data `0.9375rem`. Display letter-spacing ≥ −0.02em. Body line length 65–75ch.

## Components

shadcn/ui (Radix + Tailwind v4 + CSS variables) is the primitive layer; SkillPass-specific components compose on top. **Tailwind v4 syntax only** (`@theme` tokens, `bg-linear-*`, `shadow-xs`, etc.).

- **Buttons:** `primary` (Base-blue, primary-ink), `secondary` (surface + border), `ghost`. Min height 44px on touch. Loading = inline spinner + disabled, label stays.
- **CertificateCard:** the signature object. Serif title, issuer + recipient name (sans), issued date, token id (mono). A quiet full border + subtle surface elevation — **no side-stripe, no nested cards**. Hover lifts ~2px (transform, reduced-motion: none). Grid: `repeat(auto-fill, minmax(300px, 1fr))`.
- **CertificateDetail / verify artifact:** single focused column, the cert presented like a document. Clear block of on-chain facts (contract, token id, tx, block) in mono with copy buttons + explorer links. A "Verified on-chain" affirmation (✓ + text) that links to the chain.
- **IssueForm:** labeled fields, inline validation, address field with `isAddress` check. A **mandatory privacy disclosure** (data is permanent + public on-chain) with an explicit acknowledgement checkbox before submit. Success state shows token id + tx hash + explorer link.
- **WalletConnect / NetworkGuard:** connect button shows truncated address (mono) + chain. Wrong network → a `--danger` banner (✕ + text) with a `Switch to Base Sepolia` action; writes disabled until correct.
- **States (designed, never default):** skeleton loaders matched to card/detail shape; empty state for My Certificates (illustrative, with a "what's this" line); error states with a retry; not-found for invalid token id.

## Layout

- **App shell:** quiet top nav (logo, nav links, wallet button, theme toggle). Content max-width ~`72rem`, generous padding scaling `p-4 → md:p-6 → lg:p-8`.
- **Mobile-first:** top nav collapses to a drawer + bottom tab bar on phone; touch targets ≥ 44px; tables/lists become stacked cards.
- **Verify page (`/certificates/:tokenId`):** standalone, no app chrome required, single-column, optimized to be shared and read cold.
- **z-index scale (semantic):** dropdown → sticky → modal-backdrop → modal → toast → tooltip. No arbitrary 9999.

## Motion

Intentional, restrained. Ease-out-quart/expo (no bounce/elastic). Card hover lift, list stagger on My Certificates entrance (each cert fades+rises ~8px, 40ms stagger), success-state check draw. **`@media (prefers-reduced-motion: reduce)`** → crossfade or instant on every transition. Reveals enhance already-visible content (never gate visibility on a transition class). Durations 150–250ms for UI, ≤ 400ms for entrances.

## Logo

Pending from the user. When provided: convert to **WebP + transparent background**, store under `apps/web/public/`, use in the app-shell nav and the README hero. Until then, a wordmark (`SkillPass` in Inter 600 + a small mark slot) stands in.
