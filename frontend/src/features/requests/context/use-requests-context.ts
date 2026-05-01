import { useContext } from 'react';
import { RequestsContext } from './requests-context-state';

export function useRequestsContext() {
  const context = useContext(RequestsContext);
  if (context === undefined) {
    throw new Error('useRequestsContext must be used within a RequestsProvider');
  }
  return context;
}
