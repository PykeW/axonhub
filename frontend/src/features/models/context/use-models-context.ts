import { useContext } from 'react';
import { ModelsContext } from './models-context-state';

export function useModels() {
  const context = useContext(ModelsContext);
  if (context === undefined) {
    throw new Error('useModels must be used within a ModelsProvider');
  }
  return context;
}
