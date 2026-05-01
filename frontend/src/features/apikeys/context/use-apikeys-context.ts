import { useContext } from 'react';
import { ApiKeysContext } from './apikeys-context-state';

export function useApiKeysContext() {
  const context = useContext(ApiKeysContext);
  if (context === undefined) {
    throw new Error('useApiKeysContext must be used within a ApiKeysProvider');
  }
  return context;
}
