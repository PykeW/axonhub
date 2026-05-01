import { useCallback, useMemo, useState, type ReactNode } from 'react';
import type { Model } from '../data/schema';
import { ModelsContext, type DialogType } from './models-context-state';

export function ModelsProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState<DialogType>(null);
  const [currentRow, setCurrentRow] = useState<Model | null>(null);
  const [selectedModels, setSelectedModels] = useState<Model[]>([]);
  const [resetRowSelection, setResetRowSelection] = useState<(() => void) | null>(null);

  const handleSetOpen = useCallback((newOpen: DialogType) => {
    setOpen(newOpen);
  }, []);

  const handleSetCurrentRow = useCallback((row: Model | null) => {
    setCurrentRow(row);
  }, []);

  const handleSetSelectedModels = useCallback((models: Model[]) => {
    setSelectedModels(models);
  }, []);

  const handleSetResetRowSelection = useCallback((fn: (() => void) | null) => {
    setResetRowSelection(() => fn);
  }, []);

  const value = useMemo(
    () => ({
      open,
      setOpen: handleSetOpen,
      currentRow,
      setCurrentRow: handleSetCurrentRow,
      selectedModels,
      setSelectedModels: handleSetSelectedModels,
      resetRowSelection,
      setResetRowSelection: handleSetResetRowSelection,
    }),
    [open, handleSetOpen, currentRow, handleSetCurrentRow, selectedModels, handleSetSelectedModels, resetRowSelection, handleSetResetRowSelection]
  );

  return <ModelsContext.Provider value={value}>{children}</ModelsContext.Provider>;
}

export default ModelsProvider;
