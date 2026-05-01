/// <reference types="vite/client" />

import './routeTree.gen';

declare global {
  interface ImportMetaEnv {
    readonly VITE_RELAY_SUBKEYS_API_MODE?: 'mock' | 'rest';
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}