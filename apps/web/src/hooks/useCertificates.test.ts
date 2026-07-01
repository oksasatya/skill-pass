/**
 * TDD: pure helper in useCertificates.ts — the gateway DTO -> view mapping.
 * Hook + page: TDD: no (verify by build + render per §16 verdict).
 */

import { describe, it, expect } from 'vitest'
import { toCertificateView } from './useCertificates'
import type { GatewayCertificate } from '@/lib/api'

const baseDto: GatewayCertificate = {
  tokenId: '42',
  ownerAddress: '0xabcdef0123456789abcdef0123456789abcdef01',
  title: 'Go Bootcamp',
  recipientName: 'Alice',
  issuerName: 'Hacktiv8',
  description: 'Completed Go backend course.',
  metadataUri: 'ipfs://abc',
  issuedAt: '2023-11-14T22:13:20Z',
  txHash: '0xdeadbeef',
  blockNumber: 100,
}

describe('toCertificateView', () => {
  it('maps every field, converting tokenId to bigint and issuedAt to unix seconds', () => {
    const view = toCertificateView(baseDto)
    expect(view.tokenId).toBe(42n)
    expect(view.title).toBe('Go Bootcamp')
    expect(view.recipientName).toBe('Alice')
    expect(view.issuerName).toBe('Hacktiv8')
    expect(view.description).toBe('Completed Go backend course.')
    expect(view.metadataURI).toBe('ipfs://abc')
    expect(view.issuedAt).toBe(1700000000n)
    expect(view.recipient).toBe('0xabcdef0123456789abcdef0123456789abcdef01')
  })

  it('falls back to 0n issuedAt when the date is unparsable', () => {
    const view = toCertificateView({ ...baseDto, issuedAt: 'not-a-date' })
    expect(view.issuedAt).toBe(0n)
  })
})
