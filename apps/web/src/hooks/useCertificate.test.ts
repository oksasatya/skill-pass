/**
 * TDD: pure helpers in useCertificate.ts
 * Covers parseTokenIdParam, txExplorerUrl, nftExplorerUrl, contractExplorerUrl.
 * Hook: TDD: no (verify by build + render; live data needs a deployed contract).
 */

import { describe, it, expect } from 'vitest'
import {
  parseTokenIdParam,
  txExplorerUrl,
  nftExplorerUrl,
  contractExplorerUrl,
} from './useCertificate'

// ── parseTokenIdParam ──────────────────────────────────────────────────────────

describe('parseTokenIdParam', () => {
  it('returns null for undefined', () => {
    expect(parseTokenIdParam(undefined)).toBeNull()
  })

  it('returns null for empty string', () => {
    expect(parseTokenIdParam('')).toBeNull()
  })

  it('returns null for whitespace-only string', () => {
    expect(parseTokenIdParam('   ')).toBeNull()
  })

  it('returns null for non-numeric string', () => {
    expect(parseTokenIdParam('abc')).toBeNull()
  })

  it('returns null for negative number', () => {
    expect(parseTokenIdParam('-1')).toBeNull()
  })

  it('returns null for zero (token IDs start at 1)', () => {
    expect(parseTokenIdParam('0')).toBeNull()
  })

  it('returns null for decimal string', () => {
    expect(parseTokenIdParam('1.5')).toBeNull()
  })

  it('returns null for hex string', () => {
    expect(parseTokenIdParam('0x1')).toBeNull()
  })

  it('parses "1" correctly', () => {
    expect(parseTokenIdParam('1')).toBe(1n)
  })

  it('parses large token id correctly', () => {
    expect(parseTokenIdParam('999999')).toBe(999999n)
  })

  it('parses "42" correctly', () => {
    expect(parseTokenIdParam('42')).toBe(42n)
  })
})

// ── Explorer URL builders ──────────────────────────────────────────────────────

describe('txExplorerUrl', () => {
  it('builds a valid Basescan tx URL', () => {
    const hash = '0xabc123' as `0x${string}`
    expect(txExplorerUrl(hash)).toBe('https://sepolia.basescan.org/tx/0xabc123')
  })
})

describe('nftExplorerUrl', () => {
  it('builds a valid Basescan NFT URL', () => {
    const contract = '0xdeadbeef' as `0x${string}`
    expect(nftExplorerUrl(contract, 7n)).toBe(
      'https://sepolia.basescan.org/nft/0xdeadbeef/7',
    )
  })
})

describe('contractExplorerUrl', () => {
  it('builds a valid Basescan address URL', () => {
    const addr = '0x1234' as `0x${string}`
    expect(contractExplorerUrl(addr)).toBe('https://sepolia.basescan.org/address/0x1234')
  })
})
