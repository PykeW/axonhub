import { useContext } from 'react';
import { DataStoragesContext } from './data-storages-context-state';

export function useDataStoragesContext() {
  const context = useContext(DataStoragesContext);
  if (!context) {
    throw new Error('useDataStoragesContext must be used within DataStoragesProvider');
  }
  return context;
}
