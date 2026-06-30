import CONTRACT_ABI from './SkillPassCertificate.abi.json'

// ponytail: address is env-only; no placeholder to avoid false deploys
export const CONTRACT_ADDRESS =
  import.meta.env.VITE_CONTRACT_ADDRESS as `0x${string}`

export { CONTRACT_ABI }

export const skillPassContract = {
  address: CONTRACT_ADDRESS,
  abi: CONTRACT_ABI,
} as const
