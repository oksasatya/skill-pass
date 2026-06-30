/**
 * TDD: truncateAddress — failing tests written before implementation.
 *
 * Contract:
 *   1. Normal 42-char address → 0x{first4}…{last4} = "0x" + 4 chars + "…" + 4 chars
 *   2. Checksummed input handled identically (lowercased internally is fine — we preserve input)
 *   3. Very short input (< 10 chars after "0x") — return as-is, do not crash
 */

import { describe, it, expect } from 'vitest'
import { truncateAddress } from './format'

describe('truncateAddress', () => {
  it('truncates a normal 42-char address to 0x1234…abcd form', () => {
    const addr = '0x1234567890abcdef1234567890abcdef12345678' as `0x${string}`
    // first 6 chars = "0x1234", last 4 = "5678"
    expect(truncateAddress(addr)).toBe('0x1234…5678')
  })

  it('handles a checksummed address', () => {
    const addr = '0xAbCdEf1234567890AbCdEf1234567890AbCdEf12' as `0x${string}`
    // "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12" = 42 chars
    // slice(0,6) = "0xAbCd", slice(-4) = "Ef12"
    expect(truncateAddress(addr)).toBe('0xAbCd…Ef12')
  })

  it('handles a short address (< 10 chars total) by returning it unchanged', () => {
    const short = '0x1234' as `0x${string}`
    expect(truncateAddress(short)).toBe('0x1234')
  })

  it('produces exactly first 6 chars + ellipsis + last 4 chars for a 42-char address', () => {
    const addr = '0xaaaa1111bbbb2222cccc3333dddd4444eeee5555' as `0x${string}`
    const result = truncateAddress(addr)
    expect(result).toBe('0xaaaa…5555')
  })
})
