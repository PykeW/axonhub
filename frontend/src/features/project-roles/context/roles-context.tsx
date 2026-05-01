import { useState, type ReactNode } from 'react';
import type { Role } from '../data/schema';
import { RolesContext } from './roles-context-state';

interface RolesProviderProps {
  children: ReactNode;
}

export default function RolesProvider({ children }: RolesProviderProps) {
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [deletingRole, setDeletingRole] = useState<Role | null>(null);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);

  return (
    <RolesContext.Provider
      value={{
        editingRole,
        setEditingRole,
        deletingRole,
        setDeletingRole,
        isCreateDialogOpen,
        setIsCreateDialogOpen,
      }}
    >
      {children}
    </RolesContext.Provider>
  );
}
