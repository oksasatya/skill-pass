# SkillPass Phase 2a — Frontend Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) tracking. Every implementer brief MUST invoke `frontend-design` first and apply `DESIGN.md` tokens.

**Goal:** Stand up the SkillPass Vite SPA foundation — running app that connects a wallet, validates Base Sepolia, applies the committed design system, and serves a landing page.

**Architecture:** Vite + React + TypeScript SPA. Tailwind CSS **v4** (`@theme` tokens from `DESIGN.md`) + shadcn/ui. wagmi + viem for Web3 (read-only here; writes come in 2b). Client router. No backend.

**Tech Stack:** Vite, React 18, TypeScript (strict), Tailwind v4, shadcn/ui, wagmi, viem, TanStack Router (or React Router — implementer's call, pick one and be consistent).

## Global Constraints

- **Design contract:** `PRODUCT.md` + `DESIGN.md` at repo root are binding. Colors/type/spacing/components/motion come from `DESIGN.md` (OKLCH tokens, Inter + Fraunces + JetBrains Mono, restrained Base-blue accent, light + dark). Do not invent a palette.
- **Tailwind v4 only:** `@import "tailwindcss"`, `@theme {}` tokens, v4 utilities (`bg-linear-*`, `shadow-xs`, `rounded-xs`, `ring-3`, `shrink-*`). Any v3-era utility is a blocker.
- **TypeScript strict.** React props `readonly` (Sonar S6759). `globalThis` not `window` (S7764). No nested ternaries (S3358). Real elements over ARIA roles (S6819). `arr.at(-1)` (S7755).
- **Accessibility WCAG 2.2 AA:** keyboard-complete, visible focus, `prefers-reduced-motion` honored, status by text+icon not color alone, touch targets ≥44px, inputs ≥16px.
- **Network:** Base Sepolia, chainId `84532`. Contract ABI from Phase 1 (`contracts/out/SkillPassCertificate.sol/SkillPassCertificate.json` → export to `apps/web/src/lib/`). Contract address from `VITE_CONTRACT_ADDRESS` env (filled after a real Base Sepolia deploy; use a placeholder + `.env.example` for now).
- **Monorepo:** frontend lives at `apps/web/`. Node 20+. Use the project's package manager (npm unless a lockfile says otherwise).
- All work on branch `phase-2-frontend`.

---

### Task 1: Scaffold Vite + React + TS + Tailwind v4 + shadcn

**Files:** `apps/web/` (Vite project), `apps/web/package.json`, `apps/web/vite.config.ts`, `apps/web/tsconfig*.json`, `apps/web/index.html`, `apps/web/src/main.tsx`, `apps/web/src/App.tsx`, `apps/web/src/index.css`, `apps/web/components.json` (shadcn).

**Deliverable:** `npm run dev` serves a blank-but-styled page; `npm run build` + `tsc --noEmit` pass.

- [ ] Scaffold Vite React-TS in `apps/web/` (`npm create vite@latest apps/web -- --template react-ts`). Set `tsconfig` `strict: true`.
- [ ] Install + configure **Tailwind v4** (`tailwindcss @tailwindcss/vite`), `@import "tailwindcss"` in `index.css`, Vite plugin wired.
- [ ] `shadcn` init (`components.json`, path aliases `@/*`, CSS-variable theming). Add base primitives used in 2a: `button`.
- [ ] Add wagmi + viem + the router lib + `@tanstack/react-query` (wagmi peer).
- [ ] Verify: `npm run build` and `npx tsc --noEmit` clean; dev server renders.
- [ ] **TDD: no** (scaffold/config — verify by build + dev render).

### Task 2: Design tokens from DESIGN.md

**Files:** `apps/web/src/index.css` (`@theme` + `:root`/`.dark` OKLCH vars), font setup (self-host or Google Fonts: Inter, Fraunces, JetBrains Mono).

**Deliverable:** the DESIGN.md light + dark palettes, type roles, and radii are live as Tailwind v4 tokens; a `.dark` class swaps theme.

- [ ] Translate `DESIGN.md` OKLCH tokens (light `:root`, dark `.dark`) into `@theme` + CSS variables. Wire fonts (sans=Inter, serif=Fraunces, mono=JetBrains Mono).
- [ ] Verify contrast: body text ≥4.5:1 both themes (spot-check `--ink`/`--ink-muted` on `--bg`/`--surface`).
- [ ] **TDD: no** (visual tokens — verify by rendering a swatch/type specimen + screenshot).

### Task 3: Web3 config (chains, wagmi, contract ABI)

**Files:** `apps/web/src/lib/chains.ts`, `apps/web/src/lib/wagmi.ts`, `apps/web/src/lib/contract.ts`, `apps/web/src/lib/SkillPassCertificate.abi.json` (exported from Phase 1), `apps/web/.env.example`.

**Interfaces produced:** `config` (wagmi config), `baseSepolia` chain, `CONTRACT_ADDRESS`, `CONTRACT_ABI`, `skillPassContract` ({address, abi}).

- [ ] `chains.ts`: import `baseSepolia` from `viem/chains` (chainId 84532), http transport (RPC from `VITE_BASE_SEPOLIA_RPC` env, fallback `https://sepolia.base.org`).
- [ ] `wagmi.ts`: `createConfig` with connectors `injected()` + `coinbaseWallet({ appName: 'SkillPass' })`; SSR off.
- [ ] `contract.ts`: export ABI (copy from `contracts/out/.../SkillPassCertificate.json` `.abi`) + `CONTRACT_ADDRESS = import.meta.env.VITE_CONTRACT_ADDRESS`.
- [ ] `.env.example`: `VITE_CONTRACT_ADDRESS=`, `VITE_BASE_SEPOLIA_RPC=https://sepolia.base.org`.
- [ ] **TDD: no** (config/wiring — verify by `tsc` + the address/abi importing cleanly).

### Task 4: App shell + providers + theme toggle + nav

**Files:** `apps/web/src/main.tsx` (WagmiProvider + QueryClientProvider + router), `apps/web/src/components/layout/AppShell.tsx`, `Nav.tsx`, `ThemeToggle.tsx`, `apps/web/src/hooks/useTheme.ts`, route setup.

**Deliverable:** app shell with top nav (logo/wordmark, nav links, wallet slot, theme toggle), dark/light persisted, routes registered (`/`, `/app`, `/app/issue`, `/app/my-certificates`, `/certificates/:tokenId` — placeholders for unbuilt ones).

- [ ] Wrap app in `WagmiProvider` + `QueryClientProvider` + router. Register all 5 routes (placeholder components for 2b/2c screens).
- [ ] AppShell per `DESIGN.md` layout (quiet top nav, max-width content, responsive padding; mobile drawer/bottom-tab is acceptable as a follow-up but nav must be usable on phone). Wordmark stands in for the logo (logo pending).
- [ ] Theme toggle persists to `localStorage`, respects `prefers-color-scheme` initial.
- [ ] **TDD: split.** `useTheme` logic (resolve initial theme, persist) — **TDD: yes** (small contract). Visual shell — **TDD: no** (verify by render + screenshot).
- [ ] Invoke `frontend-design` first; apply DESIGN.md.

### Task 5: Wallet connect + NetworkGuard

**Files:** `apps/web/src/components/wallet/ConnectButton.tsx`, `NetworkGuard.tsx`, `apps/web/src/hooks/useIsCorrectNetwork.ts`, `apps/web/src/lib/format.ts` (address truncation).

**Deliverable:** connect (injected + Coinbase Wallet), show truncated address (mono) + chain, disconnect; wrong network → `--danger` banner (✕ + text) with `Switch to Base Sepolia` (`switchChain`), writes-disabled signal exposed.

- [ ] `ConnectButton`: `useConnect`/`useAccount`/`useDisconnect`; truncated address in mono; connector picker (injected, Coinbase Wallet).
- [ ] `NetworkGuard`: `useChainId` vs `84532`; off-network → danger banner + `useSwitchChain` action. Expose `isCorrectNetwork` for gating writes later.
- [ ] `format.ts`: `truncateAddress(addr)` → `0x1234…abcd`. **TDD: yes** (pure fn, clear contract — test truncation + edge cases).
- [ ] Status by text+icon, not color alone; banner keyboard-focusable; reduced-motion respected.
- [ ] Invoke `frontend-design` first; apply DESIGN.md.

### Task 6: Landing page `/`

**Files:** `apps/web/src/routes/Landing.tsx` (+ sections).

**Deliverable:** a clean, on-brand landing that states what SkillPass is, who it's for, and routes to `/app`. Honest, trustworthy, no web3 slop (per PRODUCT.md anti-references). Copy is sharp + active-voice.

- [ ] Hero (what + primary CTA → `/app`), a short "how it works" (issue → own → verify), audience note. No eyebrow-on-every-section, no identical card grid, no gradient text (DESIGN.md bans).
- [ ] Copy pass: headings/CTA/body in active voice, plain + confident (PRODUCT.md voice).
- [ ] Responsive + a11y; `prefers-reduced-motion`; LCP-friendly (no heavy hero media).
- [ ] **TDD: no** (visual/marketing — verify by render + screenshot + react-doctor at 2c finish).
- [ ] Invoke `frontend-design` first; apply DESIGN.md.

---

## Definition of Done (Phase 2a)
- `npm run dev` serves a styled app on the DESIGN.md system (light + dark).
- Wallet connects (injected + Coinbase Wallet); wrong network shows the guard + switch action.
- All 5 routes registered (placeholders for 2b/2c).
- Landing page live, on-brand, accessible.
- `tsc --noEmit` clean; `npm run build` passes.

## Notes for the executor
- ABI source: `contracts/out/SkillPassCertificate.sol/SkillPassCertificate.json` (run `forge build` in `contracts/` first; `export PATH="$HOME/.foundry/bin:$PATH"`).
- Contract address is env-driven; the app builds + connects wallet without a live deploy. Reads against the contract land in 2c and need a deployed `VITE_CONTRACT_ADDRESS` (user's Base Sepolia deploy, or a local anvil deploy for dev).
- Finishing gate (`react-doctor` + impeccable `audit`/`polish` + CWV perf check) runs at the END of 2c, over the whole frontend — not per task here.
- Commit per task on `phase-2-frontend`; no merge to `dev`/`main` until 2c finishes + final review.
