import { createContext } from 'react';
import type { Role } from '../data/schema';

export interface RolesContextType {
  editingRole: Role | null;
  setEditingRole: (role: Role | null) => void;
  deletingRole: Role | null;
  setDeletingRole: (role: Role | null) => void;
  isCreateDialogOpen: boolean;
  setIsCreateDialogOpen: (open: boolean) => void;
}

export const RolesContext = createContext<RolesContextType | undefined>(undefined);
