import { useContext } from 'react';
import { SystemContext } from './system-context-state';

export function useSystemContext() {
  const context = useContext(SystemContext);
  if (!context) {
    throw new Error('useSystemContext must be used within a SystemProvider');
  }
  return context;
}
