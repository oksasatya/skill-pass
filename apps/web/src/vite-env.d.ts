/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_CONTRACT_ADDRESS: string
  readonly VITE_BASE_SEPOLIA_RPC: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
