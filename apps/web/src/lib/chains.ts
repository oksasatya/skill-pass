import { http } from 'wagmi'
import { baseSepolia } from 'viem/chains'

const rpcUrl =
  import.meta.env.VITE_BASE_SEPOLIA_RPC ?? 'https://sepolia.base.org'

export { baseSepolia }
export const baseSepoliaTransport = http(rpcUrl)
