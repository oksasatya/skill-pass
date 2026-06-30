/**
 * useIsAdmin — reads the contract owner() and compares to the connected address.
 *
 * Guard: if CONTRACT_ADDRESS is undefined (no .env yet), returns a clear
 * contractNotConfigured flag rather than crashing the page.
 *
 * Returns:
 *   isAdmin             — true iff connected address matches owner (case-insensitive)
 *   owner               — the contract owner address (or undefined while loading)
 *   isLoading           — true while the RPC call is in flight
 *   contractNotConfigured — true when VITE_CONTRACT_ADDRESS is unset
 */

import { useAccount, useReadContract } from 'wagmi'
import { CONTRACT_ADDRESS, CONTRACT_ABI } from '@/lib/contract'

export type UseIsAdminResult = {
  readonly isAdmin: boolean
  readonly owner: `0x${string}` | undefined
  readonly isLoading: boolean
  readonly contractNotConfigured: boolean
}

export function useIsAdmin(): UseIsAdminResult {
  const { address } = useAccount()

  // ponytail: CONTRACT_ADDRESS is undefined when VITE_CONTRACT_ADDRESS is unset;
  // skip the RPC call and surface contractNotConfigured instead of crashing.
  const contractNotConfigured = !CONTRACT_ADDRESS

  const { data: owner, isLoading } = useReadContract(
    contractNotConfigured
      ? ({ query: { enabled: false } } as Parameters<typeof useReadContract>[0])
      : {
          address: CONTRACT_ADDRESS,
          abi: CONTRACT_ABI,
          functionName: 'owner',
        },
  )

  const isAdmin =
    !contractNotConfigured &&
    !!address &&
    !!owner &&
    (owner as string).toLowerCase() === address.toLowerCase()

  return {
    isAdmin,
    owner: owner as `0x${string}` | undefined,
    isLoading: !contractNotConfigured && isLoading,
    contractNotConfigured,
  }
}
