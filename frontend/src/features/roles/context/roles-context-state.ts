import { createContext } from 'react';
import type { Role } from '../data/schema';

export type RoleDialogType = 'create' | 'edit' | 'delete' | 'bulkDelete';

export interface RolesContextType {
  editingRole: Role | null;
  setEditingRole: (role: Role | null) => void;
  deletingRole: Role | null;
  setDeletingRole: (role: Role | null) => void;
  selectedRoles: Role[];
  setSelectedRoles: (roles: Role[]) => void;
  isDialogOpen: Record<RoleDialogType, boolean>;
  openDialog: (type: RoleDialogType, role?: Role | Role[]) => void;
  closeDialog: (type?: RoleDialogType) => void;
  resetRowSelection: () => void;
  setResetRowSelection: (fn: () => void) => void;
}

export const RolesContext = createContext<RolesContextType | undefined>(undefined);
