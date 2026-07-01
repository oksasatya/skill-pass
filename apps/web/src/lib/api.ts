/**
 * Gateway REST client — reads for My Certificates go through the gateway BFF
 * (paginated/searchable), never the chain directly. The public verify page is the
 * one exception (see useCertificate.ts) — it reads the chain directly because that's
 * the authoritative, walletless-verifiable source.
 */

export const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL as string | undefined

export type GatewayCertificate = {
  readonly tokenId: string
  readonly ownerAddress: string
  readonly title: string
  readonly recipientName: string
  readonly issuerName: string
  readonly description: string
  readonly metadataUri: string
  readonly issuedAt: string // RFC3339
  readonly txHash: string
  readonly blockNumber: number
}

export type ListCertificatesParams = {
  readonly owner?: string
  readonly query?: string
  readonly cursor?: string
  readonly pageSize?: number
}

export type ListCertificatesResult = {
  readonly certificates: readonly GatewayCertificate[]
  readonly nextCursor: string
  readonly hasMore: boolean
}

/** Fetches one page of certificates from the gateway. Throws on missing config, network, or HTTP error. */
export async function fetchCertificates(params: ListCertificatesParams): Promise<ListCertificatesResult> {
  if (!GATEWAY_URL) throw new Error('gateway not configured')

  const qs = new URLSearchParams()
  if (params.owner) qs.set('owner', params.owner)
  if (params.query) qs.set('q', params.query)
  if (params.cursor) qs.set('cursor', params.cursor)
  if (params.pageSize) qs.set('page_size', String(params.pageSize))

  const res = await fetch(`${GATEWAY_URL}/certificates?${qs.toString()}`)
  if (!res.ok) throw new Error(`gateway list failed: ${res.status}`)
  return res.json() as Promise<ListCertificatesResult>
}

/** Builds the SSE stream URL for live certificate-issued updates, optionally owner-scoped. */
export function certificateStreamUrl(owner?: string): string {
  const qs = owner ? `?owner=${encodeURIComponent(owner)}` : ''
  return `${GATEWAY_URL}/certificates/stream${qs}`
}
