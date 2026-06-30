import { useState, useEffect, useRef } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Menu, X, LayoutDashboard, PlusCircle, Award } from 'lucide-react'
import { ThemeToggle } from './ThemeToggle'
import { cn } from '@/lib/utils'

interface NavProps {
  /** Wallet connect button slot */
  readonly walletSlot?: React.ReactNode
}

const NAV_LINKS = [
  { to: '/app', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/app/issue', label: 'Issue', icon: PlusCircle, end: false },
  { to: '/app/my-certificates', label: 'My Certificates', icon: Award, end: false },
] as const

function NavItem({
  to,
  label,
  end,
  onClick,
}: {
  readonly to: string
  readonly label: string
  readonly end?: boolean
  readonly onClick?: () => void
}) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onClick}
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
 * Top navigation bar.
 * Desktop: logo + inline links + wallet + theme toggle.
 * Mobile (< md): logo + wallet + theme toggle + hamburger trigger.
 *   Drawer: slides in from the left with a scrim. Full-height, 80vw wide.
 *   Bottom tab bar: fixed 4-icon bar covering the three nav routes.
 */
export function Nav({ walletSlot }: NavProps) {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const location = useLocation()
  const drawerRef = useRef<HTMLDivElement>(null)

  // Close drawer on route change
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  // Trap focus + close on Escape
  useEffect(() => {
    if (!drawerOpen) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setDrawerOpen(false)
    }
    document.addEventListener('keydown', handleKey)
    // Prevent body scroll while drawer is open
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKey)
      document.body.style.overflow = ''
    }
  }, [drawerOpen])

  return (
    <>
      {/* ── Main top bar ────────────────────────────────────────────────── */}
      {/* ponytail: z-10 = "sticky" slot per DESIGN.md z-index scale */}
      <header className="sticky top-0 z-10 border-b border-border bg-bg/90 backdrop-blur-sm">
        <div className="mx-auto max-w-[72rem] px-4 md:px-6 lg:px-8">
          <div className="flex h-14 items-center justify-between gap-4">

            {/* Logo + wordmark */}
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
                <NavItem key={link.to} to={link.to} label={link.label} end={link.end} />
              ))}
            </nav>

            {/* Right cluster: wallet + theme + hamburger (mobile only) */}
            <div className="flex items-center gap-2">
              {walletSlot ?? null}
              <ThemeToggle />

              {/* Hamburger — mobile only */}
              <button
                type="button"
                aria-label={drawerOpen ? 'Close navigation menu' : 'Open navigation menu'}
                aria-expanded={drawerOpen}
                aria-controls="mobile-drawer"
                onClick={() => setDrawerOpen((v) => !v)}
                className={cn(
                  'md:hidden inline-flex size-9 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg',
                  'text-ink-muted hover:text-ink hover:bg-surface-2',
                  'transition-colors duration-150',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                )}
              >
                {drawerOpen
                  ? <X className="size-4" aria-hidden="true" />
                  : <Menu className="size-4" aria-hidden="true" />}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* ── Mobile drawer ───────────────────────────────────────────────── */}
      {/* Scrim */}
      <div
        aria-hidden="true"
        onClick={() => setDrawerOpen(false)}
        className={cn(
          'md:hidden fixed inset-0 bg-bg/60 backdrop-blur-sm transition-opacity duration-200',
          // ponytail: z-20 = modal-backdrop slot per DESIGN.md z-index scale
          'z-20',
          drawerOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none',
        )}
      />

      {/* Drawer panel */}
      <div
        id="mobile-drawer"
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
        className={cn(
          'md:hidden fixed inset-y-0 left-0 z-30 w-[80vw] max-w-xs',
          'bg-surface border-r border-border',
          'flex flex-col',
          'transition-transform duration-200 ease-out',
          // ponytail: CSS transform drawer — no animation library needed
          drawerOpen ? 'translate-x-0' : '-translate-x-full',
          // reduced-motion: skip the slide animation entirely
          'motion-reduce:transition-none',
        )}
      >
        {/* Drawer header */}
        <div className="flex h-14 items-center justify-between px-4 border-b border-border">
          <span className="font-sans text-base font-semibold text-ink">Menu</span>
          <button
            type="button"
            aria-label="Close navigation menu"
            onClick={() => setDrawerOpen(false)}
            className={cn(
              'inline-flex size-9 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg',
              'text-ink-muted hover:text-ink hover:bg-surface-2',
              'transition-colors duration-150',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            )}
          >
            <X className="size-4" aria-hidden="true" />
          </button>
        </div>

        {/* Drawer nav links */}
        <nav aria-label="Mobile navigation" className="flex flex-col gap-1 p-3">
          {NAV_LINKS.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              onClick={() => setDrawerOpen(false)}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-lg px-3 py-3 min-h-[48px]',
                  'text-sm font-medium transition-colors duration-150',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                  isActive
                    ? 'bg-primary-weak text-primary'
                    : 'text-ink-muted hover:text-ink hover:bg-surface-2',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <link.icon
                    className={cn('size-4 shrink-0', isActive ? 'text-primary' : 'text-ink-muted')}
                    aria-hidden="true"
                  />
                  {link.label}
                </>
              )}
            </NavLink>
          ))}
        </nav>
      </div>

      {/* ── Bottom tab bar — mobile only ───────────────────────────────── */}
      {/* Fixed bar at bottom; gives thumb-zone access to the 3 core routes */}
      {/* ponytail: z-10 = sticky slot; below drawer z-30 but above page content */}
      <nav
        aria-label="Bottom navigation"
        className={cn(
          'md:hidden fixed bottom-0 inset-x-0 z-10',
          'border-t border-border bg-bg/95 backdrop-blur-sm',
          'pb-[env(safe-area-inset-bottom)]', // notch / home-bar clearance
        )}
      >
        <div className="flex items-stretch">
          {NAV_LINKS.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) =>
                cn(
                  'flex flex-1 flex-col items-center justify-center gap-1',
                  'min-h-[56px] py-2 px-1',
                  'text-[0.625rem] font-medium transition-colors duration-150',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
                  isActive ? 'text-primary' : 'text-ink-muted',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <link.icon
                    className={cn('size-5 shrink-0', isActive ? 'text-primary' : 'text-ink-muted')}
                    aria-hidden="true"
                  />
                  <span>{link.label}</span>
                </>
              )}
            </NavLink>
          ))}
        </div>
      </nav>
    </>
  )
}
