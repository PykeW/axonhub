import { createContext } from 'react';

export interface SystemContextType {
  isLoading: boolean;
  setIsLoading: (loading: boolean) => void;
}

export const SystemContext = createContext<SystemContextType | undefined>(undefined);
