import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayRequestListPage } from '@/features/relay-subkeys';

function ProtectedRelayRequestList() {
  return (
    <RouteGuard requiredScopes={['read_requests']} scopeLevel='system'>
      <RelayRequestListPage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/requests/')({
  component: ProtectedRelayRequestList,
});
