/// <reference types="vite/client" />

// fontsource packages ship CSS files; declare them as modules so TS side-effect imports compile
declare module '@fontsource-variable/inter'
declare module '@fontsource-variable/fraunces'
declare module '@fontsource/jetbrains-mono/400.css'
declare module '@fontsource/jetbrains-mono/500.css'

interface ImportMetaEnv {
  readonly VITE_CONTRACT_ADDRESS: string
  readonly VITE_BASE_SEPOLIA_RPC: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
