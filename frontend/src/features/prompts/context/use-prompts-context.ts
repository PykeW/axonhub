import { useContext } from 'react';
import { PromptsContext } from './prompts-context-state';

export function usePrompts() {
  const context = useContext(PromptsContext);
  if (context === undefined) {
    throw new Error('usePrompts must be used within a PromptsProvider');
  }
  return context;
}
