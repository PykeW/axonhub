import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayKeyCreatePage } from '@/features/relay-subkeys';

function ProtectedRelayKeyCreate() {
  return (
    <RouteGuard requiredScopes={['write_api_keys']} scopeLevel='system'>
      <RelayKeyCreatePage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/create')({
  component: ProtectedRelayKeyCreate,
});
