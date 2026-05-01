import { useCallback, useMemo, useState, type ReactNode } from 'react';
import type { PromptProtectionRule } from '../data/schema';
import { RulesContext, type DialogType } from './rules-context-state';

export function PromptProtectionRulesProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState<DialogType>(null);
  const [currentRow, setCurrentRow] = useState<PromptProtectionRule | null>(null);
  const [selectedRules, setSelectedRules] = useState<PromptProtectionRule[]>([]);
  const [resetRowSelection, setResetRowSelection] = useState<(() => void) | null>(null);

  const handleSetOpen = useCallback((nextOpen: DialogType) => {
    setOpen(nextOpen);
    if (nextOpen !== 'edit' && nextOpen !== 'delete') {
      setCurrentRow(null);
    }
  }, []);

  const value = useMemo(
    () => ({
      open,
      setOpen: handleSetOpen,
      currentRow,
      setCurrentRow,
      selectedRules,
      setSelectedRules,
      resetRowSelection,
      setResetRowSelection,
    }),
    [open, handleSetOpen, currentRow, selectedRules, resetRowSelection]
  );

  return <RulesContext.Provider value={value}>{children}</RulesContext.Provider>;
}

export default PromptProtectionRulesProvider;
