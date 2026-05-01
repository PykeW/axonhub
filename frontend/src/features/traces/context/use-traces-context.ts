import { useContext } from 'react';
import { TracesContext } from './traces-context-state';

export function useTracesContext() {
  const context = useContext(TracesContext);
  if (context === undefined) {
    throw new Error('useTracesContext must be used within a TracesProvider');
  }
  return context;
}
