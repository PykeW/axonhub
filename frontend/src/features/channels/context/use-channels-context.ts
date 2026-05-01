import React from 'react';
import { ChannelsContext } from './channels-context-state';

export const useChannels = () => {
  const channelsContext = React.useContext(ChannelsContext);

  if (!channelsContext) {
    throw new Error('useChannels has to be used within <ChannelsContext>');
  }

  return channelsContext;
};
