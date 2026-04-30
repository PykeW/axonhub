import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayProductCreatePage } from '@/features/relay-subkeys';

function ProtectedRelayProductCreate() {
  return (
    <RouteGuard requiredScopes={['write_channels']} scopeLevel='system'>
      <RelayProductCreatePage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/create')({
  component: ProtectedRelayProductCreate,
});
