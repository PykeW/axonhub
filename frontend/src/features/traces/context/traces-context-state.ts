import { createContext } from 'react';
import type { Trace, RequestTrace, Span } from '../data/schema';

export interface TracesContextType {
  // Dialog states
  detailDialogOpen: boolean;
  setDetailDialogOpen: (open: boolean) => void;

  // JSON viewer dialog states
  jsonViewerOpen: boolean;
  setJsonViewerOpen: (open: boolean) => void;
  jsonViewerData: { title: string; data: any } | null;
  setJsonViewerData: (data: { title: string; data: any } | null) => void;

  // Span detail dialog states
  spanDetailOpen: boolean;
  setSpanDetailOpen: (open: boolean) => void;

  // Current selected items
  currentTrace: Trace | null;
  setCurrentTrace: (trace: Trace | null) => void;

  currentRequestTrace: RequestTrace | null;
  setCurrentRequestTrace: (requestTrace: RequestTrace | null) => void;

  currentSpan: Span | null;
  setCurrentSpan: (span: Span | null) => void;

  // Table selection
  selectedTraces: string[];
  setSelectedTraces: (ids: string[]) => void;
}

export const TracesContext = createContext<TracesContextType | undefined>(undefined);
