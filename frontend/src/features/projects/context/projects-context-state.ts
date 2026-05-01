import { createContext } from 'react';
import type { Project } from '../data/schema';

export interface ProjectsContextType {
  editingProject: Project | null;
  setEditingProject: (project: Project | null) => void;
  archivingProject: Project | null;
  setArchivingProject: (project: Project | null) => void;
  activatingProject: Project | null;
  setActivatingProject: (project: Project | null) => void;
  deletingProject: Project | null;
  setDeletingProject: (project: Project | null) => void;
  profilesProject: Project | null;
  setProfilesProject: (project: Project | null) => void;
  isCreateDialogOpen: boolean;
  setIsCreateDialogOpen: (open: boolean) => void;
}

export const ProjectsContext = createContext<ProjectsContextType | undefined>(undefined);
