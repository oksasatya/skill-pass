/**
 * useCertificates — reads through the gateway BFF (paginated/searchable REST +
 * SSE live updates), never the chain directly. See docs/superpowers/plans/
 * 2026-06-30-skillpass-phase3-backend.md BE-2 Task 5.
 *
 * Guard: if VITE_GATEWAY_URL is undefined → return gatewayNotConfigured=true, no crash.
 *
 * ponytail: single-page fetch at a generous size (DEFAULT_PAGE_SIZE); add a cursor-based
 * "load more" once a user's cert count realistically exceeds this in practice.
 */

import { useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useAccount } from 'wagmi'
import { GATEWAY_URL, fetchCertificates, certificateStreamUrl, type GatewayCertificate } from '@/lib/api'

// ── Types ──────────────────────────────────────────────────────────────────────

export type CertificateView = {
  readonly tokenId: bigint
  readonly title: string
  readonly recipientName: string
  readonly issuerName: string
  readonly description: string
  readonly metadataURI: string
  readonly issuedAt: bigint
  readonly recipient: `0x${string}`
}

export type UseCertificatesResult = {
  readonly certificates: CertificateView[]
  readonly isLoading: boolean
  readonly error: Error | null
  readonly refetch: () => void
  readonly gatewayNotConfigured: boolean
}

const DEFAULT_PAGE_SIZE = 100

// ── Pure helper (TDD: tested in useCertificates.test.ts) ──────────────────────

/** Maps a gateway certificate DTO into the view shape the UI already renders. */
export function toCertificateView(g: GatewayCertificate): CertificateView {
  const issuedAtMs = Date.parse(g.issuedAt)
  return {
    tokenId: BigInt(g.tokenId),
    title: g.title,
    recipientName: g.recipientName,
    issuerName: g.issuerName,
    description: g.description,
    metadataURI: g.metadataUri,
    issuedAt: Number.isNaN(issuedAtMs) ? 0n : BigInt(Math.floor(issuedAtMs / 1000)),
    recipient: g.ownerAddress as `0x${string}`,
  }
}

// ── Hook ───────────────────────────────────────────────────────────────────────

/**
 * Returns certificates owned by the given address, with live updates via SSE.
 * Accepts an optional address override for composability; defaults to connected wallet.
 */
export function useCertificates(ownerOverride?: `0x${string}`): UseCertificatesResult {
  const { address: connectedAddress } = useAccount()
  const queryClient = useQueryClient()

  const owner = ownerOverride ?? connectedAddress
  const gatewayNotConfigured = !GATEWAY_URL
  const queryKey = ['certificates', owner, GATEWAY_URL]

  const { data, isLoading, error, refetch } = useQuery({
    queryKey,
    enabled: !!owner && !gatewayNotConfigured,
    staleTime: 30_000,
    queryFn: async (): Promise<CertificateView[]> => {
      if (!owner) return []
      const page = await fetchCertificates({ owner, pageSize: DEFAULT_PAGE_SIZE })
      return page.certificates.map(toCertificateView)
    },
  })

  // Live updates: an "issued" SSE event for this owner invalidates the query, triggering a
  // refetch through the existing cache — no manual cache surgery.
  useEffect(() => {
    if (!owner || gatewayNotConfigured) return

    const source = new EventSource(certificateStreamUrl(owner))
    source.onmessage = () => {
      void queryClient.invalidateQueries({ queryKey: ['certificates', owner, GATEWAY_URL] })
    }
    return () => source.close()
  }, [owner, gatewayNotConfigured, queryClient])

  return {
    certificates: data ?? [],
    isLoading: !gatewayNotConfigured && !!owner && isLoading,
    error: error instanceof Error ? error : error ? new Error(String(error)) : null,
    refetch: () => { void refetch() },
    gatewayNotConfigured,
  }
}
