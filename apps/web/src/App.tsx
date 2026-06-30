// ponytail: throwaway specimen — replaced in Task 4 (shell + components)
// Verifies: token rendering, font loading, theme switching, contrast, button brand color

import { useState } from 'react'
import { Button } from '@/components/ui/button'

const swatches = [
  { token: '--bg', label: 'bg', cls: 'bg-bg', textCls: 'text-ink' },
  { token: '--surface', label: 'surface', cls: 'bg-surface', textCls: 'text-ink' },
  { token: '--surface-2', label: 'surface-2', cls: 'bg-surface-2', textCls: 'text-ink' },
  { token: '--primary', label: 'primary', cls: 'bg-primary', textCls: 'text-primary-ink' },
  { token: '--primary-weak', label: 'primary-weak', cls: 'bg-primary-weak', textCls: 'text-primary' },
  { token: '--success', label: 'success', cls: 'bg-success', textCls: 'text-white' },
  { token: '--danger', label: 'danger', cls: 'bg-danger', textCls: 'text-white' },
  { token: '--warning', label: 'warning', cls: 'bg-warning', textCls: 'text-ink' },
] as const

export default function App() {
  const [dark, setDark] = useState(false)

  const toggle = () => {
    const html = globalThis.document?.documentElement
    if (!html) return
    html.classList.toggle('dark', !dark)
    setDark(!dark)
  }

  return (
    <main className="min-h-svh bg-bg text-ink p-8 space-y-10">
      {/* Theme toggle */}
      <div className="flex items-center justify-between">
        <span className="font-sans text-sm text-ink-muted">Token specimen · Task 2</span>
        <Button variant="outline" size="sm" onClick={toggle}>
          {dark ? 'Light mode' : 'Dark mode'}
        </Button>
      </div>

      {/* Color swatches */}
      <section>
        <h2 className="font-sans text-sm font-medium text-ink-muted mb-3 uppercase tracking-widest">
          Color tokens
        </h2>
        <div className="flex flex-wrap gap-2">
          {swatches.map(({ label, cls, textCls }) => (
            <div
              key={label}
              className={`${cls} ${textCls} rounded-lg border border-border px-4 py-3 text-xs font-mono min-w-[7rem]`}
            >
              {label}
            </div>
          ))}
        </div>
      </section>

      {/* Type scale */}
      <section>
        <h2 className="font-sans text-sm font-medium text-ink-muted mb-3 uppercase tracking-widest">
          Type scale
        </h2>
        <div className="space-y-3 border border-border rounded-xl p-6 bg-surface">
          <p className="font-serif text-[clamp(1.75rem,4vw,3rem)] leading-tight tracking-[-0.02em] text-ink">
            Certificate of Completion
          </p>
          <p className="font-sans text-[clamp(1.5rem,3vw,2.25rem)] font-semibold text-ink">
            Heading 1 — Inter 600
          </p>
          <p className="font-sans text-2xl font-medium text-ink">
            Heading 2 — Inter 500
          </p>
          <p className="font-sans text-base text-ink leading-relaxed max-w-[65ch]">
            Body copy — Inter 400. SkillPass issues verifiable on-chain credentials.
            Each certificate is a token on Base; the chain is the source of truth.
          </p>
          <p className="font-sans text-sm text-ink-muted">
            Small / secondary — ink-muted (secondary labels only)
          </p>
          <p className="font-mono text-[0.9375rem] text-ink-muted tracking-tight">
            0x1a2b3c4d5e6f7890abcdef1234567890abcdef12 — JetBrains Mono
          </p>
        </div>
      </section>

      {/* Buttons */}
      <section>
        <h2 className="font-sans text-sm font-medium text-ink-muted mb-3 uppercase tracking-widest">
          Buttons
        </h2>
        <div className="flex flex-wrap gap-3">
          <Button>Primary (Base-blue)</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
        </div>
      </section>

      {/* Border + radius reference */}
      <section>
        <h2 className="font-sans text-sm font-medium text-ink-muted mb-3 uppercase tracking-widest">
          Surfaces
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-surface border border-border rounded-lg p-4 text-sm text-ink">
            surface · radius-lg
          </div>
          <div className="bg-surface-2 border border-border rounded-xl p-4 text-sm text-ink">
            surface-2 · radius-xl
          </div>
          <div className="bg-primary-weak border border-primary/20 rounded-xl p-4 text-sm text-primary">
            primary-weak · selected state
          </div>
        </div>
      </section>
    </main>
  )
}
