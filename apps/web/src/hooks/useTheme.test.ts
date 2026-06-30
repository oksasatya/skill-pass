/**
 * TDD: useTheme — failing tests written before implementation.
 *
 * Contract under test:
 *   1. Initial theme = stored localStorage value if present
 *   2. Initial theme = 'dark' when no stored value AND prefers-color-scheme: dark
 *   3. Initial theme = 'light' when no stored value AND no dark preference
 *   4. toggle() flips the theme
 *   5. toggle() persists the new value to localStorage
 *   6. toggle() adds/removes the .dark class on documentElement
 *   7. Uses globalThis (SSR-safe) — not bare window
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useTheme } from './useTheme'

// ── jsdom does not implement matchMedia — provide a configurable stub ──────
function mockMatchMedia(prefersDark: boolean) {
  Object.defineProperty(globalThis, 'matchMedia', {
    writable: true,
    value: vi.fn((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' && prefersDark,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

function setStorage(value: string | null) {
  if (value === null) {
    globalThis.localStorage.removeItem('skillpass-theme')
  } else {
    globalThis.localStorage.setItem('skillpass-theme', value)
  }
}

beforeEach(() => {
  // Reset DOM and storage between tests
  globalThis.document.documentElement.className = ''
  globalThis.localStorage.clear()
  mockMatchMedia(false)
})

describe('useTheme — initial theme resolution', () => {
  it('uses stored "dark" value from localStorage', () => {
    setStorage('dark')
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('dark')
  })

  it('uses stored "light" value from localStorage', () => {
    setStorage('light')
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('light')
  })

  it('falls back to dark when no storage and prefers-color-scheme: dark', () => {
    setStorage(null)
    mockMatchMedia(true)
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('dark')
  })

  it('falls back to light when no storage and no dark preference', () => {
    setStorage(null)
    mockMatchMedia(false)
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('light')
  })
})

describe('useTheme — DOM class sync on init', () => {
  it('adds .dark class when initial theme is dark', () => {
    setStorage('dark')
    renderHook(() => useTheme())
    expect(globalThis.document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('does NOT add .dark class when initial theme is light', () => {
    setStorage('light')
    renderHook(() => useTheme())
    expect(globalThis.document.documentElement.classList.contains('dark')).toBe(false)
  })
})

describe('useTheme — toggle()', () => {
  it('flips light → dark', () => {
    setStorage('light')
    const { result } = renderHook(() => useTheme())
    act(() => { result.current.toggle() })
    expect(result.current.theme).toBe('dark')
  })

  it('flips dark → light', () => {
    setStorage('dark')
    const { result } = renderHook(() => useTheme())
    act(() => { result.current.toggle() })
    expect(result.current.theme).toBe('light')
  })

  it('persists the new theme to localStorage after toggle', () => {
    setStorage('light')
    const { result } = renderHook(() => useTheme())
    act(() => { result.current.toggle() })
    expect(globalThis.localStorage.getItem('skillpass-theme')).toBe('dark')
  })

  it('adds .dark class on documentElement when toggling to dark', () => {
    setStorage('light')
    const { result } = renderHook(() => useTheme())
    act(() => { result.current.toggle() })
    expect(globalThis.document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('removes .dark class on documentElement when toggling to light', () => {
    setStorage('dark')
    const { result } = renderHook(() => useTheme())
    act(() => { result.current.toggle() })
    expect(globalThis.document.documentElement.classList.contains('dark')).toBe(false)
  })
})
