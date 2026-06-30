import { useChainId } from 'wagmi'
import { baseSepolia } from '@/lib/chains'

/**
 * Returns whether the connected wallet is on Base Sepolia (chainId 84532).
 * Used by NetworkGuard to gate writes.
 */
export function useIsCorrectNetwork(): { readonly isCorrectNetwork: boolean; readonly chainId: number } {
  const chainId = useChainId()
  return { isCorrectNetwork: chainId === baseSepolia.id, chainId }
}
