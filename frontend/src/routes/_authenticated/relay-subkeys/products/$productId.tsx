import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import { RelayProductDetailPage } from '@/features/relay-subkeys';

function RelayProductDetailRoute() {
  const { productId } = Route.useParams();

  return (
    <RouteGuard requiredScopes={['read_channels']} scopeLevel='system'>
      <RelayProductDetailPage productId={productId} />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/$productId')({
  component: RelayProductDetailRoute,
});
