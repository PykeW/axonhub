import { createContext } from 'react';
import type { Prompt } from '../data/schema';

export type DialogType =
  | 'create'
  | 'edit'
  | 'delete'
  | 'bulkEnable'
  | 'bulkDisable'
  | 'bulkDelete'
  | null;

export interface PromptsContextType {
  open: DialogType;
  setOpen: (open: DialogType) => void;
  currentRow: Prompt | null;
  setCurrentRow: (row: Prompt | null) => void;
  selectedPrompts: Prompt[];
  setSelectedPrompts: (prompts: Prompt[]) => void;
  resetRowSelection: (() => void) | null;
  setResetRowSelection: (fn: (() => void) | null) => void;
}

export const PromptsContext = createContext<PromptsContextType | undefined>(undefined);
