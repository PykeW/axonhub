import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayKeyListPage } from '@/features/relay-subkeys';

function ProtectedRelayKeyList() {
  return (
    <RouteGuard requiredScopes={['read_api_keys']} scopeLevel='system'>
      <RelayKeyListPage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/')({
  component: ProtectedRelayKeyList,
});
