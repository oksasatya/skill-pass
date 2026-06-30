import { Outlet } from 'react-router-dom'
import { Nav } from './Nav'

interface AppShellProps {
  /** Wallet connect button forwarded into Nav's slot — Task 5 fills this */
  readonly walletSlot?: React.ReactNode
}

/**
 * App shell: sticky nav + max-width content area with responsive padding.
 * Responsive padding: p-4 → md:p-6 → lg:p-8 per DESIGN.md.
 */
export function AppShell({ walletSlot }: AppShellProps) {
  return (
    <div className="min-h-svh bg-bg text-ink">
      <Nav walletSlot={walletSlot} />
      {/* pb-20 on mobile = bottom tab bar clearance (~56px bar + 4px gap) */}
      <main className="mx-auto max-w-[72rem] px-4 py-6 pb-24 md:px-6 md:py-8 md:pb-8 lg:px-8 lg:py-10">
        <Outlet />
      </main>
    </div>
  )
}
