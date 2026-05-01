import { useContext } from 'react';
import { RulesContext } from './rules-context-state';

export function usePromptProtectionRules() {
  const context = useContext(RulesContext);
  if (!context) {
    throw new Error('usePromptProtectionRules must be used within PromptProtectionRulesProvider');
  }

  return context;
}
