import { useContext } from 'react';
import { ProjectsContext } from './projects-context-state';

export function useProjectsContext() {
  const context = useContext(ProjectsContext);
  if (!context) {
    throw new Error('useProjectsContext must be used within a ProjectsProvider');
  }
  return context;
}
