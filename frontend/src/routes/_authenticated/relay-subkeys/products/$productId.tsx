import { createFileRoute } from '@tanstack/react-router';
import { RelayProductDetailPage } from '@/features/relay-subkeys';

function RelayProductDetailRoute() {
  const { productId } = Route.useParams();

  return <RelayProductDetailPage productId={productId} />;
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/$productId')({
  component: RelayProductDetailRoute,
});
