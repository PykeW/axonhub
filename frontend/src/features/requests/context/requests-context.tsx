'use client';

import { useState, type ReactNode } from 'react';
import type { Request, RequestExecution } from '../data/schema';
import { RequestsContext, type RequestsContextType } from './requests-context-state';

interface RequestsProviderProps {
  children: ReactNode;
}

export default function RequestsProvider({ children }: RequestsProviderProps) {
  const [detailDialogOpen, setDetailDialogOpen] = useState(false);
  const [executionDetailOpen, setExecutionDetailOpen] = useState(false);
  const [executionsDrawerOpen, setExecutionsDrawerOpen] = useState(false);
  const [currentRequest, setCurrentRequest] = useState<Request | null>(null);
  const [currentExecution, setCurrentExecution] = useState<RequestExecution | null>(null);
  const [selectedRequests, setSelectedRequests] = useState<string[]>([]);

  const value: RequestsContextType = {
    detailDialogOpen,
    setDetailDialogOpen,
    executionDetailOpen,
    setExecutionDetailOpen,
    executionsDrawerOpen,
    setExecutionsDrawerOpen,
    currentRequest,
    setCurrentRequest,
    currentExecution,
    setCurrentExecution,
    selectedRequests,
    setSelectedRequests,
  };

  return <RequestsContext.Provider value={value}>{children}</RequestsContext.Provider>;
}

export { RequestsProvider };
