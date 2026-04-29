/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_RELAY_SUBKEYS_API_MODE?: 'mock' | 'rest';
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}