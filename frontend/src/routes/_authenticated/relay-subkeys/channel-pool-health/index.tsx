import { createFileRoute } from '@tanstack/react-router';
import { RelayChannelPoolHealthPage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/channel-pool-health/')({
  component: RelayChannelPoolHealthPage,
});
