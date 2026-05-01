import { createContext } from 'react';
import type { PromptProtectionRule } from '../data/schema';

export type DialogType = 'create' | 'edit' | 'delete' | 'bulkEnable' | 'bulkDisable' | 'bulkDelete' | null;

export interface RulesContextType {
  open: DialogType;
  setOpen: (open: DialogType) => void;
  currentRow: PromptProtectionRule | null;
  setCurrentRow: (row: PromptProtectionRule | null) => void;
  selectedRules: PromptProtectionRule[];
  setSelectedRules: (rules: PromptProtectionRule[]) => void;
  resetRowSelection: (() => void) | null;
  setResetRowSelection: (fn: (() => void) | null) => void;
}

export const RulesContext = createContext<RulesContextType | undefined>(undefined);
