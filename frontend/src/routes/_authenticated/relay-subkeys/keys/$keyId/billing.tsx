import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayKeyBillingPage } from '@/features/relay-subkeys';

function RelayKeyBillingRoute() {
  const { keyId } = Route.useParams();

  return (
    <RouteGuard requiredScopes={['read_api_keys']} scopeLevel='system'>
      <RelayKeyBillingPage keyId={keyId} />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/$keyId/billing')({
  component: RelayKeyBillingRoute,
});
