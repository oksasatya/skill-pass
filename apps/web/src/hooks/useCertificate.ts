/**
 * useCertificate — walletless single-cert read by tokenId.
 *
 * Read strategy (§7.3 spec): chain direct via publicClient.readContract.
 * Works with NO wallet connected — uses a standalone publicClient from wagmi's
 * usePublicClient (chain=baseSepolia is always available in the wagmi config).
 *
 * Optional tx proof: fetches the CertificateIssued event log for txHash + blockNumber.
 * If that lookup fails, the page still renders with cert data — degraded gracefully.
 *
 * Invalid tokenId (non-existent on chain): getCertificate reverts with
 * ERC721NonexistentToken → notFound=true.
 *
 * Guard: CONTRACT_ADDRESS undefined → contractNotConfigured=true, no crash.
 *
 * TDD verdict: pure helpers (explorer URLs, param validation) → TDD: yes.
 *             Hook + page → TDD: no (verify by build + render; live data = deployed contract).
 */

import { useQuery } from '@tanstack/react-query'
import { usePublicClient } from 'wagmi'
import { CONTRACT_ADDRESS, CONTRACT_ABI } from '@/lib/contract'
import type { Abi } from 'viem'

// ── Types ──────────────────────────────────────────────────────────────────────

export type CertificateOnChain = {
  readonly tokenId: bigint
  readonly title: string
  readonly recipientName: string
  readonly issuerName: string
  readonly description: string
  readonly metadataURI: string
  readonly issuedAt: bigint
  readonly recipient: `0x${string}`
  /** From CertificateIssued event log — optional (lookup may fail) */
  readonly txHash?: `0x${string}`
  readonly blockNumber?: bigint
}

export type UseCertificateResult = {
  readonly certificate: CertificateOnChain | null
  readonly isLoading: boolean
  readonly notFound: boolean
  readonly error: Error | null
  readonly refetch: () => void
  readonly contractNotConfigured: boolean
}

// ── Pure helpers (TDD: tested in useCertificate.test.ts) ──────────────────────

/**
 * Validate and parse a tokenId route param string.
 * Returns BigInt on success, null if invalid (non-numeric / negative / empty).
 */
export function parseTokenIdParam(raw: string | undefined): bigint | null {
  if (!raw || raw.trim() === '') return null
  // Must be all digits (no decimals, no negative sign)
  if (!/^\d+$/.test(raw)) return null
  try {
    const n = BigInt(raw)
    if (n < 1n) return null  // token IDs start at 1 per contract spec
    return n
  } catch {
    return null
  }
}

/** Build a Basescan URL for a transaction hash */
export function txExplorerUrl(hash: `0x${string}`): string {
  return `https://sepolia.basescan.org/tx/${hash}`
}

/** Build a Basescan URL for an NFT (token page) */
export function nftExplorerUrl(contract: `0x${string}`, tokenId: bigint): string {
  return `https://sepolia.basescan.org/nft/${contract}/${tokenId.toString()}`
}

/** Build a Basescan URL for a contract address */
export function contractExplorerUrl(contract: `0x${string}`): string {
  return `https://sepolia.basescan.org/address/${contract}`
}

// ── Revert detection ──────────────────────────────────────────────────────────

/**
 * Returns true if the error looks like a contract revert for a non-existent token.
 * Covers ERC721NonexistentToken and generic revert messages.
 */
function isNotFoundError(err: unknown): boolean {
  if (!(err instanceof Error)) return false
  const msg = err.message.toLowerCase()
  return (
    msg.includes('erc721nonexistenttoken') ||
    msg.includes('nonexistent') ||
    msg.includes('token does not exist') ||
    msg.includes('invalid token id') ||
    msg.includes('reverted')  // catch-all for contract reverts on bad tokenId
  )
}

// ── Internal result type for queryFn ──────────────────────────────────────────

type FetchResult = {
  readonly cert: CertificateOnChain
}

// ── Hook ───────────────────────────────────────────────────────────────────────

export function useCertificate(tokenId: bigint | null): UseCertificateResult {
  const publicClient = usePublicClient()
  const contractNotConfigured = !CONTRACT_ADDRESS

  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery<FetchResult, Error>({
    queryKey: ['certificate', tokenId?.toString(), CONTRACT_ADDRESS],
    enabled: tokenId !== null && !contractNotConfigured && !!publicClient,
    staleTime: 60_000, // ponytail: 60s — cert data is immutable once issued
    retry: (failureCount, err) => {
      // Don't retry on not-found — it's deterministic
      if (isNotFoundError(err)) return false
      return failureCount < 2
    },
    queryFn: async (): Promise<FetchResult> => {
      if (!tokenId || !publicClient || !CONTRACT_ADDRESS) {
        throw new Error('Missing required params')
      }

      // Primary read: getCertificate(tokenId) → [Certificate, address]
      // Reverts with ERC721NonexistentToken if tokenId doesn't exist
      const raw = await publicClient.readContract({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI as Abi,
        functionName: 'getCertificate',
        args: [tokenId],
      })

      // viem returns the tuple as [cert_struct, recipient_address]
      if (!Array.isArray(raw) || raw.length < 2) {
        throw new Error('Unexpected getCertificate response shape')
      }
      const [certStruct, recipientAddr] = raw as [unknown, unknown]

      if (typeof certStruct !== 'object' || certStruct === null || typeof recipientAddr !== 'string') {
        throw new Error('getCertificate returned unexpected types')
      }

      const c = certStruct as Record<string, unknown>
      const cert: Omit<CertificateOnChain, 'txHash' | 'blockNumber'> = {
        tokenId,
        title: typeof c['title'] === 'string' ? c['title'] : '',
        recipientName: typeof c['recipientName'] === 'string' ? c['recipientName'] : '',
        issuerName: typeof c['issuerName'] === 'string' ? c['issuerName'] : '',
        description: typeof c['description'] === 'string' ? c['description'] : '',
        metadataURI: typeof c['metadataURI'] === 'string' ? c['metadataURI'] : '',
        issuedAt: typeof c['issuedAt'] === 'bigint' ? c['issuedAt'] : 0n,
        recipient: recipientAddr as `0x${string}`,
      }

      // Optional: fetch the CertificateIssued event log for txHash + blockNumber.
      // Failure here is swallowed — the cert display still works without it.
      let txHash: `0x${string}` | undefined
      let blockNumber: bigint | undefined
      try {
        const logs = await publicClient.getContractEvents({
          address: CONTRACT_ADDRESS,
          abi: CONTRACT_ABI as Abi,
          eventName: 'CertificateIssued',
          args: { tokenId },
          fromBlock: 0n,
        })
        const firstLog = logs[0]
        if (firstLog) {
          txHash = firstLog.transactionHash ?? undefined
          blockNumber = firstLog.blockNumber ?? undefined
        }
      } catch {
        // ponytail: event lookup best-effort; cert data is already authoritative
      }

      return {
        cert: {
          ...cert,
          ...(txHash !== undefined ? { txHash } : {}),
          ...(blockNumber !== undefined ? { blockNumber } : {}),
        },
      }
    },
  })

  // Distinguish not-found from other errors
  const isNotFound = error !== null && isNotFoundError(error)
  const surfacedError = error !== null && !isNotFound ? error : null

  return {
    certificate: data?.cert ?? null,
    isLoading: !contractNotConfigured && tokenId !== null && isLoading,
    notFound: isNotFound,
    error: surfacedError,
    refetch: () => { void refetch() },
    contractNotConfigured,
  }
}
