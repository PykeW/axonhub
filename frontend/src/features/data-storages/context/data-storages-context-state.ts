import { createContext } from 'react';
import type { DataStorage } from '../data/data-storages';

export interface DataStoragesContextType {
  isCreateDialogOpen: boolean;
  setIsCreateDialogOpen: (open: boolean) => void;
  isEditDialogOpen: boolean;
  setIsEditDialogOpen: (open: boolean) => void;
  isArchiveDialogOpen: boolean;
  setIsArchiveDialogOpen: (open: boolean) => void;
  editingDataStorage: DataStorage | null;
  setEditingDataStorage: (dataStorage: DataStorage | null) => void;
  archiveDataStorage: DataStorage | null;
  setArchiveDataStorage: (dataStorage: DataStorage | null) => void;
}

export const DataStoragesContext = createContext<DataStoragesContextType | undefined>(undefined);
