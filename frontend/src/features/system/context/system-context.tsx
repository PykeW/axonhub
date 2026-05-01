'use client';

import { useState, type ReactNode } from 'react';
import { SystemContext, type SystemContextType } from './system-context-state';

interface SystemProviderProps {
  children: ReactNode;
}

export default function SystemProvider({ children }: SystemProviderProps) {
  const [isLoading, setIsLoading] = useState(false);

  const value: SystemContextType = {
    isLoading,
    setIsLoading,
  };

  return <SystemContext.Provider value={value}>{children}</SystemContext.Provider>;
}
