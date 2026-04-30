import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayKeyDetailPage } from '@/features/relay-subkeys';

function RelayKeyDetailRoute() {
  const { keyId } = Route.useParams();

  return (
    <RouteGuard requiredScopes={['read_api_keys']} scopeLevel='system'>
      <RelayKeyDetailPage keyId={keyId} />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/$keyId')({
  component: RelayKeyDetailRoute,
});
