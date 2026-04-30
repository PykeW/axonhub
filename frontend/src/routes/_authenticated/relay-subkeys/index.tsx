import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelaySubkeysOverviewPage } from '@/features/relay-subkeys';

function ProtectedRelaySubkeysOverview() {
  return (
    <RouteGuard requiredScopes={['read_channels', 'read_api_keys', 'read_requests']} scopeLevel='system'>
      <RelaySubkeysOverviewPage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/')({
  component: ProtectedRelaySubkeysOverview,
});
