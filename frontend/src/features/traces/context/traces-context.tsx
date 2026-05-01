'use client';

import { useState, type ReactNode } from 'react';
import type { RequestTrace, Span, Trace } from '../data/schema';
import { TracesContext, type TracesContextType } from './traces-context-state';

interface TracesProviderProps {
  children: ReactNode;
}

export default function TracesProvider({ children }: TracesProviderProps) {
  const [detailDialogOpen, setDetailDialogOpen] = useState(false);
  const [jsonViewerOpen, setJsonViewerOpen] = useState(false);
  const [jsonViewerData, setJsonViewerData] = useState<{ title: string; data: any } | null>(null);
  const [spanDetailOpen, setSpanDetailOpen] = useState(false);
  const [currentTrace, setCurrentTrace] = useState<Trace | null>(null);
  const [currentRequestTrace, setCurrentRequestTrace] = useState<RequestTrace | null>(null);
  const [currentSpan, setCurrentSpan] = useState<Span | null>(null);
  const [selectedTraces, setSelectedTraces] = useState<string[]>([]);

  const value: TracesContextType = {
    detailDialogOpen,
    setDetailDialogOpen,
    jsonViewerOpen,
    setJsonViewerOpen,
    jsonViewerData,
    setJsonViewerData,
    spanDetailOpen,
    setSpanDetailOpen,
    currentTrace,
    setCurrentTrace,
    currentRequestTrace,
    setCurrentRequestTrace,
    currentSpan,
    setCurrentSpan,
    selectedTraces,
    setSelectedTraces,
  };

  return <TracesContext.Provider value={value}>{children}</TracesContext.Provider>;
}

export { TracesProvider };
