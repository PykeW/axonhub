import { createContext } from 'react';
import type { Model } from '../data/schema';

export type DialogType =
  | 'create'
  | 'batchCreate'
  | 'edit'
  | 'delete'
  | 'archive'
  | 'association'
  | 'settings'
  | 'bulkEnable'
  | 'bulkDisable'
  | 'unassociated'
  | null;

export interface ModelsContextType {
  open: DialogType;
  setOpen: (open: DialogType) => void;
  currentRow: Model | null;
  setCurrentRow: (row: Model | null) => void;
  selectedModels: Model[];
  setSelectedModels: (models: Model[]) => void;
  resetRowSelection: (() => void) | null;
  setResetRowSelection: (fn: (() => void) | null) => void;
}

export const ModelsContext = createContext<ModelsContextType | undefined>(undefined);
