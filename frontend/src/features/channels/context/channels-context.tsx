import { useRef, useState, type ReactNode } from 'react';
import useDialogState from '@/hooks/use-dialog-state';
import type { Channel } from '../data/schema';
import { ChannelsContext, type ChannelsDialogType } from './channels-context-state';

interface Props {
  children: ReactNode;
}

export default function ChannelsProvider({ children }: Props) {
  const [open, setOpen] = useDialogState<ChannelsDialogType>(null);
  const [currentRow, setCurrentRow] = useState<Channel | null>(null);
  const [selectedChannels, setSelectedChannels] = useState<Channel[]>([]);
  const resetRowSelectionRef = useRef<() => void>(() => {});

  return (
    <ChannelsContext.Provider
      value={{
        open,
        setOpen,
        currentRow,
        setCurrentRow,
        selectedChannels,
        setSelectedChannels,
        resetRowSelection: () => resetRowSelectionRef.current(),
        setResetRowSelection: (fn: () => void) => {
          resetRowSelectionRef.current = fn;
        },
      }}
    >
      {children}
    </ChannelsContext.Provider>
  );
}
