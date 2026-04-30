import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayChannelPoolHealthPage } from '@/features/relay-subkeys';

function ProtectedRelayChannelPoolHealth() {
  return (
    <RouteGuard requiredScopes={['read_channels']} scopeLevel='system'>
      <RelayChannelPoolHealthPage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/channel-pool-health/')({
  component: ProtectedRelayChannelPoolHealth,
});
