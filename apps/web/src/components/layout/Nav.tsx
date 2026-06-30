import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { Menu, X } from 'lucide-react'
import { ThemeToggle } from './ThemeToggle'
import { cn } from '@/lib/utils'

interface NavProps {
  /** Wallet connect button slot — Task 5 fills this */
  readonly walletSlot?: React.ReactNode
}

const NAV_LINKS = [
  { to: '/app', label: 'Dashboard' },
  { to: '/app/issue', label: 'Issue' },
  { to: '/app/my-certificates', label: 'My Certificates' },
] as const

function NavItem({ to, label }: { readonly to: string; readonly label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        cn(
          'text-sm font-medium transition-colors duration-150',
          'px-3 py-2 rounded-lg',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
          isActive
            ? 'text-primary bg-primary-weak'
            : 'text-ink-muted hover:text-ink hover:bg-surface-2',
        )
      }
    >
      {label}
    </NavLink>
  )
}

/**
 * Top navigation bar per DESIGN.md.
 * Mobile: hamburger collapses links into a dropdown panel.
 * Full drawer + bottom tab bar: ponytail: deferred to 2c polish — the current
 * collapse is functional and keyboard-operable, clearing the phone usability bar.
 */
export function Nav({ walletSlot }: NavProps) {
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    // S6819: <header> not div[role=banner]
    <header className="sticky top-0 z-10 border-b border-border bg-bg/90 backdrop-blur-sm">
      {/* ponytail: z-10 = "sticky" slot in semantic scale (DESIGN.md: dropdown→sticky→modal-backdrop→modal→toast→tooltip) */}
      <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8">
        <div className="flex h-14 items-center justify-between gap-4">

          {/* Logo mark + wordmark */}
          <NavLink
            to="/"
            className="flex shrink-0 items-center gap-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-lg px-1"
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
          </NavLink>

          {/* Desktop nav — hidden on mobile */}
          <nav
            aria-label="Main navigation"
            className="hidden md:flex items-center gap-1"
          >
            {NAV_LINKS.map((link) => (
              <NavItem key={link.to} to={link.to} label={link.label} />
            ))}
          </nav>

          {/* Right cluster: wallet slot (Task 5) + theme toggle + mobile menu button */}
          <div className="flex items-center gap-2">
            {/* Wallet slot — Task 5 fills this; empty div preserves layout until then */}
            {walletSlot ?? null}

            <ThemeToggle />

            {/* Mobile hamburger — visible only on small screens */}
            <button
              type="button"
              aria-label={mobileOpen ? 'Close navigation menu' : 'Open navigation menu'}
              aria-expanded={mobileOpen}
              aria-controls="mobile-nav-panel"
              onClick={() => setMobileOpen((v) => !v)}
              className={cn(
                'md:hidden inline-flex size-9 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg',
                'text-ink-muted hover:text-ink hover:bg-surface-2',
                'transition-colors duration-150',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              )}
            >
              {mobileOpen
                ? <X className="size-4" aria-hidden="true" />
                : <Menu className="size-4" aria-hidden="true" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile nav panel */}
      {mobileOpen && (
        <nav
          id="mobile-nav-panel"
          aria-label="Mobile navigation"
          className="md:hidden border-t border-border bg-surface px-4 py-3 flex flex-col gap-1"
        >
          {NAV_LINKS.map((link) => (
            <NavItem
              key={link.to}
              to={link.to}
              label={link.label}
            />
          ))}
        </nav>
      )}
    </header>
  )
}
