import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { WagmiProvider } from 'wagmi'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { config } from '@/lib/wagmi'
import { AppShell } from '@/components/layout/AppShell'
import { ConnectButton } from '@/components/wallet/ConnectButton'

import LandingPage from '@/pages/LandingPage'
import DashboardPage from '@/pages/DashboardPage'
import IssuePage from '@/pages/IssuePage'
import MyCertificatesPage from '@/pages/MyCertificatesPage'
import VerifyCertificatePage from '@/pages/VerifyCertificatePage'

import './index.css'

// ponytail: single QueryClient instance — no custom staleTime/retries yet; tune when real queries land
const queryClient = new QueryClient()

/**
 * Route table
 * /                        → LandingPage          (Task 6 — placeholder now)
 * /app                     → AppShell > Dashboard (Task 6 real content)
 * /app/issue               → AppShell > Issue     (Phase 2b)
 * /app/my-certificates     → AppShell > MyCerts   (Phase 2c)
 * /certificates/:tokenId   → VerifyCertificate    (Phase 2c — standalone, no shell)
 */
const router = createBrowserRouter([
  {
    path: '/',
    element: <LandingPage />,
  },
  {
    // AppShell wraps all /app/* routes via Outlet
    element: <AppShell walletSlot={<ConnectButton />} />,
    children: [
      { path: '/app', element: <DashboardPage /> },
      { path: '/app/issue', element: <IssuePage /> },
      { path: '/app/my-certificates', element: <MyCertificatesPage /> },
    ],
  },
  {
    // Standalone — no AppShell chrome per DESIGN.md
    path: '/certificates/:tokenId',
    element: <VerifyCertificatePage />,
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
