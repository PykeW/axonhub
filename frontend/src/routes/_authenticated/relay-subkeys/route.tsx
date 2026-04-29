import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelaySubkeysLayout } from '@/features/relay-subkeys';

function ProtectedRelaySubkeysLayout() {
  return (
    <RouteGuard requiredScopes={['read_channels', 'read_api_keys', 'read_requests']}>
      <RelaySubkeysLayout />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys')({
  component: ProtectedRelaySubkeysLayout,
});
