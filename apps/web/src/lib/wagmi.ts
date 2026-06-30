import { createConfig } from 'wagmi'
import { injected, coinbaseWallet } from 'wagmi/connectors'
import { baseSepolia, baseSepoliaTransport } from './chains'

export const config = createConfig({
  chains: [baseSepolia],
  connectors: [
    injected(),
    coinbaseWallet({ appName: 'SkillPass' }),
  ],
  transports: {
    [baseSepolia.id]: baseSepoliaTransport,
  },
})

// Module augmentation so wagmi hooks infer the correct chain/connector types.
declare module 'wagmi' {
  interface Register {
    config: typeof config
  }
}
