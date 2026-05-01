import { createContext } from 'react';
import type { ApiKey } from '../data/schema';

export type ApiKeyDialogType =
  | 'create'
  | 'edit'
  | 'delete'
  | 'status'
  | 'view'
  | 'profiles'
  | 'archive'
  | 'bulkDisable'
  | 'bulkArchive'
  | 'bulkEnable';

export interface ApiKeysContextType {
  selectedApiKey: ApiKey | null;
  setSelectedApiKey: (apiKey: ApiKey | null) => void;
  selectedApiKeys: ApiKey[];
  setSelectedApiKeys: (apiKeys: ApiKey[]) => void;
  isDialogOpen: Record<ApiKeyDialogType, boolean>;
  openDialog: (type: ApiKeyDialogType, apiKey?: ApiKey | ApiKey[]) => void;
  closeDialog: (type?: ApiKeyDialogType) => void;
  resetRowSelection: () => void;
  setResetRowSelection: (fn: () => void) => void;
}

export const ApiKeysContext = createContext<ApiKeysContextType | undefined>(undefined);
