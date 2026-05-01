import { createContext } from 'react';
import type { Request, RequestExecution } from '../data/schema';

export interface RequestsContextType {
  // Dialog states
  detailDialogOpen: boolean;
  setDetailDialogOpen: (open: boolean) => void;

  // Execution detail dialog states
  executionDetailOpen: boolean;
  setExecutionDetailOpen: (open: boolean) => void;

  // Executions drawer states
  executionsDrawerOpen: boolean;
  setExecutionsDrawerOpen: (open: boolean) => void;

  // Current selected items
  currentRequest: Request | null;
  setCurrentRequest: (request: Request | null) => void;

  currentExecution: RequestExecution | null;
  setCurrentExecution: (execution: RequestExecution | null) => void;

  // Table selection
  selectedRequests: string[];
  setSelectedRequests: (ids: string[]) => void;
}

export const RequestsContext = createContext<RequestsContextType | undefined>(undefined);
