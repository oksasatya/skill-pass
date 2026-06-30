import { StrictMode, lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { WagmiProvider } from 'wagmi'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { config } from '@/lib/wagmi'
import { AppShell } from '@/components/layout/AppShell'
import { ConnectButton } from '@/components/wallet/ConnectButton'

// Self-hosted fonts — replaces Google Fonts runtime fetch (no render-blocking external request)
// Inter: variable weight for UI/body; Fraunces: variable for certificate serif; JetBrains Mono: on-chain data
import '@fontsource-variable/inter'
import '@fontsource-variable/fraunces'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'

import './index.css'

// Route-level code splitting — each page chunk is loaded only when navigated to.
// Landing + Verify are public/light; wagmi/viem weight lands in the /app/* chunks.
// ponytail: React.lazy is stdlib — no extra dep needed
const LandingPage = lazy(() => import('@/pages/LandingPage'))
const DashboardPage = lazy(() => import('@/pages/DashboardPage'))
const IssuePage = lazy(() => import('@/pages/IssuePage'))
const MyCertificatesPage = lazy(() => import('@/pages/MyCertificatesPage'))
const VerifyCertificatePage = lazy(() => import('@/pages/VerifyCertificatePage'))

// Lightweight on-brand spinner — single ring, no animation library
// ponytail: role="status" is correct here (live-region for page loading state);
// react-doctor prefer-tag-over-role suggests <output> but that's a form-calculation element — false positive.
function PageSpinner() {
  return (
    // eslint-disable-next-line jsx-a11y/prefer-tag-over-role
    <div
      className="flex min-h-svh items-center justify-center bg-bg"
      aria-label="Loading"
      role="status"
    >
      <span
        className="size-8 rounded-full border-2 border-border border-t-primary animate-spin"
        aria-hidden="true"
      />
    </div>
  )
}

// ponytail: single QueryClient instance — no custom staleTime/retries yet; tune when real queries land
const queryClient = new QueryClient()

/**
 * Route table
 * /                        → LandingPage          (standalone, no shell)
 * /app                     → AppShell > Dashboard
 * /app/issue               → AppShell > Issue
 * /app/my-certificates     → AppShell > MyCerts
 * /certificates/:tokenId   → VerifyCertificate    (standalone, no shell)
 */
const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <Suspense fallback={<PageSpinner />}>
        <LandingPage />
      </Suspense>
    ),
  },
  {
    // AppShell wraps all /app/* routes via Outlet
    element: <AppShell walletSlot={<ConnectButton />} />,
    children: [
      {
        path: '/app',
        element: (
          <Suspense fallback={<PageSpinner />}>
            <DashboardPage />
          </Suspense>
        ),
      },
      {
        path: '/app/issue',
        element: (
          <Suspense fallback={<PageSpinner />}>
            <IssuePage />
          </Suspense>
        ),
      },
      {
        path: '/app/my-certificates',
        element: (
          <Suspense fallback={<PageSpinner />}>
            <MyCertificatesPage />
          </Suspense>
        ),
      },
    ],
  },
  {
    // Standalone — no AppShell chrome per DESIGN.md
    path: '/certificates/:tokenId',
    element: (
      <Suspense fallback={<PageSpinner />}>
        <VerifyCertificatePage />
      </Suspense>
    ),
  },
])

const rootEl = document.getElementById('root')
if (!rootEl) throw new Error('Root element #root not found')

createRoot(rootEl).render(
  <StrictMode>
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </WagmiProvider>
  </StrictMode>,
)
