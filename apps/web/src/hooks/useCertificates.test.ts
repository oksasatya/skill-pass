/**
 * TDD: pure helpers in useCertificates.ts
 * Covers extractTokenIds and mapMulticallResult — the data-shaping paths.
 * Hooks + pages: TDD: no (verify by build + render per spec §10 / §16 verdict).
 */

import { describe, it, expect } from 'vitest'
import { extractTokenIds, mapMulticallResult } from './useCertificates'

// ── extractTokenIds ────────────────────────────────────────────────────────────

describe('extractTokenIds', () => {
  it('returns empty array for no logs', () => {
    expect(extractTokenIds([])).toEqual([])
  })

  it('extracts bigint tokenIds from log args', () => {
    const logs = [
      { args: { tokenId: 1n, recipient: '0xabc' } },
      { args: { tokenId: 2n, recipient: '0xabc' } },
    ]
    expect(extractTokenIds(logs)).toEqual([1n, 2n])
  })

  it('deduplicates repeated tokenIds', () => {
    const logs = [
      { args: { tokenId: 5n } },
      { args: { tokenId: 5n } },
      { args: { tokenId: 7n } },
    ]
    expect(extractTokenIds(logs)).toEqual([5n, 7n])
  })

  it('skips logs without args', () => {
    const logs = [
      {},
      { args: { tokenId: 3n } },
    ]
    expect(extractTokenIds(logs)).toEqual([3n])
  })

  it('skips non-bigint tokenId values', () => {
    const logs = [
      { args: { tokenId: '1' } },   // string — skip
      { args: { tokenId: 42n } },   // bigint — keep
    ]
    expect(extractTokenIds(logs)).toEqual([42n])
  })
})

// ── mapMulticallResult ─────────────────────────────────────────────────────────

describe('mapMulticallResult', () => {
  const validCert = {
    title: 'Go Bootcamp',
    recipientName: 'Alice',
    issuerName: 'Hacktiv8',
    description: 'Completed Go backend course.',
    metadataURI: 'ipfs://abc',
    issuedAt: 1700000000n,
  }

  it('maps a successful multicall result to CertificateView', () => {
    const result = {
      status: 'success' as const,
      result: [validCert, '0xRecipient'],
    }
    const view = mapMulticallResult(1n, result)
    expect(view).toEqual({
      tokenId: 1n,
      title: 'Go Bootcamp',
      recipientName: 'Alice',
      issuerName: 'Hacktiv8',
      description: 'Completed Go backend course.',
      metadataURI: 'ipfs://abc',
      issuedAt: 1700000000n,
      recipient: '0xRecipient',
    })
  })

  it('returns null on failure status', () => {
    const result = { status: 'failure' as const }
    expect(mapMulticallResult(1n, result)).toBeNull()
  })

  it('returns null when result is not an array', () => {
    const result = { status: 'success' as const, result: null }
    expect(mapMulticallResult(1n, result)).toBeNull()
  })

  it('returns null when cert is not an object', () => {
    const result = { status: 'success' as const, result: ['not-a-cert', '0xAddr'] }
    expect(mapMulticallResult(1n, result)).toBeNull()
  })

  it('uses empty string fallback for missing cert string fields', () => {
    const partialCert = { title: 'Only title' } // missing other fields
    const result = { status: 'success' as const, result: [partialCert, '0xAddr'] }
    const view = mapMulticallResult(99n, result)
    expect(view?.recipientName).toBe('')
    expect(view?.issuerName).toBe('')
    expect(view?.issuedAt).toBe(0n)
  })
})
