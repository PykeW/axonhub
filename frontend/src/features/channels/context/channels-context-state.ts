import React from 'react';
import type { Channel } from '../data/schema';

export type ChannelsDialogType =
  | 'add'
  | 'duplicate'
  | 'edit'
  | 'delete'
  | 'settings'
  | 'channelSettings'
  | 'modelMapping'
  | 'overrides'
  | 'proxy'
  | 'status'
  | 'test'
  | 'testHistory'
  | 'bulkImport'
  | 'archive'
  | 'bulkOrdering'
  | 'bulkArchive'
  | 'bulkDisable'
  | 'bulkEnable'
  | 'bulkTest'
  | 'bulkDelete'
  | 'bulkApplyTemplate'
  | 'errorResolved'
  | 'viewModels'
  | 'price'
  | 'transformOptions'
  | 'rateLimit'
  | 'testAPIKeys'
  | 'disabledAPIKeys';

export interface ChannelsContextType {
  open: ChannelsDialogType | null;
  setOpen: (str: ChannelsDialogType | null) => void;
  currentRow: Channel | null;
  setCurrentRow: React.Dispatch<React.SetStateAction<Channel | null>>;
  selectedChannels: Channel[];
  setSelectedChannels: React.Dispatch<React.SetStateAction<Channel[]>>;
  resetRowSelection: () => void;
  setResetRowSelection: (fn: () => void) => void;
}

export const ChannelsContext = React.createContext<ChannelsContextType | null>(null);
