import { useState, useEffect } from 'react'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'skillpass-theme'

function resolveInitial(): Theme {
  // Sonar S7764: globalThis not window, with ?. for SSR-safety
  const stored = globalThis.localStorage?.getItem(STORAGE_KEY)
  if (stored === 'dark' || stored === 'light') return stored
  return globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light'
}

function applyToDOM(theme: Theme) {
  globalThis.document?.documentElement.classList.toggle('dark', theme === 'dark')
}

/** Manages light/dark theme: stored > system, persists to localStorage, syncs .dark on <html>. */
export function useTheme() {
  const [theme, setTheme] = useState<Theme>(resolveInitial)

  // Sync DOM on mount (handles SSR hydration edge; safe no-op on repeat)
  useEffect(() => {
    applyToDOM(theme)
  }, [theme])

  function toggle() {
    const next: Theme = theme === 'dark' ? 'light' : 'dark'
    globalThis.localStorage?.setItem(STORAGE_KEY, next)
    applyToDOM(next)
    setTheme(next)
  }

  return { theme, toggle } as const
}
