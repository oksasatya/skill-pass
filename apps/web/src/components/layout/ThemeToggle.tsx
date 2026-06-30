import { Moon, Sun } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'
import { cn } from '@/lib/utils'

interface ThemeToggleProps {
  readonly className?: string
}

/** Keyboard-operable toggle. Persisted via useTheme. */
export function ThemeToggle({ className }: ThemeToggleProps) {
  const { theme, toggle } = useTheme()
  const isDark = theme === 'dark'

  return (
    <button
      type="button"
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      aria-pressed={isDark}
      onClick={toggle}
      className={cn(
        // S6819: real <button>, not div[role=button]
        // min 44px touch target, keyboard-operable via button default
        'inline-flex size-9 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg text-ink-muted',
        'transition-colors duration-150 motion-reduce:transition-none',
        'hover:bg-surface-2 hover:text-ink',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
        className,
      )}
    >
      {isDark
        ? <Sun className="size-4" aria-hidden="true" />
        : <Moon className="size-4" aria-hidden="true" />}
    </button>
  )
}
