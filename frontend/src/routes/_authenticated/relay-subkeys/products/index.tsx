import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayProductListPage } from '@/features/relay-subkeys';

function ProtectedRelayProductList() {
  return (
    <RouteGuard requiredScopes={['read_channels']} scopeLevel='system'>
      <RelayProductListPage />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/')({
  component: ProtectedRelayProductList,
});
