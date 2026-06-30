/**
 * useCertificates — hybrid read: event getLogs (by indexed recipient) → multicall getCertificate.
 *
 * Read strategy (§7.3 spec):
 *   1. getContractEvents('CertificateIssued', filter by recipient) → tokenIds  O(k) over logs
 *   2. multicall getCertificate for all tokenIds in ONE round-trip               O(k) reads
 *
 * No N+1. k = number of certs owned by this address (typically small).
 *
 * Guard: if CONTRACT_ADDRESS is undefined → return contractNotConfigured=true, no crash.
 */

import { useQuery } from '@tanstack/react-query'
import { usePublicClient, useAccount } from 'wagmi'
import { CONTRACT_ADDRESS, CONTRACT_ABI } from '@/lib/contract'
import type { Abi } from 'viem'

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
  readonly contractNotConfigured: boolean
}

// ── Pure helpers (TDD: tested in useCertificates.test.ts) ─────────────────────

/**
 * Extract unique tokenIds from raw CertificateIssued log args.
 * Args shape: { tokenId: bigint, recipient: `0x${string}`, ... }
 * ponytail: Set deduplicates in case the same event replays (shouldn't happen on-chain; defensive).
 */
export function extractTokenIds(
  logs: ReadonlyArray<{ readonly args?: Readonly<Record<string, unknown>> }>,
): bigint[] {
  const seen = new Set<bigint>()
  const ids: bigint[] = []
  for (const log of logs) {
    const raw = log.args?.['tokenId']
    if (typeof raw === 'bigint' && !seen.has(raw)) {
      seen.add(raw)
      ids.push(raw)
    }
  }
  return ids
}

/**
 * Map a multicall result tuple [cert, recipient] into a CertificateView.
 * Returns null if the result errored or has unexpected shape (skipped silently).
 */
export function mapMulticallResult(
  tokenId: bigint,
  result: Readonly<{ status: 'success' | 'failure'; result?: unknown }>,
): CertificateView | null {
  if (result.status !== 'success' || !Array.isArray(result.result)) return null
  const [cert, recipient] = result.result as [unknown, unknown]
  if (
    typeof cert !== 'object' ||
    cert === null ||
    typeof recipient !== 'string'
  ) {
    return null
  }
  const c = cert as Record<string, unknown>
  return {
    tokenId,
    title: typeof c['title'] === 'string' ? c['title'] : '',
    recipientName: typeof c['recipientName'] === 'string' ? c['recipientName'] : '',
    issuerName: typeof c['issuerName'] === 'string' ? c['issuerName'] : '',
    description: typeof c['description'] === 'string' ? c['description'] : '',
    metadataURI: typeof c['metadataURI'] === 'string' ? c['metadataURI'] : '',
    issuedAt: typeof c['issuedAt'] === 'bigint' ? c['issuedAt'] : 0n,
    recipient: recipient as `0x${string}`,
  }
}

// ── Hook ───────────────────────────────────────────────────────────────────────

/**
 * Returns certificates owned by the given address.
 * Accepts an optional address override for composability; defaults to connected wallet.
 */
export function useCertificates(ownerOverride?: `0x${string}`): UseCertificatesResult {
  const { address: connectedAddress } = useAccount()
  const publicClient = usePublicClient()

  const owner = ownerOverride ?? connectedAddress
  const contractNotConfigured = !CONTRACT_ADDRESS

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['certificates', owner, CONTRACT_ADDRESS],
    enabled: !!owner && !contractNotConfigured && !!publicClient,
    staleTime: 30_000, // ponytail: 30s stale — chain reads are slow; balance freshness vs rpc cost
    queryFn: async (): Promise<CertificateView[]> => {
      if (!owner || !publicClient || !CONTRACT_ADDRESS) return []

      // Step 1: getLogs filtered by indexed recipient — O(k) log scan
      const logs = await publicClient.getContractEvents({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        eventName: 'CertificateIssued',
        args: { recipient: owner },
        fromBlock: 0n,
      })

      const tokenIds = extractTokenIds(logs as ReadonlyArray<{ args?: Record<string, unknown> }>)
      if (tokenIds.length === 0) return []

      // Step 2: multicall getCertificate — ONE round-trip, no N+1
      // ponytail: cast ABI to viem's Abi type — JSON import loses literal narrowing
      const calls = tokenIds.map((tokenId) => ({
        address: CONTRACT_ADDRESS as `0x${string}`,
        abi: CONTRACT_ABI as Abi,
        functionName: 'getCertificate' as const,
        args: [tokenId] as const,
      }))

      const results = await publicClient.multicall({ contracts: calls })

      // Map results → CertificateView[], filter out any failures
      return results
        .map((result, i) => mapMulticallResult(tokenIds[i]!, result as { status: 'success' | 'failure'; result?: unknown }))
        .filter((v): v is CertificateView => v !== null)
    },
  })

  return {
    certificates: data ?? [],
    isLoading: !contractNotConfigured && !!owner && isLoading,
    error: error instanceof Error ? error : error ? new Error(String(error)) : null,
    refetch: () => { void refetch() },
    contractNotConfigured,
  }
}
