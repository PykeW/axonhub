import { useContext } from 'react';
import { RolesContext } from './roles-context-state';

export function useRolesContext() {
  const context = useContext(RolesContext);
  if (!context) {
    throw new Error('useRolesContext must be used within a RolesProvider');
  }
  return context;
}
